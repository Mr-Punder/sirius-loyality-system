package telegrambot

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
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
	State       string
	Params      map[string]string
	LastMsgID   int
	LastMsgText string
}

// AdminBot представляет бота для администраторов
type AdminBot struct {
	bot       *tele.Bot
	logger    logger.Logger
	config    Config
	states    map[int64]*BotState
	apiClient *APIClient
}

// NewAdminBot создает нового бота для администраторов
func NewAdminBot(config Config, storage storage.Storage, logger logger.Logger) (*AdminBot, error) {
	pref := tele.Settings{
		Token:  config.Token,
		Poller: &tele.LongPoller{Timeout: 10},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания бота: %w", err)
	}

	apiClient := NewAPIClient(config.ServerURL, config.APIToken, logger)

	adminBot := &AdminBot{
		bot:       bot,
		logger:    logger,
		config:    config,
		states:    make(map[int64]*BotState),
		apiClient: apiClient,
	}

	// Если указан начальный админ, добавляем его в БД
	if config.AdminUserID != 0 {
		adminBot.addAdminViaAPI(config.AdminUserID, "Initial Admin")
		logger.Infof("Добавлен начальный администратор с ID %d", config.AdminUserID)
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

	// Обработчик команды /puzzles
	ab.bot.Handle("/puzzles", ab.handlePuzzles)

	// Обработчик команды /pieces
	ab.bot.Handle("/pieces", ab.handlePiecesCommand)

	// Обработчик команды /lottery
	ab.bot.Handle("/lottery", ab.handleLottery)

	// Обработчик команды /complete для засчитывания пазла
	ab.bot.Handle("/complete", ab.handleCompletePuzzle)

	// Обработчик команды /addadmin
	ab.bot.Handle("/addadmin", ab.handleAddAdmin)

	// Обработчик команды /listadmins
	ab.bot.Handle("/listadmins", ab.handleListAdmins)

	// Обработчик команды /help
	ab.bot.Handle("/help", ab.handleHelp)

	// Обработчики кнопок главного меню
	ab.bot.Handle("👥 Пользователи", ab.handleUsersButton)
	ab.bot.Handle("🧩 Пазлы", ab.handlePuzzlesButton)
	ab.bot.Handle("👮 Администраторы", ab.handleAdminsButton)
	ab.bot.Handle("📣 Рассылка", ab.handleBroadcastButton)
	ab.bot.Handle("🎲 Розыгрыш", ab.handleLotteryButton)
	ab.bot.Handle("❓ Помощь", ab.handleHelp)

	// Обработчики кнопок меню пользователей
	ab.bot.Handle("👥 Все пользователи", ab.handleAllUsersButton)
	ab.bot.Handle("👨‍👩‍👧‍👦 По группам", ab.handleUsersByGroupButton)

	// Обработчики кнопок меню пазлов
	ab.bot.Handle("🧩 Список пазлов", ab.handleListPuzzlesButton)
	ab.bot.Handle("📋 Список деталей", ab.handleListPiecesButton)
	ab.bot.Handle("✅ Засчитать пазл", ab.handleCompletePuzzleButton)

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
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этому боту.")
	}

	text := c.Text()
	userID := c.Sender().ID

	state, ok := ab.states[userID]
	if !ok {
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}

	switch state.State {
	case "broadcast_text":
		return ab.handleBroadcastText(c, state)

	case "broadcast_group":
		if text == "🌐 Все группы" {
			return ab.broadcastMessage(c, state.Params["text"], "")
		} else if GroupRegex.MatchString(text) {
			normalizedGroup, _ := NormalizeGroupName(text)
			return ab.broadcastMessage(c, state.Params["text"], normalizedGroup)
		} else {
			return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).")
		}

	case "add_admin_id":
		_, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return c.Send("Неверный формат ID пользователя. Пожалуйста, введите целое число.")
		}

		state.Params["admin_id"] = text
		state.State = "add_admin_name"

		keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
		btnNoName := keyboard.Text("🚫 Без имени")
		btnCancel := keyboard.Text("❌ Отмена")
		keyboard.Reply(
			keyboard.Row(btnNoName),
			keyboard.Row(btnCancel),
		)

		return c.Send("Введите имя администратора (для заметок) или нажмите кнопку 'Без имени':", keyboard)

	case "add_admin_name":
		state.Params["admin_name"] = text
		return ab.addAdminFromParams(c, state.Params)

	case "user_by_group":
		if !GroupRegex.MatchString(text) {
			return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).")
		}

		normalizedGroup, _ := NormalizeGroupName(text)
		ab.logger.Infof("Пользователь %d выбрал группу %s для фильтрации", c.Sender().ID, normalizedGroup)

		delete(ab.states, userID)

		return ab.showUsersByGroup(c, normalizedGroup)

	case "complete_puzzle_id":
		puzzleId, err := strconv.Atoi(text)
		if err != nil || puzzleId < 1 || puzzleId > 30 {
			return c.Send("Неверный номер пазла. Введите число от 1 до 30.")
		}

		delete(ab.states, userID)
		return ab.completePuzzleAndNotify(c, puzzleId)

	default:
		delete(ab.states, userID)
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}
}

