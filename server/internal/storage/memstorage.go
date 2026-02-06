package storage

import (
	"errors"
	"sync"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/google/uuid"
)

type Memstorage struct {
	users         sync.Map // uuid.UUID -> *models.User
	puzzles       sync.Map // int -> *models.Puzzle
	puzzlePieces  sync.Map // string -> *models.PuzzlePiece
	admins        sync.Map // int64 -> *models.Admin
	notifications sync.Map // uuid.UUID -> *models.Notification
}

func NewMemstorage() *Memstorage {
	m := &Memstorage{}
	// Инициализируем 30 пазлов
	for i := 1; i <= 30; i++ {
		m.puzzles.Store(i, &models.Puzzle{Id: i, IsCompleted: false})
	}
	return m
}

// ==================== МЕТОДЫ ДЛЯ ПОЛЬЗОВАТЕЛЕЙ ====================

func (m *Memstorage) GetUser(userId uuid.UUID) (*models.User, error) {
	userVal, ok := m.users.Load(userId)
	if !ok {
		return nil, errors.New("user not found")
	}
	user := userVal.(*models.User)
	if user.Deleted {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *Memstorage) GetUserByTelegramm(telegramm string) (*models.User, error) {
	var foundUser *models.User

	m.users.Range(func(key, value interface{}) bool {
		user := value.(*models.User)
		if user.Telegramm == telegramm && !user.Deleted {
			foundUser = user
			return false
		}
		return true
	})

	if foundUser == nil {
		return nil, errors.New("user not found")
	}

	return foundUser, nil
}

func (m *Memstorage) GetAllUsers() ([]*models.User, error) {
	var users []*models.User

	m.users.Range(func(key, value interface{}) bool {
		user := value.(*models.User)
		if !user.Deleted {
			users = append(users, user)
		}
		return true
	})

	return users, nil
}

func (m *Memstorage) AddUser(user *models.User) error {
	var exists bool

	m.users.Range(func(key, value interface{}) bool {
		existingUser := value.(*models.User)
		if existingUser.Telegramm == user.Telegramm && !existingUser.Deleted {
			exists = true
			return false
		}
		return true
	})

	if exists {
		return errors.New("user with this telegram ID already exists")
	}

	m.users.Store(user.Id, user)
	return nil
}

func (m *Memstorage) UpdateUser(user *models.User) error {
	_, ok := m.users.Load(user.Id)
	if !ok {
		return errors.New("user not found")
	}

	m.users.Store(user.Id, user)
	return nil
}

func (m *Memstorage) DeleteUser(userId uuid.UUID) error {
	_, ok := m.users.Load(userId)
	if !ok {
		return errors.New("user not found")
	}

	// Освобождаем детали пазлов пользователя
	m.puzzlePieces.Range(func(key, value interface{}) bool {
		piece := value.(*models.PuzzlePiece)
		if piece.OwnerId != nil && *piece.OwnerId == userId {
			piece.OwnerId = nil
			piece.RegisteredAt = nil
			m.puzzlePieces.Store(key, piece)
		}
		return true
	})

	// Удаляем пользователя
	m.users.Delete(userId)
	return nil
}

// ==================== МЕТОДЫ ДЛЯ ПАЗЛОВ ====================

func (m *Memstorage) GetPuzzle(puzzleId int) (*models.Puzzle, error) {
	puzzleVal, ok := m.puzzles.Load(puzzleId)
	if !ok {
		return nil, errors.New("puzzle not found")
	}
	return puzzleVal.(*models.Puzzle), nil
}

func (m *Memstorage) GetAllPuzzles() ([]*models.Puzzle, error) {
	var puzzles []*models.Puzzle

	m.puzzles.Range(func(key, value interface{}) bool {
		puzzle := value.(*models.Puzzle)
		puzzles = append(puzzles, puzzle)
		return true
	})

	return puzzles, nil
}

func (m *Memstorage) UpdatePuzzle(puzzle *models.Puzzle) error {
	_, ok := m.puzzles.Load(puzzle.Id)
	if !ok {
		return errors.New("puzzle not found")
	}

	m.puzzles.Store(puzzle.Id, puzzle)
	return nil
}

// ==================== МЕТОДЫ ДЛЯ ДЕТАЛЕЙ ПАЗЛОВ ====================

func (m *Memstorage) GetPuzzlePiece(code string) (*models.PuzzlePiece, error) {
	pieceVal, ok := m.puzzlePieces.Load(code)
	if !ok {
		return nil, errors.New("piece not found")
	}
	return pieceVal.(*models.PuzzlePiece), nil
}

func (m *Memstorage) GetPuzzlePiecesByPuzzle(puzzleId int) ([]*models.PuzzlePiece, error) {
	var pieces []*models.PuzzlePiece

	m.puzzlePieces.Range(func(key, value interface{}) bool {
		piece := value.(*models.PuzzlePiece)
		if piece.PuzzleId == puzzleId {
			pieces = append(pieces, piece)
		}
		return true
	})

	return pieces, nil
}

func (m *Memstorage) GetPuzzlePiecesByOwner(ownerId uuid.UUID) ([]*models.PuzzlePiece, error) {
	var pieces []*models.PuzzlePiece

	m.puzzlePieces.Range(func(key, value interface{}) bool {
		piece := value.(*models.PuzzlePiece)
		if piece.OwnerId != nil && *piece.OwnerId == ownerId {
			pieces = append(pieces, piece)
		}
		return true
	})

	return pieces, nil
}

func (m *Memstorage) GetAllPuzzlePieces() ([]*models.PuzzlePiece, error) {
	var pieces []*models.PuzzlePiece

	m.puzzlePieces.Range(func(key, value interface{}) bool {
		piece := value.(*models.PuzzlePiece)
		pieces = append(pieces, piece)
		return true
	})

	return pieces, nil
}

func (m *Memstorage) AddPuzzlePiece(piece *models.PuzzlePiece) error {
	_, ok := m.puzzlePieces.Load(piece.Code)
	if ok {
		return errors.New("piece already exists")
	}

	m.puzzlePieces.Store(piece.Code, piece)
	return nil
}

func (m *Memstorage) AddPuzzlePieces(pieces []*models.PuzzlePiece) error {
	for _, piece := range pieces {
		if err := m.AddPuzzlePiece(piece); err != nil {
			return err
		}
	}
	return nil
}

func (m *Memstorage) RegisterPuzzlePiece(code string, ownerId uuid.UUID) (*models.PuzzlePiece, bool, error) {
	pieceVal, ok := m.puzzlePieces.Load(code)
	if !ok {
		return nil, false, errors.New("piece not found")
	}

	piece := pieceVal.(*models.PuzzlePiece)
	if piece.OwnerId != nil {
		return nil, false, errors.New("piece already taken")
	}

	// Привязываем деталь
	now := time.Now()
	piece.OwnerId = &ownerId
	piece.RegisteredAt = &now
	m.puzzlePieces.Store(code, piece)

	// Проверяем, все ли детали пазла розданы
	pieces, _ := m.GetPuzzlePiecesByPuzzle(piece.PuzzleId)
	ownedCount := 0
	for _, p := range pieces {
		if p.OwnerId != nil {
			ownedCount++
		}
	}

	allPiecesDistributed := ownedCount == 6

	return piece, allPiecesDistributed, nil
}

// CompletePuzzle отмечает пазл как собранный и возвращает владельцев деталей для уведомления
func (m *Memstorage) CompletePuzzle(puzzleId int) ([]*models.User, error) {
	puzzleVal, ok := m.puzzles.Load(puzzleId)
	if !ok {
		return nil, errors.New("puzzle not found")
	}

	puzzle := puzzleVal.(*models.Puzzle)
	if puzzle.IsCompleted {
		return nil, errors.New("puzzle already completed")
	}

	// Отмечаем пазл как завершенный
	now := time.Now()
	puzzle.IsCompleted = true
	puzzle.CompletedAt = &now
	m.puzzles.Store(puzzleId, puzzle)

	// Собираем уникальных владельцев деталей
	ownerIds := make(map[uuid.UUID]bool)
	m.puzzlePieces.Range(func(key, value interface{}) bool {
		piece := value.(*models.PuzzlePiece)
		if piece.PuzzleId == puzzleId && piece.OwnerId != nil {
			ownerIds[*piece.OwnerId] = true
		}
		return true
	})

	var users []*models.User
	for ownerId := range ownerIds {
		userVal, ok := m.users.Load(ownerId)
		if ok {
			users = append(users, userVal.(*models.User))
		}
	}

	return users, nil
}

// ==================== МЕТОДЫ ДЛЯ СТАТИСТИКИ ====================

func (m *Memstorage) GetUserPieceCount(userId uuid.UUID) (int, error) {
	pieces, _ := m.GetPuzzlePiecesByOwner(userId)
	return len(pieces), nil
}

func (m *Memstorage) GetUserCompletedPuzzlePieceCount(userId uuid.UUID) (int, error) {
	pieces, _ := m.GetPuzzlePiecesByOwner(userId)
	count := 0
	for _, piece := range pieces {
		puzzleVal, ok := m.puzzles.Load(piece.PuzzleId)
		if ok {
			puzzle := puzzleVal.(*models.Puzzle)
			if puzzle.IsCompleted {
				count++
			}
		}
	}
	return count, nil
}

// ==================== МЕТОДЫ ДЛЯ АДМИНИСТРАТОРОВ ====================

func (m *Memstorage) GetAdmin(adminId int64) (*models.Admin, error) {
	adminVal, ok := m.admins.Load(adminId)
	if !ok {
		return nil, errors.New("admin not found")
	}
	return adminVal.(*models.Admin), nil
}

func (m *Memstorage) GetAllAdmins() ([]*models.Admin, error) {
	var admins []*models.Admin

	m.admins.Range(func(key, value interface{}) bool {
		admin := value.(*models.Admin)
		if admin.IsActive {
			admins = append(admins, admin)
		}
		return true
	})

	return admins, nil
}

func (m *Memstorage) AddAdmin(admin *models.Admin) error {
	_, ok := m.admins.Load(admin.ID)
	if ok {
		return errors.New("admin with this ID already exists")
	}

	m.admins.Store(admin.ID, admin)
	return nil
}

func (m *Memstorage) UpdateAdmin(admin *models.Admin) error {
	_, ok := m.admins.Load(admin.ID)
	if !ok {
		return errors.New("admin not found")
	}

	m.admins.Store(admin.ID, admin)
	return nil
}

func (m *Memstorage) DeleteAdmin(adminId int64) error {
	_, ok := m.admins.Load(adminId)
	if !ok {
		return errors.New("admin not found")
	}

	m.admins.Delete(adminId)
	return nil
}

// ==================== МЕТОДЫ ДЛЯ УВЕДОМЛЕНИЙ ====================

func (m *Memstorage) AddNotification(notification *models.Notification) error {
	m.notifications.Store(notification.Id, notification)
	return nil
}

func (m *Memstorage) GetPendingNotifications() ([]*models.Notification, error) {
	var notifications []*models.Notification

	m.notifications.Range(func(key, value interface{}) bool {
		notification := value.(*models.Notification)
		if notification.Status == models.NotificationPending {
			notifications = append(notifications, notification)
		}
		return true
	})

	return notifications, nil
}

func (m *Memstorage) UpdateNotification(notification *models.Notification) error {
	_, ok := m.notifications.Load(notification.Id)
	if !ok {
		return errors.New("notification not found")
	}

	m.notifications.Store(notification.Id, notification)
	return nil
}

func (m *Memstorage) GetNotification(id uuid.UUID) (*models.Notification, error) {
	notificationVal, ok := m.notifications.Load(id)
	if !ok {
		return nil, errors.New("notification not found")
	}
	return notificationVal.(*models.Notification), nil
}

func (m *Memstorage) GetAllNotifications() ([]*models.Notification, error) {
	var notifications []*models.Notification

	m.notifications.Range(func(key, value interface{}) bool {
		notification := value.(*models.Notification)
		notifications = append(notifications, notification)
		return true
	})

	return notifications, nil
}
