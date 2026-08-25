package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address             string
	MetricsAddress      string
	DatabaseURL         string
	StorageDir          string
	R2Endpoint          string
	R2AccessKeyID       string
	R2SecretAccessKey   string
	R2Bucket            string
	RabbitMQURL         string
	RabbitMQExchange    string
	MaxUploadBytes      int64
	AllowedContentTypes []string
	UploadWorkers       int
	UploadTenantRate    float64
	UploadTenantBurst   int
	UploadFailureLimit  int
	UploadCircuitOpen   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Address:             getenv("DOCUMENT_INTAKE_GRPC_ADDR", ":9002"),
		MetricsAddress:      getenv("DOCUMENT_INTAKE_METRICS_ADDR", ":9092"),
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		StorageDir:          getenv("DOCUMENT_INTAKE_STORAGE_DIR", "./data/documents"),
		R2Endpoint:          strings.TrimSpace(os.Getenv("R2_ENDPOINT")),
		R2AccessKeyID:       strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID")),
		R2SecretAccessKey:   strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY")),
		R2Bucket:            strings.TrimSpace(os.Getenv("R2_BUCKET")),
		RabbitMQURL:         strings.TrimSpace(os.Getenv("RABBITMQ_URL")),
		RabbitMQExchange:    getenv("RABBITMQ_EXCHANGE", "doclens.events"),
		MaxUploadBytes:      int64Env("DOCUMENT_INTAKE_MAX_UPLOAD_BYTES", 10*1024*1024),
		AllowedContentTypes: parseCSV(getenv("DOCUMENT_INTAKE_ALLOWED_CONTENT_TYPES", "application/pdf,image/jpeg,image/png,image/webp")),
		UploadWorkers:       intEnv("DOCUMENT_INTAKE_UPLOAD_WORKERS", 8),
		UploadTenantRate:    floatEnv("DOCUMENT_INTAKE_TENANT_UPLOAD_RATE", 5),
		UploadTenantBurst:   intEnv("DOCUMENT_INTAKE_TENANT_UPLOAD_BURST", 10),
		UploadFailureLimit:  intEnv("DOCUMENT_INTAKE_STORAGE_FAILURE_THRESHOLD", 5),
		UploadCircuitOpen:   durationEnv("DOCUMENT_INTAKE_STORAGE_CIRCUIT_OPEN", time.Second*30),
	}
	if cfg.Address == "" {
		cfg.Address = ":9002"
	}

	if len(cfg.AllowedContentTypes) == 0 {
		cfg.AllowedContentTypes = []string{"application/pdf", "image/jpeg", "image/png", "image/webp"}
	}
	return cfg, nil
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func floatEnv(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
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
