package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AdminsList представляет список администраторов
type AdminsList struct {
	Admins []*models.Admin `json:"admins"`
}

// AdminHandler обрабатывает запросы к админке
type AdminHandler struct {
	store       storage.Storage
	logger      logger.Logger
	passwordMgr *PasswordManager
	jwtManager  *JWTManager
	authMw      *AuthMiddleware
	staticDir   string
	adminsPath  string
}

// LoginRequest представляет запрос на вход
type LoginRequest struct {
	Password string `json:"password"`
}

// LoginResponse представляет ответ на запрос входа
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// UserResponse представляет ответ с информацией о пользователе
type UserResponse struct {
	ID               uuid.UUID `json:"id"`
	Telegramm        string    `json:"telegramm"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	MiddleName       string    `json:"middle_name"`
	Group            string    `json:"group"`
	PieceCount       int       `json:"piece_count"`
	RegistrationTime time.Time `json:"registration_time"`
}

// UsersResponse представляет ответ со списком пользователей
type UsersResponse struct {
	Total int            `json:"total"`
	Users []UserResponse `json:"users"`
}

// PieceResponse представляет ответ с информацией о детали пазла
type PieceResponse struct {
	Code         string     `json:"code"`
	PuzzleId     int        `json:"puzzle_id"`
	PieceNumber  int        `json:"piece_number"`
	OwnerId      *uuid.UUID `json:"owner_id,omitempty"`
	RegisteredAt *time.Time `json:"registered_at,omitempty"`
}

// PiecesResponse представляет ответ со списком деталей пазлов
type PiecesResponse struct {
	Total  int             `json:"total"`
	Pieces []PieceResponse `json:"pieces"`
}

// PuzzleResponse представляет ответ с информацией о пазле
type PuzzleResponse struct {
	Id          int        `json:"id"`
	IsCompleted bool       `json:"is_completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	TotalPieces int        `json:"total_pieces"`
	OwnedPieces int        `json:"owned_pieces"`
}

// PuzzlesResponse представляет ответ со списком пазлов
type PuzzlesResponse struct {
	Total   int              `json:"total"`
	Puzzles []PuzzleResponse `json:"puzzles"`
}

// NewAdminHandler создает новый обработчик админки
func NewAdminHandler(store storage.Storage, logger logger.Logger, dataDir string, jwtSecret string) *AdminHandler {
	passwordMgr := NewPasswordManager(dataDir)
	jwtManager := NewJWTManager(jwtSecret)

	// Получаем пути из переменных окружения или используем значения по умолчанию
	staticDir := os.Getenv("ADMIN_STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static/admin"
	}

	adminsPath := os.Getenv("ADMIN_ADMINS_PATH")
	if adminsPath == "" {
		adminsPath = "./cmd/telegrambot/admin/admins.json"
	}

	logger.Infof("Используем пути: staticDir=%s, adminsPath=%s", staticDir, adminsPath)

	handler := &AdminHandler{
		store:       store,
		logger:      logger,
		passwordMgr: passwordMgr,
		jwtManager:  jwtManager,
		staticDir:   staticDir,
		adminsPath:  adminsPath,
	}

	handler.authMw = NewAuthMiddleware(jwtManager, logger)

	return handler
}

