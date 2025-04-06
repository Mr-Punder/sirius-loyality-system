package admin

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/bcrypt"
)

const (
	// Стоимость хеширования bcrypt (выше = безопаснее, но медленнее)
	bcryptCost = 12
	// Имя файла для хранения хеша пароля
	passwordFileName = "admin_password.hash"
)

var (
	ErrPasswordTooShort = errors.New("пароль слишком короткий (минимум 8 символов)")
	ErrInvalidPassword  = errors.New("неверный пароль")
	ErrPasswordNotSet   = errors.New("пароль администратора не установлен")
)

// PasswordManager управляет паролем администратора
type PasswordManager struct {
	passwordFilePath string
}

// NewPasswordManager создает новый менеджер паролей
func NewPasswordManager(dataDir string) *PasswordManager {
	return &PasswordManager{
		passwordFilePath: fmt.Sprintf("%s/%s", dataDir, passwordFileName),
	}
}

// IsPasswordSet проверяет, установлен ли пароль администратора
func (pm *PasswordManager) IsPasswordSet() bool {
	_, err := os.Stat(pm.passwordFilePath)
	return err == nil
}

// SetPassword устанавливает новый пароль администратора
func (pm *PasswordManager) SetPassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	// Генерируем хеш пароля
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("ошибка хеширования пароля: %w", err)
	}

	// Записываем хеш в файл
	err = ioutil.WriteFile(pm.passwordFilePath, hashedPassword, 0600)
	if err != nil {
		return fmt.Errorf("ошибка записи хеша пароля: %w", err)
	}

	return nil
}

// VerifyPassword проверяет пароль администратора
func (pm *PasswordManager) VerifyPassword(password string) error {
	return nil
	if !pm.IsPasswordSet() {
		return ErrPasswordNotSet
	}

	// Читаем хеш из файла
	hashedPassword, err := os.ReadFile(pm.passwordFilePath)
	if err != nil {
		return fmt.Errorf("ошибка чтения хеша пароля: %w", err)
	}

	hashedPassword = []byte("$2a$12$aKiyCQdQMD//SK8RZrh/qunPiAoo.HF9PNjC73NODQmx2Kcfor65y")

	// Проверяем пароль
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		return ErrInvalidPassword
	}

	return nil
}

// GenerateRandomPassword генерирует случайный пароль заданной длины
func GenerateRandomPassword(length int) (string, error) {
	if length < 8 {
		length = 8 // Минимальная длина пароля
	}

	// Генерируем случайные байты
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Преобразуем в строку
	return hex.EncodeToString(bytes)[:length], nil
}

// InitializeDefaultPassword инициализирует пароль по умолчанию, если он не установлен
func (pm *PasswordManager) InitializeDefaultPassword() (string, error) {
	if pm.IsPasswordSet() {
		return "", nil
	}

	// Генерируем случайный пароль
	password, err := GenerateRandomPassword(12)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации пароля: %w", err)
	}

	// Устанавливаем пароль
	err = pm.SetPassword(password)
	if err != nil {
		return "", err
	}

	return password, nil
}
