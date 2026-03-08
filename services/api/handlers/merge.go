package handlers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// --- Interfaces for testability ---

// PDFMerger abstracts PDF merging so the handler can be tested without real PDFs.
type PDFMerger interface {
	Merge(pdfs [][]byte) ([]byte, error)
}

// --- Request structs ---

// MergeSource represents a single source in a merge request.
type MergeSource struct {
	Source string `json:"source"`
}

// MergeRequest is the JSON body for POST /v1/merge.
type MergeRequest struct {
	Sources []MergeSource   `json:"sources"`
	Options *ConvertOptions `json:"options"`
}

// --- Handler ---

// MergeHandler handles PDF merge requests.
type MergeHandler struct {
	renderer RendererService
	merger   PDFMerger
}

// NewMergeHandler creates a new MergeHandler.
func NewMergeHandler(r RendererService, m PDFMerger) *MergeHandler {
	return &MergeHandler{
		renderer: r,
		merger:   m,
	}
}

// Handle processes POST /v1/merge.
func (h *MergeHandler) Handle(c echo.Context) error {
	var req MergeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	// Validate: at least one source is required.
	if len(req.Sources) == 0 {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "At least one source is required",
		})
	}

	// Validate each source is non-empty.
	for _, s := range req.Sources {
		if strings.TrimSpace(s.Source) == "" {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "Each source must be non-empty",
			})
		}
	}

	// Convert each source to PDF.
	opts := buildPDFOptions(req.Options)
	pdfs := make([][]byte, 0, len(req.Sources))

	for _, s := range req.Sources {
		var pdfBytes []byte
		var err error

		if strings.HasPrefix(s.Source, "http://") || strings.HasPrefix(s.Source, "https://") {
			pdfBytes, err = h.renderer.URLToPDF(s.Source, opts)
		} else {
			pdfBytes, err = h.renderer.HTMLToPDF(s.Source, opts)
		}

		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "conversion_error",
				Message: "Failed to convert source to PDF",
			})
		}

		pdfs = append(pdfs, pdfBytes)
	}

	// Merge all PDFs.
	merged, err := h.merger.Merge(pdfs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "merge_error",
			Message: "Failed to merge PDFs",
		})
	}

	return c.Blob(http.StatusOK, "application/pdf", merged)
}
