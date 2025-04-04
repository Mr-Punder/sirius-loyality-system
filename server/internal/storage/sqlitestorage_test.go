package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Путь к тестовой базе данных SQLite
const testDBPath = "/tmp/loyality_system_test.db"

// getMigrationsPath возвращает абсолютный путь к миграциям SQLite
func getMigrationsPath(t *testing.T) string {
	// Получаем текущую директорию
	currentDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")

	// Формируем путь к миграциям
	migrationsPath := filepath.Join(currentDir, "..", "..", "migrations", "sqlite")

	// Проверяем, что директория существует
	_, err = os.Stat(migrationsPath)
	require.NoError(t, err, "Migrations directory does not exist: "+migrationsPath)

	return migrationsPath
}

// TestSQLiteStorage_Integration тестирует интеграцию с SQLite
func TestSQLiteStorage_Integration(t *testing.T) {
	// Пропускаем тест, если установлена переменная окружения SKIP_SQLITE_TESTS
	if os.Getenv("SKIP_SQLITE_TESTS") == "true" {
		t.Skip("Skipping SQLite integration tests")
	}

	// Удаляем тестовую базу данных, если она существует
	os.Remove(testDBPath)

	// Получаем путь к миграциям
	migrationsPath := getMigrationsPath(t)

	// Создаем новое хранилище SQLite
	store, err := NewSQLiteStorage(testDBPath, migrationsPath)
	require.NoError(t, err, "Failed to create SQLite storage")
	defer func() {
		store.Close()
		os.Remove(testDBPath)
	}()

	// Очищаем таблицы перед тестами
	err = store.CleanupTables(context.Background())
	require.NoError(t, err, "Failed to cleanup tables")

	// Запускаем тесты
	t.Run("TestAddAndGetUser", func(t *testing.T) {
		testAddAndGetUser(t, store)
	})

	t.Run("TestUpdateUser", func(t *testing.T) {
		testUpdateUser(t, store)
	})

	t.Run("TestDeleteUser", func(t *testing.T) {
		testDeleteUser(t, store)
	})

	t.Run("TestAddAndGetCode", func(t *testing.T) {
		testAddAndGetCode(t, store)
	})

	t.Run("TestUpdateCode", func(t *testing.T) {
		testUpdateCode(t, store)
	})

	t.Run("TestDeleteCode", func(t *testing.T) {
		testDeleteCode(t, store)
	})

	t.Run("TestAddAndGetTransaction", func(t *testing.T) {
		testAddAndGetTransaction(t, store)
	})

	t.Run("TestAddAndGetCodeUsage", func(t *testing.T) {
		testAddAndGetCodeUsage(t, store)
	})

	t.Run("TestUpdateCodeUsage", func(t *testing.T) {
		testUpdateCodeUsage(t, store)
	})
}

// testAddAndGetUser тестирует добавление и получение пользователя
func testAddAndGetUser(t *testing.T, store Storage) {
	// Создаем тестового пользователя
	userID := uuid.New()
	user := &models.User{
		Id:               userID,
		Telegramm:        "test_user",
		FirstName:        "Test",
		LastName:         "User",
		MiddleName:       "Testovich",
		Points:           100,
		Group:            "test_group",
		RegistrationTime: time.Now().UTC().Truncate(time.Second),
		Deleted:          false,
	}

	// Добавляем пользователя
	err := store.AddUser(user)
	require.NoError(t, err, "Failed to add user")

	// Получаем пользователя по ID
	retrievedUser, err := store.GetUser(userID)
	require.NoError(t, err, "Failed to get user")
	assert.Equal(t, user.Id, retrievedUser.Id, "User ID should match")
	assert.Equal(t, user.Telegramm, retrievedUser.Telegramm, "User Telegramm should match")
	assert.Equal(t, user.FirstName, retrievedUser.FirstName, "User FirstName should match")
	assert.Equal(t, user.LastName, retrievedUser.LastName, "User LastName should match")
	assert.Equal(t, user.MiddleName, retrievedUser.MiddleName, "User MiddleName should match")
	assert.Equal(t, user.Points, retrievedUser.Points, "User Points should match")
	assert.Equal(t, user.Group, retrievedUser.Group, "User Group should match")
	assert.Equal(t, user.RegistrationTime.Unix(), retrievedUser.RegistrationTime.Unix(), "User RegistrationTime should match")
	assert.Equal(t, user.Deleted, retrievedUser.Deleted, "User Deleted should match")

	// Получаем пользователя по Telegramm
	retrievedUserByTelegramm, err := store.GetUserByTelegramm("test_user")
	require.NoError(t, err, "Failed to get user by Telegramm")
	assert.Equal(t, user.Id, retrievedUserByTelegramm.Id, "User ID should match")

	// Получаем всех пользователей
	users, err := store.GetAllUsers()
	require.NoError(t, err, "Failed to get all users")
	assert.Len(t, users, 1, "Should have 1 user")
	assert.Equal(t, user.Id, users[0].Id, "User ID should match")

	// Получаем баллы пользователя
	points, err := store.GetUserPoints(userID)
	require.NoError(t, err, "Failed to get user points")
	assert.Equal(t, 100, points, "User points should match")
}

