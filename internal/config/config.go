package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

const (
	minConfigurePort  = 1
	maxTCPPort        = 65535
	defaultDBMaxConns = "5"
)

type Config struct {
	Port        string
	DatabaseURL string
	DBMaxConns  int32
	LogLevel    slog.Level
}

func Load() (Config, error) {
	cfg := Config{
		Port:        env("PORT", "8080"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < minConfigurePort || port > maxTCPPort {
		return Config{}, errors.New("PORT must be between 1 and 65535")
	}

	maxConns, err := strconv.ParseInt(env("DB_MAX_CONNS", defaultDBMaxConns), 10, 32)
	if err != nil || maxConns < 1 {
		return Config{}, errors.New("DB_MAX_CONNS must be a positive 32-bit integer")
	}

	cfg.maxConns = int32(maxConns)

	if err := cfg.LogLevel.UnmarshalText([]byte(env("LOG_LEVEL", "INFO"))); err != nil {
		return Config{}, fmt.Errorf("invalid LOG_LEVEL: %w", err)
	}

	return cfg, nil
}

func env(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	return value
}
