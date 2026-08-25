package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type contextKey string

const claimsContextKey contextKey = "claims"

type Claims struct {
	Subject        string   `json:"sub"`
	OrganizationID string   `json:"org_id"`
	Roles          []string `json:"roles"`
	Permissions    []string `json:"permissions"`
	ExpiresAt      int64    `json:"exp"`
}

type Verifier struct {
	secret []byte
}

func NewVerifier(secret string) Verifier {
	return Verifier{secret: []byte(secret)}
}

func (v Verifier) VerifyBearer(header string) (Claims, error) {
	if !strings.HasPrefix(header, "Bearer ") {
		return Claims{}, errors.New("missing bearer token")
	}
	return v.Verify(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
}

func (v Verifier) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("malformed token")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return Claims{}, fmt.Errorf("parse header: %w", err)
	}
	if header.Algorithm != "HS256" {
		return Claims{}, errors.New("unsupported token algorithm")
	}
	signingInput := parts[0] + "." + parts[1]
	expected := hmacSHA256([]byte(signingInput), v.secret)
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, fmt.Errorf("decode signature: %w", err)
	}
	if !hmac.Equal(actual, expected) {
		return Claims{}, errors.New("invalid signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("decode payload: %w", err)
	}
	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return Claims{}, fmt.Errorf("parse claims: %w", err)
	}
	if claims.ExpiresAt > 0 && time.Now().Unix() >= claims.ExpiresAt {
		return Claims{}, errors.New("expired token")
	}
	if claims.Subject == "" {
		return Claims{}, errors.New("missing subject")
	}
	return claims, nil
}

func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(Claims)
	return claims, ok
}

func hmacSHA256(data, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return mac.Sum(nil)
}
