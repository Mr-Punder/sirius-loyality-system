package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage реализует интерфейс Storage с хранением данных в SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage создает новое хранилище SQLite
func NewSQLiteStorage(dbPath string, migrationsPath string) (*SQLiteStorage, error) {
	// Создаем директорию для базы данных, если она не существует
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory for database: %w", err)
	}

	// Подключение к базе данных
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	// Проверка соединения
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	// Создаем хранилище
	storage := &SQLiteStorage{
		db: db,
	}

	// Применяем миграции, если указан путь
	if migrationsPath != "" {
		if err := storage.runMigrations(db, migrationsPath); err != nil {
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	return storage, nil
}

// runMigrations запускает миграции базы данных
func (s *SQLiteStorage) runMigrations(db *sql.DB, migrationsPath string) error {
	// Создаем драйвер для миграций
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Создаем экземпляр миграции
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file:%s", migrationsPath),
		"sqlite3", driver)
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
func (s *SQLiteStorage) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

// GetUser возвращает пользователя по ID
func (s *SQLiteStorage) GetUser(userId uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, points, "group", registration_time, deleted
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
		&user.Points,
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

	// Преобразуем строку времени в time.Time
	registrationTime, err := time.Parse(time.RFC3339, registrationTimeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registration time: %w", err)
	}
	user.RegistrationTime = registrationTime

	return &user, nil
}

// GetUserPoints возвращает количество баллов пользователя
func (s *SQLiteStorage) GetUserPoints(userId uuid.UUID) (int, error) {
	query := `
		SELECT points
		FROM users
		WHERE id = ? AND deleted = 0
	`

	var points int
	err := s.db.QueryRow(query, userId).Scan(&points)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("user not found")
		}
		return 0, fmt.Errorf("failed to get user points: %w", err)
	}

	return points, nil
}

