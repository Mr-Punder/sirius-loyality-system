package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/google/uuid"
)

// Константы для имен файлов
const (
	UsersFileName        = "users.json"
	TransactionsFileName = "transactions.json"
	CodesFileName        = "codes.json"
	CodeUsagesFileName   = "code_usages.json"
	AdminsFileName       = "admins.json"
)

// Filestorage реализует интерфейс Storage с хранением данных в файлах
type Filestorage struct {
	users        sync.Map // uuid.UUID -> *models.User
	transactions sync.Map // uuid.UUID -> *models.Transaction
	codes        sync.Map // uuid.UUID -> *models.Code
	codeUsages   sync.Map // uuid.UUID -> *models.CodeUsage
	admins       sync.Map // int64 -> *models.Admin
	dataDir      string   // Директория для хранения файлов данных
}

// NewFilestorage создает новое файловое хранилище
func NewFilestorage(dataDir string) (*Filestorage, error) {
	// Создаем директорию, если она не существует
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	fs := &Filestorage{
		dataDir: dataDir,
	}

	// Загружаем данные из файлов
	if err := fs.loadData(); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	return fs, nil
}

// loadData загружает данные из файлов
func (fs *Filestorage) loadData() error {
	// Загружаем пользователей
	if err := fs.loadUsers(); err != nil {
		return err
	}

	// Загружаем транзакции
	if err := fs.loadTransactions(); err != nil {
		return err
	}

	// Загружаем коды
	if err := fs.loadCodes(); err != nil {
		return err
	}

	// Загружаем использования кодов
	if err := fs.loadCodeUsages(); err != nil {
		return err
	}

	// Загружаем администраторов
	if err := fs.loadAdmins(); err != nil {
		return err
	}

	return nil
}

// loadAdmins загружает администраторов из файла
func (fs *Filestorage) loadAdmins() error {
	filePath := filepath.Join(fs.dataDir, AdminsFileName)

	// Проверяем, существует ли файл
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Файл не существует, создаем пустой файл
		if err := fs.saveAdmins(); err != nil {
			return err
		}
		return nil
	}

	// Читаем файл
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read admins file: %w", err)
	}

	// Если файл пустой, ничего не делаем
	if len(data) == 0 {
		return nil
	}

	// Десериализуем данные
	var admins []*models.Admin
	if err := json.Unmarshal(data, &admins); err != nil {
		return fmt.Errorf("failed to unmarshal admins: %w", err)
	}

	// Сохраняем администраторов в память
	for _, admin := range admins {
		fs.admins.Store(admin.ID, admin)
	}

	return nil
}

// saveAdmins сохраняет администраторов в файл
func (fs *Filestorage) saveAdmins() error {
	filePath := filepath.Join(fs.dataDir, AdminsFileName)

	// Собираем всех администраторов
	var admins []*models.Admin
	fs.admins.Range(func(key, value interface{}) bool {
		admins = append(admins, value.(*models.Admin))
		return true
	})

	// Сериализуем данные
	data, err := json.MarshalIndent(admins, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal admins: %w", err)
	}

	// Записываем в файл
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write admins file: %w", err)
	}

	return nil
}

// loadUsers загружает пользователей из файла
func (fs *Filestorage) loadUsers() error {
	filePath := filepath.Join(fs.dataDir, UsersFileName)

	// Проверяем, существует ли файл
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Файл не существует, создаем пустой файл
		if err := fs.saveUsers(); err != nil {
			return err
		}
		return nil
	}

	// Читаем файл
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read users file: %w", err)
	}

	// Если файл пустой, ничего не делаем
	if len(data) == 0 {
		return nil
	}

	// Десериализуем данные
	var users []*models.User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("failed to unmarshal users: %w", err)
	}

	// Сохраняем пользователей в память
	for _, user := range users {
		fs.users.Store(user.Id, user)
	}

	return nil
}

