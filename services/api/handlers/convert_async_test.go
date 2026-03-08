package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/docbiner/docbiner/internal/queue"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Queue Publisher ---

type mockPublisher struct {
	err        error
	lastMsg    queue.JobMessage
	publishCalled bool
}

func (m *mockPublisher) Publish(_ context.Context, msg queue.JobMessage) error {
	m.publishCalled = true
	m.lastMsg = msg
	return m.err
}

// --- Test Helpers ---

func setupAsyncConvertTest(j JobStore, p QueuePublisher) (*echo.Echo, *ConvertAsyncHandler) {
	e := echo.New()
	h := NewConvertAsyncHandler(j, p)

	e.POST("/v1/convert/async", func(c echo.Context) error {
		// Simulate auth middleware setting context values.
		c.Set("org_id", uuid.New())
		c.Set("api_key_id", uuid.New())
		c.Set("environment", domain.APIKeyEnvLive)
		return h.Handle(c)
	})

	return e, h
}

func setupAsyncConvertTestWithEnv(j JobStore, p QueuePublisher, env domain.APIKeyEnvironment) (*echo.Echo, *ConvertAsyncHandler) {
	e := echo.New()
	h := NewConvertAsyncHandler(j, p)

	e.POST("/v1/convert/async", func(c echo.Context) error {
		c.Set("org_id", uuid.New())
		c.Set("api_key_id", uuid.New())
		c.Set("environment", env)
		return h.Handle(c)
	})

	return e, h
}

func doAsyncConvert(e *echo.Echo, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/convert/async", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// --- Tests ---

func TestAsyncConvert_ValidPDFRequest(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "<html><body>Hello</body></html>", "format": "pdf"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp AsyncConvertResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, mockJ.job.ID, resp.ID)
	assert.Equal(t, "pending", resp.Status)
	assert.NotEmpty(t, resp.CreatedAt)

	// Job store should have been called.
	assert.Equal(t, domain.InputTypeHTML, mockJ.lastCreateParams.InputType)
	assert.Equal(t, domain.OutputFormatPDF, mockJ.lastCreateParams.OutputFormat)

	// Publisher should have been called with the job ID.
	assert.True(t, mockP.publishCalled)
	assert.Equal(t, mockJ.job.ID.String(), mockP.lastMsg.JobID)
	assert.Equal(t, "convert", mockP.lastMsg.Type)
}

func TestAsyncConvert_ValidPDFFromURL(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "https://example.com", "format": "pdf"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, domain.InputTypeURL, mockJ.lastCreateParams.InputType)
	assert.True(t, mockP.publishCalled)
}

func TestAsyncConvert_DefaultFormatIsPDF(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "<html>test</html>"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, domain.OutputFormatPDF, mockJ.lastCreateParams.OutputFormat)
}

func TestAsyncConvert_ValidPNGRequest(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "<html>test</html>", "format": "png"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, domain.OutputFormatPNG, mockJ.lastCreateParams.OutputFormat)
}

func TestAsyncConvert_WithWebhookDelivery(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{
		"source": "<html>test</html>",
		"format": "pdf",
		"delivery": {
			"method": "webhook",
			"config": {
				"url": "https://example.com/webhook",
				"headers": {"X-Custom": "value"},
				"secret": "hmac_secret"
			}
		}
	}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp AsyncConvertResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "pending", resp.Status)

	// Delivery method should be webhook.
	assert.Equal(t, domain.DeliveryWebhook, mockJ.lastCreateParams.DeliveryMethod)

	// Delivery config should be persisted as JSON.
	assert.NotNil(t, mockJ.lastCreateParams.DeliveryConfig)
	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(mockJ.lastCreateParams.DeliveryConfig, &config))
	assert.Equal(t, "https://example.com/webhook", config["url"])
	assert.Equal(t, "hmac_secret", config["secret"])

	assert.True(t, mockP.publishCalled)
}

func TestAsyncConvert_WithS3Delivery(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{
		"source": "https://example.com",
		"format": "png",
		"delivery": {
			"method": "s3",
			"config": {
				"bucket": "my-bucket",
				"region": "us-east-1",
				"access_key": "AKIATEST",
				"secret_key": "secret123",
				"path": "output/"
			}
		}
	}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	// Delivery method should be s3.
	assert.Equal(t, domain.DeliveryS3, mockJ.lastCreateParams.DeliveryMethod)

	// Delivery config should contain S3 settings.
	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(mockJ.lastCreateParams.DeliveryConfig, &config))
	assert.Equal(t, "my-bucket", config["bucket"])
	assert.Equal(t, "us-east-1", config["region"])
	assert.Equal(t, "output/", config["path"])

	assert.True(t, mockP.publishCalled)
}