// RegisterRoutes регистрирует маршруты для админки
func (ah *AdminHandler) RegisterRoutes(r chi.Router) {
	// Создаем подмаршрутизатор для статических файлов с CORS
	r.Route("/admin", func(r chi.Router) {
		// Middleware для CORS
		r.Use(ah.corsMiddleware)
		// Middleware для установки правильного MIME-типа
		r.Use(ah.mimeTypeMiddleware)

		// Обработчик для корневого пути
		r.Get("/", ah.handleAdminRoot)

		// Статические файлы
		fs := http.FileServer(http.Dir(ah.staticDir))
		r.Handle("/*", http.StripPrefix("/admin/", fs))
	})

	// Добавляем маршрут для CSS-файлов
	r.Route("/css", func(r chi.Router) {
		// Middleware для CORS
		r.Use(ah.corsMiddleware)
		// Middleware для установки правильного MIME-типа
		r.Use(ah.mimeTypeMiddleware)

		// Статические файлы CSS
		fs := http.FileServer(http.Dir(ah.staticDir + "/css"))
		r.Handle("/*", http.StripPrefix("/css/", fs))
	})

	// Добавляем маршрут для favicon.ico
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		http.ServeFile(w, r, ah.staticDir+"/favicon.ico")
	})

	// API
	r.Route("/api/admin", func(r chi.Router) {
		// Middleware для CORS
		r.Use(ah.corsMiddleware)

		r.Post("/login", ah.handleLogin)

		// Защищенные маршруты
		r.Group(func(r chi.Router) {
			r.Use(ah.authMw.Middleware)

			// Пользователи
			r.Get("/users", ah.handleUsers)
			r.Post("/users/update", ah.handleUpdateUser)
			r.Post("/users/delete", ah.handleDeleteUser)

			// Пазлы
			r.Get("/puzzles", ah.handlePuzzles)

			// Детали пазлов
			r.Get("/pieces", ah.handlePieces)
			r.Post("/pieces/add", ah.handleAddPieces)

			// Статистика для розыгрыша
			r.Get("/lottery", ah.handleLotteryStats)

			// Администраторы
			r.Get("/admins", ah.handleGetAdmins)
			r.Post("/admins/add", ah.handleAddAdmin)
			r.Post("/admins/remove", ah.handleRemoveAdmin)

			// Рассылки
			r.Get("/notifications", ah.handleGetNotifications)
			r.Post("/notifications", ah.handleCreateNotification)
			r.Post("/notifications/{id}/attachments", ah.handleUploadAttachment)

			// Библиотека файлов
			r.Get("/attachments", ah.handleGetAttachments)
			r.Post("/attachments", ah.handleUploadToLibrary)
			r.Patch("/attachments/{id}", ah.handleRenameAttachment)
			r.Delete("/attachments/{id}", ah.handleDeleteAttachment)
			r.Get("/attachments/{id}/file", ah.handleServeAttachment)
		})
	})
}

// handleAdminRoot обрабатывает запрос к корневому пути админки
func (ah *AdminHandler) handleAdminRoot(w http.ResponseWriter, r *http.Request) {
	// Проверяем, аутентифицирован ли пользователь
	cookie, err := r.Cookie("admin_token")
	if err != nil || cookie.Value == "" {
		// Если нет, перенаправляем на страницу входа
		http.ServeFile(w, r, ah.staticDir+"/login.html")
		return
	}

	// Проверяем токен
	_, err = ah.jwtManager.ValidateToken(cookie.Value)
	if err != nil {
		// Если токен недействителен, перенаправляем на страницу входа
		http.ServeFile(w, r, ah.staticDir+"/login.html")
		return
	}

	// Получаем запрошенную страницу из параметра запроса
	page := r.URL.Query().Get("page")
	switch page {
	case "puzzles":
		http.ServeFile(w, r, ah.staticDir+"/puzzles.html")
	case "pieces":
		http.ServeFile(w, r, ah.staticDir+"/pieces.html")
	case "admins":
		http.ServeFile(w, r, ah.staticDir+"/admins.html")
	default:
		// По умолчанию показываем страницу пользователей
		http.ServeFile(w, r, ah.staticDir+"/users.html")
	}
}

// corsMiddleware добавляет заголовки CORS
func (ah *AdminHandler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Создаем обертку для ResponseWriter, чтобы добавить заголовки CORS
		corsWriter := &corsResponseWriter{ResponseWriter: w}
		corsWriter.Header().Set("Access-Control-Allow-Origin", "*")
		corsWriter.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		corsWriter.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			corsWriter.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(corsWriter, r)
	})
}

// corsResponseWriter - обертка для http.ResponseWriter, которая добавляет заголовки CORS
type corsResponseWriter struct {
	http.ResponseWriter
}

