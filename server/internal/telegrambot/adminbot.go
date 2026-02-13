package telegrambot

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/messages"
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
	State        string
	Params       map[string]string
	Attachments  []string // Пути к загруженным файлам для рассылки
	LastMsgID    int
	LastMsgText  string
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

	// Обработчики медиа для рассылок
	ab.bot.Handle(tele.OnPhoto, ab.handleBroadcastPhoto)
	ab.bot.Handle(tele.OnDocument, ab.handleBroadcastDocument)

	// Запуск бота
	go ab.bot.Start()

	return nil
}

// handleText обрабатывает текстовые сообщения
func (ab *AdminBot) handleText(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccess)
	}

	text := c.Text()
	userID := c.Sender().ID

	state, ok := ab.states[userID]
	if !ok {
		keyboard := ab.createMainKeyboard()
		return c.Send(messages.AdminUseButtons, keyboard)
	}

	switch state.State {
	case "broadcast_text":
		return ab.handleBroadcastText(c, state)

	case "broadcast_attachments":
		if text == "✅ Готово" {
			return ab.handleBroadcastAttachmentsDone(c, state)
		}
		// Если пользователь ввел текст вместо файла, напоминаем
		keyboard := ab.createAttachmentKeyboard()
		return c.Send(messages.BroadcastSendPhotoOrDone, keyboard)

	case "broadcast_group":
		if text == "🌐 Все группы" {
			return ab.broadcastMessage(c, state.Params["text"], "", state.Attachments)
		} else if GroupRegex.MatchString(text) {
			normalizedGroup, _ := NormalizeGroupName(text)
			return ab.broadcastMessage(c, state.Params["text"], normalizedGroup, state.Attachments)
		} else {
			return c.Send(messages.InvalidGroupFormat)
		}

	case "add_admin_id":
		_, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return c.Send(messages.AdminInvalidUserId)
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

		return c.Send(messages.AdminEnterUserName, keyboard)

	case "add_admin_name":
		state.Params["admin_name"] = text
		return ab.addAdminFromParams(c, state.Params)

	case "user_by_group":
		if !GroupRegex.MatchString(text) {
			return c.Send(messages.InvalidGroupFormat)
		}

		normalizedGroup, _ := NormalizeGroupName(text)
		ab.logger.Infof("Пользователь %d выбрал группу %s для фильтрации", c.Sender().ID, normalizedGroup)

		delete(ab.states, userID)

		return ab.showUsersByGroup(c, normalizedGroup)

	case "complete_puzzle_id":
		puzzleId, err := strconv.Atoi(text)
		if err != nil || puzzleId < 1 || puzzleId > 30 {
			return c.Send(messages.AdminPuzzleInvalidNum)
		}

		delete(ab.states, userID)
		return ab.completePuzzleAndNotify(c, puzzleId)

	default:
		delete(ab.states, userID)
		keyboard := ab.createMainKeyboard()
		return c.Send(messages.AdminUseButtons, keyboard)
	}
}

// handleNoLimitsButton обрабатывает нажатие на кнопку "Без ограничений"
func (ab *AdminBot) handleNoLimitsButton(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessFunc)
	}

	userID := c.Sender().ID

	state, ok := ab.states[userID]
	if !ok {
		keyboard := ab.createMainKeyboard()
		return c.Send(messages.AdminUseButtons, keyboard)
	}

	switch state.State {
	case "add_admin_name":
		state.Params["admin_name"] = ""
		return ab.addAdminFromParams(c, state.Params)

	default:
		delete(ab.states, userID)
		keyboard := ab.createMainKeyboard()
		return c.Send(messages.AdminUseButtons, keyboard)
	}
}

// handleGroupButton обрабатывает нажатие на кнопку группы
func (ab *AdminBot) handleGroupButton(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessFunc)
	}

	userID := c.Sender().ID

	state, ok := ab.states[userID]
	if !ok {
		keyboard := ab.createMainKeyboard()
		return c.Send(messages.AdminUseButtons, keyboard)
	}

	group := c.Text()
	normalizedGroup, valid := NormalizeGroupName(group)
	if !valid {
		return c.Send(messages.InvalidGroupFormat)
	}

	switch state.State {
	case "broadcast_group":
		return ab.broadcastMessage(c, state.Params["text"], normalizedGroup, state.Attachments)

	case "user_by_group":
		ab.logger.Infof("Пользователь %d выбрал группу %s для фильтрации", c.Sender().ID, normalizedGroup)
		delete(ab.states, userID)
		return ab.showUsersByGroup(c, normalizedGroup)

	default:
		delete(ab.states, userID)
		keyboard := ab.createMainKeyboard()
		return c.Send(messages.AdminUseButtons, keyboard)
	}
}

