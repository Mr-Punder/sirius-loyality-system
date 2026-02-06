package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockLogger реализует интерфейс logger для тестов
type mockLogger struct{}

func (m *mockLogger) Info(msg string)                   {}
func (m *mockLogger) Infof(format string, args ...any)  {}
func (m *mockLogger) Error(msg string)                  {}
func (m *mockLogger) Errorf(format string, args ...any) {}
func (m *mockLogger) Debug(msg string)                  {}
func (m *mockLogger) Debugf(format string, args ...any) {}
func (m *mockLogger) Warn(msg string)                   {}
func (m *mockLogger) Warnf(format string, args ...any)  {}
func (m *mockLogger) Fatal(msg string)                  {}
func (m *mockLogger) Fatalf(format string, args ...any) {}
func (m *mockLogger) Close() error                      { return nil }
func (m *mockLogger) With(key string, value any) any    { return m }

func TestTokenAuthMiddleware_BearerToken(t *testing.T) {
	validToken := "test-api-token-12345"

	tokenAuth := NewTokenAuth(TokenAuthConfig{
		APIToken: validToken,
		Logger:   &mockLogger{},
	})

	// Простой хендлер для тестов
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := tokenAuth.Middleware(nextHandler)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		description    string
	}{
		{
			name:           "ValidBearerToken",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			description:    "Запрос с правильным Bearer токеном должен пройти",
		},
		{
			name:           "InvalidBearerToken",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
			description:    "Запрос с неправильным токеном должен вернуть 401",
		},
		{
			name:           "NoAuthHeader",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			description:    "Запрос без заголовка Authorization должен вернуть 401",
		},
		{
			name:           "NoBearerPrefix",
			authHeader:     validToken,
			expectedStatus: http.StatusUnauthorized,
			description:    "Токен без префикса Bearer должен вернуть 401",
		},
		{
			name:           "WrongPrefix",
			authHeader:     "Basic " + validToken,
			expectedStatus: http.StatusUnauthorized,
			description:    "Токен с неправильным префиксом должен вернуть 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, tt.description)
		})
	}
}

func TestTokenAuthMiddleware_SkipPaths(t *testing.T) {
	validToken := "test-token"

	tokenAuth := NewTokenAuth(TokenAuthConfig{
		APIToken: validToken,
		Logger:   &mockLogger{},
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := tokenAuth.Middleware(nextHandler)

	// Эти пути должны пропускаться без проверки токена
	skipPaths := []string{
		"/api/admin/login",
		"/admin",
		"/admin/users.html",
		"/css/style.css",
		"/favicon.ico",
	}

	for _, path := range skipPaths {
		t.Run("Skip_"+path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			// Без токена
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code, "Путь %s должен пропускаться без токена", path)
		})
	}
}

func TestTokenAuthMiddleware_RequireAuthPaths(t *testing.T) {
	validToken := "test-token"

	tokenAuth := NewTokenAuth(TokenAuthConfig{
		APIToken: validToken,
		Logger:   &mockLogger{},
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := tokenAuth.Middleware(nextHandler)

	// Эти пути требуют токен
	requireAuthPaths := []string{
		"/users",
		"/puzzles",
		"/pieces",
		"/stats/lottery",
	}

	for _, path := range requireAuthPaths {
		t.Run("RequireAuth_"+path, func(t *testing.T) {
			// Без токена - должен вернуть 401
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusUnauthorized, rr.Code, "Путь %s без токена должен вернуть 401", path)

			// С токеном - должен пройти
			req = httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer "+validToken)
			rr = httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code, "Путь %s с токеном должен пройти", path)
		})
	}
}
