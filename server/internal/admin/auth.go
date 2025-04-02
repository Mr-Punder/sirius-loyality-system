package admin

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	// Срок действия токена (2 часа)
	tokenExpiration = 2 * time.Hour
)

var (
	ErrInvalidToken = errors.New("недействительный токен")
	ErrExpiredToken = errors.New("срок действия токена истек")
)

// JWTManager управляет JWT-токенами
type JWTManager struct {
	secretKey []byte
}

// Claims представляет данные, хранящиеся в JWT-токене
type Claims struct {
	jwt.RegisteredClaims
	IsAdmin bool `json:"is_admin"`
}

// NewJWTManager создает новый менеджер JWT
func NewJWTManager(secretKey string) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
	}
}

// GenerateToken генерирует новый JWT-токен для администратора
func (jm *JWTManager) GenerateToken() (string, error) {
	// Устанавливаем время истечения срока действия токена
	expirationTime := time.Now().Add(tokenExpiration)

	// Создаем claims
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		IsAdmin: true,
	}

	// Создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен
	tokenString, err := token.SignedString(jm.secretKey)
	if err != nil {
		return "", fmt.Errorf("ошибка подписи токена: %w", err)
	}

	return tokenString, nil
}

// ValidateToken проверяет JWT-токен
func (jm *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	// Парсим токен
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// Проверяем метод подписи
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
			}
			return jm.secretKey, nil
		},
	)

	if err != nil {
		// Проверяем, истек ли срок действия токена
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	// Получаем claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Проверяем, что токен принадлежит администратору
	if !claims.IsAdmin {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
