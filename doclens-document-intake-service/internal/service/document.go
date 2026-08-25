package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/doclens/document-intake-service/internal/events"
	documentsv1 "github.com/doclens/document-intake-service/internal/gen/doclens/documents/v1"
	"github.com/doclens/document-intake-service/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EventPublisher interface {
	PublishDocumentUploaded(ctx context.Context, organizationID, documentID string, upload store.UploadRecord) error
}

type NoopEventPublisher struct{}

func (NoopEventPublisher) PublishDocumentUploaded(_ context.Context, _, _ string, _ store.UploadRecord) error {
	return nil
}

type Service struct {
	documentsv1.UnimplementedDocumentIntakeServiceServer
	store          store.Store
	objectStorage  store.ObjectStorage
	publisher      EventPublisher
	maxUploadBytes int64
	allowedTypes   map[string]struct{}
}

func NewService(store store.Store, objectStorage store.ObjectStorage, maxUploadBytes int64, allowedTypes []string) *Service {
	allowed := make(map[string]struct{}, len(allowedTypes))
	for _, t := range allowedTypes {
		trimmed := strings.ToLower(strings.TrimSpace(t))
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	return &Service{
		store:          store,
		objectStorage:  objectStorage,
		maxUploadBytes: maxUploadBytes,
		allowedTypes:   allowed,
		publisher:      NoopEventPublisher{},
	}
}

func (s *Service) WithPublisher(publisher EventPublisher) *Service {
	if publisher != nil {
		s.publisher = publisher
	}
	return s
}

func (s *Service) CreateDocument(ctx context.Context, req *documentsv1.CreateDocumentRequest) (*documentsv1.Document, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	organizationID := strings.TrimSpace(req.GetOrganizationId())
	if organizationID == "" {
		return nil, status.Error(codes.InvalidArgument, "organization_id is required")
	}
	if strings.TrimSpace(req.GetType()) == "" {
		return nil, status.Error(codes.InvalidArgument, "type is required")
	}
	createdAt := time.Now().UTC()
	doc := store.Document{
		ID:                  newID("doc"),
		OrganizationID:      organizationID,
		Type:                strings.TrimSpace(req.GetType()),
		Filename:            strings.TrimSpace(req.GetFilename()),
		ContentType:         strings.TrimSpace(req.GetContentType()),
		Status:              "created",
		ProcessingJobStatus: "pending",
		CreatedAt:           createdAt,
		UpdatedAt:           createdAt,
	}
	if content := req.GetContent(); len(content) > 0 {
		if s.maxUploadBytes > 0 && int64(len(content)) > s.maxUploadBytes {
			return nil, status.Errorf(codes.InvalidArgument, "document exceeds max upload size of %d bytes", s.maxUploadBytes)
		}
		contentType := strings.TrimSpace(req.GetContentType())
		if contentType == "" {
			contentType = http.DetectContentType(content)
		}
		if !s.isAllowedType(contentType) {
			return nil, status.Errorf(codes.InvalidArgument, "content type %q is not allowed", contentType)
		}
		if s.objectStorage == nil {
			return nil, status.Error(codes.Internal, "object storage is not configured")
		}
		storageRef, err := s.objectStorage.Put(ctx, organizationID, doc.ID, doc.Filename, contentType, content)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "store document content: %v", err)
		}
		doc.ContentType = contentType
		doc.Status = "processing"
		doc.ProcessingJobStatus = "queued"
		doc.StorageRef = storageRef
		doc.SizeBytes = int64(len(content))
		doc.UpdatedAt = time.Now().UTC()
		checksum := sha256.Sum256(content)
		if err := s.publisher.PublishDocumentUploaded(ctx, organizationID, doc.ID, store.UploadRecord{
			ID:             newID("upl"),
			DocumentID:     doc.ID,
			OrganizationID: organizationID,
			Filename:       doc.Filename,
			ContentType:    contentType,
			SizeBytes:      int64(len(content)),
			Checksum:       hex.EncodeToString(checksum[:]),
			StorageRef:     storageRef,
			CreatedAt:      time.Now().UTC(),
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "publish document uploaded event: %v", err)
		}
	}
	stored, err := s.store.CreateDocument(ctx, doc)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "persist document metadata: %v", err)
	}
	return toProtoDocument(stored), nil
}

func (s *Service) GetDocument(ctx context.Context, req *documentsv1.GetDocumentRequest) (*documentsv1.Document, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	organizationID := strings.TrimSpace(req.GetOrganizationId())
	if organizationID == "" {
		return nil, status.Error(codes.InvalidArgument, "organization_id is required")
	}
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	doc, err := s.store.GetDocument(ctx, organizationID, req.GetId())
	if errors.Is(err, store.ErrDocumentNotFound) || errors.Is(err, store.ErrOrganizationMismatch) {
		return nil, status.Error(codes.NotFound, "document not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load document: %v", err)
	}
	return toProtoDocument(doc), nil
}

