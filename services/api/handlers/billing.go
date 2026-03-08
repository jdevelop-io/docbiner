package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- Interfaces ---

// BillingProvider abstracts billing operations for testability.
type BillingProvider interface {
	CreateCustomer(ctx context.Context, orgName, email string) (string, error)
	CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (string, error)
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error)
}

// BillingOrgStore abstracts organization lookups and updates for billing.
type BillingOrgStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	UpdateStripeCustomerID(ctx context.Context, orgID uuid.UUID, customerID string) error
}

// BillingPlanStore abstracts plan lookups for billing status.
type BillingPlanStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error)
}

// --- Request/Response structs ---

// CheckoutRequest is the JSON body for POST /v1/billing/checkout.
type CheckoutRequest struct {
	PriceID    string `json:"price_id"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

// CheckoutResponse is the JSON body returned on successful checkout session creation.
type CheckoutResponse struct {
	URL string `json:"url"`
}

// PortalRequest is the JSON body for POST /v1/billing/portal.
type PortalRequest struct {
	ReturnURL string `json:"return_url"`
}

// PortalResponse is the JSON body returned on successful portal session creation.
type PortalResponse struct {
	URL string `json:"url"`
}

// BillingStatusResponse is the JSON body for GET /v1/billing/status.
type BillingStatusResponse struct {
	Plan             string `json:"plan"`
	StripeCustomerID string `json:"stripe_customer_id,omitempty"`
	PriceMonthly     float64 `json:"price_monthly"`
	PriceYearly      float64 `json:"price_yearly"`
	MonthlyQuota     int     `json:"monthly_quota"`
}

// --- Handler ---

// BillingHandler handles billing-related endpoints.
type BillingHandler struct {
	billing BillingProvider
	orgs    BillingOrgStore
	plans   BillingPlanStore
}

// NewBillingHandler creates a new BillingHandler.
func NewBillingHandler(billing BillingProvider, orgs BillingOrgStore, plans BillingPlanStore) *BillingHandler {
	return &BillingHandler{
		billing: billing,
		orgs:    orgs,
		plans:   plans,
	}
}

// HandleCheckout handles POST /v1/billing/checkout.
func (h *BillingHandler) HandleCheckout(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	var req CheckoutRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if strings.TrimSpace(req.PriceID) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "price_id is required",
		})
	}
	if strings.TrimSpace(req.SuccessURL) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "success_url is required",
		})
	}
	if strings.TrimSpace(req.CancelURL) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "cancel_url is required",
		})
	}

	ctx := c.Request().Context()

	org, err := h.orgs.GetByID(ctx, orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get organization",
		})
	}

	// Ensure org has a Stripe customer ID; create one if missing.
	customerID := org.StripeCustomerID
	if customerID == "" {
		customerID, err = h.billing.CreateCustomer(ctx, org.Name, "")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to create Stripe customer",
			})
		}
		if err := h.orgs.UpdateStripeCustomerID(ctx, orgID, customerID); err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to update organization",
			})
		}
	}

	url, err := h.billing.CreateCheckoutSession(ctx, customerID, req.PriceID, req.SuccessURL, req.CancelURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create checkout session",
		})
	}

	return c.JSON(http.StatusOK, CheckoutResponse{URL: url})
}

// HandlePortal handles POST /v1/billing/portal.
func (h *BillingHandler) HandlePortal(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	var req PortalRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if strings.TrimSpace(req.ReturnURL) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "return_url is required",
		})
	}

	ctx := c.Request().Context()

	org, err := h.orgs.GetByID(ctx, orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get organization",
		})
	}

	if org.StripeCustomerID == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "No billing account found. Please subscribe first.",
		})
	}

	url, err := h.billing.CreatePortalSession(ctx, org.StripeCustomerID, req.ReturnURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create portal session",
		})
	}

	return c.JSON(http.StatusOK, PortalResponse{URL: url})
}

// HandleStatus handles GET /v1/billing/status.
func (h *BillingHandler) HandleStatus(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing organization context",
		})
	}

	ctx := c.Request().Context()

	org, err := h.orgs.GetByID(ctx, orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get organization",
		})
	}

	plan, err := h.plans.GetByID(ctx, org.PlanID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get plan details",
		})
	}

	return c.JSON(http.StatusOK, BillingStatusResponse{
		Plan:             plan.Name,
		StripeCustomerID: org.StripeCustomerID,
		PriceMonthly:     plan.PriceMonthly,
		PriceYearly:      plan.PriceYearly,
		MonthlyQuota:     plan.MonthlyQuota,
	})
}
