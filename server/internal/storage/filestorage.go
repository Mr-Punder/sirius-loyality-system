package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/google/uuid"
)

// Константы для имен файлов
const (
	UsersFileName         = "users.json"
	PuzzlesFileName       = "puzzles.json"
	PuzzlePiecesFileName  = "puzzle_pieces.json"
	AdminsFileName        = "admins.json"
	NotificationsFileName = "notifications.json"
	AttachmentsFileName   = "attachments.json"
)

// Filestorage реализует интерфейс Storage с хранением данных в файлах
type Filestorage struct {
	users         sync.Map // uuid.UUID -> *models.User
	puzzles       sync.Map // int -> *models.Puzzle
	puzzlePieces  sync.Map // string -> *models.PuzzlePiece
	admins        sync.Map // int64 -> *models.Admin
	notifications sync.Map // uuid.UUID -> *models.Notification
	attachments   sync.Map // uuid.UUID -> *models.Attachment
	dataDir       string
	mu            sync.Mutex
}

// NewFilestorage создает новое файловое хранилище
func NewFilestorage(dataDir string) (*Filestorage, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	fs := &Filestorage{
		dataDir: dataDir,
	}

	// Инициализируем 30 пазлов
	for i := 1; i <= 30; i++ {
		fs.puzzles.Store(i, &models.Puzzle{Id: i, IsCompleted: false})
	}

	if err := fs.loadData(); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	return fs, nil
}

func (fs *Filestorage) loadData() error {
	if err := fs.loadUsers(); err != nil {
		return err
	}
	if err := fs.loadPuzzles(); err != nil {
		return err
	}
	if err := fs.loadPuzzlePieces(); err != nil {
		return err
	}
	if err := fs.loadAdmins(); err != nil {
		return err
	}
	if err := fs.loadNotifications(); err != nil {
		return err
	}
	if err := fs.loadAttachments(); err != nil {
		return err
	}
	return nil
}

func (fs *Filestorage) loadUsers() error {
	filePath := filepath.Join(fs.dataDir, UsersFileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fs.saveUsers()
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read users file: %w", err)
	}

	var users []*models.User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("failed to unmarshal users: %w", err)
	}

	for _, user := range users {
		fs.users.Store(user.Id, user)
	}

	return nil
}

func (fs *Filestorage) saveUsers() error {
	var users []*models.User
	fs.users.Range(func(key, value interface{}) bool {
		users = append(users, value.(*models.User))
		return true
	})

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	filePath := filepath.Join(fs.dataDir, UsersFileName)
	return os.WriteFile(filePath, data, 0644)
}

func (fs *Filestorage) loadPuzzles() error {
	filePath := filepath.Join(fs.dataDir, PuzzlesFileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fs.savePuzzles()
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read puzzles file: %w", err)
	}

	var puzzles []*models.Puzzle
	if err := json.Unmarshal(data, &puzzles); err != nil {
		return fmt.Errorf("failed to unmarshal puzzles: %w", err)
	}

	for _, puzzle := range puzzles {
		fs.puzzles.Store(puzzle.Id, puzzle)
	}

	return nil
}

func (fs *Filestorage) savePuzzles() error {
	var puzzles []*models.Puzzle
	fs.puzzles.Range(func(key, value interface{}) bool {
		puzzles = append(puzzles, value.(*models.Puzzle))
		return true
	})

	data, err := json.MarshalIndent(puzzles, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal puzzles: %w", err)
	}

	filePath := filepath.Join(fs.dataDir, PuzzlesFileName)
	return os.WriteFile(filePath, data, 0644)
}

func (fs *Filestorage) loadPuzzlePieces() error {
	filePath := filepath.Join(fs.dataDir, PuzzlePiecesFileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fs.savePuzzlePieces()
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read puzzle pieces file: %w", err)
	}

	var pieces []*models.PuzzlePiece
	if err := json.Unmarshal(data, &pieces); err != nil {
		return fmt.Errorf("failed to unmarshal puzzle pieces: %w", err)
	}

	for _, piece := range pieces {
		fs.puzzlePieces.Store(piece.Code, piece)
	}

	return nil
}