// loadTransactions загружает транзакции из файла
func (fs *Filestorage) loadTransactions() error {
	filePath := filepath.Join(fs.dataDir, TransactionsFileName)

	// Проверяем, существует ли файл
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Файл не существует, создаем пустой файл
		if err := fs.saveTransactions(); err != nil {
			return err
		}
		return nil
	}

	// Читаем файл
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read transactions file: %w", err)
	}

	// Если файл пустой, ничего не делаем
	if len(data) == 0 {
		return nil
	}

	// Десериализуем данные
	var transactions []*models.Transaction
	if err := json.Unmarshal(data, &transactions); err != nil {
		return fmt.Errorf("failed to unmarshal transactions: %w", err)
	}

	// Сохраняем транзакции в память
	for _, transaction := range transactions {
		fs.transactions.Store(transaction.Id, transaction)
	}

	return nil
}

// loadCodes загружает коды из файла
func (fs *Filestorage) loadCodes() error {
	filePath := filepath.Join(fs.dataDir, CodesFileName)

	// Проверяем, существует ли файл
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Файл не существует, создаем пустой файл
		if err := fs.saveCodes(); err != nil {
			return err
		}
		return nil
	}

	// Читаем файл
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read codes file: %w", err)
	}

	// Если файл пустой, ничего не делаем
	if len(data) == 0 {
		return nil
	}

	// Десериализуем данные
	var codes []*models.Code
	if err := json.Unmarshal(data, &codes); err != nil {
		return fmt.Errorf("failed to unmarshal codes: %w", err)
	}

	// Сохраняем коды в память
	for _, code := range codes {
		fs.codes.Store(code.Code, code)
	}

	return nil
}

// loadCodeUsages загружает использования кодов из файла
func (fs *Filestorage) loadCodeUsages() error {
	filePath := filepath.Join(fs.dataDir, CodeUsagesFileName)

	// Проверяем, существует ли файл
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Файл не существует, создаем пустой файл
		if err := fs.saveCodeUsages(); err != nil {
			return err
		}
		return nil
	}

	// Читаем файл
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read code usages file: %w", err)
	}

	// Если файл пустой, ничего не делаем
	if len(data) == 0 {
		return nil
	}

	// Десериализуем данные
	var codeUsages []*models.CodeUsage
	if err := json.Unmarshal(data, &codeUsages); err != nil {
		return fmt.Errorf("failed to unmarshal code usages: %w", err)
	}

	// Сохраняем использования кодов в память
	for _, usage := range codeUsages {
		fs.codeUsages.Store(usage.Id, usage)
	}

	return nil
}

// saveUsers сохраняет пользователей в файл
func (fs *Filestorage) saveUsers() error {
	filePath := filepath.Join(fs.dataDir, UsersFileName)

	// Собираем всех пользователей
	var users []*models.User
	fs.users.Range(func(key, value interface{}) bool {
		users = append(users, value.(*models.User))
		return true
	})

	// Сериализуем данные
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	// Записываем в файл
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write users file: %w", err)
	}

	return nil
}

// saveTransactions сохраняет транзакции в файл
func (fs *Filestorage) saveTransactions() error {
	filePath := filepath.Join(fs.dataDir, TransactionsFileName)

	// Собираем все транзакции
	var transactions []*models.Transaction
	fs.transactions.Range(func(key, value interface{}) bool {
		transactions = append(transactions, value.(*models.Transaction))
		return true
	})

	// Сериализуем данные
	data, err := json.MarshalIndent(transactions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal transactions: %w", err)
	}

	// Записываем в файл
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write transactions file: %w", err)
	}

	return nil
}

// saveCodes сохраняет коды в файл
func (fs *Filestorage) saveCodes() error {
	filePath := filepath.Join(fs.dataDir, CodesFileName)

	// Собираем все коды
	var codes []*models.Code
	fs.codes.Range(func(key, value interface{}) bool {
		codes = append(codes, value.(*models.Code))
		return true
	})

	// Сериализуем данные
	data, err := json.MarshalIndent(codes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal codes: %w", err)
	}

	// Записываем в файл
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write codes file: %w", err)
	}

	return nil
}

