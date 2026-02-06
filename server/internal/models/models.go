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
	Group            string    `json:"group"`
	RegistrationTime time.Time `json:"registration_time"`
	Deleted          bool      `json:"deleted"`
}

// Puzzle представляет пазл (всего 30 пазлов по 6 деталей)
type Puzzle struct {
	Id          int        `json:"id"`           // 1-30
	Name        string     `json:"name"`         // Название пазла
	IsCompleted bool       `json:"is_completed"` // Засчитан ли пазл админом
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// PuzzlePiece представляет деталь пазла
type PuzzlePiece struct {
	Code         string     `json:"code"`                    // Уникальный 7-символьный код, например "PY1GG7H"
	PuzzleId     int        `json:"puzzle_id"`               // К какому пазлу принадлежит (1-30)
	PieceNumber  int        `json:"piece_number"`            // Номер детали в пазле (1-6)
	OwnerId      *uuid.UUID `json:"owner_id,omitempty"`      // Кто зарегистрировал деталь
	RegisteredAt *time.Time `json:"registered_at,omitempty"` // Когда была зарегистрирована
}

// Константы для кодов ошибок деталей пазлов
const (
	PieceErrorNone         = 0 // Нет ошибки
	PieceErrorNotFound     = 1 // Код детали не найден
	PieceErrorAlreadyTaken = 2 // Деталь уже принадлежит другому пользователю
)

// Admin представляет информацию об администраторе
type Admin struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
	IsActive bool   `json:"is_active"`
}

// NotificationStatus статус уведомления
type NotificationStatus string

const (
	NotificationPending NotificationStatus = "pending"
	NotificationSent    NotificationStatus = "sent"
	NotificationFailed  NotificationStatus = "failed"
)

// Notification представляет уведомление для рассылки
type Notification struct {
	Id          uuid.UUID          `json:"id"`
	Message     string             `json:"message"`               // Текст сообщения
	Group       string             `json:"group,omitempty"`       // Группа (пустая = смотрим UserIds)
	UserIds     []uuid.UUID        `json:"user_ids,omitempty"`    // Конкретные пользователи (если Group пустая)
	Attachments []string           `json:"attachments,omitempty"` // Имена файлов вложений (лежат в attachments/{id}/)
	Status      NotificationStatus `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	SentAt      *time.Time         `json:"sent_at,omitempty"`
	SentCount   int                `json:"sent_count"`  // Сколько сообщений отправлено
	ErrorCount  int                `json:"error_count"` // Сколько ошибок
}

// Attachment представляет файл в библиотеке вложений
type Attachment struct {
	Id        uuid.UUID `json:"id"`
	Filename  string    `json:"filename"`   // Отображаемое имя файла
	StorePath string    `json:"store_path"` // Путь к файлу на диске (data/library/uuid.ext)
	MimeType  string    `json:"mime_type"`  // MIME-тип (image/jpeg, application/pdf, etc.)
	Size      int64     `json:"size"`       // Размер в байтах
	CreatedAt time.Time `json:"created_at"`
}

// GetCurrentTime возвращает текущее время
func GetCurrentTime() time.Time {
	return time.Now()
}