// handleNoLimitsButton обрабатывает нажатие на кнопку "Без ограничений"
func (ab *AdminBot) handleNoLimitsButton(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	userID := c.Sender().ID

	state, ok := ab.states[userID]
	if !ok {
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}

	switch state.State {
	case "add_admin_name":
		state.Params["admin_name"] = ""
		return ab.addAdminFromParams(c, state.Params)

	default:
		delete(ab.states, userID)
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}
}

// handleGroupButton обрабатывает нажатие на кнопку группы
func (ab *AdminBot) handleGroupButton(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	userID := c.Sender().ID

	state, ok := ab.states[userID]
	if !ok {
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}

	group := c.Text()
	normalizedGroup, valid := NormalizeGroupName(group)
	if !valid {
		return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).")
	}

	switch state.State {
	case "broadcast_group":
		return ab.broadcastMessage(c, state.Params["text"], normalizedGroup)

	case "user_by_group":
		ab.logger.Infof("Пользователь %d выбрал группу %s для фильтрации", c.Sender().ID, normalizedGroup)
		delete(ab.states, userID)
		return ab.showUsersByGroup(c, normalizedGroup)

	default:
		delete(ab.states, userID)
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}
}

// handleCancelButton обрабатывает нажатие на кнопку "Отмена"
func (ab *AdminBot) handleCancelButton(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	userID := c.Sender().ID
	delete(ab.states, userID)

	keyboard := ab.createMainKeyboard()
	return c.Send("Операция отменена. Выберите действие:", keyboard)
}

// addAdminFromParams добавляет администратора из параметров
func (ab *AdminBot) addAdminFromParams(c tele.Context, params map[string]string) error {
	adminID, err := strconv.ParseInt(params["admin_id"], 10, 64)
	if err != nil {
		return c.Send("Неверный формат ID пользователя.")
	}

	adminName := params["admin_name"]

	// Проверяем, не является ли уже администратором
	if ab.isAdmin(adminID) {
		return c.Send(fmt.Sprintf("Пользователь с ID %d уже является администратором.", adminID))
	}

	// Добавляем через API
	err = ab.addAdminViaAPI(adminID, adminName)
	if err != nil {
		ab.logger.Errorf("Ошибка добавления администратора через API: %v", err)
		return c.Send(fmt.Sprintf("Ошибка добавления администратора: %v", err))
	}

	delete(ab.states, c.Sender().ID)
	keyboard := ab.createMainKeyboard()

	return c.Send(fmt.Sprintf("Пользователь с ID %d успешно добавлен в список администраторов.", adminID), keyboard)
}

// handleUsersButton обрабатывает нажатие на кнопку "Пользователи"
func (ab *AdminBot) handleUsersButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Пользователи'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	keyboard := ab.createUsersKeyboard()
	return c.Send("Выберите действие для работы с пользователями:", keyboard)
}

// handlePuzzlesButton обрабатывает нажатие на кнопку "Пазлы"
func (ab *AdminBot) handlePuzzlesButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Пазлы'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	keyboard := ab.createPuzzlesKeyboard()
	return c.Send("Выберите действие для работы с пазлами:", keyboard)
}

