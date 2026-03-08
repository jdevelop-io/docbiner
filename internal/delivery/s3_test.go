package delivery

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/minio/minio-go/v7"
)

// mockS3Client implements s3Uploader for testing.
type mockS3Client struct {
	uploadedBucket  string
	uploadedKey     string
	uploadedData    []byte
	uploadedType    string
	uploadErr       error
}

func (m *mockS3Client) PutObject(_ context.Context, bucketName, objectName string, reader *bytes.Reader, _ int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	if m.uploadErr != nil {
		return minio.UploadInfo{}, m.uploadErr
	}

	m.uploadedBucket = bucketName
	m.uploadedKey = objectName

	data, _ := io_ReadAll(reader)
	m.uploadedData = data
	m.uploadedType = opts.ContentType

	return minio.UploadInfo{
		Bucket: bucketName,
		Key:    objectName,
		Size:   int64(len(data)),
	}, nil
}

// io_ReadAll reads all bytes from a bytes.Reader.
func io_ReadAll(r *bytes.Reader) ([]byte, error) {
	buf := make([]byte, r.Len())
	_, err := r.Read(buf)
	return buf, err
}

// withMockClient sets up a mock client factory for the duration of a test.
func withMockClient(t *testing.T, mock *mockS3Client) {
	t.Helper()

	original := clientFactory
	clientFactory = func(_ string, _ *minio.Options) (s3Uploader, error) {
		return mock, nil
	}
	t.Cleanup(func() {
		clientFactory = original
	})
}

// withFailingFactory sets up a factory that returns an error.
func withFailingFactory(t *testing.T, err error) {
	t.Helper()

	original := clientFactory
	clientFactory = func(_ string, _ *minio.Options) (s3Uploader, error) {
		return nil, err
	}
	t.Cleanup(func() {
		clientFactory = original
	})
}

func TestS3DeliverSuccess(t *testing.T) {
	mock := &mockS3Client{}
	withMockClient(t, mock)

	deliverer := &S3Deliverer{}
	config := S3Config{
		Bucket:    "my-bucket",
		Region:    "us-east-1",
		AccessKey: "AKIAIOSFODNN7EXAMPLE",
		SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	data := []byte("%PDF-1.4 test content")
	url, err := deliverer.Deliver(context.Background(), config, "output.pdf", data, "application/pdf")
	if err != nil {
		t.Fatalf("Deliver() unexpected error: %v", err)
	}

	expectedURL := "https://my-bucket.s3.us-east-1.amazonaws.com/output.pdf"
	if url != expectedURL {
		t.Errorf("Deliver() URL = %q, want %q", url, expectedURL)
	}

	if mock.uploadedBucket != "my-bucket" {
		t.Errorf("PutObject bucket = %q, want %q", mock.uploadedBucket, "my-bucket")
	}

	if mock.uploadedKey != "output.pdf" {
		t.Errorf("PutObject key = %q, want %q", mock.uploadedKey, "output.pdf")
	}

	if !bytes.Equal(mock.uploadedData, data) {
		t.Error("PutObject data does not match input")
	}

	if mock.uploadedType != "application/pdf" {
		t.Errorf("PutObject contentType = %q, want %q", mock.uploadedType, "application/pdf")
	}
}

func TestS3DeliverWithCustomEndpoint(t *testing.T) {
	mock := &mockS3Client{}
	withMockClient(t, mock)

	deliverer := &S3Deliverer{}

	tests := []struct {
		name        string
		config      S3Config
		expectedURL string
	}{
		{
			name: "Cloudflare R2",
			config: S3Config{
				Bucket:    "my-r2-bucket",
				AccessKey: "access",
				SecretKey: "secret",
				Endpoint:  "account-id.r2.cloudflarestorage.com",
			},
			expectedURL: "https://account-id.r2.cloudflarestorage.com/my-r2-bucket/invoice.pdf",
		},
		{
			name: "Google Cloud Storage",
			config: S3Config{
				Bucket:    "my-gcs-bucket",
				AccessKey: "access",
				SecretKey: "secret",
				Endpoint:  "storage.googleapis.com",
			},
			expectedURL: "https://storage.googleapis.com/my-gcs-bucket/invoice.pdf",
		},
		{
			name: "DigitalOcean Spaces",
			config: S3Config{
				Bucket:    "my-space",
				Region:    "nyc3",
				AccessKey: "access",
				SecretKey: "secret",
				Endpoint:  "nyc3.digitaloceanspaces.com",
			},
			expectedURL: "https://nyc3.digitaloceanspaces.com/my-space/invoice.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := deliverer.Deliver(context.Background(), tt.config, "invoice.pdf", []byte("data"), "application/pdf")
			if err != nil {
				t.Fatalf("Deliver() unexpected error: %v", err)
			}

			if url != tt.expectedURL {
				t.Errorf("Deliver() URL = %q, want %q", url, tt.expectedURL)
			}
		})
	}
}

