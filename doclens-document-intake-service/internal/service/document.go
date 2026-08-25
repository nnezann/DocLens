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

	"github.com/doclens/document-intake-service/internal/authz"
	"github.com/doclens/document-intake-service/internal/events"
	documentsv1 "github.com/doclens/document-intake-service/internal/gen/doclens/documents/v1"
	"github.com/doclens/document-intake-service/internal/observability"
	"github.com/doclens/document-intake-service/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	uploadPool     *uploadPool
	metrics        *observability.IntakeMetrics
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

func (s *Service) WithUploadPool(config UploadPoolConfig) (*Service, error) {
	pool, err := newUploadPool(config)
	if err != nil {
		return nil, err
	}
	s.uploadPool = pool
	return s, nil
}

func (s *Service) WithMetrics(metrics *observability.IntakeMetrics) *Service {
	s.metrics = metrics
	return s
}

func (s *Service) CreateDocument(ctx context.Context, req *documentsv1.CreateDocumentRequest) (*documentsv1.Document, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	organizationID := strings.TrimSpace(req.GetOrganizationId())
	if err := authz.Require(ctx, organizationID, authz.PermissionCreate); err != nil {
		return nil, err
	}
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
	content := req.GetContent()
	if len(content) > 0 {
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
		upload := store.UploadRecord{
			ID:             newID("upl"),
			DocumentID:     doc.ID,
			OrganizationID: organizationID,
			Filename:       doc.Filename,
			ContentType:    contentType,
			SizeBytes:      int64(len(content)),
			Checksum:       hex.EncodeToString(checksum[:]),
			StorageRef:     storageRef,
			CreatedAt:      time.Now().UTC(),
		}
		event, err := events.NewDocumentUploaded(newID("evt"), doc, upload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "build document uploaded event: %v", err)
		}
		if durableStore, ok := s.store.(store.DurableDocumentStore); ok {
			stored, err := durableStore.CreateDocumentAndQueue(ctx, doc, upload, event)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "persist document and event: %v", err)
			}
			return toProtoDocument(stored), nil
		}
		stored, err := s.store.CreateDocument(ctx, doc)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "persist document metadata: %v", err)
		}
		stored, err = s.store.UploadDocument(ctx, organizationID, doc.ID, upload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "persist document upload: %v", err)
		}
		if err := s.publisher.PublishDocumentUploaded(ctx, organizationID, doc.ID, upload); err != nil {
			return nil, status.Errorf(codes.Internal, "publish document uploaded event: %v", err)
		}
		return toProtoDocument(stored), nil
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
	if err := authz.Require(ctx, organizationID, authz.PermissionRead); err != nil {
		return nil, err
	}
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
	if err := authz.Require(ctx, organizationID, authz.PermissionUpload); err != nil {
		return nil, err
	}
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
	if s.uploadPool != nil {
		if err := s.uploadPool.acquire(organizationID); err != nil {
			if s.metrics != nil {
				s.metrics.UploadRejected()
			}
			_ = grpc.SetHeader(ctx, metadata.Pairs("retry-after", "1"))
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}
		if s.metrics != nil {
			s.metrics.WorkerStarted()
			defer s.metrics.WorkerFinished()
		}
		defer s.uploadPool.release()
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
		if s.uploadPool != nil {
			s.uploadPool.failure()
		}
		return nil, status.Errorf(codes.Internal, "store document content: %v", err)
	}
	if s.uploadPool != nil {
		s.uploadPool.success()
	}
	if s.metrics != nil {
		s.metrics.AddStreamedBytes(int64(len(content)))
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

func (s *Service) CreateUploadIntent(ctx context.Context, req *documentsv1.CreateUploadIntentRequest) (*documentsv1.CreateUploadIntentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	organizationID := strings.TrimSpace(req.GetOrganizationId())
	if err := authz.Require(ctx, organizationID, authz.PermissionUpload); err != nil {
		return nil, err
	}
	if organizationID == "" || strings.TrimSpace(req.GetDocumentId()) == "" || strings.TrimSpace(req.GetFilename()) == "" {
		return nil, status.Error(codes.InvalidArgument, "organization_id, document_id, and filename are required")
	}
	if req.GetSizeBytes() <= 0 || (s.maxUploadBytes > 0 && req.GetSizeBytes() > s.maxUploadBytes) {
		return nil, status.Error(codes.InvalidArgument, "invalid upload size")
	}
	contentType := strings.ToLower(strings.TrimSpace(req.GetContentType()))
	if !s.isAllowedType(contentType) {
		return nil, status.Errorf(codes.InvalidArgument, "content type %q is not allowed", contentType)
	}
	storage, ok := s.objectStorage.(store.DirectUploadStorage)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "direct upload is not supported by the configured object storage")
	}
	doc, err := s.store.GetDocument(ctx, organizationID, strings.TrimSpace(req.GetDocumentId()))
	if errors.Is(err, store.ErrDocumentNotFound) || errors.Is(err, store.ErrOrganizationMismatch) {
		return nil, status.Error(codes.NotFound, "document not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load document: %v", err)
	}
	upload := store.UploadRecord{
		ID: newID("upl"), DocumentID: doc.ID, OrganizationID: organizationID,
		Filename: strings.TrimSpace(req.GetFilename()), ContentType: contentType,
		Checksum: strings.TrimSpace(req.GetChecksum()), ExpectedSize: req.GetSizeBytes(),
		IdempotencyKey: strings.TrimSpace(req.GetIdempotencyKey()), CreatedAt: time.Now().UTC(),
	}
	expires := 15 * time.Minute
	uploadURL, storageRef, expiresAt, err := storage.PresignPut(ctx, organizationID, doc.ID, upload.ID, upload.Filename, upload.ContentType, expires)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create upload URL: %v", err)
	}
	upload.StorageRef = storageRef
	uploadStore, ok := s.store.(store.UploadIntentStore)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "upload intents are not supported by the configured metadata store")
	}
	stored, err := uploadStore.CreateUploadIntent(ctx, organizationID, doc.ID, upload)
	if err != nil {
		if errors.Is(err, store.ErrDocumentNotFound) || errors.Is(err, store.ErrOrganizationMismatch) {
			return nil, status.Error(codes.NotFound, "document not found")
		}
		return nil, status.Errorf(codes.Internal, "persist upload intent: %v", err)
	}
	if s.metrics != nil {
		s.metrics.PresignedURLIssued()
	}
	if stored.ID != upload.ID {
		uploadURL, _, expiresAt, err = storage.PresignPut(ctx, organizationID, doc.ID, stored.ID, stored.Filename, stored.ContentType, expires)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "refresh upload URL: %v", err)
		}
	}
	return &documentsv1.CreateUploadIntentResponse{
		UploadId: stored.ID, DocumentId: doc.ID, UploadUrl: uploadURL, StorageRef: stored.StorageRef,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *Service) CompleteUpload(ctx context.Context, req *documentsv1.CompleteUploadRequest) (*documentsv1.UploadDocumentResponse, error) {
	startedAt := time.Now()
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	organizationID := strings.TrimSpace(req.GetOrganizationId())
	if err := authz.Require(ctx, organizationID, authz.PermissionUpload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetDocumentId()) == "" || strings.TrimSpace(req.GetUploadId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "document_id and upload_id are required")
	}
	storage, ok := s.objectStorage.(store.DirectUploadStorage)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "direct upload is not supported by the configured object storage")
	}
	doc, err := s.store.GetDocument(ctx, organizationID, req.GetDocumentId())
	if errors.Is(err, store.ErrDocumentNotFound) || errors.Is(err, store.ErrOrganizationMismatch) {
		return nil, status.Error(codes.NotFound, "document not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load document: %v", err)
	}
	pending, ok := findUpload(doc, req.GetUploadId())
	if !ok {
		return nil, status.Error(codes.NotFound, "upload intent not found")
	}
	if pending.Status == "confirmed" {
		return &documentsv1.UploadDocumentResponse{UploadId: pending.ID, DocumentId: pending.DocumentID, StorageRef: pending.StorageRef, Checksum: pending.Checksum}, nil
	}
	if pending.Status != "pending" {
		return nil, status.Error(codes.FailedPrecondition, "upload is not pending")
	}
	head, err := storage.Head(ctx, pending.StorageRef)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "uploaded object is not available: %v", err)
	}
	size := req.GetSizeBytes()
	if size == 0 {
		size = head.SizeBytes
	}
	if head.SizeBytes != size {
		return nil, status.Error(codes.InvalidArgument, "uploaded object size does not match declared size")
	}
	checksum := strings.TrimSpace(req.GetChecksum())
	if checksum == "" {
		checksum = pending.Checksum
	}
	if checksum != "" && head.Checksum != "" && checksum != head.Checksum {
		return nil, status.Error(codes.InvalidArgument, "upload checksum does not match object")
	}
	uploadStore, ok := s.store.(store.UploadIntentStore)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "upload confirmation is not supported by the configured metadata store")
	}
	pending.SizeBytes, pending.Checksum = size, checksum
	event, err := events.NewDocumentUploaded(newID("evt"), doc, pending)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build document uploaded event: %v", err)
	}
	confirmed, err := uploadStore.ConfirmUpload(ctx, organizationID, doc.ID, req.GetUploadId(), size, checksum, event)
	if errors.Is(err, store.ErrUploadNotFound) {
		return nil, status.Error(codes.NotFound, "upload intent not found")
	}
	if errors.Is(err, store.ErrUploadMismatch) {
		return nil, status.Error(codes.InvalidArgument, "upload does not match intent")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "confirm upload: %v", err)
	}
	if s.metrics != nil {
		s.metrics.ObserveConfirmation(time.Since(startedAt))
	}
	return &documentsv1.UploadDocumentResponse{UploadId: confirmed.ID, DocumentId: confirmed.DocumentID, StorageRef: confirmed.StorageRef, Checksum: confirmed.Checksum}, nil
}