// handleAdminsButton обрабатывает нажатие на кнопку "Администраторы"
func (ab *AdminBot) handleAdminsButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Администраторы'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	keyboard := ab.createAdminsKeyboard()
	return c.Send("Выберите действие для работы с администраторами:", keyboard)
}

// handleAllUsersButton обрабатывает нажатие на кнопку "Все пользователи"
func (ab *AdminBot) handleAllUsersButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Все пользователи'", c.Sender().ID)
	return ab.handleUsers(c)
}

// handleUsersByGroupButton обрабатывает нажатие на кнопку "По группам"
func (ab *AdminBot) handleUsersByGroupButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'По группам'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "user_by_group",
		Params: make(map[string]string),
	}

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

	return c.Send("Выберите группу для фильтрации пользователей:", keyboard)
}

// handleListPuzzlesButton обрабатывает нажатие на кнопку "Список пазлов"
func (ab *AdminBot) handleListPuzzlesButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Список пазлов'", c.Sender().ID)
	return ab.handlePuzzles(c)
}

// handleListPiecesButton обрабатывает нажатие на кнопку "Список деталей"
func (ab *AdminBot) handleListPiecesButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Список деталей'", c.Sender().ID)
	return ab.handlePiecesCommand(c)
}

// handleCompletePuzzleButton обрабатывает нажатие на кнопку "Засчитать пазл"
func (ab *AdminBot) handleCompletePuzzleButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Засчитать пазл'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "complete_puzzle_id",
		Params: make(map[string]string),
	}

	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnCancel := keyboard.Text("❌ Отмена")
	keyboard.Reply(keyboard.Row(btnCancel))

	return c.Send("Введите номер пазла для засчитывания (1-30):", keyboard)
}

// handleAddAdminButton обрабатывает нажатие на кнопку "Добавить администратора"
func (ab *AdminBot) handleAddAdminButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Добавить администратора'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "add_admin_id",
		Params: make(map[string]string),
	}

	return c.Send("Введите ID пользователя:")
}

// handleLotteryButton обрабатывает нажатие на кнопку "Розыгрыш"
func (ab *AdminBot) handleLotteryButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Розыгрыш'", c.Sender().ID)
	return ab.handleLottery(c)
}

// handleBackButton обрабатывает нажатие на кнопку "Назад"
func (ab *AdminBot) handleBackButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Назад'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	keyboard := ab.createMainKeyboard()
	return c.Send("Главное меню:", keyboard)
}

// Stop останавливает бота
func (ab *AdminBot) Stop() error {
	ab.logger.Info("Остановка административного бота")
	ab.bot.Stop()
	return nil
}

// isAdmin проверяет, является ли пользователь администратором
func (ab *AdminBot) isAdmin(userID int64) bool {
	data, err := ab.apiClient.Get(fmt.Sprintf("/admins/check/%d", userID), nil)
	if err != nil {
		ab.logger.Errorf("Ошибка проверки администратора через API: %v", err)
		return false
	}

	var response struct {
		IsAdmin bool `json:"is_admin"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return false
	}

	return response.IsAdmin
}

// addAdminViaAPI добавляет администратора через API
func (ab *AdminBot) addAdminViaAPI(adminID int64, name string) error {
	reqData := map[string]interface{}{
		"id":   adminID,
		"name": name,
	}
	_, err := ab.apiClient.Post("/admins", reqData)
	return err
}

// getAdminsViaAPI получает список администраторов через API
func (ab *AdminBot) getAdminsViaAPI() ([]AdminInfo, error) {
	data, err := ab.apiClient.Get("/admins", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Admins []AdminInfo `json:"admins"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Admins, nil
}

// deleteAdminViaAPI удаляет администратора через API
func (ab *AdminBot) deleteAdminViaAPI(adminID int64) error {
	_, err := ab.apiClient.Delete(fmt.Sprintf("/admins/%d", adminID))
	return err
}

// createMainKeyboard создает основную клавиатуру с кнопками
func (ab *AdminBot) createMainKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	btnUsers := keyboard.Text("👥 Пользователи")
	btnPuzzles := keyboard.Text("🧩 Пазлы")
	btnAdmins := keyboard.Text("👮 Администраторы")
	btnBroadcast := keyboard.Text("📣 Рассылка")
	btnLottery := keyboard.Text("🎲 Розыгрыш")
	btnHelp := keyboard.Text("❓ Помощь")

	keyboard.Reply(
		keyboard.Row(btnUsers, btnPuzzles),
		keyboard.Row(btnAdmins, btnLottery),
		keyboard.Row(btnBroadcast, btnHelp),
	)

	return keyboard
}

