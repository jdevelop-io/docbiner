package handlers

import (
	"context"
	"net/http"

	"github.com/docbiner/docbiner/internal/usage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// UsageReader abstracts usage reading for testability.
type UsageReader interface {
	GetCurrent(ctx context.Context, orgID uuid.UUID) (*usage.MonthlyUsage, error)
	GetHistory(ctx context.Context, orgID uuid.UUID, months int) ([]*usage.MonthlyUsage, error)
	GetQuotaStatus(ctx context.Context, orgID uuid.UUID) (*usage.QuotaStatus, error)
}

// UsageResponse is the JSON response for GET /v1/usage.
type UsageResponse struct {
	Month           string             `json:"month"`
	Conversions     int                `json:"conversions"`
	TestConversions int                `json:"test_conversions"`
	Quota           *usage.QuotaStatus `json:"quota"`
}

// UsageHandler handles usage-related endpoints.
type UsageHandler struct {
	reader UsageReader
}

// NewUsageHandler creates a new UsageHandler.
func NewUsageHandler(r UsageReader) *UsageHandler {
	return &UsageHandler{reader: r}
}

// HandleGetUsage handles GET /v1/usage and returns current month usage + quota status.
func (h *UsageHandler) HandleGetUsage(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	ctx := c.Request().Context()

	current, err := h.reader.GetCurrent(ctx, orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get usage data",
		})
	}

	quota, err := h.reader.GetQuotaStatus(ctx, orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get quota status",
		})
	}

	return c.JSON(http.StatusOK, UsageResponse{
		Month:           current.Month.Format("2006-01"),
		Conversions:     current.Conversions,
		TestConversions: current.TestConversions,
		Quota:           quota,
	})
}

// HandleGetUsageHistory handles GET /v1/usage/history and returns the last 12 months of usage.
func (h *UsageHandler) HandleGetUsageHistory(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	ctx := c.Request().Context()

	history, err := h.reader.GetHistory(ctx, orgID, 12)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get usage history",
		})
	}

	// Return empty array instead of null.
	if history == nil {
		history = []*usage.MonthlyUsage{}
	}

	return c.JSON(http.StatusOK, history)
}
