package config

import "os"

type Config struct {
	Addr     string
	LogLevel string
}

func Load() Config {
	return Config{
		Addr:     env("TESSERA_ADDR", ":8080"),
		LogLevel: env("TESSERA_LOG_LEVEL", "info"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
