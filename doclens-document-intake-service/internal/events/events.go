package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/doclens/document-intake-service/internal/store"
)

const DocumentUploadedRoutingKey = "document.uploaded"

type Envelope struct {
	EventID        string           `json:"event_id"`
	EventType      string           `json:"event_type"`
	EventVersion   int              `json:"event_version"`
	OccurredAt     time.Time        `json:"occurred_at"`
	OrganizationID string           `json:"organization_id"`
	DocumentID     string           `json:"document_id"`
	Payload        DocumentUploaded `json:"payload"`
}

type DocumentUploaded struct {
	Type      string         `json:"type"`
	UploadIDs []string       `json:"upload_ids"`
	Uploads   []UploadedFile `json:"uploads"`
}

type UploadedFile struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Checksum    string `json:"checksum"`
	StorageRef  string `json:"storage_ref"`
}

func NewDocumentUploaded(eventID string, doc store.Document, upload store.UploadRecord) (store.OutboxEvent, error) {
	occurredAt := upload.CreatedAt.UTC()
	envelope := Envelope{
		EventID:        eventID,
		EventType:      "DocumentUploaded",
		EventVersion:   1,
		OccurredAt:     occurredAt,
		OrganizationID: doc.OrganizationID,
		DocumentID:     doc.ID,
		Payload: DocumentUploaded{
			Type:      doc.Type,
			UploadIDs: []string{upload.ID},
			Uploads: []UploadedFile{{
				ID: upload.ID, Filename: upload.Filename, ContentType: upload.ContentType,
				SizeBytes: upload.SizeBytes, Checksum: upload.Checksum, StorageRef: upload.StorageRef,
			}},
		},
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return store.OutboxEvent{}, fmt.Errorf("marshal document uploaded event: %w", err)
	}
	return store.OutboxEvent{
		ID: eventID, EventType: envelope.EventType, EventVersion: envelope.EventVersion,
		RoutingKey: DocumentUploadedRoutingKey, OrganizationID: doc.OrganizationID,
		DocumentID: doc.ID, Payload: payload, CreatedAt: occurredAt,
	}, nil
}
