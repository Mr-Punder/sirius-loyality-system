package admin

import (
	"net/http"
	"strings"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
)

// AuthMiddleware представляет middleware для аутентификации
type AuthMiddleware struct {
	jwtManager *JWTManager
	logger     logger.Logger
}

// NewAuthMiddleware создает новый middleware для аутентификации
func NewAuthMiddleware(jwtManager *JWTManager, logger logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// Middleware возвращает функцию middleware для проверки JWT-токена
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем токен из заголовка Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Проверяем, есть ли токен в cookie
			cookie, err := r.Cookie("admin_token")
			if err != nil {
				http.Error(w, "Требуется аутентификация", http.StatusUnauthorized)
				return
			}
			authHeader = "Bearer " + cookie.Value
		}

		// Проверяем формат токена
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Неверный формат токена", http.StatusUnauthorized)
			return
		}

		// Проверяем токен
		tokenString := parts[1]
		claims, err := am.jwtManager.ValidateToken(tokenString)
		if err != nil {
			if err == ErrExpiredToken {
				http.Error(w, "Срок действия токена истек", http.StatusUnauthorized)
			} else {
				http.Error(w, "Недействительный токен", http.StatusUnauthorized)
			}
			return
		}

		// Проверяем, что пользователь является администратором
		if !claims.IsAdmin {
			http.Error(w, "Доступ запрещен", http.StatusForbidden)
			return
		}

		// Продолжаем обработку запроса
		next.ServeHTTP(w, r)
	})
}

// RequireAuth проверяет, аутентифицирован ли пользователь
func (am *AuthMiddleware) RequireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем токен из заголовка Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Проверяем, есть ли токен в cookie
			cookie, err := r.Cookie("admin_token")
			if err != nil {
				http.Error(w, "Требуется аутентификация", http.StatusUnauthorized)
				return
			}
			authHeader = "Bearer " + cookie.Value
		}

		// Проверяем формат токена
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Неверный формат токена", http.StatusUnauthorized)
			return
		}

		// Проверяем токен
		tokenString := parts[1]
		claims, err := am.jwtManager.ValidateToken(tokenString)
		if err != nil {
			if err == ErrExpiredToken {
				http.Error(w, "Срок действия токена истек", http.StatusUnauthorized)
			} else {
				http.Error(w, "Недействительный токен", http.StatusUnauthorized)
			}
			return
		}

		// Проверяем, что пользователь является администратором
		if !claims.IsAdmin {
			http.Error(w, "Доступ запрещен", http.StatusForbidden)
			return
		}

		// Продолжаем обработку запроса
		handler(w, r)
	}
}
