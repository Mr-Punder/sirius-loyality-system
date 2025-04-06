package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/MrPunder/sirius-loyality-system/internal/config"
	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/MrPunder/sirius-loyality-system/internal/telegrambot"
)

func main() {
	// Парсим флаги командной строки
	var (
		token        string
		serverURL    string
		adminIDStr   string
		tokenPath    string
		apiToken     string
		apiTokenPath string
		configPath   string
	)

	flag.StringVar(&token, "token", "", "telegram bot token (deprecated, use token.txt file instead)")
	flag.StringVar(&serverURL, "server", "http://localhost:8080", "server URL")
	flag.StringVar(&adminIDStr, "admin", "", "admin user ID (используется только при первом запуске, если файл с администраторами не существует)")
	flag.StringVar(&tokenPath, "token-file", "cmd/telegrambot/admin/token.txt", "path to file with telegram bot token")
	flag.StringVar(&apiToken, "api-token", "", "API token for server authentication")
	flag.StringVar(&apiTokenPath, "api-token-file", "cmd/telegrambot/api_token.txt", "path to file with API token")
	flag.StringVar(&configPath, "config", "cmd/loyalityserver/config.yaml", "path to config")

	// Проверяем наличие переменных окружения и используем их, если они заданы
	if envTokenPath := os.Getenv("TOKEN_PATH"); envTokenPath != "" {
		tokenPath = envTokenPath
	}

	if envServerURL := os.Getenv("SERVER_URL"); envServerURL != "" {
		serverURL = envServerURL
	}

	if envApiTokenPath := os.Getenv("API_TOKEN_PATH"); envApiTokenPath != "" {
		apiTokenPath = envApiTokenPath
	}

	if envAdminsPath := os.Getenv("ADMINS_PATH"); envAdminsPath != "" {
		// Для использования в будущем, если понадобится
	}

	if envConfigPath := os.Getenv("CONFIG_PATH"); envConfigPath == "" {
		configPath = envConfigPath
	}

	flag.Parse()

	// Если токен не указан через флаг, пытаемся прочитать его из файла
	if token == "" {
		log.Printf("Чтение токена из файла: %s", tokenPath)
		tokenData, err := os.ReadFile(tokenPath)
		if err != nil {
			log.Fatalf("Ошибка чтения токена из файла: %v", err)
		}
		token = strings.TrimSpace(string(tokenData))
	}

	// Если API-токен не указан через флаг, пытаемся прочитать его из файла
	if apiToken == "" {
		log.Printf("Чтение API-токена из файла: %s", apiTokenPath)
		apiTokenData, err := os.ReadFile(apiTokenPath)
		if err != nil {
			log.Fatalf("Ошибка чтения API-токена из файла: %v", err)
		}
		apiToken = strings.TrimSpace(string(apiTokenData))
	}

	// Загружаем конфигурацию
	conf, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Инициализируем логгер
	zapLogger, err := logger.NewZapLogger(conf)
	if err != nil {
		log.Fatalf("Ошибка инициализации логгера: %v", err)
	}
	zapLogger.Info("Инициализирован логгер")

	// Инициализируем хранилище
	var store storage.Storage
	var storeErr error

	switch conf.Storage.Type {
	case "sqlite":
		zapLogger.Info("Инициализация SQLite хранилища")
		store, storeErr = storage.NewSQLiteStorage(conf.Storage.DBPath, conf.Storage.MigrationsPath)
	case "postgres":
		zapLogger.Info("Инициализация PostgreSQL хранилища")
		store, storeErr = storage.NewPgStorage(conf.Storage.ConnectionString, conf.Storage.MigrationsPath)
	case "file":
		zapLogger.Info("Инициализация файлового хранилища")
		store, storeErr = storage.NewFilestorage(conf.Storage.DataPath)
	default:
		zapLogger.Info("Инициализация хранилища в памяти")
		store = storage.NewMemstorage()
	}

	if storeErr != nil {
		zapLogger.Errorf("Ошибка инициализации хранилища: %v", storeErr)
		os.Exit(1)
	}

	zapLogger.Info("Хранилище инициализировано успешно")

	// Проверяем, указан ли токен
	if token == "" {
		zapLogger.Error("Не указан токен бота. Используйте файл token.txt или флаг -token")
		os.Exit(1)
	}

	// Создаем конфигурацию бота
	botConfig := telegrambot.Config{
		Token:     token,
		ServerURL: serverURL,
		APIToken:  apiToken,
	}

	// Если указан ID администратора, добавляем его в конфигурацию
	if adminIDStr != "" {
		adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
		if err != nil {
			zapLogger.Errorf("Неверный формат ID администратора: %v", err)
			os.Exit(1)
		}
		botConfig.AdminUserID = adminID
	}

	// Создаем бота
	bot, err := telegrambot.NewAdminBot(botConfig, store, zapLogger)
	if err != nil {
		zapLogger.Errorf("Ошибка создания бота: %v", err)
		os.Exit(1)
	}

	// Запускаем бота
	if err := bot.Start(); err != nil {
		zapLogger.Errorf("Ошибка запуска бота: %v", err)
		os.Exit(1)
	}

	zapLogger.Info("Бот запущен")

	// Ожидаем сигнала завершения
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	zapLogger.Info("Получен сигнал завершения")

	// Останавливаем бота
	if err := bot.Stop(); err != nil {
		zapLogger.Errorf("Ошибка остановки бота: %v", err)
	}

	zapLogger.Info("Бот остановлен")

	// Закрываем логгер перед завершением программы
	if err := zapLogger.Close(); err != nil {
		log.Fatalf("Ошибка закрытия логгера: %v", err)
	}
}
