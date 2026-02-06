package telegrambot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsPieceCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Валидные коды
		{"ValidCode7Chars", "ABC1234", true},
		{"ValidCodeAllLetters", "ABCDEFG", true},
		{"ValidCodeAllDigits", "1234567", true},
		{"ValidCodeMixed", "A1B2C3D", true},
		{"ValidCodeLowercase", "abc1234", true}, // будет нормализован

		// Невалидные коды
		{"TooShort", "ABC123", false},
		{"TooLong", "ABC12345", false},
		{"Empty", "", false},
		{"WithSpaces", "ABC 123", false},
		{"WithSpecialChars", "ABC-123", false},
		{"WithUnderscore", "ABC_123", false},
		{"UUID", "550e8400-e29b-41d4-a716-446655440000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPieceCode(tt.input)
			assert.Equal(t, tt.expected, result, "isPieceCode(%q) should be %v", tt.input, tt.expected)
		})
	}
}

func TestNormalizeCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Lowercase", "abc1234", "ABC1234"},
		{"Uppercase", "ABC1234", "ABC1234"},
		{"Mixed", "AbC1234", "ABC1234"},
		{"WithSpaces", "  abc1234  ", "ABC1234"},
		{"WithNewline", "abc1234\n", "ABC1234"},
		{"WithTabs", "\tabc1234\t", "ABC1234"},
		{"WithAllWhitespace", " \t\n\rabc1234\r\n ", "ABC1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCode(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeGroupName(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedGroup string
		expectedValid bool
	}{
		// Валидные группы с кириллицей
		{"CyrillicН1", "Н1", "Н1", true},
		{"CyrillicН6", "Н6", "Н6", true},
		{"Lowercaseн1", "н1", "Н1", true},

		// Валидные группы с латиницей (должны конвертироваться в кириллицу)
		{"LatinH1", "H1", "Н1", true},
		{"LatinH6", "H6", "Н6", true},
		{"Lowercaseh1", "h1", "Н1", true},

		// Невалидные группы
		{"InvalidН0", "Н0", "", false},
		{"InvalidН7", "Н7", "", false},
		{"InvalidFormat", "Group1", "", false},
		{"Empty", "", "", false},
		{"JustNumber", "1", "", false},
		{"JustLetter", "Н", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := NormalizeGroupName(tt.input)
			assert.Equal(t, tt.expectedValid, valid, "NormalizeGroupName(%q) validity", tt.input)
			if valid {
				assert.Equal(t, tt.expectedGroup, result, "NormalizeGroupName(%q) result", tt.input)
			}
		})
	}
}

func TestGroupRegex(t *testing.T) {
	validGroups := []string{"Н1", "Н2", "Н3", "Н4", "Н5", "Н6", "H1", "H2", "H3", "H4", "H5", "H6", "н1", "h1"}
	invalidGroups := []string{"Н0", "Н7", "Н10", "H0", "H7", "Group1", "1", "Н", ""}

	for _, g := range validGroups {
		t.Run("Valid_"+g, func(t *testing.T) {
			assert.True(t, GroupRegex.MatchString(g), "GroupRegex should match %q", g)
		})
	}

	for _, g := range invalidGroups {
		t.Run("Invalid_"+g, func(t *testing.T) {
			assert.False(t, GroupRegex.MatchString(g), "GroupRegex should not match %q", g)
		})
	}
}

// TestUserBlockingLogic проверяет логику блокировки пользователей
func TestUserBlockingLogic(t *testing.T) {
	// Создаем мок-бота для тестирования
	ub := &UserBot{
		failedAttempts: make(map[int64]*FailedAttempts),
	}

	userID := int64(123456789)

	// Изначально пользователь не заблокирован
	assert.False(t, ub.isUserBlocked(userID), "Новый пользователь не должен быть заблокирован")
	assert.Equal(t, 0, ub.getFailedAttemptCount(userID), "Новый пользователь должен иметь 0 неудачных попыток")

	// Записываем 2 неудачные попытки - еще не заблокирован
	ub.recordFailedAttempt(userID)
	ub.recordFailedAttempt(userID)
	assert.False(t, ub.isUserBlocked(userID), "После 2 попыток пользователь не должен быть заблокирован")
	assert.Equal(t, 2, ub.getFailedAttemptCount(userID))

	// 3-я попытка - блокировка
	ub.recordFailedAttempt(userID)
	assert.True(t, ub.isUserBlocked(userID), "После 3 попыток пользователь должен быть заблокирован")

	// Проверяем оставшееся время блокировки
	remaining := ub.getBlockTimeRemaining(userID)
	assert.True(t, remaining > 0, "Оставшееся время блокировки должно быть больше 0")
	assert.True(t, remaining <= 5*time.Minute, "Оставшееся время блокировки не должно превышать 5 минут")

	// Очищаем попытки
	ub.clearFailedAttempts(userID)
	assert.False(t, ub.isUserBlocked(userID), "После очистки пользователь не должен быть заблокирован")
	assert.Equal(t, 0, ub.getFailedAttemptCount(userID))
}

// TestBlockExpiration проверяет, что блокировка истекает
func TestBlockExpiration(t *testing.T) {
	ub := &UserBot{
		failedAttempts: make(map[int64]*FailedAttempts),
	}

	userID := int64(987654321)

	// Блокируем пользователя
	for i := 0; i < 3; i++ {
		ub.recordFailedAttempt(userID)
	}
	assert.True(t, ub.isUserBlocked(userID))

	// Симулируем истечение времени блокировки
	if attempts, ok := ub.failedAttempts[userID]; ok {
		pastTime := time.Now().Add(-6 * time.Minute)
		attempts.BlockedAt = &pastTime
	}

	// Теперь пользователь не должен быть заблокирован
	assert.False(t, ub.isUserBlocked(userID), "Блокировка должна истечь через 5 минут")
}

// TestMultipleUsers проверяет независимость блокировок разных пользователей
func TestMultipleUsers(t *testing.T) {
	ub := &UserBot{
		failedAttempts: make(map[int64]*FailedAttempts),
	}

	user1 := int64(111)
	user2 := int64(222)

	// Блокируем первого пользователя
	for i := 0; i < 3; i++ {
		ub.recordFailedAttempt(user1)
	}

	// Второй пользователь делает 1 попытку
	ub.recordFailedAttempt(user2)

	assert.True(t, ub.isUserBlocked(user1), "Первый пользователь должен быть заблокирован")
	assert.False(t, ub.isUserBlocked(user2), "Второй пользователь не должен быть заблокирован")
	assert.Equal(t, 3, ub.getFailedAttemptCount(user1))
	assert.Equal(t, 1, ub.getFailedAttemptCount(user2))
}
