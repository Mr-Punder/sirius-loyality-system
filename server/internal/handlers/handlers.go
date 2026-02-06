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
			r.Get("/{id}/pieces", handler.GetUserPiecesHandler)
		})

		// Маршруты для пазлов
		r.Route("/puzzles", func(r chi.Router) {
			r.Get("/", handler.GetPuzzlesHandler)
			r.Get("/{id}", handler.GetPuzzleHandler)
			r.Put("/{id}", handler.UpdatePuzzleHandler)
			r.Post("/{id}/complete", handler.CompletePuzzleHandler)
			r.Get("/{id}/pieces", handler.GetPuzzlePiecesHandler)
		})

		// Маршруты для деталей пазлов
		r.Route("/pieces", func(r chi.Router) {
			r.Get("/", handler.GetAllPiecesHandler)
			r.Post("/", handler.AddPiecesHandler)
			r.Get("/{code}", handler.GetPieceHandler)
			r.Post("/{code}/register", handler.RegisterPieceHandler)
		})

		// Маршруты для статистики
		r.Route("/stats", func(r chi.Router) {
			r.Get("/lottery", handler.GetLotteryStatsHandler)
		})

		// Маршруты для администраторов
		r.Route("/admins", func(r chi.Router) {
			r.Get("/", handler.GetAdminsHandler)
			r.Post("/", handler.AddAdminHandler)
			r.Get("/check/{id}", handler.CheckAdminHandler)
			r.Delete("/{id}", handler.DeleteAdminHandler)
		})

		// Маршруты для уведомлений (рассылка)
		r.Route("/notifications", func(r chi.Router) {
			r.Post("/", handler.CreateNotificationHandler)
			r.Get("/", handler.GetNotificationsHandler)
			r.Get("/pending", handler.GetPendingNotificationsHandler)
			r.Get("/{id}", handler.GetNotificationHandler)
			r.Patch("/{id}", handler.UpdateNotificationHandler)
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

// ==================== ОБРАБОТЧИКИ ДЛЯ ПОЛЬЗОВАТЕЛЕЙ ====================

// RegisterUserHandler регистрирует нового пользователя
func (h *Handler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered RegisterUserHandler")
	h.logger.Debug("Начало обработки запроса на регистрацию пользователя")

	// Декодируем запрос
	h.logger.Debug("Декодирование тела запроса")
	var request struct {
		Telegramm  string `json:"telegramm"`
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
	h.logger.Debugf("Получены данные пользователя: telegramm=%s, first_name=%s, last_name=%s, middle_name=%s, group=%s",
		request.Telegramm, request.FirstName, request.LastName, request.MiddleName, request.Group)

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
		MiddleName:       request.MiddleName,
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
		Group            string    `json:"group"`
		RegistrationTime time.Time `json:"registration_time"`
	}{
		Id:               user.Id,
		Telegramm:        user.Telegramm,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
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

	// Получаем количество деталей пользователя
	pieceCount, err := h.storage.GetUserPieceCount(id)
	if err != nil {
		h.logger.Errorf("Error getting user piece count: %v", err)
		pieceCount = 0
	}

	// Отправляем ответ с дополнительной информацией
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		*models.User
		PieceCount int `json:"piece_count"`
	}{
		User:       user,
		PieceCount: pieceCount,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
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

// GetUserPiecesHandler возвращает детали пользователя
func (h *Handler) GetUserPiecesHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetUserPiecesHandler")

	// Получаем ID пользователя из URL
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Errorf("Invalid user ID: %v", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли пользователь
	_, err = h.storage.GetUser(id)
	if err != nil {
		h.logger.Errorf("Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Получаем детали пользователя
	pieces, err := h.storage.GetPuzzlePiecesByOwner(id)
	if err != nil {
		h.logger.Errorf("Error getting user pieces: %v", err)
		http.Error(w, "Error getting user pieces", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Total  int                   `json:"total"`
		Pieces []*models.PuzzlePiece `json:"pieces"`
	}{
		Total:  len(pieces),
		Pieces: pieces,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// ==================== ОБРАБОТЧИКИ ДЛЯ ПАЗЛОВ ====================

// GetPuzzlesHandler возвращает список всех пазлов
func (h *Handler) GetPuzzlesHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetPuzzlesHandler")

	// Получаем все пазлы из хранилища
	puzzles, err := h.storage.GetAllPuzzles()
	if err != nil {
		h.logger.Errorf("Error getting puzzles: %v", err)
		http.Error(w, "Error getting puzzles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Total   int              `json:"total"`
		Puzzles []*models.Puzzle `json:"puzzles"`
	}{
		Total:   len(puzzles),
		Puzzles: puzzles,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// GetPuzzleHandler возвращает информацию о конкретном пазле
func (h *Handler) GetPuzzleHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetPuzzleHandler")

	// Получаем ID пазла из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Errorf("Invalid puzzle ID: %v", err)
		http.Error(w, "Invalid puzzle ID", http.StatusBadRequest)
		return
	}

	// Получаем пазл из хранилища
	puzzle, err := h.storage.GetPuzzle(id)
	if err != nil {
		h.logger.Errorf("Error getting puzzle: %v", err)
		http.Error(w, "Puzzle not found", http.StatusNotFound)
		return
	}

	// Получаем детали пазла для подсчета прогресса
	pieces, err := h.storage.GetPuzzlePiecesByPuzzle(id)
	if err != nil {
		h.logger.Errorf("Error getting puzzle pieces: %v", err)
		pieces = []*models.PuzzlePiece{}
	}

	ownedCount := 0
	for _, p := range pieces {
		if p.OwnerId != nil {
			ownedCount++
		}
	}

	// Отправляем ответ с дополнительной информацией
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		*models.Puzzle
		TotalPieces int `json:"total_pieces"`
		OwnedPieces int `json:"owned_pieces"`
	}{
		Puzzle:      puzzle,
		TotalPieces: len(pieces),
		OwnedPieces: ownedCount,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// UpdatePuzzleHandler обновляет информацию о пазле (название)
func (h *Handler) UpdatePuzzleHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered UpdatePuzzleHandler")

	// Получаем ID пазла из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Errorf("Invalid puzzle ID: %v", err)
		http.Error(w, "Invalid puzzle ID", http.StatusBadRequest)
		return
	}

	// Получаем текущий пазл
	puzzle, err := h.storage.GetPuzzle(id)
	if err != nil {
		h.logger.Errorf("Error getting puzzle: %v", err)
		http.Error(w, "Puzzle not found", http.StatusNotFound)
		return
	}

	// Декодируем запрос
	var request struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Errorf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Обновляем название
	puzzle.Name = request.Name
	if err := h.storage.UpdatePuzzle(puzzle); err != nil {
		h.logger.Errorf("Error updating puzzle: %v", err)
		http.Error(w, "Error updating puzzle", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(puzzle)
}

// CompletePuzzleHandler отмечает пазл как собранный и возвращает владельцев для уведомления
func (h *Handler) CompletePuzzleHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered CompletePuzzleHandler")

	// Получаем ID пазла из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Errorf("Invalid puzzle ID: %v", err)
		http.Error(w, "Invalid puzzle ID", http.StatusBadRequest)
		return
	}

	// Завершаем пазл и получаем владельцев деталей
	users, err := h.storage.CompletePuzzle(id)
	if err != nil {
		h.logger.Errorf("Error completing puzzle: %v", err)
		if err.Error() == "puzzle not found" {
			http.Error(w, "Puzzle not found", http.StatusNotFound)
		} else if err.Error() == "puzzle already completed" {
			http.Error(w, "Puzzle already completed", http.StatusBadRequest)
		} else {
			http.Error(w, "Error completing puzzle", http.StatusInternalServerError)
		}
		return
	}

	// Формируем ответ с информацией о владельцах для уведомления
	type UserInfo struct {
		Id        uuid.UUID `json:"id"`
		Telegramm string    `json:"telegramm"`
		FirstName string    `json:"first_name"`
		LastName  string    `json:"last_name"`
		Group     string    `json:"group"`
	}

	var userInfos []UserInfo
	for _, u := range users {
		userInfos = append(userInfos, UserInfo{
			Id:        u.Id,
			Telegramm: u.Telegramm,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Group:     u.Group,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"users_to_notify": userInfos,
	})
}

// GetPuzzlePiecesHandler возвращает все детали конкретного пазла
func (h *Handler) GetPuzzlePiecesHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetPuzzlePiecesHandler")

	// Получаем ID пазла из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Errorf("Invalid puzzle ID: %v", err)
		http.Error(w, "Invalid puzzle ID", http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли пазл
	_, err = h.storage.GetPuzzle(id)
	if err != nil {
		h.logger.Errorf("Error getting puzzle: %v", err)
		http.Error(w, "Puzzle not found", http.StatusNotFound)
		return
	}

	// Получаем детали пазла
	pieces, err := h.storage.GetPuzzlePiecesByPuzzle(id)
	if err != nil {
		h.logger.Errorf("Error getting puzzle pieces: %v", err)
		http.Error(w, "Error getting puzzle pieces", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Total  int                   `json:"total"`
		Pieces []*models.PuzzlePiece `json:"pieces"`
	}{
		Total:  len(pieces),
		Pieces: pieces,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// ==================== ОБРАБОТЧИКИ ДЛЯ ДЕТАЛЕЙ ПАЗЛОВ ====================

// GetAllPiecesHandler возвращает все детали пазлов с опциональными фильтрами
func (h *Handler) GetAllPiecesHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetAllPiecesHandler")

	// Получаем все детали из хранилища
	pieces, err := h.storage.GetAllPuzzlePieces()
	if err != nil {
		h.logger.Errorf("Error getting pieces: %v", err)
		http.Error(w, "Error getting pieces", http.StatusInternalServerError)
		return
	}

	// Применяем фильтры из query параметров
	puzzleIdStr := r.URL.Query().Get("puzzle_id")
	hasOwnerStr := r.URL.Query().Get("has_owner")

	var filteredPieces []*models.PuzzlePiece
	for _, piece := range pieces {
		// Фильтр по пазлу
		if puzzleIdStr != "" {
			puzzleId, err := strconv.Atoi(puzzleIdStr)
			if err == nil && piece.PuzzleId != puzzleId {
				continue
			}
		}

		// Фильтр по наличию владельца
		if hasOwnerStr != "" {
			hasOwner := hasOwnerStr == "true" || hasOwnerStr == "1"
			if hasOwner && piece.OwnerId == nil {
				continue
			}
			if !hasOwner && piece.OwnerId != nil {
				continue
			}
		}

		filteredPieces = append(filteredPieces, piece)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Total  int                   `json:"total"`
		Pieces []*models.PuzzlePiece `json:"pieces"`
	}{
		Total:  len(filteredPieces),
		Pieces: filteredPieces,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// GetPieceHandler возвращает информацию о конкретной детали
func (h *Handler) GetPieceHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetPieceHandler")

	// Получаем код детали из URL
	code := chi.URLParam(r, "code")
	if code == "" {
		h.logger.Error("Piece code is required")
		http.Error(w, "Piece code is required", http.StatusBadRequest)
		return
	}

	// Получаем деталь из хранилища
	piece, err := h.storage.GetPuzzlePiece(code)
	if err != nil {
		h.logger.Errorf("Error getting piece: %v", err)
		http.Error(w, "Piece not found", http.StatusNotFound)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(piece); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// RegisterPieceHandler привязывает деталь к пользователю
func (h *Handler) RegisterPieceHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered RegisterPieceHandler")

	// Получаем код детали из URL
	code := chi.URLParam(r, "code")
	if code == "" {
		h.logger.Error("Piece code is required")
		http.Error(w, "Piece code is required", http.StatusBadRequest)
		return
	}

	// Декодируем запрос для получения ID пользователя
	var request struct {
		UserID uuid.UUID `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Errorf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли пользователь
	_, err := h.storage.GetUser(request.UserID)
	if err != nil {
		h.logger.Errorf("Error getting user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Регистрируем деталь
	piece, puzzleCompleted, err := h.storage.RegisterPuzzlePiece(code, request.UserID)
	if err != nil {
		h.logger.Errorf("Error registering piece: %v", err)

		// Определяем тип ошибки и возвращаем соответствующий код
		w.Header().Set("Content-Type", "application/json")

		var errorCode int
		var errorMsg string

		switch err.Error() {
		case "piece not found":
			errorCode = models.PieceErrorNotFound
			errorMsg = "Деталь не найдена"
			w.WriteHeader(http.StatusNotFound)
		case "piece already taken":
			errorCode = models.PieceErrorAlreadyTaken
			errorMsg = "Деталь уже зарегистрирована"
			w.WriteHeader(http.StatusBadRequest)
		default:
			errorCode = models.PieceErrorNotFound
			errorMsg = "Ошибка регистрации детали"
			w.WriteHeader(http.StatusInternalServerError)
		}

		response := struct {
			Success   bool   `json:"success"`
			Error     string `json:"error"`
			ErrorCode int    `json:"error_code"`
		}{
			Success:   false,
			Error:     errorMsg,
			ErrorCode: errorCode,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			h.logger.Errorf("Error encoding response: %v", err)
		}
		return
	}

	// Отправляем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Success         bool               `json:"success"`
		Piece           *models.PuzzlePiece `json:"piece"`
		PuzzleCompleted bool               `json:"puzzle_completed"`
	}{
		Success:         true,
		Piece:           piece,
		PuzzleCompleted: puzzleCompleted,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// AddPiecesHandler добавляет новые детали (массовый импорт)
func (h *Handler) AddPiecesHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered AddPiecesHandler")

	// Декодируем запрос
	var request struct {
		Pieces []struct {
			Code        string `json:"code"`
			PuzzleId    int    `json:"puzzle_id"`
			PieceNumber int    `json:"piece_number"`
		} `json:"pieces"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Errorf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Создаем объекты деталей
	pieces := make([]*models.PuzzlePiece, len(request.Pieces))
	for i, p := range request.Pieces {
		pieces[i] = &models.PuzzlePiece{
			Code:        p.Code,
			PuzzleId:    p.PuzzleId,
			PieceNumber: p.PieceNumber,
			OwnerId:     nil,
			RegisteredAt: nil,
		}
	}

	// Добавляем детали в хранилище
	if err := h.storage.AddPuzzlePieces(pieces); err != nil {
		h.logger.Errorf("Error adding pieces: %v", err)
		http.Error(w, "Error adding pieces", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := struct {
		Success bool `json:"success"`
		Added   int  `json:"added"`
	}{
		Success: true,
		Added:   len(pieces),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// ==================== ОБРАБОТЧИКИ ДЛЯ СТАТИСТИКИ ====================

// GetLotteryStatsHandler возвращает статистику для розыгрыша
func (h *Handler) GetLotteryStatsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetLotteryStatsHandler")

	// Получаем всех пользователей
	users, err := h.storage.GetAllUsers()
	if err != nil {
		h.logger.Errorf("Error getting users: %v", err)
		http.Error(w, "Error getting users", http.StatusInternalServerError)
		return
	}

	// Получаем все пазлы
	puzzles, err := h.storage.GetAllPuzzles()
	if err != nil {
		h.logger.Errorf("Error getting puzzles: %v", err)
		http.Error(w, "Error getting puzzles", http.StatusInternalServerError)
		return
	}

	// Собираем статистику для каждого пользователя
	type UserStats struct {
		UserId          uuid.UUID `json:"user_id"`
		FirstName       string    `json:"first_name"`
		LastName        string    `json:"last_name"`
		Group           string    `json:"group"`
		TotalPieces     int       `json:"total_pieces"`
		CompletedPieces int       `json:"completed_pieces"`
	}

	var userStats []UserStats
	for _, user := range users {
		totalPieces, _ := h.storage.GetUserPieceCount(user.Id)
		completedPieces, _ := h.storage.GetUserCompletedPuzzlePieceCount(user.Id)

		userStats = append(userStats, UserStats{
			UserId:          user.Id,
			FirstName:       user.FirstName,
			LastName:        user.LastName,
			Group:           user.Group,
			TotalPieces:     totalPieces,
			CompletedPieces: completedPieces,
		})
	}

	// Считаем собранные пазлы
	completedPuzzles := 0
	for _, puzzle := range puzzles {
		if puzzle.IsCompleted {
			completedPuzzles++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		TotalUsers       int         `json:"total_users"`
		TotalPuzzles     int         `json:"total_puzzles"`
		CompletedPuzzles int         `json:"completed_puzzles"`
		Users            []UserStats `json:"users"`
	}{
		TotalUsers:       len(users),
		TotalPuzzles:     len(puzzles),
		CompletedPuzzles: completedPuzzles,
		Users:            userStats,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Errorf("Error encoding response: %v", err)
	}
}

// ==================== ОБРАБОТЧИКИ ДЛЯ АДМИНИСТРАТОРОВ ====================

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

// GetAdminsHandler возвращает список всех администраторов
func (h *Handler) GetAdminsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetAdminsHandler")

	admins, err := h.storage.GetAllAdmins()
	if err != nil {
		h.logger.Errorf("Ошибка получения списка администраторов: %v", err)
		http.Error(w, "Ошибка получения списка администраторов", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"admins": admins,
	})
}

// AddAdminHandler добавляет нового администратора
func (h *Handler) AddAdminHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered AddAdminHandler")

	var req struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if req.ID == 0 {
		http.Error(w, "ID администратора обязателен", http.StatusBadRequest)
		return
	}

	admin := &models.Admin{
		ID:       req.ID,
		Name:     req.Name,
		Username: req.Username,
		IsActive: true,
	}

	if err := h.storage.AddAdmin(admin); err != nil {
		h.logger.Errorf("Ошибка добавления администратора: %v", err)
		http.Error(w, "Ошибка добавления администратора", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"admin":   admin,
	})
}

// DeleteAdminHandler удаляет администратора
func (h *Handler) DeleteAdminHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered DeleteAdminHandler")

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "ID администратора не указан", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.logger.Errorf("Неверный формат ID: %v", err)
		http.Error(w, "Неверный формат ID", http.StatusBadRequest)
		return
	}

	if err := h.storage.DeleteAdmin(id); err != nil {
		h.logger.Errorf("Ошибка удаления администратора: %v", err)
		http.Error(w, "Ошибка удаления администратора", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// ==================== HANDLERS ДЛЯ УВЕДОМЛЕНИЙ ====================

// CreateNotificationHandler создает новое уведомление для рассылки
func (h *Handler) CreateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered CreateNotificationHandler")

	var req struct {
		Message     string      `json:"message"`
		Group       string      `json:"group,omitempty"`
		UserIds     []uuid.UUID `json:"user_ids,omitempty"`
		Attachments []string    `json:"attachments,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Сообщение не может быть пустым", http.StatusBadRequest)
		return
	}

	notification := &models.Notification{
		Id:          uuid.New(),
		Message:     req.Message,
		Group:       req.Group,
		UserIds:     req.UserIds,
		Attachments: req.Attachments,
		Status:      models.NotificationPending,
		CreatedAt:   time.Now(),
		SentCount:   0,
		ErrorCount:  0,
	}

	if err := h.storage.AddNotification(notification); err != nil {
		h.logger.Errorf("Ошибка создания уведомления: %v", err)
		http.Error(w, "Ошибка создания уведомления", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"notification": notification,
	})
}

// GetNotificationsHandler возвращает все уведомления
func (h *Handler) GetNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetNotificationsHandler")

	notifications, err := h.storage.GetAllNotifications()
	if err != nil {
		h.logger.Errorf("Ошибка получения уведомлений: %v", err)
		http.Error(w, "Ошибка получения уведомлений", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":         len(notifications),
		"notifications": notifications,
	})
}

// GetPendingNotificationsHandler возвращает ожидающие отправки уведомления с пользователями
func (h *Handler) GetPendingNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	notifications, err := h.storage.GetPendingNotifications()
	if err != nil {
		h.logger.Errorf("Ошибка получения ожидающих уведомлений: %v", err)
		http.Error(w, "Ошибка получения уведомлений", http.StatusInternalServerError)
		return
	}

	// Для каждого уведомления получаем список пользователей
	type NotificationWithUsers struct {
		*models.Notification
		Users []*models.User `json:"users"`
	}

	var result []NotificationWithUsers
	for _, n := range notifications {
		var filteredUsers []*models.User

		// Если указана группа - фильтруем всех пользователей по группе
		if n.Group != "" {
			users, err := h.storage.GetAllUsers()
			if err != nil {
				h.logger.Errorf("Ошибка получения пользователей: %v", err)
				continue
			}
			for _, u := range users {
				if !u.Deleted && u.Group == n.Group {
					filteredUsers = append(filteredUsers, u)
				}
			}
		} else if len(n.UserIds) > 0 {
			// Если указан список конкретных пользователей
			for _, userId := range n.UserIds {
				user, err := h.storage.GetUser(userId)
				if err != nil {
					h.logger.Errorf("Ошибка получения пользователя %s: %v", userId, err)
					continue
				}
				if !user.Deleted {
					filteredUsers = append(filteredUsers, user)
				}
			}
		} else {
			// Если ни группа, ни список пользователей не указаны - отправляем всем
			users, err := h.storage.GetAllUsers()
			if err != nil {
				h.logger.Errorf("Ошибка получения пользователей: %v", err)
				continue
			}
			for _, u := range users {
				if !u.Deleted {
					filteredUsers = append(filteredUsers, u)
				}
			}
		}

		result = append(result, NotificationWithUsers{
			Notification: n,
			Users:        filteredUsers,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":         len(result),
		"notifications": result,
	})
}

// GetNotificationHandler возвращает уведомление по ID
func (h *Handler) GetNotificationHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered GetNotificationHandler")

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "ID уведомления не указан", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Errorf("Неверный формат ID: %v", err)
		http.Error(w, "Неверный формат ID", http.StatusBadRequest)
		return
	}

	notification, err := h.storage.GetNotification(id)
	if err != nil {
		h.logger.Errorf("Уведомление не найдено: %v", err)
		http.Error(w, "Уведомление не найдено", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

// UpdateNotificationHandler обновляет статус уведомления
func (h *Handler) UpdateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered UpdateNotificationHandler")

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "ID уведомления не указан", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Errorf("Неверный формат ID: %v", err)
		http.Error(w, "Неверный формат ID", http.StatusBadRequest)
		return
	}

	notification, err := h.storage.GetNotification(id)
	if err != nil {
		h.logger.Errorf("Уведомление не найдено: %v", err)
		http.Error(w, "Уведомление не найдено", http.StatusNotFound)
		return
	}

	var req struct {
		Status     *string `json:"status,omitempty"`
		SentCount  *int    `json:"sent_count,omitempty"`
		ErrorCount *int    `json:"error_count,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if req.Status != nil {
		notification.Status = models.NotificationStatus(*req.Status)
		if notification.Status == models.NotificationSent {
			now := time.Now()
			notification.SentAt = &now
		}
	}
	if req.SentCount != nil {
		notification.SentCount = *req.SentCount
	}
	if req.ErrorCount != nil {
		notification.ErrorCount = *req.ErrorCount
	}

	if err := h.storage.UpdateNotification(notification); err != nil {
		h.logger.Errorf("Ошибка обновления уведомления: %v", err)
		http.Error(w, "Ошибка обновления уведомления", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"notification": notification,
	})
}