// WriteHeader переопределяет метод WriteHeader для добавления заголовков CORS
func (w *corsResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write переопределяет метод Write для корректной записи содержимого
func (w *corsResponseWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

// mimeTypeMiddleware устанавливает правильный MIME-тип для статических файлов
func (ah *AdminHandler) mimeTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Устанавливаем правильный MIME-тип для CSS-файлов
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(r.URL.Path, ".html") {
			w.Header().Set("Content-Type", "text/html")
		} else if strings.HasSuffix(r.URL.Path, ".json") {
			w.Header().Set("Content-Type", "application/json")
		} else if strings.HasSuffix(r.URL.Path, ".png") {
			w.Header().Set("Content-Type", "image/png")
		} else if strings.HasSuffix(r.URL.Path, ".jpg") || strings.HasSuffix(r.URL.Path, ".jpeg") {
			w.Header().Set("Content-Type", "image/jpeg")
		} else if strings.HasSuffix(r.URL.Path, ".gif") {
			w.Header().Set("Content-Type", "image/gif")
		} else if strings.HasSuffix(r.URL.Path, ".svg") {
			w.Header().Set("Content-Type", "image/svg+xml")
		} else if strings.HasSuffix(r.URL.Path, ".ico") {
			w.Header().Set("Content-Type", "image/x-icon")
		}

		next.ServeHTTP(w, r)
	})
}

