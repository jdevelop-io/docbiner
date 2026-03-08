package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/docbiner/docbiner/internal/pdfutil"
	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- Interfaces for testability ---

// RendererService abstracts the renderer so handlers can be tested without Chromium.
type RendererService interface {
	HTMLToPDF(html string, opts renderer.PDFOptions) ([]byte, error)
	URLToPDF(url string, opts renderer.PDFOptions) ([]byte, error)
	HTMLToScreenshot(html string, opts renderer.ScreenshotOptions) ([]byte, error)
	URLToScreenshot(url string, opts renderer.ScreenshotOptions) ([]byte, error)
}

// JobStore abstracts job persistence for the convert handler.
type JobStore interface {
	Create(ctx context.Context, params JobCreateParams) (*domain.Job, error)
	Complete(ctx context.Context, id uuid.UUID, resultSize int64, durationMs int64) error
	Fail(ctx context.Context, id uuid.UUID, errMsg string, durationMs int64) error
}

// JobCreateParams holds parameters for creating a new job.
type JobCreateParams struct {
	OrgID          uuid.UUID
	APIKeyID       uuid.UUID
	InputType      domain.InputType
	InputSource    string
	OutputFormat   domain.OutputFormat
	Options        []byte
	DeliveryMethod domain.DeliveryMethod
	DeliveryConfig []byte
	IsTest         bool
}

// --- Request/Response structs ---

// ConvertRequest is the JSON body for POST /v1/convert.
type ConvertRequest struct {
	Source  string          `json:"source" validate:"required"`
	Format string          `json:"format"`
	Options *ConvertOptions `json:"options"`
}

// ConvertOptions configures the conversion.
type ConvertOptions struct {
	PageSize       string          `json:"page_size"`
	Landscape      bool            `json:"landscape"`
	MarginTop      string          `json:"margin_top"`
	MarginRight    string          `json:"margin_right"`
	MarginBottom   string          `json:"margin_bottom"`
	MarginLeft     string          `json:"margin_left"`
	HeaderHTML     string          `json:"header_html"`
	FooterHTML     string          `json:"footer_html"`
	CSS            string          `json:"css"`
	JS             string          `json:"js"`
	WaitFor        string          `json:"wait_for"`
	DelayMs        int             `json:"delay_ms"`
	Scale          float64         `json:"scale"`
	PrintBackground bool           `json:"print_background"`
	Width          int             `json:"width"`
	Height         int             `json:"height"`
	Quality        int             `json:"quality"`
	FullPage       bool            `json:"full_page"`
	Encrypt        *EncryptOptions `json:"encrypt"`
}

// EncryptOptions configures PDF encryption.
type EncryptOptions struct {
	UserPassword  string `json:"user_password"`
	OwnerPassword string `json:"owner_password"`
}

// ErrorResponse is a standard error payload.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// --- Valid formats ---

var validFormats = map[string]domain.OutputFormat{
	"pdf":  domain.OutputFormatPDF,
	"png":  domain.OutputFormatPNG,
	"jpeg": domain.OutputFormatJPEG,
	"webp": domain.OutputFormatWebP,
}

var formatContentType = map[string]string{
	"pdf":  "application/pdf",
	"png":  "image/png",
	"jpeg": "image/jpeg",
	"webp": "image/webp",
}

// --- Handler ---

// ConvertHandler handles synchronous conversion requests.
type ConvertHandler struct {
	renderer RendererService
	jobs     JobStore
}

// NewConvertHandler creates a new ConvertHandler.
func NewConvertHandler(r RendererService, j JobStore) *ConvertHandler {
	return &ConvertHandler{
		renderer: r,
		jobs:     j,
	}
}

