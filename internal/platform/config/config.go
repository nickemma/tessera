package config

import "os"

type Config struct {
	Addr              string
	LogLevel          string
	Storage           string
	DatabaseURL       string
	RedisURL          string
	ModelURL          string
	ModelName         string
	MaxConcurrent     int
	RateLimitPerSec   int
	RateLimitBurst    int
	TenantBudget      int
	RequestTimeoutSec int
	CostPerMillion    float64
}

func Load() Config {
	return Config{
		Addr:              env("TESSERA_ADDR", ":8080"),
		LogLevel:          env("TESSERA_LOG_LEVEL", "info"),
		Storage:           env("TESSERA_STORAGE", "memory"),
		DatabaseURL:       env("TESSERA_DATABASE_URL", ""),
		RedisURL:          env("TESSERA_REDIS_URL", ""),
		ModelURL:          env("TESSERA_MODEL_URL", ""),
		ModelName:         env("TESSERA_MODEL_NAME", "canned-local"),
		MaxConcurrent:     envInt("TESSERA_MAX_CONCURRENT", 32),
		RateLimitPerSec:   envInt("TESSERA_RATE_LIMIT_PER_SEC", 10),
		RateLimitBurst:    envInt("TESSERA_RATE_LIMIT_BURST", 20),
		TenantBudget:      envInt("TESSERA_TENANT_BUDGET", 5000),
		RequestTimeoutSec: envInt("TESSERA_REQUEST_TIMEOUT_SEC", 60),
		CostPerMillion:    envFloat("TESSERA_COST_PER_MILLION", 0),
	}
}

func envFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var whole, fraction float64
	divisor := 1.0
	seenDecimal := false
	for _, digit := range value {
		if digit == '.' && !seenDecimal {
			seenDecimal = true
			continue
		}
		if digit < '0' || digit > '9' {
			return fallback
		}
		if seenDecimal {
			divisor *= 10
			fraction = fraction*10 + float64(digit-'0')
		} else {
			whole = whole*10 + float64(digit-'0')
		}
	}
	return whole + fraction/divisor
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var parsed int
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return fallback
		}
		parsed = parsed*10 + int(digit-'0')
	}
	if parsed <= 0 {
		return fallback
	}
	return parsed
}
