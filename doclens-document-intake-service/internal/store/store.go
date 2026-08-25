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

type Document struct {
	ID             string
	OrganizationID string
	Type           string
	Filename       string
	ContentType    string
	Status         string
	StorageRef     string
	SizeBytes      int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Store interface {
	CreateDocument(ctx context.Context, doc Document) (Document, error)
	GetDocument(ctx context.Context, organizationID, id string) (Document, error)
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

type ObjectStorage interface {
	Put(ctx context.Context, organizationID, documentID, filename, contentType string, content []byte) (string, error)
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
