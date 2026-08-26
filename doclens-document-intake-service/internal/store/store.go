package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ErrDocumentNotFound = fmt.Errorf("document not found")
var ErrOrganizationMismatch = fmt.Errorf("document belongs to a different organization")
var ErrUploadNotFound = fmt.Errorf("upload not found")
var ErrUploadNotPending = fmt.Errorf("upload is not pending")
var ErrUploadMismatch = fmt.Errorf("upload verification failed")

type UploadRecord struct {
	ID             string
	DocumentID     string
	OrganizationID string
	Filename       string
	ContentType    string
	SizeBytes      int64
	Checksum       string
	StorageRef     string
	IdempotencyKey string
	Status         string
	UploadMethod   string
	ExpectedSize   int64
	ConfirmedAt    *time.Time
	CreatedAt      time.Time
}

type Document struct {
	ID                  string
	OrganizationID      string
	Type                string
	Filename            string
	ContentType         string
	Status              string
	ProcessingJobStatus string
	StorageRef          string
	SizeBytes           int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Uploads             []UploadRecord
}

type Store interface {
	CreateDocument(ctx context.Context, doc Document) (Document, error)
	GetDocument(ctx context.Context, organizationID, id string) (Document, error)
	UploadDocument(ctx context.Context, organizationID, documentID string, upload UploadRecord) (Document, error)
	GetDocumentStatus(ctx context.Context, organizationID, documentID string) (Document, error)
}

type UploadIntentStore interface {
	CreateUploadIntent(ctx context.Context, organizationID, documentID string, upload UploadRecord) (UploadRecord, error)
	ConfirmUpload(ctx context.Context, organizationID, documentID, uploadID string, sizeBytes int64, checksum string, event OutboxEvent) (UploadRecord, error)
}

type OutboxEvent struct {
	ID             string
	EventType      string
	EventVersion   int
	RoutingKey     string
	OrganizationID string
	DocumentID     string
	Payload        []byte
	CreatedAt      time.Time
}

type DurableUploadStore interface {
	UploadDocumentAndQueue(ctx context.Context, organizationID, documentID string, upload UploadRecord, event OutboxEvent) (Document, error)
}

type DurableDocumentStore interface {
	CreateDocumentAndQueue(ctx context.Context, doc Document, upload UploadRecord, event OutboxEvent) (Document, error)
}

type MemoryStore struct {
	mu   sync.RWMutex
	docs map[string]Document
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{docs: make(map[string]Document)}
}

func (m *MemoryStore) CreateDocument(_ context.Context, doc Document) (Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.docs[doc.ID]; exists {
		return Document{}, fmt.Errorf("document already exists")
	}
	clone := doc
	m.docs[doc.ID] = clone
	return clone, nil
}

func (m *MemoryStore) CreateDocumentAndQueue(ctx context.Context, doc Document, upload UploadRecord, _ OutboxEvent) (Document, error) {
	if _, err := m.CreateDocument(ctx, doc); err != nil {
		return Document{}, err
	}
	return m.UploadDocument(ctx, doc.OrganizationID, doc.ID, upload)
}

func (m *MemoryStore) GetDocument(_ context.Context, organizationID, id string) (Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	doc, exists := m.docs[id]
	if !exists {
		return Document{}, ErrDocumentNotFound
	}
	if strings.TrimSpace(doc.OrganizationID) != strings.TrimSpace(organizationID) {
		return Document{}, ErrOrganizationMismatch
	}
	return doc, nil
}

func (m *MemoryStore) UploadDocument(_ context.Context, organizationID, documentID string, upload UploadRecord) (Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if upload.Status == "" {
		upload.Status = "confirmed"
	}
	if upload.UploadMethod == "" {
		upload.UploadMethod = "proxied"
	}
	doc, exists := m.docs[documentID]
	if !exists {
		return Document{}, ErrDocumentNotFound
	}
	if strings.TrimSpace(doc.OrganizationID) != strings.TrimSpace(organizationID) {
		return Document{}, ErrOrganizationMismatch
	}
	for _, existing := range doc.Uploads {
		if upload.IdempotencyKey != "" && existing.IdempotencyKey == upload.IdempotencyKey {
			return doc, nil
		}
	}
	doc.Uploads = append(doc.Uploads, upload)
	doc.Status = "processing"
	doc.ProcessingJobStatus = "queued"
	doc.UpdatedAt = time.Now().UTC()
	if upload.StorageRef != "" {
		doc.StorageRef = upload.StorageRef
		doc.SizeBytes = upload.SizeBytes
	}
	m.docs[documentID] = doc
	return doc, nil
}

func (m *MemoryStore) CreateUploadIntent(_ context.Context, organizationID, documentID string, upload UploadRecord) (UploadRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc, exists := m.docs[documentID]
	if !exists {
		return UploadRecord{}, ErrDocumentNotFound
	}
	if doc.OrganizationID != organizationID {
		return UploadRecord{}, ErrOrganizationMismatch
	}
	for _, existing := range doc.Uploads {
		if upload.IdempotencyKey != "" && existing.IdempotencyKey == upload.IdempotencyKey {
			return existing, nil
		}
	}
	upload.Status, upload.UploadMethod = "pending", "presigned_direct"
	doc.Uploads = append(doc.Uploads, upload)
	doc.UpdatedAt = time.Now().UTC()
	m.docs[documentID] = doc
	return upload, nil
}

