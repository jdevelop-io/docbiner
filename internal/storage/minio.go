// Package storage provides a wrapper around Minio/S3-compatible object storage
// for temporary file storage in Docbiner.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectStorage defines the interface for object storage operations.
type ObjectStorage interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)
	SignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

// MinioStorage wraps a Minio client for temporary file storage.
type MinioStorage struct {
	client *minio.Client
	bucket string
}

// New creates a new MinioStorage instance connected to the given endpoint.
func New(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: failed to create minio client: %w", err)
	}

	return &MinioStorage{
		client: client,
		bucket: bucket,
	}, nil
}

// Upload puts an object in the bucket and returns the object key.
func (s *MinioStorage) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)

	_, err := s.client.PutObject(ctx, s.bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("storage: failed to upload object %q: %w", key, err)
	}

	return key, nil
}

// SignedURL generates a presigned GET URL for the given key with the specified expiry duration.
func (s *MinioStorage) SignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)

	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("storage: failed to generate signed URL for %q: %w", key, err)
	}

	return presignedURL.String(), nil
}

// Delete removes an object from the bucket.
func (s *MinioStorage) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("storage: failed to delete object %q: %w", key, err)
	}

	return nil
}

// EnsureBucket creates the bucket if it does not already exist.
func (s *MinioStorage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("storage: failed to check bucket %q: %w", s.bucket, err)
	}

	if exists {
		return nil
	}

	err = s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf("storage: failed to create bucket %q: %w", s.bucket, err)
	}

	return nil
}