// testUpdateUser тестирует обновление пользователя
func testUpdateUser(t *testing.T, store Storage) {
	// Создаем тестового пользователя
	userID := uuid.New()
	user := &models.User{
		Id:               userID,
		Telegramm:        "update_user",
		FirstName:        "Update",
		LastName:         "User",
		MiddleName:       "Updateovich",
		Points:           100,
		Group:            "update_group",
		RegistrationTime: time.Now().UTC().Truncate(time.Second),
		Deleted:          false,
	}

	// Добавляем пользователя
	err := store.AddUser(user)
	require.NoError(t, err, "Failed to add user")

	// Обновляем пользователя
	user.FirstName = "Updated"
	user.LastName = "UserUpdated"
	user.Points = 200
	err = store.UpdateUser(user)
	require.NoError(t, err, "Failed to update user")

	// Получаем обновленного пользователя
	retrievedUser, err := store.GetUser(userID)
	require.NoError(t, err, "Failed to get updated user")
	assert.Equal(t, "Updated", retrievedUser.FirstName, "User FirstName should be updated")
	assert.Equal(t, "UserUpdated", retrievedUser.LastName, "User LastName should be updated")
	assert.Equal(t, 200, retrievedUser.Points, "User Points should be updated")
}

// testDeleteUser тестирует удаление пользователя
func testDeleteUser(t *testing.T, store Storage) {
	// Создаем тестового пользователя
	userID := uuid.New()
	user := &models.User{
		Id:               userID,
		Telegramm:        "delete_user",
		FirstName:        "Delete",
		LastName:         "User",
		MiddleName:       "Deleteovich",
		Points:           100,
		Group:            "delete_group",
		RegistrationTime: time.Now().UTC().Truncate(time.Second),
		Deleted:          false,
	}

	// Добавляем пользователя
	err := store.AddUser(user)
	require.NoError(t, err, "Failed to add user")

	// Удаляем пользователя
	err = store.DeleteUser(userID)
	require.NoError(t, err, "Failed to delete user")

	// Пытаемся получить удаленного пользователя
	_, err = store.GetUser(userID)
	assert.Error(t, err, "Should not be able to get deleted user")
}

// testAddAndGetCode тестирует добавление и получение кода
func testAddAndGetCode(t *testing.T, store Storage) {
	// Создаем тестовый код
	codeID := uuid.New()
	code := &models.Code{
		Code:         codeID,
		Amount:       100,
		PerUser:      1,
		Total:        10,
		AppliedCount: 0,
		IsActive:     true,
		Group:        "test_group",
		ErrorCode:    models.ErrorCodeNone,
	}

	// Добавляем код
	err := store.AddCode(code)
	require.NoError(t, err, "Failed to add code")

	// Получаем код
	retrievedCode, err := store.GetCodeInfo(codeID)
	require.NoError(t, err, "Failed to get code")
	assert.Equal(t, code.Code, retrievedCode.Code, "Code ID should match")
	assert.Equal(t, code.Amount, retrievedCode.Amount, "Code Amount should match")
	assert.Equal(t, code.PerUser, retrievedCode.PerUser, "Code PerUser should match")
	assert.Equal(t, code.Total, retrievedCode.Total, "Code Total should match")
	assert.Equal(t, code.AppliedCount, retrievedCode.AppliedCount, "Code AppliedCount should match")
	assert.Equal(t, code.IsActive, retrievedCode.IsActive, "Code IsActive should match")
	assert.Equal(t, code.Group, retrievedCode.Group, "Code Group should match")
	assert.Equal(t, code.ErrorCode, retrievedCode.ErrorCode, "Code ErrorCode should match")

	// Получаем все коды
	codes, err := store.GetAllCodes()
	require.NoError(t, err, "Failed to get all codes")
	assert.Len(t, codes, 1, "Should have 1 code")
	assert.Equal(t, code.Code, codes[0].Code, "Code ID should match")
}

