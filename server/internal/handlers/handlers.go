package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	logger  logger.Logger
	storage storage.Storage
	timeout time.Duration
}

func NewHandler(logger logger.Logger, storage storage.Storage) *Handler {
	return &Handler{
		logger:  logger,
		storage: storage,
		timeout: 3 * time.Second,
	}
}

func NewRouter(logger logger.Logger, storage storage.Storage) chi.Router {
	r := chi.NewRouter()

	handler := NewHandler(logger, storage)

	return r.Route("/", func(r chi.Router) {
		r.Get("/ping", handler.PingHandler)

		// Маршруты для пользователей
		r.Route("/users", func(r chi.Router) {
			r.Post("/register", handler.RegisterUserHandler)
			r.Get("/", handler.GetUsersHandler)
			r.Get("/{id}", handler.GetUserHandler)
			r.Put("/{id}", handler.UpdateUserHandler)
			r.Delete("/{id}", handler.DeleteUserHandler)
		})

		// Маршруты для QR-кодов
		r.Route("/codes", func(r chi.Router) {
			r.Post("/", handler.CreateCodeHandler)
			r.Get("/", handler.GetCodesHandler)
			r.Get("/{code}", handler.GetCodeHandler)
			r.Put("/{code}", handler.UpdateCodeHandler)
			r.Delete("/{code}", handler.DeleteCodeHandler)
			r.Post("/{code}/apply", handler.ApplyCodeHandler)
		})

		// Маршруты для транзакций
		r.Route("/transactions", func(r chi.Router) {
			r.Get("/", handler.GetTransactionsHandler)
			r.Post("/", handler.CreateTransactionHandler)
		})

		// Маршруты для администраторов
		r.Route("/admins", func(r chi.Router) {
			r.Get("/check/{id}", handler.CheckAdminHandler)
		})

		// Обработчик по умолчанию для неправильных запросов
		r.Get("/{}", handler.DefoultHandler)
		r.Post("/{}", handler.DefoultHandler)
	})
}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered PingHandler")

	if r.Method != http.MethodGet {
		h.logger.Error("wrong request method")
		http.Error(w, "Only GET requests are allowed for ping!", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("Method checked")

	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("pong"))
	if err != nil {
		h.logger.Errorf("Error writing response $v", err)
	}
	h.logger.Info("PingHandler exited")
}

// DefoultHandler for incorrect requests
func (h *Handler) DefoultHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered DefoultHandler")
	http.Error(w, "wrong requests", http.StatusBadRequest)
}

// Обработчики для пользователей

