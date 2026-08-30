package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nickemma/tessera/internal/app"
	"github.com/nickemma/tessera/internal/platform/config"
	"github.com/nickemma/tessera/internal/platform/logging"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.LogLevel)
	slog.SetDefault(logger)

	application, err := app.NewWithConfig(cfg, logger)
	if err != nil {
		logger.Error("application initialization failed", "err", err)
		os.Exit(1)
	}
	defer application.Close()
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           application.Handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownContext.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "err", err)
		}
	}()

	logger.Info("gateway starting", "addr", cfg.Addr)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("gateway stopped unexpectedly", "err", err)
		os.Exit(1)
	}
	logger.Info("gateway stopped")
}
