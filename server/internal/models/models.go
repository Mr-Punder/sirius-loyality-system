package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id               uuid.UUID `json:"id"`
	Telegramm        string    `json:"telegramm"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	MiddleName       string    `json:"middle_name"`
	Points           int       `json:"points"`
	Group            string    `json:"group"`
	RegistrationTime time.Time `json:"registration_time"`
	Deleted          bool      `json:"deleted"`
}

type Transaction struct {
	Id     uuid.UUID `json:"id"`
	UserId uuid.UUID `json:"user_id"`
	Code   uuid.UUID `json:"code"`
	Diff   int       `json:"diff"`
	Time   time.Time `json:"time"`
}

type Code struct {
	Code         uuid.UUID `json:"code"`
	Amount       int       `json:"amount"`
	PerUser      int       `json:"per_user"`
	Total        int       `json:"total"`
	AppliedCount int       `json:"applied_count"`
	IsActive     bool      `json:"is_active"`
	Group        string    `json:"group"`
	ErrorCode    int       `json:"error_code"`
}

// Константы для кодов ошибок
const (
	ErrorCodeNone               = 0
	ErrorCodeUserLimitExceeded  = 1 // Превышено количество использований кода пользователем
	ErrorCodeTotalLimitExceeded = 2 // Превышено общее количество использований кода
	ErrorCodeInvalidGroup       = 3 // Пользователь не принадлежит к группе, для которой предназначен код
	ErrorCodeCodeInactive       = 4 // Код не активен
)

type CodeUsage struct {
	Id     uuid.UUID `json:"id"`
	Code   uuid.UUID `json:"code"`
	UserId uuid.UUID `json:"user_id"`
	Count  int       `json:"count"`
}

// GetCurrentTime возвращает текущее время
func GetCurrentTime() time.Time {
	return time.Now()
}