// Handle processes POST /v1/convert.
func (h *ConvertHandler) Handle(c echo.Context) error {
	start := time.Now()

	// Parse request body.
	var req ConvertRequest
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

	// When called from the dashboard (JWT auth), there is no api_key_id.
	// Treat as a playground/test request without persisting a job.
	hasAPIKey := apiKeyID != uuid.Nil

	// If no API key, force test mode (playground).
	if !hasAPIKey {
		isTest = true
	}

	// Serialize options for DB storage.
	var optsJSON []byte
	if req.Options != nil {
		optsJSON, _ = json.Marshal(req.Options)
	}

	// Create job in DB only when called via API key auth.
	var jobID uuid.UUID
	if hasAPIKey {
		job, err := h.jobs.Create(c.Request().Context(), JobCreateParams{
			OrgID:          orgID,
			APIKeyID:       apiKeyID,
			InputType:      inputType,
			InputSource:    req.Source,
			OutputFormat:   outputFormat,
			Options:        optsJSON,
			DeliveryMethod: domain.DeliverySync,
			IsTest:         isTest,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to create job",
			})
		}
		jobID = job.ID
	}

	// Perform conversion.
	result, convErr := h.convert(req, inputType, isTest)

	durationMs := time.Since(start).Milliseconds()

	if convErr != nil {
		if hasAPIKey {
			_ = h.jobs.Fail(c.Request().Context(), jobID, convErr.Error(), durationMs)
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "conversion_error",
			Message: convErr.Error(),
		})
	}

	// Apply post-processing: PDF encryption.
	if req.Format == "pdf" && req.Options != nil && req.Options.Encrypt != nil {
		enc := req.Options.Encrypt
		if enc.UserPassword != "" || enc.OwnerPassword != "" {
			encrypted, encErr := pdfutil.Encrypt(result, pdfutil.EncryptOptions{
				UserPassword:  enc.UserPassword,
				OwnerPassword: enc.OwnerPassword,
			})
			if encErr != nil {
				if hasAPIKey {
					_ = h.jobs.Fail(c.Request().Context(), jobID, encErr.Error(), durationMs)
				}
				return c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error:   "encryption_error",
					Message: "Failed to encrypt PDF",
				})
			}
			result = encrypted
		}
	}

	// Update job to completed (only if tracked).
	if hasAPIKey {
		_ = h.jobs.Complete(c.Request().Context(), jobID, int64(len(result)), durationMs)
	}

	// Return file bytes with correct Content-Type.
	contentType := formatContentType[req.Format]
	return c.Blob(http.StatusOK, contentType, result)
}

// convert dispatches to the appropriate renderer method.
func (h *ConvertHandler) convert(req ConvertRequest, inputType domain.InputType, isTest bool) ([]byte, error) {
	if req.Format == "pdf" {
		return h.convertPDF(req, inputType, isTest)
	}
	return h.convertScreenshot(req, inputType)
}

// convertPDF handles PDF conversion.
func (h *ConvertHandler) convertPDF(req ConvertRequest, inputType domain.InputType, isTest bool) ([]byte, error) {
	opts := buildPDFOptions(req.Options)

	// Add watermark for test environment.
	if isTest {
		opts.WatermarkText = "TEST"
		opts.WatermarkOpacity = 0.15
	}

	if inputType == domain.InputTypeURL {
		return h.renderer.URLToPDF(req.Source, opts)
	}
	return h.renderer.HTMLToPDF(req.Source, opts)
}

// convertScreenshot handles image conversion.
func (h *ConvertHandler) convertScreenshot(req ConvertRequest, inputType domain.InputType) ([]byte, error) {
	opts := buildScreenshotOptions(req.Format, req.Options)

	if inputType == domain.InputTypeURL {
		return h.renderer.URLToScreenshot(req.Source, opts)
	}
	return h.renderer.HTMLToScreenshot(req.Source, opts)
}

// buildPDFOptions converts ConvertOptions to renderer.PDFOptions.
func buildPDFOptions(opts *ConvertOptions) renderer.PDFOptions {
	if opts == nil {
		return renderer.PDFOptions{
			PrintBG: true,
		}
	}
	return renderer.PDFOptions{
		PageSize:     opts.PageSize,
		Landscape:    opts.Landscape,
		MarginTop:    opts.MarginTop,
		MarginBottom: opts.MarginBottom,
		MarginLeft:   opts.MarginLeft,
		MarginRight:  opts.MarginRight,
		HeaderHTML:   opts.HeaderHTML,
		FooterHTML:   opts.FooterHTML,
		CSS:          opts.CSS,
		JS:           opts.JS,
		WaitFor:      opts.WaitFor,
		DelayMs:      opts.DelayMs,
		Scale:        opts.Scale,
		PrintBG:      opts.PrintBackground,
	}
}

// buildScreenshotOptions converts ConvertOptions to renderer.ScreenshotOptions.
func buildScreenshotOptions(format string, opts *ConvertOptions) renderer.ScreenshotOptions {
	if opts == nil {
		return renderer.ScreenshotOptions{
			Format: format,
		}
	}
	return renderer.ScreenshotOptions{
		Format:   format,
		Quality:  opts.Quality,
		FullPage: opts.FullPage,
		Width:    opts.Width,
		Height:   opts.Height,
		CSS:      opts.CSS,
		JS:       opts.JS,
		WaitFor:  opts.WaitFor,
		DelayMs:  opts.DelayMs,
	}
}
