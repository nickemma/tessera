package logging

import (
	"log/slog"
	"os"
)

func New(level string) *slog.Logger {
	var slogLevel slog.Level
	if err := slogLevel.UnmarshalText([]byte(level)); err != nil {
		slogLevel = slog.LevelInfo
	}

	options := &slog.HandlerOptions{Level: slogLevel}
	return slog.New(slog.NewJSONHandler(os.Stdout, options))
}
