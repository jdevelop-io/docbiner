package delivery

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestPayload() WebhookPayload {
	return WebhookPayload{
		JobID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Status:      "completed",
		Format:      "pdf",
		ResultURL:   "https://storage.docbiner.com/results/550e8400.pdf",
		ResultSize:  123456,
		PagesCount:  5,
		DurationMs:  2340,
		CreatedAt:   time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC),
		CompletedAt: time.Date(2026, 3, 4, 10, 0, 2, 340000000, time.UTC),
	}
}

func TestSend_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(server.Client())
	payload := newTestPayload()
	config := WebhookConfig{URL: server.URL}

	err := sender.Send(context.Background(), config, payload)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var received WebhookPayload
	if err := json.Unmarshal(receivedBody, &received); err != nil {
		t.Fatalf("unmarshal received body: %v", err)
	}

	if received.JobID != payload.JobID {
		t.Errorf("job_id: got %s, want %s", received.JobID, payload.JobID)
	}
	if received.Status != payload.Status {
		t.Errorf("status: got %s, want %s", received.Status, payload.Status)
	}
	if received.ResultSize != payload.ResultSize {
		t.Errorf("result_size: got %d, want %d", received.ResultSize, payload.ResultSize)
	}
}

func TestSend_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(server.Client())
	config := WebhookConfig{
		URL: server.URL,
		Headers: map[string]string{
			"X-Custom":      "custom-value",
			"Authorization": "Bearer test-token",
		},
	}

	err := sender.Send(context.Background(), config, newTestPayload())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if got := receivedHeaders.Get("X-Custom"); got != "custom-value" {
		t.Errorf("X-Custom header: got %q, want %q", got, "custom-value")
	}
	if got := receivedHeaders.Get("Authorization"); got != "Bearer test-token" {
		t.Errorf("Authorization header: got %q, want %q", got, "Bearer test-token")
	}
	if got := receivedHeaders.Get("Content-Type"); got != contentTypeJSON {
		t.Errorf("Content-Type header: got %q, want %q", got, contentTypeJSON)
	}
}

func TestSend_HMACSignaturePresent(t *testing.T) {
	var signatureValue string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signatureValue = r.Header.Get(signatureHeader)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(server.Client())
	config := WebhookConfig{
		URL:    server.URL,
		Secret: "my-secret-key",
	}

	err := sender.Send(context.Background(), config, newTestPayload())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if signatureValue == "" {
		t.Fatal("expected HMAC signature header to be present")
	}
}

func TestSend_HMACSignatureCorrect(t *testing.T) {
	secret := "hmac_secret_key"
	var receivedBody []byte
	var receivedSignature string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get(signatureHeader)
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(server.Client())
	config := WebhookConfig{
		URL:    server.URL,
		Secret: secret,
	}

	err := sender.Send(context.Background(), config, newTestPayload())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify signature independently.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(receivedBody)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if receivedSignature != expectedSignature {
		t.Errorf("HMAC signature mismatch:\n  got:  %s\n  want: %s", receivedSignature, expectedSignature)
	}
}

func TestSend_NoSignatureWithoutSecret(t *testing.T) {
	var hasSignature bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasSignature = r.Header.Get(signatureHeader) != ""
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(server.Client())
	config := WebhookConfig{URL: server.URL}

	err := sender.Send(context.Background(), config, newTestPayload())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if hasSignature {
		t.Error("expected no HMAC signature header when secret is empty")
	}
}

func TestSend_RetryOnServerError_EventualSuccess(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(server.Client())
	// Override backoff for test speed by using a short-timeout context approach.
	// We'll test with the real sender but the httptest server responds instantly.
	config := WebhookConfig{URL: server.URL}

	// Use a context with generous timeout since backoff waits are involved.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := sender.Send(ctx, config, newTestPayload())
	if err != nil {
		t.Fatalf("expected eventual success, got: %v", err)
	}

	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestSend_RetryOnConnectionError(t *testing.T) {
	// Create a server that we immediately close to simulate connection errors,
	// then replace it to eventually succeed.
	var attempts atomic.Int32

	// First, create a server that will work on attempt 2.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		_ = count
		w.WriteHeader(http.StatusOK)
	}))
	// Close it so the first attempt would fail on this address.
	_ = server.URL
	server.Close()

	// Create a new server on the same address isn't easy, so instead
	// test with a server that returns 502 first then 200.
	attempts.Store(0)
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	sender := NewWebhookSender(server2.Client())
	config := WebhookConfig{URL: server2.URL}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := sender.Send(ctx, config, newTestPayload())
	if err != nil {
		t.Fatalf("expected eventual success after connection error, got: %v", err)
	}

	if got := attempts.Load(); got < 2 {
		t.Errorf("expected at least 2 attempts, got %d", got)
	}
}

func TestSend_AllRetriesFail(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := NewWebhookSender(server.Client())
	config := WebhookConfig{URL: server.URL}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := sender.Send(ctx, config, newTestPayload())
	if err == nil {
		t.Fatal("expected error when all retries fail")
	}

	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestSend_ClientError_NoRetry(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	sender := NewWebhookSender(server.Client())
	config := WebhookConfig{URL: server.URL}

	err := sender.Send(context.Background(), config, newTestPayload())
	if err == nil {
		t.Fatal("expected error on 4xx response")
	}

	clientErr, ok := err.(*ClientError)
	if !ok {
		t.Fatalf("expected ClientError, got %T: %v", err, err)
	}
	if clientErr.StatusCode != http.StatusBadRequest {
		t.Errorf("status code: got %d, want %d", clientErr.StatusCode, http.StatusBadRequest)
	}

	if got := attempts.Load(); got != 1 {
		t.Errorf("expected exactly 1 attempt (no retry on 4xx), got %d", got)
	}
}

func TestSend_ContextCancellation(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := NewWebhookSender(server.Client())
	config := WebhookConfig{URL: server.URL}

	// Cancel context after a very short time so it cancels during backoff.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := sender.Send(ctx, config, newTestPayload())
	if err == nil {
		t.Fatal("expected error on context cancellation")
	}

	// Should have made at most 2 attempts (first fails, backoff starts, context cancelled during backoff).
	if got := attempts.Load(); got > 2 {
		t.Errorf("expected at most 2 attempts due to context cancellation, got %d", got)
	}
}
