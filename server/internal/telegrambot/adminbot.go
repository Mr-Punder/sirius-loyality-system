package telegrambot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/google/uuid"
	tele "gopkg.in/telebot.v3"
)

// AdminInfo представляет информацию об администраторе
type AdminInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name,omitempty"`
}

// AdminsList представляет список администраторов
type AdminsList struct {
	Admins []AdminInfo `json:"admins"`
}

// BotState представляет состояние бота для конкретного пользователя
type BotState struct {
	State       string            // Текущее состояние
	Params      map[string]string // Параметры, собранные на предыдущих шагах
	LastMsgID   int               // ID последнего сообщения бота для возможного редактирования
	LastMsgText string            // Текст последнего сообщения
}

// AdminBot представляет бота для администраторов
type AdminBot struct {
	bot        *tele.Bot
	storage    storage.Storage
	logger     logger.Logger
	config     Config
	admins     AdminsList
	adminsPath string
	states     map[int64]*BotState // Состояния пользователей по их ID
	apiClient  *APIClient
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

	// Создаем API-клиент
	apiClient := NewAPIClient(config.ServerURL, config.APIToken, logger)

	// Создаем бота
	adminBot := &AdminBot{
		bot:        bot,
		storage:    storage,
		logger:     logger,
		config:     config,
		adminsPath: adminsPath,
		states:     make(map[int64]*BotState),
		apiClient:  apiClient,
	}

	// Загружаем список администраторов
	if err := adminBot.loadAdmins(); err != nil {
		logger.Errorf("Ошибка загрузки списка администраторов: %v", err)
		// Создаем пустой список
		adminBot.admins = AdminsList{
			Admins: []AdminInfo{},
		}

		// Если указан ID администратора в конфигурации, добавляем его
		if config.AdminUserID != 0 {
			adminBot.admins.Admins = append(adminBot.admins.Admins, AdminInfo{
				ID: config.AdminUserID,
			})
			logger.Infof("Добавлен администратор с ID %d из параметров запуска", config.AdminUserID)

			// Сохраняем список администраторов только если он был изменен
			if err := adminBot.saveAdmins(); err != nil {
				logger.Errorf("Ошибка сохранения списка администраторов: %v", err)
			}
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

	// Обработчики кнопок главного меню
	ab.bot.Handle("👥 Пользователи", ab.handleUsersButton)
	ab.bot.Handle("🔑 QR-коды", ab.handleCodesButton)
	ab.bot.Handle("👮 Администраторы", ab.handleAdminsButton)
	ab.bot.Handle("📣 Рассылка", ab.handleBroadcastButton)
	ab.bot.Handle("❓ Помощь", ab.handleHelp)

	// Обработчики кнопок меню пользователей
	ab.bot.Handle("👥 Все пользователи", ab.handleAllUsersButton)
	ab.bot.Handle("👨‍👩‍👧‍👦 По группам", ab.handleUsersByGroupButton)
	ab.bot.Handle("➕ Добавить баллы", ab.handleAddPointsButton)

	// Обработчики кнопок меню QR-кодов
	ab.bot.Handle("🔑 Список QR-кодов", ab.handleListCodesButton)
	ab.bot.Handle("➕ Сгенерировать QR-код", ab.handleGenerateCodeButton)

	// Обработчики кнопок меню администраторов
	ab.bot.Handle("👮 Список администраторов", ab.handleListAdmins)
	ab.bot.Handle("➕ Добавить администратора", ab.handleAddAdminButton)

	// Обработчик кнопки "Назад"
	ab.bot.Handle("🔙 Назад", ab.handleBackButton)

	// Обработчики кнопок для ввода параметров
	ab.bot.Handle("🚫 Без ограничений", ab.handleNoLimitsButton)
	ab.bot.Handle("🌐 Все группы", ab.handleAllGroupsButton)
	ab.bot.Handle("Н1", ab.handleGroupButton)
	ab.bot.Handle("Н2", ab.handleGroupButton)
	ab.bot.Handle("Н3", ab.handleGroupButton)
	ab.bot.Handle("Н4", ab.handleGroupButton)
	ab.bot.Handle("Н5", ab.handleGroupButton)
	ab.bot.Handle("Н6", ab.handleGroupButton)
	ab.bot.Handle("❌ Отмена", ab.handleCancelButton)

	// Обработчик текстовых сообщений
	ab.bot.Handle(tele.OnText, ab.handleText)

	// Запуск бота
	go ab.bot.Start()

	return nil
}

// handleText обрабатывает текстовые сообщения
func (ab *AdminBot) handleText(c tele.Context) error {
	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этому боту.")
	}

	// Получаем текст сообщения
	text := c.Text()
	userID := c.Sender().ID

	// Проверяем, есть ли у пользователя активное состояние
	state, ok := ab.states[userID]
	if !ok {
		// Если нет активного состояния, отправляем справку
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}

	// Обрабатываем сообщение в зависимости от текущего состояния
	switch state.State {
	case "broadcast_text":
		// Обрабатываем ввод текста для рассылки
		return ab.handleBroadcastText(c, state)

	case "broadcast_group":
		// Пользователь вводит группу для рассылки
		// Проверяем, соответствует ли группа формату
		if text == "🌐 Все группы" {
			// Рассылка всем пользователям
			return ab.broadcastMessage(c, state.Params["text"], "")
		} else if GroupRegex.MatchString(text) {
			// Нормализуем группу
			normalizedGroup, _ := NormalizeGroupName(text)
			// Рассылка пользователям выбранной группы
			return ab.broadcastMessage(c, state.Params["text"], normalizedGroup)
		} else {
			return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).")
		}

	case "generate_code_amount":
		// Пользователь вводит количество баллов для QR-кода
		_, err := strconv.Atoi(text)
		if err != nil {
			return c.Send("Неверный формат количества баллов. Пожалуйста, введите целое число.")
		}

		// Сохраняем количество баллов
		state.Params["amount"] = text

		// Переходим к следующему шагу - ввод ограничения на пользователя
		state.State = "generate_code_per_user"

		// Создаем клавиатуру с кнопкой "Без ограничений"
		keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
		btnNoLimits := keyboard.Text("🚫 Без ограничений")
		btnCancel := keyboard.Text("❌ Отмена")
		keyboard.Reply(
			keyboard.Row(btnNoLimits),
			keyboard.Row(btnCancel),
		)

		// Отправляем сообщение с запросом ограничения на пользователя
		return c.Send("Введите максимальное количество использований одним пользователем или нажмите кнопку 'Без ограничений':", keyboard)

	case "generate_code_per_user":
		// Пользователь вводит ограничение на пользователя
		_, err := strconv.Atoi(text)
		if err != nil {
			return c.Send("Неверный формат ограничения. Пожалуйста, введите целое число.")
		}

		// Сохраняем ограничение на пользователя
		state.Params["per_user"] = text

		// Переходим к следующему шагу - ввод общего ограничения
		state.State = "generate_code_total"

		// Создаем клавиатуру с кнопкой "Без ограничений"
		keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
		btnNoLimits := keyboard.Text("🚫 Без ограничений")
		btnCancel := keyboard.Text("❌ Отмена")
		keyboard.Reply(
			keyboard.Row(btnNoLimits),
			keyboard.Row(btnCancel),
		)

		// Отправляем сообщение с запросом общего ограничения
		return c.Send("Введите общее количество использований или нажмите кнопку 'Без ограничений':", keyboard)

	case "generate_code_total":
		// Пользователь вводит общее ограничение
		_, err := strconv.Atoi(text)
		if err != nil {
			return c.Send("Неверный формат ограничения. Пожалуйста, введите целое число.")
		}

		// Сохраняем общее ограничение
		state.Params["total"] = text

		// Переходим к следующему шагу - ввод группы
		state.State = "generate_code_group"

		// Создаем клавиатуру с кнопками групп
		keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
		btnN1 := keyboard.Text("Н1")
		btnN2 := keyboard.Text("Н2")
		btnN3 := keyboard.Text("Н3")
		btnN4 := keyboard.Text("Н4")
		btnN5 := keyboard.Text("Н5")
		btnN6 := keyboard.Text("Н6")
		btnNoLimits := keyboard.Text("🚫 Без ограничений")
		btnCancel := keyboard.Text("❌ Отмена")
		keyboard.Reply(
			keyboard.Row(btnN1, btnN2, btnN3),
			keyboard.Row(btnN4, btnN5, btnN6),
			keyboard.Row(btnNoLimits),
			keyboard.Row(btnCancel),
		)

		// Отправляем сообщение с запросом группы
		return c.Send("Выберите группу или нажмите кнопку 'Без ограничений':", keyboard)

	case "generate_code_group":
		// Пользователь вводит группу
		// Проверяем, соответствует ли группа формату
		if !GroupRegex.MatchString(text) {
			return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).")
		}

		// Нормализуем группу
		normalizedGroup, _ := NormalizeGroupName(text)

		// Сохраняем группу
		state.Params["group"] = normalizedGroup

		// Генерируем QR-код
		return ab.generateCodeFromParams(c, state.Params)

	case "add_admin_id":
		// Пользователь вводит ID администратора
		_, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return c.Send("Неверный формат ID пользователя. Пожалуйста, введите целое число.")
		}

		// Сохраняем ID администратора
		state.Params["admin_id"] = text

		// Переходим к следующему шагу - ввод имени администратора
		state.State = "add_admin_name"

		// Создаем клавиатуру с кнопкой "Без имени"
		keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
		btnNoName := keyboard.Text("🚫 Без имени")
		btnCancel := keyboard.Text("❌ Отмена")
		keyboard.Reply(
			keyboard.Row(btnNoName),
			keyboard.Row(btnCancel),
		)

		// Отправляем сообщение с запросом имени администратора
		return c.Send("Введите имя администратора (для заметок) или нажмите кнопку 'Без имени':", keyboard)

	case "add_admin_name":
		// Пользователь вводит имя администратора
		// Сохраняем имя администратора
		state.Params["admin_name"] = text

		// Добавляем администратора
		return ab.addAdminFromParams(c, state.Params)

	case "user_by_group":
		// Пользователь вводит группу для фильтрации пользователей
		// Проверяем, соответствует ли группа формату
		if !GroupRegex.MatchString(text) {
			return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).")
		}

		// Нормализуем группу
		normalizedGroup, _ := NormalizeGroupName(text)
		ab.logger.Infof("Пользователь %d выбрал группу %s для фильтрации", c.Sender().ID, normalizedGroup)

		// Сбрасываем состояние пользователя
		delete(ab.states, userID)

		// Получаем всех пользователей через API
		usersData, err := ab.apiClient.Get("/users", nil)
		if err != nil {
			ab.logger.Errorf("Ошибка получения пользователей через API: %v", err)
			return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
		}

		// Декодируем ответ
		var usersResponse struct {
			Total int            `json:"total"`
			Users []*models.User `json:"users"`
		}
		if err := json.Unmarshal(usersData, &usersResponse); err != nil {
			ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
			return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
		}

		// Фильтруем пользователей по группе и исключаем удаленных
		var filteredUsers []*models.User
		for _, user := range usersResponse.Users {
			if user.Group == normalizedGroup && !user.Deleted {
				filteredUsers = append(filteredUsers, user)
			}
		}

		if len(filteredUsers) == 0 {
			return c.Send(fmt.Sprintf("Пользователи в группе %s не найдены.", normalizedGroup))
		}

		// Формируем сообщение со списком пользователей
		var message strings.Builder
		message.WriteString(fmt.Sprintf("Список пользователей в группе %s:\n\n", normalizedGroup))

		for i, user := range filteredUsers {
			message.WriteString(fmt.Sprintf("%d. %s %s (Баллы: %d)\n",
				i+1, user.FirstName, user.LastName, user.Points))
		}

		// Отправляем сообщение
		return c.Send(message.String())

	case "add_points_user_id":
		// Пользователь вводит ID пользователя
		_, err := uuid.Parse(text)
		if err != nil {
			return c.Send("Неверный формат ID пользователя. Пожалуйста, введите UUID.")
		}

		// Сохраняем ID пользователя
		state.Params["user_id"] = text

		// Переходим к следующему шагу - ввод количества баллов
		state.State = "add_points_amount"

		// Отправляем сообщение с запросом количества баллов
		return c.Send("Введите количество баллов для добавления:")

	case "add_points_amount":
		// Пользователь вводит количество баллов
		_, err := strconv.Atoi(text)
		if err != nil {
			return c.Send("Неверный формат количества баллов. Пожалуйста, введите целое число.")
		}

		// Сохраняем количество баллов
		state.Params["points"] = text

		// Добавляем баллы пользователю
		c.Message().Payload = state.Params["user_id"] + " " + state.Params["points"]
		return ab.handleAddPoints(c)

	default:
		// Неизвестное состояние, сбрасываем его
		delete(ab.states, userID)
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}
}

