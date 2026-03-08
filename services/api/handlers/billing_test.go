package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock BillingProvider ---

type mockBillingProvider struct {
	customerID string
	sessionURL string
	portalURL  string
	err        error

	lastOrgName    string
	lastEmail      string
	lastCustomerID string
	lastPriceID    string
	lastSuccessURL string
	lastCancelURL  string
	lastReturnURL  string
}

func (m *mockBillingProvider) CreateCustomer(_ context.Context, orgName, email string) (string, error) {
	m.lastOrgName = orgName
	m.lastEmail = email
	if m.err != nil {
		return "", m.err
	}
	return m.customerID, nil
}

func (m *mockBillingProvider) CreateCheckoutSession(_ context.Context, customerID, priceID, successURL, cancelURL string) (string, error) {
	m.lastCustomerID = customerID
	m.lastPriceID = priceID
	m.lastSuccessURL = successURL
	m.lastCancelURL = cancelURL
	if m.err != nil {
		return "", m.err
	}
	return m.sessionURL, nil
}

func (m *mockBillingProvider) CreatePortalSession(_ context.Context, customerID, returnURL string) (string, error) {
	m.lastCustomerID = customerID
	m.lastReturnURL = returnURL
	if m.err != nil {
		return "", m.err
	}
	return m.portalURL, nil
}

// --- Mock BillingOrgStore ---

type mockBillingOrgStore struct {
	org *domain.Organization
	err error

	lastUpdateOrgID      uuid.UUID
	lastUpdateCustomerID string
}

func (m *mockBillingOrgStore) GetByID(_ context.Context, id uuid.UUID) (*domain.Organization, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.org, nil
}

func (m *mockBillingOrgStore) UpdateStripeCustomerID(_ context.Context, orgID uuid.UUID, customerID string) error {
	m.lastUpdateOrgID = orgID
	m.lastUpdateCustomerID = customerID
	return nil
}

// --- Mock BillingPlanStore ---

type mockBillingPlanStore struct {
	plan *domain.Plan
	err  error
}

func (m *mockBillingPlanStore) GetByID(_ context.Context, id uuid.UUID) (*domain.Plan, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.plan, nil
}

// --- Test Helpers ---

func newTestOrg(stripeCustomerID string) *domain.Organization {
	return &domain.Organization{
		ID:               uuid.New(),
		Name:             "Test Org",
		Slug:             "test-org",
		PlanID:           uuid.New(),
		StripeCustomerID: stripeCustomerID,
	}
}

func newTestPlan() *domain.Plan {
	return &domain.Plan{
		ID:           uuid.New(),
		Name:         "starter",
		MonthlyQuota: 2500,
		PriceMonthly: 19.00,
		PriceYearly:  190.00,
	}
}

func setupBillingTest(billing BillingProvider, orgs BillingOrgStore, plans BillingPlanStore) *echo.Echo {
	e := echo.New()
	h := NewBillingHandler(billing, orgs, plans)

	orgID := uuid.New()

	setAuth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("org_id", orgID)
			return next(c)
		}
	}

	v1 := e.Group("/v1", setAuth)
	v1.POST("/billing/checkout", h.HandleCheckout)
	v1.POST("/billing/portal", h.HandlePortal)
	v1.GET("/billing/status", h.HandleStatus)

	return e
}

func doBillingRequest(e *echo.Echo, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// --- Tests: Checkout ---

func TestBilling_Checkout_Success(t *testing.T) {
	billing := &mockBillingProvider{
		sessionURL: "https://checkout.stripe.com/session/test123",
	}
	orgs := &mockBillingOrgStore{
		org: newTestOrg("cus_existing"),
	}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{
		"price_id": "price_test123",
		"success_url": "https://app.docbiner.com/success",
		"cancel_url": "https://app.docbiner.com/cancel"
	}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/checkout", body)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp CheckoutResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "https://checkout.stripe.com/session/test123", resp.URL)
	assert.Equal(t, "cus_existing", billing.lastCustomerID)
	assert.Equal(t, "price_test123", billing.lastPriceID)
}

func TestBilling_Checkout_CreatesCustomerIfMissing(t *testing.T) {
	billing := &mockBillingProvider{
		customerID: "cus_new123",
		sessionURL: "https://checkout.stripe.com/session/new",
	}
	orgs := &mockBillingOrgStore{
		org: newTestOrg(""), // No Stripe customer
	}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{
		"price_id": "price_test",
		"success_url": "https://app.docbiner.com/success",
		"cancel_url": "https://app.docbiner.com/cancel"
	}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/checkout", body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Test Org", billing.lastOrgName)
	assert.Equal(t, "cus_new123", orgs.lastUpdateCustomerID)
	assert.Equal(t, "cus_new123", billing.lastCustomerID)
}

