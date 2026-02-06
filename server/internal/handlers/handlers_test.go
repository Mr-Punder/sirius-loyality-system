package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLogger реализует интерфейс logger.Logger для тестирования
type MockLogger struct{}

func (m *MockLogger) Info(msg string)                                        {}
func (m *MockLogger) Infof(format string, args ...interface{})               {}
func (m *MockLogger) Error(msg string)                                       {}
func (m *MockLogger) Errorf(format string, args ...interface{})              {}
func (m *MockLogger) Debug(msg string)                                       {}
func (m *MockLogger) Debugf(format string, args ...interface{})              {}
func (m *MockLogger) Warning(msg string)                                     {}
func (m *MockLogger) Warningf(format string, args ...interface{})            {}
func (m *MockLogger) Fatal(msg string)                                       {}
func (m *MockLogger) Fatalf(format string, args ...interface{})              {}
func (m *MockLogger) WithField(key string, value interface{}) logger.Logger  { return m }
func (m *MockLogger) WithFields(fields map[string]interface{}) logger.Logger { return m }
func (m *MockLogger) Close() error                                           { return nil }

func setupTest(t *testing.T) (*Handler, *storage.Memstorage) {
	mockLogger := &MockLogger{}
	memStorage := storage.NewMemstorage()
	handler := NewHandler(mockLogger, memStorage)
	return handler, memStorage
}

func TestRegisterUserHandler(t *testing.T) {
	handler, _ := setupTest(t)

	// Создаем запрос на регистрацию пользователя
	requestBody := map[string]interface{}{
		"telegramm":  "test_user",
		"first_name": "Test",
		"last_name":  "User",
		"group":      "Test Group",
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/users/register", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.RegisterUserHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Проверяем тело ответа
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test_user", response["telegramm"])
	assert.Equal(t, "Test", response["first_name"])
	assert.Equal(t, "User", response["last_name"])
	assert.Equal(t, "Test Group", response["group"])
}

func TestGetUserHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем пользователя
	user := &models.User{
		Id:               uuid.New(),
		Telegramm:        "test_user",
		FirstName:        "Test",
		LastName:         "User",
		MiddleName:       "",
		Group:            "Test Group",
		RegistrationTime: models.GetCurrentTime(),
		Deleted:          false,
	}
	err := storage.AddUser(user)
	require.NoError(t, err)

	// Создаем запрос на получение информации о пользователе
	req, err := http.NewRequest("GET", "/users/"+user.Id.String(), nil)
	require.NoError(t, err)

	// Добавляем параметр id в контекст запроса
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", user.Id.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.GetUserHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем тело ответа
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, user.Id.String(), response["id"])
	assert.Equal(t, user.Telegramm, response["telegramm"])
	assert.Equal(t, user.FirstName, response["first_name"])
	assert.Equal(t, user.LastName, response["last_name"])
	assert.Equal(t, user.Group, response["group"])
	assert.Equal(t, float64(0), response["piece_count"])
}

func TestUpdateUserHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем пользователя
	user := &models.User{
		Id:               uuid.New(),
		Telegramm:        "test_user",
		FirstName:        "Test",
		LastName:         "User",
		MiddleName:       "",
		Group:            "Test Group",
		RegistrationTime: models.GetCurrentTime(),
		Deleted:          false,
	}
	err := storage.AddUser(user)
	require.NoError(t, err)

	// Создаем запрос на обновление информации о пользователе
	requestBody := map[string]interface{}{
		"first_name":  "Updated",
		"last_name":   "Name",
		"middle_name": "Middle",
		"group":       "Updated Group",
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", "/users/"+user.Id.String(), bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Добавляем параметр id в контекст запроса
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", user.Id.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.UpdateUserHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем тело ответа
	var response models.User
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, user.Id, response.Id)
	assert.Equal(t, "Updated", response.FirstName)
	assert.Equal(t, "Name", response.LastName)
	assert.Equal(t, "Middle", response.MiddleName)
	assert.Equal(t, "Updated Group", response.Group)

	// Проверяем, что данные пользователя обновились в хранилище
	updatedUser, err := storage.GetUser(user.Id)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updatedUser.FirstName)
	assert.Equal(t, "Name", updatedUser.LastName)
	assert.Equal(t, "Middle", updatedUser.MiddleName)
	assert.Equal(t, "Updated Group", updatedUser.Group)
}

func TestDeleteUserHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем пользователя
	user := &models.User{
		Id:               uuid.New(),
		Telegramm:        "test_user",
		FirstName:        "Test",
		LastName:         "User",
		MiddleName:       "",
		Group:            "Test Group",
		RegistrationTime: models.GetCurrentTime(),
		Deleted:          false,
	}
	err := storage.AddUser(user)
	require.NoError(t, err)

	// Создаем запрос на удаление пользователя
	req, err := http.NewRequest("DELETE", "/users/"+user.Id.String(), nil)
	require.NoError(t, err)

	// Добавляем параметр id в контекст запроса
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", user.Id.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.DeleteUserHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем тело ответа
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])

	// Проверяем, что пользователь помечен как удаленный
	_, err = storage.GetUser(user.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestGetPuzzlesHandler(t *testing.T) {
	handler, _ := setupTest(t)

	// Создаем запрос на получение списка пазлов
	req, err := http.NewRequest("GET", "/puzzles", nil)
	require.NoError(t, err)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.GetPuzzlesHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем тело ответа
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Memstorage инициализирует 30 пазлов
	assert.Equal(t, float64(30), response["total"])
	puzzles := response["puzzles"].([]interface{})
	assert.Len(t, puzzles, 30)
}

func TestGetPuzzleHandler(t *testing.T) {
	handler, _ := setupTest(t)

	// Создаем запрос на получение информации о пазле
	req, err := http.NewRequest("GET", "/puzzles/1", nil)
	require.NoError(t, err)

	// Добавляем параметр id в контекст запроса
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.GetPuzzleHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем тело ответа
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(1), response["id"])
	assert.Equal(t, false, response["is_completed"])
	assert.Equal(t, float64(0), response["total_pieces"])
	assert.Equal(t, float64(0), response["owned_pieces"])
}

func TestAddPiecesHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем запрос на добавление деталей
	requestBody := map[string]interface{}{
		"pieces": []map[string]interface{}{
			{"code": "ABC1234", "puzzle_id": 1, "piece_number": 1},
			{"code": "DEF5678", "puzzle_id": 1, "piece_number": 2},
			{"code": "GHI9012", "puzzle_id": 1, "piece_number": 3},
		},
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/pieces", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.AddPiecesHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Проверяем тело ответа
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
	assert.Equal(t, float64(3), response["added"])

	// Проверяем, что детали добавлены в хранилище
	piece, err := storage.GetPuzzlePiece("ABC1234")
	require.NoError(t, err)
	assert.Equal(t, "ABC1234", piece.Code)
	assert.Equal(t, 1, piece.PuzzleId)
	assert.Equal(t, 1, piece.PieceNumber)
	assert.Nil(t, piece.OwnerId)
}

func TestGetAllPiecesHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Добавляем тестовые детали
	pieces := []*models.PuzzlePiece{
		{Code: "TEST001", PuzzleId: 1, PieceNumber: 1},
		{Code: "TEST002", PuzzleId: 1, PieceNumber: 2},
		{Code: "TEST003", PuzzleId: 2, PieceNumber: 1},
	}
	err := storage.AddPuzzlePieces(pieces)
	require.NoError(t, err)

	// Создаем запрос на получение списка деталей
	req, err := http.NewRequest("GET", "/pieces", nil)
	require.NoError(t, err)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.GetAllPiecesHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем тело ответа
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(3), response["total"])
}

func TestGetAllPiecesHandler_WithFilters(t *testing.T) {
	handler, storage := setupTest(t)

	// Добавляем тестовые детали
	userId := uuid.New()
	pieces := []*models.PuzzlePiece{
		{Code: "TEST001", PuzzleId: 1, PieceNumber: 1, OwnerId: &userId},
		{Code: "TEST002", PuzzleId: 1, PieceNumber: 2},
		{Code: "TEST003", PuzzleId: 2, PieceNumber: 1},
	}
	err := storage.AddPuzzlePieces(pieces)
	require.NoError(t, err)

	// Тест фильтра по пазлу
	t.Run("FilterByPuzzle", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/pieces?puzzle_id=1", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.GetAllPiecesHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(2), response["total"])
	})

	// Тест фильтра по наличию владельца
	t.Run("FilterByOwner", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/pieces?has_owner=true", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.GetAllPiecesHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(1), response["total"])
	})
}

func TestGetPieceHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Добавляем тестовую деталь
	piece := &models.PuzzlePiece{
		Code:        "TESTCODE",
		PuzzleId:    1,
		PieceNumber: 1,
	}
	err := storage.AddPuzzlePiece(piece)
	require.NoError(t, err)

	// Создаем запрос на получение информации о детали
	req, err := http.NewRequest("GET", "/pieces/TESTCODE", nil)
	require.NoError(t, err)

	// Добавляем параметр code в контекст запроса
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", "TESTCODE")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.GetPieceHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем тело ответа
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "TESTCODE", response["code"])
	assert.Equal(t, float64(1), response["puzzle_id"])
	assert.Equal(t, float64(1), response["piece_number"])
}

func TestRegisterPieceHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем пользователя
	user := &models.User{
		Id:               uuid.New(),
		Telegramm:        "test_user",
		FirstName:        "Test",
		LastName:         "User",
		Group:            "Test Group",
		RegistrationTime: models.GetCurrentTime(),
		Deleted:          false,
	}
	err := storage.AddUser(user)
	require.NoError(t, err)

	// Добавляем тестовую деталь
	piece := &models.PuzzlePiece{
		Code:        "REG_TEST",
		PuzzleId:    1,
		PieceNumber: 1,
	}
	err = storage.AddPuzzlePiece(piece)
	require.NoError(t, err)

	// Тест успешной регистрации
	t.Run("SuccessfulRegister", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"user_id": user.Id.String(),
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/pieces/REG_TEST/register", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", "REG_TEST")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.RegisterPieceHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["success"])
		assert.Equal(t, false, response["puzzle_completed"])

		// Проверяем, что деталь зарегистрирована
		registeredPiece, err := storage.GetPuzzlePiece("REG_TEST")
		require.NoError(t, err)
		assert.NotNil(t, registeredPiece.OwnerId)
		assert.Equal(t, user.Id, *registeredPiece.OwnerId)
	})

	// Тест: деталь уже занята
	t.Run("PieceAlreadyTaken", func(t *testing.T) {
		// Создаем второго пользователя
		user2 := &models.User{
			Id:               uuid.New(),
			Telegramm:        "test_user2",
			FirstName:        "Test2",
			LastName:         "User2",
			Group:            "Test Group",
			RegistrationTime: models.GetCurrentTime(),
			Deleted:          false,
		}
		err := storage.AddUser(user2)
		require.NoError(t, err)

		// Пытаемся зарегистрировать уже занятую деталь
		requestBody := map[string]interface{}{
			"user_id": user2.Id.String(),
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/pieces/REG_TEST/register", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", "REG_TEST")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.RegisterPieceHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, false, response["success"])
		assert.Equal(t, float64(models.PieceErrorAlreadyTaken), response["error_code"])
	})

	// Тест: деталь не найдена
	t.Run("PieceNotFound", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"user_id": user.Id.String(),
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/pieces/NONEXISTENT/register", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", "NONEXISTENT")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.RegisterPieceHandler(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, false, response["success"])
		assert.Equal(t, float64(models.PieceErrorNotFound), response["error_code"])
	})
}

