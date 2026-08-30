package main

import (
	"log/slog"
	"net/http"
	"os"

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

	slog.Info("gateway starting", "addr", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
