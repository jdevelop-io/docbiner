package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/docbiner/docbiner/internal/pdfutil"
	"github.com/docbiner/docbiner/internal/queue"
	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/google/uuid"
)

// JobStore abstracts persistence operations on jobs.
type JobStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus) error
	Complete(ctx context.Context, id uuid.UUID, resultURL string, resultSize int64, pagesCount int, durationMs int64) error
	Fail(ctx context.Context, id uuid.UUID, errMsg string, durationMs int64) error
}

// Renderer abstracts PDF/screenshot rendering operations.
type Renderer interface {
	HTMLToPDF(ctx context.Context, html string, opts renderer.PDFOptions) ([]byte, error)
	URLToPDF(ctx context.Context, url string, opts renderer.PDFOptions) ([]byte, error)
	HTMLToScreenshot(ctx context.Context, html string, opts renderer.ScreenshotOptions) ([]byte, error)
	URLToScreenshot(ctx context.Context, url string, opts renderer.ScreenshotOptions) ([]byte, error)
}

// StorageUploader abstracts object storage uploads.
type StorageUploader interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)
}

// DeliveryDispatcher abstracts delivery of completed job results.
type DeliveryDispatcher interface {
	Dispatch(ctx context.Context, job *domain.Job, resultData []byte) error
}

// JobHandler processes conversion jobs received from the queue.
type JobHandler struct {
	jobs     JobStore
	renderer Renderer
	storage  StorageUploader
	delivery DeliveryDispatcher
}

// NewJobHandler creates a new JobHandler with the provided dependencies.
func NewJobHandler(jobs JobStore, r Renderer, storage StorageUploader, delivery DeliveryDispatcher) *JobHandler {
	return &JobHandler{
		jobs:     jobs,
		renderer: r,
		storage:  storage,
		delivery: delivery,
	}
}

// jobOptions represents the JSONB options field from the job, including
// both rendering and post-processing options.
type jobOptions struct {
	// PDF options
	PageSize         string  `json:"page_size"`
	Landscape        bool    `json:"landscape"`
	MarginTop        string  `json:"margin_top"`
	MarginBottom     string  `json:"margin_bottom"`
	MarginLeft       string  `json:"margin_left"`
	MarginRight      string  `json:"margin_right"`
	HeaderHTML       string  `json:"header_html"`
	FooterHTML       string  `json:"footer_html"`
	CSS              string  `json:"css"`
	JS               string  `json:"js"`
	WaitFor          string  `json:"wait_for"`
	DelayMs          int     `json:"delay_ms"`
	Scale            float64 `json:"scale"`
	PrintBG          bool    `json:"print_background"`
	WatermarkText    string  `json:"watermark_text"`
	WatermarkOpacity float64 `json:"watermark_opacity"`

	// Screenshot options
	Format   string `json:"format"`
	Quality  int    `json:"quality"`
	FullPage bool   `json:"full_page"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`

	// Encryption options
	Encrypt *encryptOptions `json:"encrypt,omitempty"`
}

// encryptOptions represents encryption settings for PDF output.
type encryptOptions struct {
	UserPassword  string   `json:"user_password"`
	OwnerPassword string   `json:"owner_password"`
	Restrict      []string `json:"restrict"`
}

