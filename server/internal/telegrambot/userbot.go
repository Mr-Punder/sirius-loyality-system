package telegrambot

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/google/uuid"
	tele "gopkg.in/telebot.v3"
)

// Константы для этапов регистрации
const (
	RegistrationStepLastName   = 1
	RegistrationStepFirstName  = 2
	RegistrationStepMiddleName = 3
	RegistrationStepGroup      = 4
)

// RegistrationState хранит состояние регистрации пользователя
type RegistrationState struct {
	Step       int
	LastName   string
	FirstName  string
	MiddleName string
	Group      string
}

// UserBot представляет бота для пользователей
type UserBot struct {
	bot                *tele.Bot
	logger             logger.Logger
	config             Config
	apiClient          *APIClient
	registrationStates map[int64]*RegistrationState // Карта для хранения состояний регистрации
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
		bot:                bot,
		logger:             logger,
		config:             config,
		apiClient:          apiClient,
		registrationStates: make(map[int64]*RegistrationState),
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

	// Обработчик для QR-кодов и текстовых сообщений
	ub.bot.Handle(tele.OnText, ub.handleText)

	// Обработчики кнопок
	ub.bot.Handle("💰 Мои баллы", ub.handlePointsButton)
	ub.bot.Handle("📷 Сканировать QR-код", ub.handleScanQRButton)
	ub.bot.Handle("❓ Помощь", ub.handleHelpButton)
	ub.bot.Handle("📝 Регистрация", ub.handleRegisterButton)
	ub.bot.Handle("⏭️ Пропустить", ub.handleSkipButton)
	ub.bot.Handle("❌ Отменить", ub.handleCancelButton)

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

	// Получаем список пользователей через API
	usersData, err := ub.apiClient.Get("/users", nil)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send("Произошла ошибка при проверке регистрации. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при проверке регистрации. Пожалуйста, попробуйте позже.")
	}

	var user *models.User
	for _, u := range usersResponse.Users {
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

// createRegistrationKeyboard создает клавиатуру для регистрации
func (ub *UserBot) createRegistrationKeyboard(withSkip bool) *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnCancel := keyboard.Text("❌ Отменить")

	if withSkip {
		// Клавиатура с кнопками "Пропустить" и "Отменить"
		btnSkip := keyboard.Text("⏭️ Пропустить")
		keyboard.Reply(
			keyboard.Row(btnSkip),
			keyboard.Row(btnCancel),
		)
	} else {
		// Клавиатура только с кнопкой "Отменить"
		keyboard.Reply(keyboard.Row(btnCancel))
	}

	return keyboard
}

// handleSkipButton обрабатывает нажатие на кнопку "Пропустить"
func (ub *UserBot) handleSkipButton(c tele.Context) error {
	// Получаем состояние регистрации
	state, exists := ub.registrationStates[c.Sender().ID]
	if !exists {
		return c.Send("Ты не находишься в процессе регистрации. Используй /register для начала регистрации.")
	}

	// Обрабатываем пропуск в зависимости от текущего шага
	if state.Step == RegistrationStepMiddleName {
		// Пропускаем отчество
		state.MiddleName = ""
		state.Step = RegistrationStepGroup
		// Отправляем запрос группы с кнопкой "Отменить"
		keyboard := ub.createRegistrationKeyboard(false)
		return c.Send("Введи свою группу (Н1-Н6):", keyboard)
	}

	return c.Send("На данном этапе нельзя пропустить ввод.")
}

// handleCancelButton обрабатывает нажатие на кнопку "Отменить"
func (ub *UserBot) handleCancelButton(c tele.Context) error {
	// Проверяем, находится ли пользователь в процессе регистрации
	_, exists := ub.registrationStates[c.Sender().ID]
	if !exists {
		return c.Send("Ты не находишься в процессе регистрации.")
	}

	// Удаляем состояние регистрации
	delete(ub.registrationStates, c.Sender().ID)

	// Отправляем сообщение об отмене регистрации с основной клавиатурой
	keyboard := ub.createMainKeyboard(false)
	return c.Send("Регистрация отменена.", keyboard)
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

	// Получаем список пользователей через API
	usersData, err := ub.apiClient.Get("/users", nil)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send("Произошла ошибка при проверке регистрации. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при проверке регистрации. Пожалуйста, попробуйте позже.")
	}

	var user *models.User
	for _, u := range usersResponse.Users {
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

	// Получаем список пользователей через API
	usersData, err := ub.apiClient.Get("/users", nil)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send("Произошла ошибка при проверке регистрации. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при проверке регистрации. Пожалуйста, попробуйте позже.")
	}

	for _, u := range usersResponse.Users {
		if u.Telegramm == telegramID && !u.Deleted {
			// Пользователь уже зарегистрирован
			keyboard := ub.createMainKeyboard(true)
			return c.Send(fmt.Sprintf("Ты уже зарегистрирован в системе как %s %s.", u.FirstName, u.LastName), keyboard)
		}
	}

	// Начинаем процесс регистрации
	ub.registrationStates[c.Sender().ID] = &RegistrationState{
		Step: RegistrationStepLastName,
	}

	// Запрашиваем фамилию с кнопкой "Отменить"
	keyboard := ub.createRegistrationKeyboard(false)
	return c.Send("Для регистрации введи свою фамилию:", keyboard)
}

