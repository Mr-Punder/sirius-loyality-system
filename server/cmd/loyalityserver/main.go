package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/MrPunder/sirius-loyality-system/internal/admin"
	"github.com/MrPunder/sirius-loyality-system/internal/config"
	"github.com/MrPunder/sirius-loyality-system/internal/handlers"
	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/loyalityserver"
	"github.com/MrPunder/sirius-loyality-system/internal/middleware"
	"github.com/MrPunder/sirius-loyality-system/internal/storage"
)

func main() {
	conf, err := config.LoadConfig("")
	if err != nil {
		panic(err)
	}
	log, err := logger.NewZapLogger(conf)
	if err != nil {
		panic(err)
	}
	log.Info("Initialized logger")

	env := os.Environ()
	log.Infof("Env values: %s", env)
	log.Infof("Config parametrs: %s", *conf)

	// Инициализация хранилища
	var store storage.Storage
	var storeErr error

	switch conf.Storage.Type {
	case "file":
		log.Info("Initializing file storage")
		store, storeErr = storage.NewFilestorage(conf.Storage.DataPath)
	case "postgres":
		log.Info("Initializing PostgreSQL storage")
		store, storeErr = storage.NewPgStorage(conf.Storage.ConnectionString, conf.Storage.MigrationsPath)
	case "sqlite":
		log.Info("Initializing SQLite storage")
		store, storeErr = storage.NewSQLiteStorage(conf.Storage.DBPath, conf.Storage.MigrationsPath)
	default:
		log.Info("Initializing memory storage")
		store = storage.NewMemstorage()
	}

	if storeErr != nil {
		log.Errorf("Failed to initialize storage: %v", storeErr)
		panic(storeErr)
	}

	// Закрываем хранилище при завершении работы
	defer func() {
		switch s := store.(type) {
		case *storage.PgStorage:
			log.Info("Closing PostgreSQL connection")
			s.Close()
		case *storage.SQLiteStorage:
			log.Info("Closing SQLite connection")
			s.Close()
		}
	}()

	log.Info("Storage initialized successfully")

	// Инициализация обработчиков API
	router := handlers.NewRouter(log, store)

	// Инициализация обработчиков админки
	var dataPath string
	if conf.Storage.Type == "sqlite" {
		// Для SQLite используем директорию, содержащую файл базы данных
		dataPath = filepath.Dir(conf.Storage.DBPath)
	} else {
		dataPath = conf.Storage.DataPath
	}
	adminHandler := admin.NewAdminHandler(store, log, dataPath, conf.Admin.JWTSecret)
	adminHandler.RegisterRoutes(router)
	log.Info("Admin handlers initialized")

	// Инициализация сервера
	lsserver := loyalityserver.NewLoyalityServer(conf.Server.RunAddress, router, log)

	hLogger := middleware.NewHTTPLoger(log)
	log.Info("Initialized middleware functions")

	// Инициализация middleware для проверки токена API
	tokenAuth := middleware.NewTokenAuth(middleware.TokenAuthConfig{
		APIToken: conf.API.Token,
		Logger:   log,
	})
	log.Info("Initialized token auth middleware")

	lsserver.AddMidleware(hLogger.HTTPLogHandler, tokenAuth.Middleware)
	// lsserver.AddMidleware(hLogger.HTTPLogHandler, tokenAuth.Middleware)

	go lsserver.RunServer()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("Initialized shutdown")
	if err := lsserver.Shutdown(context.Background()); err != nil {
		log.Errorf("Cann't stop server %s", err)
	}

	// Закрываем логгер перед завершением программы
	if err := log.Close(); err != nil {
		panic(err)
	}
}