func (fs *Filestorage) savePuzzlePieces() error {
	var pieces []*models.PuzzlePiece
	fs.puzzlePieces.Range(func(key, value interface{}) bool {
		pieces = append(pieces, value.(*models.PuzzlePiece))
		return true
	})

	data, err := json.MarshalIndent(pieces, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal puzzle pieces: %w", err)
	}

	filePath := filepath.Join(fs.dataDir, PuzzlePiecesFileName)
	return os.WriteFile(filePath, data, 0644)
}

func (fs *Filestorage) loadAdmins() error {
	filePath := filepath.Join(fs.dataDir, AdminsFileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fs.saveAdmins()
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read admins file: %w", err)
	}

	var admins []*models.Admin
	if err := json.Unmarshal(data, &admins); err != nil {
		return fmt.Errorf("failed to unmarshal admins: %w", err)
	}

	for _, admin := range admins {
		fs.admins.Store(admin.ID, admin)
	}

	return nil
}

func (fs *Filestorage) saveAdmins() error {
	var admins []*models.Admin
	fs.admins.Range(func(key, value interface{}) bool {
		admins = append(admins, value.(*models.Admin))
		return true
	})

	data, err := json.MarshalIndent(admins, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal admins: %w", err)
	}

	filePath := filepath.Join(fs.dataDir, AdminsFileName)
	return os.WriteFile(filePath, data, 0644)
}

// ==================== МЕТОДЫ ДЛЯ ПОЛЬЗОВАТЕЛЕЙ ====================