func TestS3DeliverWithPathPrefix(t *testing.T) {
	mock := &mockS3Client{}
	withMockClient(t, mock)

	deliverer := &S3Deliverer{}
	config := S3Config{
		Bucket:    "my-bucket",
		Region:    "eu-west-1",
		AccessKey: "access",
		SecretKey: "secret",
		Path:      "renders/2024/03",
	}

	url, err := deliverer.Deliver(context.Background(), config, "output.pdf", []byte("pdf data"), "application/pdf")
	if err != nil {
		t.Fatalf("Deliver() unexpected error: %v", err)
	}

	expectedURL := "https://my-bucket.s3.eu-west-1.amazonaws.com/renders/2024/03/output.pdf"
	if url != expectedURL {
		t.Errorf("Deliver() URL = %q, want %q", url, expectedURL)
	}

	expectedKey := "renders/2024/03/output.pdf"
	if mock.uploadedKey != expectedKey {
		t.Errorf("PutObject key = %q, want %q", mock.uploadedKey, expectedKey)
	}
}

func TestS3DeliverWithPathPrefixAndCustomEndpoint(t *testing.T) {
	mock := &mockS3Client{}
	withMockClient(t, mock)

	deliverer := &S3Deliverer{}
	config := S3Config{
		Bucket:    "docs",
		AccessKey: "access",
		SecretKey: "secret",
		Endpoint:  "account.r2.cloudflarestorage.com",
		Path:      "invoices",
	}

	url, err := deliverer.Deliver(context.Background(), config, "inv-001.pdf", []byte("data"), "application/pdf")
	if err != nil {
		t.Fatalf("Deliver() unexpected error: %v", err)
	}

	expectedURL := "https://account.r2.cloudflarestorage.com/docs/invoices/inv-001.pdf"
	if url != expectedURL {
		t.Errorf("Deliver() URL = %q, want %q", url, expectedURL)
	}
}

func TestS3DeliverUploadError(t *testing.T) {
	mock := &mockS3Client{
		uploadErr: errors.New("access denied"),
	}
	withMockClient(t, mock)

	deliverer := &S3Deliverer{}
	config := S3Config{
		Bucket:    "my-bucket",
		Region:    "us-east-1",
		AccessKey: "access",
		SecretKey: "secret",
	}

	_, err := deliverer.Deliver(context.Background(), config, "output.pdf", []byte("data"), "application/pdf")
	if err == nil {
		t.Fatal("Deliver() expected error, got nil")
	}

	if !errors.Is(err, mock.uploadErr) {
		t.Errorf("Deliver() error should wrap original: %v", err)
	}
}

func TestS3DeliverClientCreationError(t *testing.T) {
	withFailingFactory(t, errors.New("invalid credentials"))

	deliverer := &S3Deliverer{}
	config := S3Config{
		Bucket:    "my-bucket",
		Region:    "us-east-1",
		AccessKey: "bad",
		SecretKey: "bad",
	}

	_, err := deliverer.Deliver(context.Background(), config, "output.pdf", []byte("data"), "application/pdf")
	if err == nil {
		t.Fatal("Deliver() expected error, got nil")
	}

	expected := "s3 delivery: failed to create client: invalid credentials"
	if err.Error() != expected {
		t.Errorf("Deliver() error = %q, want %q", err.Error(), expected)
	}
}

func TestS3DeliverMissingRegionNoEndpoint(t *testing.T) {
	deliverer := &S3Deliverer{}
	config := S3Config{
		Bucket:    "my-bucket",
		AccessKey: "access",
		SecretKey: "secret",
	}

	_, err := deliverer.Deliver(context.Background(), config, "output.pdf", []byte("data"), "application/pdf")
	if err == nil {
		t.Fatal("Deliver() expected error when region is empty and no endpoint, got nil")
	}

	expected := "s3 delivery: region is required when no custom endpoint is set"
	if err.Error() != expected {
		t.Errorf("Deliver() error = %q, want %q", err.Error(), expected)
	}
}

func TestS3DeliverVariousContentTypes(t *testing.T) {
	mock := &mockS3Client{}
	withMockClient(t, mock)

	deliverer := &S3Deliverer{}
	config := S3Config{
		Bucket:    "bucket",
		Region:    "us-east-1",
		AccessKey: "access",
		SecretKey: "secret",
	}

	tests := []struct {
		key         string
		contentType string
	}{
		{"doc.pdf", "application/pdf"},
		{"image.png", "image/png"},
		{"image.jpg", "image/jpeg"},
		{"page.html", "text/html"},
		{"photo.webp", "image/webp"},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			_, err := deliverer.Deliver(context.Background(), config, tt.key, []byte("data"), tt.contentType)
			if err != nil {
				t.Fatalf("Deliver() unexpected error: %v", err)
			}

			if mock.uploadedType != tt.contentType {
				t.Errorf("PutObject contentType = %q, want %q", mock.uploadedType, tt.contentType)
			}
		})
	}
}