// handleNoLimitsButton обрабатывает нажатие на кнопку "Без ограничений"
func (ab *AdminBot) handleNoLimitsButton(c tele.Context) error {
	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Получаем ID пользователя
	userID := c.Sender().ID

	// Проверяем, есть ли у пользователя активное состояние
	state, ok := ab.states[userID]
	if !ok {
		// Если нет активного состояния, отправляем справку
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}

	// Обрабатываем нажатие в зависимости от текущего состояния
	switch state.State {
	case "generate_code_per_user":
		// Пользователь выбрал "Без ограничений" для ограничения на пользователя
		// Устанавливаем значение 0 (без ограничений)
		state.Params["per_user"] = "0"

		// Переходим к следующему шагу - ввод общего ограничения
		state.State = "generate_code_total"

		// Создаем клавиатуру с кнопкой "Без ограничений"
		keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
		btnNoLimits := keyboard.Text("🚫 Без ограничений")
		btnCancel := keyboard.Text("❌ Отмена")
		keyboard.Reply(
			keyboard.Row(btnNoLimits),
			keyboard.Row(btnCancel),
		)

		// Отправляем сообщение с запросом общего ограничения
		return c.Send("Введите общее количество использований или нажмите кнопку 'Без ограничений':", keyboard)

	case "generate_code_total":
		// Пользователь выбрал "Без ограничений" для общего ограничения
		// Устанавливаем значение 0 (без ограничений)
		state.Params["total"] = "0"

		// Переходим к следующему шагу - ввод группы
		state.State = "generate_code_group"

		// Создаем клавиатуру с кнопками групп
		keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
		btnN1 := keyboard.Text("Н1")
		btnN2 := keyboard.Text("Н2")
		btnN3 := keyboard.Text("Н3")
		btnN4 := keyboard.Text("Н4")
		btnN5 := keyboard.Text("Н5")
		btnN6 := keyboard.Text("Н6")
		btnNoLimits := keyboard.Text("🚫 Без ограничений")
		btnCancel := keyboard.Text("❌ Отмена")
		keyboard.Reply(
			keyboard.Row(btnN1, btnN2, btnN3),
			keyboard.Row(btnN4, btnN5, btnN6),
			keyboard.Row(btnNoLimits),
			keyboard.Row(btnCancel),
		)

		// Отправляем сообщение с запросом группы
		return c.Send("Выберите группу или нажмите кнопку 'Без ограничений':", keyboard)

	case "generate_code_group":
		// Пользователь выбрал "Без ограничений" для группы
		// Устанавливаем пустую строку (без ограничений по группе)
		state.Params["group"] = ""

		// Генерируем QR-код
		return ab.generateCodeFromParams(c, state.Params)

	case "add_admin_name":
		// Пользователь выбрал "Без имени" для имени администратора
		// Устанавливаем пустую строку (без имени)
		state.Params["admin_name"] = ""

		// Добавляем администратора
		return ab.addAdminFromParams(c, state.Params)

	default:
		// Неизвестное состояние, сбрасываем его
		delete(ab.states, userID)
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}
}

