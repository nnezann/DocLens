package auth

import (
	"strings"
	"testing"
)

func TestPasswordUsesArgon2id(t *testing.T) {
	encoded, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$") {
		t.Fatalf("encoded password = %q, want argon2id", encoded)
	}
	if !VerifyPassword(encoded, "correct horse battery staple") {
		t.Fatal("VerifyPassword() rejected the original password")
	}
	if VerifyPassword(encoded, "wrong password") {
		t.Fatal("VerifyPassword() accepted the wrong password")
	}
}