func TestRegisterPieceHandler_AllPiecesDistributed(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем пользователей
	users := make([]*models.User, 6)
	for i := 0; i < 6; i++ {
		users[i] = &models.User{
			Id:               uuid.New(),
			Telegramm:        "user" + string(rune('0'+i)),
			FirstName:        "User",
			LastName:         string(rune('0' + i)),
			Group:            "Test",
			RegistrationTime: models.GetCurrentTime(),
			Deleted:          false,
		}
		err := storage.AddUser(users[i])
		require.NoError(t, err)
	}

	// Добавляем 6 деталей для пазла 1
	for i := 1; i <= 6; i++ {
		piece := &models.PuzzlePiece{
			Code:        "PUZZLE1_" + string(rune('0'+i)),
			PuzzleId:    1,
			PieceNumber: i,
		}
		err := storage.AddPuzzlePiece(piece)
		require.NoError(t, err)
	}

	// Регистрируем первые 5 деталей
	for i := 0; i < 5; i++ {
		requestBody := map[string]interface{}{
			"user_id": users[i].Id.String(),
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		code := "PUZZLE1_" + string(rune('0'+i+1))
		req, err := http.NewRequest("POST", "/pieces/"+code+"/register", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", code)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.RegisterPieceHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Первые 5 регистраций - не все детали розданы
		assert.Equal(t, false, response["puzzle_completed"], "После %d регистрации не все детали должны быть розданы", i+1)
	}

	// Регистрируем 6-ю деталь - все детали розданы
	requestBody := map[string]interface{}{
		"user_id": users[5].Id.String(),
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/pieces/PUZZLE1_6/register", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", "PUZZLE1_6")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.RegisterPieceHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// 6-я регистрация - все детали розданы (но пазл НЕ завершен автоматически)
	assert.Equal(t, true, response["puzzle_completed"], "После 6-й регистрации все детали должны быть розданы")

	// Проверяем, что пазл НЕ помечен как завершенный (требуется ручное завершение админом)
	puzzle, err := storage.GetPuzzle(1)
	require.NoError(t, err)
	assert.False(t, puzzle.IsCompleted, "Пазл не должен быть автоматически завершен")
	assert.Nil(t, puzzle.CompletedAt, "Время завершения должно быть nil")
}

func TestCompletePuzzleHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем пользователей и регистрируем все 6 деталей
	users := make([]*models.User, 6)
	for i := 0; i < 6; i++ {
		users[i] = &models.User{
			Id:               uuid.New(),
			Telegramm:        "complete_user" + string(rune('0'+i)),
			FirstName:        "User",
			LastName:         string(rune('0' + i)),
			Group:            "Test",
			RegistrationTime: models.GetCurrentTime(),
			Deleted:          false,
		}
		err := storage.AddUser(users[i])
		require.NoError(t, err)

		piece := &models.PuzzlePiece{
			Code:        "COMPLETE_" + string(rune('0'+i+1)),
			PuzzleId:    1,
			PieceNumber: i + 1,
			OwnerId:     &users[i].Id,
		}
		err = storage.AddPuzzlePiece(piece)
		require.NoError(t, err)
	}

	// Завершаем пазл через API
	req, err := http.NewRequest("POST", "/puzzles/1/complete", nil)
	require.NoError(t, err)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.CompletePuzzleHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
	usersToNotify := response["users_to_notify"].([]interface{})
	assert.Len(t, usersToNotify, 6, "Должно быть 6 владельцев для уведомления")

	// Проверяем, что пазл помечен как завершенный
	puzzle, err := storage.GetPuzzle(1)
	require.NoError(t, err)
	assert.True(t, puzzle.IsCompleted)
	assert.NotNil(t, puzzle.CompletedAt)

	// Попытка повторного завершения должна вернуть ошибку
	rr2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/puzzles/1/complete", nil)
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("id", "1")
	req2 = req2.WithContext(context.WithValue(req2.Context(), chi.RouteCtxKey, rctx2))

	handler.CompletePuzzleHandler(rr2, req2)
	assert.Equal(t, http.StatusBadRequest, rr2.Code)
}

func TestGetUserPiecesHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем пользователя
	user := &models.User{
		Id:               uuid.New(),
		Telegramm:        "test_user",
		FirstName:        "Test",
		LastName:         "User",
		Group:            "Test Group",
		RegistrationTime: models.GetCurrentTime(),
		Deleted:          false,
	}
	err := storage.AddUser(user)
	require.NoError(t, err)

	// Добавляем детали и регистрируем их на пользователя
	pieces := []*models.PuzzlePiece{
		{Code: "USER_P1", PuzzleId: 1, PieceNumber: 1, OwnerId: &user.Id},
		{Code: "USER_P2", PuzzleId: 1, PieceNumber: 2, OwnerId: &user.Id},
		{Code: "OTHER_P", PuzzleId: 2, PieceNumber: 1}, // Без владельца
	}
	err = storage.AddPuzzlePieces(pieces)
	require.NoError(t, err)

	// Создаем запрос на получение деталей пользователя
	req, err := http.NewRequest("GET", "/users/"+user.Id.String()+"/pieces", nil)
	require.NoError(t, err)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", user.Id.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetUserPiecesHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// У пользователя должно быть 2 детали
	assert.Equal(t, float64(2), response["total"])
}

func TestGetLotteryStatsHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем пользователей
	user1 := &models.User{
		Id:               uuid.New(),
		Telegramm:        "user1",
		FirstName:        "User",
		LastName:         "One",
		Group:            "Group1",
		RegistrationTime: models.GetCurrentTime(),
		Deleted:          false,
	}
	user2 := &models.User{
		Id:               uuid.New(),
		Telegramm:        "user2",
		FirstName:        "User",
		LastName:         "Two",
		Group:            "Group2",
		RegistrationTime: models.GetCurrentTime(),
		Deleted:          false,
	}
	err := storage.AddUser(user1)
	require.NoError(t, err)
	err = storage.AddUser(user2)
	require.NoError(t, err)

	// Добавляем детали
	pieces := []*models.PuzzlePiece{
		{Code: "STAT1", PuzzleId: 1, PieceNumber: 1, OwnerId: &user1.Id},
		{Code: "STAT2", PuzzleId: 1, PieceNumber: 2, OwnerId: &user1.Id},
		{Code: "STAT3", PuzzleId: 2, PieceNumber: 1, OwnerId: &user2.Id},
	}
	err = storage.AddPuzzlePieces(pieces)
	require.NoError(t, err)

	// Создаем запрос на получение статистики
	req, err := http.NewRequest("GET", "/stats/lottery", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.GetLotteryStatsHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(2), response["total_users"])
	assert.Equal(t, float64(30), response["total_puzzles"]) // Memstorage инициализирует 30 пазлов
	assert.Equal(t, float64(0), response["completed_puzzles"])

	users := response["users"].([]interface{})
	assert.Len(t, users, 2)
}
