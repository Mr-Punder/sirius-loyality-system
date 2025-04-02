package main

import (
	"context"
	"os"
	"os/signal"
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
	conf, err := config.LoadConfig()
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
	default:
		log.Info("Initializing memory storage")
		store = storage.NewMemstorage()
	}

	if storeErr != nil {
		log.Errorf("Failed to initialize storage: %v", storeErr)
		panic(storeErr)
	}

	log.Info("Storage initialized successfully")

	// Инициализация обработчиков API
	router := handlers.NewRouter(log, store)

	// Инициализация обработчиков админки
	adminHandler := admin.NewAdminHandler(store, log, conf.Storage.DataPath, conf.Admin.JWTSecret)
	adminHandler.RegisterRoutes(router)
	log.Info("Admin handlers initialized")

	// Инициализация сервера
	lsserver := loyalityserver.NewLoyalityServer(conf.Server.RunAddress, router, log)

	// Инициализация middleware
	comp := middleware.NewGzipCompressor(log)
	log.Info("Initialized compressor")

	hLogger := middleware.NewHTTPLoger(log)
	log.Info("Initialized middleware functions")

	lsserver.AddMidleware(comp.CompressHandler, hLogger.HTTPLogHandler)

	go lsserver.RunServer()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("Initialized shutdown")
	if err := lsserver.Shutdown(context.Background()); err != nil {
		log.Errorf("Cann't stop server %s", err)
	}

}
