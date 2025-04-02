package telegrambot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/google/uuid"
	tele "gopkg.in/telebot.v3"
)

// Регулярное выражение для проверки группы
var groupRegex = regexp.MustCompile(`^[НHнh][1-6]$`)

// UserBot представляет бота для пользователей
type UserBot struct {
	bot     *tele.Bot
	storage storage.Storage
	logger  logger.Logger
	config  Config
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

	return &UserBot{
		bot:     bot,
		storage: storage,
		logger:  logger,
		config:  config,
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

	// Запуск бота
	go ub.bot.Start()

	return nil
}

// Stop останавливает бота
func (ub *UserBot) Stop() error {
	ub.logger.Info("Остановка пользовательского бота")
	ub.bot.Stop()
	return nil
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

	if user != nil {
		// Пользователь уже зарегистрирован
		return c.Send(fmt.Sprintf("Привет, %s! Ты уже зарегистрирован в системе. Используй /points для просмотра своих баллов.", user.FirstName))
	}

	// Пользователь не зарегистрирован
	return c.Send("Привет! Я бот системы лояльности. Для регистрации используй команду /register.")
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
			return c.Send(fmt.Sprintf("Ты уже зарегистрирован в системе как %s %s.", u.FirstName, u.LastName))
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
		return c.Send("Ты не зарегистрирован в системе. Используй /register для регистрации.")
	}

	// Отправляем информацию о баллах
	return c.Send(fmt.Sprintf("У тебя %d баллов.", user.Points))
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
		normalizedGroup, valid := normalizeGroup(groupInput)
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

		// Отправляем сообщение об успешной регистрации
		return c.Send(fmt.Sprintf("Ты успешно зарегистрирован как %s %s в группе %s.", firstName, lastName, normalizedGroup))
	}

	// Если сообщение не является QR-кодом или данными для регистрации, отправляем справку
	return c.Send("Я не понимаю это сообщение. Используй /register для регистрации или /points для просмотра баллов.")
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

	// Отправляем сообщение об успешном применении QR-кода
	return c.Send(fmt.Sprintf("QR-код успешно применен! Добавлено %d баллов. Теперь у тебя %d баллов.", codeInfo.Amount, points))
}

// isUUID проверяет, является ли строка UUID
func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// normalizeGroup нормализует группу (Н1-Н6, H1-H6, н1-н6, h1-h6) -> Н1-Н6
func normalizeGroup(group string) (string, bool) {
	if !groupRegex.MatchString(group) {
		return "", false
	}

	// Получаем номер группы
	number := group[len(group)-1]

	// Возвращаем нормализованную группу
	return fmt.Sprintf("Н%c", number), true
}
