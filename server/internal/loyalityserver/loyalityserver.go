package loyalityserver

import (
	"context"
	"net/http"

	"github.com/MrPunder/sirius-loyality-system/internal/logger"
)

type middlewareFunc func(next http.Handler) http.Handler

type LoyalityServer struct {
	Log        logger.Logger
	middlwares []middlewareFunc
	mux        http.Handler
	address    string
	server     *http.Server
}

func NewLoyalityServer(adress string, mux http.Handler, Log logger.Logger) *LoyalityServer {

	return &LoyalityServer{
		address: adress,
		mux:     mux,
		Log:     Log,
	}
}

func (ls *LoyalityServer) AddMidleware(funcs ...middlewareFunc) {
	ls.middlwares = append(ls.middlwares, funcs...)
}

func (ls *LoyalityServer) RunServer() {
	handler := ls.mux

	for _, f := range ls.middlwares {
		handler = f(handler)
	}

	ls.server = &http.Server{
		Addr:    ls.address,
		Handler: handler,
	}
	ls.Log.Infof("Starting server on %s", ls.address)
	if err := ls.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		ls.Log.Errorf("starting server on %s error: %s", ls.address, err)
	}

}

func (ls *LoyalityServer) Shutdown(ctx context.Context) error {
	return ls.server.Shutdown(ctx)
}
