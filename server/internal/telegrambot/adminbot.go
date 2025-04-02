package telegrambot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/google/uuid"
	tele "gopkg.in/telebot.v3"
)

// AdminsList представляет список администраторов
type AdminsList struct {
	Admins []int64 `json:"admins"`
}

// AdminBot представляет бота для администраторов
type AdminBot struct {
	bot        *tele.Bot
	storage    storage.Storage
	logger     logger.Logger
	config     Config
	admins     AdminsList
	adminsPath string
}

// NewAdminBot создает нового бота для администраторов
func NewAdminBot(config Config, storage storage.Storage, logger logger.Logger) (*AdminBot, error) {
	// Определяем путь к файлу с администраторами
	adminsPath := filepath.Join("cmd", "telegrambot", "admin", "admins.json")
	pref := tele.Settings{
		Token:  config.Token,
		Poller: &tele.LongPoller{Timeout: 10},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания бота: %w", err)
	}

	// Создаем бота
	adminBot := &AdminBot{
		bot:        bot,
		storage:    storage,
		logger:     logger,
		config:     config,
		adminsPath: adminsPath,
	}

	// Загружаем список администраторов
	if err := adminBot.loadAdmins(); err != nil {
		logger.Errorf("Ошибка загрузки списка администраторов: %v", err)
		// Создаем пустой список, если файл не существует
		adminBot.admins = AdminsList{
			Admins: []int64{config.AdminUserID}, // Добавляем ID из конфигурации
		}
		// Сохраняем список администраторов
		if err := adminBot.saveAdmins(); err != nil {
			logger.Errorf("Ошибка сохранения списка администраторов: %v", err)
		}
	}

	return adminBot, nil
}

// Start запускает бота
func (ab *AdminBot) Start() error {
	ab.logger.Info("Запуск административного бота")

	// Обработчик команды /start
	ab.bot.Handle("/start", ab.handleStart)

	// Обработчик команды /users
	ab.bot.Handle("/users", ab.handleUsers)

	// Обработчик команды /user
	ab.bot.Handle("/user", ab.handleUser)

	// Обработчик команды /addpoints
	ab.bot.Handle("/addpoints", ab.handleAddPoints)

	// Обработчик команды /generatecode
	ab.bot.Handle("/generatecode", ab.handleGenerateCode)

	// Обработчик команды /addadmin
	ab.bot.Handle("/addadmin", ab.handleAddAdmin)

	// Обработчик команды /listadmins
	ab.bot.Handle("/listadmins", ab.handleListAdmins)

	// Обработчик команды /help
	ab.bot.Handle("/help", ab.handleHelp)

	// Запуск бота
	go ab.bot.Start()

	return nil
}

// Stop останавливает бота
func (ab *AdminBot) Stop() error {
	ab.logger.Info("Остановка административного бота")
	ab.bot.Stop()
	return nil
}

// loadAdmins загружает список администраторов из файла
func (ab *AdminBot) loadAdmins() error {
	// Проверяем, существует ли файл
	if _, err := os.Stat(ab.adminsPath); os.IsNotExist(err) {
		return err
	}

	// Читаем файл
	data, err := os.ReadFile(ab.adminsPath)
	if err != nil {
		return err
	}

	// Декодируем JSON
	if err := json.Unmarshal(data, &ab.admins); err != nil {
		return err
	}

	return nil
}

// saveAdmins сохраняет список администраторов в файл
func (ab *AdminBot) saveAdmins() error {
	// Кодируем JSON
	data, err := json.MarshalIndent(ab.admins, "", "    ")
	if err != nil {
		return err
	}

	// Создаем директорию, если она не существует
	dir := filepath.Dir(ab.adminsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Записываем файл
	if err := os.WriteFile(ab.adminsPath, data, 0644); err != nil {
		return err
	}

	return nil
}

// isAdmin проверяет, является ли пользователь администратором
func (ab *AdminBot) isAdmin(userID int64) bool {
	// Проверяем, есть ли пользователь в списке администраторов
	for _, adminID := range ab.admins.Admins {
		if userID == adminID {
			return true
		}
	}

	return false
}

// handleStart обрабатывает команду /start
func (ab *AdminBot) handleStart(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запустил бота", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этому боту.")
	}

	// Отправляем приветственное сообщение
	return c.Send("Привет, администратор! Я бот для управления системой лояльности. Используй /help для просмотра доступных команд.")
}

