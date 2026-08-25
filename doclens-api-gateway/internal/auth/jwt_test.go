package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestVerifierAcceptsValidHS256Token(t *testing.T) {
	token := signToken(t, "secret", Claims{
		Subject:        "user_1",
		OrganizationID: "org_1",
		Roles:          []string{"admin"},
		ExpiresAt:      time.Now().Add(time.Hour).Unix(),
	})

	claims, err := NewVerifier("secret").Verify(token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if claims.Subject != "user_1" || claims.OrganizationID != "org_1" {
		t.Fatalf("claims = %+v", claims)
	}
}

func TestVerifierRejectsExpiredToken(t *testing.T) {
	token := signToken(t, "secret", Claims{
		Subject:   "user_1",
		ExpiresAt: time.Now().Add(-time.Minute).Unix(),
	})

	if _, err := NewVerifier("secret").Verify(token); err == nil {
		t.Fatal("Verify() error = nil, want expired token error")
	}
}

func signToken(t *testing.T, secret string, claims Claims) string {
	t.Helper()
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimsBytes)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
