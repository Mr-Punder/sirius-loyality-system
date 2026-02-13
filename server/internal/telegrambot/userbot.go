package telegrambot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/messages"
	"github.com/MrPunder/sirius-loyality-system/internal/models"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	tele "gopkg.in/telebot.v3"
)

// Константы для этапов регистрации
const (
	RegistrationStepLastName   = 1
	RegistrationStepFirstName  = 2
	RegistrationStepMiddleName = 3
	RegistrationStepGroup      = 4
)

// Константы для анти-спам системы
const (
	MaxFailedAttempts = 3
	BlockDuration     = 5 * time.Minute
)

// RegistrationState хранит состояние регистрации пользователя
type RegistrationState struct {
	Step       int
	LastName   string
	FirstName  string
	MiddleName string
	Group      string
}

// FailedAttempts хранит информацию о неудачных попытках ввода кода
type FailedAttempts struct {
	Count     int
	LastTry   time.Time
	BlockedAt *time.Time
}

// UserBot представляет бота для пользователей
type UserBot struct {
	bot                *tele.Bot
	logger             logger.Logger
	config             Config
	apiClient          *APIClient
	registrationStates map[int64]*RegistrationState
	failedAttempts     map[int64]*FailedAttempts
	attemptsMutex      sync.RWMutex
	stopChan           chan struct{} // Канал для остановки горутины уведомлений
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
		failedAttempts:     make(map[int64]*FailedAttempts),
		stopChan:           make(chan struct{}),
	}, nil
}

// Start запускает бота
func (ub *UserBot) Start() error {
	ub.logger.Info("Запуск пользовательского бота")

	// Обработчик команды /start
	ub.bot.Handle("/start", ub.handleStart)

	// Обработчик команды /register
	ub.bot.Handle("/register", ub.handleRegister)

	// Обработчик команды /pieces
	ub.bot.Handle("/pieces", ub.handlePieces)

	// Обработчик для текстовых сообщений
	ub.bot.Handle(tele.OnText, ub.handleText)

	// Обработчики кнопок
	ub.bot.Handle("🧩 Мои детали", ub.handlePiecesButton)
	ub.bot.Handle("📷 Ввести код детали", ub.handleEnterCodeButton)
	ub.bot.Handle("❓ Помощь", ub.handleHelpButton)
	ub.bot.Handle("📝 Регистрация", ub.handleRegisterButton)
	ub.bot.Handle("⏭️ Пропустить", ub.handleSkipButton)
	ub.bot.Handle("❌ Отменить", ub.handleCancelButton)

	// Запуск бота
	go ub.bot.Start()

	// Запуск горутины для обработки уведомлений
	go ub.notificationPoller()

	return nil
}

// handlePiecesButton обрабатывает нажатие на кнопку "Мои детали"
func (ub *UserBot) handlePiecesButton(c tele.Context) error {
	return ub.handlePieces(c)
}

// handleEnterCodeButton обрабатывает нажатие на кнопку "Ввести код детали"
func (ub *UserBot) handleEnterCodeButton(c tele.Context) error {
	ub.logger.Infof("Пользователь %d нажал на кнопку 'Ввести код детали'", c.Sender().ID)

	// Проверяем блокировку за спам
	if ub.isUserBlocked(c.Sender().ID) {
		remaining := ub.getBlockTimeRemaining(c.Sender().ID)
		return c.Send(messages.TooManyAttemptsMsg(remaining))
	}

	// Проверяем, зарегистрирован ли пользователь
	user, err := ub.getUser(c.Sender().ID)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователя: %v", err)
		return c.Send(messages.ErrCheckRegistration)
	}

	if user == nil {
		return c.Send(messages.UserNotRegistered)
	}

	// Отправляем сообщение с инструкцией
	return c.Send(messages.UserEnterPieceCode)
}

// handleHelpButton обрабатывает нажатие на кнопку "Помощь"
func (ub *UserBot) handleHelpButton(c tele.Context) error {
	ub.logger.Infof("Пользователь %d нажал на кнопку 'Помощь'", c.Sender().ID)
	return c.Send(messages.UserHelpMessage)
}

