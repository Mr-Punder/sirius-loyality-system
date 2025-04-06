package admin

import (
	"encoding/json"
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

// AdminInfo представляет информацию об администраторе
type AdminInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name,omitempty"`
}

// AdminsList представляет список администраторов
type AdminsList struct {
	Admins []AdminInfo `json:"admins"`
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
	Points           int       `json:"points"`
	Group            string    `json:"group"`
	RegistrationTime time.Time `json:"registration_time"`
}

// UsersResponse представляет ответ со списком пользователей
type UsersResponse struct {
	Total int            `json:"total"`
	Users []UserResponse `json:"users"`
}

// CodeRequest представляет запрос на создание QR-кода
type CodeRequest struct {
	Amount  int    `json:"amount"`
	PerUser int    `json:"per_user"`
	Total   int    `json:"total"`
	Group   string `json:"group"`
}

// CodeResponse представляет ответ с информацией о QR-коде
type CodeResponse struct {
	Code         uuid.UUID `json:"code"`
	Amount       int       `json:"amount"`
	PerUser      int       `json:"per_user"`
	Total        int       `json:"total"`
	AppliedCount int       `json:"applied_count"`
	IsActive     bool      `json:"is_active"`
	Group        string    `json:"group"`
	ErrorCode    int       `json:"error_code"`
}