// handleGroupButton обрабатывает нажатие на кнопку группы
func (ab *AdminBot) handleGroupButton(c tele.Context) error {
	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Получаем ID пользователя
	userID := c.Sender().ID

	// Проверяем, есть ли у пользователя активное состояние
	state, ok := ab.states[userID]
	if !ok {
		// Если нет активного состояния, отправляем справку
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}

	// Получаем выбранную группу
	group := c.Text()

	// Нормализуем группу
	normalizedGroup, valid := NormalizeGroupName(group)
	if !valid {
		return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).")
	}

	// Обрабатываем нажатие в зависимости от текущего состояния
	switch state.State {
	case "generate_code_group":
		// Сохраняем группу
		state.Params["group"] = normalizedGroup

		// Генерируем QR-код
		return ab.generateCodeFromParams(c, state.Params)

	case "broadcast_group":
		// Рассылка пользователям выбранной группы
		return ab.broadcastMessage(c, state.Params["text"], normalizedGroup)

	case "user_by_group":
		// Фильтрация пользователей по группе
		ab.logger.Infof("Пользователь %d выбрал группу %s для фильтрации", c.Sender().ID, normalizedGroup)

		// Сбрасываем состояние пользователя
		delete(ab.states, userID)

		// Получаем всех пользователей через API
		usersData, err := ab.apiClient.Get("/users", nil)
		if err != nil {
			ab.logger.Errorf("Ошибка получения пользователей через API: %v", err)
			return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
		}

		// Декодируем ответ
		var usersResponse struct {
			Total int            `json:"total"`
			Users []*models.User `json:"users"`
		}
		if err := json.Unmarshal(usersData, &usersResponse); err != nil {
			ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
			return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
		}

		// Фильтруем пользователей по группе и исключаем удаленных
		var filteredUsers []*models.User
		for _, user := range usersResponse.Users {
			if user.Group == normalizedGroup && !user.Deleted {
				filteredUsers = append(filteredUsers, user)
			}
		}

		if len(filteredUsers) == 0 {
			return c.Send(fmt.Sprintf("Пользователи в группе %s не найдены.", normalizedGroup))
		}

		// Формируем сообщение со списком пользователей
		var message strings.Builder
		message.WriteString(fmt.Sprintf("Список пользователей в группе %s:\n\n", normalizedGroup))

		for i, user := range filteredUsers {
			message.WriteString(fmt.Sprintf("%d. %s %s (Баллы: %d)\n",
				i+1, user.FirstName, user.LastName, user.Points))
		}

		// Отправляем сообщение
		return c.Send(message.String())

	default:
		// Неизвестное состояние, сбрасываем его
		delete(ab.states, userID)
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}
}

