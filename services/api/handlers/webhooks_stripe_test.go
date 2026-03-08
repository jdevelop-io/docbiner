package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v82"
)

// --- Mock WebhookVerifier ---

type mockWebhookVerifier struct {
	event *stripe.Event
	err   error
}

func (m *mockWebhookVerifier) VerifyWebhookSignature(_ []byte, _ string) (*stripe.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.event, nil
}

// --- Mock WebhookOrgStore ---

type mockWebhookOrgStore struct {
	org *OrgInfo
	err error

	lastUpdateOrgID  uuid.UUID
	lastUpdatePlanID uuid.UUID
}

func (m *mockWebhookOrgStore) GetByStripeCustomerID(_ context.Context, _ string) (*OrgInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.org, nil
}

func (m *mockWebhookOrgStore) UpdatePlan(_ context.Context, orgID, planID uuid.UUID) error {
	m.lastUpdateOrgID = orgID
	m.lastUpdatePlanID = planID
	return nil
}

// --- Mock WebhookPlanStore ---

type mockWebhookPlanStore struct {
	plan *PlanInfo
	err  error
}

func (m *mockWebhookPlanStore) GetByName(_ context.Context, _ string) (*PlanInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.plan, nil
}

// --- Test Helpers ---

func makeEvent(eventType string, data interface{}) *stripe.Event {
	raw, _ := json.Marshal(data)
	return &stripe.Event{
		Type: stripe.EventType(eventType),
		Data: &stripe.EventData{
			Raw: raw,
		},
	}
}

func setupWebhookTest(verifier WebhookVerifier, orgs WebhookOrgStore, plans WebhookPlanStore) *echo.Echo {
	e := echo.New()
	logger := slog.Default()
	h := NewStripeWebhookHandler(verifier, orgs, plans, logger)
	e.POST("/v1/webhooks/stripe", h.Handle)
	return e
}

func doWebhookRequest(e *echo.Echo, body, signature string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("Stripe-Signature", signature)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// --- Tests: Checkout Session Completed ---

func TestWebhook_CheckoutCompleted_PlanUpgraded(t *testing.T) {
	orgID := uuid.New()
	planID := uuid.New()

	verifier := &mockWebhookVerifier{
		event: makeEvent("checkout.session.completed", map[string]string{
			"customer":     "cus_test123",
			"subscription": "sub_test123",
		}),
	}
	orgs := &mockWebhookOrgStore{
		org: &OrgInfo{ID: orgID, PlanID: uuid.New()},
	}
	plans := &mockWebhookPlanStore{
		plan: &PlanInfo{ID: planID, Name: "starter"},
	}
	e := setupWebhookTest(verifier, orgs, plans)

	rec := doWebhookRequest(e, `{}`, "t=123,v1=abc")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, orgID, orgs.lastUpdateOrgID)
	assert.Equal(t, planID, orgs.lastUpdatePlanID)
}

func TestWebhook_CheckoutCompleted_OrgNotFound(t *testing.T) {
	verifier := &mockWebhookVerifier{
		event: makeEvent("checkout.session.completed", map[string]string{
			"customer": "cus_unknown",
		}),
	}
	orgs := &mockWebhookOrgStore{err: errors.New("not found")}
	plans := &mockWebhookPlanStore{
		plan: &PlanInfo{ID: uuid.New(), Name: "starter"},
	}
	e := setupWebhookTest(verifier, orgs, plans)

	rec := doWebhookRequest(e, `{}`, "t=123,v1=abc")

	// Should return 200 even if org not found (don't retry).
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- Tests: Subscription Deleted ---

func TestWebhook_SubscriptionDeleted_DowngradedToFree(t *testing.T) {
	orgID := uuid.New()
	freePlanID := uuid.New()

	verifier := &mockWebhookVerifier{
		event: makeEvent("customer.subscription.deleted", map[string]string{
			"customer": "cus_test456",
		}),
	}
	orgs := &mockWebhookOrgStore{
		org: &OrgInfo{ID: orgID, PlanID: uuid.New()},
	}
	plans := &mockWebhookPlanStore{
		plan: &PlanInfo{ID: freePlanID, Name: "free"},
	}
	e := setupWebhookTest(verifier, orgs, plans)

	rec := doWebhookRequest(e, `{}`, "t=123,v1=abc")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, orgID, orgs.lastUpdateOrgID)
	assert.Equal(t, freePlanID, orgs.lastUpdatePlanID)
}

// --- Tests: Invalid Signature ---

func TestWebhook_InvalidSignature_Returns400(t *testing.T) {
	verifier := &mockWebhookVerifier{err: errors.New("invalid signature")}
	orgs := &mockWebhookOrgStore{}
	plans := &mockWebhookPlanStore{}
	e := setupWebhookTest(verifier, orgs, plans)

	rec := doWebhookRequest(e, `{}`, "bad_sig")

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "bad_request", errResp.Error)
	assert.Contains(t, errResp.Message, "Invalid webhook signature")
}

func TestWebhook_MissingSignature_Returns400(t *testing.T) {
	verifier := &mockWebhookVerifier{}
	orgs := &mockWebhookOrgStore{}
	plans := &mockWebhookPlanStore{}
	e := setupWebhookTest(verifier, orgs, plans)

	rec := doWebhookRequest(e, `{}`, "") // No signature

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp.Message, "Missing Stripe-Signature header")
}

// --- Tests: Unknown Event Type ---

func TestWebhook_UnknownEventType_Returns200(t *testing.T) {
	verifier := &mockWebhookVerifier{
		event: makeEvent("some.unknown.event", map[string]string{}),
	}
	orgs := &mockWebhookOrgStore{}
	plans := &mockWebhookPlanStore{}
	e := setupWebhookTest(verifier, orgs, plans)

	rec := doWebhookRequest(e, `{}`, "t=123,v1=abc")

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
}

// --- Tests: Payment Failed ---

func TestWebhook_PaymentFailed_Returns200(t *testing.T) {
	verifier := &mockWebhookVerifier{
		event: makeEvent("invoice.payment_failed", map[string]string{
			"customer":     "cus_test",
			"subscription": "sub_test",
		}),
	}
	orgs := &mockWebhookOrgStore{}
	plans := &mockWebhookPlanStore{}
	e := setupWebhookTest(verifier, orgs, plans)

	rec := doWebhookRequest(e, `{}`, "t=123,v1=abc")

	assert.Equal(t, http.StatusOK, rec.Code)
}
