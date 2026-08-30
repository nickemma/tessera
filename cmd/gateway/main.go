package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/nickemma/tessera/internal/chat"
	"github.com/nickemma/tessera/internal/config"
	"github.com/nickemma/tessera/internal/httpapi"
)

func main() {
	cfg := config.Load()
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	router := httpapi.NewRouter(
		httpapi.NewChatHandler(chat.NewService()),
	)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	slog.Info("gateway starting", "addr", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