// handleCancelButton обрабатывает нажатие на кнопку "Отмена"
func (ab *AdminBot) handleCancelButton(c tele.Context) error {
	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Получаем ID пользователя
	userID := c.Sender().ID

	// Сбрасываем состояние пользователя
	delete(ab.states, userID)

	// Создаем основную клавиатуру
	keyboard := ab.createMainKeyboard()

	// Отправляем сообщение с клавиатурой
	return c.Send("Операция отменена. Выберите действие:", keyboard)
}

// generateCodeFromParams генерирует QR-код из параметров
func (ab *AdminBot) generateCodeFromParams(c tele.Context, params map[string]string) error {
	// Парсим параметры
	amount, err := strconv.Atoi(params["amount"])
	if err != nil {
		return c.Send("Неверный формат количества баллов.")
	}

	perUser, err := strconv.Atoi(params["per_user"])
	if err != nil {
		return c.Send("Неверный формат ограничения на пользователя.")
	}

	total, err := strconv.Atoi(params["total"])
	if err != nil {
		return c.Send("Неверный формат общего ограничения.")
	}

	group := params["group"]

	// Создаем запрос для API
	codeRequest := map[string]interface{}{
		"amount":   amount,
		"per_user": perUser,
		"total":    total,
		"group":    group,
	}

	// Отправляем запрос на создание кода через API
	codeData, err := ab.apiClient.Post("/codes", codeRequest)
	if err != nil {
		ab.logger.Errorf("Ошибка создания кода через API: %v", err)
		return c.Send("Произошла ошибка при генерации QR-кода. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var code models.Code
	if err := json.Unmarshal(codeData, &code); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
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

	// Сбрасываем состояние пользователя
	delete(ab.states, c.Sender().ID)

	// Создаем основную клавиатуру
	keyboard := ab.createMainKeyboard()

	// Генерируем QR-код в виде изображения
	qrCodeContent := code.Code.String()
	qrCodeImage, err := GenerateQRCode(qrCodeContent, 300)
	if err != nil {
		ab.logger.Errorf("Ошибка генерации QR-кода: %v", err)
		// Отправляем сообщение с информацией о коде без изображения
		return c.Send(message, keyboard)
	}

	// Создаем объект фото для отправки
	photo := &tele.Photo{
		File: tele.File{
			FileReader: bytes.NewReader(qrCodeImage),
		},
		Caption: message,
	}

	// Отправляем фото с QR-кодом и информацией
	_, err = c.Bot().Send(c.Recipient(), photo, keyboard)
	if err != nil {
		ab.logger.Errorf("Ошибка отправки QR-кода: %v", err)
		// В случае ошибки отправляем обычное сообщение
		return c.Send(message, keyboard)
	}

	return nil
}

// addAdminFromParams добавляет администратора из параметров
func (ab *AdminBot) addAdminFromParams(c tele.Context, params map[string]string) error {
	// Парсим ID администратора
	adminID, err := strconv.ParseInt(params["admin_id"], 10, 64)
	if err != nil {
		return c.Send("Неверный формат ID пользователя.")
	}

	// Получаем имя администратора
	adminName := params["admin_name"]

	// Проверяем, есть ли уже такой администратор
	for _, admin := range ab.admins.Admins {
		if admin.ID == adminID {
			return c.Send(fmt.Sprintf("Пользователь с ID %d уже является администратором.", adminID))
		}
	}

	// Добавляем нового администратора
	ab.admins.Admins = append(ab.admins.Admins, AdminInfo{
		ID:   adminID,
		Name: adminName,
	})

	// Сохраняем список администраторов
	if err := ab.saveAdmins(); err != nil {
		ab.logger.Errorf("Ошибка сохранения списка администраторов: %v", err)
		return c.Send("Произошла ошибка при сохранении списка администраторов.")
	}

	// Сбрасываем состояние пользователя
	delete(ab.states, c.Sender().ID)

	// Создаем основную клавиатуру
	keyboard := ab.createMainKeyboard()

	// Отправляем сообщение об успешном добавлении
	return c.Send(fmt.Sprintf("Пользователь с ID %d успешно добавлен в список администраторов.", adminID), keyboard)
}

// Используем NormalizeGroupName из utils.go

// handleUsersButton обрабатывает нажатие на кнопку "Пользователи"
func (ab *AdminBot) handleUsersButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Пользователи'", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Создаем клавиатуру для работы с пользователями
	keyboard := ab.createUsersKeyboard()

	// Отправляем сообщение с клавиатурой
	return c.Send("Выберите действие для работы с пользователями:", keyboard)
}

// handleCodesButton обрабатывает нажатие на кнопку "QR-коды"
func (ab *AdminBot) handleCodesButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'QR-коды'", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Создаем клавиатуру для работы с QR-кодами
	keyboard := ab.createCodesKeyboard()

	// Отправляем сообщение с клавиатурой
	return c.Send("Выберите действие для работы с QR-кодами:", keyboard)
}

