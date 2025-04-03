package telegrambot

import (
	"fmt"
	"strings"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/google/uuid"
	tele "gopkg.in/telebot.v3"
)

// UserBot представляет бота для пользователей
type UserBot struct {
	bot       *tele.Bot
	storage   storage.Storage
	logger    logger.Logger
	config    Config
	apiClient *APIClient
}

// NewUserBot создает нового бота для пользователей
func NewUserBot(config Config, storage storage.Storage, logger logger.Logger) (*UserBot, error) {
	pref := tele.Settings{
		Token:  config.Token,
		Poller: &tele.LongPoller{Timeout: 10},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания бота: %w", err)
	}

	// Создаем API-клиент
	apiClient := NewAPIClient(config.ServerURL, config.APIToken, logger)

	return &UserBot{
		bot:       bot,
		storage:   storage,
		logger:    logger,
		config:    config,
		apiClient: apiClient,
	}, nil
}

// Start запускает бота
func (ub *UserBot) Start() error {
	ub.logger.Info("Запуск пользовательского бота")

	// Обработчик команды /start
	ub.bot.Handle("/start", ub.handleStart)

	// Обработчик команды /register
	ub.bot.Handle("/register", ub.handleRegister)

	// Обработчик команды /points
	ub.bot.Handle("/points", ub.handlePoints)

	// Обработчик для QR-кодов
	ub.bot.Handle(tele.OnText, ub.handleText)

	// Обработчики кнопок
	ub.bot.Handle("💰 Мои баллы", ub.handlePointsButton)
	ub.bot.Handle("📷 Сканировать QR-код", ub.handleScanQRButton)
	ub.bot.Handle("❓ Помощь", ub.handleHelpButton)
	ub.bot.Handle("📝 Регистрация", ub.handleRegisterButton)

	// Запуск бота
	go ub.bot.Start()

	return nil
}

// handlePointsButton обрабатывает нажатие на кнопку "Мои баллы"
func (ub *UserBot) handlePointsButton(c tele.Context) error {
	// Просто вызываем обработчик команды /points
	return ub.handlePoints(c)
}

// handleScanQRButton обрабатывает нажатие на кнопку "Сканировать QR-код"
func (ub *UserBot) handleScanQRButton(c tele.Context) error {
	ub.logger.Infof("Пользователь %d нажал на кнопку 'Сканировать QR-код'", c.Sender().ID)

	// Проверяем, зарегистрирован ли пользователь
	telegramID := fmt.Sprintf("%d", c.Sender().ID)
	users, err := ub.storage.GetAllUsers()
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей: %v", err)
		return c.Send("Произошла ошибка при проверке регистрации. Пожалуйста, попробуйте позже.")
	}

	var user *models.User
	for _, u := range users {
		if u.Telegramm == telegramID && !u.Deleted {
			user = u
			break
		}
	}

	if user == nil {
		// Пользователь не зарегистрирован
		return c.Send("Ты не зарегистрирован в системе. Используй кнопку 'Регистрация' для регистрации.")
	}

	// Отправляем сообщение с инструкцией
	return c.Send("Отправь мне QR-код в виде текста (UUID).")
}

// handleHelpButton обрабатывает нажатие на кнопку "Помощь"
func (ub *UserBot) handleHelpButton(c tele.Context) error {
	ub.logger.Infof("Пользователь %d нажал на кнопку 'Помощь'", c.Sender().ID)

	// Отправляем справку
	message := "Я бот системы лояльности. Вот что я умею:\n\n" +
		"- Регистрация в системе\n" +
		"- Просмотр баллов\n" +
		"- Сканирование QR-кодов для получения баллов\n\n" +
		"Используй кнопки внизу экрана для навигации."

	return c.Send(message)
}

// handleRegisterButton обрабатывает нажатие на кнопку "Регистрация"
func (ub *UserBot) handleRegisterButton(c tele.Context) error {
	// Просто вызываем обработчик команды /register
	return ub.handleRegister(c)
}

// Stop останавливает бота
func (ub *UserBot) Stop() error {
	ub.logger.Info("Остановка пользовательского бота")
	ub.bot.Stop()
	return nil
}

// createMainKeyboard создает основную клавиатуру с кнопками
func (ub *UserBot) createMainKeyboard(isRegistered bool) *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	// Создаем кнопки
	btnPoints := keyboard.Text("💰 Мои баллы")
	btnScanQR := keyboard.Text("📷 Сканировать QR-код")
	btnHelp := keyboard.Text("❓ Помощь")
	btnRegister := keyboard.Text("📝 Регистрация")

	// Добавляем кнопки на клавиатуру в зависимости от статуса регистрации
	if isRegistered {
		keyboard.Reply(
			keyboard.Row(btnPoints, btnScanQR),
			keyboard.Row(btnHelp),
		)
	} else {
		keyboard.Reply(
			keyboard.Row(btnRegister),
			keyboard.Row(btnHelp),
		)
	}

	return keyboard
}

