package middleware

import (
	"net/http"
	"strings"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
)

// TokenAuthConfig представляет конфигурацию для middleware проверки токена
type TokenAuthConfig struct {
	APIToken string
	Logger   logger.Logger
	// Пути, которые не требуют аутентификации
	ExcludePaths []string
}

// TokenAuth представляет middleware для проверки токена
type TokenAuth struct {
	config TokenAuthConfig
}

// NewTokenAuth создает новый middleware для проверки токена
func NewTokenAuth(config TokenAuthConfig) *TokenAuth {
	return &TokenAuth{
		config: config,
	}
}

// Middleware обрабатывает HTTP-запрос и проверяет токен
func (ta *TokenAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, нужно ли проверять токен для этого пути
		if ta.isExcludedPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Получаем токен из заголовка Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			ta.config.Logger.Errorf("Отсутствует заголовок Authorization: %s", r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Проверяем формат токена (Bearer <token>)
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			ta.config.Logger.Errorf("Неверный формат заголовка Authorization: %s", authHeader)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Проверяем токен
		token := parts[1]
		if token != ta.config.APIToken {
			ta.config.Logger.Errorf("Неверный токен: %s", token)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Токен верный, продолжаем обработку запроса
		next.ServeHTTP(w, r)
	})
}

// isExcludedPath проверяет, исключен ли путь из проверки токена
func (ta *TokenAuth) isExcludedPath(path string) bool {
	// Исключаем пути для админки
	if strings.HasPrefix(path, "/admin") {
		return true
	}

	// Исключаем API админки
	if strings.HasPrefix(path, "/api/admin") {
		return true
	}

	// Исключаем статические файлы
	if strings.HasPrefix(path, "/static") {
		return true
	}

	// Проверяем пути из конфигурации
	for _, excludePath := range ta.config.ExcludePaths {
		if strings.HasPrefix(path, excludePath) {
			return true
		}
	}

	return false
}
