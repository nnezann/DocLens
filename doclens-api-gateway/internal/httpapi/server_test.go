package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	documentsv1 "github.com/doclens/api-gateway/internal/gen/doclens/documents/v1"
	identityv1 "github.com/doclens/api-gateway/internal/gen/doclens/identity/v1"
	verificationv1 "github.com/doclens/api-gateway/internal/gen/doclens/verification/v1"
	"google.golang.org/grpc"
)

func TestProtectedRoutesRequireJWT(t *testing.T) {
	handler := testServer()
	req := httptest.NewRequest(http.MethodGet, "/documents/doc_1?organization_id=org_1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLoginForwardsToIdentity(t *testing.T) {
	identity := &fakeIdentity{}
	handler := testServer(func(deps *Deps) { deps.Identity = identity })
	body := bytes.NewBufferString(`{"email":"ada@example.com","password":"pw"}`)
	req := httptest.NewRequest(http.MethodPost, "/identity/login", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if identity.login.Email != "ada@example.com" {
		t.Fatalf("forwarded login = %+v", identity.login)
	}
}

func TestCreateDocumentUsesOrganizationFromClaims(t *testing.T) {
	documents := &fakeDocuments{}
	handler := testServer(func(deps *Deps) { deps.Documents = documents })
	body := bytes.NewBufferString(`{"organization_id":"ignored","type":"certificate","filename":"degree.pdf","content_type":"application/pdf","content_base64":"SGVsbG8="}`)
	req := httptest.NewRequest(http.MethodPost, "/documents", body)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "secret", map[string]any{
		"sub":    "user_1",
		"org_id": "org_claim",
		"exp":    time.Now().Add(time.Hour).Unix(),
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if documents.create.OrganizationId != "org_claim" {
		t.Fatalf("organization id = %q, want claim org", documents.create.OrganizationId)
	}
}

func testServer(options ...func(*Deps)) http.Handler {
	deps := Deps{
		Identity:       &fakeIdentity{},
		Documents:      &fakeDocuments{},
		Verification:   &fakeVerification{},
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		JWTSecret:      "secret",
		RequestTimeout: time.Second,
	}
	for _, option := range options {
		option(&deps)
	}
	return New(deps)
}

type fakeIdentity struct {
	login identityv1.LoginRequest
}

func (f *fakeIdentity) CreateUser(_ context.Context, in *identityv1.CreateUserRequest, _ ...grpc.CallOption) (*identityv1.User, error) {
	return &identityv1.User{Id: "user_1", Email: in.Email, OrganizationId: in.OrganizationId}, nil
}

func (f *fakeIdentity) Login(_ context.Context, in *identityv1.LoginRequest, _ ...grpc.CallOption) (*identityv1.LoginResponse, error) {
	f.login = *in
	return &identityv1.LoginResponse{AccessToken: "access", RefreshToken: "refresh"}, nil
}

type fakeDocuments struct {
	create documentsv1.CreateDocumentRequest
}

func (f *fakeDocuments) CreateDocument(_ context.Context, in *documentsv1.CreateDocumentRequest, _ ...grpc.CallOption) (*documentsv1.Document, error) {
	f.create = *in
	return &documentsv1.Document{Id: "doc_1", OrganizationId: in.OrganizationId, Type: in.Type, Filename: in.Filename}, nil
}

func (f *fakeDocuments) GetDocument(_ context.Context, in *documentsv1.GetDocumentRequest, _ ...grpc.CallOption) (*documentsv1.Document, error) {
	return &documentsv1.Document{Id: in.Id, OrganizationId: in.OrganizationId}, nil
}

type fakeVerification struct{}

func (f *fakeVerification) StartVerification(_ context.Context, in *verificationv1.StartVerificationRequest, _ ...grpc.CallOption) (*verificationv1.Verification, error) {
	return &verificationv1.Verification{Id: "ver_1", DocumentId: in.DocumentId, OrganizationId: in.OrganizationId, Status: "pending"}, nil
}

func (f *fakeVerification) GetVerification(_ context.Context, in *verificationv1.GetVerificationRequest, _ ...grpc.CallOption) (*verificationv1.Verification, error) {
	return &verificationv1.Verification{Id: in.Id, OrganizationId: in.OrganizationId, Status: "approved"}, nil
}

func testJWT(t *testing.T, secret string, claims map[string]any) string {
	t.Helper()
	headerBytes, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimsBytes)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return input + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
