package domain

import "time"

type User struct {
	ID             string
	OrganizationID string
	Email          string
	PasswordHash   string
	Roles          []string
	Disabled       bool
	CreatedAt      time.Time
}

type RefreshToken struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}
