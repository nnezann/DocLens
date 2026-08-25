package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"
)

type Claims struct {
	Subject        string   `json:"sub"`
	OrganizationID string   `json:"org_id"`
	Roles          []string `json:"roles"`
	ExpiresAt      int64    `json:"exp"`
}

type TokenIssuer struct {
	secret []byte
}

func NewTokenIssuer(secret string) TokenIssuer {
	return TokenIssuer{secret: []byte(secret)}
}

func (i TokenIssuer) AccessToken(userID, organizationID string, roles []string, ttl time.Duration) (string, error) {
	return i.sign(Claims{
		Subject:        userID,
		OrganizationID: organizationID,
		Roles:          roles,
		ExpiresAt:      time.Now().Add(ttl).Unix(),
	})
}

func RefreshToken() (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(random), nil
}

func (i TokenIssuer) sign(claims Claims) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, i.secret)
	mac.Write([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