type ObjectStorageNotification struct {
	OrganizationID string
	DocumentID     string
	UploadID       string
	SizeBytes      int64
	Checksum       string
}

// ConfirmUploadNotification is the provider-neutral entry point for bucket
// notification adapters. It deliberately reuses client confirmation logic.
func (s *Service) ConfirmUploadNotification(ctx context.Context, notification ObjectStorageNotification) (*documentsv1.UploadDocumentResponse, error) {
	if strings.TrimSpace(notification.OrganizationID) == "" ||
		strings.TrimSpace(notification.DocumentID) == "" ||
		strings.TrimSpace(notification.UploadID) == "" {
		return nil, status.Error(codes.InvalidArgument, "notification organization, document, and upload IDs are required")
	}
	internalCtx := metadata.NewIncomingContext(ctx, metadata.Pairs(
		"x-organization-id", notification.OrganizationID,
		"x-roles", "platform_admin",
	))
	return s.CompleteUpload(internalCtx, &documentsv1.CompleteUploadRequest{
		OrganizationId: notification.OrganizationID,
		DocumentId:     notification.DocumentID,
		UploadId:       notification.UploadID,
		SizeBytes:      notification.SizeBytes,
		Checksum:       notification.Checksum,
	})
}

func findUpload(doc store.Document, uploadID string) (store.UploadRecord, bool) {
	for _, upload := range doc.Uploads {
		if upload.ID == uploadID {
			return upload, true
		}
	}
	return store.UploadRecord{}, false
}

func (s *Service) GetDocumentStatus(ctx context.Context, req *documentsv1.GetDocumentStatusRequest) (*documentsv1.DocumentStatus, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	organizationID := strings.TrimSpace(req.GetOrganizationId())
	if err := authz.Require(ctx, organizationID, authz.PermissionRead); err != nil {
		return nil, err
	}
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
			Id:           upload.ID,
			Filename:     upload.Filename,
			ContentType:  upload.ContentType,
			SizeBytes:    upload.SizeBytes,
			Checksum:     upload.Checksum,
			StorageRef:   upload.StorageRef,
			CreatedAt:    upload.CreatedAt.Format(time.RFC3339),
			Status:       upload.Status,
			UploadMethod: upload.UploadMethod,
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