// Handle processes a single job message from the queue.
func (h *JobHandler) Handle(msg queue.JobMessage) error {
	ctx := context.Background()
	start := time.Now()

	jobID, err := uuid.Parse(msg.JobID)
	if err != nil {
		log.Printf("invalid job ID %q: %v", msg.JobID, err)
		return fmt.Errorf("invalid job ID: %w", err)
	}

	// 1. Load job from DB.
	job, err := h.jobs.GetByID(ctx, jobID)
	if err != nil {
		log.Printf("job %s not found: %v", jobID, err)
		return fmt.Errorf("get job: %w", err)
	}

	// 2. Update status to processing.
	if err := h.jobs.UpdateStatus(ctx, jobID, domain.JobStatusProcessing); err != nil {
		log.Printf("job %s: failed to update status: %v", jobID, err)
		return fmt.Errorf("update status: %w", err)
	}

	// 3. Parse options.
	var opts jobOptions
	if len(job.Options) > 0 {
		if err := json.Unmarshal(job.Options, &opts); err != nil {
			durationMs := time.Since(start).Milliseconds()
			_ = h.jobs.Fail(ctx, jobID, fmt.Sprintf("invalid options: %v", err), durationMs)
			return nil // message is consumed, job is marked as failed
		}
	}

	// 4-5. Determine conversion type and apply watermark if test.
	pdfOpts := renderer.PDFOptions{
		PageSize:         opts.PageSize,
		Landscape:        opts.Landscape,
		MarginTop:        opts.MarginTop,
		MarginBottom:     opts.MarginBottom,
		MarginLeft:       opts.MarginLeft,
		MarginRight:      opts.MarginRight,
		HeaderHTML:       opts.HeaderHTML,
		FooterHTML:       opts.FooterHTML,
		CSS:              opts.CSS,
		JS:               opts.JS,
		WaitFor:          opts.WaitFor,
		DelayMs:          opts.DelayMs,
		Scale:            opts.Scale,
		PrintBG:          opts.PrintBG,
		WatermarkText:    opts.WatermarkText,
		WatermarkOpacity: opts.WatermarkOpacity,
	}

	ssOpts := renderer.ScreenshotOptions{
		Format:   opts.Format,
		Quality:  opts.Quality,
		FullPage: opts.FullPage,
		Width:    opts.Width,
		Height:   opts.Height,
		CSS:      opts.CSS,
		JS:       opts.JS,
		WaitFor:  opts.WaitFor,
		DelayMs:  opts.DelayMs,
	}

	// Apply watermark for test jobs.
	if job.IsTest {
		pdfOpts.WatermarkText = "TEST"
		if pdfOpts.WatermarkOpacity <= 0 {
			pdfOpts.WatermarkOpacity = 0.15
		}
	}

	// 6. Call renderer.
	var result []byte
	isPDF := job.OutputFormat == domain.OutputFormatPDF

	switch {
	case isPDF && job.InputType == domain.InputTypeHTML:
		result, err = h.renderer.HTMLToPDF(ctx, job.InputSource, pdfOpts)
	case isPDF && job.InputType == domain.InputTypeURL:
		result, err = h.renderer.URLToPDF(ctx, job.InputSource, pdfOpts)
	case !isPDF && job.InputType == domain.InputTypeHTML:
		ssOpts.Format = string(job.OutputFormat)
		result, err = h.renderer.HTMLToScreenshot(ctx, job.InputSource, ssOpts)
	case !isPDF && job.InputType == domain.InputTypeURL:
		ssOpts.Format = string(job.OutputFormat)
		result, err = h.renderer.URLToScreenshot(ctx, job.InputSource, ssOpts)
	default:
		err = fmt.Errorf("unsupported combination: input_type=%s output_format=%s", job.InputType, job.OutputFormat)
	}

	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		_ = h.jobs.Fail(ctx, jobID, fmt.Sprintf("render failed: %v", err), durationMs)
		log.Printf("job %s: render failed: %v", jobID, err)
		return nil
	}

	// 7. Post-processing: encrypt PDF if requested.
	if isPDF && opts.Encrypt != nil {
		result, err = pdfutil.Encrypt(result, pdfutil.EncryptOptions{
			UserPassword:  opts.Encrypt.UserPassword,
			OwnerPassword: opts.Encrypt.OwnerPassword,
			Restrict:      opts.Encrypt.Restrict,
		})
		if err != nil {
			durationMs := time.Since(start).Milliseconds()
			_ = h.jobs.Fail(ctx, jobID, fmt.Sprintf("encryption failed: %v", err), durationMs)
			log.Printf("job %s: encryption failed: %v", jobID, err)
			return nil
		}
	}

	// 8. Upload result to storage.
	contentType := contentTypeForFormat(job.OutputFormat)
	storageKey := fmt.Sprintf("jobs/%s/result.%s", jobID, job.OutputFormat)

	resultURL, err := h.storage.Upload(ctx, storageKey, result, contentType)
	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		_ = h.jobs.Fail(ctx, jobID, fmt.Sprintf("upload failed: %v", err), durationMs)
		log.Printf("job %s: upload failed: %v", jobID, err)
		return nil
	}

	// 9. Update job as completed.
	durationMs := time.Since(start).Milliseconds()
	resultSize := int64(len(result))
	pagesCount := 0 // page count estimation is not yet implemented

	if err := h.jobs.Complete(ctx, jobID, resultURL, resultSize, pagesCount, durationMs); err != nil {
		log.Printf("job %s: failed to mark complete: %v", jobID, err)
		return fmt.Errorf("complete job: %w", err)
	}

	// 10. Handle delivery.
	if job.DeliveryMethod == domain.DeliveryWebhook || job.DeliveryMethod == domain.DeliveryS3 {
		if h.delivery != nil {
			if err := h.delivery.Dispatch(ctx, job, result); err != nil {
				log.Printf("job %s: delivery failed: %v", jobID, err)
				// Delivery failure does not fail the job — it was already completed.
			}
		}
	}

	log.Printf("job %s: completed in %dms (size=%d)", jobID, durationMs, resultSize)
	return nil
}

// contentTypeForFormat returns the MIME type for the given output format.
func contentTypeForFormat(f domain.OutputFormat) string {
	switch f {
	case domain.OutputFormatPDF:
		return "application/pdf"
	case domain.OutputFormatPNG:
		return "image/png"
	case domain.OutputFormatJPEG:
		return "image/jpeg"
	case domain.OutputFormatWebP:
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// rendererAdapter wraps a concrete *renderer.Renderer to satisfy the Renderer interface.
type rendererAdapter struct {
	r *renderer.Renderer
}

func (a *rendererAdapter) HTMLToPDF(_ context.Context, html string, opts renderer.PDFOptions) ([]byte, error) {
	return a.r.HTMLToPDF(html, opts)
}

func (a *rendererAdapter) URLToPDF(_ context.Context, url string, opts renderer.PDFOptions) ([]byte, error) {
	return a.r.URLToPDF(url, opts)
}

func (a *rendererAdapter) HTMLToScreenshot(_ context.Context, html string, opts renderer.ScreenshotOptions) ([]byte, error) {
	return a.r.HTMLToScreenshot(html, opts)
}

func (a *rendererAdapter) URLToScreenshot(_ context.Context, url string, opts renderer.ScreenshotOptions) ([]byte, error) {
	return a.r.URLToScreenshot(url, opts)
}