// handleRegisterButton обрабатывает нажатие на кнопку "Регистрация"
func (ub *UserBot) handleRegisterButton(c tele.Context) error {
	return ub.handleRegister(c)
}

// createRegistrationKeyboard создает клавиатуру для регистрации
func (ub *UserBot) createRegistrationKeyboard(withSkip bool) *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}
	btnCancel := keyboard.Text("❌ Отменить")

	if withSkip {
		btnSkip := keyboard.Text("⏭️ Пропустить")
		keyboard.Reply(
			keyboard.Row(btnSkip),
			keyboard.Row(btnCancel),
		)
	} else {
		keyboard.Reply(keyboard.Row(btnCancel))
	}

	return keyboard
}

// handleSkipButton обрабатывает нажатие на кнопку "Пропустить"
func (ub *UserBot) handleSkipButton(c tele.Context) error {
	state, exists := ub.registrationStates[c.Sender().ID]
	if !exists {
		return c.Send(messages.RegNotInProgressUseCommand)
	}

	if state.Step == RegistrationStepMiddleName {
		state.MiddleName = ""
		state.Step = RegistrationStepGroup
		keyboard := ub.createRegistrationKeyboard(false)
		return c.Send(messages.RegEnterGroup, keyboard)
	}

	return c.Send(messages.RegCannotSkip)
}

// handleCancelButton обрабатывает нажатие на кнопку "Отменить"
func (ub *UserBot) handleCancelButton(c tele.Context) error {
	_, exists := ub.registrationStates[c.Sender().ID]
	if !exists {
		return c.Send(messages.RegNotInProgress)
	}

	delete(ub.registrationStates, c.Sender().ID)
	keyboard := ub.createMainKeyboard(false)
	return c.Send(messages.RegCancelled, keyboard)
}

// Stop останавливает бота
func (ub *UserBot) Stop() error {
	ub.logger.Info("Остановка пользовательского бота")
	close(ub.stopChan) // Останавливаем горутину уведомлений
	ub.bot.Stop()
	return nil
}

// createMainKeyboard создает основную клавиатуру с кнопками
func (ub *UserBot) createMainKeyboard(isRegistered bool) *tele.ReplyMarkup {
	keyboard := &tele.ReplyMarkup{ResizeKeyboard: true}

	btnPieces := keyboard.Text("🧩 Мои детали")
	btnEnterCode := keyboard.Text("📷 Ввести код детали")
	btnHelp := keyboard.Text("❓ Помощь")
	btnRegister := keyboard.Text("📝 Регистрация")

	if isRegistered {
		keyboard.Reply(
			keyboard.Row(btnPieces, btnEnterCode),
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

	user, err := ub.getUser(c.Sender().ID)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователя: %v", err)
		return c.Send(messages.ErrCheckRegistration)
	}

	var keyboard *tele.ReplyMarkup
	var message string

	if user != nil {
		keyboard = ub.createMainKeyboard(true)
		message = messages.UserWelcome(user.FirstName)
	} else {
		keyboard = ub.createMainKeyboard(false)
		message = messages.UserWelcomeUnregistered
	}

	return c.Send(message, keyboard)
}

// handleRegister обрабатывает команду /register
func (ub *UserBot) handleRegister(c tele.Context) error {
	ub.logger.Infof("Пользователь %d запросил регистрацию", c.Sender().ID)

	user, err := ub.getUser(c.Sender().ID)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователя: %v", err)
		return c.Send(messages.ErrCheckRegistration)
	}

	if user != nil {
		keyboard := ub.createMainKeyboard(true)
		return c.Send(messages.UserAlreadyRegisteredMsg(user.FirstName, user.LastName), keyboard)
	}

	ub.registrationStates[c.Sender().ID] = &RegistrationState{
		Step: RegistrationStepLastName,
	}

	keyboard := ub.createRegistrationKeyboard(false)
	return c.Send(messages.RegEnterLastName, keyboard)
}

