package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/MrPunder/sirius-loyality-system/internal/config"
	"github.com/MrPunder/sirius-loyality-system/internal/handlers"
	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/MrPunder/sirius-loyality-system/internal/loyalityserver"
	"github.com/MrPunder/sirius-loyality-system/internal/middleware"
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

	router := handlers.NewRouter(log)
	lsserver := loyalityserver.NewLoyalityServer(conf.Server.RunAddress, router, log)

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
