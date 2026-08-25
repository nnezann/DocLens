package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) Migrate(ctx context.Context, migration string) error {
	_, err := s.pool.Exec(ctx, migration)
	return err
}

func (s *PostgresStore) CreateDocument(ctx context.Context, doc Document) (Document, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO documents
			(id, organization_id, type, filename, content_type, status, processing_job_status, storage_ref, size_bytes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		doc.ID, doc.OrganizationID, doc.Type, doc.Filename, doc.ContentType,
		doc.Status, doc.ProcessingJobStatus, doc.StorageRef, doc.SizeBytes, doc.CreatedAt, doc.UpdatedAt)
	if err != nil {
		return Document{}, fmt.Errorf("insert document: %w", err)
	}
	return doc, nil
}

func (s *PostgresStore) GetDocument(ctx context.Context, organizationID, id string) (Document, error) {
	doc, err := s.scanDocument(ctx, s.pool.QueryRow(ctx, `
		SELECT id, organization_id, type, filename, content_type, status, processing_job_status, storage_ref, size_bytes, created_at, updated_at
		FROM documents
		WHERE organization_id = $1 AND id = $2`, organizationID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrDocumentNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("select document: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, document_id, organization_id, filename, content_type, size_bytes, checksum, storage_ref, idempotency_key, created_at
		FROM uploads
		WHERE organization_id = $1 AND document_id = $2
		ORDER BY created_at`, organizationID, id)
	if err != nil {
		return Document{}, fmt.Errorf("select uploads: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var upload UploadRecord
		if err := rows.Scan(&upload.ID, &upload.DocumentID, &upload.OrganizationID, &upload.Filename, &upload.ContentType, &upload.SizeBytes, &upload.Checksum, &upload.StorageRef, &upload.IdempotencyKey, &upload.CreatedAt); err != nil {
			return Document{}, fmt.Errorf("scan upload: %w", err)
		}
		doc.Uploads = append(doc.Uploads, upload)
	}
	if err := rows.Err(); err != nil {
		return Document{}, fmt.Errorf("iterate uploads: %w", err)
	}
	return doc, nil
}

func (s *PostgresStore) UploadDocument(ctx context.Context, organizationID, documentID string, upload UploadRecord) (Document, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Document{}, fmt.Errorf("begin upload transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var storedOrganization string
	if err := tx.QueryRow(ctx, `SELECT organization_id FROM documents WHERE id = $1 FOR UPDATE`, documentID).Scan(&storedOrganization); errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrDocumentNotFound
	} else if err != nil {
		return Document{}, fmt.Errorf("lock document: %w", err)
	}
	if storedOrganization != organizationID {
		return Document{}, ErrOrganizationMismatch
	}
	if upload.IdempotencyKey != "" {
		var existing UploadRecord
		err := tx.QueryRow(ctx, `
			SELECT id, document_id, organization_id, filename, content_type, size_bytes, checksum, storage_ref, idempotency_key, created_at
			FROM uploads WHERE organization_id = $1 AND idempotency_key = $2`,
			organizationID, upload.IdempotencyKey).Scan(&existing.ID, &existing.DocumentID, &existing.OrganizationID, &existing.Filename, &existing.ContentType, &existing.SizeBytes, &existing.Checksum, &existing.StorageRef, &existing.IdempotencyKey, &existing.CreatedAt)
		if err == nil {
			doc, getErr := s.getDocumentTx(ctx, tx, organizationID, documentID)
			if getErr != nil {
				return Document{}, getErr
			}
			doc.Uploads = append(doc.Uploads, existing)
			return doc, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return Document{}, fmt.Errorf("check idempotency key: %w", err)
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO uploads
			(id, document_id, organization_id, filename, content_type, size_bytes, checksum, storage_ref, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		upload.ID, upload.DocumentID, upload.OrganizationID, upload.Filename, upload.ContentType,
		upload.SizeBytes, upload.Checksum, upload.StorageRef, upload.IdempotencyKey, upload.CreatedAt)
	if err != nil {
		return Document{}, fmt.Errorf("insert upload: %w", err)
	}
	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		UPDATE documents
		SET status = 'processing', processing_job_status = 'queued', storage_ref = $1, size_bytes = $2, updated_at = $3
		WHERE organization_id = $4 AND id = $5`,
		upload.StorageRef, upload.SizeBytes, now, organizationID, documentID)
	if err != nil {
		return Document{}, fmt.Errorf("update document status: %w", err)
	}
	doc, err := s.getDocumentTx(ctx, tx, organizationID, documentID)
	if err != nil {
		return Document{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, fmt.Errorf("commit upload transaction: %w", err)
	}
	doc.Uploads = append(doc.Uploads, upload)
	doc.Status = "processing"
	doc.ProcessingJobStatus = "queued"
	doc.UpdatedAt = now
	return doc, nil
}

func (s *PostgresStore) GetDocumentStatus(ctx context.Context, organizationID, documentID string) (Document, error) {
	return s.GetDocument(ctx, organizationID, documentID)
}

func (s *PostgresStore) getDocumentTx(ctx context.Context, tx pgx.Tx, organizationID, id string) (Document, error) {
	return s.scanDocument(ctx, tx.QueryRow(ctx, `
		SELECT id, organization_id, type, filename, content_type, status, processing_job_status, storage_ref, size_bytes, created_at, updated_at
		FROM documents WHERE organization_id = $1 AND id = $2`, organizationID, id))
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *PostgresStore) scanDocument(_ context.Context, row rowScanner) (Document, error) {
	var doc Document
	err := row.Scan(&doc.ID, &doc.OrganizationID, &doc.Type, &doc.Filename, &doc.ContentType, &doc.Status, &doc.ProcessingJobStatus, &doc.StorageRef, &doc.SizeBytes, &doc.CreatedAt, &doc.UpdatedAt)
	return doc, err
}
