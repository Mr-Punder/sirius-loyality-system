package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStorage реализует интерфейс Storage с хранением данных в PostgreSQL
type PgStorage struct {
	pool *pgxpool.Pool
}

// NewPgStorage создает новое хранилище PostgreSQL
func NewPgStorage(connString string) (*PgStorage, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	storage := &PgStorage{
		pool: pool,
	}

	return storage, nil
}

func (p *PgStorage) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// ==================== МЕТОДЫ ДЛЯ ПОЛЬЗОВАТЕЛЕЙ ====================

func (p *PgStorage) GetUser(userId uuid.UUID) (*models.User, error) {
	ctx := context.Background()
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, "group", registration_time, deleted
		FROM users
		WHERE id = $1 AND deleted = false
	`

	var user models.User
	err := p.pool.QueryRow(ctx, query, userId).Scan(
		&user.Id,
		&user.Telegramm,
		&user.FirstName,
		&user.LastName,
		&user.MiddleName,
		&user.Group,
		&user.RegistrationTime,
		&user.Deleted,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (p *PgStorage) GetUserByTelegramm(telegramm string) (*models.User, error) {
	ctx := context.Background()
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, "group", registration_time, deleted
		FROM users
		WHERE telegramm = $1 AND deleted = false
	`

	var user models.User
	err := p.pool.QueryRow(ctx, query, telegramm).Scan(
		&user.Id,
		&user.Telegramm,
		&user.FirstName,
		&user.LastName,
		&user.MiddleName,
		&user.Group,
		&user.RegistrationTime,
		&user.Deleted,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user by telegramm: %w", err)
	}

	return &user, nil
}

func (p *PgStorage) GetAllUsers() ([]*models.User, error) {
	ctx := context.Background()
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, "group", registration_time, deleted
		FROM users
		WHERE deleted = false
		ORDER BY registration_time DESC
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.Id,
			&user.Telegramm,
			&user.FirstName,
			&user.LastName,
			&user.MiddleName,
			&user.Group,
			&user.RegistrationTime,
			&user.Deleted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}

func (p *PgStorage) AddUser(user *models.User) error {
	ctx := context.Background()

	// Проверяем существование
	var count int
	err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE telegramm = $1 AND deleted = false`, user.Telegramm).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}
	if count > 0 {
		return errors.New("user with this telegram ID already exists")
	}

	query := `
		INSERT INTO users (id, telegramm, first_name, last_name, middle_name, "group", registration_time, deleted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = p.pool.Exec(ctx, query,
		user.Id,
		user.Telegramm,
		user.FirstName,
		user.LastName,
		user.MiddleName,
		user.Group,
		user.RegistrationTime,
		user.Deleted,
	)

	if err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	return nil
}

func (p *PgStorage) UpdateUser(user *models.User) error {
	ctx := context.Background()
	query := `
		UPDATE users
		SET telegramm = $1, first_name = $2, last_name = $3, middle_name = $4, "group" = $5, registration_time = $6, deleted = $7
		WHERE id = $8
	`

	result, err := p.pool.Exec(ctx, query,
		user.Telegramm,
		user.FirstName,
		user.LastName,
		user.MiddleName,
		user.Group,
		user.RegistrationTime,
		user.Deleted,
		user.Id,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (p *PgStorage) DeleteUser(userId uuid.UUID) error {
	ctx := context.Background()

	// Сначала освобождаем детали пазлов, принадлежащие пользователю
	_, err := p.pool.Exec(ctx, `UPDATE puzzle_pieces SET owner_id = NULL, registered_at = NULL WHERE owner_id = $1`, userId)
	if err != nil {
		return fmt.Errorf("failed to release puzzle pieces: %w", err)
	}

	// Удаляем пользователя
	query := `DELETE FROM users WHERE id = $1`
	result, err := p.pool.Exec(ctx, query, userId)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("user not found")
	}

	return nil
}

// ==================== МЕТОДЫ ДЛЯ ПАЗЛОВ ====================

func (p *PgStorage) GetPuzzle(puzzleId int) (*models.Puzzle, error) {
	ctx := context.Background()
	query := `SELECT id, COALESCE(name, ''), is_completed, completed_at FROM puzzles WHERE id = $1`

	var puzzle models.Puzzle
	err := p.pool.QueryRow(ctx, query, puzzleId).Scan(&puzzle.Id, &puzzle.Name, &puzzle.IsCompleted, &puzzle.CompletedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("puzzle not found")
		}
		return nil, fmt.Errorf("failed to get puzzle: %w", err)
	}

	return &puzzle, nil
}

func (p *PgStorage) GetAllPuzzles() ([]*models.Puzzle, error) {
	ctx := context.Background()
	query := `SELECT id, COALESCE(name, ''), is_completed, completed_at FROM puzzles ORDER BY id`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query puzzles: %w", err)
	}
	defer rows.Close()

	var puzzles []*models.Puzzle
	for rows.Next() {
		var puzzle models.Puzzle
		err := rows.Scan(&puzzle.Id, &puzzle.Name, &puzzle.IsCompleted, &puzzle.CompletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan puzzle: %w", err)
		}
		puzzles = append(puzzles, &puzzle)
	}

	return puzzles, nil
}