// createUsersKeyboard создает клавиатуру для работы с пользователями
func (ab *AdminBot) createUsersKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	btnAllUsers := keyboard.Text("👥 Все пользователи")
	btnUsersByGroup := keyboard.Text("👨‍👩‍👧‍👦 По группам")
	btnBack := keyboard.Text("🔙 Назад")

	keyboard.Reply(
		keyboard.Row(btnAllUsers, btnUsersByGroup),
		keyboard.Row(btnBack),
	)

	return keyboard
}

// createPuzzlesKeyboard создает клавиатуру для работы с пазлами
func (ab *AdminBot) createPuzzlesKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	btnListPuzzles := keyboard.Text("🧩 Список пазлов")
	btnListPieces := keyboard.Text("📋 Список деталей")
	btnCompletePuzzle := keyboard.Text("✅ Засчитать пазл")
	btnBack := keyboard.Text("🔙 Назад")

	keyboard.Reply(
		keyboard.Row(btnListPuzzles),
		keyboard.Row(btnListPieces),
		keyboard.Row(btnCompletePuzzle),
		keyboard.Row(btnBack),
	)

	return keyboard
}

// createAdminsKeyboard создает клавиатуру для работы с администраторами
func (ab *AdminBot) createAdminsKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	btnListAdmins := keyboard.Text("👮 Список администраторов")
	btnAddAdmin := keyboard.Text("➕ Добавить администратора")
	btnBack := keyboard.Text("🔙 Назад")

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

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этому боту.")
	}

	keyboard := ab.createMainKeyboard()
	return c.Send("Привет, администратор! Я бот для управления системой пазлов. Выберите действие на клавиатуре или используйте /help для просмотра доступных команд.", keyboard)
}

// handleUsers обрабатывает команду /users
func (ab *AdminBot) handleUsers(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил список пользователей", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	args := strings.Fields(c.Message().Payload)
	var group string
	if len(args) > 0 {
		normalizedGroup, valid := NormalizeGroupName(args[0])
		if !valid {
			return c.Send("Неверный формат группы. Группа должна быть от Н1 до Н6 (или H1 до H6).")
		}
		group = normalizedGroup
		ab.logger.Infof("Фильтрация пользователей по группе: %s", group)
	}

	usersData, err := ab.apiClient.Get("/users", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
	}

	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
	}

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

	var message strings.Builder
	if group == "" {
		message.WriteString("Список всех пользователей:\n\n")
	} else {
		message.WriteString(fmt.Sprintf("Список пользователей в группе %s:\n\n", group))
	}

	for i, user := range filteredUsers {
		pieceCount, _ := ab.getUserPieceCount(user.Id.String())
		message.WriteString(fmt.Sprintf("%d. %s %s (Группа: %s, Деталей: %d)\n",
			i+1, user.FirstName, user.LastName, user.Group, pieceCount))
	}

	return c.Send(message.String())
}

// handleUser обрабатывает команду /user
func (ab *AdminBot) handleUser(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил информацию о пользователе", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	args := strings.Fields(c.Message().Payload)
	if len(args) == 0 {
		return c.Send("Укажите ID пользователя. Например: /user 123e4567-e89b-12d3-a456-426614174000")
	}

	userID := args[0]

	userData, err := ab.apiClient.Get("/users/"+userID, nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователя через API: %v", err)
		return c.Send("Пользователь не найден.")
	}

	var userResp struct {
		models.User
		PieceCount int `json:"piece_count"`
	}
	if err := json.Unmarshal(userData, &userResp); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении информации о пользователе. Пожалуйста, попробуйте позже.")
	}

	if userResp.Deleted {
		return c.Send("Пользователь удален.")
	}

	message := fmt.Sprintf("Информация о пользователе:\n\n"+
		"ID: %s\n"+
		"Имя: %s\n"+
		"Фамилия: %s\n"+
		"Отчество: %s\n"+
		"Telegram: %s\n"+
		"Группа: %s\n"+
		"Деталей: %d\n"+
		"Дата регистрации: %s",
		userResp.Id, userResp.FirstName, userResp.LastName, userResp.MiddleName,
		userResp.Telegramm, userResp.Group, userResp.PieceCount, userResp.RegistrationTime.Format("02.01.2006 15:04:05"))

	return c.Send(message)
}

