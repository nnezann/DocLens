package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/doclens/identity-service/internal/auth"
	identityv1 "github.com/doclens/identity-service/internal/gen/doclens/identity/v1"
	"github.com/doclens/identity-service/internal/domain"
	"github.com/doclens/identity-service/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Identity struct {
	identityv1.UnimplementedIdentityServiceServer

	store           store.Store
	issuer          auth.TokenIssuer
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewIdentity(store store.Store, issuer auth.TokenIssuer, accessTokenTTL, refreshTokenTTL time.Duration) *Identity {
	return &Identity{
		store:           store,
		issuer:          issuer,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (s *Identity) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.User, error) {
	if req.GetOrganizationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "organization_id is required")
	}
	email, err := normalizeEmail(req.GetEmail())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "valid email is required")
	}
	passwordHash, err := auth.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	roles := cleanRoles(req.GetRoles())
	if len(roles) == 0 {
		roles = []string{"member"}
	}
	user := domain.User{
		ID:             newID("usr"),
		OrganizationID: strings.TrimSpace(req.GetOrganizationId()),
		Email:          email,
		PasswordHash:   passwordHash,
		Roles:          roles,
		CreatedAt:      time.Now().UTC(),
	}
	created, err := s.store.CreateUser(ctx, user)
	if errors.Is(err, store.ErrUserExists) {
		return nil, status.Error(codes.AlreadyExists, "user already exists")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "create user")
	}
	return toProtoUser(created), nil
}

func (s *Identity) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.LoginResponse, error) {
	email, err := normalizeEmail(req.GetEmail())
	if err != nil || req.GetPassword() == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}
	user, err := s.store.UserByEmail(ctx, email)
	if err != nil || user.Disabled || !auth.VerifyPassword(user.PasswordHash, req.GetPassword()) {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}
	accessToken, err := s.issuer.AccessToken(user.ID, user.OrganizationID, user.Roles, s.accessTokenTTL)
	if err != nil {
		return nil, status.Error(codes.Internal, "issue access token")
	}
	refreshToken, err := auth.RefreshToken()
	if err != nil {
		return nil, status.Error(codes.Internal, "issue refresh token")
	}
	now := time.Now().UTC()
	if err := s.store.SaveRefreshToken(ctx, domain.RefreshToken{
		Token:     refreshToken,
		UserID:    user.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(s.refreshTokenTTL),
	}); err != nil {
		return nil, status.Error(codes.Internal, "save refresh token")
	}
	return &identityv1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toProtoUser(user),
	}, nil
}

func SeedDevelopmentUser(ctx context.Context, svc *Identity, orgID, email, password string) error {
	_, err := svc.CreateUser(ctx, &identityv1.CreateUserRequest{
		OrganizationId: orgID,
		Email:          email,
		Password:       password,
		Roles:          []string{"admin"},
	})
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	return err
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return "", err
	}
	return email, nil
}

func cleanRoles(roles []string) []string {
	seen := map[string]struct{}{}
	cleaned := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.ToLower(strings.TrimSpace(role))
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		cleaned = append(cleaned, role)
	}
	return cleaned
}

func toProtoUser(user domain.User) *identityv1.User {
	return &identityv1.User{
		Id:             user.ID,
		OrganizationId: user.OrganizationID,
		Email:          user.Email,
		Roles:          append([]string(nil), user.Roles...),
		Disabled:       user.Disabled,
		CreatedAt:      user.CreatedAt.Format(time.RFC3339),
	}
}

func newID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "_" + hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
