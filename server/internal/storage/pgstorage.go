package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStorage реализует интерфейс Storage с хранением данных в PostgreSQL
type PgStorage struct {
	pool *pgxpool.Pool
}

// NewPgStorage создает новое хранилище PostgreSQL
func NewPgStorage(connString string, migrationsPath string) (*PgStorage, error) {
	// Подключение к базе данных
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	// Проверка соединения
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	// Создаем хранилище
	storage := &PgStorage{
		pool: pool,
	}

	// Применяем миграции, если указан путь
	if migrationsPath != "" {
		if err := storage.runMigrations(connString, migrationsPath); err != nil {
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	return storage, nil
}

// runMigrations запускает миграции базы данных
func (p *PgStorage) runMigrations(connString, migrationsPath string) error {
	// Создаем экземпляр миграции напрямую из строки подключения
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connString)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Запускаем миграции
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// Close закрывает соединение с базой данных
func (p *PgStorage) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// GetUser возвращает пользователя по ID
func (p *PgStorage) GetUser(userId uuid.UUID) (*models.User, error) {
	ctx := context.Background()
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, points, "group", registration_time, deleted
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
		&user.Points,
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

// GetUserPoints возвращает количество баллов пользователя
func (p *PgStorage) GetUserPoints(userId uuid.UUID) (int, error) {
	ctx := context.Background()
	query := `
		SELECT points
		FROM users
		WHERE id = $1 AND deleted = false
	`

	var points int
	err := p.pool.QueryRow(ctx, query, userId).Scan(&points)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("user not found")
		}
		return 0, fmt.Errorf("failed to get user points: %w", err)
	}

	return points, nil
}

// GetUserByTelegramm возвращает пользователя по Telegram ID
func (p *PgStorage) GetUserByTelegramm(telegramm string) (*models.User, error) {
	ctx := context.Background()
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, points, "group", registration_time, deleted
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
		&user.Points,
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

// GetAllUsers возвращает список всех пользователей
func (p *PgStorage) GetAllUsers() ([]*models.User, error) {
	ctx := context.Background()
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, points, "group", registration_time, deleted
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
			&user.Points,
			&user.Group,
			&user.RegistrationTime,
			&user.Deleted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// GetTransaction возвращает транзакцию по ID
func (p *PgStorage) GetTransaction(transactionId uuid.UUID) (*models.Transaction, error) {
	ctx := context.Background()
	query := `
		SELECT id, user_id, code, diff, "time"
		FROM transactions
		WHERE id = $1
	`

	var transaction models.Transaction
	var codeNullable *uuid.UUID // Для обработки NULL значений
	err := p.pool.QueryRow(ctx, query, transactionId).Scan(
		&transaction.Id,
		&transaction.UserId,
		&codeNullable,
		&transaction.Diff,
		&transaction.Time,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Если code не NULL, присваиваем значение
	if codeNullable != nil {
		transaction.Code = *codeNullable
	}

	return &transaction, nil
}

// GetUserTransactions возвращает список транзакций пользователя
func (p *PgStorage) GetUserTransactions(userId uuid.UUID) ([]*models.Transaction, error) {
	ctx := context.Background()
	query := `
		SELECT id, user_id, code, diff, "time"
		FROM transactions
		WHERE user_id = $1
		ORDER BY "time" DESC
	`

	rows, err := p.pool.Query(ctx, query, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to query user transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		var codeNullable *uuid.UUID // Для обработки NULL значений
		err := rows.Scan(
			&transaction.Id,
			&transaction.UserId,
			&codeNullable,
			&transaction.Diff,
			&transaction.Time,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		// Если code не NULL, присваиваем значение
		if codeNullable != nil {
			transaction.Code = *codeNullable
		}

		transactions = append(transactions, &transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// GetAllTransactions возвращает список всех транзакций
func (p *PgStorage) GetAllTransactions() ([]*models.Transaction, error) {
	ctx := context.Background()
	query := `
		SELECT id, user_id, code, diff, "time"
		FROM transactions
		ORDER BY "time" DESC
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		var codeNullable *uuid.UUID // Для обработки NULL значений
		err := rows.Scan(
			&transaction.Id,
			&transaction.UserId,
			&codeNullable,
			&transaction.Diff,
			&transaction.Time,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		// Если code не NULL, присваиваем значение
		if codeNullable != nil {
			transaction.Code = *codeNullable
		}

		transactions = append(transactions, &transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// GetCodeInfo возвращает информацию о коде
func (p *PgStorage) GetCodeInfo(code uuid.UUID) (*models.Code, error) {
	ctx := context.Background()
	query := `
		SELECT code, amount, per_user, total, applied_count, is_active, "group", error_code
		FROM codes
		WHERE code = $1
	`

	var codeInfo models.Code
	err := p.pool.QueryRow(ctx, query, code).Scan(
		&codeInfo.Code,
		&codeInfo.Amount,
		&codeInfo.PerUser,
		&codeInfo.Total,
		&codeInfo.AppliedCount,
		&codeInfo.IsActive,
		&codeInfo.Group,
		&codeInfo.ErrorCode,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("code not found")
		}
		return nil, fmt.Errorf("failed to get code info: %w", err)
	}

	return &codeInfo, nil
}

// GetAllCodes возвращает список всех кодов
func (p *PgStorage) GetAllCodes() ([]*models.Code, error) {
	ctx := context.Background()
	query := `
		SELECT code, amount, per_user, total, applied_count, is_active, "group", error_code
		FROM codes
		ORDER BY is_active DESC, applied_count DESC
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query codes: %w", err)
	}
	defer rows.Close()

	var codes []*models.Code
	for rows.Next() {
		var code models.Code
		err := rows.Scan(
			&code.Code,
			&code.Amount,
			&code.PerUser,
			&code.Total,
			&code.AppliedCount,
			&code.IsActive,
			&code.Group,
			&code.ErrorCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan code: %w", err)
		}
		codes = append(codes, &code)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating codes: %w", err)
	}

	return codes, nil
}

// GetCodeUsage возвращает список использований кода
func (p *PgStorage) GetCodeUsage(code uuid.UUID) ([]*models.CodeUsage, error) {
	ctx := context.Background()
	query := `
		SELECT id, code, user_id, count
		FROM code_usages
		WHERE code = $1
	`

	rows, err := p.pool.Query(ctx, query, code)
	if err != nil {
		return nil, fmt.Errorf("failed to query code usages: %w", err)
	}
	defer rows.Close()

	var usages []*models.CodeUsage
	for rows.Next() {
		var usage models.CodeUsage
		err := rows.Scan(
			&usage.Id,
			&usage.Code,
			&usage.UserId,
			&usage.Count,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan code usage: %w", err)
		}
		usages = append(usages, &usage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating code usages: %w", err)
	}

	return usages, nil
}

// GetAllCodeUsages возвращает список всех использований кодов
func (p *PgStorage) GetAllCodeUsages() ([]*models.CodeUsage, error) {
	ctx := context.Background()
	query := `
		SELECT id, code, user_id, count
		FROM code_usages
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all code usages: %w", err)
	}
	defer rows.Close()

	var usages []*models.CodeUsage
	for rows.Next() {
		var usage models.CodeUsage
		err := rows.Scan(
			&usage.Id,
			&usage.Code,
			&usage.UserId,
			&usage.Count,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan code usage: %w", err)
		}
		usages = append(usages, &usage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating code usages: %w", err)
	}

	return usages, nil
}

// GetCodeUsageByUser возвращает информацию об использовании кода пользователем
func (p *PgStorage) GetCodeUsageByUser(code uuid.UUID, userId uuid.UUID) (*models.CodeUsage, error) {
	ctx := context.Background()
	query := `
		SELECT id, code, user_id, count
		FROM code_usages
		WHERE code = $1 AND user_id = $2
	`

	var usage models.CodeUsage
	err := p.pool.QueryRow(ctx, query, code, userId).Scan(
		&usage.Id,
		&usage.Code,
		&usage.UserId,
		&usage.Count,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("code usage not found")
		}
		return nil, fmt.Errorf("failed to get code usage by user: %w", err)
	}

	return &usage, nil
}

// AddUser добавляет нового пользователя
func (p *PgStorage) AddUser(user *models.User) error {
	ctx := context.Background()

	// Проверяем, существует ли пользователь с таким Telegram ID
	checkQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE telegramm = $1 AND deleted = false
	`
	var count int
	err := p.pool.QueryRow(ctx, checkQuery, user.Telegramm).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	if count > 0 {
		return errors.New("user with this telegram ID already exists")
	}

	// Добавляем пользователя
	query := `
		INSERT INTO users (id, telegramm, first_name, last_name, middle_name, points, "group", registration_time, deleted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = p.pool.Exec(ctx, query,
		user.Id,
		user.Telegramm,
		user.FirstName,
		user.LastName,
		user.MiddleName,
		user.Points,
		user.Group,
		user.RegistrationTime,
		user.Deleted,
	)

	if err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	return nil
}

// AddTransaction добавляет новую транзакцию
func (p *PgStorage) AddTransaction(transaction *models.Transaction) error {
	ctx := context.Background()

	// Начинаем транзакцию
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Проверяем, существует ли пользователь
	checkUserQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE id = $1 AND deleted = false
	`
	var count int
	err = tx.QueryRow(ctx, checkUserQuery, transaction.UserId).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if count == 0 {
		return errors.New("user not found")
	}

	// Добавляем транзакцию
	insertQuery := `
		INSERT INTO transactions (id, user_id, code, diff, "time")
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = tx.Exec(ctx, insertQuery,
		transaction.Id,
		transaction.UserId,
		uuid.NullUUID{UUID: transaction.Code, Valid: transaction.Code != uuid.Nil},
		transaction.Diff,
		transaction.Time,
	)

	if err != nil {
		return fmt.Errorf("failed to add transaction: %w", err)
	}

	// Обновляем баллы пользователя
	updateUserQuery := `
		UPDATE users
		SET points = points + $1
		WHERE id = $2
	`

	_, err = tx.Exec(ctx, updateUserQuery, transaction.Diff, transaction.UserId)
	if err != nil {
		return fmt.Errorf("failed to update user points: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// AddCode добавляет новый код
func (p *PgStorage) AddCode(code *models.Code) error {
	ctx := context.Background()

	// Проверяем, существует ли код с таким ID
	checkQuery := `
		SELECT COUNT(*)
		FROM codes
		WHERE code = $1
	`
	var count int
	err := p.pool.QueryRow(ctx, checkQuery, code.Code).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing code: %w", err)
	}

	if count > 0 {
		return errors.New("code already exists")
	}

	// Добавляем код
	query := `
		INSERT INTO codes (code, amount, per_user, total, applied_count, is_active, "group", error_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = p.pool.Exec(ctx, query,
		code.Code,
		code.Amount,
		code.PerUser,
		code.Total,
		code.AppliedCount,
		code.IsActive,
		code.Group,
		code.ErrorCode,
	)

	if err != nil {
		return fmt.Errorf("failed to add code: %w", err)
	}

	return nil
}

// AddCodeUsage добавляет использование кода
func (p *PgStorage) AddCodeUsage(usage *models.CodeUsage) error {
	ctx := context.Background()

	// Начинаем транзакцию
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Проверяем, существует ли пользователь
	checkUserQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE id = $1 AND deleted = false
	`
	var userCount int
	err = tx.QueryRow(ctx, checkUserQuery, usage.UserId).Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if userCount == 0 {
		return errors.New("user not found")
	}

	// Проверяем, существует ли код
	var code models.Code
	checkCodeQuery := `
		SELECT code, amount, per_user, total, applied_count, is_active, "group", error_code
		FROM codes
		WHERE code = $1
	`
	err = tx.QueryRow(ctx, checkCodeQuery, usage.Code).Scan(
		&code.Code,
		&code.Amount,
		&code.PerUser,
		&code.Total,
		&code.AppliedCount,
		&code.IsActive,
		&code.Group,
		&code.ErrorCode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("code not found")
		}
		return fmt.Errorf("failed to get code info: %w", err)
	}

	// Проверяем, активен ли код
	if !code.IsActive {
		return errors.New("code is not active")
	}

	// Проверяем, принадлежит ли пользователь к нужной группе
	if code.Group != "" {
		var userGroup string
		getUserGroupQuery := `
			SELECT "group"
			FROM users
			WHERE id = $1
		`
		err = tx.QueryRow(ctx, getUserGroupQuery, usage.UserId).Scan(&userGroup)
		if err != nil {
			return fmt.Errorf("failed to get user group: %w", err)
		}

		if userGroup != code.Group {
			return errors.New("user group does not match code group")
		}
	}

	// Проверяем, не превышено ли количество использований кода пользователем
	var existingUsage *models.CodeUsage
	checkUsageQuery := `
		SELECT id, code, user_id, count
		FROM code_usages
		WHERE code = $1 AND user_id = $2
	`
	row := tx.QueryRow(ctx, checkUsageQuery, usage.Code, usage.UserId)
	var usageExists bool
	existingUsage = &models.CodeUsage{}
	err = row.Scan(
		&existingUsage.Id,
		&existingUsage.Code,
		&existingUsage.UserId,
		&existingUsage.Count,
	)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to check code usage: %w", err)
		}
		usageExists = false
	} else {
		usageExists = true
		if code.PerUser > 0 && existingUsage.Count >= code.PerUser {
			return errors.New("user code usage limit exceeded")
		}
	}

	// Проверяем, не превышено ли общее количество использований кода
	if code.Total > 0 && code.AppliedCount >= code.Total {
		return errors.New("code usage limit exceeded")
	}

	// Если использование кода пользователем уже существует, обновляем его
	if usageExists {
		updateUsageQuery := `
			UPDATE code_usages
			SET count = count + 1
			WHERE id = $1
		`
		_, err = tx.Exec(ctx, updateUsageQuery, existingUsage.Id)
		if err != nil {
			return fmt.Errorf("failed to update code usage: %w", err)
		}
	} else {
		// Иначе создаем новое использование
		insertUsageQuery := `
			INSERT INTO code_usages (id, code, user_id, count)
			VALUES ($1, $2, $3, $4)
		`
		_, err = tx.Exec(ctx, insertUsageQuery, usage.Id, usage.Code, usage.UserId, usage.Count)
		if err != nil {
			return fmt.Errorf("failed to add code usage: %w", err)
		}
	}

	// Увеличиваем счетчик использований кода
	updateCodeQuery := `
		UPDATE codes
		SET applied_count = applied_count + 1
		WHERE code = $1
	`
	_, err = tx.Exec(ctx, updateCodeQuery, code.Code)
	if err != nil {
		return fmt.Errorf("failed to update code applied count: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateUser обновляет информацию о пользователе
func (p *PgStorage) UpdateUser(user *models.User) error {
	ctx := context.Background()

	// Проверяем, существует ли пользователь
	checkQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE id = $1
	`
	var count int
	err := p.pool.QueryRow(ctx, checkQuery, user.Id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if count == 0 {
		return errors.New("user not found")
	}

	// Обновляем пользователя
	query := `
		UPDATE users
		SET telegramm = $2, first_name = $3, last_name = $4, middle_name = $5, points = $6, "group" = $7, registration_time = $8, deleted = $9
		WHERE id = $1
	`

	_, err = p.pool.Exec(ctx, query,
		user.Id,
		user.Telegramm,
		user.FirstName,
		user.LastName,
		user.MiddleName,
		user.Points,
		user.Group,
		user.RegistrationTime,
		user.Deleted,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateCode обновляет информацию о коде
func (p *PgStorage) UpdateCode(code *models.Code) error {
	ctx := context.Background()

	// Проверяем, существует ли код
	checkQuery := `
		SELECT COUNT(*)
		FROM codes
		WHERE code = $1
	`
	var count int
	err := p.pool.QueryRow(ctx, checkQuery, code.Code).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check code existence: %w", err)
	}

	if count == 0 {
		return errors.New("code not found")
	}

	// Обновляем код
	query := `
		UPDATE codes
		SET amount = $2, per_user = $3, total = $4, applied_count = $5, is_active = $6, "group" = $7, error_code = $8
		WHERE code = $1
	`

	_, err = p.pool.Exec(ctx, query,
		code.Code,
		code.Amount,
		code.PerUser,
		code.Total,
		code.AppliedCount,
		code.IsActive,
		code.Group,
		code.ErrorCode,
	)

	if err != nil {
		return fmt.Errorf("failed to update code: %w", err)
	}

	return nil
}

// UpdateCodeUsage обновляет информацию об использовании кода
func (p *PgStorage) UpdateCodeUsage(usage *models.CodeUsage) error {
	ctx := context.Background()

	// Проверяем, существует ли использование кода
	checkQuery := `
		SELECT COUNT(*)
		FROM code_usages
		WHERE id = $1
	`
	var count int
	err := p.pool.QueryRow(ctx, checkQuery, usage.Id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check code usage existence: %w", err)
	}

	if count == 0 {
		return errors.New("code usage not found")
	}

	// Обновляем использование кода
	query := `
		UPDATE code_usages
		SET code = $2, user_id = $3, count = $4
		WHERE id = $1
	`

	_, err = p.pool.Exec(ctx, query,
		usage.Id,
		usage.Code,
		usage.UserId,
		usage.Count,
	)

	if err != nil {
		return fmt.Errorf("failed to update code usage: %w", err)
	}

	return nil
}

// DeleteUser помечает пользователя как удаленного (мягкое удаление)
func (p *PgStorage) DeleteUser(userId uuid.UUID) error {
	ctx := context.Background()

	// Проверяем, существует ли пользователь
	checkQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE id = $1 AND deleted = false
	`
	var count int
	err := p.pool.QueryRow(ctx, checkQuery, userId).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if count == 0 {
		return errors.New("user not found")
	}

	// Помечаем пользователя как удаленного
	query := `
		UPDATE users
		SET deleted = true
		WHERE id = $1
	`

	_, err = p.pool.Exec(ctx, query, userId)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// DeleteCode деактивирует код
func (p *PgStorage) DeleteCode(code uuid.UUID) error {
	ctx := context.Background()

	// Проверяем, существует ли код
	checkQuery := `
		SELECT COUNT(*)
		FROM codes
		WHERE code = $1
	`
	var count int
	err := p.pool.QueryRow(ctx, checkQuery, code).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check code existence: %w", err)
	}

	if count == 0 {
		return errors.New("code not found")
	}

	// Деактивируем код
	query := `
		UPDATE codes
		SET is_active = false
		WHERE code = $1
	`

	_, err = p.pool.Exec(ctx, query, code)
	if err != nil {
		return fmt.Errorf("failed to deactivate code: %w", err)
	}

	return nil
}

// CleanupTables очищает все таблицы в базе данных (для тестов)
func (p *PgStorage) CleanupTables(ctx context.Context) error {
	// Очищаем таблицы в правильном порядке из-за внешних ключей
	_, err := p.pool.Exec(ctx, "DELETE FROM transactions")
	if err != nil {
		return fmt.Errorf("failed to clean transactions table: %w", err)
	}

	_, err = p.pool.Exec(ctx, "DELETE FROM code_usages")
	if err != nil {
		return fmt.Errorf("failed to clean code_usages table: %w", err)
	}

	_, err = p.pool.Exec(ctx, "DELETE FROM codes")
	if err != nil {
		return fmt.Errorf("failed to clean codes table: %w", err)
	}

	_, err = p.pool.Exec(ctx, "DELETE FROM users")
	if err != nil {
		return fmt.Errorf("failed to clean users table: %w", err)
	}

	return nil
}