// RegisterUserHandler регистрирует нового пользователя
func (h *Handler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered RegisterUserHandler")
	h.logger.Debug("Начало обработки запроса на регистрацию пользователя")

	// Декодируем запрос
	h.logger.Debug("Декодирование тела запроса")
	var request struct {
		Telegramm string `json:"telegramm"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Group     string `json:"group"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Errorf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	h.logger.Debugf("Получены данные пользователя: telegramm=%s, first_name=%s, last_name=%s, group=%s",
		request.Telegramm, request.FirstName, request.LastName, request.Group)

	// Создаем нового пользователя
	h.logger.Debug("Создание объекта нового пользователя")
	userId := uuid.New()
	h.logger.Debugf("Сгенерирован ID пользователя: %s", userId)

	registrationTime := time.Now()
	h.logger.Debugf("Время регистрации: %s", registrationTime)

	user := &models.User{
		Id:               userId,
		Telegramm:        request.Telegramm,
		FirstName:        request.FirstName,
		LastName:         request.LastName,
		MiddleName:       "", // Пустое значение по умолчанию
		Points:           0,  // Начальное количество баллов
		Group:            request.Group,
		RegistrationTime: registrationTime,
		Deleted:          false,
	}
	h.logger.Debugf("Создан объект пользователя: %+v", user)

	// Добавляем пользователя в хранилище
	h.logger.Debug("Добавление пользователя в хранилище")
	if err := h.storage.AddUser(user); err != nil {
		h.logger.Errorf("Error adding user: %v", err)
		http.Error(w, "Error adding user", http.StatusInternalServerError)
		return
	}
	h.logger.Debug("Пользователь успешно добавлен в хранилище")

	// Отправляем ответ
	h.logger.Debug("Подготовка ответа")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := struct {
		Id               uuid.UUID `json:"id"`
		Telegramm        string    `json:"telegramm"`
		FirstName        string    `json:"first_name"`
		LastName         string    `json:"last_name"`
		Points           int       `json:"points"`
		Group            string    `json:"group"`
		RegistrationTime time.Time `json:"registration_time"`
	}{
		Id:               user.Id,
		Telegramm:        user.Telegramm,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
		Points:           user.Points,
		Group:            user.Group,
		RegistrationTime: user.RegistrationTime,
	}
	h.logger.Debugf("Подготовлен ответ: %+v", response)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
	h.logger.Info("RegisterUserHandler завершен успешно")
}

// GetUsersHandler возвращает список всех пользователей
func (h *Handler) GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetUsersHandler")

	// Получаем всех пользователей из хранилища
	users, err := h.storage.GetAllUsers()
	if err != nil {
		h.logger.Errorf("Error getting users: %v", err)
		http.Error(w, "Error getting users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}{
		Total: len(users),
		Users: users,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// GetUserHandler возвращает информацию о конкретном пользователе
func (h *Handler) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetUserHandler")

	// Получаем ID пользователя из URL
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Errorf("Invalid user ID: %v", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Получаем пользователя из хранилища
	user, err := h.storage.GetUser(id)
	if err != nil {
		h.logger.Errorf("Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(user); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// UpdateUserHandler обновляет информацию о пользователе
func (h *Handler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered UpdateUserHandler")

	// Получаем ID пользователя из URL
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Errorf("Invalid user ID: %v", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Получаем пользователя из хранилища
	user, err := h.storage.GetUser(id)
	if err != nil {
		h.logger.Errorf("Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Декодируем запрос
	var request struct {
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		MiddleName string `json:"middle_name"`
		Group      string `json:"group"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Errorf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Обновляем данные пользователя
	user.FirstName = request.FirstName
	user.LastName = request.LastName
	user.MiddleName = request.MiddleName
	user.Group = request.Group

	// Сохраняем изменения
	if err := h.storage.UpdateUser(user); err != nil {
		h.logger.Errorf("Error updating user: %v", err)
		http.Error(w, "Error updating user", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(user); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// DeleteUserHandler удаляет пользователя (мягкое удаление)
func (h *Handler) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered DeleteUserHandler")

	// Получаем ID пользователя из URL
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Errorf("Invalid user ID: %v", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Удаляем пользователя
	if err := h.storage.DeleteUser(id); err != nil {
		h.logger.Errorf("Error deleting user: %v", err)
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Success bool `json:"success"`
	}{
		Success: true,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// Обработчики для QR-кодов

// CreateCodeHandler создает новый QR-код
func (h *Handler) CreateCodeHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered CreateCodeHandler")
	h.logger.Debug("Начало обработки запроса на создание QR-кода")

	// Декодируем запрос
	h.logger.Debug("Декодирование тела запроса")
	var request struct {
		Amount  int    `json:"amount"`
		PerUser int    `json:"per_user"`
		Total   int    `json:"total"`
		Group   string `json:"group"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Errorf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	h.logger.Debugf("Получены параметры кода: amount=%d, perUser=%d, total=%d, group=%s",
		request.Amount, request.PerUser, request.Total, request.Group)

	// Создаем новый код
	h.logger.Debug("Создание объекта нового QR-кода")
	codeUUID := uuid.New()
	h.logger.Debugf("Сгенерирован UUID кода: %s", codeUUID)

	code := &models.Code{
		Code:         codeUUID,
		Amount:       request.Amount,
		PerUser:      request.PerUser,
		Total:        request.Total,
		AppliedCount: 0,
		IsActive:     true,
		Group:        request.Group,
		ErrorCode:    models.ErrorCodeNone,
	}
	h.logger.Debugf("Создан объект QR-кода: %+v", code)

	// Добавляем код в хранилище
	h.logger.Debug("Добавление QR-кода в хранилище")
	if err := h.storage.AddCode(code); err != nil {
		h.logger.Errorf("Error adding code: %v", err)
		http.Error(w, "Error adding code", http.StatusInternalServerError)
		return
	}
	h.logger.Debug("QR-код успешно добавлен в хранилище")

	// Отправляем ответ
	h.logger.Debug("Подготовка ответа")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	h.logger.Debugf("Отправка ответа с кодом: %+v", code)
	if err := json.NewEncoder(w).Encode(code); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
	h.logger.Info("CreateCodeHandler завершен успешно")
}

// GetCodesHandler возвращает список всех QR-кодов
func (h *Handler) GetCodesHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetCodesHandler")

	// Получаем все коды из хранилища
	codes, err := h.storage.GetAllCodes()
	if err != nil {
		h.logger.Errorf("Error getting codes: %v", err)
		http.Error(w, "Error getting codes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Total int            `json:"total"`
		Codes []*models.Code `json:"codes"`
	}{
		Total: len(codes),
		Codes: codes,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// GetCodeHandler возвращает информацию о конкретном QR-коде
func (h *Handler) GetCodeHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetCodeHandler")

	// Получаем код из URL
	codeStr := chi.URLParam(r, "code")
	codeUUID, err := uuid.Parse(codeStr)
	if err != nil {
		h.logger.Errorf("Invalid code: %v", err)
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}

	// Получаем информацию о коде из хранилища
	code, err := h.storage.GetCodeInfo(codeUUID)
	if err != nil {
		h.logger.Errorf("Error getting code: %v", err)
		http.Error(w, "Code not found", http.StatusNotFound)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(code); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// UpdateCodeHandler обновляет информацию о QR-коде
func (h *Handler) UpdateCodeHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered UpdateCodeHandler")

	// Получаем код из URL
	codeStr := chi.URLParam(r, "code")
	codeUUID, err := uuid.Parse(codeStr)
	if err != nil {
		h.logger.Errorf("Invalid code: %v", err)
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}

	// Получаем информацию о коде из хранилища
	code, err := h.storage.GetCodeInfo(codeUUID)
	if err != nil {
		h.logger.Errorf("Error getting code: %v", err)
		http.Error(w, "Code not found", http.StatusNotFound)
		return
	}

	// Декодируем запрос
	var request struct {
		Amount   int    `json:"amount"`
		PerUser  int    `json:"per_user"`
		Total    int    `json:"total"`
		IsActive bool   `json:"is_active"`
		Group    string `json:"group"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Errorf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Обновляем данные кода
	code.Amount = request.Amount
	code.PerUser = request.PerUser
	code.Total = request.Total
	code.IsActive = request.IsActive
	code.Group = request.Group

	// Сохраняем изменения
	if err := h.storage.UpdateCode(code); err != nil {
		h.logger.Errorf("Error updating code: %v", err)
		http.Error(w, "Error updating code", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(code); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// DeleteCodeHandler деактивирует QR-код
func (h *Handler) DeleteCodeHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered DeleteCodeHandler")

	// Получаем код из URL
	codeStr := chi.URLParam(r, "code")
	codeUUID, err := uuid.Parse(codeStr)
	if err != nil {
		h.logger.Errorf("Invalid code: %v", err)
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}

	// Деактивируем код
	if err := h.storage.DeleteCode(codeUUID); err != nil {
		h.logger.Errorf("Error deleting code: %v", err)
		http.Error(w, "Error deleting code", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Success bool `json:"success"`
	}{
		Success: true,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// ApplyCodeHandler применяет QR-код для пользователя
func (h *Handler) ApplyCodeHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered ApplyCodeHandler")
	h.logger.Debug("Начало обработки запроса на применение QR-кода")

	// Получаем код из URL
	codeStr := chi.URLParam(r, "code")
	h.logger.Debugf("Получен код из URL: %s", codeStr)

	codeUUID, err := uuid.Parse(codeStr)
	if err != nil {
		h.logger.Errorf("Invalid code: %v", err)
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}
	h.logger.Debugf("Код успешно преобразован в UUID: %s", codeUUID)

	// Декодируем запрос для получения ID пользователя
	var request struct {
		UserID uuid.UUID `json:"user_id"`
	}

	h.logger.Debug("Декодирование тела запроса для получения ID пользователя")
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Errorf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := request.UserID
	h.logger.Debugf("Получен ID пользователя: %s", userID)

	// Получаем информацию о коде
	h.logger.Debugf("Запрос информации о коде %s из хранилища", codeUUID)
	code, err := h.storage.GetCodeInfo(codeUUID)
	if err != nil {
		h.logger.Errorf("Error getting code: %v", err)
		http.Error(w, "Code not found", http.StatusNotFound)
		return
	}
	h.logger.Debugf("Получена информация о коде: %+v", code)

	// Проверяем, активен ли код
	h.logger.Debugf("Проверка активности кода. Текущий статус: %v", code.IsActive)
	if !code.IsActive {
		h.logger.Error("Code is not active")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := struct {
			Success   bool   `json:"success"`
			Error     string `json:"error"`
			ErrorCode int    `json:"error_code"`
		}{
			Success:   false,
			Error:     "Код не активен",
			ErrorCode: models.ErrorCodeCodeInactive,
		}
		h.logger.Debug("Отправка ответа: код не активен")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			h.logger.Errorf("Error encoding response: %v", err)
		}
		return
	}
	h.logger.Debug("Код активен, продолжаем обработку")

	// Проверяем, принадлежит ли пользователь к нужной группе
	if code.Group != "" {
		h.logger.Debugf("Код имеет ограничение по группе: %s. Проверяем группу пользователя", code.Group)

		h.logger.Debugf("Запрос информации о пользователе %s из хранилища", userID)
		user, err := h.storage.GetUser(userID)
		if err != nil {
			h.logger.Errorf("Error getting user: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		h.logger.Debugf("Получена информация о пользователе: %+v", user)

		h.logger.Debugf("Сравнение групп: пользователь %s, код %s", user.Group, code.Group)
		if user.Group != code.Group {
			h.logger.Errorf("User group %s does not match code group %s", user.Group, code.Group)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			response := struct {
				Success   bool   `json:"success"`
				Error     string `json:"error"`
				ErrorCode int    `json:"error_code"`
			}{
				Success:   false,
				Error:     "Пользователь не принадлежит к группе, для которой предназначен код",
				ErrorCode: models.ErrorCodeInvalidGroup,
			}
			h.logger.Debug("Отправка ответа: несоответствие группы")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				h.logger.Errorf("Error encoding response: %v", err)
			}
			return
		}
		h.logger.Debug("Группа пользователя соответствует группе кода")
	} else {
		h.logger.Debug("Код не имеет ограничений по группе")
	}

	// Создаем новое использование кода
	usage := &models.CodeUsage{
		Id:     uuid.New(),
		Code:   codeUUID,
		UserId: userID,
		Count:  1,
	}
	h.logger.Debugf("Создано новое использование кода: %+v", usage)

	// Добавляем использование кода
	h.logger.Debug("Добавление использования кода в хранилище")
	if err := h.storage.AddCodeUsage(usage); err != nil {
		h.logger.Errorf("Error applying code: %v", err)

		// Возвращаем более информативные ошибки в зависимости от типа ошибки
		switch err.Error() {
		case "code usage limit exceeded":
			h.logger.Debug("Ошибка: превышено общее количество использований кода")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			response := struct {
				Success   bool   `json:"success"`
				Error     string `json:"error"`
				ErrorCode int    `json:"error_code"`
			}{
				Success:   false,
				Error:     "Превышено общее количество использований кода",
				ErrorCode: models.ErrorCodeTotalLimitExceeded,
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				h.logger.Errorf("Error encoding response: %v", err)
			}
			return
		case "user code usage limit exceeded":
			h.logger.Debug("Ошибка: превышено количество использований кода пользователем")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			response := struct {
				Success   bool   `json:"success"`
				Error     string `json:"error"`
				ErrorCode int    `json:"error_code"`
			}{
				Success:   false,
				Error:     "Превышено количество использований кода пользователем",
				ErrorCode: models.ErrorCodeUserLimitExceeded,
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				h.logger.Errorf("Error encoding response: %v", err)
			}
			return
		case "code is not active":
			h.logger.Debug("Ошибка: код не активен")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			response := struct {
				Success   bool   `json:"success"`
				Error     string `json:"error"`
				ErrorCode int    `json:"error_code"`
			}{
				Success:   false,
				Error:     "Код не активен",
				ErrorCode: models.ErrorCodeCodeInactive,
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				h.logger.Errorf("Error encoding response: %v", err)
			}
			return
		default:
			h.logger.Debugf("Неизвестная ошибка: %v", err)
			http.Error(w, "Error applying code", http.StatusInternalServerError)
			return
		}
	}
	h.logger.Debug("Использование кода успешно добавлено")

	// Создаем транзакцию
	transaction := &models.Transaction{
		Id:     uuid.New(),
		UserId: userID,
		Code:   codeUUID,
		Diff:   code.Amount,
		Time:   time.Now(),
	}
	h.logger.Debugf("Создана новая транзакция: %+v", transaction)

	// Добавляем транзакцию
	h.logger.Debug("Добавление транзакции в хранилище")
	if err := h.storage.AddTransaction(transaction); err != nil {
		h.logger.Errorf("Error adding transaction: %v", err)
		http.Error(w, "Error adding transaction", http.StatusInternalServerError)
		return
	}
	h.logger.Debug("Транзакция успешно добавлена")

	// Получаем обновленные баллы пользователя
	h.logger.Debugf("Запрос обновленных баллов пользователя %s", userID)
	points, err := h.storage.GetUserPoints(userID)
	if err != nil {
		h.logger.Errorf("Error getting user points: %v", err)
		http.Error(w, "Error getting user points", http.StatusInternalServerError)
		return
	}
	h.logger.Debugf("Получены обновленные баллы пользователя: %d", points)

	// Отправляем ответ
	h.logger.Debug("Подготовка успешного ответа")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Success     bool `json:"success"`
		PointsAdded int  `json:"points_added"`
		TotalPoints int  `json:"total_points"`
	}{
		Success:     true,
		PointsAdded: code.Amount,
		TotalPoints: points,
	}
	h.logger.Debugf("Отправка ответа: %+v", response)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
	h.logger.Info("ApplyCodeHandler завершен успешно")
}

// Обработчики для транзакций

// GetTransactionsHandler возвращает список всех транзакций
func (h *Handler) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetTransactionsHandler")

	// Получаем все транзакции из хранилища
	transactions, err := h.storage.GetAllTransactions()
	if err != nil {
		h.logger.Errorf("Error getting transactions: %v", err)
		http.Error(w, "Error getting transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Total        int                   `json:"total"`
		Transactions []*models.Transaction `json:"transactions"`
	}{
		Total:        len(transactions),
		Transactions: transactions,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// CheckAdminHandler проверяет, является ли пользователь администратором
func (h *Handler) CheckAdminHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered CheckAdminHandler")

	// Получаем ID пользователя из URL
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		h.logger.Error("ID пользователя не указан")
		http.Error(w, "ID пользователя не указан", http.StatusBadRequest)
		return
	}

	// Парсим ID пользователя
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Errorf("Неверный формат ID пользователя: %v", err)
		http.Error(w, "Неверный формат ID пользователя", http.StatusBadRequest)
		return
	}

	// Проверяем, является ли пользователь администратором
	admin, err := h.storage.GetAdmin(id)
	if err != nil {
		// Если пользователь не найден, возвращаем false
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"is_admin": false})
		return
	}

	// Проверяем, активен ли администратор
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"is_admin": admin.IsActive})
}

// CreateTransactionHandler создает новую транзакцию
func (h *Handler) CreateTransactionHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered CreateTransactionHandler")

	// Декодируем запрос
	var request struct {
		UserID uuid.UUID `json:"user_id"`
		Diff   int       `json:"diff"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Errorf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Создаем новую транзакцию
	transaction := &models.Transaction{
		Id:     uuid.New(),
		UserId: request.UserID,
		Diff:   request.Diff,
		Time:   time.Now(),
	}

	// Добавляем транзакцию
	if err := h.storage.AddTransaction(transaction); err != nil {
		h.logger.Errorf("Error adding transaction: %v", err)
		http.Error(w, "Error adding transaction", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(transaction); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}
