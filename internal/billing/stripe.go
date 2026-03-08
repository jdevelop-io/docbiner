package billing

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v82"
	portalsession "github.com/stripe/stripe-go/v82/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/webhook"
)

// BillingProvider abstracts billing operations for testability.
type BillingProvider interface {
	CreateCustomer(ctx context.Context, orgName, email string) (string, error)
	CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (string, error)
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error)
}

// StripeService wraps the Stripe API for billing operations.
type StripeService struct {
	secretKey     string
	webhookSecret string
}

// New creates a new StripeService with the given API keys.
func New(secretKey, webhookSecret string) *StripeService {
	stripe.Key = secretKey
	return &StripeService{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

// CreateCustomer creates a Stripe customer for an org.
func (s *StripeService) CreateCustomer(_ context.Context, orgName, email string) (string, error) {
	params := &stripe.CustomerParams{
		Name:  stripe.String(orgName),
		Email: stripe.String(email),
	}

	c, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("create stripe customer: %w", err)
	}

	return c.ID, nil
}

// CreateCheckoutSession creates a checkout session for plan upgrade.
func (s *StripeService) CreateCheckoutSession(_ context.Context, customerID, priceID, successURL, cancelURL string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}

	return sess.URL, nil
}

// CreatePortalSession creates a billing portal session.
func (s *StripeService) CreatePortalSession(_ context.Context, customerID, returnURL string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	sess, err := portalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create portal session: %w", err)
	}

	return sess.URL, nil
}

// VerifyWebhookSignature verifies a Stripe webhook event.
func (s *StripeService) VerifyWebhookSignature(payload []byte, signature string) (*stripe.Event, error) {
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("verify webhook signature: %w", err)
	}

	return &event, nil
}