func (fs *Filestorage) GetUser(userId uuid.UUID) (*models.User, error) {
	userVal, ok := fs.users.Load(userId)
	if !ok {
		return nil, errors.New("user not found")
	}
	user := userVal.(*models.User)
	if user.Deleted {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (fs *Filestorage) GetUserByTelegramm(telegramm string) (*models.User, error) {
	var foundUser *models.User
	fs.users.Range(func(key, value interface{}) bool {
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

func (fs *Filestorage) GetAllUsers() ([]*models.User, error) {
	var users []*models.User
	fs.users.Range(func(key, value interface{}) bool {
		user := value.(*models.User)
		if !user.Deleted {
			users = append(users, user)
		}
		return true
	})
	return users, nil
}

func (fs *Filestorage) AddUser(user *models.User) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var exists bool
	fs.users.Range(func(key, value interface{}) bool {
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

	fs.users.Store(user.Id, user)
	return fs.saveUsers()
}

func (fs *Filestorage) UpdateUser(user *models.User) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.users.Load(user.Id)
	if !ok {
		return errors.New("user not found")
	}

	fs.users.Store(user.Id, user)
	return fs.saveUsers()
}

func (fs *Filestorage) DeleteUser(userId uuid.UUID) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.users.Load(userId)
	if !ok {
		return errors.New("user not found")
	}

	// Освобождаем детали пазлов пользователя
	fs.puzzlePieces.Range(func(key, value interface{}) bool {
		piece := value.(*models.PuzzlePiece)
		if piece.OwnerId != nil && *piece.OwnerId == userId {
			piece.OwnerId = nil
			piece.RegisteredAt = nil
			fs.puzzlePieces.Store(key, piece)
		}
		return true
	})

	// Удаляем пользователя
	fs.users.Delete(userId)
	return fs.saveUsers()
}

// ==================== МЕТОДЫ ДЛЯ ПАЗЛОВ ====================

func (fs *Filestorage) GetPuzzle(puzzleId int) (*models.Puzzle, error) {
	puzzleVal, ok := fs.puzzles.Load(puzzleId)
	if !ok {
		return nil, errors.New("puzzle not found")
	}
	return puzzleVal.(*models.Puzzle), nil
}

func (fs *Filestorage) GetAllPuzzles() ([]*models.Puzzle, error) {
	var puzzles []*models.Puzzle
	fs.puzzles.Range(func(key, value interface{}) bool {
		puzzles = append(puzzles, value.(*models.Puzzle))
		return true
	})
	return puzzles, nil
}

func (fs *Filestorage) UpdatePuzzle(puzzle *models.Puzzle) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.puzzles.Load(puzzle.Id)
	if !ok {
		return errors.New("puzzle not found")
	}

	fs.puzzles.Store(puzzle.Id, puzzle)
	return fs.savePuzzles()
}

// ==================== МЕТОДЫ ДЛЯ ДЕТАЛЕЙ ПАЗЛОВ ====================

func (fs *Filestorage) GetPuzzlePiece(code string) (*models.PuzzlePiece, error) {
	pieceVal, ok := fs.puzzlePieces.Load(code)
	if !ok {
		return nil, errors.New("piece not found")
	}
	return pieceVal.(*models.PuzzlePiece), nil
}

func (fs *Filestorage) GetPuzzlePiecesByPuzzle(puzzleId int) ([]*models.PuzzlePiece, error) {
	var pieces []*models.PuzzlePiece
	fs.puzzlePieces.Range(func(key, value interface{}) bool {
		piece := value.(*models.PuzzlePiece)
		if piece.PuzzleId == puzzleId {
			pieces = append(pieces, piece)
		}
		return true
	})
	return pieces, nil
}

func (fs *Filestorage) GetPuzzlePiecesByOwner(ownerId uuid.UUID) ([]*models.PuzzlePiece, error) {
	var pieces []*models.PuzzlePiece
	fs.puzzlePieces.Range(func(key, value interface{}) bool {
		piece := value.(*models.PuzzlePiece)
		if piece.OwnerId != nil && *piece.OwnerId == ownerId {
			pieces = append(pieces, piece)
		}
		return true
	})
	return pieces, nil
}

func (fs *Filestorage) GetAllPuzzlePieces() ([]*models.PuzzlePiece, error) {
	var pieces []*models.PuzzlePiece
	fs.puzzlePieces.Range(func(key, value interface{}) bool {
		pieces = append(pieces, value.(*models.PuzzlePiece))
		return true
	})
	return pieces, nil
}

func (fs *Filestorage) AddPuzzlePiece(piece *models.PuzzlePiece) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.puzzlePieces.Load(piece.Code)
	if ok {
		return errors.New("piece already exists")
	}

	fs.puzzlePieces.Store(piece.Code, piece)
	return fs.savePuzzlePieces()
}

func (fs *Filestorage) AddPuzzlePieces(pieces []*models.PuzzlePiece) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, piece := range pieces {
		fs.puzzlePieces.Store(piece.Code, piece)
	}
	return fs.savePuzzlePieces()
}

func (fs *Filestorage) RegisterPuzzlePiece(code string, ownerId uuid.UUID) (*models.PuzzlePiece, bool, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	pieceVal, ok := fs.puzzlePieces.Load(code)
	if !ok {
		return nil, false, errors.New("piece not found")
	}

	piece := pieceVal.(*models.PuzzlePiece)
	if piece.OwnerId != nil {
		return nil, false, errors.New("piece already taken")
	}

	now := time.Now()
	piece.OwnerId = &ownerId
	piece.RegisteredAt = &now
	fs.puzzlePieces.Store(code, piece)

	// Проверяем, все ли детали пазла розданы
	pieces, _ := fs.GetPuzzlePiecesByPuzzle(piece.PuzzleId)
	ownedCount := 0
	for _, p := range pieces {
		if p.OwnerId != nil {
			ownedCount++
		}
	}

	allPiecesDistributed := ownedCount == 6

	fs.savePuzzlePieces()
	return piece, allPiecesDistributed, nil
}

// CompletePuzzle отмечает пазл как собранный и возвращает владельцев деталей для уведомления
func (fs *Filestorage) CompletePuzzle(puzzleId int) ([]*models.User, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	puzzleVal, ok := fs.puzzles.Load(puzzleId)
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
	fs.puzzles.Store(puzzleId, puzzle)
	fs.savePuzzles()

	// Собираем уникальных владельцев деталей
	ownerIds := make(map[uuid.UUID]bool)
	fs.puzzlePieces.Range(func(key, value interface{}) bool {
		piece := value.(*models.PuzzlePiece)
		if piece.PuzzleId == puzzleId && piece.OwnerId != nil {
			ownerIds[*piece.OwnerId] = true
		}
		return true
	})

	var users []*models.User
	for ownerId := range ownerIds {
		userVal, ok := fs.users.Load(ownerId)
		if ok {
			users = append(users, userVal.(*models.User))
		}
	}

	return users, nil
}

// ==================== МЕТОДЫ ДЛЯ СТАТИСТИКИ ====================

func (fs *Filestorage) GetUserPieceCount(userId uuid.UUID) (int, error) {
	pieces, _ := fs.GetPuzzlePiecesByOwner(userId)
	return len(pieces), nil
}

func (fs *Filestorage) GetUserCompletedPuzzlePieceCount(userId uuid.UUID) (int, error) {
	pieces, _ := fs.GetPuzzlePiecesByOwner(userId)
	count := 0
	for _, piece := range pieces {
		puzzleVal, ok := fs.puzzles.Load(piece.PuzzleId)
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

func (fs *Filestorage) GetAdmin(adminId int64) (*models.Admin, error) {
	adminVal, ok := fs.admins.Load(adminId)
	if !ok {
		return nil, errors.New("admin not found")
	}
	return adminVal.(*models.Admin), nil
}

func (fs *Filestorage) GetAllAdmins() ([]*models.Admin, error) {
	var admins []*models.Admin
	fs.admins.Range(func(key, value interface{}) bool {
		admin := value.(*models.Admin)
		if admin.IsActive {
			admins = append(admins, admin)
		}
		return true
	})
	return admins, nil
}

func (fs *Filestorage) AddAdmin(admin *models.Admin) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.admins.Load(admin.ID)
	if ok {
		return errors.New("admin with this ID already exists")
	}

	fs.admins.Store(admin.ID, admin)
	return fs.saveAdmins()
}

func (fs *Filestorage) UpdateAdmin(admin *models.Admin) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.admins.Load(admin.ID)
	if !ok {
		return errors.New("admin not found")
	}

	fs.admins.Store(admin.ID, admin)
	return fs.saveAdmins()
}

func (fs *Filestorage) DeleteAdmin(adminId int64) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.admins.Load(adminId)
	if !ok {
		return errors.New("admin not found")
	}

	fs.admins.Delete(adminId)
	return fs.saveAdmins()
}

