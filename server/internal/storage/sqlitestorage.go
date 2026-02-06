package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage реализует интерфейс Storage с хранением данных в SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage создает новое хранилище SQLite
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	// Создаем директорию для базы данных, если она не существует
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory for database: %w", err)
	}

	// Подключение к базе данных с улучшенными параметрами для конкурентного доступа
	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_timeout=5000&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Проверка соединения
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	// Создаем хранилище
	storage := &SQLiteStorage{
		db: db,
	}

	return storage, nil
}

// Close закрывает соединение с базой данных
func (s *SQLiteStorage) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

// ==================== МЕТОДЫ ДЛЯ ПОЛЬЗОВАТЕЛЕЙ ====================

// GetUser возвращает пользователя по ID
func (s *SQLiteStorage) GetUser(userId uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, "group", registration_time, deleted
		FROM users
		WHERE id = ? AND deleted = 0
	`

	var user models.User
	var registrationTimeStr string
	err := s.db.QueryRow(query, userId).Scan(
		&user.Id,
		&user.Telegramm,
		&user.FirstName,
		&user.LastName,
		&user.MiddleName,
		&user.Group,
		&registrationTimeStr,
		&user.Deleted,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	registrationTime, err := time.Parse(time.RFC3339, registrationTimeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registration time: %w", err)
	}
	user.RegistrationTime = registrationTime

	return &user, nil
}

// GetUserByTelegramm возвращает пользователя по Telegram ID
func (s *SQLiteStorage) GetUserByTelegramm(telegramm string) (*models.User, error) {
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, "group", registration_time, deleted
		FROM users
		WHERE telegramm = ? AND deleted = 0
	`

	var user models.User
	var registrationTimeStr string
	err := s.db.QueryRow(query, telegramm).Scan(
		&user.Id,
		&user.Telegramm,
		&user.FirstName,
		&user.LastName,
		&user.MiddleName,
		&user.Group,
		&registrationTimeStr,
		&user.Deleted,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user by telegramm: %w", err)
	}

	registrationTime, err := time.Parse(time.RFC3339, registrationTimeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registration time: %w", err)
	}
	user.RegistrationTime = registrationTime

	return &user, nil
}

