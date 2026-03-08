// Package delivery provides delivery mechanisms for rendered documents.
// This file implements S3-compatible object storage delivery.
package delivery

import (
	"bytes"
	"context"
	"fmt"
	"path"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config holds the configuration for an S3-compatible storage destination.
type S3Config struct {
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Endpoint  string `json:"endpoint"` // for R2, GCS, etc.
	Path      string `json:"path"`     // prefix path in bucket
}

// S3Deliverer delivers rendered documents to S3-compatible object storage.
type S3Deliverer struct{}

// s3ClientFactory abstracts Minio client creation for testing.
type s3ClientFactory func(endpoint string, opts *minio.Options) (s3Uploader, error)

// s3Uploader abstracts the Minio client methods used by S3Deliverer.
type s3Uploader interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader *bytes.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

// defaultS3ClientFactory creates a real Minio client.
func defaultS3ClientFactory(endpoint string, opts *minio.Options) (s3Uploader, error) {
	client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, err
	}

	return &minioClientAdapter{client: client}, nil
}

// minioClientAdapter adapts *minio.Client to the s3Uploader interface
// by wrapping PutObject to accept *bytes.Reader instead of io.Reader.
type minioClientAdapter struct {
	client *minio.Client
}

func (a *minioClientAdapter) PutObject(ctx context.Context, bucketName, objectName string, reader *bytes.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	return a.client.PutObject(ctx, bucketName, objectName, reader, objectSize, opts)
}

// clientFactory is the factory used to create S3 clients.
// It can be overridden in tests.
var clientFactory s3ClientFactory = defaultS3ClientFactory

// Deliver uploads a file to the customer's S3-compatible bucket.
// It creates a new client with the customer's credentials, uploads the file
// at {path}/{key}, and returns the S3 URL of the uploaded object.
func (d *S3Deliverer) Deliver(ctx context.Context, config S3Config, key string, data []byte, contentType string) (string, error) {
	endpoint := config.Endpoint
	useSSL := true

	if endpoint == "" {
		if config.Region == "" {
			return "", fmt.Errorf("s3 delivery: region is required when no custom endpoint is set")
		}

		endpoint = "s3." + config.Region + ".amazonaws.com"
	}

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: useSSL,
		Region: config.Region,
	}

	client, err := clientFactory(endpoint, opts)
	if err != nil {
		return "", fmt.Errorf("s3 delivery: failed to create client: %w", err)
	}

	objectKey := key
	if config.Path != "" {
		objectKey = path.Join(config.Path, key)
	}

	reader := bytes.NewReader(data)

	_, err = client.PutObject(ctx, config.Bucket, objectKey, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("s3 delivery: failed to upload to %s/%s: %w", config.Bucket, objectKey, err)
	}

	var url string
	if config.Endpoint != "" {
		url = fmt.Sprintf("https://%s/%s/%s", config.Endpoint, config.Bucket, objectKey)
	} else {
		url = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", config.Bucket, config.Region, objectKey)
	}

	return url, nil
}
