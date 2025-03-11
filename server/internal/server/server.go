package server

import (
	"fmt"
	"net/http"

	"github.com/MrPunder/sirius-loyality-system/internal/config"
	"github.com/MrPunder/sirius-loyality-system/internal/logger"
)

// Server представляет базовую структуру нашего сервера
type Server struct {
	address string
	log     logger.Logger
}

// NewServer создает новый экземпляр Server
func NewServer(conf config.ServerConfig, log logger.Logger) *Server {
	return &Server{address: conf.RunAddress, log: log}
}

// Run запускает HTTP сервер на указанном порту
func (s *Server) Run() error {
	// Установка маршрутов
	http.HandleFunc("/", s.homeHandler)

	// Запуск сервера
	if err := http.ListenAndServe(s.address, nil); err != nil {
		return err
	}
	return nil
}

// homeHandler обрабатывает запросы к корневому маршруту "/"
func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Welcome to my HTTP Server!")
}
