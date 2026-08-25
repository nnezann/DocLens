package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address              string
	JWTSecret            string
	AuthDisabled         bool
	RequestTimeout       time.Duration
	RateLimitRPS         int
	RateLimitBurst       int
	PublicRateLimitRPS   int
	PublicRateLimitBurst int
	IdentityTarget       string
	DocumentsTarget      string
	VerificationTarget   string
	GRPCInsecure         bool
}

func Load() (Config, error) {
	cfg := Config{
		Address:              env("GATEWAY_ADDR", ":8080"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		AuthDisabled:         envBool("GATEWAY_AUTH_DISABLED", false),
		RequestTimeout:       envDuration("REQUEST_TIMEOUT", 10*time.Second),
		RateLimitRPS:         envInt("RATE_LIMIT_RPS", 20),
		RateLimitBurst:       envInt("RATE_LIMIT_BURST", 40),
		PublicRateLimitRPS:   envInt("PUBLIC_RATE_LIMIT_RPS", 5),
		PublicRateLimitBurst: envInt("PUBLIC_RATE_LIMIT_BURST", 10),
		IdentityTarget:       env("IDENTITY_GRPC_ADDR", "localhost:9001"),
		DocumentsTarget:      env("DOCUMENTS_GRPC_ADDR", "localhost:9002"),
		VerificationTarget:   env("VERIFICATION_GRPC_ADDR", "localhost:9003"),
		GRPCInsecure:         envBool("GRPC_INSECURE", true),
	}
	if !cfg.AuthDisabled && cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required unless GATEWAY_AUTH_DISABLED=true")
	}
	if cfg.RateLimitRPS <= 0 || cfg.RateLimitBurst <= 0 || cfg.PublicRateLimitRPS <= 0 || cfg.PublicRateLimitBurst <= 0 {
		return Config{}, errors.New("rate limit values must be positive")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