// handleCancelButton обрабатывает нажатие на кнопку "Отмена"
func (ab *AdminBot) handleCancelButton(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessFunc)
	}

	userID := c.Sender().ID
	delete(ab.states, userID)

	keyboard := ab.createMainKeyboard()
	return c.Send(messages.AdminCancelled, keyboard)
}

// addAdminFromParams добавляет администратора из параметров
func (ab *AdminBot) addAdminFromParams(c tele.Context, params map[string]string) error {
	adminID, err := strconv.ParseInt(params["admin_id"], 10, 64)
	if err != nil {
		return c.Send(messages.AdminInvalidUserIdShort)
	}

	adminName := params["admin_name"]

	// Проверяем, не является ли уже администратором
	if ab.isAdmin(adminID) {
		return c.Send(messages.AdminAlreadyAdmin(adminID))
	}

	// Добавляем через API
	err = ab.addAdminViaAPI(adminID, adminName)
	if err != nil {
		ab.logger.Errorf("Ошибка добавления администратора через API: %v", err)
		return c.Send(messages.AdminAddError(err))
	}

	delete(ab.states, c.Sender().ID)
	keyboard := ab.createMainKeyboard()

	return c.Send(messages.AdminAddedSuccess(adminID), keyboard)
}

// handleUsersButton обрабатывает нажатие на кнопку "Пользователи"
func (ab *AdminBot) handleUsersButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Пользователи'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessFunc)
	}

	keyboard := ab.createUsersKeyboard()
	return c.Send(messages.AdminUsersMenu, keyboard)
}

// handlePuzzlesButton обрабатывает нажатие на кнопку "Пазлы"
func (ab *AdminBot) handlePuzzlesButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Пазлы'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessFunc)
	}

	keyboard := ab.createPuzzlesKeyboard()
	return c.Send(messages.AdminPuzzlesMenu, keyboard)
}

// handleAdminsButton обрабатывает нажатие на кнопку "Администраторы"
func (ab *AdminBot) handleAdminsButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Администраторы'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessFunc)
	}

	keyboard := ab.createAdminsKeyboard()
	return c.Send(messages.AdminAdminsMenu, keyboard)
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
		return c.Send(messages.AdminNoAccessFunc)
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

	return c.Send(messages.AdminSelectGroup, keyboard)
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
		return c.Send(messages.AdminNoAccessFunc)
	}

	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "complete_puzzle_id",
		Params: make(map[string]string),
	}

	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnCancel := keyboard.Text("❌ Отмена")
	keyboard.Reply(keyboard.Row(btnCancel))

	return c.Send(messages.AdminPuzzleEnterNum, keyboard)
}

// handleAddAdminButton обрабатывает нажатие на кнопку "Добавить администратора"
func (ab *AdminBot) handleAddAdminButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Добавить администратора'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessFunc)
	}

	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "add_admin_id",
		Params: make(map[string]string),
	}

	return c.Send(messages.AdminEnterUserId)
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
		return c.Send(messages.AdminNoAccessFunc)
	}

	keyboard := ab.createMainKeyboard()
	return c.Send(messages.AdminMainMenu, keyboard)
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
		return c.Send(messages.AdminNoAccess)
	}

	keyboard := ab.createMainKeyboard()
	return c.Send(messages.AdminWelcome, keyboard)
}