// ==================== МЕТОДЫ ДЛЯ УВЕДОМЛЕНИЙ ====================

func (fs *Filestorage) AddNotification(notification *models.Notification) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.notifications.Store(notification.Id, notification)
	return fs.saveNotifications()
}

func (fs *Filestorage) GetPendingNotifications() ([]*models.Notification, error) {
	var notifications []*models.Notification

	fs.notifications.Range(func(key, value interface{}) bool {
		notification := value.(*models.Notification)
		if notification.Status == models.NotificationPending {
			notifications = append(notifications, notification)
		}
		return true
	})

	return notifications, nil
}

func (fs *Filestorage) UpdateNotification(notification *models.Notification) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.notifications.Load(notification.Id)
	if !ok {
		return errors.New("notification not found")
	}

	fs.notifications.Store(notification.Id, notification)
	return fs.saveNotifications()
}

func (fs *Filestorage) GetNotification(id uuid.UUID) (*models.Notification, error) {
	notificationVal, ok := fs.notifications.Load(id)
	if !ok {
		return nil, errors.New("notification not found")
	}
	return notificationVal.(*models.Notification), nil
}

func (fs *Filestorage) GetAllNotifications() ([]*models.Notification, error) {
	var notifications []*models.Notification

	fs.notifications.Range(func(key, value interface{}) bool {
		notification := value.(*models.Notification)
		notifications = append(notifications, notification)
		return true
	})

	return notifications, nil
}

