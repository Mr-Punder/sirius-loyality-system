package storage

import (
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/google/uuid"
)

type Storage interface {
	// Методы для получения данных
	GetUser(userId uuid.UUID) (*models.User, error)
	GetUserPoints(userid uuid.UUID) (int, error)
	GetUserByTelegramm(telega string) (*models.User, error)
	GetAllUsers() ([]*models.User, error)
	GetTransaction(transactionId uuid.UUID) (*models.Transaction, error)
	GetUserTransactions(userId uuid.UUID) ([]*models.Transaction, error)
	GetAllTransactions() ([]*models.Transaction, error)
	GetCodeInfo(code uuid.UUID) (*models.Code, error)
	GetAllCodes() ([]*models.Code, error)
	GetCodeUsage(code uuid.UUID) ([]*models.CodeUsage, error)
	GetAllCodeUsages() ([]*models.CodeUsage, error)
	GetCodeUsageByUser(code uuid.UUID, userId uuid.UUID) (*models.CodeUsage, error)

	// Методы для добавления данных
	AddUser(user *models.User) error
	AddTransaction(transaction *models.Transaction) error
	AddCode(code *models.Code) error
	AddCodeUsage(usage *models.CodeUsage) error

	// Методы для обновления данных
	UpdateUser(user *models.User) error
	UpdateCode(code *models.Code) error
	UpdateCodeUsage(usage *models.CodeUsage) error

	// Методы для удаления данных
	DeleteUser(userId uuid.UUID) error
	DeleteCode(code uuid.UUID) error

	// Методы для работы с администраторами
	GetAdmin(adminId int64) (*models.Admin, error)
	GetAllAdmins() ([]*models.Admin, error)
	AddAdmin(admin *models.Admin) error
	UpdateAdmin(admin *models.Admin) error
	DeleteAdmin(adminId int64) error
}
