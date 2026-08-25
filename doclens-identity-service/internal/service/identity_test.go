package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/doclens/identity-service/internal/auth"
	identityv1 "github.com/doclens/identity-service/internal/gen/doclens/identity/v1"
	"github.com/doclens/identity-service/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateUserAndLogin(t *testing.T) {
	svc := NewIdentity(store.NewMemory(), auth.NewTokenIssuer("test-secret"), time.Minute, time.Hour)

	user, err := svc.CreateUser(context.Background(), &identityv1.CreateUserRequest{
		OrganizationId: "org-1",
		Email:          "Admin@Example.com",
		Password:       "super-secret",
		Roles:          []string{"Admin", "admin", " reviewer "},
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if user.GetEmail() != "admin@example.com" {
		t.Fatalf("normalized email = %q", user.GetEmail())
	}
	if got := strings.Join(user.GetRoles(), ","); got != "admin,reviewer" {
		t.Fatalf("roles = %q", got)
	}

	resp, err := svc.Login(context.Background(), &identityv1.LoginRequest{
		Email:    "admin@example.com",
		Password: "super-secret",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if resp.GetAccessToken() == "" || resp.GetRefreshToken() == "" {
		t.Fatalf("expected access and refresh tokens")
	}
	if resp.GetUser().GetId() != user.GetId() {
		t.Fatalf("login user id = %q, want %q", resp.GetUser().GetId(), user.GetId())
	}
}

func TestCreateUserRejectsDuplicateEmail(t *testing.T) {
	svc := NewIdentity(store.NewMemory(), auth.NewTokenIssuer("test-secret"), time.Minute, time.Hour)
	req := &identityv1.CreateUserRequest{
		OrganizationId: "org-1",
		Email:          "dupe@example.com",
		Password:       "super-secret",
	}
	if _, err := svc.CreateUser(context.Background(), req); err != nil {
		t.Fatalf("CreateUser() first error = %v", err)
	}
	_, err := svc.CreateUser(context.Background(), req)
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("CreateUser() duplicate code = %v, want %v", status.Code(err), codes.AlreadyExists)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	svc := NewIdentity(store.NewMemory(), auth.NewTokenIssuer("test-secret"), time.Minute, time.Hour)
	_, err := svc.CreateUser(context.Background(), &identityv1.CreateUserRequest{
		OrganizationId: "org-1",
		Email:          "admin@example.com",
		Password:       "super-secret",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	_, err = svc.Login(context.Background(), &identityv1.LoginRequest{
		Email:    "admin@example.com",
		Password: "not-it",
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("Login() code = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
}