func (s *Service) UploadDocument(ctx context.Context, req *documentsv1.UploadDocumentRequest) (*documentsv1.UploadDocumentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	organizationID := strings.TrimSpace(req.GetOrganizationId())
	if organizationID == "" {
		return nil, status.Error(codes.InvalidArgument, "organization_id is required")
	}
	documentID := strings.TrimSpace(req.GetDocumentId())
	if documentID == "" {
		return nil, status.Error(codes.InvalidArgument, "document_id is required")
	}
	if strings.TrimSpace(req.GetFilename()) == "" {
		return nil, status.Error(codes.InvalidArgument, "filename is required")
	}
	content := req.GetContent()
	if len(content) == 0 {
		return nil, status.Error(codes.InvalidArgument, "document content is required")
	}
	if s.maxUploadBytes > 0 && int64(len(content)) > s.maxUploadBytes {
		return nil, status.Errorf(codes.InvalidArgument, "document exceeds max upload size of %d bytes", s.maxUploadBytes)
	}
	contentType := strings.TrimSpace(req.GetContentType())
	if contentType == "" {
		contentType = http.DetectContentType(content)
	}
	if !s.isAllowedType(contentType) {
		return nil, status.Errorf(codes.InvalidArgument, "content type %q is not allowed", contentType)
	}
	if s.objectStorage == nil {
		return nil, status.Error(codes.Internal, "object storage is not configured")
	}
	doc, err := s.store.GetDocument(ctx, organizationID, documentID)
	if errors.Is(err, store.ErrDocumentNotFound) || errors.Is(err, store.ErrOrganizationMismatch) {
		return nil, status.Error(codes.NotFound, "document not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load document: %v", err)
	}
	for _, existing := range doc.Uploads {
		if req.GetIdempotencyKey() != "" && existing.IdempotencyKey == req.GetIdempotencyKey() {
			return &documentsv1.UploadDocumentResponse{
				UploadId:   existing.ID,
				DocumentId: existing.DocumentID,
				StorageRef: existing.StorageRef,
				Checksum:   existing.Checksum,
			}, nil
		}
	}
	storageRef, err := s.objectStorage.Put(ctx, organizationID, documentID, req.GetFilename(), contentType, content)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "store document content: %v", err)
	}
	checksum := sha256.Sum256(content)
	upload := store.UploadRecord{
		ID:             newID("upl"),
		DocumentID:     documentID,
		OrganizationID: organizationID,
		Filename:       strings.TrimSpace(req.GetFilename()),
		ContentType:    contentType,
		SizeBytes:      int64(len(content)),
		Checksum:       hex.EncodeToString(checksum[:]),
		StorageRef:     storageRef,
		IdempotencyKey: req.GetIdempotencyKey(),
		CreatedAt:      time.Now().UTC(),
	}
	docForEvent := doc
	event, err := events.NewDocumentUploaded(newID("evt"), docForEvent, upload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build document uploaded event: %v", err)
	}
	var updatedDoc store.Document
	if durableStore, ok := s.store.(store.DurableUploadStore); ok {
		updatedDoc, err = durableStore.UploadDocumentAndQueue(ctx, organizationID, documentID, upload, event)
	} else {
		updatedDoc, err = s.store.UploadDocument(ctx, organizationID, documentID, upload)
		if err == nil {
			err = s.publisher.PublishDocumentUploaded(ctx, organizationID, updatedDoc.ID, upload)
		}
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "persist upload and event: %v", err)
	}
	return &documentsv1.UploadDocumentResponse{
		UploadId:   upload.ID,
		DocumentId: updatedDoc.ID,
		StorageRef: storageRef,
		Checksum:   upload.Checksum,
	}, nil
}

func (s *Service) GetDocumentStatus(ctx context.Context, req *documentsv1.GetDocumentStatusRequest) (*documentsv1.DocumentStatus, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	organizationID := strings.TrimSpace(req.GetOrganizationId())
	if organizationID == "" {
		return nil, status.Error(codes.InvalidArgument, "organization_id is required")
	}
	documentID := strings.TrimSpace(req.GetDocumentId())
	if documentID == "" {
		return nil, status.Error(codes.InvalidArgument, "document_id is required")
	}
	doc, err := s.store.GetDocumentStatus(ctx, organizationID, documentID)
	if errors.Is(err, store.ErrDocumentNotFound) || errors.Is(err, store.ErrOrganizationMismatch) {
		return nil, status.Error(codes.NotFound, "document not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load document status: %v", err)
	}
	return &documentsv1.DocumentStatus{
		DocumentId:          doc.ID,
		Status:              doc.Status,
		ProcessingJobStatus: doc.ProcessingJobStatus,
		UpdatedAt:           doc.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func toProtoDocument(doc store.Document) *documentsv1.Document {
	protoDoc := &documentsv1.Document{
		Id:             doc.ID,
		OrganizationId: doc.OrganizationID,
		Type:           doc.Type,
		Filename:       doc.Filename,
		ContentType:    doc.ContentType,
		Status:         doc.Status,
		CreatedAt:      doc.CreatedAt.Format(time.RFC3339),
	}
	for _, upload := range doc.Uploads {
		protoDoc.Uploads = append(protoDoc.Uploads, &documentsv1.Upload{
			Id:          upload.ID,
			Filename:    upload.Filename,
			ContentType: upload.ContentType,
			SizeBytes:   upload.SizeBytes,
			Checksum:    upload.Checksum,
			StorageRef:  upload.StorageRef,
			CreatedAt:   upload.CreatedAt.Format(time.RFC3339),
		})
	}
	return protoDoc
}

func (s *Service) isAllowedType(contentType string) bool {
	if len(s.allowedTypes) == 0 {
		return true
	}
	_, ok := s.allowedTypes[strings.ToLower(strings.TrimSpace(contentType))]
	return ok
}

func newID(prefix string) string {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(random[:]))
}
