package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	documentsv1 "github.com/doclens/document-intake-service/internal/gen/doclens/documents/v1"
	"github.com/doclens/document-intake-service/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

type Service struct {
	documentsv1.UnimplementedDocumentIntakeServiceServer
	store          store.Store
	objectStorage  store.ObjectStorage
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
	}
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
	documentID := newID("doc")
	storageRef, err := s.objectStorage.Put(ctx, organizationID, documentID, req.GetFilename(), contentType, content)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "store document content: %v", err)
	}
	createdAt := time.Now().UTC()
	doc := store.Document{
		ID:             documentID,
		OrganizationID: organizationID,
		Type:           strings.TrimSpace(req.GetType()),
		Filename:       strings.TrimSpace(req.GetFilename()),
		ContentType:    contentType,
		Status:         "created",
		StorageRef:     storageRef,
		SizeBytes:      int64(len(content)),
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
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

func toProtoDocument(doc store.Document) *documentsv1.Document {
	return &documentsv1.Document{
		Id:             doc.ID,
		OrganizationId: doc.OrganizationID,
		Type:           doc.Type,
		Filename:       doc.Filename,
		ContentType:    doc.ContentType,
		Status:         doc.Status,
		CreatedAt:      doc.CreatedAt.Format(time.RFC3339),
	}
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