func (fs *Filestorage) saveNotifications() error {
	var notifications []*models.Notification

	fs.notifications.Range(func(key, value interface{}) bool {
		notification := value.(*models.Notification)
		notifications = append(notifications, notification)
		return true
	})

	data, err := json.MarshalIndent(notifications, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal notifications: %w", err)
	}

	filePath := filepath.Join(fs.dataDir, NotificationsFileName)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write notifications file: %w", err)
	}

	return nil
}

func (fs *Filestorage) loadNotifications() error {
	filePath := filepath.Join(fs.dataDir, NotificationsFileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read notifications file: %w", err)
	}

	var notifications []*models.Notification
	if err := json.Unmarshal(data, &notifications); err != nil {
		return fmt.Errorf("failed to unmarshal notifications: %w", err)
	}

	for _, notification := range notifications {
		fs.notifications.Store(notification.Id, notification)
	}

	return nil
}

// ==================== МЕТОДЫ ДЛЯ БИБЛИОТЕКИ ВЛОЖЕНИЙ ====================

func (fs *Filestorage) AddAttachment(attachment *models.Attachment) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.attachments.Store(attachment.Id, attachment)
	return fs.saveAttachments()
}

func (fs *Filestorage) GetAttachment(id uuid.UUID) (*models.Attachment, error) {
	attachmentVal, ok := fs.attachments.Load(id)
	if !ok {
		return nil, errors.New("attachment not found")
	}
	return attachmentVal.(*models.Attachment), nil
}

func (fs *Filestorage) GetAllAttachments() ([]*models.Attachment, error) {
	var attachments []*models.Attachment

	fs.attachments.Range(func(key, value interface{}) bool {
		attachment := value.(*models.Attachment)
		attachments = append(attachments, attachment)
		return true
	})

	return attachments, nil
}

func (fs *Filestorage) UpdateAttachment(attachment *models.Attachment) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.attachments.Load(attachment.Id)
	if !ok {
		return errors.New("attachment not found")
	}

	fs.attachments.Store(attachment.Id, attachment)
	return fs.saveAttachments()
}

func (fs *Filestorage) DeleteAttachment(id uuid.UUID) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	_, ok := fs.attachments.Load(id)
	if !ok {
		return errors.New("attachment not found")
	}

	fs.attachments.Delete(id)
	return fs.saveAttachments()
}

func (fs *Filestorage) saveAttachments() error {
	var attachments []*models.Attachment

	fs.attachments.Range(func(key, value interface{}) bool {
		attachment := value.(*models.Attachment)
		attachments = append(attachments, attachment)
		return true
	})

	data, err := json.MarshalIndent(attachments, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal attachments: %w", err)
	}

	filePath := filepath.Join(fs.dataDir, AttachmentsFileName)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write attachments file: %w", err)
	}

	return nil
}

func (fs *Filestorage) loadAttachments() error {
	filePath := filepath.Join(fs.dataDir, AttachmentsFileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read attachments file: %w", err)
	}

	var attachments []*models.Attachment
	if err := json.Unmarshal(data, &attachments); err != nil {
		return fmt.Errorf("failed to unmarshal attachments: %w", err)
	}

	for _, attachment := range attachments {
		fs.attachments.Store(attachment.Id, attachment)
	}

	return nil
}
