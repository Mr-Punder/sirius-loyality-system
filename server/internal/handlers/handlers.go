package handlers

import (
	"net/http"
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	logger  logger.Logger
	timeout time.Duration
}

func NewHandler(logger logger.Logger) *Handler {
	return &Handler{logger, 3 * time.Second}
}

func NewRouter(logger logger.Logger) chi.Router {
	r := chi.NewRouter()

	handler := NewHandler(logger)

	return r.Route("/", func(r chi.Router) {
		r.Get("/ping", handler.PingHandler)
		r.Get("/{}", handler.DefoultHandler)
		r.Post("/{}", handler.DefoultHandler)
	})

}

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Entered PingHandler")

	if r.Method != http.MethodGet {
		h.logger.Error("wrong request method")
		http.Error(w, "Only GET requests are allowed for ping!", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("Method checked")

	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("pong"))
	if err != nil {
		h.logger.Errorf("Error writing response $v", err)
	}
	h.logger.Info("PingHandler exited")
}

// DefoultHandler for incorrect requests
func (h *Handler) DefoultHandler(w http.ResponseWriter, r *http.Request) {

	h.logger.Info("Entered DefoultHandler")

	http.Error(w, "wrong requests", http.StatusBadRequest)

}
