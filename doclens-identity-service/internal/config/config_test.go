package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadAllowsMissingDatabaseOnlyInDevelopment(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("IDENTITY_ENV", "development")
	t.Setenv("DATABASE_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
}

func TestLoadRejectsMissingDatabaseOutsideDevelopment(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("IDENTITY_ENV", "production")
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("Load() error = %v, want missing DATABASE_URL error", err)
	}
}

func TestLoadDatabaseSettings(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("IDENTITY_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("DATABASE_TIMEOUT", "2s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseTimeout != 2*time.Second {
		t.Fatalf("DatabaseTimeout = %s, want 2s", cfg.DatabaseTimeout)
	}
}