// testUpdateCode тестирует обновление кода
func testUpdateCode(t *testing.T, store Storage) {
	// Создаем тестовый код
	codeID := uuid.New()
	code := &models.Code{
		Code:         codeID,
		Amount:       100,
		PerUser:      1,
		Total:        10,
		AppliedCount: 0,
		IsActive:     true,
		Group:        "update_group",
		ErrorCode:    models.ErrorCodeNone,
	}

	// Добавляем код
	err := store.AddCode(code)
	require.NoError(t, err, "Failed to add code")

	// Обновляем код
	code.Amount = 200
	code.PerUser = 2
	code.Total = 20
	code.AppliedCount = 1
	err = store.UpdateCode(code)
	require.NoError(t, err, "Failed to update code")

	// Получаем обновленный код
	retrievedCode, err := store.GetCodeInfo(codeID)
	require.NoError(t, err, "Failed to get updated code")
	assert.Equal(t, 200, retrievedCode.Amount, "Code Amount should be updated")
	assert.Equal(t, 2, retrievedCode.PerUser, "Code PerUser should be updated")
	assert.Equal(t, 20, retrievedCode.Total, "Code Total should be updated")
	assert.Equal(t, 1, retrievedCode.AppliedCount, "Code AppliedCount should be updated")
}

// testDeleteCode тестирует деактивацию кода
func testDeleteCode(t *testing.T, store Storage) {
	// Создаем тестовый код
	codeID := uuid.New()
	code := &models.Code{
		Code:         codeID,
		Amount:       100,
		PerUser:      1,
		Total:        10,
		AppliedCount: 0,
		IsActive:     true,
		Group:        "delete_group",
		ErrorCode:    models.ErrorCodeNone,
	}

	// Добавляем код
	err := store.AddCode(code)
	require.NoError(t, err, "Failed to add code")

	// Деактивируем код
	err = store.DeleteCode(codeID)
	require.NoError(t, err, "Failed to deactivate code")

	// Получаем деактивированный код
	retrievedCode, err := store.GetCodeInfo(codeID)
	require.NoError(t, err, "Failed to get deactivated code")
	assert.False(t, retrievedCode.IsActive, "Code should be deactivated")
}

// testAddAndGetTransaction тестирует добавление и получение транзакции
func testAddAndGetTransaction(t *testing.T, store Storage) {
	// Создаем тестового пользователя
	userID := uuid.New()
	user := &models.User{
		Id:               userID,
		Telegramm:        "transaction_user",
		FirstName:        "Transaction",
		LastName:         "User",
		MiddleName:       "Transactionovich",
		Points:           100,
		Group:            "transaction_group",
		RegistrationTime: time.Now().UTC().Truncate(time.Second),
		Deleted:          false,
	}

	// Добавляем пользователя
	err := store.AddUser(user)
	require.NoError(t, err, "Failed to add user")

	// Создаем тестовую транзакцию
	transactionID := uuid.New()
	codeID := uuid.New()
	transaction := &models.Transaction{
		Id:     transactionID,
		UserId: userID,
		Code:   codeID,
		Diff:   50,
		Time:   time.Now().UTC().Truncate(time.Second),
	}

	// Добавляем код для транзакции
	code := &models.Code{
		Code:         codeID,
		Amount:       50,
		PerUser:      1,
		Total:        10,
		AppliedCount: 0,
		IsActive:     true,
		Group:        "transaction_group",
		ErrorCode:    models.ErrorCodeNone,
	}
	err = store.AddCode(code)
	require.NoError(t, err, "Failed to add code for transaction")

	// Добавляем транзакцию
	err = store.AddTransaction(transaction)
	require.NoError(t, err, "Failed to add transaction")

	// Получаем транзакцию
	retrievedTransaction, err := store.GetTransaction(transactionID)
	require.NoError(t, err, "Failed to get transaction")
	assert.Equal(t, transaction.Id, retrievedTransaction.Id, "Transaction ID should match")
	assert.Equal(t, transaction.UserId, retrievedTransaction.UserId, "Transaction UserId should match")
	assert.Equal(t, transaction.Code, retrievedTransaction.Code, "Transaction Code should match")
	assert.Equal(t, transaction.Diff, retrievedTransaction.Diff, "Transaction Diff should match")
	assert.Equal(t, transaction.Time.Unix(), retrievedTransaction.Time.Unix(), "Transaction Time should match")

	// Получаем транзакции пользователя
	userTransactions, err := store.GetUserTransactions(userID)
	require.NoError(t, err, "Failed to get user transactions")
	assert.Len(t, userTransactions, 1, "Should have 1 transaction")
	assert.Equal(t, transaction.Id, userTransactions[0].Id, "Transaction ID should match")

	// Получаем все транзакции
	allTransactions, err := store.GetAllTransactions()
	require.NoError(t, err, "Failed to get all transactions")
	assert.GreaterOrEqual(t, len(allTransactions), 1, "Should have at least 1 transaction")

	// Проверяем, что баллы пользователя обновились
	updatedPoints, err := store.GetUserPoints(userID)
	require.NoError(t, err, "Failed to get updated user points")
	assert.Equal(t, 150, updatedPoints, "User points should be updated")
}

