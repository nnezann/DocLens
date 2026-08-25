package service

import (
	"context"
	"testing"

	documentsv1 "github.com/doclens/document-intake-service/internal/gen/doclens/documents/v1"
	"github.com/doclens/document-intake-service/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func authenticatedContext() context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-organization-id", "org_1",
		"x-roles", "org_admin",
	))
}

func TestCreateAndGetDocument(t *testing.T) {
	storage, err := store.NewLocalObjectStorage(t.TempDir())
	if err != nil {
		t.Fatalf("new object storage: %v", err)
	}
	service := NewService(store.NewMemoryStore(), storage, 10*1024*1024, []string{"application/pdf"})

	resp, err := service.CreateDocument(authenticatedContext(), &documentsv1.CreateDocumentRequest{
		OrganizationId: "org_1",
		Type:           "certificate",
		Filename:       "certificate.pdf",
		ContentType:    "application/pdf",
		Content:        []byte("%PDF-1.4\n..."),
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if resp.GetOrganizationId() != "org_1" {
		t.Fatalf("organization_id mismatch: %q", resp.GetOrganizationId())
	}
	if resp.GetStatus() != "processing" {
		t.Fatalf("status mismatch: %q", resp.GetStatus())
	}

	got, err := service.GetDocument(authenticatedContext(), &documentsv1.GetDocumentRequest{OrganizationId: "org_1", Id: resp.GetId()})
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if got.GetId() != resp.GetId() {
		t.Fatalf("id mismatch: %q != %q", got.GetId(), resp.GetId())
	}
}

func TestUploadDocumentAndStatus(t *testing.T) {
	storage, err := store.NewLocalObjectStorage(t.TempDir())
	if err != nil {
		t.Fatalf("new object storage: %v", err)
	}
	service := NewService(store.NewMemoryStore(), storage, 10*1024*1024, []string{"application/pdf"})

	created, err := service.CreateDocument(authenticatedContext(), &documentsv1.CreateDocumentRequest{
		OrganizationId: "org_1",
		Type:           "certificate",
		Filename:       "certificate.pdf",
		ContentType:    "application/pdf",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	uploaded, err := service.UploadDocument(authenticatedContext(), &documentsv1.UploadDocumentRequest{
		OrganizationId: "org_1",
		DocumentId:     created.GetId(),
		Filename:       "front.pdf",
		ContentType:    "application/pdf",
		Content:        []byte("%PDF-1.4\nfront"),
		IdempotencyKey: "upload-1",
	})
	if err != nil {
		t.Fatalf("upload document: %v", err)
	}
	if uploaded.GetDocumentId() != created.GetId() {
		t.Fatalf("upload document id mismatch: %q != %q", uploaded.GetDocumentId(), created.GetId())
	}

	statusResp, err := service.GetDocumentStatus(authenticatedContext(), &documentsv1.GetDocumentStatusRequest{
		OrganizationId: "org_1",
		DocumentId:     created.GetId(),
	})
	if err != nil {
		t.Fatalf("get document status: %v", err)
	}
	if statusResp.GetStatus() != "processing" {
		t.Fatalf("status mismatch: %q", statusResp.GetStatus())
	}
	if statusResp.GetProcessingJobStatus() != "queued" {
		t.Fatalf("processing status mismatch: %q", statusResp.GetProcessingJobStatus())
	}
}

func TestTenantIsolationOnGet(t *testing.T) {
	storage, err := store.NewLocalObjectStorage(t.TempDir())
	if err != nil {
		t.Fatalf("new object storage: %v", err)
	}
	service := NewService(store.NewMemoryStore(), storage, 10*1024*1024, []string{"application/pdf"})

	created, err := service.CreateDocument(authenticatedContext(), &documentsv1.CreateDocumentRequest{
		OrganizationId: "org_1",
		Type:           "certificate",
		Filename:       "certificate.pdf",
		ContentType:    "application/pdf",
		Content:        []byte("%PDF-1.4\n..."),
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	_, err = service.GetDocument(metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-organization-id", "org_2",
		"x-roles", "org_admin",
	)), &documentsv1.GetDocumentRequest{OrganizationId: "org_2", Id: created.GetId()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}