// saveCodeUsages сохраняет использования кодов в файл
func (fs *Filestorage) saveCodeUsages() error {
	filePath := filepath.Join(fs.dataDir, CodeUsagesFileName)

	// Собираем все использования кодов
	var codeUsages []*models.CodeUsage
	fs.codeUsages.Range(func(key, value interface{}) bool {
		codeUsages = append(codeUsages, value.(*models.CodeUsage))
		return true
	})

	// Сериализуем данные
	data, err := json.MarshalIndent(codeUsages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal code usages: %w", err)
	}

	// Записываем в файл
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write code usages file: %w", err)
	}

	return nil
}

// Методы для получения данных

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

func (fs *Filestorage) GetUserPoints(userId uuid.UUID) (int, error) {
	userVal, ok := fs.users.Load(userId)
	if !ok {
		return 0, errors.New("user not found")
	}
	user := userVal.(*models.User)
	if user.Deleted {
		return 0, errors.New("user not found")
	}
	return user.Points, nil
}

func (fs *Filestorage) GetUserByTelegramm(telegramm string) (*models.User, error) {
	var foundUser *models.User

	fs.users.Range(func(key, value interface{}) bool {
		user := value.(*models.User)
		if user.Telegramm == telegramm && !user.Deleted {
			foundUser = user
			return false // прекращаем итерацию
		}
		return true // продолжаем итерацию
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

func (fs *Filestorage) GetTransaction(transactionId uuid.UUID) (*models.Transaction, error) {
	transactionVal, ok := fs.transactions.Load(transactionId)
	if !ok {
		return nil, errors.New("transaction not found")
	}
	return transactionVal.(*models.Transaction), nil
}

func (fs *Filestorage) GetUserTransactions(userId uuid.UUID) ([]*models.Transaction, error) {
	var transactions []*models.Transaction

	fs.transactions.Range(func(key, value interface{}) bool {
		transaction := value.(*models.Transaction)
		if transaction.UserId == userId {
			transactions = append(transactions, transaction)
		}
		return true // продолжаем итерацию
	})

	return transactions, nil
}

func (fs *Filestorage) GetAllTransactions() ([]*models.Transaction, error) {
	var transactions []*models.Transaction

	fs.transactions.Range(func(key, value interface{}) bool {
		transaction := value.(*models.Transaction)
		transactions = append(transactions, transaction)
		return true
	})

	return transactions, nil
}

func (fs *Filestorage) GetCodeInfo(code uuid.UUID) (*models.Code, error) {
	codeVal, ok := fs.codes.Load(code)
	if !ok {
		return nil, errors.New("code not found")
	}
	return codeVal.(*models.Code), nil
}

func (fs *Filestorage) GetAllCodes() ([]*models.Code, error) {
	var codes []*models.Code

	fs.codes.Range(func(key, value interface{}) bool {
		code := value.(*models.Code)
		codes = append(codes, code)
		return true
	})

	return codes, nil
}

func (fs *Filestorage) GetCodeUsage(code uuid.UUID) ([]*models.CodeUsage, error) {
	var usages []*models.CodeUsage

	fs.codeUsages.Range(func(key, value interface{}) bool {
		usage := value.(*models.CodeUsage)
		if usage.Code == code {
			usages = append(usages, usage)
		}
		return true // продолжаем итерацию
	})

	return usages, nil
}

func (fs *Filestorage) GetAllCodeUsages() ([]*models.CodeUsage, error) {
	var usages []*models.CodeUsage

	fs.codeUsages.Range(func(key, value interface{}) bool {
		usage := value.(*models.CodeUsage)
		usages = append(usages, usage)
		return true
	})

	return usages, nil
}

func (fs *Filestorage) GetCodeUsageByUser(code uuid.UUID, userId uuid.UUID) (*models.CodeUsage, error) {
	var foundUsage *models.CodeUsage

	fs.codeUsages.Range(func(key, value interface{}) bool {
		usage := value.(*models.CodeUsage)
		if usage.Code == code && usage.UserId == userId {
			foundUsage = usage
			return false // прекращаем итерацию
		}
		return true // продолжаем итерацию
	})

	if foundUsage == nil {
		return nil, errors.New("code usage not found")
	}

	return foundUsage, nil
}

// Методы для добавления данных

func (fs *Filestorage) AddUser(user *models.User) error {
	// Проверяем, существует ли пользователь с таким Telegram ID
	var exists bool

	fs.users.Range(func(key, value interface{}) bool {
		existingUser := value.(*models.User)
		if existingUser.Telegramm == user.Telegramm && !existingUser.Deleted {
			exists = true
			return false // прекращаем итерацию
		}
		return true // продолжаем итерацию
	})

	if exists {
		return errors.New("user with this telegram ID already exists")
	}

	fs.users.Store(user.Id, user)

	// Сохраняем изменения в файл
	if err := fs.saveUsers(); err != nil {
		return fmt.Errorf("failed to save users: %w", err)
	}

	return nil
}

func (fs *Filestorage) AddTransaction(transaction *models.Transaction) error {
	// Проверяем, существует ли пользователь
	userVal, ok := fs.users.Load(transaction.UserId)
	if !ok {
		return errors.New("user not found")
	}

	user := userVal.(*models.User)
	if user.Deleted {
		return errors.New("user not found")
	}

	// Обновляем баллы пользователя
	user.Points += transaction.Diff
	fs.users.Store(user.Id, user)

	// Сохраняем транзакцию
	fs.transactions.Store(transaction.Id, transaction)

	// Сохраняем изменения в файлы
	if err := fs.saveUsers(); err != nil {
		return fmt.Errorf("failed to save users: %w", err)
	}

	if err := fs.saveTransactions(); err != nil {
		return fmt.Errorf("failed to save transactions: %w", err)
	}

	return nil
}

func (fs *Filestorage) AddCode(code *models.Code) error {
	// Проверяем, существует ли код с таким ID
	_, ok := fs.codes.Load(code.Code)
	if ok {
		return errors.New("code already exists")
	}

	fs.codes.Store(code.Code, code)

	// Сохраняем изменения в файл
	if err := fs.saveCodes(); err != nil {
		return fmt.Errorf("failed to save codes: %w", err)
	}

	return nil
}

func (fs *Filestorage) AddCodeUsage(usage *models.CodeUsage) error {
	// Проверяем, существует ли пользователь
	userVal, ok := fs.users.Load(usage.UserId)
	if !ok {
		return errors.New("user not found")
	}

	user := userVal.(*models.User)
	if user.Deleted {
		return errors.New("user not found")
	}

	// Проверяем, существует ли код
	codeVal, ok := fs.codes.Load(usage.Code)
	if !ok {
		return errors.New("code not found")
	}

	code := codeVal.(*models.Code)

	// Проверяем, активен ли код
	if !code.IsActive {
		return errors.New("code is not active")
	}

	// Проверяем, принадлежит ли пользователь к нужной группе
	if code.Group != "" && user.Group != code.Group {
		return errors.New("user group does not match code group")
	}

	// Проверяем, не превышено ли количество использований кода пользователем
	existingUsage, err := fs.GetCodeUsageByUser(usage.Code, usage.UserId)
	if err == nil && code.PerUser > 0 && existingUsage.Count >= code.PerUser {
		return errors.New("user code usage limit exceeded")
	}

	// Проверяем, не превышено ли общее количество использований кода
	if code.Total > 0 && code.AppliedCount >= code.Total {
		return errors.New("code usage limit exceeded")
	}

	// Если использование кода пользователем уже существует, обновляем его
	if err == nil {
		existingUsage.Count++
		fs.codeUsages.Store(existingUsage.Id, existingUsage)
	} else {
		// Иначе создаем новое использование
		fs.codeUsages.Store(usage.Id, usage)
	}

	// Увеличиваем счетчик использований кода
	code.AppliedCount++
	fs.codes.Store(code.Code, code)

	// Сохраняем изменения в файлы
	if err := fs.saveCodes(); err != nil {
		return fmt.Errorf("failed to save codes: %w", err)
	}

	if err := fs.saveCodeUsages(); err != nil {
		return fmt.Errorf("failed to save code usages: %w", err)
	}

	return nil
}

// Методы для обновления данных

func (fs *Filestorage) UpdateUser(user *models.User) error {
	// Проверяем, существует ли пользователь
	_, ok := fs.users.Load(user.Id)
	if !ok {
		return errors.New("user not found")
	}

	fs.users.Store(user.Id, user)

	// Сохраняем изменения в файл
	if err := fs.saveUsers(); err != nil {
		return fmt.Errorf("failed to save users: %w", err)
	}

	return nil
}

func (fs *Filestorage) UpdateCode(code *models.Code) error {
	// Проверяем, существует ли код
	_, ok := fs.codes.Load(code.Code)
	if !ok {
		return errors.New("code not found")
	}

	fs.codes.Store(code.Code, code)

	// Сохраняем изменения в файл
	if err := fs.saveCodes(); err != nil {
		return fmt.Errorf("failed to save codes: %w", err)
	}

	return nil
}

func (fs *Filestorage) UpdateCodeUsage(usage *models.CodeUsage) error {
	// Проверяем, существует ли использование кода
	_, ok := fs.codeUsages.Load(usage.Id)
	if !ok {
		return errors.New("code usage not found")
	}

	fs.codeUsages.Store(usage.Id, usage)

	// Сохраняем изменения в файл
	if err := fs.saveCodeUsages(); err != nil {
		return fmt.Errorf("failed to save code usages: %w", err)
	}

	return nil
}

// Методы для удаления данных

func (fs *Filestorage) DeleteUser(userId uuid.UUID) error {
	// Проверяем, существует ли пользователь
	userVal, ok := fs.users.Load(userId)
	if !ok {
		return errors.New("user not found")
	}

	// Помечаем пользователя как удаленного (мягкое удаление)
	user := userVal.(*models.User)
	user.Deleted = true
	fs.users.Store(userId, user)

	// Сохраняем изменения в файл
	if err := fs.saveUsers(); err != nil {
		return fmt.Errorf("failed to save users: %w", err)
	}

	return nil
}

func (fs *Filestorage) DeleteCode(code uuid.UUID) error {
	// Проверяем, существует ли код
	codeVal, ok := fs.codes.Load(code)
	if !ok {
		return errors.New("code not found")
	}

	// Деактивируем код
	codeInfo := codeVal.(*models.Code)
	codeInfo.IsActive = false
	fs.codes.Store(code, codeInfo)

	// Сохраняем изменения в файл
	if err := fs.saveCodes(); err != nil {
		return fmt.Errorf("failed to save codes: %w", err)
	}

	return nil
}

// Методы для работы с администраторами

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
	// Проверяем, существует ли администратор с таким ID
	_, ok := fs.admins.Load(admin.ID)
	if ok {
		return errors.New("admin with this ID already exists")
	}

	fs.admins.Store(admin.ID, admin)

	// Сохраняем изменения в файл
	if err := fs.saveAdmins(); err != nil {
		return fmt.Errorf("failed to save admins: %w", err)
	}

	return nil
}

func (fs *Filestorage) UpdateAdmin(admin *models.Admin) error {
	// Проверяем, существует ли администратор
	_, ok := fs.admins.Load(admin.ID)
	if !ok {
		return errors.New("admin not found")
	}

	fs.admins.Store(admin.ID, admin)

	// Сохраняем изменения в файл
	if err := fs.saveAdmins(); err != nil {
		return fmt.Errorf("failed to save admins: %w", err)
	}

	return nil
}

func (fs *Filestorage) DeleteAdmin(adminId int64) error {
	// Проверяем, существует ли администратор
	_, ok := fs.admins.Load(adminId)
	if !ok {
		return errors.New("admin not found")
	}

	// Удаляем администратора
	fs.admins.Delete(adminId)

	// Сохраняем изменения в файл
	if err := fs.saveAdmins(); err != nil {
		return fmt.Errorf("failed to save admins: %w", err)
	}

	return nil
}
