package payment

import (
	"fmt"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
)

type StripeProvider struct {
	secretKey     string
	webhookSecret string
}

func NewStripeProvider(secretKey, webhookSecret string) *StripeProvider {
	stripe.Key = secretKey
	return &StripeProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

type CheckoutSessionParams struct {
	OrganizationID string
	BillingCycleID string
	Amount         int64
	Currency       string
	SuccessURL     string
	CancelURL      string
	CustomerEmail  string
}

func (s *StripeProvider) CreateCheckoutSession(params CheckoutSessionParams) (*stripe.CheckoutSession, error) {
	sessionParams := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(params.Currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String("API Usage"),
						Description: stripe.String(fmt.Sprintf("Invoice #%s", params.BillingCycleID)),
					},
					UnitAmount: stripe.Int64(params.Amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL:    stripe.String(params.SuccessURL),
		CancelURL:     stripe.String(params.CancelURL),
		CustomerEmail: stripe.String(params.CustomerEmail),
		Metadata: map[string]string{
			"organization_id": params.OrganizationID,
			"billing_cycle_id": params.BillingCycleID,
		},
	}

	sess, err := session.New(sessionParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return sess, nil
}

func (s *StripeProvider) VerifyWebhookSignature(payload []byte, signature string) (stripe.Event, error) {
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return event, fmt.Errorf("webhook signature verification failed: %w", err)
	}
	return event, nil
}