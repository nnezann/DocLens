package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/doclens/document-intake-service/internal/store"
)

func TestDocumentUploadedEnvelope(t *testing.T) {
	createdAt := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	event, err := NewDocumentUploaded("evt_123", store.Document{
		ID: "doc_456", OrganizationID: "org_123", Type: "certificate",
	}, store.UploadRecord{
		ID: "upl_789", DocumentID: "doc_456", OrganizationID: "org_123",
		Filename: "front.pdf", ContentType: "application/pdf", SizeBytes: 12,
		Checksum: "abc", StorageRef: "organizations/org/documents/doc/uploads/front.pdf",
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("build event: %v", err)
	}
	if event.RoutingKey != DocumentUploadedRoutingKey {
		t.Fatalf("routing key = %q", event.RoutingKey)
	}
	var envelope Envelope
	if err := json.Unmarshal(event.Payload, &envelope); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if envelope.EventID != "evt_123" || envelope.EventType != "DocumentUploaded" || envelope.EventVersion != 1 {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	if envelope.OrganizationID != "org_123" || envelope.DocumentID != "doc_456" {
		t.Fatalf("unexpected tenant scope: %+v", envelope)
	}
	if len(envelope.Payload.Uploads) != 1 || envelope.Payload.Uploads[0].StorageRef == "" {
		t.Fatalf("upload metadata missing: %+v", envelope.Payload)
	}
}
