package service

import (
	"context"
	"testing"

	documentsv1 "github.com/doclens/document-intake-service/internal/gen/doclens/documents/v1"
	"github.com/doclens/document-intake-service/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateAndGetDocument(t *testing.T) {
	storage, err := store.NewLocalObjectStorage(t.TempDir())
	if err != nil {
		t.Fatalf("new object storage: %v", err)
	}
	service := NewService(store.NewMemoryStore(), storage, 10*1024*1024, []string{"application/pdf"})

	resp, err := service.CreateDocument(context.Background(), &documentsv1.CreateDocumentRequest{
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
	if resp.GetStatus() != "created" {
		t.Fatalf("status mismatch: %q", resp.GetStatus())
	}

	got, err := service.GetDocument(context.Background(), &documentsv1.GetDocumentRequest{OrganizationId: "org_1", Id: resp.GetId()})
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if got.GetId() != resp.GetId() {
		t.Fatalf("id mismatch: %q != %q", got.GetId(), resp.GetId())
	}
}

func TestTenantIsolationOnGet(t *testing.T) {
	storage, err := store.NewLocalObjectStorage(t.TempDir())
	if err != nil {
		t.Fatalf("new object storage: %v", err)
	}
	service := NewService(store.NewMemoryStore(), storage, 10*1024*1024, []string{"application/pdf"})

	created, err := service.CreateDocument(context.Background(), &documentsv1.CreateDocumentRequest{
		OrganizationId: "org_1",
		Type:           "certificate",
		Filename:       "certificate.pdf",
		ContentType:    "application/pdf",
		Content:        []byte("%PDF-1.4\n..."),
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	_, err = service.GetDocument(context.Background(), &documentsv1.GetDocumentRequest{OrganizationId: "org_2", Id: created.GetId()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}