func (p *PgStorage) UpdatePuzzle(puzzle *models.Puzzle) error {
	ctx := context.Background()
	query := `UPDATE puzzles SET name = $1, is_completed = $2, completed_at = $3 WHERE id = $4`

	result, err := p.pool.Exec(ctx, query, puzzle.Name, puzzle.IsCompleted, puzzle.CompletedAt, puzzle.Id)
	if err != nil {
		return fmt.Errorf("failed to update puzzle: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("puzzle not found")
	}

	return nil
}

// CompletePuzzle отмечает пазл как собранный и возвращает владельцев деталей для уведомления
func (p *PgStorage) CompletePuzzle(puzzleId int) ([]*models.User, error) {
	ctx := context.Background()

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Проверяем, что пазл существует и еще не завершен
	var isCompleted bool
	err = tx.QueryRow(ctx, `SELECT is_completed FROM puzzles WHERE id = $1`, puzzleId).Scan(&isCompleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("puzzle not found")
		}
		return nil, fmt.Errorf("failed to check puzzle: %w", err)
	}

	if isCompleted {
		return nil, errors.New("puzzle already completed")
	}

	// Отмечаем пазл как завершенный
	now := time.Now()
	_, err = tx.Exec(ctx, `UPDATE puzzles SET is_completed = true, completed_at = $1 WHERE id = $2`, now, puzzleId)
	if err != nil {
		return nil, fmt.Errorf("failed to complete puzzle: %w", err)
	}

	// Получаем владельцев деталей этого пазла
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT u.id, u.telegramm, u.first_name, u.last_name, u.middle_name, u."group", u.registration_time, u.deleted
		FROM puzzle_pieces pp
		JOIN users u ON pp.owner_id = u.id
		WHERE pp.puzzle_id = $1 AND pp.owner_id IS NOT NULL
	`, puzzleId)
	if err != nil {
		return nil, fmt.Errorf("failed to get piece owners: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.Id, &user.Telegramm, &user.FirstName, &user.LastName, &user.MiddleName, &user.Group, &user.RegistrationTime, &user.Deleted)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return users, nil
}

// ==================== МЕТОДЫ ДЛЯ ДЕТАЛЕЙ ПАЗЛОВ ====================

func (p *PgStorage) GetPuzzlePiece(code string) (*models.PuzzlePiece, error) {
	ctx := context.Background()
	query := `SELECT code, puzzle_id, piece_number, owner_id, registered_at FROM puzzle_pieces WHERE code = $1`

	var piece models.PuzzlePiece
	err := p.pool.QueryRow(ctx, query, code).Scan(
		&piece.Code,
		&piece.PuzzleId,
		&piece.PieceNumber,
		&piece.OwnerId,
		&piece.RegisteredAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("piece not found")
		}
		return nil, fmt.Errorf("failed to get puzzle piece: %w", err)
	}

	return &piece, nil
}

func (p *PgStorage) GetPuzzlePiecesByPuzzle(puzzleId int) ([]*models.PuzzlePiece, error) {
	ctx := context.Background()
	query := `SELECT code, puzzle_id, piece_number, owner_id, registered_at FROM puzzle_pieces WHERE puzzle_id = $1 ORDER BY piece_number`

	rows, err := p.pool.Query(ctx, query, puzzleId)
	if err != nil {
		return nil, fmt.Errorf("failed to query puzzle pieces: %w", err)
	}
	defer rows.Close()

	return p.scanPuzzlePieces(rows)
}

func (p *PgStorage) GetPuzzlePiecesByOwner(ownerId uuid.UUID) ([]*models.PuzzlePiece, error) {
	ctx := context.Background()
	query := `SELECT code, puzzle_id, piece_number, owner_id, registered_at FROM puzzle_pieces WHERE owner_id = $1 ORDER BY puzzle_id, piece_number`

	rows, err := p.pool.Query(ctx, query, ownerId)
	if err != nil {
		return nil, fmt.Errorf("failed to query puzzle pieces by owner: %w", err)
	}
	defer rows.Close()

	return p.scanPuzzlePieces(rows)
}

func (p *PgStorage) GetAllPuzzlePieces() ([]*models.PuzzlePiece, error) {
	ctx := context.Background()
	query := `SELECT code, puzzle_id, piece_number, owner_id, registered_at FROM puzzle_pieces ORDER BY puzzle_id, piece_number`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all puzzle pieces: %w", err)
	}
	defer rows.Close()

	return p.scanPuzzlePieces(rows)
}

func (p *PgStorage) scanPuzzlePieces(rows pgx.Rows) ([]*models.PuzzlePiece, error) {
	var pieces []*models.PuzzlePiece
	for rows.Next() {
		var piece models.PuzzlePiece
		err := rows.Scan(
			&piece.Code,
			&piece.PuzzleId,
			&piece.PieceNumber,
			&piece.OwnerId,
			&piece.RegisteredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan puzzle piece: %w", err)
		}
		pieces = append(pieces, &piece)
	}
	return pieces, nil
}

func (p *PgStorage) AddPuzzlePiece(piece *models.PuzzlePiece) error {
	ctx := context.Background()
	query := `INSERT INTO puzzle_pieces (code, puzzle_id, piece_number, owner_id, registered_at) VALUES ($1, $2, $3, $4, $5)`

	_, err := p.pool.Exec(ctx, query, piece.Code, piece.PuzzleId, piece.PieceNumber, piece.OwnerId, piece.RegisteredAt)
	if err != nil {
		return fmt.Errorf("failed to add puzzle piece: %w", err)
	}

	return nil
}

func (p *PgStorage) AddPuzzlePieces(pieces []*models.PuzzlePiece) error {
	ctx := context.Background()

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, piece := range pieces {
		_, err := tx.Exec(ctx,
			`INSERT INTO puzzle_pieces (code, puzzle_id, piece_number, owner_id, registered_at) VALUES ($1, $2, $3, $4, $5)`,
			piece.Code, piece.PuzzleId, piece.PieceNumber, piece.OwnerId, piece.RegisteredAt,
		)
		if err != nil {
			return fmt.Errorf("failed to add puzzle piece %s: %w", piece.Code, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (p *PgStorage) RegisterPuzzlePiece(code string, ownerId uuid.UUID) (*models.PuzzlePiece, bool, error) {
	ctx := context.Background()

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Проверяем существование и статус детали
	var puzzleId, pieceNumber int
	var existingOwnerId *uuid.UUID
	checkQuery := `SELECT puzzle_id, piece_number, owner_id FROM puzzle_pieces WHERE code = $1`
	err = tx.QueryRow(ctx, checkQuery, code).Scan(&puzzleId, &pieceNumber, &existingOwnerId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, errors.New("piece not found")
		}
		return nil, false, fmt.Errorf("failed to check piece: %w", err)
	}

	if existingOwnerId != nil {
		return nil, false, errors.New("piece already taken")
	}

	// Привязываем деталь
	now := time.Now()
	_, err = tx.Exec(ctx, `UPDATE puzzle_pieces SET owner_id = $1, registered_at = $2 WHERE code = $3`, ownerId, now, code)
	if err != nil {
		return nil, false, fmt.Errorf("failed to register piece: %w", err)
	}

	// Проверяем, все ли детали пазла розданы (для информирования пользователя)
	var count int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM puzzle_pieces WHERE puzzle_id = $1 AND owner_id IS NOT NULL`, puzzleId).Scan(&count)
	if err != nil {
		return nil, false, fmt.Errorf("failed to count pieces: %w", err)
	}

	allPiecesDistributed := count == 6

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	piece := &models.PuzzlePiece{
		Code:         code,
		PuzzleId:     puzzleId,
		PieceNumber:  pieceNumber,
		OwnerId:      &ownerId,
		RegisteredAt: &now,
	}

	return piece, allPiecesDistributed, nil
}