// handleAdminsButton обрабатывает нажатие на кнопку "Администраторы"
func (ab *AdminBot) handleAdminsButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Администраторы'", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Создаем клавиатуру для работы с администраторами
	keyboard := ab.createAdminsKeyboard()

	// Отправляем сообщение с клавиатурой
	return c.Send("Выберите действие для работы с администраторами:", keyboard)
}

// handleAllUsersButton обрабатывает нажатие на кнопку "Все пользователи"
func (ab *AdminBot) handleAllUsersButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Все пользователи'", c.Sender().ID)

	// Вызываем обработчик команды /users без параметров
	return ab.handleUsers(c)
}

// handleUsersByGroupButton обрабатывает нажатие на кнопку "По группам"
func (ab *AdminBot) handleUsersByGroupButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'По группам'", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Инициализируем состояние для выбора группы
	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "user_by_group",
		Params: make(map[string]string),
	}

	// Создаем клавиатуру с кнопками групп
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnN1 := keyboard.Text("Н1")
	btnN2 := keyboard.Text("Н2")
	btnN3 := keyboard.Text("Н3")
	btnN4 := keyboard.Text("Н4")
	btnN5 := keyboard.Text("Н5")
	btnN6 := keyboard.Text("Н6")
	btnCancel := keyboard.Text("❌ Отмена")
	keyboard.Reply(
		keyboard.Row(btnN1, btnN2, btnN3),
		keyboard.Row(btnN4, btnN5, btnN6),
		keyboard.Row(btnCancel),
	)

	// Отправляем сообщение с запросом группы
	return c.Send("Выберите группу для фильтрации пользователей:", keyboard)
}

// handleAddPointsButton обрабатывает нажатие на кнопку "Добавить баллы"
func (ab *AdminBot) handleAddPointsButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Добавить баллы'", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Инициализируем состояние для добавления баллов
	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "add_points_user_id",
		Params: make(map[string]string),
	}

	// Отправляем сообщение с запросом ID пользователя
	return c.Send("Введите ID пользователя (UUID):")
}

// handleGenerateCodeButton обрабатывает нажатие на кнопку "Сгенерировать QR-код"
func (ab *AdminBot) handleGenerateCodeButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Сгенерировать QR-код'", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Инициализируем состояние для генерации QR-кода
	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "generate_code_amount",
		Params: make(map[string]string),
	}

	// Отправляем сообщение с запросом количества баллов
	return c.Send("Введите количество баллов для QR-кода:")
}

// handleAddAdminButton обрабатывает нажатие на кнопку "Добавить администратора"
func (ab *AdminBot) handleAddAdminButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Добавить администратора'", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Инициализируем состояние для добавления администратора
	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "add_admin_id",
		Params: make(map[string]string),
	}

	// Отправляем сообщение с запросом ID администратора
	return c.Send("Введите ID пользователя:")
}

