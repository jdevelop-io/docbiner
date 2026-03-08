package handlers

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- Interfaces for testability ---

// JobReader reads jobs from the store.
type JobReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, params ListParams) ([]*domain.Job, int, error)
}

// JobDeleter deletes jobs from the store.
type JobDeleter interface {
	Delete(ctx context.Context, id uuid.UUID) error
}

// FileStore abstracts file storage operations.
type FileStore interface {
	SignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

// ListParams holds pagination and filter parameters for listing jobs.
type ListParams struct {
	Page    int
	PerPage int
	Status  string
	Format  string
}

// --- Response structs ---

// PaginationMeta holds pagination metadata.
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// JobListResponse wraps paginated job results.
type JobListResponse struct {
	Data       []*domain.Job  `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// --- Handler ---

// JobsHandler handles job CRUD operations.
type JobsHandler struct {
	reader    JobReader
	deleter   JobDeleter
	fileStore FileStore
}

// NewJobsHandler creates a new JobsHandler.
func NewJobsHandler(r JobReader, d JobDeleter, fs FileStore) *JobsHandler {
	return &JobsHandler{
		reader:    r,
		deleter:   d,
		fileStore: fs,
	}
}

// List handles GET /v1/jobs — list jobs for the org with pagination.
func (h *JobsHandler) List(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	// Parse pagination params.
	page := parseIntParam(c.QueryParam("page"), 1)
	if page < 1 {
		page = 1
	}

	perPage := parseIntParam(c.QueryParam("per_page"), 20)
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	params := ListParams{
		Page:    page,
		PerPage: perPage,
		Status:  c.QueryParam("status"),
		Format:  c.QueryParam("format"),
	}

	jobs, total, err := h.reader.ListByOrg(c.Request().Context(), orgID, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list jobs",
		})
	}

	// Ensure we return an empty array, not null.
	if jobs == nil {
		jobs = []*domain.Job{}
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	return c.JSON(http.StatusOK, JobListResponse{
		Data: jobs,
		Pagination: PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetByID handles GET /v1/jobs/:id — get job details.
func (h *JobsHandler) GetByID(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid job ID",
		})
	}

	job, err := h.reader.GetByID(c.Request().Context(), jobID)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Job not found",
		})
	}

	// Ensure the job belongs to the requesting org.
	if job.OrgID != orgID {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Job not found",
		})
	}

	return c.JSON(http.StatusOK, job)
}

// Download handles GET /v1/jobs/:id/download — download result file.
func (h *JobsHandler) Download(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid job ID",
		})
	}

	job, err := h.reader.GetByID(c.Request().Context(), jobID)
	if err != nil || job.OrgID != orgID {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Job not found",
		})
	}

	// Check job status.
	if job.Status != domain.JobStatusCompleted {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Job not completed",
		})
	}

	// Check result URL exists.
	if job.ResultURL == nil || *job.ResultURL == "" {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "No result available",
		})
	}

	// Generate signed URL.
	signedURL, err := h.fileStore.SignedURL(c.Request().Context(), *job.ResultURL, 15*time.Minute)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate download URL",
		})
	}

	return c.Redirect(http.StatusTemporaryRedirect, signedURL)
}

// Delete handles DELETE /v1/jobs/:id — delete a job.
func (h *JobsHandler) Delete(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid job ID",
		})
	}

	// Verify the job exists and belongs to this org.
	job, err := h.reader.GetByID(c.Request().Context(), jobID)
	if err != nil || job.OrgID != orgID {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Job not found",
		})
	}

	// Delete file from storage if it exists.
	if job.ResultURL != nil && *job.ResultURL != "" {
		_ = h.fileStore.Delete(c.Request().Context(), *job.ResultURL)
	}

	// Delete the job from the database.
	if err := h.deleter.Delete(c.Request().Context(), jobID); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete job",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// parseIntParam parses a string as int, returning defaultVal on failure.
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