// CodesResponse представляет ответ со списком QR-кодов
type CodesResponse struct {
	Total int            `json:"total"`
	Codes []CodeResponse `json:"codes"`
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

			r.Get("/users", ah.handleUsers)
			r.Post("/users/update", ah.handleUpdateUser)
			r.Post("/users/delete", ah.handleDeleteUser)
			r.Get("/codes", ah.handleCodes)
			r.Post("/codes/generate", ah.handleGenerateCode)
			r.Get("/admins", ah.handleGetAdmins)
			r.Post("/admins/add", ah.handleAddAdmin)
			r.Post("/admins/remove", ah.handleRemoveAdmin)
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
	case "codes":
		http.ServeFile(w, r, ah.staticDir+"/codes.html")
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
		userResponses = append(userResponses, UserResponse{
			ID:               user.Id,
			Telegramm:        user.Telegramm,
			FirstName:        user.FirstName,
			LastName:         user.LastName,
			MiddleName:       user.MiddleName,
			Points:           user.Points,
			Group:            user.Group,
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

	if points, ok := updateData["points"].(float64); ok {
		// Если баллы изменились, создаем транзакцию
		diff := int(points) - user.Points
		if diff != 0 {
			transaction := &models.Transaction{
				Id:     uuid.New(),
				UserId: user.Id,
				Diff:   diff,
				Time:   models.GetCurrentTime(),
			}

			if err := ah.store.AddTransaction(transaction); err != nil {
				ah.logger.Errorf("Ошибка создания транзакции: %v", err)
				http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
				return
			}
		}

		user.Points = int(points)
	}

	// Сохраняем обновленного пользователя
	if err := ah.store.UpdateUser(user); err != nil {
		ah.logger.Errorf("Ошибка обновления пользователя: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	userResp := UserResponse{
		ID:               user.Id,
		Telegramm:        user.Telegramm,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
		MiddleName:       user.MiddleName,
		Points:           user.Points,
		Group:            user.Group,
		RegistrationTime: user.RegistrationTime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userResp)
}

// handleCodes обрабатывает запрос на получение списка QR-кодов
func (ah *AdminHandler) handleCodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры фильтрации
	isActiveStr := r.URL.Query().Get("is_active")

	// Получаем все коды
	codes, err := ah.store.GetAllCodes()
	if err != nil {
		ah.logger.Errorf("Ошибка получения кодов: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Фильтруем коды
	var filteredCodes []*models.Code
	for _, code := range codes {
		// Фильтр по активности
		if isActiveStr != "" {
			isActive, err := strconv.ParseBool(isActiveStr)
			if err == nil && code.IsActive != isActive {
				continue
			}
		}

		filteredCodes = append(filteredCodes, code)
	}

	// Преобразуем коды в ответ
	var codeResponses []CodeResponse
	for _, code := range filteredCodes {
		codeResponses = append(codeResponses, CodeResponse{
			Code:         code.Code,
			Amount:       code.Amount,
			PerUser:      code.PerUser,
			Total:        code.Total,
			AppliedCount: code.AppliedCount,
			IsActive:     code.IsActive,
			Group:        code.Group,
			ErrorCode:    code.ErrorCode,
		})
	}

	// Отправляем ответ
	resp := CodesResponse{
		Total: len(codeResponses),
		Codes: codeResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleGenerateCode обрабатывает запрос на генерацию QR-кода
func (ah *AdminHandler) handleGenerateCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req CodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Создаем новый код
	code := &models.Code{
		Code:         uuid.New(),
		Amount:       req.Amount,
		PerUser:      req.PerUser,
		Total:        req.Total,
		AppliedCount: 0,
		IsActive:     true,
		Group:        req.Group,
		ErrorCode:    models.ErrorCodeNone,
	}

	// Сохраняем код
	if err := ah.store.AddCode(code); err != nil {
		ah.logger.Errorf("Ошибка создания кода: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	codeResp := CodeResponse{
		Code:         code.Code,
		Amount:       code.Amount,
		PerUser:      code.PerUser,
		Total:        code.Total,
		AppliedCount: code.AppliedCount,
		IsActive:     code.IsActive,
		Group:        code.Group,
		ErrorCode:    code.ErrorCode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(codeResp)
}

// loadAdmins загружает список администраторов из файла
func (ah *AdminHandler) loadAdmins() (AdminsList, error) {
	var admins AdminsList

	// Проверяем, существует ли файл
	if _, err := os.Stat(ah.adminsPath); os.IsNotExist(err) {
		// Если файл не существует, возвращаем пустой список
		return admins, nil
	}

	// Читаем файл
	data, err := os.ReadFile(ah.adminsPath)
	if err != nil {
		return admins, err
	}

	// Декодируем JSON
	var rawAdmins struct {
		Admins []struct {
			ID   int64  `json:"id"`
			Name string `json:"name,omitempty"`
		} `json:"admins"`
	}
	if err := json.Unmarshal(data, &rawAdmins); err != nil {
		// Пробуем старый формат
		var oldFormat struct {
			Admins []int64 `json:"admins"`
		}
		if err := json.Unmarshal(data, &oldFormat); err != nil {
			return admins, err
		}

		// Преобразуем из старого формата
		for _, id := range oldFormat.Admins {
			admins.Admins = append(admins.Admins, AdminInfo{
				ID: id,
			})
		}
		return admins, nil
	}

	// Преобразуем в AdminsList
	for _, admin := range rawAdmins.Admins {
		admins.Admins = append(admins.Admins, AdminInfo{
			ID:   admin.ID,
			Name: admin.Name,
		})
	}

	return admins, nil
}

// saveAdmins сохраняет список администраторов в файл
func (ah *AdminHandler) saveAdmins(admins []AdminInfo) error {
	// Преобразуем в формат для сохранения
	var rawAdmins struct {
		Admins []struct {
			ID   int64  `json:"id"`
			Name string `json:"name,omitempty"`
		} `json:"admins"`
	}

	rawAdmins.Admins = make([]struct {
		ID   int64  `json:"id"`
		Name string `json:"name,omitempty"`
	}, len(admins))

	for i, admin := range admins {
		rawAdmins.Admins[i].ID = admin.ID
		rawAdmins.Admins[i].Name = admin.Name
	}

	// Кодируем JSON
	data, err := json.MarshalIndent(rawAdmins, "", "    ")
	if err != nil {
		return err
	}

	// Создаем директорию, если она не существует
	dir := filepath.Dir(ah.adminsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Записываем файл
	if err := os.WriteFile(ah.adminsPath, data, 0644); err != nil {
		return err
	}

	return nil
}

// handleGetAdmins обрабатывает запрос на получение списка администраторов
func (ah *AdminHandler) handleGetAdmins(w http.ResponseWriter, r *http.Request) {
	ah.logger.Info("Запрос на получение списка администраторов")

	// Загружаем список администраторов
	admins, err := ah.loadAdmins()
	if err != nil {
		ah.logger.Errorf("Ошибка загрузки списка администраторов: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(admins)
}

// handleAddAdmin обрабатывает запрос на добавление администратора
func (ah *AdminHandler) handleAddAdmin(w http.ResponseWriter, r *http.Request) {
	ah.logger.Info("Запрос на добавление администратора")

	// Декодируем запрос
	var request struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ah.logger.Errorf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Загружаем список администраторов
	admins, err := ah.loadAdmins()
	if err != nil {
		ah.logger.Errorf("Ошибка загрузки списка администраторов: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Проверяем, есть ли уже такой администратор
	for _, admin := range admins.Admins {
		if admin.ID == request.ID {
			http.Error(w, "Администратор с таким ID уже существует", http.StatusBadRequest)
			return
		}
	}

	// Добавляем нового администратора
	admins.Admins = append(admins.Admins, AdminInfo{
		ID:   request.ID,
		Name: request.Name,
	})

	// Сохраняем список администраторов
	if err := ah.saveAdmins(admins.Admins); err != nil {
		ah.logger.Errorf("Ошибка сохранения списка администраторов: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
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

	// Загружаем список администраторов
	admins, err := ah.loadAdmins()
	if err != nil {
		ah.logger.Errorf("Ошибка загрузки списка администраторов: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Удаляем администратора
	var newAdmins []AdminInfo
	found := false
	for _, admin := range admins.Admins {
		if admin.ID != request.ID {
			newAdmins = append(newAdmins, admin)
		} else {
			found = true
		}
	}

	if !found {
		http.Error(w, "Администратор с таким ID не найден", http.StatusNotFound)
		return
	}

	// Сохраняем список администраторов
	if err := ah.saveAdmins(newAdmins); err != nil {
		ah.logger.Errorf("Ошибка сохранения списка администраторов: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
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
		case "codes":
			http.ServeFile(w, r, ah.staticDir+"/codes.html")
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
