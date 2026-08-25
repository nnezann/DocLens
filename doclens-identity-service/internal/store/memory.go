package store

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/doclens/identity-service/internal/domain"
)

var (
	ErrUserExists = errors.New("user already exists")
	ErrNotFound   = errors.New("not found")
)

type Store interface {
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	UserByEmail(ctx context.Context, email string) (domain.User, error)
	SaveRefreshToken(ctx context.Context, token domain.RefreshToken) error
}

type Memory struct {
	mu            sync.RWMutex
	usersByID     map[string]domain.User
	userIDByEmail map[string]string
	refreshTokens map[string]domain.RefreshToken
}

func NewMemory() *Memory {
	return &Memory{
		usersByID:     map[string]domain.User{},
		userIDByEmail: map[string]string{},
		refreshTokens: map[string]domain.RefreshToken{},
	}
}

func (m *Memory) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	email := normalizeEmail(user.Email)
	if _, ok := m.userIDByEmail[email]; ok {
		return domain.User{}, ErrUserExists
	}
	user.Email = email
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	m.usersByID[user.ID] = user
	m.userIDByEmail[email] = user.ID
	return cloneUser(user), nil
}

func (m *Memory) UserByEmail(_ context.Context, email string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.userIDByEmail[normalizeEmail(email)]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	return cloneUser(m.usersByID[id]), nil
}

func (m *Memory) SaveRefreshToken(_ context.Context, token domain.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshTokens[token.TokenHash] = token
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func cloneUser(user domain.User) domain.User {
	user.Roles = append([]string(nil), user.Roles...)
	return user
}
