package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/MrPunder/sirius-loyality-system/internal/config"
	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
	"github.com/MrPunder/sirius-loyality-system/internal/telegrambot"
)

func main() {
	// Парсим флаги командной строки
	var (
		configPath string
		token      string
		serverURL  string
		adminIDStr string
	)

	flag.StringVar(&configPath, "c", "cmd/loyalityserver/config.yaml", "config path")
	flag.StringVar(&token, "token", "", "telegram bot token")
	flag.StringVar(&serverURL, "server", "http://localhost:8080", "server URL")
	flag.StringVar(&adminIDStr, "admin", "", "admin user ID (используется только при первом запуске, если файл с администраторами не существует)")
	flag.Parse()

	// Загружаем конфигурацию
	conf, err := config.LoadConfig()
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
		zapLogger.Error("Не указан токен бота. Используйте флаг -token")
		os.Exit(1)
	}

	// Создаем конфигурацию бота
	botConfig := telegrambot.Config{
		Token:     token,
		ServerURL: serverURL,
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
}
