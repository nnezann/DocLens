package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2ObjectStorage struct {
	client *s3.Client
	bucket string
}

func NewR2ObjectStorage(client *s3.Client, bucket string) (*R2ObjectStorage, error) {
	if client == nil {
		return nil, fmt.Errorf("r2 client is required")
	}
	if strings.TrimSpace(bucket) == "" {
		return nil, fmt.Errorf("r2 bucket is required")
	}
	return &R2ObjectStorage{client: client, bucket: bucket}, nil
}

func (s *R2ObjectStorage) Put(ctx context.Context, organizationID, documentID, filename, contentType string, content []byte) (string, error) {
	checksum := sha256.Sum256(content)
	key := fmt.Sprintf("organizations/%s/documents/%s/uploads/%s-%s", url.PathEscape(organizationID), url.PathEscape(documentID), hex.EncodeToString(checksum[:8]), url.PathEscape(sanitizeFilename(filename)))
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}
	return key, nil
}

func (s *R2ObjectStorage) PresignPut(ctx context.Context, organizationID, documentID, uploadID, filename, contentType string, expires time.Duration) (string, string, time.Time, error) {
	key := objectKey(organizationID, documentID, uploadID, filename)
	presigner := s3.NewPresignClient(s.client)
	result, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), ContentType: aws.String(contentType),
	}, func(options *s3.PresignOptions) { options.Expires = expires })
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("presign upload: %w", err)
	}
	return result.URL, key, time.Now().UTC().Add(expires), nil
}

func (s *R2ObjectStorage) Head(ctx context.Context, storageRef string) (ObjectHead, error) {
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(storageRef)})
	if err != nil {
		return ObjectHead{}, fmt.Errorf("head object: %w", err)
	}
	head := ObjectHead{}
	if result.ContentLength != nil {
		head.SizeBytes = *result.ContentLength
	}
	if result.ChecksumSHA256 != nil {
		head.Checksum = *result.ChecksumSHA256
	}
	return head, nil
}

func objectKey(organizationID, documentID, uploadID, filename string) string {
	return fmt.Sprintf("organizations/%s/documents/%s/uploads/%s-%s",
		url.PathEscape(organizationID), url.PathEscape(documentID), url.PathEscape(uploadID),
		url.PathEscape(sanitizeFilename(filename)))
}
