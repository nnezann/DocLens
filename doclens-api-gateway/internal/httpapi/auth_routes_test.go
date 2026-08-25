package httpapi

import (
	"net/http"
	"testing"
)

func TestPublicIdentityRoutesDoNotRequireAuth(t *testing.T) {
	for _, route := range []string{
		"/identity/signup",
		"/identity/signup/verify-email",
		"/identity/signup/organization",
		"/identity/login",
		"/identity/login/google",
		"/identity/forgot-password",
		"/identity/reset-password",
	} {
		req, err := http.NewRequest(http.MethodPost, route, nil)
		if err != nil {
			t.Fatal(err)
		}
		if requiresAuth(req) {
			t.Fatalf("%s unexpectedly requires authentication", route)
		}
	}
}

func TestProtectedIdentityRoutesRequireAuth(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/identity/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !requiresAuth(req) {
		t.Fatal("logout unexpectedly allows anonymous access")
	}
}
