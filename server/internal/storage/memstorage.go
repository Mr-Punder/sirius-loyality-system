package storage

import (
	"errors"
	"sync"

	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/google/uuid"
)

type Memstorage struct {
	users        sync.Map // uuid.UUID -> *models.User
	transactions sync.Map // uuid.UUID -> *models.Transaction
	codes        sync.Map // uuid.UUID -> *models.Code
	codeUsages   sync.Map // uuid.UUID -> *models.CodeUsage
	admins       sync.Map // int64 -> *models.Admin
}

func NewMemstorage() *Memstorage {
	return &Memstorage{}
}

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

func (m *Memstorage) GetUserPoints(userId uuid.UUID) (int, error) {
	userVal, ok := m.users.Load(userId)
	if !ok {
		return 0, errors.New("user not found")
	}
	user := userVal.(*models.User)
	if user.Deleted {
		return 0, errors.New("user not found")
	}
	return user.Points, nil
}

func (m *Memstorage) GetUserByTelegramm(telegramm string) (*models.User, error) {
	var foundUser *models.User

	m.users.Range(func(key, value interface{}) bool {
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

func (m *Memstorage) GetTransaction(transactionId uuid.UUID) (*models.Transaction, error) {
	transactionVal, ok := m.transactions.Load(transactionId)
	if !ok {
		return nil, errors.New("transaction not found")
	}
	return transactionVal.(*models.Transaction), nil
}

func (m *Memstorage) GetUserTransactions(userId uuid.UUID) ([]*models.Transaction, error) {
	var transactions []*models.Transaction

	m.transactions.Range(func(key, value interface{}) bool {
		transaction := value.(*models.Transaction)
		if transaction.UserId == userId {
			transactions = append(transactions, transaction)
		}
		return true // продолжаем итерацию
	})

	return transactions, nil
}

func (m *Memstorage) GetAllTransactions() ([]*models.Transaction, error) {
	var transactions []*models.Transaction

	m.transactions.Range(func(key, value interface{}) bool {
		transaction := value.(*models.Transaction)
		transactions = append(transactions, transaction)
		return true
	})

	return transactions, nil
}

func (m *Memstorage) GetCodeInfo(code uuid.UUID) (*models.Code, error) {
	codeVal, ok := m.codes.Load(code)
	if !ok {
		return nil, errors.New("code not found")
	}
	return codeVal.(*models.Code), nil
}

func (m *Memstorage) GetAllCodes() ([]*models.Code, error) {
	var codes []*models.Code

	m.codes.Range(func(key, value interface{}) bool {
		code := value.(*models.Code)
		codes = append(codes, code)
		return true
	})

	return codes, nil
}

func (m *Memstorage) GetCodeUsage(code uuid.UUID) ([]*models.CodeUsage, error) {
	var usages []*models.CodeUsage

	m.codeUsages.Range(func(key, value interface{}) bool {
		usage := value.(*models.CodeUsage)
		if usage.Code == code {
			usages = append(usages, usage)
		}
		return true // продолжаем итерацию
	})

	return usages, nil
}

func (m *Memstorage) GetAllCodeUsages() ([]*models.CodeUsage, error) {
	var usages []*models.CodeUsage

	m.codeUsages.Range(func(key, value interface{}) bool {
		usage := value.(*models.CodeUsage)
		usages = append(usages, usage)
		return true
	})

	return usages, nil
}

func (m *Memstorage) GetCodeUsageByUser(code uuid.UUID, userId uuid.UUID) (*models.CodeUsage, error) {
	var foundUsage *models.CodeUsage

	m.codeUsages.Range(func(key, value interface{}) bool {
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
func (m *Memstorage) AddUser(user *models.User) error {
	// Проверяем, существует ли пользователь с таким Telegram ID
	var exists bool

	m.users.Range(func(key, value interface{}) bool {
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

	m.users.Store(user.Id, user)
	return nil
}

func (m *Memstorage) AddTransaction(transaction *models.Transaction) error {
	// Проверяем, существует ли пользователь
	userVal, ok := m.users.Load(transaction.UserId)
	if !ok {
		return errors.New("user not found")
	}

	user := userVal.(*models.User)
	if user.Deleted {
		return errors.New("user not found")
	}

	// Обновляем баллы пользователя
	user.Points += transaction.Diff
	m.users.Store(user.Id, user)

	// Сохраняем транзакцию
	m.transactions.Store(transaction.Id, transaction)
	return nil
}

func (m *Memstorage) AddCode(code *models.Code) error {
	// Проверяем, существует ли код с таким ID
	_, ok := m.codes.Load(code.Code)
	if ok {
		return errors.New("code already exists")
	}

	m.codes.Store(code.Code, code)
	return nil
}

func (m *Memstorage) AddCodeUsage(usage *models.CodeUsage) error {
	// Проверяем, существует ли пользователь
	userVal, ok := m.users.Load(usage.UserId)
	if !ok {
		return errors.New("user not found")
	}

	user := userVal.(*models.User)
	if user.Deleted {
		return errors.New("user not found")
	}

	// Проверяем, существует ли код
	codeVal, ok := m.codes.Load(usage.Code)
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
	existingUsage, err := m.GetCodeUsageByUser(usage.Code, usage.UserId)
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
		m.codeUsages.Store(existingUsage.Id, existingUsage)
	} else {
		// Иначе создаем новое использование
		m.codeUsages.Store(usage.Id, usage)
	}

	// Увеличиваем счетчик использований кода
	code.AppliedCount++
	m.codes.Store(code.Code, code)

	return nil
}

// Методы для обновления данных
func (m *Memstorage) UpdateUser(user *models.User) error {
	// Проверяем, существует ли пользователь
	_, ok := m.users.Load(user.Id)
	if !ok {
		return errors.New("user not found")
	}

	m.users.Store(user.Id, user)
	return nil
}

func (m *Memstorage) UpdateCode(code *models.Code) error {
	// Проверяем, существует ли код
	_, ok := m.codes.Load(code.Code)
	if !ok {
		return errors.New("code not found")
	}

	m.codes.Store(code.Code, code)
	return nil
}

func (m *Memstorage) UpdateCodeUsage(usage *models.CodeUsage) error {
	// Проверяем, существует ли использование кода
	_, ok := m.codeUsages.Load(usage.Id)
	if !ok {
		return errors.New("code usage not found")
	}

	m.codeUsages.Store(usage.Id, usage)
	return nil
}

// Методы для удаления данных
func (m *Memstorage) DeleteUser(userId uuid.UUID) error {
	// Проверяем, существует ли пользователь
	userVal, ok := m.users.Load(userId)
	if !ok {
		return errors.New("user not found")
	}

	// Помечаем пользователя как удаленного (мягкое удаление)
	user := userVal.(*models.User)
	user.Deleted = true
	m.users.Store(userId, user)

	return nil
}

func (m *Memstorage) DeleteCode(code uuid.UUID) error {
	// Проверяем, существует ли код
	codeVal, ok := m.codes.Load(code)
	if !ok {
		return errors.New("code not found")
	}

	// Деактивируем код
	codeInfo := codeVal.(*models.Code)
	codeInfo.IsActive = false
	m.codes.Store(code, codeInfo)

	return nil
}

// Методы для работы с администраторами
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
	// Проверяем, существует ли администратор с таким ID
	_, ok := m.admins.Load(admin.ID)
	if ok {
		return errors.New("admin with this ID already exists")
	}

	m.admins.Store(admin.ID, admin)
	return nil
}

func (m *Memstorage) UpdateAdmin(admin *models.Admin) error {
	// Проверяем, существует ли администратор
	_, ok := m.admins.Load(admin.ID)
	if !ok {
		return errors.New("admin not found")
	}

	m.admins.Store(admin.ID, admin)
	return nil
}

func (m *Memstorage) DeleteAdmin(adminId int64) error {
	// Проверяем, существует ли администратор
	_, ok := m.admins.Load(adminId)
	if !ok {
		return errors.New("admin not found")
	}

	// Удаляем администратора
	m.admins.Delete(adminId)
	return nil
}