// handleStart обрабатывает команду /start
func (ub *UserBot) handleStart(c tele.Context) error {
	ub.logger.Infof("Пользователь %d запустил бота", c.Sender().ID)

	// Проверяем, зарегистрирован ли пользователь
	telegramID := fmt.Sprintf("%d", c.Sender().ID)
	users, err := ub.storage.GetAllUsers()
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей: %v", err)
		return c.Send("Произошла ошибка при проверке регистрации. Пожалуйста, попробуйте позже.")
	}

	var user *models.User
	for _, u := range users {
		if u.Telegramm == telegramID && !u.Deleted {
			user = u
			break
		}
	}

	// Создаем клавиатуру
	var keyboard *tele.ReplyMarkup
	var message string

	if user != nil {
		// Пользователь уже зарегистрирован
		keyboard = ub.createMainKeyboard(true)
		message = fmt.Sprintf("Привет, %s! Ты уже зарегистрирован в системе. Используй кнопки для навигации.", user.FirstName)
	} else {
		// Пользователь не зарегистрирован
		keyboard = ub.createMainKeyboard(false)
		message = "Привет! Я бот системы лояльности. Для начала работы тебе нужно зарегистрироваться."
	}

	// Отправляем сообщение с клавиатурой
	return c.Send(message, keyboard)
}

// handleRegister обрабатывает команду /register
func (ub *UserBot) handleRegister(c tele.Context) error {
	ub.logger.Infof("Пользователь %d запросил регистрацию", c.Sender().ID)

	// Проверяем, зарегистрирован ли пользователь
	telegramID := fmt.Sprintf("%d", c.Sender().ID)
	users, err := ub.storage.GetAllUsers()
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей: %v", err)
		return c.Send("Произошла ошибка при проверке регистрации. Пожалуйста, попробуйте позже.")
	}

	for _, u := range users {
		if u.Telegramm == telegramID && !u.Deleted {
			// Пользователь уже зарегистрирован
			keyboard := ub.createMainKeyboard(true)
			return c.Send(fmt.Sprintf("Ты уже зарегистрирован в системе как %s %s.", u.FirstName, u.LastName), keyboard)
		}
	}

	// Запрашиваем данные для регистрации
	return c.Send("Для регистрации отправь свои данные в формате: Имя Фамилия Группа\nНапример: Иван Иванов Н1")
}

// handlePoints обрабатывает команду /points
func (ub *UserBot) handlePoints(c tele.Context) error {
	ub.logger.Infof("Пользователь %d запросил баллы", c.Sender().ID)

	// Получаем пользователя
	telegramID := fmt.Sprintf("%d", c.Sender().ID)
	users, err := ub.storage.GetAllUsers()
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей: %v", err)
		return c.Send("Произошла ошибка при получении баллов. Пожалуйста, попробуйте позже.")
	}

	var user *models.User
	for _, u := range users {
		if u.Telegramm == telegramID && !u.Deleted {
			user = u
			break
		}
	}

	if user == nil {
		// Пользователь не зарегистрирован
		keyboard := ub.createMainKeyboard(false)
		return c.Send("Ты не зарегистрирован в системе. Используй кнопку 'Регистрация' для регистрации.", keyboard)
	}

	// Отправляем информацию о баллах
	keyboard := ub.createMainKeyboard(true)
	return c.Send(fmt.Sprintf("У тебя %d баллов.", user.Points), keyboard)
}

// handleText обрабатывает текстовые сообщения
func (ub *UserBot) handleText(c tele.Context) error {
	text := c.Text()
	telegramID := fmt.Sprintf("%d", c.Sender().ID)

	// Проверяем, является ли сообщение QR-кодом (UUID)
	if isUUID(text) {
		return ub.handleQRCode(c, text)
	}

	// Проверяем, является ли сообщение данными для регистрации
	parts := strings.Fields(text)
	if len(parts) >= 3 {
		// Получаем имя, фамилию и группу
		firstName := parts[0]
		lastName := parts[1]
		groupInput := parts[2]

		// Нормализуем группу
		normalizedGroup, valid := NormalizeGroupName(groupInput)
		if !valid {
			return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).")
		}

		// Проверяем, зарегистрирован ли пользователь
		users, err := ub.storage.GetAllUsers()
		if err != nil {
			ub.logger.Errorf("Ошибка получения пользователей: %v", err)
			return c.Send("Произошла ошибка при регистрации. Пожалуйста, попробуйте позже.")
		}

		for _, u := range users {
			if u.Telegramm == telegramID && !u.Deleted {
				// Пользователь уже зарегистрирован
				return c.Send(fmt.Sprintf("Ты уже зарегистрирован в системе как %s %s.", u.FirstName, u.LastName))
			}
		}

		// Создаем нового пользователя
		user := &models.User{
			Id:               uuid.New(),
			Telegramm:        telegramID,
			FirstName:        firstName,
			LastName:         lastName,
			MiddleName:       "",
			Points:           0,
			Group:            normalizedGroup,
			RegistrationTime: models.GetCurrentTime(),
			Deleted:          false,
		}

		// Добавляем пользователя в хранилище
		if err := ub.storage.AddUser(user); err != nil {
			ub.logger.Errorf("Ошибка добавления пользователя: %v", err)
			return c.Send("Произошла ошибка при регистрации. Пожалуйста, попробуйте позже.")
		}

		// Отправляем сообщение об успешной регистрации с клавиатурой
		keyboard := ub.createMainKeyboard(true)
		return c.Send(fmt.Sprintf("Ты успешно зарегистрирован как %s %s в группе %s.", firstName, lastName, normalizedGroup), keyboard)
	}

	// Если сообщение не является QR-кодом или данными для регистрации, отправляем справку
	// Определяем, зарегистрирован ли пользователь
	users, err := ub.storage.GetAllUsers()
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей: %v", err)
		return c.Send("Произошла ошибка. Пожалуйста, попробуйте позже.")
	}

	var user *models.User
	for _, u := range users {
		if u.Telegramm == telegramID && !u.Deleted {
			user = u
			break
		}
	}

	var keyboard *tele.ReplyMarkup
	if user != nil {
		keyboard = ub.createMainKeyboard(true)
	} else {
		keyboard = ub.createMainKeyboard(false)
	}

	return c.Send("Я не понимаю это сообщение. Используй кнопки для навигации.", keyboard)
}