// ==================== МЕТОДЫ ДЛЯ СТАТИСТИКИ ====================

func (p *PgStorage) GetUserPieceCount(userId uuid.UUID) (int, error) {
	ctx := context.Background()
	var count int
	err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM puzzle_pieces WHERE owner_id = $1`, userId).Scan(&count)
	return count, err
}

func (p *PgStorage) GetUserCompletedPuzzlePieceCount(userId uuid.UUID) (int, error) {
	ctx := context.Background()
	query := `
		SELECT COUNT(*) FROM puzzle_pieces pp
		JOIN puzzles p ON pp.puzzle_id = p.id
		WHERE pp.owner_id = $1 AND p.is_completed = true
	`
	var count int
	err := p.pool.QueryRow(ctx, query, userId).Scan(&count)
	return count, err
}

// ==================== МЕТОДЫ ДЛЯ АДМИНИСТРАТОРОВ ====================

func (p *PgStorage) GetAdmin(adminId int64) (*models.Admin, error) {
	ctx := context.Background()
	query := `SELECT id, name, username, is_active FROM admins WHERE id = $1`

	var admin models.Admin
	err := p.pool.QueryRow(ctx, query, adminId).Scan(&admin.ID, &admin.Name, &admin.Username, &admin.IsActive)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("admin not found")
		}
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	return &admin, nil
}

func (p *PgStorage) GetAllAdmins() ([]*models.Admin, error) {
	ctx := context.Background()
	query := `SELECT id, name, username, is_active FROM admins ORDER BY id`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query admins: %w", err)
	}
	defer rows.Close()

	var admins []*models.Admin
	for rows.Next() {
		var admin models.Admin
		err := rows.Scan(&admin.ID, &admin.Name, &admin.Username, &admin.IsActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin: %w", err)
		}
		admins = append(admins, &admin)
	}

	return admins, nil
}

func (p *PgStorage) AddAdmin(admin *models.Admin) error {
	ctx := context.Background()

	var count int
	err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admins WHERE id = $1`, admin.ID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing admin: %w", err)
	}
	if count > 0 {
		return errors.New("admin with this ID already exists")
	}

	query := `INSERT INTO admins (id, name, username, is_active) VALUES ($1, $2, $3, $4)`
	_, err = p.pool.Exec(ctx, query, admin.ID, admin.Name, admin.Username, admin.IsActive)
	if err != nil {
		return fmt.Errorf("failed to add admin: %w", err)
	}

	return nil
}

