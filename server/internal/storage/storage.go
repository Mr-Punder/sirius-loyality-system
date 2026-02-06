package storage

import (
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/google/uuid"
)

type Storage interface {
	// Методы для пользователей
	GetUser(userId uuid.UUID) (*models.User, error)
	GetUserByTelegramm(telega string) (*models.User, error)
	GetAllUsers() ([]*models.User, error)
	AddUser(user *models.User) error
	UpdateUser(user *models.User) error
	DeleteUser(userId uuid.UUID) error

	// Методы для пазлов
	GetPuzzle(puzzleId int) (*models.Puzzle, error)
	GetAllPuzzles() ([]*models.Puzzle, error)
	UpdatePuzzle(puzzle *models.Puzzle) error

	// Методы для деталей пазлов
	GetPuzzlePiece(code string) (*models.PuzzlePiece, error)
	GetPuzzlePiecesByPuzzle(puzzleId int) ([]*models.PuzzlePiece, error)
	GetPuzzlePiecesByOwner(ownerId uuid.UUID) ([]*models.PuzzlePiece, error)
	GetAllPuzzlePieces() ([]*models.PuzzlePiece, error)
	AddPuzzlePiece(piece *models.PuzzlePiece) error
	AddPuzzlePieces(pieces []*models.PuzzlePiece) error                                    // Массовое добавление
	RegisterPuzzlePiece(code string, ownerId uuid.UUID) (*models.PuzzlePiece, bool, error) // Возвращает деталь, флаг "все детали розданы", ошибку
	CompletePuzzle(puzzleId int) ([]*models.User, error)                                   // Засчитать пазл, вернуть владельцев деталей для уведомления

	// Методы для статистики
	GetUserPieceCount(userId uuid.UUID) (int, error)
	GetUserCompletedPuzzlePieceCount(userId uuid.UUID) (int, error)

	// Методы для работы с администраторами
	GetAdmin(adminId int64) (*models.Admin, error)
	GetAllAdmins() ([]*models.Admin, error)
	AddAdmin(admin *models.Admin) error
	UpdateAdmin(admin *models.Admin) error
	DeleteAdmin(adminId int64) error

	// Методы для уведомлений (очередь рассылки)
	AddNotification(notification *models.Notification) error
	GetPendingNotifications() ([]*models.Notification, error)
	UpdateNotification(notification *models.Notification) error
	GetNotification(id uuid.UUID) (*models.Notification, error)
	GetAllNotifications() ([]*models.Notification, error)
}
