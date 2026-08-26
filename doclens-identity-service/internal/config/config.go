package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address             string
	JWTSecret           string
	Environment         string
	DatabaseURL         string
	DatabaseTimeout     time.Duration
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	DevelopmentSeed     bool
	DevelopmentOrgID    string
	DevelopmentEmail    string
	DevelopmentPassword string
}

func Load() (Config, error) {
	cfg := Config{
		Address:             env("IDENTITY_GRPC_ADDR", ":9001"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		Environment:         strings.ToLower(strings.TrimSpace(env("IDENTITY_ENV", "development"))),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		DatabaseTimeout:     durationEnv("DATABASE_TIMEOUT", 5*time.Second),
		AccessTokenTTL:      durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:     durationEnv("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		DevelopmentSeed:     boolEnv("IDENTITY_DEV_SEED", true),
		DevelopmentOrgID:    env("IDENTITY_DEV_ORG_ID", "dev-org"),
		DevelopmentEmail:    env("IDENTITY_DEV_EMAIL", "admin@doclens.local"),
		DevelopmentPassword: env("IDENTITY_DEV_PASSWORD", "doclens-dev"),
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	if cfg.DatabaseURL == "" && cfg.Environment != "development" {
		return Config{}, errors.New("DATABASE_URL is required outside development")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
