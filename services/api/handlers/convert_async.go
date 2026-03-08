package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/docbiner/docbiner/internal/queue"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// QueuePublisher abstracts the NATS publish operation for testability.
type QueuePublisher interface {
	Publish(ctx context.Context, msg queue.JobMessage) error
}

// --- Request/Response structs ---

// AsyncConvertRequest is the JSON body for POST /v1/convert/async.
type AsyncConvertRequest struct {
	Source   string          `json:"source"`
	Format   string          `json:"format"`
	Options  *ConvertOptions `json:"options"`
	Delivery *DeliveryConfig `json:"delivery"`
}

// DeliveryConfig describes how the result should be delivered.
type DeliveryConfig struct {
	Method string                 `json:"method"`
	Config map[string]interface{} `json:"config"`
}

// AsyncConvertResponse is returned on successful async job creation.
type AsyncConvertResponse struct {
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"created_at"`
}

// --- Valid delivery methods ---

var validDeliveryMethods = map[string]domain.DeliveryMethod{
	"webhook": domain.DeliveryWebhook,
	"s3":      domain.DeliveryS3,
}

// --- Handler ---

// ConvertAsyncHandler handles asynchronous conversion requests.
type ConvertAsyncHandler struct {
	jobs      JobStore
	publisher QueuePublisher
}

// NewConvertAsyncHandler creates a new ConvertAsyncHandler.
func NewConvertAsyncHandler(j JobStore, p QueuePublisher) *ConvertAsyncHandler {
	return &ConvertAsyncHandler{
		jobs:      j,
		publisher: p,
	}
}

// Handle processes POST /v1/convert/async.
func (h *ConvertAsyncHandler) Handle(c echo.Context) error {
	// Parse request body.
	var req AsyncConvertRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	// Validate: source is required.
	if strings.TrimSpace(req.Source) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "source is required",
		})
	}

	// Default format to "pdf".
	if req.Format == "" {
		req.Format = "pdf"
	}

	// Validate format.
	outputFormat, ok := validFormats[req.Format]
	if !ok {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid format: must be one of pdf, png, jpeg, webp",
		})
	}

	// Determine delivery method. Default to webhook if delivery is provided
	// without an explicit method, or reject if method is invalid.
	deliveryMethod := domain.DeliverySync
	var deliveryConfigJSON []byte
	if req.Delivery != nil {
		if req.Delivery.Method == "" {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "delivery.method is required when delivery is specified",
			})
		}
		dm, valid := validDeliveryMethods[req.Delivery.Method]
		if !valid {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "Invalid delivery method: must be one of webhook, s3",
			})
		}
		deliveryMethod = dm

		if req.Delivery.Config != nil {
			var err error
			deliveryConfigJSON, err = json.Marshal(req.Delivery.Config)
			if err != nil {
				return c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "bad_request",
					Message: "Invalid delivery config",
				})
			}
		}
	}

	// Determine input type.
	inputType := domain.InputTypeHTML
	if strings.HasPrefix(req.Source, "http://") || strings.HasPrefix(req.Source, "https://") {
		inputType = domain.InputTypeURL
	}

	// Extract auth context values.
	orgID, _ := c.Get("org_id").(uuid.UUID)
	apiKeyID, _ := c.Get("api_key_id").(uuid.UUID)
	env, _ := c.Get("environment").(domain.APIKeyEnvironment)
	isTest := env == domain.APIKeyEnvTest

	// Serialize options for DB storage.
	var optsJSON []byte
	if req.Options != nil {
		optsJSON, _ = json.Marshal(req.Options)
	}

	// Create job in DB with status=pending (async job).
	job, err := h.jobs.Create(c.Request().Context(), JobCreateParams{
		OrgID:          orgID,
		APIKeyID:       apiKeyID,
		InputType:      inputType,
		InputSource:    req.Source,
		OutputFormat:   outputFormat,
		Options:        optsJSON,
		DeliveryMethod: deliveryMethod,
		DeliveryConfig: deliveryConfigJSON,
		IsTest:         isTest,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create job",
		})
	}

	// Publish job to NATS queue.
	pubErr := h.publisher.Publish(c.Request().Context(), queue.JobMessage{
		JobID: job.ID.String(),
		Type:  "convert",
	})
	if pubErr != nil {
		// Clean up: mark the job as failed since we couldn't enqueue it.
		_ = h.jobs.Fail(c.Request().Context(), job.ID, "failed to enqueue job", 0)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to enqueue job",
		})
	}

	// Return 202 Accepted with job info.
	return c.JSON(http.StatusAccepted, AsyncConvertResponse{
		ID:        job.ID,
		Status:    string(job.Status),
		CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}