// handlePoints обрабатывает команду /points
func (ub *UserBot) handlePoints(c tele.Context) error {
	ub.logger.Infof("Пользователь %d запросил баллы", c.Sender().ID)

	// Получаем пользователя
	telegramID := fmt.Sprintf("%d", c.Sender().ID)

	// Получаем список пользователей через API
	usersData, err := ub.apiClient.Get("/users", nil)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send("Произошла ошибка при получении баллов. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении баллов. Пожалуйста, попробуйте позже.")
	}

	var user *models.User
	for _, u := range usersResponse.Users {
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

	// Проверяем, находится ли пользователь в процессе регистрации
	state, inRegistration := ub.registrationStates[c.Sender().ID]
	if inRegistration {
		return ub.handleRegistrationStep(c, text, state)
	}

	// Если сообщение не является QR-кодом или пользователь не в процессе регистрации, отправляем справку
	// Определяем, зарегистрирован ли пользователь
	// Получаем список пользователей через API
	usersData, err := ub.apiClient.Get("/users", nil)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send("Произошла ошибка. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка. Пожалуйста, попробуйте позже.")
	}

	var user *models.User
	for _, u := range usersResponse.Users {
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

// handleRegistrationStep обрабатывает шаги регистрации
func (ub *UserBot) handleRegistrationStep(c tele.Context, text string, state *RegistrationState) error {
	// Проверяем, является ли текст командой "Пропустить" или "Отменить"
	if text == "⏭️ Пропустить" {
		return ub.handleSkipButton(c)
	} else if text == "❌ Отменить" {
		return ub.handleCancelButton(c)
	}

	switch state.Step {
	case RegistrationStepLastName:
		// Сохраняем фамилию
		state.LastName = text
		state.Step = RegistrationStepFirstName
		// Отправляем запрос имени с кнопкой "Отменить"
		keyboard := ub.createRegistrationKeyboard(false)
		return c.Send("Теперь введи своё имя:", keyboard)

	case RegistrationStepFirstName:
		// Сохраняем имя
		state.FirstName = text
		state.Step = RegistrationStepMiddleName
		// Отправляем запрос отчества с кнопками "Пропустить" и "Отменить"
		keyboard := ub.createRegistrationKeyboard(true)
		return c.Send("Введи своё отчество (или нажми кнопку 'Пропустить'):", keyboard)

	case RegistrationStepMiddleName:
		// Сохраняем отчество
		state.MiddleName = text
		state.Step = RegistrationStepGroup
		// Отправляем запрос группы с кнопкой "Отменить"
		keyboard := ub.createRegistrationKeyboard(false)
		return c.Send("Введи свою группу (Н1-Н6):", keyboard)

	case RegistrationStepGroup:
		// Нормализуем группу
		normalizedGroup, valid := NormalizeGroupName(text)
		if !valid {
			// Отправляем сообщение об ошибке с кнопкой "Отменить"
			keyboard := ub.createRegistrationKeyboard(false)
			return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).", keyboard)
		}

		// Сохраняем группу
		state.Group = normalizedGroup

		// Проверяем, зарегистрирован ли пользователь
		telegramID := fmt.Sprintf("%d", c.Sender().ID)

		// Получаем список пользователей через API
		usersData, err := ub.apiClient.Get("/users", nil)
		if err != nil {
			ub.logger.Errorf("Ошибка получения пользователей через API: %v", err)
			return c.Send("Произошла ошибка при регистрации. Пожалуйста, попробуйте позже.")
		}

		// Декодируем ответ
		var usersResponse struct {
			Total int            `json:"total"`
			Users []*models.User `json:"users"`
		}
		if err := json.Unmarshal(usersData, &usersResponse); err != nil {
			ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
			return c.Send("Произошла ошибка при регистрации. Пожалуйста, попробуйте позже.")
		}

		for _, u := range usersResponse.Users {
			if u.Telegramm == telegramID && !u.Deleted {
				// Пользователь уже зарегистрирован
				delete(ub.registrationStates, c.Sender().ID) // Удаляем состояние регистрации
				return c.Send(fmt.Sprintf("Ты уже зарегистрирован в системе как %s %s.", u.FirstName, u.LastName))
			}
		}

		// Добавляем пользователя через API
		_, err = ub.apiClient.Post("/users/register", map[string]interface{}{
			"telegramm":   telegramID,
			"first_name":  state.FirstName,
			"last_name":   state.LastName,
			"middle_name": state.MiddleName,
			"group":       state.Group,
		})
		if err != nil {
			ub.logger.Errorf("Ошибка добавления пользователя через API: %v", err)
			return c.Send("Произошла ошибка при регистрации. Пожалуйста, попробуйте позже.")
		}

		// Удаляем состояние регистрации
		delete(ub.registrationStates, c.Sender().ID)

		// Отправляем сообщение об успешной регистрации с клавиатурой
		keyboard := ub.createMainKeyboard(true)

		// Формируем сообщение об успешной регистрации
		successMessage := fmt.Sprintf("Ты успешно зарегистрирован как %s %s", state.LastName, state.FirstName)
		if state.MiddleName != "" {
			successMessage += fmt.Sprintf(" %s", state.MiddleName)
		}
		successMessage += fmt.Sprintf(" в группе %s.", state.Group)

		return c.Send(successMessage, keyboard)
	}

	return nil
}