// handlePieces обрабатывает команду /pieces
func (ub *UserBot) handlePieces(c tele.Context) error {
	ub.logger.Infof("Пользователь %d запросил свои детали", c.Sender().ID)

	user, err := ub.getUser(c.Sender().ID)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователя: %v", err)
		return c.Send(messages.ErrGetPieces)
	}

	if user == nil {
		keyboard := ub.createMainKeyboard(false)
		return c.Send(messages.UserNotRegistered, keyboard)
	}

	// Получаем детали пользователя через API
	piecesData, err := ub.apiClient.Get(fmt.Sprintf("/users/%s/pieces", user.Id), nil)
	if err != nil {
		ub.logger.Errorf("Ошибка получения деталей через API: %v", err)
		return c.Send(messages.ErrGetPieces)
	}

	var piecesResponse struct {
		Total  int                   `json:"total"`
		Pieces []*models.PuzzlePiece `json:"pieces"`
	}
	if err := json.Unmarshal(piecesData, &piecesResponse); err != nil {
		ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send(messages.ErrGetPieces)
	}

	keyboard := ub.createMainKeyboard(true)

	if piecesResponse.Total == 0 {
		return c.Send(messages.UserNoPieces, keyboard)
	}

	// Группируем детали по пазлам
	puzzlePieces := make(map[int][]*models.PuzzlePiece)
	for _, piece := range piecesResponse.Pieces {
		puzzlePieces[piece.PuzzleId] = append(puzzlePieces[piece.PuzzleId], piece)
	}

	message := messages.UserPiecesListHeader(piecesResponse.Total)
	for puzzleId, pieces := range puzzlePieces {
		message += messages.UserPuzzlePiecesInfo(puzzleId, len(pieces))
		for _, piece := range pieces {
			message += messages.UserPieceInfo(piece.PieceNumber, piece.Code)
		}
	}

	return c.Send(message, keyboard)
}

// handleText обрабатывает текстовые сообщения
func (ub *UserBot) handleText(c tele.Context) error {
	text := c.Text()

	// Проверяем, является ли сообщение кодом детали
	if isPieceCode(text) {
		return ub.handlePieceCode(c, text)
	}

	// Проверяем, находится ли пользователь в процессе регистрации
	state, inRegistration := ub.registrationStates[c.Sender().ID]
	if inRegistration {
		return ub.handleRegistrationStep(c, text, state)
	}

	// Если сообщение не является кодом детали или пользователь не в процессе регистрации
	user, err := ub.getUser(c.Sender().ID)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователя: %v", err)
		return c.Send(messages.ErrGeneral)
	}

	var keyboard *tele.ReplyMarkup
	if user != nil {
		keyboard = ub.createMainKeyboard(true)
	} else {
		keyboard = ub.createMainKeyboard(false)
	}

	// Проверяем, похоже ли на попытку ввода кода (короткая строка без пробелов)
	if looksLikeCodeAttempt(text) {
		return c.Send(messages.PieceInvalidCodeFormat, keyboard)
	}

	return c.Send(messages.UnknownMessage, keyboard)
}

// handleRegistrationStep обрабатывает шаги регистрации
func (ub *UserBot) handleRegistrationStep(c tele.Context, text string, state *RegistrationState) error {
	if text == "⏭️ Пропустить" {
		return ub.handleSkipButton(c)
	} else if text == "❌ Отменить" {
		return ub.handleCancelButton(c)
	}

	switch state.Step {
	case RegistrationStepLastName:
		state.LastName = text
		state.Step = RegistrationStepFirstName
		keyboard := ub.createRegistrationKeyboard(false)
		return c.Send(messages.RegEnterFirstName, keyboard)

	case RegistrationStepFirstName:
		state.FirstName = text
		state.Step = RegistrationStepMiddleName
		keyboard := ub.createRegistrationKeyboard(true)
		return c.Send(messages.RegEnterMiddleName, keyboard)

	case RegistrationStepMiddleName:
		state.MiddleName = text
		state.Step = RegistrationStepGroup
		keyboard := ub.createRegistrationKeyboard(false)
		return c.Send(messages.RegEnterGroup, keyboard)

	case RegistrationStepGroup:
		normalizedGroup, valid := NormalizeGroupName(text)
		if !valid {
			keyboard := ub.createRegistrationKeyboard(false)
			return c.Send(messages.InvalidGroupFormat, keyboard)
		}

		state.Group = normalizedGroup
		telegramID := fmt.Sprintf("%d", c.Sender().ID)

		// Проверяем, не зарегистрирован ли уже пользователь
		user, err := ub.getUser(c.Sender().ID)
		if err != nil {
			ub.logger.Errorf("Ошибка получения пользователя: %v", err)
			return c.Send(messages.ErrRegistration)
		}

		if user != nil {
			delete(ub.registrationStates, c.Sender().ID)
			return c.Send(messages.UserAlreadyRegisteredMsg(user.FirstName, user.LastName))
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
			return c.Send(messages.ErrRegistration)
		}

		delete(ub.registrationStates, c.Sender().ID)
		keyboard := ub.createMainKeyboard(true)

		return c.Send(messages.UserRegSuccess(state.LastName, state.FirstName, state.MiddleName, state.Group), keyboard)
	}

	return nil
}