// handleUsers обрабатывает команду /users
func (ab *AdminBot) handleUsers(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил список пользователей", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessCommand)
	}

	args := strings.Fields(c.Message().Payload)
	var group string
	if len(args) > 0 {
		normalizedGroup, valid := NormalizeGroupName(args[0])
		if !valid {
			return c.Send(messages.InvalidGroupFormat)
		}
		group = normalizedGroup
		ab.logger.Infof("Фильтрация пользователей по группе: %s", group)
	}

	usersData, err := ab.apiClient.Get("/users", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send(messages.AdminErrGetUsers)
	}

	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send(messages.AdminErrGetUsers)
	}

	var filteredUsers []*models.User
	for _, user := range usersResponse.Users {
		if (group == "" || user.Group == group) && !user.Deleted {
			filteredUsers = append(filteredUsers, user)
		}
	}

	if len(filteredUsers) == 0 {
		if group == "" {
			return c.Send(messages.AdminUsersNotFound)
		} else {
			return c.Send(messages.AdminUsersNotFoundInGroup(group))
		}
	}

	var message strings.Builder
	if group == "" {
		message.WriteString(messages.AdminUsersListAll)
	} else {
		message.WriteString(messages.AdminUsersListGroupHeader(group))
	}

	for i, user := range filteredUsers {
		pieceCount, _ := ab.getUserPieceCount(user.Id.String())
		message.WriteString(messages.AdminUserLine(i+1, user.FirstName, user.LastName, user.Group, pieceCount))
	}

	return c.Send(message.String())
}

// handleUser обрабатывает команду /user
func (ab *AdminBot) handleUser(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил информацию о пользователе", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessCommand)
	}

	args := strings.Fields(c.Message().Payload)
	if len(args) == 0 {
		return c.Send(messages.AdminUserSpecifyId)
	}

	userID := args[0]

	userData, err := ab.apiClient.Get("/users/"+userID, nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователя через API: %v", err)
		return c.Send(messages.AdminUserNotFound)
	}

	var userResp struct {
		models.User
		PieceCount int `json:"piece_count"`
	}
	if err := json.Unmarshal(userData, &userResp); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send(messages.AdminErrGetUser)
	}

	if userResp.Deleted {
		return c.Send(messages.AdminUserDeleted)
	}

	message := messages.AdminUserInfo(
		userResp.Id.String(), userResp.FirstName, userResp.LastName, userResp.MiddleName,
		userResp.Telegramm, userResp.Group, userResp.PieceCount, userResp.RegistrationTime)

	return c.Send(message)
}

// handlePuzzles обрабатывает команду /puzzles
func (ab *AdminBot) handlePuzzles(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил список пазлов", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessCommand)
	}

	puzzlesData, err := ab.apiClient.Get("/puzzles", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пазлов через API: %v", err)
		return c.Send(messages.AdminErrGetPuzzles)
	}

	type PuzzleWithProgress struct {
		Id          int        `json:"id"`
		Name        string     `json:"name"`
		IsCompleted bool       `json:"is_completed"`
		CompletedAt *time.Time `json:"completed_at,omitempty"`
		OwnedPieces int        `json:"owned_pieces"`
		TotalPieces int        `json:"total_pieces"`
	}

	var puzzlesResponse struct {
		Total   int                  `json:"total"`
		Puzzles []PuzzleWithProgress `json:"puzzles"`
	}
	if err := json.Unmarshal(puzzlesData, &puzzlesResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send(messages.AdminErrGetPuzzles)
	}

	if len(puzzlesResponse.Puzzles) == 0 {
		return c.Send(messages.AdminPuzzlesNotFound)
	}

	var message strings.Builder
	message.WriteString(messages.AdminPuzzlesListHeader(len(puzzlesResponse.Puzzles)))

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
		message.WriteString(messages.AdminPuzzleLine(puzzle.Id, puzzle.OwnedPieces, name, status))
	}

	message.WriteString(messages.AdminPuzzlesCompleted(completedCount, len(puzzlesResponse.Puzzles)))
	message.WriteString(messages.AdminPuzzlesFooter)

	return c.Send(message.String())
}

