package storage

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

// mockStorage implements ObjectStorage for testing.
type mockStorage struct {
	objects     map[string]mockObject
	uploadErr   error
	signedErr   error
	deleteErr   error
	signedURL   string
}

type mockObject struct {
	data        []byte
	contentType string
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		objects: make(map[string]mockObject),
	}
}

func (m *mockStorage) Upload(_ context.Context, key string, data []byte, contentType string) (string, error) {
	if m.uploadErr != nil {
		return "", m.uploadErr
	}

	m.objects[key] = mockObject{
		data:        data,
		contentType: contentType,
	}

	return key, nil
}

func (m *mockStorage) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	if m.signedErr != nil {
		return "", m.signedErr
	}

	if _, ok := m.objects[key]; !ok {
		return "", errors.New("object not found")
	}

	if m.signedURL != "" {
		return m.signedURL, nil
	}

	return "https://storage.example.com/" + key + "?token=abc123", nil
}

func (m *mockStorage) Delete(_ context.Context, key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}

	if _, ok := m.objects[key]; !ok {
		return errors.New("object not found")
	}

	delete(m.objects, key)

	return nil
}

func TestInterfaceCompliance(t *testing.T) {
	// Verify that MinioStorage implements ObjectStorage at compile time.
	var _ ObjectStorage = (*MinioStorage)(nil)
}

func TestMockUpload(t *testing.T) {
	ctx := context.Background()
	store := newMockStorage()

	key := "renders/abc123/output.pdf"
	data := []byte("%PDF-1.4 mock content")
	contentType := "application/pdf"

	returnedKey, err := store.Upload(ctx, key, data, contentType)
	if err != nil {
		t.Fatalf("Upload() unexpected error: %v", err)
	}

	if returnedKey != key {
		t.Errorf("Upload() returned key = %q, want %q", returnedKey, key)
	}

	obj, ok := store.objects[key]
	if !ok {
		t.Fatal("Upload() object not found in store")
	}

	if !bytes.Equal(obj.data, data) {
		t.Error("Upload() stored data does not match input")
	}

	if obj.contentType != contentType {
		t.Errorf("Upload() stored contentType = %q, want %q", obj.contentType, contentType)
	}
}

func TestMockUploadError(t *testing.T) {
	ctx := context.Background()
	store := newMockStorage()
	store.uploadErr = errors.New("connection refused")

	_, err := store.Upload(ctx, "test.pdf", []byte("data"), "application/pdf")
	if err == nil {
		t.Fatal("Upload() expected error, got nil")
	}

	if err.Error() != "connection refused" {
		t.Errorf("Upload() error = %q, want %q", err.Error(), "connection refused")
	}
}

func TestMockSignedURL(t *testing.T) {
	ctx := context.Background()
	store := newMockStorage()

	key := "renders/abc123/output.png"
	store.objects[key] = mockObject{
		data:        []byte("png data"),
		contentType: "image/png",
	}

	url, err := store.SignedURL(ctx, key, 15*time.Minute)
	if err != nil {
		t.Fatalf("SignedURL() unexpected error: %v", err)
	}

	if url == "" {
		t.Error("SignedURL() returned empty URL")
	}

	expected := "https://storage.example.com/renders/abc123/output.png?token=abc123"
	if url != expected {
		t.Errorf("SignedURL() = %q, want %q", url, expected)
	}
}

func TestMockSignedURLNotFound(t *testing.T) {
	ctx := context.Background()
	store := newMockStorage()

	_, err := store.SignedURL(ctx, "nonexistent.pdf", 15*time.Minute)
	if err == nil {
		t.Fatal("SignedURL() expected error for missing object, got nil")
	}
}

func TestMockSignedURLError(t *testing.T) {
	ctx := context.Background()
	store := newMockStorage()
	store.signedErr = errors.New("presign failed")

	store.objects["test.pdf"] = mockObject{}

	_, err := store.SignedURL(ctx, "test.pdf", 15*time.Minute)
	if err == nil {
		t.Fatal("SignedURL() expected error, got nil")
	}
}

func TestMockDelete(t *testing.T) {
	ctx := context.Background()
	store := newMockStorage()

	key := "renders/abc123/output.pdf"
	store.objects[key] = mockObject{
		data:        []byte("pdf data"),
		contentType: "application/pdf",
	}

	err := store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	if _, ok := store.objects[key]; ok {
		t.Error("Delete() object still present in store after deletion")
	}
}

func TestMockDeleteNotFound(t *testing.T) {
	ctx := context.Background()
	store := newMockStorage()

	err := store.Delete(ctx, "nonexistent.pdf")
	if err == nil {
		t.Fatal("Delete() expected error for missing object, got nil")
	}
}

func TestMockDeleteError(t *testing.T) {
	ctx := context.Background()
	store := newMockStorage()
	store.deleteErr = errors.New("permission denied")

	store.objects["test.pdf"] = mockObject{}

	err := store.Delete(ctx, "test.pdf")
	if err == nil {
		t.Fatal("Delete() expected error, got nil")
	}

	if err.Error() != "permission denied" {
		t.Errorf("Delete() error = %q, want %q", err.Error(), "permission denied")
	}
}

func TestMockUploadVariousContentTypes(t *testing.T) {
	ctx := context.Background()
	store := newMockStorage()

	tests := []struct {
		key         string
		contentType string
	}{
		{"output.pdf", "application/pdf"},
		{"output.png", "image/png"},
		{"output.jpg", "image/jpeg"},
		{"output.html", "text/html"},
		{"output.webp", "image/webp"},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			returnedKey, err := store.Upload(ctx, tt.key, []byte("data"), tt.contentType)
			if err != nil {
				t.Fatalf("Upload() unexpected error: %v", err)
			}

			if returnedKey != tt.key {
				t.Errorf("Upload() returned key = %q, want %q", returnedKey, tt.key)
			}

			obj := store.objects[tt.key]
			if obj.contentType != tt.contentType {
				t.Errorf("Upload() stored contentType = %q, want %q", obj.contentType, tt.contentType)
			}
		})
	}
}

func TestMockUploadThenSignedURLThenDelete(t *testing.T) {
	ctx := context.Background()
	store := newMockStorage()

	key := "workflow/test/doc.pdf"
	data := []byte("full workflow test")

	// Upload
	_, err := store.Upload(ctx, key, data, "application/pdf")
	if err != nil {
		t.Fatalf("Upload() unexpected error: %v", err)
	}

	// SignedURL
	url, err := store.SignedURL(ctx, key, 30*time.Minute)
	if err != nil {
		t.Fatalf("SignedURL() unexpected error: %v", err)
	}

	if url == "" {
		t.Error("SignedURL() returned empty URL")
	}

	// Delete
	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	// Verify deleted
	_, err = store.SignedURL(ctx, key, 30*time.Minute)
	if err == nil {
		t.Error("SignedURL() after Delete() should return error")
	}
}

func TestNewMinioStorageInvalidEndpoint(t *testing.T) {
	// New should not fail even with seemingly invalid endpoints,
	// as Minio client validates lazily. We just verify it returns
	// a non-nil storage.
	s, err := New("localhost:9000", "minioadmin", "minioadmin", "test-bucket", false)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if s == nil {
		t.Fatal("New() returned nil storage")
	}

	if s.bucket != "test-bucket" {
		t.Errorf("New() bucket = %q, want %q", s.bucket, "test-bucket")
	}
}