// handlePuzzles обрабатывает команду /puzzles
func (ab *AdminBot) handlePuzzles(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил список пазлов", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	puzzlesData, err := ab.apiClient.Get("/puzzles", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пазлов через API: %v", err)
		return c.Send("Произошла ошибка при получении списка пазлов. Пожалуйста, попробуйте позже.")
	}

	var puzzlesResponse struct {
		Total   int              `json:"total"`
		Puzzles []*models.Puzzle `json:"puzzles"`
	}
	if err := json.Unmarshal(puzzlesData, &puzzlesResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении списка пазлов. Пожалуйста, попробуйте позже.")
	}

	if len(puzzlesResponse.Puzzles) == 0 {
		return c.Send("Пазлы не найдены.")
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("Список пазлов (%d):\n\n", len(puzzlesResponse.Puzzles)))

	completedCount := 0
	for _, puzzle := range puzzlesResponse.Puzzles {
		status := "❌"
		if puzzle.IsCompleted {
			status = "✅"
			completedCount++
		}
		name := puzzle.Name
		if name == "" {
			name = fmt.Sprintf("Пазл %d", puzzle.Id)
		}
		message.WriteString(fmt.Sprintf("#%d %s: %s\n", puzzle.Id, name, status))
	}

	message.WriteString(fmt.Sprintf("\nЗасчитано: %d из %d", completedCount, len(puzzlesResponse.Puzzles)))
	message.WriteString("\n\nДля засчитывания пазла используйте:\n/complete <номер_пазла>")

	return c.Send(message.String())
}

