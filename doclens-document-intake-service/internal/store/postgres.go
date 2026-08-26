package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

type OutboxRecord struct {
	ID           string
	RoutingKey   string
	Payload      []byte
	AttemptCount int
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

func (s *PostgresStore) ClaimOutbox(ctx context.Context, limit int) ([]OutboxRecord, error) {
	rows, err := s.pool.Query(ctx, `
		WITH claimed AS (
			SELECT id
			FROM event_outbox
			WHERE status IN ('pending', 'failed') AND next_attempt_at <= NOW()
			ORDER BY next_attempt_at, created_at
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE event_outbox e
		SET status = 'publishing', attempt_count = e.attempt_count + 1
		FROM claimed
		WHERE e.id = claimed.id
		RETURNING e.id, e.routing_key, e.payload, e.attempt_count`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim event outbox records: %w", err)
	}
	defer rows.Close()
	var records []OutboxRecord
	for rows.Next() {
		var record OutboxRecord
		if err := rows.Scan(&record.ID, &record.RoutingKey, &record.Payload, &record.AttemptCount); err != nil {
			return nil, fmt.Errorf("scan event outbox record: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *PostgresStore) MarkOutboxPublished(ctx context.Context, id string, publishedAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE event_outbox SET status = 'published', published_at = $1, last_error = ''
		WHERE id = $2`, publishedAt, id)
	return err
}

func (s *PostgresStore) MarkOutboxFailed(ctx context.Context, id, lastError string, nextAttemptAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE event_outbox SET status = 'failed', last_error = LEFT($1, 1000), next_attempt_at = $2
		WHERE id = $3`, lastError, nextAttemptAt, id)
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
	_, err = s.pool.Exec(ctx, `
		INSERT INTO processing_jobs
			(id, document_id, organization_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		"job_"+doc.ID, doc.ID, doc.OrganizationID, doc.ProcessingJobStatus, doc.CreatedAt, doc.UpdatedAt)
	if err != nil {
		return Document{}, fmt.Errorf("insert processing job: %w", err)
	}
	return doc, nil
}

func (s *PostgresStore) CreateDocumentAndQueue(ctx context.Context, doc Document, upload UploadRecord, event OutboxEvent) (Document, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Document{}, fmt.Errorf("begin document transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO documents
			(id, organization_id, type, filename, content_type, status, processing_job_status, storage_ref, size_bytes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		doc.ID, doc.OrganizationID, doc.Type, doc.Filename, doc.ContentType,
		doc.Status, doc.ProcessingJobStatus, doc.StorageRef, doc.SizeBytes, doc.CreatedAt, doc.UpdatedAt)
	if err != nil {
		return Document{}, fmt.Errorf("insert document: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO processing_jobs
			(id, document_id, organization_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		"job_"+doc.ID, doc.ID, doc.OrganizationID, doc.ProcessingJobStatus, doc.CreatedAt, doc.UpdatedAt)
	if err != nil {
		return Document{}, fmt.Errorf("insert processing job: %w", err)
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
	if !json.Valid(event.Payload) {
		return Document{}, fmt.Errorf("outbox payload is not valid json")
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO event_outbox
			(id, event_type, event_version, routing_key, organization_id, document_id, payload, status, next_attempt_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $8)`,
		event.ID, event.EventType, event.EventVersion, event.RoutingKey, event.OrganizationID, event.DocumentID, event.Payload, event.CreatedAt)
	if err != nil {
		return Document{}, fmt.Errorf("insert event outbox record: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Document{}, fmt.Errorf("commit document transaction: %w", err)
	}
	doc.Uploads = append(doc.Uploads, upload)
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
		SELECT id, document_id, organization_id, filename, content_type, size_bytes, checksum, storage_ref,
			idempotency_key, status, upload_method, expected_size_bytes, confirmed_at, created_at
		FROM uploads
		WHERE organization_id = $1 AND document_id = $2
		ORDER BY created_at`, organizationID, id)
	if err != nil {
		return Document{}, fmt.Errorf("select uploads: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var upload UploadRecord
		if err := rows.Scan(&upload.ID, &upload.DocumentID, &upload.OrganizationID, &upload.Filename, &upload.ContentType,
			&upload.SizeBytes, &upload.Checksum, &upload.StorageRef, &upload.IdempotencyKey, &upload.Status,
			&upload.UploadMethod, &upload.ExpectedSize, &upload.ConfirmedAt, &upload.CreatedAt); err != nil {
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
	return s.uploadDocument(ctx, organizationID, documentID, upload, nil)
}

func (s *PostgresStore) CreateUploadIntent(ctx context.Context, organizationID, documentID string, upload UploadRecord) (UploadRecord, error) {
	var owner string
	if err := s.pool.QueryRow(ctx, `SELECT organization_id FROM documents WHERE id = $1`, documentID).Scan(&owner); errors.Is(err, pgx.ErrNoRows) {
		return UploadRecord{}, ErrDocumentNotFound
	} else if err != nil {
		return UploadRecord{}, fmt.Errorf("load document for upload intent: %w", err)
	} else if owner != organizationID {
		return UploadRecord{}, ErrOrganizationMismatch
	}
	if upload.IdempotencyKey != "" {
		var existing UploadRecord
		err := s.pool.QueryRow(ctx, `
			SELECT id, document_id, organization_id, filename, content_type, size_bytes, checksum, storage_ref,
				idempotency_key, status, upload_method, expected_size_bytes, confirmed_at, created_at
			FROM uploads WHERE organization_id = $1 AND idempotency_key = $2`,
			organizationID, upload.IdempotencyKey).Scan(
			&existing.ID, &existing.DocumentID, &existing.OrganizationID, &existing.Filename,
			&existing.ContentType, &existing.SizeBytes, &existing.Checksum, &existing.StorageRef,
			&existing.IdempotencyKey, &existing.Status, &existing.UploadMethod, &existing.ExpectedSize,
			&existing.ConfirmedAt, &existing.CreatedAt)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return UploadRecord{}, fmt.Errorf("check upload intent idempotency: %w", err)
		}
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO uploads
			(id, document_id, organization_id, filename, content_type, size_bytes, checksum, storage_ref,
			 idempotency_key, status, upload_method, expected_size_bytes, created_at)
		VALUES ($1, $2, $3, $4, $5, 0, $6, $7, $8, 'pending', 'presigned_direct', $9, $10)`,
		upload.ID, upload.DocumentID, upload.OrganizationID, upload.Filename, upload.ContentType,
		upload.Checksum, upload.StorageRef, upload.IdempotencyKey, upload.ExpectedSize, upload.CreatedAt)
	if err != nil {
		return UploadRecord{}, fmt.Errorf("insert upload intent: %w", err)
	}
	upload.Status, upload.UploadMethod = "pending", "presigned_direct"
	return upload, nil
}

func (s *PostgresStore) ConfirmUpload(ctx context.Context, organizationID, documentID, uploadID string, sizeBytes int64, checksum string, event OutboxEvent) (UploadRecord, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return UploadRecord{}, fmt.Errorf("begin upload confirmation: %w", err)
	}
	defer tx.Rollback(ctx)
	var upload UploadRecord
	err = tx.QueryRow(ctx, `
		SELECT id, document_id, organization_id, filename, content_type, size_bytes, checksum, storage_ref,
			idempotency_key, status, upload_method, expected_size_bytes, confirmed_at, created_at
		FROM uploads WHERE organization_id = $1 AND document_id = $2 AND id = $3 FOR UPDATE`,
		organizationID, documentID, uploadID).Scan(
		&upload.ID, &upload.DocumentID, &upload.OrganizationID, &upload.Filename, &upload.ContentType,
		&upload.SizeBytes, &upload.Checksum, &upload.StorageRef, &upload.IdempotencyKey, &upload.Status,
		&upload.UploadMethod, &upload.ExpectedSize, &upload.ConfirmedAt, &upload.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return UploadRecord{}, ErrUploadNotFound
	}
	if err != nil {
		return UploadRecord{}, fmt.Errorf("load upload intent: %w", err)
	}
	if upload.Status == "confirmed" {
		return upload, nil
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
	_, err = tx.Exec(ctx, `
		UPDATE uploads SET status = 'confirmed', size_bytes = $1, checksum = $2, confirmed_at = $3
		WHERE organization_id = $4 AND document_id = $5 AND id = $6`,
		sizeBytes, checksum, now, organizationID, documentID, uploadID)
	if err != nil {
		return UploadRecord{}, fmt.Errorf("confirm upload: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE documents SET status = 'processing', processing_job_status = 'queued', storage_ref = $1, size_bytes = $2, updated_at = $3
		WHERE organization_id = $4 AND id = $5`, upload.StorageRef, sizeBytes, now, organizationID, documentID)
	if err != nil {
		return UploadRecord{}, fmt.Errorf("update confirmed document: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO processing_jobs (id, document_id, organization_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'queued', $4, $4)
		ON CONFLICT (id) DO UPDATE SET status = 'queued', updated_at = EXCLUDED.updated_at`,
		"job_"+documentID, documentID, organizationID, now)
	if err != nil {
		return UploadRecord{}, fmt.Errorf("update processing job: %w", err)
	}
	if !json.Valid(event.Payload) {
		return UploadRecord{}, fmt.Errorf("outbox payload is not valid json")
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO event_outbox
			(id, event_type, event_version, routing_key, organization_id, document_id, payload, status, next_attempt_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $8)
		ON CONFLICT (id) DO NOTHING`,
		event.ID, event.EventType, event.EventVersion, event.RoutingKey, event.OrganizationID,
		event.DocumentID, event.Payload, now)
	if err != nil {
		return UploadRecord{}, fmt.Errorf("queue confirmed upload event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return UploadRecord{}, fmt.Errorf("commit upload confirmation: %w", err)
	}
	upload.Status, upload.SizeBytes, upload.Checksum, upload.ConfirmedAt = "confirmed", sizeBytes, checksum, &now
	return upload, nil
}

func (s *PostgresStore) UploadDocumentAndQueue(ctx context.Context, organizationID, documentID string, upload UploadRecord, event OutboxEvent) (Document, error) {
	return s.uploadDocument(ctx, organizationID, documentID, upload, &event)
}

func (s *PostgresStore) uploadDocument(ctx context.Context, organizationID, documentID string, upload UploadRecord, event *OutboxEvent) (Document, error) {
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
			SELECT id, document_id, organization_id, filename, content_type, size_bytes, checksum, storage_ref,
				idempotency_key, status, upload_method, expected_size_bytes, confirmed_at, created_at
			FROM uploads WHERE organization_id = $1 AND idempotency_key = $2`,
			organizationID, upload.IdempotencyKey).Scan(&existing.ID, &existing.DocumentID, &existing.OrganizationID, &existing.Filename, &existing.ContentType,
			&existing.SizeBytes, &existing.Checksum, &existing.StorageRef, &existing.IdempotencyKey, &existing.Status,
			&existing.UploadMethod, &existing.ExpectedSize, &existing.ConfirmedAt, &existing.CreatedAt)
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
	_, err = tx.Exec(ctx, `
		INSERT INTO processing_jobs
			(id, document_id, organization_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'queued', $4, $4)
		ON CONFLICT (id) DO UPDATE SET status = 'queued', updated_at = EXCLUDED.updated_at`,
		"job_"+documentID, documentID, organizationID, now)
	if err != nil {
		return Document{}, fmt.Errorf("update processing job: %w", err)
	}
	if event != nil {
		if !json.Valid(event.Payload) {
			return Document{}, fmt.Errorf("outbox payload is not valid json")
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO event_outbox
				(id, event_type, event_version, routing_key, organization_id, document_id, payload, status, next_attempt_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $8)`,
			event.ID, event.EventType, event.EventVersion, event.RoutingKey, event.OrganizationID, event.DocumentID, event.Payload, event.CreatedAt)
		if err != nil {
			return Document{}, fmt.Errorf("insert event outbox record: %w", err)
		}
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