// handleLogin обрабатывает запрос на вход
func (ah *AdminHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Проверяем пароль
	if err := ah.passwordMgr.VerifyPassword(req.Password); err != nil {
		if err == ErrPasswordNotSet {
			// Если пароль не установлен, инициализируем его
			password, err := ah.passwordMgr.InitializeDefaultPassword()
			if err != nil {
				ah.logger.Errorf("Ошибка инициализации пароля: %v", err)
				http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
				return
			}

			ah.logger.Infof("Инициализирован пароль по умолчанию: %s", password)

			// Проверяем, совпадает ли введенный пароль с сгенерированным
			if req.Password != password {
				http.Error(w, "Неверный пароль", http.StatusUnauthorized)
				return
			}
		} else {
			http.Error(w, "Неверный пароль", http.StatusUnauthorized)
			return
		}
	}

	// Генерируем токен
	token, err := ah.jwtManager.GenerateToken()
	if err != nil {
		ah.logger.Errorf("Ошибка генерации токена: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Устанавливаем cookie с токеном
	expirationTime := time.Now().Add(tokenExpiration)
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_token",
		Value:    token,
		Expires:  expirationTime,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	// Отправляем ответ
	resp := LoginResponse{
		Token:     token,
		ExpiresAt: expirationTime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUsers обрабатывает запрос на получение списка пользователей
func (ah *AdminHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры фильтрации
	year := r.URL.Query().Get("year")
	group := r.URL.Query().Get("group")

	// Получаем всех пользователей
	users, err := ah.store.GetAllUsers()
	if err != nil {
		ah.logger.Errorf("Ошибка получения пользователей: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Фильтруем пользователей
	var filteredUsers []*models.User
	for _, user := range users {
		// Фильтр по году
		if year != "" {
			yearInt, err := strconv.Atoi(year)
			if err == nil {
				if user.RegistrationTime.Year() != yearInt {
					continue
				}
			}
		}

		// Фильтр по группе
		if group != "" && user.Group != group {
			continue
		}

		// Пропускаем удаленных пользователей
		if user.Deleted {
			continue
		}

		filteredUsers = append(filteredUsers, user)
	}

	// Преобразуем пользователей в ответ
	var userResponses []UserResponse
	for _, user := range filteredUsers {
		// Получаем количество деталей пользователя
		pieceCount, _ := ah.store.GetUserPieceCount(user.Id)

		userResponses = append(userResponses, UserResponse{
			ID:               user.Id,
			Telegramm:        user.Telegramm,
			FirstName:        user.FirstName,
			LastName:         user.LastName,
			MiddleName:       user.MiddleName,
			Group:            user.Group,
			PieceCount:       pieceCount,
			RegistrationTime: user.RegistrationTime,
		})
	}

	// Отправляем ответ
	resp := UsersResponse{
		Total: len(userResponses),
		Users: userResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateUser обрабатывает запрос на обновление пользователя
func (ah *AdminHandler) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получаем ID пользователя
	userIDStr := r.URL.Query().Get("id")
	if userIDStr == "" {
		http.Error(w, "ID пользователя не указан", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Неверный формат ID пользователя", http.StatusBadRequest)
		return
	}

	// Получаем пользователя
	user, err := ah.store.GetUser(userID)
	if err != nil {
		ah.logger.Errorf("Ошибка получения пользователя: %v", err)
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Получаем данные для обновления
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Обновляем данные пользователя
	if firstName, ok := updateData["first_name"].(string); ok {
		user.FirstName = firstName
	}

	if lastName, ok := updateData["last_name"].(string); ok {
		user.LastName = lastName
	}

	if middleName, ok := updateData["middle_name"].(string); ok {
		user.MiddleName = middleName
	}

	if group, ok := updateData["group"].(string); ok {
		user.Group = group
	}

	// Сохраняем обновленного пользователя
	if err := ah.store.UpdateUser(user); err != nil {
		ah.logger.Errorf("Ошибка обновления пользователя: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Получаем количество деталей пользователя
	pieceCount, _ := ah.store.GetUserPieceCount(user.Id)

	// Отправляем ответ
	userResp := UserResponse{
		ID:               user.Id,
		Telegramm:        user.Telegramm,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
		MiddleName:       user.MiddleName,
		Group:            user.Group,
		PieceCount:       pieceCount,
		RegistrationTime: user.RegistrationTime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userResp)
}

// handlePuzzles обрабатывает запрос на получение списка пазлов
func (ah *AdminHandler) handlePuzzles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получаем все пазлы
	puzzles, err := ah.store.GetAllPuzzles()
	if err != nil {
		ah.logger.Errorf("Ошибка получения пазлов: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Преобразуем пазлы в ответ
	var puzzleResponses []PuzzleResponse
	for _, puzzle := range puzzles {
		// Получаем детали пазла для подсчета прогресса
		pieces, _ := ah.store.GetPuzzlePiecesByPuzzle(puzzle.Id)
		ownedCount := 0
		for _, p := range pieces {
			if p.OwnerId != nil {
				ownedCount++
			}
		}

		puzzleResponses = append(puzzleResponses, PuzzleResponse{
			Id:          puzzle.Id,
			IsCompleted: puzzle.IsCompleted,
			CompletedAt: puzzle.CompletedAt,
			TotalPieces: len(pieces),
			OwnedPieces: ownedCount,
		})
	}

	// Отправляем ответ
	resp := PuzzlesResponse{
		Total:   len(puzzleResponses),
		Puzzles: puzzleResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handlePieces обрабатывает запрос на получение списка деталей пазлов
func (ah *AdminHandler) handlePieces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры фильтрации
	puzzleIdStr := r.URL.Query().Get("puzzle_id")
	hasOwnerStr := r.URL.Query().Get("has_owner")

	// Получаем все детали
	pieces, err := ah.store.GetAllPuzzlePieces()
	if err != nil {
		ah.logger.Errorf("Ошибка получения деталей: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Фильтруем детали
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

	// Преобразуем детали в ответ
	var pieceResponses []PieceResponse
	for _, piece := range filteredPieces {
		pieceResponses = append(pieceResponses, PieceResponse{
			Code:         piece.Code,
			PuzzleId:     piece.PuzzleId,
			PieceNumber:  piece.PieceNumber,
			OwnerId:      piece.OwnerId,
			RegisteredAt: piece.RegisteredAt,
		})
	}

	// Отправляем ответ
	resp := PiecesResponse{
		Total:  len(pieceResponses),
		Pieces: pieceResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleAddPieces обрабатывает запрос на добавление деталей пазлов
func (ah *AdminHandler) handleAddPieces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Декодируем запрос
	var request struct {
		Pieces []struct {
			Code        string `json:"code"`
			PuzzleId    int    `json:"puzzle_id"`
			PieceNumber int    `json:"piece_number"`
		} `json:"pieces"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.logger.Errorf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Создаем объекты деталей
	pieces := make([]*models.PuzzlePiece, len(request.Pieces))
	for i, p := range request.Pieces {
		pieces[i] = &models.PuzzlePiece{
			Code:         p.Code,
			PuzzleId:     p.PuzzleId,
			PieceNumber:  p.PieceNumber,
			OwnerId:      nil,
			RegisteredAt: nil,
		}
	}

	// Добавляем детали в хранилище
	if err := ah.store.AddPuzzlePieces(pieces); err != nil {
		ah.logger.Errorf("Ошибка добавления деталей: %v", err)
		http.Error(w, "Ошибка добавления деталей", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"added":   len(pieces),
	})
}

// handleLotteryStats обрабатывает запрос на получение статистики для розыгрыша
func (ah *AdminHandler) handleLotteryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получаем всех пользователей
	users, err := ah.store.GetAllUsers()
	if err != nil {
		ah.logger.Errorf("Ошибка получения пользователей: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Получаем все пазлы
	puzzles, err := ah.store.GetAllPuzzles()
	if err != nil {
		ah.logger.Errorf("Ошибка получения пазлов: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
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
		if user.Deleted {
			continue
		}

		totalPieces, _ := ah.store.GetUserPieceCount(user.Id)
		completedPieces, _ := ah.store.GetUserCompletedPuzzlePieceCount(user.Id)

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

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_users":       len(userStats),
		"total_puzzles":     len(puzzles),
		"completed_puzzles": completedPuzzles,
		"users":             userStats,
	})
}

// handleGetAdmins обрабатывает запрос на получение списка администраторов
func (ah *AdminHandler) handleGetAdmins(w http.ResponseWriter, r *http.Request) {
	ah.logger.Info("Запрос на получение списка администраторов")

	// Получаем список администраторов из базы данных
	admins, err := ah.store.GetAllAdmins()
	if err != nil {
		ah.logger.Errorf("Ошибка получения списка администраторов: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := AdminsList{
		Admins: admins,
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAddAdmin обрабатывает запрос на добавление администратора
func (ah *AdminHandler) handleAddAdmin(w http.ResponseWriter, r *http.Request) {
	ah.logger.Info("Запрос на добавление администратора")

	// Декодируем запрос
	var request struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.logger.Errorf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Создаем нового администратора
	admin := &models.Admin{
		ID:       request.ID,
		Name:     request.Name,
		Username: request.Username,
		IsActive: true,
	}

	// Добавляем администратора в базу данных
	if err := ah.store.AddAdmin(admin); err != nil {
		ah.logger.Errorf("Ошибка добавления администратора: %v", err)
		http.Error(w, "Ошибка добавления администратора", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleRemoveAdmin обрабатывает запрос на удаление администратора
func (ah *AdminHandler) handleRemoveAdmin(w http.ResponseWriter, r *http.Request) {
	ah.logger.Info("Запрос на удаление администратора")

	// Декодируем запрос
	var request struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.logger.Errorf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Удаляем администратора из базы данных
	if err := ah.store.DeleteAdmin(request.ID); err != nil {
		ah.logger.Errorf("Ошибка удаления администратора: %v", err)
		http.Error(w, "Ошибка удаления администратора", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleDeleteUser обрабатывает запрос на удаление пользователя
func (ah *AdminHandler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	ah.logger.Info("Запрос на удаление пользователя")

	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Декодируем запрос
	var request struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.logger.Errorf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Парсим UUID пользователя
	userID, err := uuid.Parse(request.ID)
	if err != nil {
		ah.logger.Errorf("Неверный формат ID пользователя: %v", err)
		http.Error(w, "Неверный формат ID пользователя", http.StatusBadRequest)
		return
	}

	// Удаляем пользователя
	if err := ah.store.DeleteUser(userID); err != nil {
		ah.logger.Errorf("Ошибка удаления пользователя: %v", err)
		http.Error(w, "Ошибка удаления пользователя", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleGetNotifications обрабатывает запрос на получение списка рассылок
func (ah *AdminHandler) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	ah.logger.Info("Запрос на получение списка рассылок")

	notifications, err := ah.store.GetAllNotifications()
	if err != nil {
		ah.logger.Errorf("Ошибка получения рассылок: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"notifications": notifications,
	})
}

// handleCreateNotification обрабатывает запрос на создание рассылки
func (ah *AdminHandler) handleCreateNotification(w http.ResponseWriter, r *http.Request) {
	ah.logger.Info("Запрос на создание рассылки")

	var request struct {
		Message string   `json:"message"`
		Group   string   `json:"group"`
		UserIds []string `json:"user_ids,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.logger.Errorf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if request.Message == "" {
		http.Error(w, "Сообщение не может быть пустым", http.StatusBadRequest)
		return
	}

	// Конвертируем user_ids в []uuid.UUID
	var userIds []uuid.UUID
	for _, idStr := range request.UserIds {
		if id, err := uuid.Parse(idStr); err == nil {
			userIds = append(userIds, id)
		}
	}

	notification := &models.Notification{
		Id:        uuid.New(),
		Message:   request.Message,
		Group:     request.Group,
		UserIds:   userIds,
		Status:    models.NotificationPending,
		CreatedAt: time.Now(),
	}

	if err := ah.store.AddNotification(notification); err != nil {
		ah.logger.Errorf("Ошибка создания рассылки: %v", err)
		http.Error(w, "Ошибка создания рассылки", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"notification": notification,
	})
}

// handleUploadAttachment обрабатывает загрузку файла для рассылки (сохраняет в библиотеку)
func (ah *AdminHandler) handleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	notificationId := chi.URLParam(r, "id")
	ah.logger.Infof("Загрузка вложения для рассылки %s", notificationId)

	// Парсим notification ID
	notifUUID, err := uuid.Parse(notificationId)
	if err != nil {
		http.Error(w, "Неверный ID рассылки", http.StatusBadRequest)
		return
	}

	// Получаем notification
	notification, err := ah.store.GetNotification(notifUUID)
	if err != nil {
		ah.logger.Errorf("Рассылка не найдена: %v", err)
		http.Error(w, "Рассылка не найдена", http.StatusNotFound)
		return
	}

	// Парсим multipart form (10 MB лимит)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		ah.logger.Errorf("Ошибка парсинга формы: %v", err)
		http.Error(w, "Ошибка загрузки файла", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		ah.logger.Errorf("Ошибка получения файла: %v", err)
		http.Error(w, "Файл не найден в запросе", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Создаём директорию библиотеки
	libraryDir := filepath.Join("data", "library")
	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		ah.logger.Errorf("Ошибка создания директории: %v", err)
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}

	// Генерируем уникальное имя файла
	originalFilename := filepath.Base(header.Filename)
	fileId := uuid.New()
	ext := filepath.Ext(originalFilename)
	storedFilename := fileId.String() + ext
	dstPath := filepath.Join(libraryDir, storedFilename)

	// Сохраняем файл
	dst, err := os.Create(dstPath)
	if err != nil {
		ah.logger.Errorf("Ошибка создания файла: %v", err)
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		ah.logger.Errorf("Ошибка записи файла: %v", err)
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}

	// Сохраняем полный путь к файлу в notification.Attachments
	notification.Attachments = append(notification.Attachments, dstPath)
	if err := ah.store.UpdateNotification(notification); err != nil {
		ah.logger.Errorf("Ошибка обновления рассылки: %v", err)
		http.Error(w, "Ошибка обновления рассылки", http.StatusInternalServerError)
		return
	}

	ah.logger.Infof("Файл %s загружен в библиотеку как %s для рассылки %s", originalFilename, dstPath, notificationId)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"filename": originalFilename,
		"path":     dstPath,
		"size":     written,
	})
}

// ==================== БИБЛИОТЕКА ФАЙЛОВ ====================

// handleGetAttachments возвращает список всех файлов в библиотеке
func (ah *AdminHandler) handleGetAttachments(w http.ResponseWriter, r *http.Request) {
	ah.logger.Info("Запрос на получение списка файлов")

	// Синхронизируем папку library с БД
	ah.syncLibraryWithDB()

	attachments, err := ah.store.GetAllAttachments()
	if err != nil {
		ah.logger.Errorf("Ошибка получения файлов: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"attachments": attachments,
	})
}

// syncLibraryWithDB синхронизирует папку data/library с базой данных
func (ah *AdminHandler) syncLibraryWithDB() {
	libraryDir := filepath.Join("data", "library")

	ah.logger.Info("Начинаем синхронизацию библиотеки с БД")

	// Создаём директорию библиотеки если её нет
	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		ah.logger.Errorf("Ошибка создания директории library: %v", err)
		return
	}

	// Получаем все записи из БД
	dbAttachments, err := ah.store.GetAllAttachments()
	if err != nil {
		ah.logger.Errorf("Ошибка получения списка файлов из БД: %v", err)
		return
	}

	// Карта файлов в БД по пути
	dbFiles := make(map[string]*models.Attachment)
	for _, att := range dbAttachments {
		dbFiles[att.StorePath] = att
	}

	// Сканируем папку library
	filesInFolder := make(map[string]bool)
	addedCount := 0

	err = filepath.Walk(libraryDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			ah.logger.Errorf("Ошибка доступа к %s: %v", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		filesInFolder[path] = true

		// Если файл есть в папке, но нет в БД - добавляем
		if _, exists := dbFiles[path]; !exists {
			filename := info.Name()
			ext := filepath.Ext(filename)

			// Пытаемся извлечь UUID из имени файла
			baseName := strings.TrimSuffix(filename, ext)
			attachmentId, err := uuid.Parse(baseName)
			if err != nil {
				// Если имя не UUID, генерируем новый
				attachmentId = uuid.New()
			}

			attachment := &models.Attachment{
				Id:        attachmentId,
				Filename:  filename,
				StorePath: path,
				MimeType:  getMimeType(ext),
				Size:      info.Size(),
				CreatedAt: info.ModTime(),
			}

			if err := ah.store.AddAttachment(attachment); err != nil {
				ah.logger.Errorf("Ошибка добавления файла %s в БД: %v", path, err)
			} else {
				addedCount++
				ah.logger.Infof("Добавлен файл в БД: %s", path)
			}
		}

		return nil
	})

	if err != nil {
		ah.logger.Errorf("Ошибка обхода директории: %v", err)
	}

	// Удаляем из БД записи для файлов, которых нет в папке
	removedCount := 0
	for path, att := range dbFiles {
		if !filesInFolder[path] {
			if err := ah.store.DeleteAttachment(att.Id); err != nil {
				ah.logger.Errorf("Ошибка удаления записи %s из БД: %v", att.Id, err)
			} else {
				removedCount++
				ah.logger.Infof("Удалена запись из БД (файл отсутствует): %s", path)
			}
		}
	}

	ah.logger.Infof("Синхронизация завершена: добавлено %d, удалено %d", addedCount, removedCount)
}

// handleUploadToLibrary загружает файл в библиотеку
func (ah *AdminHandler) handleUploadToLibrary(w http.ResponseWriter, r *http.Request) {
	ah.logger.Info("Загрузка файла в библиотеку")

	// Парсим multipart form (10 MB лимит)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		ah.logger.Errorf("Ошибка парсинга формы: %v", err)
		http.Error(w, "Ошибка загрузки файла", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		ah.logger.Errorf("Ошибка получения файла: %v", err)
		http.Error(w, "Файл не найден в запросе", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Генерируем ID для файла
	attachmentId := uuid.New()

	// Создаём директорию для библиотеки
	libraryDir := filepath.Join("data", "library")
	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		ah.logger.Errorf("Ошибка создания директории: %v", err)
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}

	// Определяем расширение файла
	ext := filepath.Ext(header.Filename)
	storePath := filepath.Join(libraryDir, attachmentId.String()+ext)

	// Сохраняем файл
	dst, err := os.Create(storePath)
	if err != nil {
		ah.logger.Errorf("Ошибка создания файла: %v", err)
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		ah.logger.Errorf("Ошибка записи файла: %v", err)
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}

	// Определяем MIME-тип
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Создаём запись в БД
	attachment := &models.Attachment{
		Id:        attachmentId,
		Filename:  header.Filename,
		StorePath: storePath,
		MimeType:  mimeType,
		Size:      written,
		CreatedAt: time.Now(),
	}

	if err := ah.store.AddAttachment(attachment); err != nil {
		ah.logger.Errorf("Ошибка сохранения в БД: %v", err)
		os.Remove(storePath) // Удаляем файл если не удалось сохранить в БД
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}

	ah.logger.Infof("Файл %s загружен в библиотеку с ID %s", header.Filename, attachmentId)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"attachment": attachment,
	})
}

// handleRenameAttachment переименовывает файл
func (ah *AdminHandler) handleRenameAttachment(w http.ResponseWriter, r *http.Request) {
	attachmentId := chi.URLParam(r, "id")
	ah.logger.Infof("Переименование файла %s", attachmentId)

	attachmentUUID, err := uuid.Parse(attachmentId)
	if err != nil {
		http.Error(w, "Неверный ID файла", http.StatusBadRequest)
		return
	}

	var request struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if request.Filename == "" {
		http.Error(w, "Имя файла не может быть пустым", http.StatusBadRequest)
		return
	}

	attachment, err := ah.store.GetAttachment(attachmentUUID)
	if err != nil {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	attachment.Filename = request.Filename
	if err := ah.store.UpdateAttachment(attachment); err != nil {
		ah.logger.Errorf("Ошибка обновления файла: %v", err)
		http.Error(w, "Ошибка обновления файла", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"attachment": attachment,
	})
}

// handleDeleteAttachment удаляет файл из библиотеки
func (ah *AdminHandler) handleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	attachmentId := chi.URLParam(r, "id")
	ah.logger.Infof("Удаление файла %s", attachmentId)

	attachmentUUID, err := uuid.Parse(attachmentId)
	if err != nil {
		http.Error(w, "Неверный ID файла", http.StatusBadRequest)
		return
	}

	attachment, err := ah.store.GetAttachment(attachmentUUID)
	if err != nil {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	// Удаляем файл с диска
	if err := os.Remove(attachment.StorePath); err != nil && !os.IsNotExist(err) {
		ah.logger.Errorf("Ошибка удаления файла с диска: %v", err)
	}

	// Удаляем запись из БД
	if err := ah.store.DeleteAttachment(attachmentUUID); err != nil {
		ah.logger.Errorf("Ошибка удаления из БД: %v", err)
		http.Error(w, "Ошибка удаления файла", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// getMimeType возвращает MIME-тип по расширению файла
func getMimeType(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

// handleServeAttachment отдаёт файл для превью/скачивания
func (ah *AdminHandler) handleServeAttachment(w http.ResponseWriter, r *http.Request) {
	attachmentId := chi.URLParam(r, "id")

	attachmentUUID, err := uuid.Parse(attachmentId)
	if err != nil {
		http.Error(w, "Неверный ID файла", http.StatusBadRequest)
		return
	}

	attachment, err := ah.store.GetAttachment(attachmentUUID)
	if err != nil {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	// Проверяем существование файла
	if _, err := os.Stat(attachment.StorePath); os.IsNotExist(err) {
		http.Error(w, "Файл не найден на диске", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", attachment.MimeType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+attachment.Filename+"\"")
	http.ServeFile(w, r, attachment.StorePath)
}

// ServeStaticFiles обслуживает статические файлы для админки
func (ah *AdminHandler) ServeStaticFiles(w http.ResponseWriter, r *http.Request) {
	// Проверяем, запрашивается ли HTML-файл
	if r.URL.Path == "/admin" || r.URL.Path == "/admin/" {
		// Проверяем, аутентифицирован ли пользователь
		cookie, err := r.Cookie("admin_token")
		if err != nil || cookie.Value == "" {
			// Если нет, перенаправляем на страницу входа
			http.ServeFile(w, r, ah.staticDir+"/login.html")
			return
		}

		// Проверяем токен
		_, err = ah.jwtManager.ValidateToken(cookie.Value)
		if err != nil {
			// Если токен недействителен, перенаправляем на страницу входа
			http.ServeFile(w, r, ah.staticDir+"/login.html")
			return
		}

		// Получаем запрошенную страницу из параметра запроса
		page := r.URL.Query().Get("page")
		switch page {
		case "puzzles":
			http.ServeFile(w, r, ah.staticDir+"/puzzles.html")
		case "pieces":
			http.ServeFile(w, r, ah.staticDir+"/pieces.html")
		case "admins":
			http.ServeFile(w, r, ah.staticDir+"/admins.html")
		default:
			// По умолчанию показываем страницу пользователей
			http.ServeFile(w, r, ah.staticDir+"/users.html")
		}
		return
	}

	// Обслуживаем остальные статические файлы
	http.StripPrefix("/admin/", http.FileServer(http.Dir(ah.staticDir))).ServeHTTP(w, r)
}