func TestAsyncConvert_MissingSource(t *testing.T) {
	mockJ := newMockJobStore()
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"format": "pdf"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_error", errResp.Error)
	assert.Contains(t, errResp.Message, "source is required")

	// Nothing should have been published.
	assert.False(t, mockP.publishCalled)
}

func TestAsyncConvert_EmptySource(t *testing.T) {
	mockJ := newMockJobStore()
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "", "format": "pdf"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, mockP.publishCalled)
}

func TestAsyncConvert_InvalidFormat(t *testing.T) {
	mockJ := newMockJobStore()
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "<html>test</html>", "format": "bmp"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_error", errResp.Error)
	assert.Contains(t, errResp.Message, "Invalid format")

	assert.False(t, mockP.publishCalled)
}

func TestAsyncConvert_InvalidDeliveryMethod(t *testing.T) {
	mockJ := newMockJobStore()
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{
		"source": "<html>test</html>",
		"format": "pdf",
		"delivery": {
			"method": "email",
			"config": {}
		}
	}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_error", errResp.Error)
	assert.Contains(t, errResp.Message, "Invalid delivery method")

	assert.False(t, mockP.publishCalled)
}

func TestAsyncConvert_DeliveryWithoutMethod(t *testing.T) {
	mockJ := newMockJobStore()
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{
		"source": "<html>test</html>",
		"format": "pdf",
		"delivery": {
			"config": {"url": "https://example.com"}
		}
	}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_error", errResp.Error)
	assert.Contains(t, errResp.Message, "delivery.method is required")
}

func TestAsyncConvert_NATSPublishError(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{err: errors.New("nats connection lost")}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "internal_error", errResp.Error)
	assert.Contains(t, errResp.Message, "Failed to enqueue job")

	// Job should have been cleaned up (marked as failed).
	assert.Equal(t, mockJ.job.ID, mockJ.failedID)
	assert.Equal(t, "failed to enqueue job", mockJ.failedMsg)
}

func TestAsyncConvert_DBCreateError(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.createErr = errors.New("db connection failed")
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "internal_error", errResp.Error)
	assert.Contains(t, errResp.Message, "Failed to create job")

	// Publisher should NOT have been called.
	assert.False(t, mockP.publishCalled)
}

func TestAsyncConvert_InvalidJSON(t *testing.T) {
	mockJ := newMockJobStore()
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{not valid json}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, mockP.publishCalled)
}

func TestAsyncConvert_TestEnvironmentSetsIsTest(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTestWithEnv(mockJ, mockP, domain.APIKeyEnvTest)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.True(t, mockJ.lastCreateParams.IsTest)
}

func TestAsyncConvert_LiveEnvironmentNotTest(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTestWithEnv(mockJ, mockP, domain.APIKeyEnvLive)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.False(t, mockJ.lastCreateParams.IsTest)
}

func TestAsyncConvert_ResponseContainsCreatedAt(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockJ.job.CreatedAt = time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp AsyncConvertResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "2024-01-15T10:30:00Z", resp.CreatedAt)
}

func TestAsyncConvert_NoDeliveryDefaultsToSync(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	// When no delivery is specified, delivery_method defaults to sync.
	assert.Equal(t, domain.DeliverySync, mockJ.lastCreateParams.DeliveryMethod)
	assert.Nil(t, mockJ.lastCreateParams.DeliveryConfig)
}

func TestAsyncConvert_OptionsSerializedToJSON(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{
		"source": "<html>test</html>",
		"format": "pdf",
		"options": {
			"page_size": "A4",
			"landscape": true
		}
	}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.NotNil(t, mockJ.lastCreateParams.Options)

	var opts map[string]interface{}
	require.NoError(t, json.Unmarshal(mockJ.lastCreateParams.Options, &opts))
	assert.Equal(t, "A4", opts["page_size"])
	assert.Equal(t, true, opts["landscape"])
}

func TestAsyncConvert_HTTPSourceDetectedAsURL(t *testing.T) {
	mockJ := newMockJobStore()
	mockJ.job.Status = domain.JobStatusPending
	mockP := &mockPublisher{}
	e, _ := setupAsyncConvertTest(mockJ, mockP)

	body := `{"source": "http://example.com"}`
	rec := doAsyncConvert(e, body)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, domain.InputTypeURL, mockJ.lastCreateParams.InputType)
}
