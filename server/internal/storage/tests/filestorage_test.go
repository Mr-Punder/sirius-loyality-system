package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilestorage(t *testing.T) {
	// Создаем директорию для тестов
	testDir := filepath.Join(os.TempDir(), "filestorage-test")
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Создаем файловое хранилище
	fs, err := storage.NewFilestorage(testDir)
	require.NoError(t, err)

	// Тестируем работу с пользователями
	t.Run("Users", func(t *testing.T) {
		// Создаем пользователя
		user := &models.User{
			Id:               uuid.New(),
			Telegramm:        "test_user",
			FirstName:        "Test",
			LastName:         "User",
			MiddleName:       "Middle",
			Points:           0,
			Group:            "Test Group",
			RegistrationTime: models.GetCurrentTime(),
			Deleted:          false,
		}

		// Добавляем пользователя
		err := fs.AddUser(user)
		require.NoError(t, err)

		// Проверяем, что файл с пользователями создан
		_, err = os.Stat(filepath.Join(testDir, storage.UsersFileName))
		require.NoError(t, err)

		// Получаем пользователя
		retrievedUser, err := fs.GetUser(user.Id)
		require.NoError(t, err)
		assert.Equal(t, user.Id, retrievedUser.Id)
		assert.Equal(t, user.Telegramm, retrievedUser.Telegramm)
		assert.Equal(t, user.FirstName, retrievedUser.FirstName)
		assert.Equal(t, user.LastName, retrievedUser.LastName)
		assert.Equal(t, user.MiddleName, retrievedUser.MiddleName)
		assert.Equal(t, user.Points, retrievedUser.Points)
		assert.Equal(t, user.Group, retrievedUser.Group)
		assert.Equal(t, user.Deleted, retrievedUser.Deleted)

		// Обновляем пользователя
		user.FirstName = "Updated"
		user.LastName = "Name"
		err = fs.UpdateUser(user)
		require.NoError(t, err)

		// Получаем обновленного пользователя
		retrievedUser, err = fs.GetUser(user.Id)
		require.NoError(t, err)
		assert.Equal(t, "Updated", retrievedUser.FirstName)
		assert.Equal(t, "Name", retrievedUser.LastName)

		// Удаляем пользователя (мягкое удаление)
		err = fs.DeleteUser(user.Id)
		require.NoError(t, err)

		// Проверяем, что пользователь помечен как удаленный
		retrievedUser, err = fs.GetUser(user.Id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")

		// Получаем всех пользователей
		users, err := fs.GetAllUsers()
		require.NoError(t, err)
		assert.Equal(t, 0, len(users))
	})

	// Тестируем работу с QR-кодами
	t.Run("Codes", func(t *testing.T) {
		// Создаем код
		code := &models.Code{
			Code:         uuid.New(),
			Amount:       100,
			PerUser:      1,
			Total:        10,
			AppliedCount: 0,
			IsActive:     true,
			Group:        "Test Group",
			ErrorCode:    models.ErrorCodeNone,
		}

		// Добавляем код
		err := fs.AddCode(code)
		require.NoError(t, err)

		// Проверяем, что файл с кодами создан
		_, err = os.Stat(filepath.Join(testDir, storage.CodesFileName))
		require.NoError(t, err)

		// Получаем код
		retrievedCode, err := fs.GetCodeInfo(code.Code)
		require.NoError(t, err)
		assert.Equal(t, code.Code, retrievedCode.Code)
		assert.Equal(t, code.Amount, retrievedCode.Amount)
		assert.Equal(t, code.PerUser, retrievedCode.PerUser)
		assert.Equal(t, code.Total, retrievedCode.Total)
		assert.Equal(t, code.AppliedCount, retrievedCode.AppliedCount)
		assert.Equal(t, code.IsActive, retrievedCode.IsActive)
		assert.Equal(t, code.Group, retrievedCode.Group)
		assert.Equal(t, code.ErrorCode, retrievedCode.ErrorCode)

		// Обновляем код
		code.Amount = 200
		code.PerUser = 2
		code.Total = 20
		code.Group = "Updated Group"
		err = fs.UpdateCode(code)
		require.NoError(t, err)

		// Получаем обновленный код
		retrievedCode, err = fs.GetCodeInfo(code.Code)
		require.NoError(t, err)
		assert.Equal(t, 200, retrievedCode.Amount)
		assert.Equal(t, 2, retrievedCode.PerUser)
		assert.Equal(t, 20, retrievedCode.Total)
		assert.Equal(t, "Updated Group", retrievedCode.Group)

		// Деактивируем код
		err = fs.DeleteCode(code.Code)
		require.NoError(t, err)

		// Проверяем, что код деактивирован
		retrievedCode, err = fs.GetCodeInfo(code.Code)
		require.NoError(t, err)
		assert.False(t, retrievedCode.IsActive)

		// Получаем все коды
		codes, err := fs.GetAllCodes()
		require.NoError(t, err)
		assert.Equal(t, 1, len(codes))
	})

	// Тестируем применение QR-кодов с ограничениями
	t.Run("CodeUsage", func(t *testing.T) {
		// Создаем пользователей
		user1 := &models.User{
			Id:               uuid.New(),
			Telegramm:        "test_user1",
			FirstName:        "Test",
			LastName:         "User1",
			Points:           0,
			Group:            "Test Group",
			RegistrationTime: models.GetCurrentTime(),
			Deleted:          false,
		}

		user2 := &models.User{
			Id:               uuid.New(),
			Telegramm:        "test_user2",
			FirstName:        "Test",
			LastName:         "User2",
			Points:           0,
			Group:            "Another Group",
			RegistrationTime: models.GetCurrentTime(),
			Deleted:          false,
		}

		// Добавляем пользователей
		err := fs.AddUser(user1)
		require.NoError(t, err)
		err = fs.AddUser(user2)
		require.NoError(t, err)

		// Тест 1: Код с ограничением на количество использований одним пользователем
		t.Run("PerUserLimit", func(t *testing.T) {
			// Создаем код с ограничением per_user = 2, total = 0
			code := &models.Code{
				Code:         uuid.New(),
				Amount:       100,
				PerUser:      2,
				Total:        0,
				AppliedCount: 0,
				IsActive:     true,
				Group:        "",
				ErrorCode:    models.ErrorCodeNone,
			}

			// Добавляем код
			err := fs.AddCode(code)
			require.NoError(t, err)

			// Применяем код к пользователю 1 (первый раз)
			usage1 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user1.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage1)
			require.NoError(t, err)

			// Применяем код к пользователю 1 (второй раз)
			usage2 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user1.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage2)
			require.NoError(t, err)

			// Применяем код к пользователю 1 (третий раз) - должна быть ошибка
			usage3 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user1.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage3)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "user code usage limit exceeded")

			// Применяем код к пользователю 2 (первый раз)
			usage4 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user2.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage4)
			require.NoError(t, err)

			// Проверяем, что файл с использованиями кодов создан
			_, err = os.Stat(filepath.Join(testDir, storage.CodeUsagesFileName))
			require.NoError(t, err)

			// Получаем использования кода
			usages, err := fs.GetCodeUsage(code.Code)
			require.NoError(t, err)
			assert.Equal(t, 2, len(usages))

			// Получаем использование кода пользователем
			userUsage, err := fs.GetCodeUsageByUser(code.Code, user1.Id)
			require.NoError(t, err)
			assert.Equal(t, 2, userUsage.Count)
		})

		// Тест 2: Код с ограничением на общее количество использований
		t.Run("TotalLimit", func(t *testing.T) {
			// Создаем код с ограничением per_user = 0, total = 2
			code := &models.Code{
				Code:         uuid.New(),
				Amount:       100,
				PerUser:      0,
				Total:        2,
				AppliedCount: 0,
				IsActive:     true,
				Group:        "",
				ErrorCode:    models.ErrorCodeNone,
			}

			// Добавляем код
			err := fs.AddCode(code)
			require.NoError(t, err)

			// Применяем код к пользователю 1 (первый раз)
			usage1 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user1.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage1)
			require.NoError(t, err)

			// Применяем код к пользователю 2 (первый раз)
			usage2 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user2.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage2)
			require.NoError(t, err)

			// Применяем код к пользователю 1 (второй раз) - должна быть ошибка
			usage3 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user1.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage3)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "code usage limit exceeded")

			// Получаем код
			retrievedCode, err := fs.GetCodeInfo(code.Code)
			require.NoError(t, err)
			assert.Equal(t, 2, retrievedCode.AppliedCount)
		})

		// Тест 3: Код с ограничением на группу пользователей
		t.Run("GroupLimit", func(t *testing.T) {
			// Создаем код с ограничением на группу
			code := &models.Code{
				Code:         uuid.New(),
				Amount:       100,
				PerUser:      0,
				Total:        0,
				AppliedCount: 0,
				IsActive:     true,
				Group:        "Test Group",
				ErrorCode:    models.ErrorCodeNone,
			}

			// Добавляем код
			err := fs.AddCode(code)
			require.NoError(t, err)

			// Применяем код к пользователю 1 (из группы "Test Group")
			usage1 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user1.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage1)
			require.NoError(t, err)

			// Применяем код к пользователю 2 (из группы "Another Group") - должна быть ошибка
			usage2 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user2.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage2)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "user group does not match code group")
		})

		// Тест 4: Код с ограничением на количество использований одним пользователем и общее количество использований
		t.Run("PerUserAndTotalLimit", func(t *testing.T) {
			// Создаем код с ограничением per_user = 1, total = 2
			code := &models.Code{
				Code:         uuid.New(),
				Amount:       100,
				PerUser:      1,
				Total:        2,
				AppliedCount: 0,
				IsActive:     true,
				Group:        "",
				ErrorCode:    models.ErrorCodeNone,
			}

			// Добавляем код
			err := fs.AddCode(code)
			require.NoError(t, err)

			// Применяем код к пользователю 1 (первый раз)
			usage1 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user1.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage1)
			require.NoError(t, err)

			// Применяем код к пользователю 1 (второй раз) - должна быть ошибка
			usage2 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user1.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage2)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "user code usage limit exceeded")

			// Применяем код к пользователю 2 (первый раз)
			usage3 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user2.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage3)
			require.NoError(t, err)

			// Применяем код к пользователю 2 (второй раз) - должна быть ошибка
			usage4 := &models.CodeUsage{
				Id:     uuid.New(),
				Code:   code.Code,
				UserId: user2.Id,
				Count:  1,
			}
			err = fs.AddCodeUsage(usage4)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "user code usage limit exceeded")

			// Получаем код
			retrievedCode, err := fs.GetCodeInfo(code.Code)
			require.NoError(t, err)
			assert.Equal(t, 2, retrievedCode.AppliedCount)
		})
	})

	// Тестируем работу с транзакциями
	t.Run("Transactions", func(t *testing.T) {
		// Создаем пользователя
		user := &models.User{
			Id:               uuid.New(),
			Telegramm:        "test_user3",
			FirstName:        "Test",
			LastName:         "User3",
			Points:           0,
			Group:            "Test Group",
			RegistrationTime: models.GetCurrentTime(),
			Deleted:          false,
		}

		// Добавляем пользователя
		err := fs.AddUser(user)
		require.NoError(t, err)

		// Создаем транзакцию
		transaction := &models.Transaction{
			Id:     uuid.New(),
			UserId: user.Id,
			Diff:   100,
			Time:   models.GetCurrentTime(),
		}

		// Добавляем транзакцию
		err = fs.AddTransaction(transaction)
		require.NoError(t, err)

		// Проверяем, что файл с транзакциями создан
		_, err = os.Stat(filepath.Join(testDir, storage.TransactionsFileName))
		require.NoError(t, err)

		// Получаем транзакцию
		retrievedTransaction, err := fs.GetTransaction(transaction.Id)
		require.NoError(t, err)
		assert.Equal(t, transaction.Id, retrievedTransaction.Id)
		assert.Equal(t, transaction.UserId, retrievedTransaction.UserId)
		assert.Equal(t, transaction.Diff, retrievedTransaction.Diff)

		// Получаем транзакции пользователя
		transactions, err := fs.GetUserTransactions(user.Id)
		require.NoError(t, err)
		assert.Equal(t, 1, len(transactions))

		// Получаем все транзакции
		allTransactions, err := fs.GetAllTransactions()
		require.NoError(t, err)
		assert.Equal(t, 1, len(allTransactions))

		// Проверяем, что баллы пользователя обновились
		retrievedUser, err := fs.GetUser(user.Id)
		require.NoError(t, err)
		assert.Equal(t, 100, retrievedUser.Points)

		// Получаем баллы пользователя
		points, err := fs.GetUserPoints(user.Id)
		require.NoError(t, err)
		assert.Equal(t, 100, points)
	})
}
