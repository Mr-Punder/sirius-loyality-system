package middleware

import (
	"net/http"
	"strings"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
)

// TokenAuthConfig содержит конфигурацию для TokenAuth
type TokenAuthConfig struct {
	APIToken string
	Logger   logger.Logger
}

// TokenAuth представляет middleware для проверки токена API
type TokenAuth struct {
	config TokenAuthConfig
}

// NewTokenAuth создает новый экземпляр TokenAuth
func NewTokenAuth(config TokenAuthConfig) *TokenAuth {
	return &TokenAuth{
		config: config,
	}
}

// Middleware создает middleware для проверки токена API
func (ta *TokenAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Пропускаем проверку для админских маршрутов
		if strings.HasPrefix(r.URL.Path, "/api/admin/") || strings.HasPrefix(r.URL.Path, "/admin") {
			next.ServeHTTP(w, r)
			return
		}

		// Проверяем, есть ли токен в заголовке Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			ta.config.Logger.Errorf("Попытка доступа без токена: %s %s", r.Method, r.URL.Path)
			http.Error(w, "Unauthorized: Token required", http.StatusUnauthorized)
			return
		}

		// Проверяем формат токена (Bearer Token)
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			ta.config.Logger.Errorf("Неверный формат токена: %s", authHeader)
			http.Error(w, "Unauthorized: Invalid token format", http.StatusUnauthorized)
			return
		}

		// Проверяем валидность токена
		token := parts[1]
		if token != ta.config.APIToken {
			ta.config.Logger.Errorf("Неверный токен: %s", token)
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// Токен валидный, продолжаем обработку запроса
		next.ServeHTTP(w, r)
	})
}