// handleCompletePuzzle обрабатывает команду /complete
func (ab *AdminBot) handleCompletePuzzle(c tele.Context) error {
	ab.logger.Infof("Пользователь %d вызвал команду /complete", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessCommand)
	}

	args := strings.Fields(c.Message().Payload)
	if len(args) == 0 {
		return c.Send(messages.AdminPuzzleSpecifyNum)
	}

	puzzleId, err := strconv.Atoi(args[0])
	if err != nil || puzzleId < 1 || puzzleId > 30 {
		return c.Send(messages.AdminPuzzleInvalidNum)
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
		return c.Send(messages.AdminPuzzleNotFound)
	}

	var puzzleInfo struct {
		Id          int    `json:"id"`
		Name        string `json:"name"`
		IsCompleted bool   `json:"is_completed"`
	}
	json.Unmarshal(puzzleData, &puzzleInfo)

	if puzzleInfo.IsCompleted {
		return c.Send(messages.AdminPuzzleAlreadyCompleted(puzzleId))
	}

	// Засчитываем пазл через API
	completeData, err := ab.apiClient.Post(fmt.Sprintf("/puzzles/%d/complete", puzzleId), nil)
	if err != nil {
		ab.logger.Errorf("Ошибка засчитывания пазла: %v", err)
		return c.Send(messages.AdminPuzzleCompleteErr(err))
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
		return c.Send(messages.AdminErrCompletePzl)
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
		return c.Send(messages.AdminPuzzleCompletedNoUsers(puzzleName, puzzleId), keyboard)
	}

	// Создаем уведомление через API с конкретными пользователями
	notificationMsg := messages.PuzzleCompletedNotification(puzzleName)

	notificationData := map[string]interface{}{
		"message":  notificationMsg,
		"user_ids": userIds,
	}

	_, err = ab.apiClient.Post("/notifications", notificationData)
	if err != nil {
		ab.logger.Errorf("Ошибка создания уведомления: %v", err)
		return c.Send(messages.AdminPuzzleCompletedNotifyErr(puzzleName, puzzleId, err, len(userIds)), keyboard)
	}

	return c.Send(messages.AdminPuzzleCompletedSuccess(puzzleName, puzzleId, len(userIds)), keyboard)
}

// handlePiecesCommand обрабатывает команду /pieces
func (ab *AdminBot) handlePiecesCommand(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил список деталей", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessCommand)
	}

	piecesData, err := ab.apiClient.Get("/pieces", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения деталей через API: %v", err)
		return c.Send(messages.AdminErrGetPieces)
	}

	var piecesResponse struct {
		Total  int                   `json:"total"`
		Pieces []*models.PuzzlePiece `json:"pieces"`
	}
	if err := json.Unmarshal(piecesData, &piecesResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send(messages.AdminErrGetPieces)
	}

	if piecesResponse.Total == 0 {
		return c.Send(messages.AdminPiecesNotFound)
	}

	// Считаем статистику
	registeredCount := 0
	for _, piece := range piecesResponse.Pieces {
		if piece.OwnerId != nil {
			registeredCount++
		}
	}

	return c.Send(messages.AdminPiecesStats(piecesResponse.Total, registeredCount, piecesResponse.Total-registeredCount))
}

// handleLottery обрабатывает команду /lottery
func (ab *AdminBot) handleLottery(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил статистику розыгрыша", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessCommand)
	}

	lotteryData, err := ab.apiClient.Get("/stats/lottery", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения статистики розыгрыша через API: %v", err)
		return c.Send(messages.AdminErrGetLottery)
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
		return c.Send(messages.AdminErrGetLottery)
	}

	var message strings.Builder
	message.WriteString(messages.AdminLotteryHeader)
	message.WriteString(messages.AdminLotteryStats(lotteryResponse.TotalUsers, lotteryResponse.TotalPuzzles, lotteryResponse.CompletedPuzzles))

	if len(lotteryResponse.Users) > 0 {
		message.WriteString(messages.AdminLotteryUsersHeader)
		for i, user := range lotteryResponse.Users {
			if user.CompletedPieces > 0 {
				message.WriteString(messages.AdminLotteryUserLine(i+1, user.FirstName, user.LastName, user.Group, user.CompletedPieces))
			}
		}
	}

	return c.Send(message.String())
}

// handleAddAdmin обрабатывает команду /addadmin
func (ab *AdminBot) handleAddAdmin(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил добавление администратора", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessCommand)
	}

	args := strings.Fields(c.Message().Payload)
	if len(args) == 0 {
		return c.Send(messages.AdminSpecifyUserId)
	}

	adminID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return c.Send(messages.AdminInvalidUserIdUse)
	}

	var adminName string
	if len(args) > 1 {
		adminName = strings.Join(args[1:], " ")
	}

	// Проверяем, не является ли уже администратором
	if ab.isAdmin(adminID) {
		return c.Send(messages.AdminAlreadyAdmin(adminID))
	}

	// Добавляем через API
	err = ab.addAdminViaAPI(adminID, adminName)
	if err != nil {
		ab.logger.Errorf("Ошибка добавления администратора через API: %v", err)
		return c.Send(messages.AdminAddError(err))
	}

	return c.Send(messages.AdminAddedSuccess(adminID))
}