// handleUsers обрабатывает команду /users
func (ab *AdminBot) handleUsers(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил список пользователей", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	// Получаем параметры команды
	args := strings.Fields(c.Message().Payload)
	var group string
	if len(args) > 0 {
		group = args[0]
	}

	// Получаем всех пользователей
	users, err := ab.storage.GetAllUsers()
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователей: %v", err)
		return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
	}

	// Фильтруем пользователей по группе и исключаем удаленных
	var filteredUsers []*models.User
	for _, user := range users {
		if (group == "" || user.Group == group) && !user.Deleted {
			filteredUsers = append(filteredUsers, user)
		}
	}

	if len(filteredUsers) == 0 {
		if group == "" {
			return c.Send("Пользователи не найдены.")
		} else {
			return c.Send(fmt.Sprintf("Пользователи в группе %s не найдены.", group))
		}
	}

	// Формируем сообщение со списком пользователей
	var message strings.Builder
	if group == "" {
		message.WriteString("Список всех пользователей:\n\n")
	} else {
		message.WriteString(fmt.Sprintf("Список пользователей в группе %s:\n\n", group))
	}

	for i, user := range filteredUsers {
		message.WriteString(fmt.Sprintf("%d. %s %s (ID: %s, Группа: %s, Баллы: %d)\n",
			i+1, user.FirstName, user.LastName, user.Id, user.Group, user.Points))
	}

	// Отправляем сообщение
	return c.Send(message.String())
}

// handleUser обрабатывает команду /user
func (ab *AdminBot) handleUser(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил информацию о пользователе", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	// Получаем ID пользователя из параметров команды
	args := strings.Fields(c.Message().Payload)
	if len(args) == 0 {
		return c.Send("Укажите ID пользователя. Например: /user 123e4567-e89b-12d3-a456-426614174000")
	}

	userID, err := uuid.Parse(args[0])
	if err != nil {
		return c.Send("Неверный формат ID пользователя. Используйте UUID.")
	}

	// Получаем пользователя
	user, err := ab.storage.GetUser(userID)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователя: %v", err)
		return c.Send("Пользователь не найден.")
	}

	if user.Deleted {
		return c.Send("Пользователь удален.")
	}

	// Формируем сообщение с информацией о пользователе
	message := fmt.Sprintf("Информация о пользователе:\n\n"+
		"ID: %s\n"+
		"Имя: %s\n"+
		"Фамилия: %s\n"+
		"Отчество: %s\n"+
		"Telegram: %s\n"+
		"Группа: %s\n"+
		"Баллы: %d\n"+
		"Дата регистрации: %s",
		user.Id, user.FirstName, user.LastName, user.MiddleName,
		user.Telegramm, user.Group, user.Points, user.RegistrationTime.Format("02.01.2006 15:04:05"))

	// Отправляем сообщение
	return c.Send(message)
}

// handleAddPoints обрабатывает команду /addpoints
func (ab *AdminBot) handleAddPoints(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил добавление баллов", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	// Получаем параметры команды
	args := strings.Fields(c.Message().Payload)
	if len(args) < 2 {
		return c.Send("Укажите ID пользователя и количество баллов. Например: /addpoints 123e4567-e89b-12d3-a456-426614174000 10")
	}

	userID, err := uuid.Parse(args[0])
	if err != nil {
		return c.Send("Неверный формат ID пользователя. Используйте UUID.")
	}

	points, err := strconv.Atoi(args[1])
	if err != nil {
		return c.Send("Неверный формат количества баллов. Используйте целое число.")
	}

	// Получаем пользователя
	user, err := ab.storage.GetUser(userID)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователя: %v", err)
		return c.Send("Пользователь не найден.")
	}

	if user.Deleted {
		return c.Send("Пользователь удален.")
	}

	// Создаем транзакцию
	transaction := &models.Transaction{
		Id:     uuid.New(),
		UserId: userID,
		Diff:   points,
		Time:   models.GetCurrentTime(),
	}

	// Добавляем транзакцию
	if err := ab.storage.AddTransaction(transaction); err != nil {
		ab.logger.Errorf("Ошибка добавления транзакции: %v", err)
		return c.Send("Произошла ошибка при добавлении баллов. Пожалуйста, попробуйте позже.")
	}

	// Получаем обновленные баллы пользователя
	updatedPoints, err := ab.storage.GetUserPoints(userID)
	if err != nil {
		ab.logger.Errorf("Ошибка получения баллов пользователя: %v", err)
		return c.Send("Произошла ошибка при получении баллов. Пожалуйста, попробуйте позже.")
	}

	// Отправляем сообщение об успешном добавлении баллов
	return c.Send(fmt.Sprintf("Баллы успешно добавлены!\nПользователь: %s %s\nДобавлено: %d\nВсего баллов: %d",
		user.FirstName, user.LastName, points, updatedPoints))
}

