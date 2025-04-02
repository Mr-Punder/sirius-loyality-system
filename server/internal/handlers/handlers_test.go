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
	assert.Equal(t, float64(0), response["points"])
}

func TestCreateCodeHandler(t *testing.T) {
	handler, _ := setupTest(t)

	// Создаем запрос на создание кода
	requestBody := map[string]interface{}{
		"amount":   100,
		"per_user": 2,
		"total":    5,
		"group":    "Test Group",
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/codes", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.CreateCodeHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Проверяем тело ответа
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(100), response["amount"])
	assert.Equal(t, float64(2), response["per_user"])
	assert.Equal(t, float64(5), response["total"])
	assert.Equal(t, "Test Group", response["group"])
	assert.Equal(t, float64(0), response["applied_count"])
	assert.Equal(t, true, response["is_active"])
	assert.Equal(t, float64(0), response["error_code"])
}

func TestApplyCodeHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем пользователя
	user := &models.User{
		Id:               uuid.New(),
		Telegramm:        "test_user",
		FirstName:        "Test",
		LastName:         "User",
		MiddleName:       "",
		Points:           0,
		Group:            "Test Group",
		RegistrationTime: models.GetCurrentTime(),
		Deleted:          false,
	}
	err := storage.AddUser(user)
	require.NoError(t, err)

	// Создаем код
	code := &models.Code{
		Code:         uuid.New(),
		Amount:       100,
		PerUser:      2,
		Total:        5,
		AppliedCount: 0,
		IsActive:     true,
		Group:        "Test Group",
		ErrorCode:    models.ErrorCodeNone,
	}
	err = storage.AddCode(code)
	require.NoError(t, err)

	// Тест 1: Успешное применение кода
	t.Run("SuccessfulApply", func(t *testing.T) {
		// Создаем запрос на применение кода
		requestBody := map[string]interface{}{
			"user_id": user.Id.String(),
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/codes/"+code.Code.String()+"/apply", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Добавляем параметр code в контекст запроса
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", code.Code.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.ApplyCodeHandler(rr, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем тело ответа
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["success"])
		assert.Equal(t, float64(100), response["points_added"])
		assert.Equal(t, float64(100), response["total_points"])

		// Проверяем, что баллы пользователя обновились
		points, err := storage.GetUserPoints(user.Id)
		require.NoError(t, err)
		assert.Equal(t, 100, points)

		// Проверяем, что счетчик использований кода увеличился
		codeInfo, err := storage.GetCodeInfo(code.Code)
		require.NoError(t, err)
		assert.Equal(t, 1, codeInfo.AppliedCount)
	})

	// Тест 2: Применение кода с ограничением на группу пользователей
	t.Run("GroupLimitApply", func(t *testing.T) {
		// Создаем пользователя из другой группы
		user2 := &models.User{
			Id:               uuid.New(),
			Telegramm:        "test_user2",
			FirstName:        "Test",
			LastName:         "User2",
			MiddleName:       "",
			Points:           0,
			Group:            "Another Group",
			RegistrationTime: models.GetCurrentTime(),
			Deleted:          false,
		}
		err := storage.AddUser(user2)
		require.NoError(t, err)

		// Создаем код с ограничением на группу
		code2 := &models.Code{
			Code:         uuid.New(),
			Amount:       100,
			PerUser:      2,
			Total:        5,
			AppliedCount: 0,
			IsActive:     true,
			Group:        "Test Group",
			ErrorCode:    models.ErrorCodeNone,
		}
		err = storage.AddCode(code2)
		require.NoError(t, err)

		// Создаем запрос на применение кода пользователем из другой группы
		requestBody := map[string]interface{}{
			"user_id": user2.Id.String(),
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/codes/"+code2.Code.String()+"/apply", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Добавляем параметр code в контекст запроса
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", code2.Code.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.ApplyCodeHandler(rr, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Проверяем тело ответа
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, false, response["success"])
		assert.Equal(t, "Пользователь не принадлежит к группе, для которой предназначен код", response["error"])
		assert.Equal(t, float64(models.ErrorCodeInvalidGroup), response["error_code"])
	})

	// Тест 3: Применение кода с ограничением на количество использований одним пользователем
	t.Run("PerUserLimitApply", func(t *testing.T) {
		// Создаем код с ограничением per_user = 1
		code3 := &models.Code{
			Code:         uuid.New(),
			Amount:       100,
			PerUser:      1,
			Total:        5,
			AppliedCount: 0,
			IsActive:     true,
			Group:        "",
			ErrorCode:    models.ErrorCodeNone,
		}
		err := storage.AddCode(code3)
		require.NoError(t, err)

		// Применяем код первый раз
		requestBody := map[string]interface{}{
			"user_id": user.Id.String(),
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/codes/"+code3.Code.String()+"/apply", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Добавляем параметр code в контекст запроса
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", code3.Code.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.ApplyCodeHandler(rr, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusOK, rr.Code)

		// Применяем код второй раз - должна быть ошибка
		req, err = http.NewRequest("POST", "/codes/"+code3.Code.String()+"/apply", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Добавляем параметр code в контекст запроса
		rctx = chi.NewRouteContext()
		rctx.URLParams.Add("code", code3.Code.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Создаем ResponseRecorder для записи ответа
		rr = httptest.NewRecorder()

		// Вызываем обработчик
		handler.ApplyCodeHandler(rr, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Проверяем тело ответа
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, false, response["success"])
		assert.Equal(t, "Превышено количество использований кода пользователем", response["error"])
		assert.Equal(t, float64(models.ErrorCodeUserLimitExceeded), response["error_code"])
	})

	// Тест 4: Применение кода с ограничением на общее количество использований
	t.Run("TotalLimitApply", func(t *testing.T) {
		// Создаем код с ограничением total = 1
		code4 := &models.Code{
			Code:         uuid.New(),
			Amount:       100,
			PerUser:      0,
			Total:        1,
			AppliedCount: 0,
			IsActive:     true,
			Group:        "",
			ErrorCode:    models.ErrorCodeNone,
		}
		err := storage.AddCode(code4)
		require.NoError(t, err)

		// Применяем код первый раз
		requestBody := map[string]interface{}{
			"user_id": user.Id.String(),
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/codes/"+code4.Code.String()+"/apply", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Добавляем параметр code в контекст запроса
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", code4.Code.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.ApplyCodeHandler(rr, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusOK, rr.Code)

		// Создаем второго пользователя
		user3 := &models.User{
			Id:               uuid.New(),
			Telegramm:        "test_user3",
			FirstName:        "Test",
			LastName:         "User3",
			MiddleName:       "",
			Points:           0,
			Group:            "Test Group",
			RegistrationTime: models.GetCurrentTime(),
			Deleted:          false,
		}
		err = storage.AddUser(user3)
		require.NoError(t, err)

		// Применяем код второй раз другим пользователем - должна быть ошибка
		requestBody = map[string]interface{}{
			"user_id": user3.Id.String(),
		}
		body, err = json.Marshal(requestBody)
		require.NoError(t, err)

		req, err = http.NewRequest("POST", "/codes/"+code4.Code.String()+"/apply", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Добавляем параметр code в контекст запроса
		rctx = chi.NewRouteContext()
		rctx.URLParams.Add("code", code4.Code.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Создаем ResponseRecorder для записи ответа
		rr = httptest.NewRecorder()

		// Вызываем обработчик
		handler.ApplyCodeHandler(rr, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Проверяем тело ответа
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, false, response["success"])
		assert.Equal(t, "Превышено общее количество использований кода", response["error"])
		assert.Equal(t, float64(models.ErrorCodeTotalLimitExceeded), response["error_code"])
	})

	// Тест 5: Применение неактивного кода
	t.Run("InactiveCodeApply", func(t *testing.T) {
		// Создаем неактивный код
		code5 := &models.Code{
			Code:         uuid.New(),
			Amount:       100,
			PerUser:      0,
			Total:        0,
			AppliedCount: 0,
			IsActive:     false,
			Group:        "",
			ErrorCode:    models.ErrorCodeNone,
		}
		err := storage.AddCode(code5)
		require.NoError(t, err)

		// Применяем неактивный код
		requestBody := map[string]interface{}{
			"user_id": user.Id.String(),
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/codes/"+code5.Code.String()+"/apply", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Добавляем параметр code в контекст запроса
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", code5.Code.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.ApplyCodeHandler(rr, req)

		// Проверяем статус код
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Проверяем тело ответа
		var response map[string]interface{}
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, false, response["success"])
		assert.Equal(t, "Код не активен", response["error"])
		assert.Equal(t, float64(models.ErrorCodeCodeInactive), response["error_code"])
	})
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
		Points:           0,
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
	var response models.User
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, user.Id, response.Id)
	assert.Equal(t, user.Telegramm, response.Telegramm)
	assert.Equal(t, user.FirstName, response.FirstName)
	assert.Equal(t, user.LastName, response.LastName)
	assert.Equal(t, user.Group, response.Group)
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
		Points:           0,
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
		Points:           0,
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

func TestUpdateCodeHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем код
	code := &models.Code{
		Code:         uuid.New(),
		Amount:       100,
		PerUser:      2,
		Total:        5,
		AppliedCount: 0,
		IsActive:     true,
		Group:        "Test Group",
		ErrorCode:    models.ErrorCodeNone,
	}
	err := storage.AddCode(code)
	require.NoError(t, err)

	// Создаем запрос на обновление информации о коде
	requestBody := map[string]interface{}{
		"amount":    200,
		"per_user":  3,
		"total":     10,
		"is_active": true,
		"group":     "Updated Group",
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", "/codes/"+code.Code.String(), bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Добавляем параметр code в контекст запроса
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", code.Code.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.UpdateCodeHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем тело ответа
	var response models.Code
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, code.Code, response.Code)
	assert.Equal(t, 200, response.Amount)
	assert.Equal(t, 3, response.PerUser)
	assert.Equal(t, 10, response.Total)
	assert.Equal(t, true, response.IsActive)
	assert.Equal(t, "Updated Group", response.Group)

	// Проверяем, что данные кода обновились в хранилище
	updatedCode, err := storage.GetCodeInfo(code.Code)
	require.NoError(t, err)
	assert.Equal(t, 200, updatedCode.Amount)
	assert.Equal(t, 3, updatedCode.PerUser)
	assert.Equal(t, 10, updatedCode.Total)
	assert.Equal(t, true, updatedCode.IsActive)
	assert.Equal(t, "Updated Group", updatedCode.Group)
}

func TestDeleteCodeHandler(t *testing.T) {
	handler, storage := setupTest(t)

	// Создаем код
	code := &models.Code{
		Code:         uuid.New(),
		Amount:       100,
		PerUser:      2,
		Total:        5,
		AppliedCount: 0,
		IsActive:     true,
		Group:        "Test Group",
		ErrorCode:    models.ErrorCodeNone,
	}
	err := storage.AddCode(code)
	require.NoError(t, err)

	// Создаем запрос на деактивацию кода
	req, err := http.NewRequest("DELETE", "/codes/"+code.Code.String(), nil)
	require.NoError(t, err)

	// Добавляем параметр code в контекст запроса
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", code.Code.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.DeleteCodeHandler(rr, req)

	// Проверяем статус код
	assert.Equal(t, http.StatusOK, rr.Code)

	// Проверяем тело ответа
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])

	// Проверяем, что код деактивирован
	codeInfo, err := storage.GetCodeInfo(code.Code)
	require.NoError(t, err)
	assert.False(t, codeInfo.IsActive)
}
