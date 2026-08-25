package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/doclens/api-gateway/internal/auth"
	documentsv1 "github.com/doclens/api-gateway/internal/gen/doclens/documents/v1"
	identityv1 "github.com/doclens/api-gateway/internal/gen/doclens/identity/v1"
	verificationv1 "github.com/doclens/api-gateway/internal/gen/doclens/verification/v1"
	"github.com/doclens/api-gateway/internal/observability"
	"google.golang.org/grpc"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

type IdentityClient interface {
	CreateUser(ctx context.Context, in *identityv1.CreateUserRequest, opts ...grpc.CallOption) (*identityv1.User, error)
	Login(ctx context.Context, in *identityv1.LoginRequest, opts ...grpc.CallOption) (*identityv1.LoginResponse, error)
}

type DocumentsClient interface {
	CreateDocument(ctx context.Context, in *documentsv1.CreateDocumentRequest, opts ...grpc.CallOption) (*documentsv1.Document, error)
	GetDocument(ctx context.Context, in *documentsv1.GetDocumentRequest, opts ...grpc.CallOption) (*documentsv1.Document, error)
}

type VerificationClient interface {
	StartVerification(ctx context.Context, in *verificationv1.StartVerificationRequest, opts ...grpc.CallOption) (*verificationv1.Verification, error)
	GetVerification(ctx context.Context, in *verificationv1.GetVerificationRequest, opts ...grpc.CallOption) (*verificationv1.Verification, error)
}

type handlers struct {
	identity     IdentityClient
	documents    DocumentsClient
	verification VerificationClient
}

func (h handlers) health(w http.ResponseWriter, _ *http.Request) {
	observability.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	for name, client := range s.healthChecks {
		resp, err := client.Check(r.Context(), &healthv1.HealthCheckRequest{})
		if err != nil || resp.GetStatus() != healthv1.HealthCheckResponse_SERVING {
			observability.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unready", "service": name})
			return
		}
	}
	observability.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h handlers) createUser(w http.ResponseWriter, r *http.Request) {
	var req identityv1.CreateUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Email == "" || req.Password == "" || req.OrganizationId == "" {
		observability.Error(w, http.StatusBadRequest, "organization_id, email, and password are required")
		return
	}
	user, err := h.identity.CreateUser(r.Context(), &req)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	observability.WriteJSON(w, http.StatusCreated, user)
}

func (h handlers) login(w http.ResponseWriter, r *http.Request) {
	var req identityv1.LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Email == "" || req.Password == "" {
		observability.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}
	resp, err := h.identity.Login(r.Context(), &req)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	observability.WriteJSON(w, http.StatusOK, resp)
}

func (h handlers) createDocument(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrganizationID string `json:"organization_id"`
		Type           string `json:"type"`
		Filename       string `json:"filename"`
		ContentType    string `json:"content_type"`
		ContentBase64  string `json:"content_base64"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	orgID := organizationID(r, req.OrganizationID)
	if orgID == "" || req.Type == "" || req.Filename == "" || req.ContentBase64 == "" {
		observability.Error(w, http.StatusBadRequest, "organization_id, type, filename, and content_base64 are required")
		return
	}
	content, err := base64.StdEncoding.DecodeString(req.ContentBase64)
	if err != nil {
		observability.Error(w, http.StatusBadRequest, "content_base64 must be valid base64")
		return
	}
	doc, err := h.documents.CreateDocument(r.Context(), &documentsv1.CreateDocumentRequest{
		OrganizationId: orgID,
		Type:           req.Type,
		Filename:       req.Filename,
		ContentType:    req.ContentType,
		Content:        content,
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	observability.WriteJSON(w, http.StatusCreated, doc)
}

func (h handlers) getDocument(w http.ResponseWriter, r *http.Request) {
	orgID := organizationID(r, r.URL.Query().Get("organization_id"))
	if orgID == "" {
		observability.Error(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	doc, err := h.documents.GetDocument(r.Context(), &documentsv1.GetDocumentRequest{
		OrganizationId: orgID,
		Id:             r.PathValue("id"),
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	observability.WriteJSON(w, http.StatusOK, doc)
}

func (h handlers) startVerification(w http.ResponseWriter, r *http.Request) {
	var req verificationv1.StartVerificationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.OrganizationId = organizationID(r, req.OrganizationId)
	if req.OrganizationId == "" || req.DocumentId == "" {
		observability.Error(w, http.StatusBadRequest, "organization_id and document_id are required")
		return
	}
	verification, err := h.verification.StartVerification(r.Context(), &req)
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	observability.WriteJSON(w, http.StatusAccepted, verification)
}

func (h handlers) getVerification(w http.ResponseWriter, r *http.Request) {
	orgID := organizationID(r, r.URL.Query().Get("organization_id"))
	if orgID == "" {
		observability.Error(w, http.StatusBadRequest, "organization_id is required")
		return
	}
	verification, err := h.verification.GetVerification(r.Context(), &verificationv1.GetVerificationRequest{
		OrganizationId: orgID,
		Id:             r.PathValue("id"),
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	observability.WriteJSON(w, http.StatusOK, verification)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		observability.Error(w, http.StatusBadRequest, "invalid json body")
		return false
	}
	return true
}

func organizationID(r *http.Request, fallback string) string {
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok && claims.OrganizationID != "" {
		return claims.OrganizationID
	}
	return fallback
}