// handleQRCode обрабатывает QR-код
func (ub *UserBot) handleQRCode(c tele.Context, code string) error {
	ub.logger.Infof("Пользователь %d отправил QR-код: %s", c.Sender().ID, code)

	// Получаем пользователя
	telegramID := fmt.Sprintf("%d", c.Sender().ID)
	users, err := ub.storage.GetAllUsers()
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей: %v", err)
		return c.Send("Произошла ошибка при применении QR-кода. Пожалуйста, попробуйте позже.")
	}

	var user *models.User
	for _, u := range users {
		if u.Telegramm == telegramID && !u.Deleted {
			user = u
			break
		}
	}

	if user == nil {
		// Пользователь не зарегистрирован
		return c.Send("Ты не зарегистрирован в системе. Используй /register для регистрации.")
	}

	// Парсим UUID
	codeUUID, err := uuid.Parse(code)
	if err != nil {
		ub.logger.Errorf("Ошибка парсинга UUID: %v", err)
		return c.Send("Неверный формат QR-кода.")
	}

	// Получаем информацию о коде
	codeInfo, err := ub.storage.GetCodeInfo(codeUUID)
	if err != nil {
		ub.logger.Errorf("Ошибка получения информации о коде: %v", err)
		return c.Send("QR-код не найден.")
	}

	// Проверяем, активен ли код
	if !codeInfo.IsActive {
		return c.Send("Этот QR-код неактивен.")
	}

	// Проверяем, принадлежит ли пользователь к нужной группе
	if codeInfo.Group != "" && user.Group != codeInfo.Group {
		return c.Send(fmt.Sprintf("Этот QR-код предназначен только для группы %s.", codeInfo.Group))
	}

	// Создаем новое использование кода
	usage := &models.CodeUsage{
		Id:     uuid.New(),
		Code:   codeUUID,
		UserId: user.Id,
		Count:  1,
	}

	// Добавляем использование кода
	if err := ub.storage.AddCodeUsage(usage); err != nil {
		ub.logger.Errorf("Ошибка применения кода: %v", err)

		// Возвращаем более информативные ошибки в зависимости от типа ошибки
		switch err.Error() {
		case "code usage limit exceeded":
			return c.Send("Превышено общее количество использований QR-кода.")
		case "user code usage limit exceeded":
			return c.Send("Ты уже использовал этот QR-код максимальное количество раз.")
		case "code is not active":
			return c.Send("Этот QR-код неактивен.")
		default:
			return c.Send("Произошла ошибка при применении QR-кода. Пожалуйста, попробуйте позже.")
		}
	}

	// Создаем транзакцию
	transaction := &models.Transaction{
		Id:     uuid.New(),
		UserId: user.Id,
		Code:   codeUUID,
		Diff:   codeInfo.Amount,
		Time:   models.GetCurrentTime(),
	}

	// Добавляем транзакцию
	if err := ub.storage.AddTransaction(transaction); err != nil {
		ub.logger.Errorf("Ошибка добавления транзакции: %v", err)
		return c.Send("Произошла ошибка при добавлении баллов. Пожалуйста, попробуйте позже.")
	}

	// Получаем обновленные баллы пользователя
	points, err := ub.storage.GetUserPoints(user.Id)
	if err != nil {
		ub.logger.Errorf("Ошибка получения баллов пользователя: %v", err)
		return c.Send("Произошла ошибка при получении баллов. Пожалуйста, попробуйте позже.")
	}

	// Отправляем сообщение об успешном применении QR-кода с клавиатурой
	keyboard := ub.createMainKeyboard(true)
	return c.Send(fmt.Sprintf("QR-код успешно применен! Добавлено %d баллов. Теперь у тебя %d баллов.", codeInfo.Amount, points), keyboard)
}

// isUUID проверяет, является ли строка UUID
func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// Используем NormalizeGroupName из utils.go
