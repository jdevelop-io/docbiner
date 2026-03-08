package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v82"
)

// --- Interfaces ---

// WebhookVerifier abstracts Stripe webhook signature verification.
type WebhookVerifier interface {
	VerifyWebhookSignature(payload []byte, signature string) (*stripe.Event, error)
}

// WebhookOrgStore abstracts organization updates for webhook handling.
type WebhookOrgStore interface {
	GetByStripeCustomerID(ctx context.Context, customerID string) (*OrgInfo, error)
	UpdatePlan(ctx context.Context, orgID, planID uuid.UUID) error
}

// WebhookPlanStore abstracts plan lookups for webhook handling.
type WebhookPlanStore interface {
	GetByName(ctx context.Context, name string) (*PlanInfo, error)
}

// OrgInfo is a lightweight org representation for webhooks.
type OrgInfo struct {
	ID     uuid.UUID
	PlanID uuid.UUID
}

// PlanInfo is a lightweight plan representation for webhooks.
type PlanInfo struct {
	ID   uuid.UUID
	Name string
}

// --- Handler ---

// StripeWebhookHandler handles Stripe webhook events.
type StripeWebhookHandler struct {
	verifier WebhookVerifier
	orgs     WebhookOrgStore
	plans    WebhookPlanStore
	logger   *slog.Logger
}

// NewStripeWebhookHandler creates a new StripeWebhookHandler.
func NewStripeWebhookHandler(verifier WebhookVerifier, orgs WebhookOrgStore, plans WebhookPlanStore, logger *slog.Logger) *StripeWebhookHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &StripeWebhookHandler{
		verifier: verifier,
		orgs:     orgs,
		plans:    plans,
		logger:   logger,
	}
}

// Handle processes POST /v1/webhooks/stripe.
func (h *StripeWebhookHandler) Handle(c echo.Context) error {
	// Limit request body size to prevent abuse.
	const maxBodyBytes = 65536
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxBodyBytes))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Unable to read request body",
		})
	}

	signature := c.Request().Header.Get("Stripe-Signature")
	if signature == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Missing Stripe-Signature header",
		})
	}

	event, err := h.verifier.VerifyWebhookSignature(body, signature)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid webhook signature",
		})
	}

	ctx := c.Request().Context()

	switch event.Type {
	case "checkout.session.completed":
		return h.handleCheckoutCompleted(ctx, c, event)
	case "customer.subscription.deleted":
		return h.handleSubscriptionDeleted(ctx, c, event)
	case "invoice.payment_failed":
		return h.handlePaymentFailed(event)
	default:
		h.logger.Info("unhandled stripe event", "type", event.Type)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// checkoutSessionData holds the fields we extract from a checkout.session.completed event.
type checkoutSessionData struct {
	Customer     string `json:"customer"`
	Subscription string `json:"subscription"`
}

// subscriptionData holds the fields we extract from a customer.subscription.deleted event.
type subscriptionData struct {
	Customer string `json:"customer"`
}

// invoiceData holds the fields we extract from an invoice.payment_failed event.
type invoiceData struct {
	Customer     string `json:"customer"`
	Subscription string `json:"subscription"`
}

// handleCheckoutCompleted processes checkout.session.completed events.
func (h *StripeWebhookHandler) handleCheckoutCompleted(ctx context.Context, c echo.Context, event *stripe.Event) error {
	var session checkoutSessionData
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		h.logger.Error("failed to parse checkout session", "error", err)
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid event data",
		})
	}

	if session.Customer == "" {
		h.logger.Warn("checkout session missing customer ID")
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	org, err := h.orgs.GetByStripeCustomerID(ctx, session.Customer)
	if err != nil {
		h.logger.Error("org not found for stripe customer", "customer_id", session.Customer, "error", err)
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	// For now, upgrade to "starter" plan on checkout completion.
	// In a full implementation, we would look at the subscription items to
	// determine the correct plan.
	plan, err := h.plans.GetByName(ctx, "starter")
	if err != nil {
		h.logger.Error("plan not found", "plan", "starter", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to look up plan",
		})
	}

	if err := h.orgs.UpdatePlan(ctx, org.ID, plan.ID); err != nil {
		h.logger.Error("failed to update org plan", "org_id", org.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update plan",
		})
	}

	h.logger.Info("org plan upgraded", "org_id", org.ID, "plan", plan.Name)
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// handleSubscriptionDeleted processes customer.subscription.deleted events.
func (h *StripeWebhookHandler) handleSubscriptionDeleted(ctx context.Context, c echo.Context, event *stripe.Event) error {
	var sub subscriptionData
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		h.logger.Error("failed to parse subscription", "error", err)
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid event data",
		})
	}

	if sub.Customer == "" {
		h.logger.Warn("subscription event missing customer ID")
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	org, err := h.orgs.GetByStripeCustomerID(ctx, sub.Customer)
	if err != nil {
		h.logger.Error("org not found for stripe customer", "customer_id", sub.Customer, "error", err)
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	// Downgrade to free plan.
	plan, err := h.plans.GetByName(ctx, "free")
	if err != nil {
		h.logger.Error("plan not found", "plan", "free", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to look up plan",
		})
	}

	if err := h.orgs.UpdatePlan(ctx, org.ID, plan.ID); err != nil {
		h.logger.Error("failed to downgrade org plan", "org_id", org.ID, "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update plan",
		})
	}

	h.logger.Info("org plan downgraded to free", "org_id", org.ID)
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// handlePaymentFailed logs invoice.payment_failed events.
func (h *StripeWebhookHandler) handlePaymentFailed(event *stripe.Event) error {
	var inv invoiceData
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		h.logger.Error("failed to parse invoice", "error", err)
		return nil
	}

	h.logger.Warn("payment failed",
		"customer_id", inv.Customer,
		"subscription_id", inv.Subscription,
	)

	return nil
}
