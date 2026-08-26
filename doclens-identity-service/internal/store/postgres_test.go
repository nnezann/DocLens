package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/doclens/identity-service/internal/domain"
)

func TestPostgresPersistence(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	postgres, err := NewPostgres(ctx, databaseURL, 5*time.Second)
	if err != nil {
		t.Fatalf("NewPostgres() error = %v", err)
	}
	defer postgres.Close()
	if err := postgres.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	user := domain.User{
		ID:             "test_" + time.Now().UTC().Format("20060102150405.000000000"),
		OrganizationID: "test-org",
		Email:          "Persistence-Test@Example.com",
		PasswordHash:   "hash",
		Roles:          []string{"member"},
		CreatedAt:      time.Now().UTC(),
	}
	created, err := postgres.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	found, err := postgres.UserByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("UserByEmail() error = %v", err)
	}
	if found.ID != created.ID || found.Email != "persistence-test@example.com" {
		t.Fatalf("found user = %#v", found)
	}
	if err := postgres.SaveRefreshToken(ctx, domain.RefreshToken{
		Token:     "test-refresh-token-" + user.ID,
		UserID:    user.ID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveRefreshToken() error = %v", err)
	}
}

func TestPostgresRequiresDatabaseURL(t *testing.T) {
	if _, err := NewPostgres(context.Background(), "", time.Second); err == nil {
		t.Fatal("NewPostgres() succeeded without a database URL")
	}
}
