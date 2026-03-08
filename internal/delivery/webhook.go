package delivery

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	maxRetries       = 3
	signatureHeader  = "X-Docbiner-Signature"
	contentTypeJSON  = "application/json"
	initialBackoff   = 1 * time.Second
	backoffMultipler = 2
)

// WebhookConfig holds the configuration for a webhook delivery target.
type WebhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Secret  string            `json:"secret"`
}

// WebhookPayload represents the data sent to a webhook endpoint.
type WebhookPayload struct {
	JobID       uuid.UUID `json:"job_id"`
	Status      string    `json:"status"`
	Format      string    `json:"format"`
	ResultURL   string    `json:"result_url"`
	ResultSize  int64     `json:"result_size"`
	PagesCount  int       `json:"pages_count"`
	DurationMs  int64     `json:"duration_ms"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// WebhookSender handles sending webhook notifications with retry and HMAC signing.
type WebhookSender struct {
	client *http.Client
}

// NewWebhookSender creates a new WebhookSender with the given HTTP client.
// If client is nil, a default client with a 10-second timeout is used.
func NewWebhookSender(client *http.Client) *WebhookSender {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &WebhookSender{client: client}
}

// Send delivers a webhook payload to the configured URL.
// It retries up to 3 times with exponential backoff (1s, 2s, 4s) on server errors.
// Client errors (4xx) are not retried.
func (s *WebhookSender) Send(ctx context.Context, config WebhookConfig, payload WebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook: marshal payload: %w", err)
	}

	var signature string
	if config.Secret != "" {
		signature = computeHMAC(body, config.Secret)
	}

	backoff := initialBackoff
	var lastErr error

	for attempt := range maxRetries {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("webhook: context cancelled: %w", err)
		}

		lastErr = s.doRequest(ctx, config, body, signature)
		if lastErr == nil {
			return nil
		}

		// Don't retry on client errors (4xx).
		if isClientError(lastErr) {
			return lastErr
		}

		// Don't wait after the last attempt.
		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("webhook: context cancelled during backoff: %w", ctx.Err())
			case <-time.After(backoff):
			}
			backoff *= backoffMultipler
		}
	}

	return fmt.Errorf("webhook: all %d retries failed: %w", maxRetries, lastErr)
}

// doRequest performs a single HTTP POST to the webhook URL.
func (s *WebhookSender) doRequest(ctx context.Context, config WebhookConfig, body []byte, signature string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook: create request: %w", err)
	}

	req.Header.Set("Content-Type", contentTypeJSON)

	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	if signature != "" {
		req.Header.Set(signatureHeader, signature)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: request failed: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return &ClientError{StatusCode: resp.StatusCode}
	}

	return &ServerError{StatusCode: resp.StatusCode}
}

// computeHMAC computes the HMAC-SHA256 of data using the given secret and returns the hex-encoded result.
func computeHMAC(data []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// ClientError represents a 4xx HTTP response from the webhook target.
type ClientError struct {
	StatusCode int
}

func (e *ClientError) Error() string {
	return fmt.Sprintf("webhook: client error: HTTP %d", e.StatusCode)
}

// ServerError represents a 5xx HTTP response from the webhook target.
type ServerError struct {
	StatusCode int
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("webhook: server error: HTTP %d", e.StatusCode)
}

// isClientError checks if an error is a ClientError (4xx response).
func isClientError(err error) bool {
	_, ok := err.(*ClientError)
	return ok
}