// GetUserByTelegramm возвращает пользователя по Telegram ID
func (s *SQLiteStorage) GetUserByTelegramm(telegramm string) (*models.User, error) {
	query := `
		SELECT id, telegramm, first_name, last_name, middle_name, points, "group", registration_time, deleted
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
		&user.Points,
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

	// Преобразуем строку времени в time.Time
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
		SELECT id, telegramm, first_name, last_name, middle_name, points, "group", registration_time, deleted
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
			&user.Points,
			&user.Group,
			&registrationTimeStr,
			&user.Deleted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// Преобразуем строку времени в time.Time
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

// GetTransaction возвращает транзакцию по ID
func (s *SQLiteStorage) GetTransaction(transactionId uuid.UUID) (*models.Transaction, error) {
	query := `
		SELECT id, user_id, code, diff, time
		FROM transactions
		WHERE id = ?
	`

	var transaction models.Transaction
	var codeStr sql.NullString
	var timeStr string
	err := s.db.QueryRow(query, transactionId).Scan(
		&transaction.Id,
		&transaction.UserId,
		&codeStr,
		&transaction.Diff,
		&timeStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Если code не NULL, присваиваем значение
	if codeStr.Valid {
		code, err := uuid.Parse(codeStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse code UUID: %w", err)
		}
		transaction.Code = code
	}

	// Преобразуем строку времени в time.Time
	transactionTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction time: %w", err)
	}
	transaction.Time = transactionTime

	return &transaction, nil
}

// GetUserTransactions возвращает список транзакций пользователя
func (s *SQLiteStorage) GetUserTransactions(userId uuid.UUID) ([]*models.Transaction, error) {
	query := `
		SELECT id, user_id, code, diff, time
		FROM transactions
		WHERE user_id = ?
		ORDER BY time DESC
	`

	rows, err := s.db.Query(query, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to query user transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		var codeStr sql.NullString
		var timeStr string
		err := rows.Scan(
			&transaction.Id,
			&transaction.UserId,
			&codeStr,
			&transaction.Diff,
			&timeStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		// Если code не NULL, присваиваем значение
		if codeStr.Valid {
			code, err := uuid.Parse(codeStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse code UUID: %w", err)
			}
			transaction.Code = code
		}

		// Преобразуем строку времени в time.Time
		transactionTime, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse transaction time: %w", err)
		}
		transaction.Time = transactionTime

		transactions = append(transactions, &transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// GetAllTransactions возвращает список всех транзакций
func (s *SQLiteStorage) GetAllTransactions() ([]*models.Transaction, error) {
	query := `
		SELECT id, user_id, code, diff, time
		FROM transactions
		ORDER BY time DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		var transaction models.Transaction
		var codeStr sql.NullString
		var timeStr string
		err := rows.Scan(
			&transaction.Id,
			&transaction.UserId,
			&codeStr,
			&transaction.Diff,
			&timeStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		// Если code не NULL, присваиваем значение
		if codeStr.Valid {
			code, err := uuid.Parse(codeStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse code UUID: %w", err)
			}
			transaction.Code = code
		}

		// Преобразуем строку времени в time.Time
		transactionTime, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse transaction time: %w", err)
		}
		transaction.Time = transactionTime

		transactions = append(transactions, &transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// GetCodeInfo возвращает информацию о коде
func (s *SQLiteStorage) GetCodeInfo(code uuid.UUID) (*models.Code, error) {
	query := `
		SELECT code, amount, per_user, total, applied_count, is_active, "group", error_code
		FROM codes
		WHERE code = ?
	`

	var codeInfo models.Code
	var isActive int
	err := s.db.QueryRow(query, code).Scan(
		&codeInfo.Code,
		&codeInfo.Amount,
		&codeInfo.PerUser,
		&codeInfo.Total,
		&codeInfo.AppliedCount,
		&isActive,
		&codeInfo.Group,
		&codeInfo.ErrorCode,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("code not found")
		}
		return nil, fmt.Errorf("failed to get code info: %w", err)
	}

	codeInfo.IsActive = isActive == 1

	return &codeInfo, nil
}

// GetAllCodes возвращает список всех кодов
func (s *SQLiteStorage) GetAllCodes() ([]*models.Code, error) {
	query := `
		SELECT code, amount, per_user, total, applied_count, is_active, "group", error_code
		FROM codes
		ORDER BY is_active DESC, applied_count DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query codes: %w", err)
	}
	defer rows.Close()

	var codes []*models.Code
	for rows.Next() {
		var code models.Code
		var isActive int
		err := rows.Scan(
			&code.Code,
			&code.Amount,
			&code.PerUser,
			&code.Total,
			&code.AppliedCount,
			&isActive,
			&code.Group,
			&code.ErrorCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan code: %w", err)
		}
		code.IsActive = isActive == 1
		codes = append(codes, &code)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating codes: %w", err)
	}

	return codes, nil
}

// GetCodeUsage возвращает список использований кода
func (s *SQLiteStorage) GetCodeUsage(code uuid.UUID) ([]*models.CodeUsage, error) {
	query := `
		SELECT id, code, user_id, count
		FROM code_usages
		WHERE code = ?
	`

	rows, err := s.db.Query(query, code)
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
func (s *SQLiteStorage) GetAllCodeUsages() ([]*models.CodeUsage, error) {
	query := `
		SELECT id, code, user_id, count
		FROM code_usages
	`

	rows, err := s.db.Query(query)
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
func (s *SQLiteStorage) GetCodeUsageByUser(code uuid.UUID, userId uuid.UUID) (*models.CodeUsage, error) {
	query := `
		SELECT id, code, user_id, count
		FROM code_usages
		WHERE code = ? AND user_id = ?
	`

	var usage models.CodeUsage
	err := s.db.QueryRow(query, code, userId).Scan(
		&usage.Id,
		&usage.Code,
		&usage.UserId,
		&usage.Count,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("code usage not found")
		}
		return nil, fmt.Errorf("failed to get code usage by user: %w", err)
	}

	return &usage, nil
}

// AddUser добавляет нового пользователя
func (s *SQLiteStorage) AddUser(user *models.User) error {
	// Проверяем, существует ли пользователь с таким Telegram ID
	checkQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE telegramm = ? AND deleted = 0
	`
	var count int
	err := s.db.QueryRow(checkQuery, user.Telegramm).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	if count > 0 {
		return errors.New("user with this telegram ID already exists")
	}

	// Добавляем пользователя
	query := `
		INSERT INTO users (id, telegramm, first_name, last_name, middle_name, points, "group", registration_time, deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		user.Id,
		user.Telegramm,
		user.FirstName,
		user.LastName,
		user.MiddleName,
		user.Points,
		user.Group,
		user.RegistrationTime.Format(time.RFC3339),
		boolToInt(user.Deleted),
	)

	if err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	return nil
}

// AddTransaction добавляет новую транзакцию
func (s *SQLiteStorage) AddTransaction(transaction *models.Transaction) error {
	// Начинаем транзакцию
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Проверяем, существует ли пользователь
	checkUserQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE id = ? AND deleted = 0
	`
	var count int
	err = tx.QueryRow(checkUserQuery, transaction.UserId).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if count == 0 {
		return errors.New("user not found")
	}

	// Добавляем транзакцию
	insertQuery := `
		INSERT INTO transactions (id, user_id, code, diff, time)
		VALUES (?, ?, ?, ?, ?)
	`

	var codeValue interface{}
	if transaction.Code == uuid.Nil {
		codeValue = nil
	} else {
		codeValue = transaction.Code
	}

	_, err = tx.Exec(insertQuery,
		transaction.Id,
		transaction.UserId,
		codeValue,
		transaction.Diff,
		transaction.Time.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to add transaction: %w", err)
	}

	// Обновляем баллы пользователя
	updateUserQuery := `
		UPDATE users
		SET points = points + ?
		WHERE id = ?
	`

	_, err = tx.Exec(updateUserQuery, transaction.Diff, transaction.UserId)
	if err != nil {
		return fmt.Errorf("failed to update user points: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// AddCode добавляет новый код
func (s *SQLiteStorage) AddCode(code *models.Code) error {
	// Проверяем, существует ли код с таким ID
	checkQuery := `
		SELECT COUNT(*)
		FROM codes
		WHERE code = ?
	`
	var count int
	err := s.db.QueryRow(checkQuery, code.Code).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing code: %w", err)
	}

	if count > 0 {
		return errors.New("code already exists")
	}

	// Добавляем код
	query := `
		INSERT INTO codes (code, amount, per_user, total, applied_count, is_active, "group", error_code)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		code.Code,
		code.Amount,
		code.PerUser,
		code.Total,
		code.AppliedCount,
		boolToInt(code.IsActive),
		code.Group,
		code.ErrorCode,
	)

	if err != nil {
		return fmt.Errorf("failed to add code: %w", err)
	}

	return nil
}

// AddCodeUsage добавляет использование кода
func (s *SQLiteStorage) AddCodeUsage(usage *models.CodeUsage) error {
	// Начинаем транзакцию
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Проверяем, существует ли пользователь
	checkUserQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE id = ? AND deleted = 0
	`
	var userCount int
	err = tx.QueryRow(checkUserQuery, usage.UserId).Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if userCount == 0 {
		return errors.New("user not found")
	}

	// Проверяем, существует ли код
	var code models.Code
	var isActive int
	checkCodeQuery := `
		SELECT code, amount, per_user, total, applied_count, is_active, "group", error_code
		FROM codes
		WHERE code = ?
	`
	err = tx.QueryRow(checkCodeQuery, usage.Code).Scan(
		&code.Code,
		&code.Amount,
		&code.PerUser,
		&code.Total,
		&code.AppliedCount,
		&isActive,
		&code.Group,
		&code.ErrorCode,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("code not found")
		}
		return fmt.Errorf("failed to get code info: %w", err)
	}
	code.IsActive = isActive == 1

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
			WHERE id = ?
		`
		err = tx.QueryRow(getUserGroupQuery, usage.UserId).Scan(&userGroup)
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
		WHERE code = ? AND user_id = ?
	`
	row := tx.QueryRow(checkUsageQuery, usage.Code, usage.UserId)
	var usageExists bool
	existingUsage = &models.CodeUsage{}
	err = row.Scan(
		&existingUsage.Id,
		&existingUsage.Code,
		&existingUsage.UserId,
		&existingUsage.Count,
	)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
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
			WHERE id = ?
		`
		_, err = tx.Exec(updateUsageQuery, existingUsage.Id)
		if err != nil {
			return fmt.Errorf("failed to update code usage: %w", err)
		}
	} else {
		// Иначе создаем новое использование
		insertUsageQuery := `
			INSERT INTO code_usages (id, code, user_id, count)
			VALUES (?, ?, ?, ?)
		`
		_, err = tx.Exec(insertUsageQuery, usage.Id, usage.Code, usage.UserId, usage.Count)
		if err != nil {
			return fmt.Errorf("failed to add code usage: %w", err)
		}
	}

	// Увеличиваем счетчик использований кода
	updateCodeQuery := `
		UPDATE codes
		SET applied_count = applied_count + 1
		WHERE code = ?
	`
	_, err = tx.Exec(updateCodeQuery, code.Code)
	if err != nil {
		return fmt.Errorf("failed to update code applied count: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateUser обновляет информацию о пользователе
func (s *SQLiteStorage) UpdateUser(user *models.User) error {
	// Проверяем, существует ли пользователь
	checkQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE id = ?
	`
	var count int
	err := s.db.QueryRow(checkQuery, user.Id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if count == 0 {
		return errors.New("user not found")
	}

	// Обновляем пользователя
	query := `
		UPDATE users
		SET telegramm = ?, first_name = ?, last_name = ?, middle_name = ?, points = ?, "group" = ?, registration_time = ?, deleted = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(query,
		user.Telegramm,
		user.FirstName,
		user.LastName,
		user.MiddleName,
		user.Points,
		user.Group,
		user.RegistrationTime.Format(time.RFC3339),
		boolToInt(user.Deleted),
		user.Id,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateCode обновляет информацию о коде
func (s *SQLiteStorage) UpdateCode(code *models.Code) error {
	// Проверяем, существует ли код
	checkQuery := `
		SELECT COUNT(*)
		FROM codes
		WHERE code = ?
	`
	var count int
	err := s.db.QueryRow(checkQuery, code.Code).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check code existence: %w", err)
	}

	if count == 0 {
		return errors.New("code not found")
	}

	// Обновляем код
	query := `
		UPDATE codes
		SET amount = ?, per_user = ?, total = ?, applied_count = ?, is_active = ?, "group" = ?, error_code = ?
		WHERE code = ?
	`

	_, err = s.db.Exec(query,
		code.Amount,
		code.PerUser,
		code.Total,
		code.AppliedCount,
		boolToInt(code.IsActive),
		code.Group,
		code.ErrorCode,
		code.Code,
	)

	if err != nil {
		return fmt.Errorf("failed to update code: %w", err)
	}

	return nil
}

// UpdateCodeUsage обновляет информацию об использовании кода
func (s *SQLiteStorage) UpdateCodeUsage(usage *models.CodeUsage) error {
	// Проверяем, существует ли использование кода
	checkQuery := `
		SELECT COUNT(*)
		FROM code_usages
		WHERE id = ?
	`
	var count int
	err := s.db.QueryRow(checkQuery, usage.Id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check code usage existence: %w", err)
	}

	if count == 0 {
		return errors.New("code usage not found")
	}

	// Обновляем использование кода
	query := `
		UPDATE code_usages
		SET code = ?, user_id = ?, count = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(query,
		usage.Code,
		usage.UserId,
		usage.Count,
		usage.Id,
	)

	if err != nil {
		return fmt.Errorf("failed to update code usage: %w", err)
	}

	return nil
}

// DeleteUser помечает пользователя как удаленного (мягкое удаление)
func (s *SQLiteStorage) DeleteUser(userId uuid.UUID) error {
	// Проверяем, существует ли пользователь
	checkQuery := `
		SELECT COUNT(*)
		FROM users
		WHERE id = ? AND deleted = 0
	`
	var count int
	err := s.db.QueryRow(checkQuery, userId).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	if count == 0 {
		return errors.New("user not found")
	}

	// Помечаем пользователя как удаленного
	query := `
		UPDATE users
		SET deleted = 1
		WHERE id = ?
	`

	_, err = s.db.Exec(query, userId)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// DeleteCode деактивирует код
func (s *SQLiteStorage) DeleteCode(code uuid.UUID) error {
	// Проверяем, существует ли код
	checkQuery := `
		SELECT COUNT(*)
		FROM codes
		WHERE code = ?
	`
	var count int
	err := s.db.QueryRow(checkQuery, code).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check code existence: %w", err)
	}

	if count == 0 {
		return errors.New("code not found")
	}

	// Деактивируем код
	query := `
		UPDATE codes
		SET is_active = 0
		WHERE code = ?
	`

	_, err = s.db.Exec(query, code)
	if err != nil {
		return fmt.Errorf("failed to deactivate code: %w", err)
	}

	return nil
}

// CleanupTables очищает все таблицы в базе данных (для тестов)
func (s *SQLiteStorage) CleanupTables(ctx context.Context) error {
	// Очищаем таблицы в правильном порядке из-за внешних ключей
	_, err := s.db.ExecContext(ctx, "DELETE FROM transactions")
	if err != nil {
		return fmt.Errorf("failed to clean transactions table: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM code_usages")
	if err != nil {
		return fmt.Errorf("failed to clean code_usages table: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM codes")
	if err != nil {
		return fmt.Errorf("failed to clean codes table: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "DELETE FROM users")
	if err != nil {
		return fmt.Errorf("failed to clean users table: %w", err)
	}

	return nil
}

// boolToInt преобразует bool в int (для SQLite)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