func TestBilling_Checkout_MissingPriceID(t *testing.T) {
	billing := &mockBillingProvider{}
	orgs := &mockBillingOrgStore{org: newTestOrg("cus_test")}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{"success_url": "https://a.com", "cancel_url": "https://b.com"}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/checkout", body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_error", errResp.Error)
	assert.Contains(t, errResp.Message, "price_id is required")
}

func TestBilling_Checkout_MissingSuccessURL(t *testing.T) {
	billing := &mockBillingProvider{}
	orgs := &mockBillingOrgStore{org: newTestOrg("cus_test")}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{"price_id": "price_test", "cancel_url": "https://b.com"}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/checkout", body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp.Message, "success_url is required")
}

func TestBilling_Checkout_MissingCancelURL(t *testing.T) {
	billing := &mockBillingProvider{}
	orgs := &mockBillingOrgStore{org: newTestOrg("cus_test")}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{"price_id": "price_test", "success_url": "https://a.com"}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/checkout", body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp.Message, "cancel_url is required")
}

func TestBilling_Checkout_StripeError(t *testing.T) {
	billing := &mockBillingProvider{
		err: errors.New("stripe unavailable"),
	}
	orgs := &mockBillingOrgStore{org: newTestOrg("cus_test")}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{
		"price_id": "price_test",
		"success_url": "https://a.com",
		"cancel_url": "https://b.com"
	}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/checkout", body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBilling_Checkout_OrgNotFound(t *testing.T) {
	billing := &mockBillingProvider{}
	orgs := &mockBillingOrgStore{err: errors.New("not found")}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{
		"price_id": "price_test",
		"success_url": "https://a.com",
		"cancel_url": "https://b.com"
	}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/checkout", body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- Tests: Portal ---

func TestBilling_Portal_Success(t *testing.T) {
	billing := &mockBillingProvider{
		portalURL: "https://billing.stripe.com/session/portal123",
	}
	orgs := &mockBillingOrgStore{
		org: newTestOrg("cus_existing"),
	}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{"return_url": "https://app.docbiner.com/billing"}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/portal", body)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp PortalResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "https://billing.stripe.com/session/portal123", resp.URL)
	assert.Equal(t, "cus_existing", billing.lastCustomerID)
}

func TestBilling_Portal_NoStripeCustomer(t *testing.T) {
	billing := &mockBillingProvider{}
	orgs := &mockBillingOrgStore{
		org: newTestOrg(""), // No Stripe customer
	}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{"return_url": "https://app.docbiner.com/billing"}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/portal", body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp.Message, "No billing account found")
}

func TestBilling_Portal_MissingReturnURL(t *testing.T) {
	billing := &mockBillingProvider{}
	orgs := &mockBillingOrgStore{org: newTestOrg("cus_test")}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/portal", body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp.Message, "return_url is required")
}

func TestBilling_Portal_StripeError(t *testing.T) {
	billing := &mockBillingProvider{
		err: errors.New("stripe unavailable"),
	}
	orgs := &mockBillingOrgStore{org: newTestOrg("cus_test")}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	body := `{"return_url": "https://app.docbiner.com/billing"}`
	rec := doBillingRequest(e, http.MethodPost, "/v1/billing/portal", body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- Tests: Status ---

func TestBilling_Status_Success(t *testing.T) {
	billing := &mockBillingProvider{}
	orgs := &mockBillingOrgStore{org: newTestOrg("cus_abc")}
	plan := newTestPlan()
	plans := &mockBillingPlanStore{plan: plan}
	e := setupBillingTest(billing, orgs, plans)

	rec := doBillingRequest(e, http.MethodGet, "/v1/billing/status", "")

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp BillingStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "starter", resp.Plan)
	assert.Equal(t, "cus_abc", resp.StripeCustomerID)
	assert.Equal(t, 19.00, resp.PriceMonthly)
	assert.Equal(t, 190.00, resp.PriceYearly)
	assert.Equal(t, 2500, resp.MonthlyQuota)
}

func TestBilling_Status_OrgError(t *testing.T) {
	billing := &mockBillingProvider{}
	orgs := &mockBillingOrgStore{err: errors.New("not found")}
	plans := &mockBillingPlanStore{plan: newTestPlan()}
	e := setupBillingTest(billing, orgs, plans)

	rec := doBillingRequest(e, http.MethodGet, "/v1/billing/status", "")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBilling_Status_PlanError(t *testing.T) {
	billing := &mockBillingProvider{}
	orgs := &mockBillingOrgStore{org: newTestOrg("cus_test")}
	plans := &mockBillingPlanStore{err: errors.New("not found")}
	e := setupBillingTest(billing, orgs, plans)

	rec := doBillingRequest(e, http.MethodGet, "/v1/billing/status", "")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
