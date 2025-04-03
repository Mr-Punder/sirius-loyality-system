package telegrambot

import (
	"errors"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
)

// Ошибки
var (
	ErrInvalidBotType = errors.New("неверный тип бота")
)

// Config представляет конфигурацию Telegram-бота
type Config struct {
	Token       string // Токен бота
	AdminUserID int64  // ID администратора
	ServerURL   string // URL сервера для API-запросов
	APIToken    string // Токен для API-запросов
}

// Bot представляет интерфейс для Telegram-бота
type Bot interface {
	Start() error
	Stop() error
}

// BotType представляет тип бота
type BotType int

const (
	UserBotType BotType = iota
	AdminBotType
)

// NewBot создает новый экземпляр бота
func NewBot(botType BotType, config Config, storage storage.Storage, logger logger.Logger) (Bot, error) {
	switch botType {
	case UserBotType:
		return NewUserBot(config, storage, logger)
	case AdminBotType:
		return NewAdminBot(config, storage, logger)
	default:
		return nil, ErrInvalidBotType
	}
}