// handleListAdmins обрабатывает команду /listadmins
func (ab *AdminBot) handleListAdmins(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил список администраторов", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessCommand)
	}

	admins, err := ab.getAdminsViaAPI()
	if err != nil {
		ab.logger.Errorf("Ошибка получения списка администраторов: %v", err)
		return c.Send(messages.AdminErrGetAdmins)
	}

	if len(admins) == 0 {
		return c.Send(messages.AdminListEmpty)
	}

	var message strings.Builder
	message.WriteString(messages.AdminListHeader)

	for i, admin := range admins {
		message.WriteString(messages.AdminListLine(i+1, admin.ID, admin.Name))
	}

	return c.Send(message.String())
}

// handleBroadcastButton обрабатывает нажатие на кнопку "Рассылка"
func (ab *AdminBot) handleBroadcastButton(c tele.Context) error {
	ab.logger.Infof("Пользователь %d нажал на кнопку 'Рассылка'", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessFunc)
	}

	keyboard := ab.createBroadcastKeyboard()

	userID := c.Sender().ID
	ab.states[userID] = &BotState{
		State:  "broadcast_text",
		Params: make(map[string]string),
	}

	return c.Send(messages.BroadcastEnterText, keyboard)
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
	state.State = "broadcast_attachments"
	state.Attachments = nil // Очищаем предыдущие вложения

	keyboard := ab.createAttachmentKeyboard()
	return c.Send(messages.BroadcastTextSaved, keyboard)
}

// createAttachmentKeyboard создает клавиатуру для добавления вложений
func (ab *AdminBot) createAttachmentKeyboard() *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnDone := keyboard.Text("✅ Готово")
	btnCancel := keyboard.Text("❌ Отмена")
	keyboard.Reply(
		keyboard.Row(btnDone),
		keyboard.Row(btnCancel),
	)
	return keyboard
}

// handleBroadcastAttachmentsDone обрабатывает завершение добавления вложений
func (ab *AdminBot) handleBroadcastAttachmentsDone(c tele.Context, state *BotState) error {
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

	return c.Send(messages.BroadcastSelectGroupWithAttach(len(state.Attachments)), keyboard)
}

// broadcastMessage создает уведомление для рассылки через очередь
func (ab *AdminBot) broadcastMessage(c tele.Context, text string, group string, attachments []string) error {
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
		return c.Send(messages.BroadcastErrCreate, keyboard)
	}

	var response struct {
		Success      bool `json:"success"`
		Notification struct {
			Id string `json:"id"`
		} `json:"notification"`
	}
	json.Unmarshal(responseData, &response)

	// Загружаем файлы вложений
	uploadedCount := 0
	for _, attachPath := range attachments {
		err := ab.uploadAttachment(response.Notification.Id, attachPath)
		if err != nil {
			ab.logger.Errorf("Ошибка загрузки вложения %s: %v", attachPath, err)
		} else {
			uploadedCount++
		}
		// Удаляем временный файл после загрузки
		os.Remove(attachPath)
	}

	// Сбрасываем состояние
	delete(ab.states, c.Sender().ID)
	keyboard := ab.createMainKeyboard()

	return c.Send(messages.BroadcastCreated(uploadedCount, group), keyboard)
}

// handleAllGroupsButton обрабатывает нажатие на кнопку "Все группы"
func (ab *AdminBot) handleAllGroupsButton(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccessFunc)
	}

	userID := c.Sender().ID

	state, ok := ab.states[userID]
	if !ok || state.State != "broadcast_group" {
		keyboard := ab.createMainKeyboard()
		return c.Send(messages.AdminUseButtons, keyboard)
	}

	return ab.broadcastMessage(c, state.Params["text"], "", state.Attachments)
}

// uploadAttachment загружает файл вложения на сервер
func (ab *AdminBot) uploadAttachment(notificationId string, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer file.Close()

	return ab.apiClient.PostFile("/notifications/"+notificationId+"/attachments", file, filepath.Base(filePath))
}