// handleBackButton обрабатывает нажатие на кнопку "Назад"
func (ab *AdminBot) handleBackButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Назад'", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Создаем основную клавиатуру
	keyboard := ab.createMainKeyboard()

	// Отправляем сообщение с клавиатурой
	return c.Send("Главное меню:", keyboard)
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
		// Если файл не существует, но директория существует, создаем пустой файл
		dir := filepath.Dir(ab.adminsPath)
		if _, err := os.Stat(dir); err == nil {
			// Создаем пустой список администраторов
			emptyAdmins := AdminsList{
				Admins: []AdminInfo{},
			}
			data, err := json.MarshalIndent(emptyAdmins, "", "    ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(ab.adminsPath, data, 0644); err != nil {
				return err
			}
			ab.admins = emptyAdmins
			return nil
		}
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
	// Проверяем через API, является ли пользователь администратором
	data, err := ab.apiClient.Get(fmt.Sprintf("/admins/check/%d", userID), nil)
	if err != nil {
		ab.logger.Errorf("Ошибка проверки администратора через API: %v", err)

		// Если API недоступен, используем локальный список администраторов
		for _, admin := range ab.admins.Admins {
			if userID == admin.ID {
				return true
			}
		}

		return false
	}

	// Декодируем ответ
	var response struct {
		IsAdmin bool `json:"is_admin"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)

		// Если не удалось декодировать ответ, используем локальный список администраторов
		for _, admin := range ab.admins.Admins {
			if userID == admin.ID {
				return true
			}
		}

		return false
	}

	return response.IsAdmin
}

// createMainKeyboard создает основную клавиатуру с кнопками
func (ab *AdminBot) createMainKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	// Создаем кнопки
	btnUsers := keyboard.Text("👥 Пользователи")
	btnCodes := keyboard.Text("🔑 QR-коды")
	btnAdmins := keyboard.Text("👮 Администраторы")
	btnBroadcast := keyboard.Text("📣 Рассылка")
	btnHelp := keyboard.Text("❓ Помощь")

	// Добавляем кнопки на клавиатуру
	keyboard.Reply(
		keyboard.Row(btnUsers, btnCodes),
		keyboard.Row(btnAdmins, btnBroadcast),
		keyboard.Row(btnHelp),
	)

	return keyboard
}

// createUsersKeyboard создает клавиатуру для работы с пользователями
func (ab *AdminBot) createUsersKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	// Создаем кнопки
	btnAllUsers := keyboard.Text("👥 Все пользователи")
	btnUsersByGroup := keyboard.Text("👨‍👩‍👧‍👦 По группам")
	btnAddPoints := keyboard.Text("➕ Добавить баллы")
	btnBack := keyboard.Text("🔙 Назад")

	// Добавляем кнопки на клавиатуру
	keyboard.Reply(
		keyboard.Row(btnAllUsers, btnUsersByGroup),
		keyboard.Row(btnAddPoints),
		keyboard.Row(btnBack),
	)

	return keyboard
}

// createCodesKeyboard создает клавиатуру для работы с QR-кодами
func (ab *AdminBot) createCodesKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	// Создаем кнопки
	btnListCodes := keyboard.Text("🔑 Список QR-кодов")
	btnGenerateCode := keyboard.Text("➕ Сгенерировать QR-код")
	btnBack := keyboard.Text("🔙 Назад")

	// Добавляем кнопки на клавиатуру
	keyboard.Reply(
		keyboard.Row(btnListCodes),
		keyboard.Row(btnGenerateCode),
		keyboard.Row(btnBack),
	)

	return keyboard
}

// createAdminsKeyboard создает клавиатуру для работы с администраторами
func (ab *AdminBot) createAdminsKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	// Создаем кнопки
	btnListAdmins := keyboard.Text("👮 Список администраторов")
	btnAddAdmin := keyboard.Text("➕ Добавить администратора")
	btnBack := keyboard.Text("🔙 Назад")

	// Добавляем кнопки на клавиатуру
	keyboard.Reply(
		keyboard.Row(btnListAdmins),
		keyboard.Row(btnAddAdmin),
		keyboard.Row(btnBack),
	)

	return keyboard
}

// handleStart обрабатывает команду /start
func (ab *AdminBot) handleStart(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запустил бота", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этому боту.")
	}

	// Создаем клавиатуру
	keyboard := ab.createMainKeyboard()

	// Отправляем приветственное сообщение с клавиатурой
	return c.Send("Привет, администратор! Я бот для управления системой лояльности. Выберите действие на клавиатуре или используйте /help для просмотра доступных команд.", keyboard)
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
		// Нормализуем группу, если она указана
		normalizedGroup, valid := NormalizeGroupName(args[0])
		if !valid {
			return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).")
		}
		group = normalizedGroup
		ab.logger.Infof("Фильтрация пользователей по группе: %s", group)
	}

	// Получаем всех пользователей через API
	usersData, err := ab.apiClient.Get("/users", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
	}

	// Фильтруем пользователей по группе и исключаем удаленных
	var filteredUsers []*models.User
	for _, user := range usersResponse.Users {
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
		message.WriteString(fmt.Sprintf("%d. %s %s (Группа: %s, Баллы: %d)\n",
			i+1, user.FirstName, user.LastName, user.Group, user.Points))
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

	// Получаем пользователя через API
	userData, err := ab.apiClient.Get("/users/"+userID.String(), nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователя через API: %v", err)
		return c.Send("Пользователь не найден.")
	}

	// Декодируем ответ
	var user models.User
	if err := json.Unmarshal(userData, &user); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении информации о пользователе. Пожалуйста, попробуйте позже.")
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

	// Получаем пользователя через API
	userData, err := ab.apiClient.Get("/users/"+userID.String(), nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователя через API: %v", err)
		return c.Send("Пользователь не найден.")
	}

	// Декодируем ответ
	var user models.User
	if err := json.Unmarshal(userData, &user); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении информации о пользователе. Пожалуйста, попробуйте позже.")
	}

	if user.Deleted {
		return c.Send("Пользователь удален.")
	}

	// Создаем запрос для добавления баллов через API
	transactionRequest := map[string]interface{}{
		"user_id": userID.String(),
		"points":  points,
	}

	// Отправляем запрос на добавление баллов через API
	transactionData, err := ab.apiClient.Post("/transactions", transactionRequest)
	if err != nil {
		ab.logger.Errorf("Ошибка добавления баллов через API: %v", err)
		return c.Send("Произошла ошибка при добавлении баллов. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var transactionResponse struct {
		Success     bool `json:"success"`
		TotalPoints int  `json:"total_points"`
	}
	if err := json.Unmarshal(transactionData, &transactionResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при добавлении баллов. Пожалуйста, попробуйте позже.")
	}

	// Отправляем сообщение об успешном добавлении баллов
	return c.Send(fmt.Sprintf("Баллы успешно добавлены!\nПользователь: %s %s\nДобавлено: %d\nВсего баллов: %d",
		user.FirstName, user.LastName, points, transactionResponse.TotalPoints))
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

	// Создаем запрос для API
	codeRequest := map[string]interface{}{
		"amount":   amount,
		"per_user": perUser,
		"total":    total,
		"group":    group,
	}

	// Отправляем запрос на создание кода через API
	codeData, err := ab.apiClient.Post("/codes", codeRequest)
	if err != nil {
		ab.logger.Errorf("Ошибка создания кода через API: %v", err)
		return c.Send("Произошла ошибка при генерации QR-кода. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var code models.Code
	if err := json.Unmarshal(codeData, &code); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
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

	// Получаем имя администратора (если указано)
	var adminName string
	if len(args) > 1 {
		adminName = strings.Join(args[1:], " ")
	}

	// Проверяем, есть ли уже такой администратор
	for _, admin := range ab.admins.Admins {
		if admin.ID == adminID {
			return c.Send(fmt.Sprintf("Пользователь с ID %d уже является администратором.", adminID))
		}
	}

	// Добавляем нового администратора
	ab.admins.Admins = append(ab.admins.Admins, AdminInfo{
		ID:   adminID,
		Name: adminName,
	})

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

	for i, admin := range ab.admins.Admins {
		if admin.Name != "" {
			message.WriteString(fmt.Sprintf("%d. %d (%s)\n", i+1, admin.ID, admin.Name))
		} else {
			message.WriteString(fmt.Sprintf("%d. %d\n", i+1, admin.ID))
		}
	}

	// Отправляем сообщение
	return c.Send(message.String())
}

// handleListCodesButton обрабатывает нажатие на кнопку "Список QR-кодов"
func (ab *AdminBot) handleListCodesButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Список QR-кодов'", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Получаем все коды через API
	codesData, err := ab.apiClient.Get("/codes", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения кодов через API: %v", err)
		return c.Send("Произошла ошибка при получении списка QR-кодов. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var codesResponse struct {
		Total int            `json:"total"`
		Codes []*models.Code `json:"codes"`
	}
	if err := json.Unmarshal(codesData, &codesResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении списка QR-кодов. Пожалуйста, попробуйте позже.")
	}

	// Фильтруем только активные коды
	var activeCodes []*models.Code
	for _, code := range codesResponse.Codes {
		if code.IsActive {
			activeCodes = append(activeCodes, code)
		}
	}

	if len(activeCodes) == 0 {
		return c.Send("Активные QR-коды не найдены.")
	}

	// Формируем сообщение со списком кодов
	var message strings.Builder
	message.WriteString("Список активных QR-кодов:\n\n")

	for i, code := range activeCodes {
		groupInfo := "без ограничений"
		if code.Group != "" {
			groupInfo = code.Group
		}

		perUserInfo := "без ограничений"
		if code.PerUser > 0 {
			perUserInfo = fmt.Sprintf("%d", code.PerUser)
		}

		totalInfo := "без ограничений"
		if code.Total > 0 {
			totalInfo = fmt.Sprintf("%d", code.Total)
		}

		message.WriteString(fmt.Sprintf("%d. Код: %s\n   Баллы: %d\n   Использовано: %d\n   Лимит на пользователя: %s\n   Общий лимит: %s\n   Группа: %s\n\n",
			i+1, code.Code, code.Amount, code.AppliedCount, perUserInfo, totalInfo, groupInfo))
	}

	// Отправляем сообщение
	return c.Send(message.String())
}

// handleBroadcastButton обрабатывает нажатие на кнопку "Рассылка"
func (ab *AdminBot) handleBroadcastButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Рассылка'", c.Sender().ID)

	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Создаем клавиатуру для рассылки
	keyboard := ab.createBroadcastKeyboard()

	// Инициализируем состояние для рассылки
	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "broadcast_text",
		Params: make(map[string]string),
	}

	// Отправляем сообщение с инструкцией
	return c.Send("Введите текст сообщения для рассылки всем пользователям:", keyboard)
}

// createBroadcastKeyboard создает клавиатуру для рассылки
func (ab *AdminBot) createBroadcastKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	// Создаем кнопки
	btnCancel := keyboard.Text("❌ Отмена")
	btnBack := keyboard.Text("🔙 Назад")

	// Добавляем кнопки на клавиатуру
	keyboard.Reply(
		keyboard.Row(btnCancel),
		keyboard.Row(btnBack),
	)

	return keyboard
}

// handleText обрабатывает текстовые сообщения (дополнение для рассылки)
func (ab *AdminBot) handleBroadcastText(c tele.Context, state *BotState) error {
	// Получаем текст сообщения
	text := c.Text()

	// Сохраняем текст сообщения
	state.Params["text"] = text

	// Переходим к следующему шагу - выбор группы
	state.State = "broadcast_group"

	// Создаем клавиатуру с кнопками групп
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnN1 := keyboard.Text("Н1")
	btnN2 := keyboard.Text("Н2")
	btnN3 := keyboard.Text("Н3")
	btnN4 := keyboard.Text("Н4")
	btnN5 := keyboard.Text("Н5")
	btnN6 := keyboard.Text("Н6")
	btnAllGroups := keyboard.Text("🌐 Все группы")
	btnCancel := keyboard.Text("❌ Отмена")
	keyboard.Reply(
		keyboard.Row(btnN1, btnN2, btnN3),
		keyboard.Row(btnN4, btnN5, btnN6),
		keyboard.Row(btnAllGroups),
		keyboard.Row(btnCancel),
	)

	// Отправляем сообщение с запросом группы
	return c.Send("Выберите группу для рассылки или нажмите кнопку 'Все группы':", keyboard)
}

// broadcastMessage отправляет сообщение всем пользователям
func (ab *AdminBot) broadcastMessage(c tele.Context, text string, group string) error {
	// Получаем всех пользователей через API
	usersData, err := ab.apiClient.Get("/users", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
	}

	// Декодируем ответ
	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
	}

	// Фильтруем пользователей по группе и исключаем удаленных
	var filteredUsers []*models.User
	for _, user := range usersResponse.Users {
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

	// Отправляем сообщение о начале рассылки
	statusMsg, err := c.Bot().Send(c.Recipient(), fmt.Sprintf("Начинаем рассылку для %d пользователей...", len(filteredUsers)))
	if err != nil {
		ab.logger.Errorf("Ошибка отправки статусного сообщения: %v", err)
	}

	// Счетчики для статистики
	successCount := 0
	errorCount := 0

	// Отправляем сообщение каждому пользователю
	for i, user := range filteredUsers {
		// Проверяем, что у пользователя есть Telegram ID
		if user.Telegramm == "" {
			ab.logger.Errorf("Пользователь %s не имеет Telegram ID", user.Id)
			errorCount++
			continue
		}

		// Логируем ID пользователя для отладки
		ab.logger.Infof("Отправка сообщения пользователю %s с Telegram ID: %s", user.Id, user.Telegramm)

		// Парсим Telegram ID из строки, удаляя все нецифровые символы
		telegramIDStr := strings.TrimSpace(user.Telegramm)
		// Удаляем все нецифровые символы
		telegramIDStr = strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, telegramIDStr)

		if telegramIDStr == "" {
			ab.logger.Errorf("Пустой Telegram ID после очистки для пользователя %s", user.Id)
			errorCount++
			continue
		}

		telegramID, parseErr := strconv.ParseInt(telegramIDStr, 10, 64)
		if parseErr != nil {
			ab.logger.Errorf("Ошибка парсинга Telegram ID пользователя %s (%s): %v", user.Id, telegramIDStr, parseErr)
			errorCount++
			continue
		}

		// Создаем получателя
		recipient := &tele.User{
			ID: telegramID,
		}

		// Отправляем сообщение
		_, err := c.Bot().Send(recipient, text)
		if err != nil {
			ab.logger.Errorf("Ошибка отправки сообщения пользователю %s (Telegram ID: %d): %v", user.Id, telegramID, err)
			errorCount++
		} else {
			ab.logger.Infof("Успешно отправлено сообщение пользователю %s (Telegram ID: %d)", user.Id, telegramID)
			successCount++
		}

		// Обновляем статус каждые 10 пользователей
		if i%10 == 0 && i > 0 {
			c.Bot().Edit(statusMsg, fmt.Sprintf("Рассылка в процессе... %d/%d", i, len(filteredUsers)))
		}

		// Небольшая задержка, чтобы не перегружать API Telegram
		time.Sleep(100 * time.Millisecond)
	}

	// Отправляем сообщение о завершении рассылки
	return c.Send(fmt.Sprintf("Рассылка завершена!\nУспешно отправлено: %d\nОшибок: %d", successCount, errorCount))
}

// handleAllGroupsButton обрабатывает нажатие на кнопку "Все группы"
func (ab *AdminBot) handleAllGroupsButton(c tele.Context) error {
	// Проверяем, является ли пользователь администратором
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	// Получаем ID пользователя
	userID := c.Sender().ID

	// Проверяем, есть ли у пользователя активное состояние
	state, ok := ab.states[userID]
	if !ok || state.State != "broadcast_group" {
		// Если нет активного состояния или состояние не соответствует, отправляем справку
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}

	// Рассылка всем пользователям
	return ab.broadcastMessage(c, state.Params["text"], "")
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
