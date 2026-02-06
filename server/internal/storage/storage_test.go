package storage

import (
	"testing"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStorageImplementations тестирует все реализации Storage
func TestStorageImplementations(t *testing.T) {
	implementations := map[string]func() Storage{
		"memory": func() Storage {
			return NewMemstorage()
		},
	}

	for name, createStorage := range implementations {
		t.Run(name, func(t *testing.T) {
			t.Run("Users", func(t *testing.T) {
				testUserOperations(t, createStorage())
			})
			t.Run("Puzzles", func(t *testing.T) {
				testPuzzleOperations(t, createStorage())
			})
			t.Run("PuzzlePieces", func(t *testing.T) {
				testPuzzlePieceOperations(t, createStorage())
			})
			t.Run("RegisterPuzzlePiece", func(t *testing.T) {
				testRegisterPuzzlePiece(t, createStorage())
			})
			t.Run("PuzzleCompletion", func(t *testing.T) {
				testPuzzleCompletion(t, createStorage())
			})
			t.Run("DeleteUserReleasesPieces", func(t *testing.T) {
				testDeleteUserReleasesPieces(t, createStorage())
			})
			t.Run("Admins", func(t *testing.T) {
				testAdminOperations(t, createStorage())
			})
		})
	}
}

func testUserOperations(t *testing.T, store Storage) {
	// Создаем пользователя
	user := &models.User{
		Id:               uuid.New(),
		Telegramm:        "123456789",
		FirstName:        "Иван",
		LastName:         "Иванов",
		MiddleName:       "Иванович",
		Group:            "Н1",
		RegistrationTime: time.Now(),
		Deleted:          false,
	}

	// Добавляем пользователя
	err := store.AddUser(user)
	require.NoError(t, err)

	// Получаем пользователя по ID
	fetchedUser, err := store.GetUser(user.Id)
	require.NoError(t, err)
	assert.Equal(t, user.Id, fetchedUser.Id)
	assert.Equal(t, user.FirstName, fetchedUser.FirstName)
	assert.Equal(t, user.LastName, fetchedUser.LastName)
	assert.Equal(t, user.MiddleName, fetchedUser.MiddleName)
	assert.Equal(t, user.Group, fetchedUser.Group)

	// Получаем пользователя по Telegram
	fetchedUser, err = store.GetUserByTelegramm(user.Telegramm)
	require.NoError(t, err)
	assert.Equal(t, user.Id, fetchedUser.Id)

	// Обновляем пользователя
	user.FirstName = "Пётр"
	err = store.UpdateUser(user)
	require.NoError(t, err)

	fetchedUser, err = store.GetUser(user.Id)
	require.NoError(t, err)
	assert.Equal(t, "Пётр", fetchedUser.FirstName)

	// Получаем всех пользователей
	users, err := store.GetAllUsers()
	require.NoError(t, err)
	assert.Len(t, users, 1)

	// Удаляем пользователя
	err = store.DeleteUser(user.Id)
	require.NoError(t, err)

	// Проверяем, что пользователь удален
	_, err = store.GetUser(user.Id)
	assert.Error(t, err)

	// Проверяем, что можно создать пользователя с тем же Telegram после удаления
	newUser := &models.User{
		Id:               uuid.New(),
		Telegramm:        "123456789",
		FirstName:        "Новый",
		LastName:         "Пользователь",
		Group:            "Н2",
		RegistrationTime: time.Now(),
		Deleted:          false,
	}
	err = store.AddUser(newUser)
	require.NoError(t, err, "Должна быть возможность создать пользователя с тем же Telegram после удаления")
}

func testPuzzleOperations(t *testing.T, store Storage) {
	// Memstorage по умолчанию создает 30 пазлов

	// Получаем пазл
	puzzle, err := store.GetPuzzle(1)
	require.NoError(t, err)
	assert.Equal(t, 1, puzzle.Id)
	assert.False(t, puzzle.IsCompleted)

	// Получаем все пазлы
	puzzles, err := store.GetAllPuzzles()
	require.NoError(t, err)
	assert.Len(t, puzzles, 30) // Memstorage создает 30 пазлов по умолчанию

	// Обновляем пазл
	now := time.Now()
	puzzle.IsCompleted = true
	puzzle.CompletedAt = &now
	err = store.UpdatePuzzle(puzzle)
	require.NoError(t, err)

	// Проверяем обновление
	puzzle, err = store.GetPuzzle(1)
	require.NoError(t, err)
	assert.True(t, puzzle.IsCompleted)
	assert.NotNil(t, puzzle.CompletedAt)

	// Проверяем несуществующий пазл
	_, err = store.GetPuzzle(999)
	assert.Error(t, err)
}

func testPuzzlePieceOperations(t *testing.T, store Storage) {
	// Создаем пазл
	ms, ok := store.(*Memstorage)
	if ok {
		ms.puzzles.Store(1, &models.Puzzle{Id: 1, IsCompleted: false})
	}

	// Добавляем деталь
	piece := &models.PuzzlePiece{
		Code:        "ABC1234",
		PuzzleId:    1,
		PieceNumber: 1,
		OwnerId:     nil,
	}
	err := store.AddPuzzlePiece(piece)
	require.NoError(t, err)

	// Получаем деталь
	fetchedPiece, err := store.GetPuzzlePiece("ABC1234")
	require.NoError(t, err)
	assert.Equal(t, piece.Code, fetchedPiece.Code)
	assert.Equal(t, piece.PuzzleId, fetchedPiece.PuzzleId)
	assert.Nil(t, fetchedPiece.OwnerId)

	// Добавляем несколько деталей
	pieces := []*models.PuzzlePiece{
		{Code: "DEF5678", PuzzleId: 1, PieceNumber: 2},
		{Code: "GHI9012", PuzzleId: 1, PieceNumber: 3},
	}
	err = store.AddPuzzlePieces(pieces)
	require.NoError(t, err)

	// Получаем детали по пазлу
	puzzlePieces, err := store.GetPuzzlePiecesByPuzzle(1)
	require.NoError(t, err)
	assert.Len(t, puzzlePieces, 3)

	// Получаем все детали
	allPieces, err := store.GetAllPuzzlePieces()
	require.NoError(t, err)
	assert.Len(t, allPieces, 3)

	// Проверяем несуществующую деталь
	_, err = store.GetPuzzlePiece("NOTEXIST")
	assert.Error(t, err)
}

func testRegisterPuzzlePiece(t *testing.T, store Storage) {
	// Подготовка: создаем пазл и детали
	ms, ok := store.(*Memstorage)
	if ok {
		ms.puzzles.Store(10, &models.Puzzle{Id: 10, IsCompleted: false})
	}

	piece := &models.PuzzlePiece{
		Code:        "REG1234",
		PuzzleId:    10,
		PieceNumber: 1,
		OwnerId:     nil,
	}
	err := store.AddPuzzlePiece(piece)
	require.NoError(t, err)

	// Создаем пользователя
	user := &models.User{
		Id:               uuid.New(),
		Telegramm:        "reg_test_user",
		FirstName:        "Test",
		LastName:         "User",
		Group:            "Н1",
		RegistrationTime: time.Now(),
	}
	err = store.AddUser(user)
	require.NoError(t, err)

	// Регистрируем деталь
	registeredPiece, puzzleCompleted, err := store.RegisterPuzzlePiece("REG1234", user.Id)
	require.NoError(t, err)
	assert.NotNil(t, registeredPiece)
	assert.False(t, puzzleCompleted) // Пазл не завершен, так как только 1 деталь
	assert.NotNil(t, registeredPiece.OwnerId)
	assert.Equal(t, user.Id, *registeredPiece.OwnerId)
	assert.NotNil(t, registeredPiece.RegisteredAt)

	// Пытаемся зарегистрировать ту же деталь еще раз
	_, _, err = store.RegisterPuzzlePiece("REG1234", uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already taken")

	// Пытаемся зарегистрировать несуществующую деталь
	_, _, err = store.RegisterPuzzlePiece("NOTEXIST", user.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Проверяем количество деталей у пользователя
	count, err := store.GetUserPieceCount(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Проверяем детали по владельцу
	userPieces, err := store.GetPuzzlePiecesByOwner(user.Id)
	require.NoError(t, err)
	assert.Len(t, userPieces, 1)
}

func testPuzzleCompletion(t *testing.T, store Storage) {
	// Подготовка: создаем пазл
	ms, ok := store.(*Memstorage)
	if ok {
		ms.puzzles.Store(20, &models.Puzzle{Id: 20, Name: "Тестовый пазл", IsCompleted: false})
	}

	// Создаем 6 деталей для пазла
	for i := 1; i <= 6; i++ {
		piece := &models.PuzzlePiece{
			Code:        "COMP" + string(rune('A'+i-1)) + "00",
			PuzzleId:    20,
			PieceNumber: i,
		}
		err := store.AddPuzzlePiece(piece)
		require.NoError(t, err)
	}

	// Создаем пользователей и регистрируем детали
	var createdUsers []*models.User
	for i := 1; i <= 6; i++ {
		user := &models.User{
			Id:               uuid.New(),
			Telegramm:        "completion_user_" + string(rune('0'+i)),
			FirstName:        "User",
			LastName:         string(rune('A' + i - 1)),
			Group:            "Н1",
			RegistrationTime: time.Now(),
		}
		err := store.AddUser(user)
		require.NoError(t, err)
		createdUsers = append(createdUsers, user)

		code := "COMP" + string(rune('A'+i-1)) + "00"
		_, allDistributed, err := store.RegisterPuzzlePiece(code, user.Id)
		require.NoError(t, err)

		if i < 6 {
			assert.False(t, allDistributed, "Не все детали должны быть розданы после %d деталей", i)
		} else {
			assert.True(t, allDistributed, "Все детали должны быть розданы после 6 деталей")
		}
	}

	// Пазл еще не должен быть завершен (только все детали розданы)
	puzzle, err := store.GetPuzzle(20)
	require.NoError(t, err)
	assert.False(t, puzzle.IsCompleted, "Пазл не должен быть автоматически завершен")

	// Админ вручную завершает пазл
	users, err := store.CompletePuzzle(20)
	require.NoError(t, err)
	assert.Len(t, users, 6, "Должно быть 6 уникальных владельцев")

	// Теперь пазл должен быть завершен
	puzzle, err = store.GetPuzzle(20)
	require.NoError(t, err)
	assert.True(t, puzzle.IsCompleted)
	assert.NotNil(t, puzzle.CompletedAt)

	// Повторное завершение должно вернуть ошибку
	_, err = store.CompletePuzzle(20)
	assert.Error(t, err, "Повторное завершение должно вернуть ошибку")
}

func testDeleteUserReleasesPieces(t *testing.T, store Storage) {
	// Подготовка: создаем пазл и деталь
	ms, ok := store.(*Memstorage)
	if ok {
		ms.puzzles.Store(30, &models.Puzzle{Id: 30, IsCompleted: false})
	}

	piece := &models.PuzzlePiece{
		Code:        "DEL1234",
		PuzzleId:    30,
		PieceNumber: 1,
	}
	err := store.AddPuzzlePiece(piece)
	require.NoError(t, err)

	// Создаем пользователя
	user := &models.User{
		Id:               uuid.New(),
		Telegramm:        "delete_test_user",
		FirstName:        "Delete",
		LastName:         "Test",
		Group:            "Н1",
		RegistrationTime: time.Now(),
	}
	err = store.AddUser(user)
	require.NoError(t, err)

	// Регистрируем деталь
	_, _, err = store.RegisterPuzzlePiece("DEL1234", user.Id)
	require.NoError(t, err)

	// Проверяем, что деталь привязана
	fetchedPiece, err := store.GetPuzzlePiece("DEL1234")
	require.NoError(t, err)
	assert.NotNil(t, fetchedPiece.OwnerId)

	// Удаляем пользователя
	err = store.DeleteUser(user.Id)
	require.NoError(t, err)

	// Проверяем, что деталь освобождена
	fetchedPiece, err = store.GetPuzzlePiece("DEL1234")
	require.NoError(t, err)
	assert.Nil(t, fetchedPiece.OwnerId, "Деталь должна быть освобождена после удаления пользователя")
	assert.Nil(t, fetchedPiece.RegisteredAt)
}

func testAdminOperations(t *testing.T, store Storage) {
	// Создаем админа
	admin := &models.Admin{
		ID:       12345678,
		Name:     "Test Admin",
		Username: "testadmin",
		IsActive: true,
	}

	// Добавляем админа
	err := store.AddAdmin(admin)
	require.NoError(t, err)

	// Получаем админа
	fetchedAdmin, err := store.GetAdmin(admin.ID)
	require.NoError(t, err)
	assert.Equal(t, admin.ID, fetchedAdmin.ID)
	assert.Equal(t, admin.Name, fetchedAdmin.Name)
	assert.True(t, fetchedAdmin.IsActive)

	// Получаем всех админов
	admins, err := store.GetAllAdmins()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(admins), 1)

	// Обновляем админа
	admin.Name = "Updated Admin"
	admin.IsActive = false
	err = store.UpdateAdmin(admin)
	require.NoError(t, err)

	fetchedAdmin, err = store.GetAdmin(admin.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Admin", fetchedAdmin.Name)
	assert.False(t, fetchedAdmin.IsActive)

	// Удаляем админа
	err = store.DeleteAdmin(admin.ID)
	require.NoError(t, err)

	// Проверяем, что админ удален
	_, err = store.GetAdmin(admin.ID)
	assert.Error(t, err)

	// Проверяем, что нельзя добавить дубликат
	admin2 := &models.Admin{ID: 99999999, Name: "Admin 2", IsActive: true}
	err = store.AddAdmin(admin2)
	require.NoError(t, err)

	err = store.AddAdmin(admin2)
	assert.Error(t, err, "Должна быть ошибка при добавлении дубликата")
}

// TestDuplicateTelegramUser проверяет, что нельзя создать двух пользователей с одним Telegram
func TestDuplicateTelegramUser(t *testing.T) {
	store := NewMemstorage()

	user1 := &models.User{
		Id:               uuid.New(),
		Telegramm:        "duplicate_test",
		FirstName:        "User1",
		LastName:         "Test",
		Group:            "Н1",
		RegistrationTime: time.Now(),
	}

	user2 := &models.User{
		Id:               uuid.New(),
		Telegramm:        "duplicate_test",
		FirstName:        "User2",
		LastName:         "Test",
		Group:            "Н2",
		RegistrationTime: time.Now(),
	}

	err := store.AddUser(user1)
	require.NoError(t, err)

	err = store.AddUser(user2)
	assert.Error(t, err, "Должна быть ошибка при создании пользователя с существующим Telegram")
}

// TestUserCompletedPuzzlePieceCount проверяет подсчет деталей в завершенных пазлах
func TestUserCompletedPuzzlePieceCount(t *testing.T) {
	ms := NewMemstorage()
	store := Storage(ms)

	// Создаем завершенный и незавершенный пазлы
	ms.puzzles.Store(100, &models.Puzzle{Id: 100, IsCompleted: true})
	ms.puzzles.Store(101, &models.Puzzle{Id: 101, IsCompleted: false})

	// Создаем пользователя
	user := &models.User{
		Id:               uuid.New(),
		Telegramm:        "count_test",
		FirstName:        "Count",
		LastName:         "Test",
		Group:            "Н1",
		RegistrationTime: time.Now(),
	}
	err := store.AddUser(user)
	require.NoError(t, err)

	// Добавляем детали
	piece1 := &models.PuzzlePiece{Code: "CNT001", PuzzleId: 100, PieceNumber: 1, OwnerId: &user.Id}
	piece2 := &models.PuzzlePiece{Code: "CNT002", PuzzleId: 100, PieceNumber: 2, OwnerId: &user.Id}
	piece3 := &models.PuzzlePiece{Code: "CNT003", PuzzleId: 101, PieceNumber: 1, OwnerId: &user.Id}

	store.AddPuzzlePiece(piece1)
	store.AddPuzzlePiece(piece2)
	store.AddPuzzlePiece(piece3)

	// Проверяем общее количество
	totalCount, err := store.GetUserPieceCount(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 3, totalCount)

	// Проверяем количество в завершенных пазлах
	completedCount, err := store.GetUserCompletedPuzzlePieceCount(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 2, completedCount, "Должно быть 2 детали в завершенных пазлах")
}