// testAddAndGetCodeUsage тестирует добавление и получение использования кода
func testAddAndGetCodeUsage(t *testing.T, store Storage) {
	// Создаем тестового пользователя
	userID := uuid.New()
	user := &models.User{
		Id:               userID,
		Telegramm:        "usage_user",
		FirstName:        "Usage",
		LastName:         "User",
		MiddleName:       "Usageovich",
		Points:           100,
		Group:            "usage_group",
		RegistrationTime: time.Now().UTC().Truncate(time.Second),
		Deleted:          false,
	}

	// Добавляем пользователя
	err := store.AddUser(user)
	require.NoError(t, err, "Failed to add user")

	// Создаем тестовый код
	codeID := uuid.New()
	code := &models.Code{
		Code:         codeID,
		Amount:       100,
		PerUser:      2,
		Total:        10,
		AppliedCount: 0,
		IsActive:     true,
		Group:        "usage_group",
		ErrorCode:    models.ErrorCodeNone,
	}

	// Добавляем код
	err = store.AddCode(code)
	require.NoError(t, err, "Failed to add code")

	// Создаем тестовое использование кода
	usageID := uuid.New()
	usage := &models.CodeUsage{
		Id:     usageID,
		Code:   codeID,
		UserId: userID,
		Count:  1,
	}

	// Добавляем использование кода
	err = store.AddCodeUsage(usage)
	require.NoError(t, err, "Failed to add code usage")

	// Получаем использование кода
	retrievedUsage, err := store.GetCodeUsageByUser(codeID, userID)
	require.NoError(t, err, "Failed to get code usage")
	assert.Equal(t, usage.Code, retrievedUsage.Code, "CodeUsage Code should match")
	assert.Equal(t, usage.UserId, retrievedUsage.UserId, "CodeUsage UserId should match")
	assert.Equal(t, usage.Count, retrievedUsage.Count, "CodeUsage Count should match")

	// Получаем использования кода
	codeUsages, err := store.GetCodeUsage(codeID)
	require.NoError(t, err, "Failed to get code usages")
	assert.Len(t, codeUsages, 1, "Should have 1 code usage")
	assert.Equal(t, usage.Code, codeUsages[0].Code, "CodeUsage Code should match")

	// Получаем все использования кодов
	allUsages, err := store.GetAllCodeUsages()
	require.NoError(t, err, "Failed to get all code usages")
	assert.GreaterOrEqual(t, len(allUsages), 1, "Should have at least 1 code usage")

	// Проверяем, что счетчик использований кода обновился
	updatedCode, err := store.GetCodeInfo(codeID)
	require.NoError(t, err, "Failed to get updated code")
	assert.Equal(t, 1, updatedCode.AppliedCount, "Code AppliedCount should be updated")
}

// testUpdateCodeUsage тестирует обновление использования кода
func testUpdateCodeUsage(t *testing.T, store Storage) {
	// Создаем тестового пользователя
	userID := uuid.New()
	user := &models.User{
		Id:               userID,
		Telegramm:        "update_usage_user",
		FirstName:        "UpdateUsage",
		LastName:         "User",
		MiddleName:       "UpdateUsageovich",
		Points:           100,
		Group:            "update_usage_group",
		RegistrationTime: time.Now().UTC().Truncate(time.Second),
		Deleted:          false,
	}

	// Добавляем пользователя
	err := store.AddUser(user)
	require.NoError(t, err, "Failed to add user")

	// Создаем тестовый код
	codeID := uuid.New()
	code := &models.Code{
		Code:         codeID,
		Amount:       100,
		PerUser:      2,
		Total:        10,
		AppliedCount: 0,
		IsActive:     true,
		Group:        "update_usage_group",
		ErrorCode:    models.ErrorCodeNone,
	}

	// Добавляем код
	err = store.AddCode(code)
	require.NoError(t, err, "Failed to add code")

	// Создаем тестовое использование кода
	usageID := uuid.New()
	usage := &models.CodeUsage{
		Id:     usageID,
		Code:   codeID,
		UserId: userID,
		Count:  1,
	}

	// Добавляем использование кода
	err = store.AddCodeUsage(usage)
	require.NoError(t, err, "Failed to add code usage")

	// Получаем использование кода
	retrievedUsage, err := store.GetCodeUsageByUser(codeID, userID)
	require.NoError(t, err, "Failed to get code usage")

	// Обновляем использование кода
	retrievedUsage.Count = 2
	err = store.UpdateCodeUsage(retrievedUsage)
	require.NoError(t, err, "Failed to update code usage")

	// Получаем обновленное использование кода
	updatedUsage, err := store.GetCodeUsageByUser(codeID, userID)
	require.NoError(t, err, "Failed to get updated code usage")
	assert.Equal(t, 2, updatedUsage.Count, "CodeUsage Count should be updated")
}