// handlePieceCode обрабатывает код детали пазла
func (ub *UserBot) handlePieceCode(c tele.Context, code string) error {
	ub.logger.Infof("Пользователь %d отправил код детали: %s", c.Sender().ID, code)

	// Проверяем блокировку за спам
	if ub.isUserBlocked(c.Sender().ID) {
		remaining := ub.getBlockTimeRemaining(c.Sender().ID)
		return c.Send(messages.TooManyAttemptsMsg(remaining))
	}

	// Получаем пользователя
	user, err := ub.getUser(c.Sender().ID)
	if err != nil {
		ub.logger.Errorf("Ошибка получения пользователя: %v", err)
		return c.Send(messages.ErrRegisterPiece)
	}

	if user == nil {
		return c.Send(messages.UserNotRegisteredUseCommand)
	}

	// Нормализуем код (приводим к верхнему регистру)
	normalizedCode := normalizeCode(code)

	// Регистрируем деталь через API
	registerData, err := ub.apiClient.Post("/pieces/"+normalizedCode+"/register", map[string]interface{}{
		"user_id": user.Id,
	})

	if err != nil {
		ub.logger.Errorf("Ошибка регистрации детали через API: %v", err)
		return c.Send(messages.ErrRegisterPiece)
	}

	// Декодируем ответ
	var registerResponse struct {
		Success         bool                `json:"success"`
		Piece           *models.PuzzlePiece `json:"piece"`
		PuzzleCompleted bool                `json:"puzzle_completed"`
		Error           string              `json:"error"`
		ErrorCode       int                 `json:"error_code"`
	}
	if err := json.Unmarshal(registerData, &registerResponse); err != nil {
		ub.logger.Errorf("Ошибка декодирования ответа API: %v", err)
		return c.Send(messages.ErrRegisterPiece)
	}

	if !registerResponse.Success {
		// Обрабатываем ошибку в зависимости от кода
		switch registerResponse.ErrorCode {
		case models.PieceErrorNotFound:
			ub.recordFailedAttempt(c.Sender().ID)
			attemptsLeft := MaxFailedAttempts - ub.getFailedAttemptCount(c.Sender().ID)
			if attemptsLeft <= 0 {
				return c.Send(messages.PieceNotFoundBlockedMsg(int(BlockDuration.Minutes())))
			}
			return c.Send(messages.PieceNotFoundMsg(attemptsLeft))
		case models.PieceErrorAlreadyTaken:
			// Это не считается неудачной попыткой для анти-спама
			ub.clearFailedAttempts(c.Sender().ID)
			return c.Send(messages.PieceAlreadyRegistered)
		default:
			return c.Send(messages.PieceRegisterFailed)
		}
	}

	// Успешная регистрация - сбрасываем счетчик неудачных попыток
	ub.clearFailedAttempts(c.Sender().ID)

	keyboard := ub.createMainKeyboard(true)

	message := messages.PieceRegisteredSuccessMsg(
		registerResponse.Piece.PuzzleId,
		registerResponse.Piece.PieceNumber,
		registerResponse.PuzzleCompleted,
	)

	return c.Send(message, keyboard)
}