// handleQRCode обрабатывает QR-код
func (ub *UserBot) handleQRCode(c tele.Context, code string) error {
	ub.logger.Infof("Пользователь %d отправил QR-код: %s", c.Sender().ID, code)

	// Получаем пользователя
	telegramID := fmt.Sprintf("%d", c.Sender().ID)

	// Получаем список пользователей через API
	usersData, err := ub.apiClient.Get("/users", nil)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send("Произошла ошибка при применении QR-кода. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при применении QR-кода. Пожалуйста, попробуйте позже.")
	}

	var user *models.User
	for _, u := range usersResponse.Users {
		if u.Telegramm == telegramID && !u.Deleted {
			user = u
			break
		}
	}

	if user == nil {
		// Пользователь не зарегистрирован
		return c.Send("Ты не зарегистрирован в системе. Используй /register для регистрации.")
	}

	// Проверяем, что код является валидным UUID
	if _, err := uuid.Parse(code); err != nil {
		ub.logger.Errorf("Ошибка парсинга UUID: %v", err)
		return c.Send("Неверный формат QR-кода.")
	}

	// Получаем информацию о коде через API
	codeData, err := ub.apiClient.Get("/codes/"+code, nil)
	if err != nil {
		ub.logger.Errorf("Ошибка получения информации о коде через API: %v", err)
		return c.Send("QR-код не найден.")
	}

	// Декодируем ответ
	var codeInfo models.Code
	if err := json.Unmarshal(codeData, &codeInfo); err != nil {
		ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении информации о коде. Пожалуйста, попробуйте позже.")
	}

	// Проверяем, активен ли код
	if !codeInfo.IsActive {
		return c.Send("Этот QR-код неактивен.")
	}

	// Проверяем, принадлежит ли пользователь к нужной группе
	if codeInfo.Group != "" && user.Group != codeInfo.Group {
		return c.Send(fmt.Sprintf("Этот QR-код предназначен только для группы %s.", codeInfo.Group))
	}

	// Применяем код через API
	applyData, err := ub.apiClient.Post("/codes/"+code+"/apply", map[string]interface{}{
		"user_id": user.Id,
	})
	if err != nil {
		ub.logger.Errorf("Ошибка применения кода через API: %v", err)

		// Возвращаем более информативные ошибки в зависимости от типа ошибки
		errorMsg := "Произошла ошибка при применении QR-кода. Пожалуйста, попробуйте позже."
		if strings.Contains(err.Error(), "code usage limit exceeded") {
			errorMsg = "Превышено общее количество использований QR-кода."
		} else if strings.Contains(err.Error(), "user code usage limit exceeded") {
			errorMsg = "Ты уже использовал этот QR-код максимальное количество раз."
		} else if strings.Contains(err.Error(), "code is not active") {
			errorMsg = "Этот QR-код неактивен."
		}

		return c.Send(errorMsg)
	}

	// Декодируем ответ
	var applyResponse struct {
		Success     bool `json:"success"`
		PointsAdded int  `json:"points_added"`
		TotalPoints int  `json:"total_points"`
	}
	if err := json.Unmarshal(applyData, &applyResponse); err != nil {
		ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при применении QR-кода. Пожалуйста, попробуйте позже.")
	}

	// Отправляем сообщение об успешном применении QR-кода с клавиатурой
	keyboard := ub.createMainKeyboard(true)
	return c.Send(fmt.Sprintf("QR-код успешно применен! Добавлено %d баллов. Теперь у тебя %d баллов.",
		applyResponse.PointsAdded, applyResponse.TotalPoints), keyboard)
}

// isUUID проверяет, является ли строка UUID
func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