// handleCompletePuzzle обрабатывает команду /complete
func (ab *AdminBot) handleCompletePuzzle(c tele.Context) error {
	ab.logger.Infof("Пользователь %d вызвал команду /complete", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	args := strings.Fields(c.Message().Payload)
	if len(args) == 0 {
		return c.Send("Укажите номер пазла. Например: /complete 5")
	}

	puzzleId, err := strconv.Atoi(args[0])
	if err != nil || puzzleId < 1 || puzzleId > 30 {
		return c.Send("Неверный номер пазла. Укажите число от 1 до 30.")
	}

	return ab.completePuzzleAndNotify(c, puzzleId)
}

// completePuzzleAndNotify засчитывает пазл и создает уведомление для владельцев деталей
func (ab *AdminBot) completePuzzleAndNotify(c tele.Context, puzzleId int) error {
	ab.logger.Infof("Засчитывание пазла #%d", puzzleId)

	// Получаем информацию о пазле до засчитывания
	puzzleData, err := ab.apiClient.Get(fmt.Sprintf("/puzzles/%d", puzzleId), nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пазла: %v", err)
		return c.Send("Ошибка: пазл не найден.")
	}

	var puzzleInfo struct {
		Id          int    `json:"id"`
		Name        string `json:"name"`
		IsCompleted bool   `json:"is_completed"`
	}
	json.Unmarshal(puzzleData, &puzzleInfo)

	if puzzleInfo.IsCompleted {
		return c.Send(fmt.Sprintf("Пазл #%d уже был засчитан ранее.", puzzleId))
	}

	// Засчитываем пазл через API
	completeData, err := ab.apiClient.Post(fmt.Sprintf("/puzzles/%d/complete", puzzleId), nil)
	if err != nil {
		ab.logger.Errorf("Ошибка засчитывания пазла: %v", err)
		return c.Send(fmt.Sprintf("Ошибка при засчитывании пазла: %v", err))
	}

	var completeResponse struct {
		Success       bool `json:"success"`
		UsersToNotify []struct {
			Id        string `json:"id"`
			Telegramm string `json:"telegramm"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Group     string `json:"group"`
		} `json:"users_to_notify"`
	}
	if err := json.Unmarshal(completeData, &completeResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа: %v", err)
		return c.Send("Ошибка при обработке ответа сервера.")
	}

	puzzleName := puzzleInfo.Name
	if puzzleName == "" {
		puzzleName = fmt.Sprintf("Пазл #%d", puzzleId)
	}

	// Собираем список ID пользователей для уведомления
	var userIds []string
	for _, user := range completeResponse.UsersToNotify {
		if user.Id != "" {
			userIds = append(userIds, user.Id)
		}
	}

	keyboard := ab.createMainKeyboard()

	if len(userIds) == 0 {
		resultMsg := fmt.Sprintf("✅ Пазл \"%s\" (#%d) успешно засчитан!\n\n"+
			"⚠️ Нет пользователей для уведомления.",
			puzzleName, puzzleId)
		return c.Send(resultMsg, keyboard)
	}

	// Создаем уведомление через API с конкретными пользователями
	notificationMsg := fmt.Sprintf("🎉 Поздравляем!\n\n"+
		"Ваш пазл \"%s\" засчитан!\n"+
		"Теперь вы участвуете в розыгрыше призов.\n\n"+
		"Спасибо за участие!", puzzleName)

	notificationData := map[string]interface{}{
		"message":  notificationMsg,
		"user_ids": userIds,
	}

	_, err = ab.apiClient.Post("/notifications", notificationData)
	if err != nil {
		ab.logger.Errorf("Ошибка создания уведомления: %v", err)
		resultMsg := fmt.Sprintf("✅ Пазл \"%s\" (#%d) успешно засчитан!\n\n"+
			"⚠️ Ошибка создания уведомления: %v\n"+
			"Участников: %d",
			puzzleName, puzzleId, err, len(userIds))
		return c.Send(resultMsg, keyboard)
	}

	resultMsg := fmt.Sprintf("✅ Пазл \"%s\" (#%d) успешно засчитан!\n\n"+
		"📨 Уведомление создано для %d участников.\n"+
		"Сообщения будут отправлены автоматически.",
		puzzleName, puzzleId, len(userIds))

	return c.Send(resultMsg, keyboard)
}

// handlePiecesCommand обрабатывает команду /pieces
func (ab *AdminBot) handlePiecesCommand(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил список деталей", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	piecesData, err := ab.apiClient.Get("/pieces", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения деталей через API: %v", err)
		return c.Send("Произошла ошибка при получении списка деталей. Пожалуйста, попробуйте позже.")
	}

	var piecesResponse struct {
		Total  int                   `json:"total"`
		Pieces []*models.PuzzlePiece `json:"pieces"`
	}
	if err := json.Unmarshal(piecesData, &piecesResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении списка деталей. Пожалуйста, попробуйте позже.")
	}

	if piecesResponse.Total == 0 {
		return c.Send("Детали не найдены. Используйте веб-интерфейс для импорта деталей.")
	}

	// Считаем статистику
	registeredCount := 0
	for _, piece := range piecesResponse.Pieces {
		if piece.OwnerId != nil {
			registeredCount++
		}
	}

	message := fmt.Sprintf("Статистика деталей:\n\n"+
		"Всего деталей: %d\n"+
		"Зарегистрировано: %d\n"+
		"Свободно: %d\n\n"+
		"Для детального просмотра используйте веб-интерфейс.",
		piecesResponse.Total, registeredCount, piecesResponse.Total-registeredCount)

	return c.Send(message)
}

// handleLottery обрабатывает команду /lottery
func (ab *AdminBot) handleLottery(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил статистику розыгрыша", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	lotteryData, err := ab.apiClient.Get("/stats/lottery", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения статистики розыгрыша через API: %v", err)
		return c.Send("Произошла ошибка при получении статистики. Пожалуйста, попробуйте позже.")
	}

	var lotteryResponse struct {
		TotalUsers       int `json:"total_users"`
		TotalPuzzles     int `json:"total_puzzles"`
		CompletedPuzzles int `json:"completed_puzzles"`
		Users            []struct {
			FirstName       string `json:"first_name"`
			LastName        string `json:"last_name"`
			Group           string `json:"group"`
			TotalPieces     int    `json:"total_pieces"`
			CompletedPieces int    `json:"completed_pieces"`
		} `json:"users"`
	}
	if err := json.Unmarshal(lotteryData, &lotteryResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении статистики. Пожалуйста, попробуйте позже.")
	}

	var message strings.Builder
	message.WriteString("📊 Статистика розыгрыша\n\n")
	message.WriteString(fmt.Sprintf("Всего пользователей: %d\n", lotteryResponse.TotalUsers))
	message.WriteString(fmt.Sprintf("Всего пазлов: %d\n", lotteryResponse.TotalPuzzles))
	message.WriteString(fmt.Sprintf("Собрано пазлов: %d\n\n", lotteryResponse.CompletedPuzzles))

	if len(lotteryResponse.Users) > 0 {
		message.WriteString("Пользователи с деталями собранных пазлов:\n\n")
		for i, user := range lotteryResponse.Users {
			if user.CompletedPieces > 0 {
				message.WriteString(fmt.Sprintf("%d. %s %s (%s) - %d деталей в собранных пазлах\n",
					i+1, user.FirstName, user.LastName, user.Group, user.CompletedPieces))
			}
		}
	}

	return c.Send(message.String())
}

// handleAddAdmin обрабатывает команду /addadmin
func (ab *AdminBot) handleAddAdmin(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил добавление администратора", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	args := strings.Fields(c.Message().Payload)
	if len(args) == 0 {
		return c.Send("Укажите ID пользователя. Например: /addadmin 123456789")
	}

	adminID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return c.Send("Неверный формат ID пользователя. Используйте целое число.")
	}

	var adminName string
	if len(args) > 1 {
		adminName = strings.Join(args[1:], " ")
	}

	// Проверяем, не является ли уже администратором
	if ab.isAdmin(adminID) {
		return c.Send(fmt.Sprintf("Пользователь с ID %d уже является администратором.", adminID))
	}

	// Добавляем через API
	err = ab.addAdminViaAPI(adminID, adminName)
	if err != nil {
		ab.logger.Errorf("Ошибка добавления администратора через API: %v", err)
		return c.Send(fmt.Sprintf("Ошибка добавления администратора: %v", err))
	}

	return c.Send(fmt.Sprintf("Пользователь с ID %d успешно добавлен в список администраторов.", adminID))
}

// handleListAdmins обрабатывает команду /listadmins
func (ab *AdminBot) handleListAdmins(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил список администраторов", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой команде.")
	}

	admins, err := ab.getAdminsViaAPI()
	if err != nil {
		ab.logger.Errorf("Ошибка получения списка администраторов: %v", err)
		return c.Send("Ошибка получения списка администраторов.")
	}

	if len(admins) == 0 {
		return c.Send("Список администраторов пуст.")
	}

	var message strings.Builder
	message.WriteString("Список администраторов:\n\n")

	for i, admin := range admins {
		if admin.Name != "" {
			message.WriteString(fmt.Sprintf("%d. %d (%s)\n", i+1, admin.ID, admin.Name))
		} else {
			message.WriteString(fmt.Sprintf("%d. %d\n", i+1, admin.ID))
		}
	}

	return c.Send(message.String())
}

// handleBroadcastButton обрабатывает нажатие на кнопку "Рассылка"
func (ab *AdminBot) handleBroadcastButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Рассылка'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	keyboard := ab.createBroadcastKeyboard()

	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "broadcast_text",
		Params: make(map[string]string),
	}

	return c.Send("Введите текст сообщения для рассылки всем пользователям:", keyboard)
}

// createBroadcastKeyboard создает клавиатуру для рассылки
func (ab *AdminBot) createBroadcastKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	btnCancel := keyboard.Text("❌ Отмена")
	btnBack := keyboard.Text("🔙 Назад")

	keyboard.Reply(
		keyboard.Row(btnCancel),
		keyboard.Row(btnBack),
	)

	return keyboard
}

// handleBroadcastText обрабатывает ввод текста для рассылки
func (ab *AdminBot) handleBroadcastText(c tele.Context, state *BotState) error {
	text := c.Text()
	state.Params["text"] = text
	state.State = "broadcast_group"

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

	return c.Send("Выберите группу для рассылки или нажмите кнопку 'Все группы':", keyboard)
}

// broadcastMessage создает уведомление для рассылки через очередь
func (ab *AdminBot) broadcastMessage(c tele.Context, text string, group string) error {
	// Создаем уведомление через API
	notificationData := map[string]interface{}{
		"message": text,
		"group":   group,
	}

	responseData, err := ab.apiClient.Post("/notifications", notificationData)
	if err != nil {
		ab.logger.Errorf("Ошибка создания уведомления: %v", err)
		delete(ab.states, c.Sender().ID)
		keyboard := ab.createMainKeyboard()
		return c.Send("Ошибка при создании рассылки. Попробуйте позже.", keyboard)
	}

	var response struct {
		Success      bool `json:"success"`
		Notification struct {
			Id string `json:"id"`
		} `json:"notification"`
	}
	json.Unmarshal(responseData, &response)

	// Сбрасываем состояние
	delete(ab.states, c.Sender().ID)
	keyboard := ab.createMainKeyboard()

	groupText := "всем пользователям"
	if group != "" {
		groupText = fmt.Sprintf("группе %s", group)
	}

	return c.Send(fmt.Sprintf("✅ Рассылка создана!\n\nСообщение будет отправлено %s в ближайшее время.", groupText), keyboard)
}

// handleAllGroupsButton обрабатывает нажатие на кнопку "Все группы"
func (ab *AdminBot) handleAllGroupsButton(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этой функции.")
	}

	userID := c.Sender().ID

	state, ok := ab.states[userID]
	if !ok || state.State != "broadcast_group" {
		keyboard := ab.createMainKeyboard()
		return c.Send("Используйте кнопки для навигации или /help для просмотра доступных команд.", keyboard)
	}

	return ab.broadcastMessage(c, state.Params["text"], "")
}

// handleHelp обрабатывает команду /help
func (ab *AdminBot) handleHelp(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил справку", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send("У вас нет доступа к этому боту.")
	}

	message := "Доступные команды:\n\n" +
		"/users [группа] - Список пользователей (опционально фильтр по группе)\n" +
		"/user <ID> - Информация о пользователе\n" +
		"/puzzles - Список пазлов\n" +
		"/pieces - Статистика деталей\n" +
		"/complete <номер> - Засчитать пазл и уведомить участников\n" +
		"/lottery - Статистика для розыгрыша\n" +
		"/addadmin <ID> - Добавить администратора\n" +
		"/listadmins - Список администраторов\n" +
		"/help - Показать эту справку"

	return c.Send(message)
}

// showUsersByGroup показывает пользователей выбранной группы
func (ab *AdminBot) showUsersByGroup(c tele.Context, group string) error {
	usersData, err := ab.apiClient.Get("/users", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
	}

	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send("Произошла ошибка при получении пользователей. Пожалуйста, попробуйте позже.")
	}

	var filteredUsers []*models.User
	for _, user := range usersResponse.Users {
		if user.Group == group && !user.Deleted {
			filteredUsers = append(filteredUsers, user)
		}
	}

	if len(filteredUsers) == 0 {
		return c.Send(fmt.Sprintf("Пользователи в группе %s не найдены.", group))
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("Список пользователей в группе %s:\n\n", group))

	for i, user := range filteredUsers {
		pieceCount, _ := ab.getUserPieceCount(user.Id.String())
		message.WriteString(fmt.Sprintf("%d. %s %s (Деталей: %d)\n",
			i+1, user.FirstName, user.LastName, pieceCount))
	}

	return c.Send(message.String())
}

// getUserPieceCount получает количество деталей у пользователя
func (ab *AdminBot) getUserPieceCount(userID string) (int, error) {
	piecesData, err := ab.apiClient.Get(fmt.Sprintf("/users/%s/pieces", userID), nil)
	if err != nil {
		return 0, err
	}

	var piecesResponse struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(piecesData, &piecesResponse); err != nil {
		return 0, err
	}

	return piecesResponse.Total, nil
}