// getUser получает пользователя по Telegram ID
func (ub *UserBot) getUser(telegramID int64) (*models.User, error) {
	telegramIDStr := fmt.Sprintf("%d", telegramID)

	usersData, err := ub.apiClient.Get("/users", nil)
	if err != nil {
		return nil, err
	}

	var usersResponse struct {
		Total int            `json:"total"`
		Users []*models.User `json:"users"`
	}
	if err := json.Unmarshal(usersData, &usersResponse); err != nil {
		return nil, err
	}

	for _, u := range usersResponse.Users {
		if u.Telegramm == telegramIDStr && !u.Deleted {
			return u, nil
		}
	}

	return nil, nil
}

// isPieceCode проверяет, является ли строка кодом детали (7 символов A-Z, 0-9)
func isPieceCode(s string) bool {
	if len(s) != 7 {
		return false
	}
	matched, _ := regexp.MatchString(`^[A-Za-z0-9]{7}$`, s)
	return matched
}

// looksLikeCodeAttempt проверяет, похоже ли сообщение на попытку ввести код
// (короткая строка без пробелов, преимущественно буквы/цифры)
func looksLikeCodeAttempt(s string) bool {
	s = strings.TrimSpace(s)
	// Если содержит пробелы — это не код
	if strings.Contains(s, " ") {
		return false
	}
	// Если слишком длинная или слишком короткая — не похоже на код
	if len(s) < 4 || len(s) > 10 {
		return false
	}
	// Проверяем, что состоит в основном из букв и цифр
	alphanumCount := 0
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			alphanumCount++
		}
	}
	// Если больше половины символов — буквы/цифры, похоже на код
	return alphanumCount > len(s)/2
}

// normalizeCode приводит код к верхнему регистру и обрезает пробелы
func normalizeCode(code string) string {
	result := ""
	for _, c := range code {
		if c >= 'a' && c <= 'z' {
			result += string(c - 32)
		} else if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			result += string(c)
		}
	}
	return result
}

// isUserBlocked проверяет, заблокирован ли пользователь за спам
func (ub *UserBot) isUserBlocked(userID int64) bool {
	ub.attemptsMutex.RLock()
	defer ub.attemptsMutex.RUnlock()

	attempts, exists := ub.failedAttempts[userID]
	if !exists {
		return false
	}

	if attempts.BlockedAt == nil {
		return false
	}

	// Проверяем, истек ли срок блокировки
	if time.Since(*attempts.BlockedAt) > BlockDuration {
		return false
	}

	return true
}

// getBlockTimeRemaining возвращает оставшееся время блокировки
func (ub *UserBot) getBlockTimeRemaining(userID int64) time.Duration {
	ub.attemptsMutex.RLock()
	defer ub.attemptsMutex.RUnlock()

	attempts, exists := ub.failedAttempts[userID]
	if !exists || attempts.BlockedAt == nil {
		return 0
	}

	remaining := BlockDuration - time.Since(*attempts.BlockedAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// recordFailedAttempt записывает неудачную попытку
func (ub *UserBot) recordFailedAttempt(userID int64) {
	ub.attemptsMutex.Lock()
	defer ub.attemptsMutex.Unlock()

	attempts, exists := ub.failedAttempts[userID]
	if !exists {
		attempts = &FailedAttempts{}
		ub.failedAttempts[userID] = attempts
	}

	// Если блокировка истекла, сбрасываем счетчик
	if attempts.BlockedAt != nil && time.Since(*attempts.BlockedAt) > BlockDuration {
		attempts.Count = 0
		attempts.BlockedAt = nil
	}

	attempts.Count++
	attempts.LastTry = time.Now()

	// Если превышен лимит, блокируем
	if attempts.Count >= MaxFailedAttempts {
		now := time.Now()
		attempts.BlockedAt = &now
	}
}

// getFailedAttemptCount возвращает количество неудачных попыток
func (ub *UserBot) getFailedAttemptCount(userID int64) int {
	ub.attemptsMutex.RLock()
	defer ub.attemptsMutex.RUnlock()

	attempts, exists := ub.failedAttempts[userID]
	if !exists {
		return 0
	}

	// Если блокировка истекла, считаем что попыток 0
	if attempts.BlockedAt != nil && time.Since(*attempts.BlockedAt) > BlockDuration {
		return 0
	}

	return attempts.Count
}

// clearFailedAttempts сбрасывает счетчик неудачных попыток
func (ub *UserBot) clearFailedAttempts(userID int64) {
	ub.attemptsMutex.Lock()
	defer ub.attemptsMutex.Unlock()

	delete(ub.failedAttempts, userID)
}

// ==================== РАССЫЛКА УВЕДОМЛЕНИЙ ====================

// notificationPoller периодически проверяет очередь уведомлений и отправляет их
func (ub *UserBot) notificationPoller() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ub.stopChan:
			return
		case <-ticker.C:
			ub.processNotifications()
		}
	}
}

