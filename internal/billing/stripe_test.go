package billing

import (
	"context"
	"testing"
)

// --- Mock BillingProvider ---

type mockBillingProvider struct {
	customerID string
	sessionURL string
	portalURL  string
	err        error
}

func (m *mockBillingProvider) CreateCustomer(_ context.Context, _, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.customerID, nil
}

func (m *mockBillingProvider) CreateCheckoutSession(_ context.Context, _, _, _, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.sessionURL, nil
}

func (m *mockBillingProvider) CreatePortalSession(_ context.Context, _, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.portalURL, nil
}

// --- Interface compliance ---

var _ BillingProvider = (*StripeService)(nil)
var _ BillingProvider = (*mockBillingProvider)(nil)

// --- Tests ---

func TestMockBillingProvider_CreateCustomer(t *testing.T) {
	mock := &mockBillingProvider{customerID: "cus_test123"}

	id, err := mock.CreateCustomer(context.Background(), "Test Org", "test@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cus_test123" {
		t.Errorf("expected cus_test123, got %s", id)
	}
}

func TestMockBillingProvider_CreateCheckoutSession(t *testing.T) {
	mock := &mockBillingProvider{sessionURL: "https://checkout.stripe.com/session/test"}

	url, err := mock.CreateCheckoutSession(
		context.Background(),
		"cus_test123",
		"price_test",
		"https://app.docbiner.com/success",
		"https://app.docbiner.com/cancel",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://checkout.stripe.com/session/test" {
		t.Errorf("expected checkout URL, got %s", url)
	}
}

func TestMockBillingProvider_CreatePortalSession(t *testing.T) {
	mock := &mockBillingProvider{portalURL: "https://billing.stripe.com/session/test"}

	url, err := mock.CreatePortalSession(
		context.Background(),
		"cus_test123",
		"https://app.docbiner.com/billing",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://billing.stripe.com/session/test" {
		t.Errorf("expected portal URL, got %s", url)
	}
}

func TestNew_SetsAPIKey(t *testing.T) {
	svc := New("sk_test_key", "whsec_test")
	if svc.secretKey != "sk_test_key" {
		t.Errorf("expected sk_test_key, got %s", svc.secretKey)
	}
	if svc.webhookSecret != "whsec_test" {
		t.Errorf("expected whsec_test, got %s", svc.webhookSecret)
	}
}

func TestVerifyWebhookSignature_InvalidSignature(t *testing.T) {
	svc := New("sk_test_key", "whsec_test_secret")

	_, err := svc.VerifyWebhookSignature([]byte(`{"type":"test"}`), "invalid_signature")
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
}