// handleBroadcastPhoto обрабатывает фото для рассылки
func (ab *AdminBot) handleBroadcastPhoto(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return nil
	}

	userID := c.Sender().ID
	state, ok := ab.states[userID]
	if !ok || state.State != "broadcast_attachments" {
		return nil
	}

	photo := c.Message().Photo
	if photo == nil {
		return nil
	}

	// Скачиваем файл
	reader, err := ab.bot.File(&photo.File)
	if err != nil {
		ab.logger.Errorf("Ошибка получения файла: %v", err)
		return c.Send(messages.BroadcastErrPhoto)
	}

	// Сохраняем во временный файл
	tempDir := filepath.Join("data", "temp")
	os.MkdirAll(tempDir, 0755)

	filename := fmt.Sprintf("photo_%d_%s.jpg", userID, photo.FileID[:8])
	tempPath := filepath.Join(tempDir, filename)

	tempFile, err := os.Create(tempPath)
	if err != nil {
		ab.logger.Errorf("Ошибка создания временного файла: %v", err)
		return c.Send(messages.BroadcastErrSavePhoto)
	}

	_, err = io.Copy(tempFile, reader)
	tempFile.Close()
	if err != nil {
		ab.logger.Errorf("Ошибка записи файла: %v", err)
		os.Remove(tempPath)
		return c.Send(messages.BroadcastErrSavePhoto)
	}

	state.Attachments = append(state.Attachments, tempPath)

	keyboard := ab.createAttachmentKeyboard()
	return c.Send(messages.BroadcastPhotoAdded(len(state.Attachments)), keyboard)
}

// handleBroadcastDocument обрабатывает документ для рассылки
func (ab *AdminBot) handleBroadcastDocument(c tele.Context) error {
	if !ab.isAdmin(c.Sender().ID) {
		return nil
	}

	userID := c.Sender().ID
	state, ok := ab.states[userID]
	if !ok || state.State != "broadcast_attachments" {
		return nil
	}

	doc := c.Message().Document
	if doc == nil {
		return nil
	}

	// Скачиваем файл
	reader, err := ab.bot.File(&doc.File)
	if err != nil {
		ab.logger.Errorf("Ошибка получения файла: %v", err)
		return c.Send(messages.BroadcastErrDoc)
	}

	// Сохраняем во временный файл
	tempDir := filepath.Join("data", "temp")
	os.MkdirAll(tempDir, 0755)

	filename := fmt.Sprintf("%d_%s", userID, doc.FileName)
	tempPath := filepath.Join(tempDir, filename)

	tempFile, err := os.Create(tempPath)
	if err != nil {
		ab.logger.Errorf("Ошибка создания временного файла: %v", err)
		return c.Send(messages.BroadcastErrSaveDoc)
	}

	_, err = io.Copy(tempFile, reader)
	tempFile.Close()
	if err != nil {
		ab.logger.Errorf("Ошибка записи файла: %v", err)
		os.Remove(tempPath)
		return c.Send(messages.BroadcastErrSaveDoc)
	}

	state.Attachments = append(state.Attachments, tempPath)

	keyboard := ab.createAttachmentKeyboard()
	return c.Send(messages.BroadcastDocAdded(doc.FileName, len(state.Attachments)), keyboard)
}

// handleHelp обрабатывает команду /help
func (ab *AdminBot) handleHelp(c tele.Context) error {
	ab.logger.Infof("Пользователь %d запросил справку", c.Sender().ID)

	if !ab.isAdmin(c.Sender().ID) {
		return c.Send(messages.AdminNoAccess)
	}

	return c.Send(messages.AdminHelpMessage)
}

// showUsersByGroup показывает пользователей выбранной группы
func (ab *AdminBot) showUsersByGroup(c tele.Context, group string) error {
	usersData, err := ab.apiClient.Get("/users", nil)
	if err != nil {
		ab.logger.Errorf("Ошибка получения пользователей через API: %v", err)
		return c.Send(messages.AdminErrGetUsers)
	}

	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		ab.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send(messages.AdminErrGetUsers)
	}

	var filteredUsers []*models.User
	for _, user := range usersResponse.Users {
		if user.Group == group && !user.Deleted {
			filteredUsers = append(filteredUsers, user)
		}
	}

	if len(filteredUsers) == 0 {
		return c.Send(messages.AdminUsersNotFoundInGroup(group))
	}

	var message strings.Builder
	message.WriteString(messages.AdminUsersListGroupHeader(group))

	for i, user := range filteredUsers {
		pieceCount, _ := ab.getUserPieceCount(user.Id.String())
		message.WriteString(messages.AdminUserLineShort(i+1, user.FirstName, user.LastName, pieceCount))
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