// GetAllUsers возвращает список всех пользователей
func (s *SQLiteStorage) GetAllUsers() ([]*models.User, error) {
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, "group", registration_time, deleted
		FROM users
		WHERE deleted = 0
		ORDER BY registration_time DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var registrationTimeStr string
		err := rows.Scan(
			&user.Id,
			&user.Telegramm,
			&user.FirstName,
			&user.LastName,
			&user.MiddleName,
			&user.Group,
			&registrationTimeStr,
			&user.Deleted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		registrationTime, err := time.Parse(time.RFC3339, registrationTimeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse registration time: %w", err)
		}
		user.RegistrationTime = registrationTime

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// AddUser добавляет нового пользователя
func (s *SQLiteStorage) AddUser(user *models.User) error {
	checkQuery := `SELECT COUNT(*) FROM users WHERE telegramm = ? AND deleted = 0`
	var count int
	err := s.db.QueryRow(checkQuery, user.Telegramm).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	if count > 0 {
		return errors.New("user with this telegram ID already exists")
	}

	query := `
		INSERT INTO users (id, telegramm, first_name, last_name, middle_name, "group", registration_time, deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	return s.execWithRetry(query,
		user.Id,
		user.Telegramm,
		user.FirstName,
		user.LastName,
		user.MiddleName,
		user.Group,
		user.RegistrationTime.Format(time.RFC3339),
		boolToInt(user.Deleted),
	)
}

// UpdateUser обновляет информацию о пользователе
func (s *SQLiteStorage) UpdateUser(user *models.User) error {
	checkQuery := `SELECT COUNT(*) FROM users WHERE id = ?`
	var count int
	err := s.db.QueryRow(checkQuery, user.Id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if count == 0 {
		return errors.New("user not found")
	}

	query := `
		UPDATE users
		SET telegramm = ?, first_name = ?, last_name = ?, middle_name = ?, "group" = ?, registration_time = ?, deleted = ?
		WHERE id = ?
	`

	return s.execWithRetry(query,
		user.Telegramm,
		user.FirstName,
		user.LastName,
		user.MiddleName,
		user.Group,
		user.RegistrationTime.Format(time.RFC3339),
		boolToInt(user.Deleted),
		user.Id,
	)
}

// DeleteUser удаляет пользователя из базы данных
func (s *SQLiteStorage) DeleteUser(userId uuid.UUID) error {
	checkQuery := `SELECT COUNT(*) FROM users WHERE id = ?`
	var count int
	err := s.db.QueryRow(checkQuery, userId).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if count == 0 {
		return errors.New("user not found")
	}

	// Сначала освобождаем детали пазлов, принадлежащие пользователю
	err = s.execWithRetry(`UPDATE puzzle_pieces SET owner_id = NULL, registered_at = NULL WHERE owner_id = ?`, userId)
	if err != nil {
		return fmt.Errorf("failed to release puzzle pieces: %w", err)
	}

	// Удаляем пользователя
	return s.execWithRetry(`DELETE FROM users WHERE id = ?`, userId)
}

// ==================== МЕТОДЫ ДЛЯ ПАЗЛОВ ====================

// GetPuzzle возвращает пазл по ID
func (s *SQLiteStorage) GetPuzzle(puzzleId int) (*models.Puzzle, error) {
	query := `SELECT id, COALESCE(name, ''), is_completed, completed_at FROM puzzles WHERE id = ?`

	var puzzle models.Puzzle
	var isCompleted int
	var completedAtStr sql.NullString

	err := s.db.QueryRow(query, puzzleId).Scan(&puzzle.Id, &puzzle.Name, &isCompleted, &completedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("puzzle not found")
		}
		return nil, fmt.Errorf("failed to get puzzle: %w", err)
	}

	puzzle.IsCompleted = isCompleted == 1
	if completedAtStr.Valid {
		completedAt, _ := time.Parse(time.RFC3339, completedAtStr.String)
		puzzle.CompletedAt = &completedAt
	}

	return &puzzle, nil
}

// GetAllPuzzles возвращает все пазлы
func (s *SQLiteStorage) GetAllPuzzles() ([]*models.Puzzle, error) {
	query := `SELECT id, COALESCE(name, ''), is_completed, completed_at FROM puzzles ORDER BY id`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query puzzles: %w", err)
	}
	defer rows.Close()

	var puzzles []*models.Puzzle
	for rows.Next() {
		var puzzle models.Puzzle
		var isCompleted int
		var completedAtStr sql.NullString

		err := rows.Scan(&puzzle.Id, &puzzle.Name, &isCompleted, &completedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan puzzle: %w", err)
		}

		puzzle.IsCompleted = isCompleted == 1
		if completedAtStr.Valid {
			completedAt, _ := time.Parse(time.RFC3339, completedAtStr.String)
			puzzle.CompletedAt = &completedAt
		}

		puzzles = append(puzzles, &puzzle)
	}

	return puzzles, nil
}

// UpdatePuzzle обновляет информацию о пазле
func (s *SQLiteStorage) UpdatePuzzle(puzzle *models.Puzzle) error {
	var completedAtStr interface{}
	if puzzle.CompletedAt != nil {
		completedAtStr = puzzle.CompletedAt.Format(time.RFC3339)
	} else {
		completedAtStr = nil
	}

	query := `UPDATE puzzles SET name = ?, is_completed = ?, completed_at = ? WHERE id = ?`
	return s.execWithRetry(query, puzzle.Name, boolToInt(puzzle.IsCompleted), completedAtStr, puzzle.Id)
}

// CompletePuzzle отмечает пазл как собранный и возвращает владельцев деталей для уведомления
func (s *SQLiteStorage) CompletePuzzle(puzzleId int) ([]*models.User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Проверяем, что пазл существует и еще не завершен
	var isCompleted int
	err = tx.QueryRow(`SELECT is_completed FROM puzzles WHERE id = ?`, puzzleId).Scan(&isCompleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("puzzle not found")
		}
		return nil, fmt.Errorf("failed to check puzzle: %w", err)
	}

	if isCompleted == 1 {
		return nil, errors.New("puzzle already completed")
	}

	// Отмечаем пазл как завершенный
	now := time.Now().Format(time.RFC3339)
	_, err = tx.Exec(`UPDATE puzzles SET is_completed = 1, completed_at = ? WHERE id = ?`, now, puzzleId)
	if err != nil {
		return nil, fmt.Errorf("failed to complete puzzle: %w", err)
	}

	// Получаем владельцев деталей этого пазла
	rows, err := tx.Query(`
		SELECT DISTINCT u.id, u.telegramm, u.first_name, u.last_name, u.middle_name, u."group", u.registration_time, u.deleted
		FROM puzzle_pieces pp
		JOIN users u ON pp.owner_id = u.id
		WHERE pp.puzzle_id = ? AND pp.owner_id IS NOT NULL
	`, puzzleId)
	if err != nil {
		return nil, fmt.Errorf("failed to get piece owners: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var deletedInt int
		var regTimeStr string

		err := rows.Scan(&user.Id, &user.Telegramm, &user.FirstName, &user.LastName, &user.MiddleName, &user.Group, &regTimeStr, &deletedInt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		user.Deleted = deletedInt == 1
		user.RegistrationTime, _ = time.Parse(time.RFC3339, regTimeStr)
		users = append(users, &user)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return users, nil
}

// ==================== МЕТОДЫ ДЛЯ ДЕТАЛЕЙ ПАЗЛОВ ====================

// GetPuzzlePiece возвращает деталь по коду
func (s *SQLiteStorage) GetPuzzlePiece(code string) (*models.PuzzlePiece, error) {
	query := `SELECT code, puzzle_id, piece_number, owner_id, registered_at FROM puzzle_pieces WHERE code = ?`

	var piece models.PuzzlePiece
	var ownerIdStr sql.NullString
	var registeredAtStr sql.NullString

	err := s.db.QueryRow(query, code).Scan(
		&piece.Code,
		&piece.PuzzleId,
		&piece.PieceNumber,
		&ownerIdStr,
		&registeredAtStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("piece not found")
		}
		return nil, fmt.Errorf("failed to get puzzle piece: %w", err)
	}

	if ownerIdStr.Valid {
		ownerId, _ := uuid.Parse(ownerIdStr.String)
		piece.OwnerId = &ownerId
	}
	if registeredAtStr.Valid {
		registeredAt, _ := time.Parse(time.RFC3339, registeredAtStr.String)
		piece.RegisteredAt = &registeredAt
	}

	return &piece, nil
}

// GetPuzzlePiecesByPuzzle возвращает все детали пазла
func (s *SQLiteStorage) GetPuzzlePiecesByPuzzle(puzzleId int) ([]*models.PuzzlePiece, error) {
	query := `SELECT code, puzzle_id, piece_number, owner_id, registered_at FROM puzzle_pieces WHERE puzzle_id = ? ORDER BY piece_number`

	rows, err := s.db.Query(query, puzzleId)
	if err != nil {
		return nil, fmt.Errorf("failed to query puzzle pieces: %w", err)
	}
	defer rows.Close()

	return s.scanPuzzlePieces(rows)
}

// GetPuzzlePiecesByOwner возвращает все детали пользователя
func (s *SQLiteStorage) GetPuzzlePiecesByOwner(ownerId uuid.UUID) ([]*models.PuzzlePiece, error) {
	query := `SELECT code, puzzle_id, piece_number, owner_id, registered_at FROM puzzle_pieces WHERE owner_id = ? ORDER BY puzzle_id, piece_number`

	rows, err := s.db.Query(query, ownerId)
	if err != nil {
		return nil, fmt.Errorf("failed to query puzzle pieces by owner: %w", err)
	}
	defer rows.Close()

	return s.scanPuzzlePieces(rows)
}

// GetAllPuzzlePieces возвращает все детали
func (s *SQLiteStorage) GetAllPuzzlePieces() ([]*models.PuzzlePiece, error) {
	query := `SELECT code, puzzle_id, piece_number, owner_id, registered_at FROM puzzle_pieces ORDER BY puzzle_id, piece_number`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all puzzle pieces: %w", err)
	}
	defer rows.Close()

	return s.scanPuzzlePieces(rows)
}

// scanPuzzlePieces вспомогательная функция для сканирования деталей
func (s *SQLiteStorage) scanPuzzlePieces(rows *sql.Rows) ([]*models.PuzzlePiece, error) {
	var pieces []*models.PuzzlePiece
	for rows.Next() {
		var piece models.PuzzlePiece
		var ownerIdStr sql.NullString
		var registeredAtStr sql.NullString

		err := rows.Scan(
			&piece.Code,
			&piece.PuzzleId,
			&piece.PieceNumber,
			&ownerIdStr,
			&registeredAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan puzzle piece: %w", err)
		}

		if ownerIdStr.Valid {
			ownerId, _ := uuid.Parse(ownerIdStr.String)
			piece.OwnerId = &ownerId
		}
		if registeredAtStr.Valid {
			registeredAt, _ := time.Parse(time.RFC3339, registeredAtStr.String)
			piece.RegisteredAt = &registeredAt
		}

		pieces = append(pieces, &piece)
	}
	return pieces, nil
}

// AddPuzzlePiece добавляет одну деталь
func (s *SQLiteStorage) AddPuzzlePiece(piece *models.PuzzlePiece) error {
	query := `INSERT INTO puzzle_pieces (code, puzzle_id, piece_number, owner_id, registered_at) VALUES (?, ?, ?, ?, ?)`

	var ownerIdVal interface{}
	var registeredAtVal interface{}

	if piece.OwnerId != nil {
		ownerIdVal = piece.OwnerId.String()
	}
	if piece.RegisteredAt != nil {
		registeredAtVal = piece.RegisteredAt.Format(time.RFC3339)
	}

	return s.execWithRetry(query, piece.Code, piece.PuzzleId, piece.PieceNumber, ownerIdVal, registeredAtVal)
}

// AddPuzzlePieces добавляет несколько деталей
func (s *SQLiteStorage) AddPuzzlePieces(pieces []*models.PuzzlePiece) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `INSERT INTO puzzle_pieces (code, puzzle_id, piece_number, owner_id, registered_at) VALUES (?, ?, ?, ?, ?)`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, piece := range pieces {
		var ownerIdVal interface{}
		var registeredAtVal interface{}

		if piece.OwnerId != nil {
			ownerIdVal = piece.OwnerId.String()
		}
		if piece.RegisteredAt != nil {
			registeredAtVal = piece.RegisteredAt.Format(time.RFC3339)
		}

		_, err = stmt.Exec(piece.Code, piece.PuzzleId, piece.PieceNumber, ownerIdVal, registeredAtVal)
		if err != nil {
			return fmt.Errorf("failed to add puzzle piece %s: %w", piece.Code, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RegisterPuzzlePiece привязывает деталь к пользователю
// Возвращает: деталь, флаг "все детали пазла розданы", ошибку
func (s *SQLiteStorage) RegisterPuzzlePiece(code string, ownerId uuid.UUID) (*models.PuzzlePiece, bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Проверяем существование детали и что она еще не занята
	var existingOwnerId sql.NullString
	var puzzleId int
	var pieceNumber int
	checkQuery := `SELECT puzzle_id, piece_number, owner_id FROM puzzle_pieces WHERE code = ?`
	err = tx.QueryRow(checkQuery, code).Scan(&puzzleId, &pieceNumber, &existingOwnerId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, errors.New("piece not found")
		}
		return nil, false, fmt.Errorf("failed to check piece: %w", err)
	}

	if existingOwnerId.Valid {
		return nil, false, errors.New("piece already taken")
	}

	// Привязываем деталь к пользователю
	now := time.Now()
	updateQuery := `UPDATE puzzle_pieces SET owner_id = ?, registered_at = ? WHERE code = ?`
	_, err = tx.Exec(updateQuery, ownerId.String(), now.Format(time.RFC3339), code)
	if err != nil {
		return nil, false, fmt.Errorf("failed to register piece: %w", err)
	}

	// Проверяем, все ли детали пазла розданы
	countQuery := `SELECT COUNT(*) FROM puzzle_pieces WHERE puzzle_id = ? AND owner_id IS NOT NULL`
	var count int
	err = tx.QueryRow(countQuery, puzzleId).Scan(&count)
	if err != nil {
		return nil, false, fmt.Errorf("failed to count pieces: %w", err)
	}

	allPiecesDistributed := count == 6

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Возвращаем обновленную деталь
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

// GetUserPieceCount возвращает количество деталей пользователя
func (s *SQLiteStorage) GetUserPieceCount(userId uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM puzzle_pieces WHERE owner_id = ?`
	var count int
	err := s.db.QueryRow(query, userId).Scan(&count)
	return count, err
}

// GetUserCompletedPuzzlePieceCount возвращает количество деталей пользователя из собранных пазлов
func (s *SQLiteStorage) GetUserCompletedPuzzlePieceCount(userId uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM puzzle_pieces pp
		JOIN puzzles p ON pp.puzzle_id = p.id
		WHERE pp.owner_id = ? AND p.is_completed = 1
	`
	var count int
	err := s.db.QueryRow(query, userId).Scan(&count)
	return count, err
}

// ==================== МЕТОДЫ ДЛЯ АДМИНИСТРАТОРОВ ====================

// GetAdmin возвращает администратора по ID
func (s *SQLiteStorage) GetAdmin(adminId int64) (*models.Admin, error) {
	query := `SELECT id, name, username, is_active FROM admins WHERE id = ?`

	var admin models.Admin
	var isActive int
	err := s.db.QueryRow(query, adminId).Scan(
		&admin.ID,
		&admin.Name,
		&admin.Username,
		&isActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("admin not found")
		}
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	admin.IsActive = isActive == 1
	return &admin, nil
}

// GetAllAdmins возвращает список всех администраторов
func (s *SQLiteStorage) GetAllAdmins() ([]*models.Admin, error) {
	query := `SELECT id, name, username, is_active FROM admins ORDER BY id`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query admins: %w", err)
	}
	defer rows.Close()

	var admins []*models.Admin
	for rows.Next() {
		var admin models.Admin
		var isActive int
		err := rows.Scan(&admin.ID, &admin.Name, &admin.Username, &isActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin: %w", err)
		}
		admin.IsActive = isActive == 1
		admins = append(admins, &admin)
	}

	return admins, nil
}

// AddAdmin добавляет нового администратора
func (s *SQLiteStorage) AddAdmin(admin *models.Admin) error {
	checkQuery := `SELECT COUNT(*) FROM admins WHERE id = ?`
	var count int
	err := s.db.QueryRow(checkQuery, admin.ID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing admin: %w", err)
	}

	if count > 0 {
		return errors.New("admin with this ID already exists")
	}

	query := `INSERT INTO admins (id, name, username, is_active) VALUES (?, ?, ?, ?)`
	return s.execWithRetry(query, admin.ID, admin.Name, admin.Username, boolToInt(admin.IsActive))
}

// UpdateAdmin обновляет информацию об администраторе
func (s *SQLiteStorage) UpdateAdmin(admin *models.Admin) error {
	checkQuery := `SELECT COUNT(*) FROM admins WHERE id = ?`
	var count int
	err := s.db.QueryRow(checkQuery, admin.ID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check admin existence: %w", err)
	}

	if count == 0 {
		return errors.New("admin not found")
	}

	query := `UPDATE admins SET name = ?, username = ?, is_active = ? WHERE id = ?`
	return s.execWithRetry(query, admin.Name, admin.Username, boolToInt(admin.IsActive), admin.ID)
}

// DeleteAdmin удаляет администратора
func (s *SQLiteStorage) DeleteAdmin(adminId int64) error {
	checkQuery := `SELECT COUNT(*) FROM admins WHERE id = ?`
	var count int
	err := s.db.QueryRow(checkQuery, adminId).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check admin existence: %w", err)
	}

	if count == 0 {
		return errors.New("admin not found")
	}

	query := `DELETE FROM admins WHERE id = ?`
	return s.execWithRetry(query, adminId)
}

// ==================== МЕТОДЫ ДЛЯ УВЕДОМЛЕНИЙ ====================

func (s *SQLiteStorage) AddNotification(notification *models.Notification) error {
	query := `INSERT INTO notifications (id, message, group_filter, attachments, user_ids, status, created_at, sent_at, sent_count, error_count)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	attachments := strings.Join(notification.Attachments, ",")

	// Сериализуем user_ids в строку через запятую
	var userIdsStr string
	if len(notification.UserIds) > 0 {
		ids := make([]string, len(notification.UserIds))
		for i, id := range notification.UserIds {
			ids[i] = id.String()
		}
		userIdsStr = strings.Join(ids, ",")
	}

	return s.execWithRetry(query,
		notification.Id.String(),
		notification.Message,
		notification.Group,
		attachments,
		userIdsStr,
		string(notification.Status),
		notification.CreatedAt.Format(time.RFC3339),
		formatNullableTime(notification.SentAt),
		notification.SentCount,
		notification.ErrorCount,
	)
}

func (s *SQLiteStorage) GetPendingNotifications() ([]*models.Notification, error) {
	query := `SELECT id, message, group_filter, attachments, user_ids, status, created_at, sent_at, sent_count, error_count
	          FROM notifications WHERE status = ? ORDER BY created_at`

	rows, err := s.db.Query(query, string(models.NotificationPending))
	if err != nil {
		return nil, fmt.Errorf("failed to query pending notifications: %w", err)
	}
	defer rows.Close()

	return s.scanNotifications(rows)
}

func (s *SQLiteStorage) UpdateNotification(notification *models.Notification) error {
	query := `UPDATE notifications SET message = ?, group_filter = ?, attachments = ?, user_ids = ?, status = ?,
	          sent_at = ?, sent_count = ?, error_count = ? WHERE id = ?`

	attachments := strings.Join(notification.Attachments, ",")

	// Сериализуем user_ids в строку через запятую
	var userIdsStr string
	if len(notification.UserIds) > 0 {
		ids := make([]string, len(notification.UserIds))
		for i, id := range notification.UserIds {
			ids[i] = id.String()
		}
		userIdsStr = strings.Join(ids, ",")
	}

	return s.execWithRetry(query,
		notification.Message,
		notification.Group,
		attachments,
		userIdsStr,
		string(notification.Status),
		formatNullableTime(notification.SentAt),
		notification.SentCount,
		notification.ErrorCount,
		notification.Id.String(),
	)
}

func (s *SQLiteStorage) GetNotification(id uuid.UUID) (*models.Notification, error) {
	query := `SELECT id, message, group_filter, attachments, user_ids, status, created_at, sent_at, sent_count, error_count
	          FROM notifications WHERE id = ?`

	var notification models.Notification
	var idStr, attachmentsStr, userIdsStr, statusStr, createdAtStr string
	var sentAtStr sql.NullString

	err := s.db.QueryRow(query, id.String()).Scan(
		&idStr,
		&notification.Message,
		&notification.Group,
		&attachmentsStr,
		&userIdsStr,
		&statusStr,
		&createdAtStr,
		&sentAtStr,
		&notification.SentCount,
		&notification.ErrorCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	notification.Id, _ = uuid.Parse(idStr)
	notification.Status = models.NotificationStatus(statusStr)
	notification.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	if sentAtStr.Valid && sentAtStr.String != "" {
		t, _ := time.Parse(time.RFC3339, sentAtStr.String)
		notification.SentAt = &t
	}
	if attachmentsStr != "" {
		notification.Attachments = strings.Split(attachmentsStr, ",")
	}
	if userIdsStr != "" {
		ids := strings.Split(userIdsStr, ",")
		for _, idStr := range ids {
			if uid, err := uuid.Parse(idStr); err == nil {
				notification.UserIds = append(notification.UserIds, uid)
			}
		}
	}

	return &notification, nil
}

func (s *SQLiteStorage) GetAllNotifications() ([]*models.Notification, error) {
	query := `SELECT id, message, group_filter, attachments, user_ids, status, created_at, sent_at, sent_count, error_count
	          FROM notifications ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all notifications: %w", err)
	}
	defer rows.Close()

	return s.scanNotifications(rows)
}

func (s *SQLiteStorage) scanNotifications(rows *sql.Rows) ([]*models.Notification, error) {
	var notifications []*models.Notification
	for rows.Next() {
		var notification models.Notification
		var idStr, attachmentsStr, userIdsStr, statusStr, createdAtStr string
		var sentAtStr sql.NullString

		err := rows.Scan(
			&idStr,
			&notification.Message,
			&notification.Group,
			&attachmentsStr,
			&userIdsStr,
			&statusStr,
			&createdAtStr,
			&sentAtStr,
			&notification.SentCount,
			&notification.ErrorCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		notification.Id, _ = uuid.Parse(idStr)
		notification.Status = models.NotificationStatus(statusStr)
		notification.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		if sentAtStr.Valid && sentAtStr.String != "" {
			t, _ := time.Parse(time.RFC3339, sentAtStr.String)
			notification.SentAt = &t
		}
		if attachmentsStr != "" {
			notification.Attachments = strings.Split(attachmentsStr, ",")
		}
		if userIdsStr != "" {
			ids := strings.Split(userIdsStr, ",")
			for _, idStr := range ids {
				if uid, err := uuid.Parse(idStr); err == nil {
					notification.UserIds = append(notification.UserIds, uid)
				}
			}
		}

		notifications = append(notifications, &notification)
	}
	return notifications, nil
}

func formatNullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ====================

// CleanupTables очищает все таблицы в базе данных (для тестов)
func (s *SQLiteStorage) CleanupTables(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM puzzle_pieces")
	if err != nil {
		return fmt.Errorf("failed to clean puzzle_pieces table: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM users")
	if err != nil {
		return fmt.Errorf("failed to clean users table: %w", err)
	}

	// Сбрасываем статус пазлов
	_, err = s.db.ExecContext(ctx, "UPDATE puzzles SET is_completed = 0, completed_at = NULL")
	if err != nil {
		return fmt.Errorf("failed to reset puzzles: %w", err)
	}

	return nil
}

// execWithRetry выполняет запрос с механизмом повторных попыток
func (s *SQLiteStorage) execWithRetry(query string, args ...interface{}) error {
	maxRetries := 5
	baseDelay := 100 * time.Millisecond

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		_, err := s.db.Exec(query, args...)
		if err == nil {
			return nil
		}

		lastErr = err

		if strings.Contains(err.Error(), "database is locked") {
			delay := baseDelay * time.Duration(1<<uint(i))
			jitter := time.Duration(rand.Int63n(int64(delay / 10)))
			time.Sleep(delay + jitter)
			continue
		}

		break
	}

	return fmt.Errorf("failed to execute query: %w", lastErr)
}

// boolToInt преобразует bool в int (для SQLite)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ==================== МЕТОДЫ ДЛЯ БИБЛИОТЕКИ ВЛОЖЕНИЙ ====================

func (s *SQLiteStorage) AddAttachment(attachment *models.Attachment) error {
	query := `INSERT INTO attachments (id, filename, store_path, mime_type, size, created_at)
	          VALUES (?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		attachment.Id.String(),
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

func (s *SQLiteStorage) GetAttachment(id uuid.UUID) (*models.Attachment, error) {
	query := `SELECT id, filename, store_path, mime_type, size, created_at
	          FROM attachments WHERE id = ?`

	var attachment models.Attachment
	var idStr string
	var createdAtStr string
	err := s.db.QueryRow(query, id.String()).Scan(
		&idStr,
		&attachment.Filename,
		&attachment.StorePath,
		&attachment.MimeType,
		&attachment.Size,
		&createdAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("attachment not found")
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}
	attachment.Id, _ = uuid.Parse(idStr)
	attachment.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	return &attachment, nil
}

func (s *SQLiteStorage) GetAllAttachments() ([]*models.Attachment, error) {
	query := `SELECT id, filename, store_path, mime_type, size, created_at
	          FROM attachments ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query attachments: %w", err)
	}
	defer rows.Close()

	var attachments []*models.Attachment
	for rows.Next() {
		var attachment models.Attachment
		var idStr string
		var createdAtStr string
		err := rows.Scan(
			&idStr,
			&attachment.Filename,
			&attachment.StorePath,
			&attachment.MimeType,
			&attachment.Size,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachment.Id, _ = uuid.Parse(idStr)
		attachment.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		attachments = append(attachments, &attachment)
	}
	return attachments, nil
}

func (s *SQLiteStorage) UpdateAttachment(attachment *models.Attachment) error {
	query := `UPDATE attachments SET filename = ? WHERE id = ?`

	result, err := s.db.Exec(query, attachment.Filename, attachment.Id.String())
	if err != nil {
		return fmt.Errorf("failed to update attachment: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("attachment not found")
	}
	return nil
}

func (s *SQLiteStorage) DeleteAttachment(id uuid.UUID) error {
	query := `DELETE FROM attachments WHERE id = ?`

	result, err := s.db.Exec(query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("attachment not found")
	}
	return nil
}