func (m *MemoryStore) ConfirmUpload(_ context.Context, organizationID, documentID, uploadID string, sizeBytes int64, checksum string, _ OutboxEvent) (UploadRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc, exists := m.docs[documentID]
	if !exists {
		return UploadRecord{}, ErrDocumentNotFound
	}
	if doc.OrganizationID != organizationID {
		return UploadRecord{}, ErrOrganizationMismatch
	}
	for i := range doc.Uploads {
		upload := &doc.Uploads[i]
		if upload.ID != uploadID {
			continue
		}
		if upload.Status == "confirmed" {
			return *upload, nil
		}
		if upload.Status != "pending" {
			return UploadRecord{}, ErrUploadNotPending
		}
		if upload.ExpectedSize > 0 && upload.ExpectedSize != sizeBytes {
			return UploadRecord{}, ErrUploadMismatch
		}
		if upload.Checksum != "" && checksum != "" && upload.Checksum != checksum {
			return UploadRecord{}, ErrUploadMismatch
		}
		now := time.Now().UTC()
		upload.Status, upload.ConfirmedAt = "confirmed", &now
		upload.SizeBytes, upload.Checksum = sizeBytes, checksum
		doc.Status, doc.ProcessingJobStatus, doc.UpdatedAt = "processing", "queued", now
		m.docs[documentID] = doc
		return *upload, nil
	}
	return UploadRecord{}, ErrUploadNotFound
}

func (m *MemoryStore) UploadDocumentAndQueue(ctx context.Context, organizationID, documentID string, upload UploadRecord, _ OutboxEvent) (Document, error) {
	return m.UploadDocument(ctx, organizationID, documentID, upload)
}

func (m *MemoryStore) GetDocumentStatus(_ context.Context, organizationID, documentID string) (Document, error) {
	return m.GetDocument(context.Background(), organizationID, documentID)
}

type ObjectStorage interface {
	Put(ctx context.Context, organizationID, documentID, filename, contentType string, content []byte) (string, error)
}

type ObjectHead struct {
	SizeBytes int64
	Checksum  string
}

type DirectUploadStorage interface {
	PresignPut(ctx context.Context, organizationID, documentID, uploadID, filename, contentType string, expires time.Duration) (url string, storageRef string, expiresAt time.Time, err error)
	Head(ctx context.Context, storageRef string) (ObjectHead, error)
}

type LocalObjectStorage struct {
	root string
}

func NewLocalObjectStorage(root string) (*LocalObjectStorage, error) {
	if strings.TrimSpace(root) == "" {
		root = "./data/documents"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalObjectStorage{root: filepath.Clean(root)}, nil
}

func (s *LocalObjectStorage) Put(_ context.Context, organizationID, documentID, filename, _ string, content []byte) (string, error) {
	orgDir := filepath.Join(s.root, sanitizeSegment(organizationID))
	if err := os.MkdirAll(orgDir, 0o755); err != nil {
		return "", err
	}
	safeName := sanitizeFilename(filename)
	if safeName == "" {
		safeName = "upload.bin"
	}
	storageRef := filepath.Join(orgDir, fmt.Sprintf("%s_%s", sanitizeSegment(documentID), safeName))
	if err := os.WriteFile(storageRef, content, 0o600); err != nil {
		return "", err
	}
	return storageRef, nil
}

func (s *LocalObjectStorage) PresignPut(_ context.Context, organizationID, documentID, uploadID, filename, _ string, expires time.Duration) (string, string, time.Time, error) {
	safeName := sanitizeFilename(filename)
	if safeName == "" {
		safeName = "upload.bin"
	}
	orgDir := filepath.Join(s.root, sanitizeSegment(organizationID))
	if err := os.MkdirAll(orgDir, 0o755); err != nil {
		return "", "", time.Time{}, err
	}
	ref := filepath.Join(orgDir, sanitizeSegment(documentID)+"_"+sanitizeSegment(uploadID)+"_"+safeName)
	expiresAt := time.Now().UTC().Add(expires)
	return "file://" + ref, ref, expiresAt, nil
}

func (s *LocalObjectStorage) Head(_ context.Context, storageRef string) (ObjectHead, error) {
	info, err := os.Stat(storageRef)
	if err != nil {
		return ObjectHead{}, err
	}
	return ObjectHead{SizeBytes: info.Size()}, nil
}

func sanitizeFilename(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	base := filepath.Base(trimmed)
	base = strings.Trim(base, ".")
	if base == "" {
		return "upload.bin"
	}
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-', r == '_', r == '.', r == ' ':
			return r
		default:
			return '_'
		}
	}, base)
	if base == "" {
		return "upload.bin"
	}
	return base
}

func sanitizeSegment(value string) string {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return "unknown"
	}
	hash := sha256.Sum256([]byte(clean))
	return hex.EncodeToString(hash[:8])
}