// handleGenerateCode обрабатывает команду /generatecode
func (ab *AdminBot) handleGenerateCode(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил генерацию QR-кода", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	// Получаем параметры команды
	args := strings.Fields(c.Message().Payload)
	if len(args) < 2 {
		return c.Send("Укажите количество баллов и ограничения. Например: /generatecode 10 3 5 Н1\n" +
			"Где:\n" +
			"10 - количество баллов\n" +
			"3 - максимальное количество использований одним пользователем\n" +
			"5 - общее количество использований\n" +
			"Н1 - группа (необязательно)")
	}

	// Парсим параметры
	amount, err := strconv.Atoi(args[0])
	if err != nil {
		return c.Send("Неверный формат количества баллов. Используйте целое число.")
	}

	perUser := 1
	if len(args) > 1 {
		perUser, err = strconv.Atoi(args[1])
		if err != nil {
			return c.Send("Неверный формат ограничения на пользователя. Используйте целое число.")
		}
	}

	total := 0
	if len(args) > 2 {
		total, err = strconv.Atoi(args[2])
		if err != nil {
			return c.Send("Неверный формат общего ограничения. Используйте целое число.")
		}
	}

	var group string
	if len(args) > 3 {
		group = args[3]
	}

	// Создаем новый код
	code := &models.Code{
		Code:         uuid.New(),
		Amount:       amount,
		PerUser:      perUser,
		Total:        total,
		AppliedCount: 0,
		IsActive:     true,
		Group:        group,
		ErrorCode:    models.ErrorCodeNone,
	}

	// Добавляем код в хранилище
	if err := ab.storage.AddCode(code); err != nil {
		ab.logger.Errorf("Ошибка добавления кода: %v", err)
		return c.Send("Произошла ошибка при генерации QR-кода. Пожалуйста, попробуйте позже.")
	}

	// Формируем сообщение с информацией о коде
	message := fmt.Sprintf("QR-код успешно сгенерирован!\n\n"+
		"Код: %s\n"+
		"Баллы: %d\n"+
		"Ограничение на пользователя: %d\n"+
		"Общее ограничение: %d\n"+
		"Группа: %s\n\n"+
		"Пользователи могут применить этот код, отправив его текстом боту.",
		code.Code, code.Amount, code.PerUser, code.Total, code.Group)

	// Отправляем сообщение
	return c.Send(message)
}

// handleAddAdmin обрабатывает команду /addadmin
func (ab *AdminBot) handleAddAdmin(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил добавление администратора", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	// Получаем ID нового администратора из параметров команды
	args := strings.Fields(c.Message().Payload)
	if len(args) == 0 {
		return c.Send("Укажите ID пользователя. Например: /addadmin 123456789")
	}

	// Парсим ID
	adminID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return c.Send("Неверный формат ID пользователя. Используйте целое число.")
	}

	// Проверяем, есть ли уже такой администратор
	for _, id := range ab.admins.Admins {
		if id == adminID {
			return c.Send(fmt.Sprintf("Пользователь с ID %d уже является администратором.", adminID))
		}
	}

	// Добавляем нового администратора
	ab.admins.Admins = append(ab.admins.Admins, adminID)

	// Сохраняем список администраторов
	if err := ab.saveAdmins(); err != nil {
		ab.logger.Errorf("Ошибка сохранения списка администраторов: %v", err)
		return c.Send("Произошла ошибка при сохранении списка администраторов.")
	}

	// Отправляем сообщение об успешном добавлении
	return c.Send(fmt.Sprintf("Пользователь с ID %d успешно добавлен в список администраторов.", adminID))
}

// handleListAdmins обрабатывает команду /listadmins
func (ab *AdminBot) handleListAdmins(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил список администраторов", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	// Формируем сообщение со списком администраторов
	var message strings.Builder
	message.WriteString("Список администраторов:\n\n")

	for i, adminID := range ab.admins.Admins {
		message.WriteString(fmt.Sprintf("%d. %d\n", i+1, adminID))
	}

	// Отправляем сообщение
	return c.Send(message.String())
}

// handleHelp обрабатывает команду /help
func (ab *AdminBot) handleHelp(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил справку", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этому боту.")
	}

	// Формируем сообщение со справкой
	message := "Доступные команды:\n\n" +
		"/users [группа] - Список пользователей (опционально фильтр по группе)\n" +
		"/user <ID> - Информация о пользователе\n" +
		"/addpoints <ID> <баллы> - Добавить баллы пользователю\n" +
		"/generatecode <баллы> [перПользователя] [всего] [группа] - Сгенерировать QR-код\n" +
		"/addadmin <ID> - Добавить администратора\n" +
		"/listadmins - Список администраторов\n" +
		"/help - Показать эту справку"

	// Отправляем сообщение
	return c.Send(message)
}
