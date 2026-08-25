package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

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
