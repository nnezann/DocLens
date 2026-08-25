package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Address             string
	StorageDir          string
	MaxUploadBytes      int64
	AllowedContentTypes []string
}

func Load() (Config, error) {
	cfg := Config{
		Address:             getenv("DOCUMENT_INTAKE_GRPC_ADDR", ":9002"),
		StorageDir:          getenv("DOCUMENT_INTAKE_STORAGE_DIR", "./data/documents"),
		MaxUploadBytes:      int64Env("DOCUMENT_INTAKE_MAX_UPLOAD_BYTES", 10*1024*1024),
		AllowedContentTypes: parseCSV(getenv("DOCUMENT_INTAKE_ALLOWED_CONTENT_TYPES", "application/pdf,image/jpeg,image/png,image/webp")),
	}
	if cfg.Address == "" {
		cfg.Address = ":9002"
	}
	if len(cfg.AllowedContentTypes) == 0 {
		cfg.AllowedContentTypes = []string{"application/pdf", "image/jpeg", "image/png", "image/webp"}
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func int64Env(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, strings.ToLower(trimmed))
	}
	return result
}