func (p *PgStorage) UpdateAdmin(admin *models.Admin) error {
	ctx := context.Background()
	query := `UPDATE admins SET name = $1, username = $2, is_active = $3 WHERE id = $4`

	result, err := p.pool.Exec(ctx, query, admin.Name, admin.Username, admin.IsActive, admin.ID)
	if err != nil {
		return fmt.Errorf("failed to update admin: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("admin not found")
	}

	return nil
}

func (p *PgStorage) DeleteAdmin(adminId int64) error {
	ctx := context.Background()
	query := `DELETE FROM admins WHERE id = $1`

	result, err := p.pool.Exec(ctx, query, adminId)
	if err != nil {
		return fmt.Errorf("failed to delete admin: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("admin not found")
	}

	return nil
}

// ==================== МЕТОДЫ ДЛЯ УВЕДОМЛЕНИЙ ====================

func (p *PgStorage) AddNotification(notification *models.Notification) error {
	ctx := context.Background()
	query := `INSERT INTO notifications (id, message, group_filter, attachments, user_ids, status, created_at, sent_at, sent_count, error_count)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := p.pool.Exec(ctx, query,
		notification.Id,
		notification.Message,
		notification.Group,
		notification.Attachments,
		notification.UserIds,
		notification.Status,
		notification.CreatedAt,
		notification.SentAt,
		notification.SentCount,
		notification.ErrorCount,
	)
	if err != nil {
		return fmt.Errorf("failed to add notification: %w", err)
	}

	return nil
}

func (p *PgStorage) GetPendingNotifications() ([]*models.Notification, error) {
	ctx := context.Background()
	query := `SELECT id, message, group_filter, attachments, user_ids, status, created_at, sent_at, sent_count, error_count
	          FROM notifications WHERE status = $1 ORDER BY created_at`

	rows, err := p.pool.Query(ctx, query, models.NotificationPending)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending notifications: %w", err)
	}
	defer rows.Close()

	return p.scanNotifications(rows)
}

func (p *PgStorage) UpdateNotification(notification *models.Notification) error {
	ctx := context.Background()
	query := `UPDATE notifications SET message = $1, group_filter = $2, attachments = $3, user_ids = $4, status = $5,
	          sent_at = $6, sent_count = $7, error_count = $8 WHERE id = $9`

	result, err := p.pool.Exec(ctx, query,
		notification.Message,
		notification.Group,
		notification.Attachments,
		notification.UserIds,
		notification.Status,
		notification.SentAt,
		notification.SentCount,
		notification.ErrorCount,
		notification.Id,
	)
	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("notification not found")
	}

	return nil
}

func (p *PgStorage) GetNotification(id uuid.UUID) (*models.Notification, error) {
	ctx := context.Background()
	query := `SELECT id, message, group_filter, attachments, user_ids, status, created_at, sent_at, sent_count, error_count
	          FROM notifications WHERE id = $1`

	var notification models.Notification
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&notification.Id,
		&notification.Message,
		&notification.Group,
		&notification.Attachments,
		&notification.UserIds,
		&notification.Status,
		&notification.CreatedAt,
		&notification.SentAt,
		&notification.SentCount,
		&notification.ErrorCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notification, nil
}

func (p *PgStorage) GetAllNotifications() ([]*models.Notification, error) {
	ctx := context.Background()
	query := `SELECT id, message, group_filter, attachments, user_ids, status, created_at, sent_at, sent_count, error_count
	          FROM notifications ORDER BY created_at DESC`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all notifications: %w", err)
	}
	defer rows.Close()

	return p.scanNotifications(rows)
}

func (p *PgStorage) scanNotifications(rows pgx.Rows) ([]*models.Notification, error) {
	var notifications []*models.Notification
	for rows.Next() {
		var notification models.Notification
		err := rows.Scan(
			&notification.Id,
			&notification.Message,
			&notification.Group,
			&notification.Attachments,
			&notification.UserIds,
			&notification.Status,
			&notification.CreatedAt,
			&notification.SentAt,
			&notification.SentCount,
			&notification.ErrorCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, &notification)
	}
	return notifications, nil
}

// ==================== МЕТОДЫ ДЛЯ БИБЛИОТЕКИ ВЛОЖЕНИЙ ====================

func (p *PgStorage) AddAttachment(attachment *models.Attachment) error {
	ctx := context.Background()
	query := `INSERT INTO attachments (id, filename, store_path, mime_type, size, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := p.pool.Exec(ctx, query,
		attachment.Id,
		attachment.Filename,
		attachment.StorePath,
		attachment.MimeType,
		attachment.Size,
		attachment.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add attachment: %w", err)
	}
	return nil
}

func (p *PgStorage) GetAttachment(id uuid.UUID) (*models.Attachment, error) {
	ctx := context.Background()
	query := `SELECT id, filename, store_path, mime_type, size, created_at
	          FROM attachments WHERE id = $1`

	var attachment models.Attachment
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&attachment.Id,
		&attachment.Filename,
		&attachment.StorePath,
		&attachment.MimeType,
		&attachment.Size,
		&attachment.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("attachment not found")
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}
	return &attachment, nil
}

func (p *PgStorage) GetAllAttachments() ([]*models.Attachment, error) {
	ctx := context.Background()
	query := `SELECT id, filename, store_path, mime_type, size, created_at
	          FROM attachments ORDER BY created_at DESC`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query attachments: %w", err)
	}
	defer rows.Close()

	var attachments []*models.Attachment
	for rows.Next() {
		var attachment models.Attachment
		err := rows.Scan(
			&attachment.Id,
			&attachment.Filename,
			&attachment.StorePath,
			&attachment.MimeType,
			&attachment.Size,
			&attachment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, &attachment)
	}
	return attachments, nil
}

func (p *PgStorage) UpdateAttachment(attachment *models.Attachment) error {
	ctx := context.Background()
	query := `UPDATE attachments SET filename = $2 WHERE id = $1`

	result, err := p.pool.Exec(ctx, query, attachment.Id, attachment.Filename)
	if err != nil {
		return fmt.Errorf("failed to update attachment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("attachment not found")
	}
	return nil
}

func (p *PgStorage) DeleteAttachment(id uuid.UUID) error {
	ctx := context.Background()
	query := `DELETE FROM attachments WHERE id = $1`

	result, err := p.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("attachment not found")
	}
	return nil
}