// processNotifications обрабатывает ожидающие уведомления
func (ub *UserBot) processNotifications() {
	// Получаем ожидающие уведомления с пользователями
	data, err := ub.apiClient.Get("/notifications/pending", nil)
	if err != nil {
		ub.logger.Errorf("Ошибка получения уведомлений: %v", err)
		return
	}

	var response struct {
		Total         int `json:"total"`
		Notifications []struct {
			Id          string   `json:"id"`
			Message     string   `json:"message"`
			Attachments []string `json:"attachments,omitempty"`
			Users       []struct {
				Telegramm string `json:"telegramm"`
			} `json:"users"`
		} `json:"notifications"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		ub.logger.Errorf("Ошибка декодирования уведомлений: %v", err)
		return
	}

	if response.Total == 0 {
		return
	}

	ub.logger.Infof("Найдено %d уведомлений для отправки", response.Total)

	for _, notification := range response.Notifications {
		sentCount := 0
		errorCount := 0

		for _, user := range notification.Users {
			if user.Telegramm == "" {
				errorCount++
				continue
			}

			telegramID, err := parseTelegramID(user.Telegramm)
			if err != nil {
				ub.logger.Errorf("Ошибка парсинга Telegram ID: %v", err)
				errorCount++
				continue
			}

			recipient := &tele.User{ID: telegramID}

			// Отправляем текстовое сообщение
			_, err = ub.bot.Send(recipient, notification.Message)
			if err != nil {
				ub.logger.Errorf("Ошибка отправки уведомления пользователю %d: %v", telegramID, err)
				errorCount++
				continue
			}

			// Отправляем вложения (attachment содержит полный путь к файлу в библиотеке)
			for _, filePath := range notification.Attachments {
				// Проверяем существование файла
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					ub.logger.Errorf("Файл вложения не найден: %s", filePath)
					continue
				}

				// Определяем тип по расширению
				ext := strings.ToLower(filepath.Ext(filePath))
				filename := filepath.Base(filePath)

				switch ext {
				case ".jpg", ".jpeg", ".png", ".gif":
					photo := &tele.Photo{File: tele.FromDisk(filePath)}
					_, err = ub.bot.Send(recipient, photo)
				default:
					doc := &tele.Document{
						File:     tele.FromDisk(filePath),
						FileName: filename,
					}
					_, err = ub.bot.Send(recipient, doc)
				}

				if err != nil {
					ub.logger.Errorf("Ошибка отправки вложения %s пользователю %d: %v", filePath, telegramID, err)
				}

				time.Sleep(50 * time.Millisecond) // Задержка между вложениями
			}

			sentCount++
			time.Sleep(50 * time.Millisecond) // Небольшая задержка между пользователями
		}

		// Обновляем статус уведомления
		status := "sent"
		if errorCount > 0 && sentCount == 0 {
			status = "failed"
		}

		updateData := map[string]interface{}{
			"status":      status,
			"sent_count":  sentCount,
			"error_count": errorCount,
		}

		_, err := ub.apiClient.Patch("/notifications/"+notification.Id, updateData)
		if err != nil {
			ub.logger.Errorf("Ошибка обновления статуса уведомления: %v", err)
		} else {
			ub.logger.Infof("Уведомление %s обработано: отправлено %d, ошибок %d", notification.Id, sentCount, errorCount)
		}
	}
}

// parseTelegramID парсит Telegram ID из строки
func parseTelegramID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}
