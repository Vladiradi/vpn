// Package payments defines the payment provider interface used by billing.
package payments

import "context"

// Provider handles payment creation and verification.
type Provider interface {
	// CreatePayment initiates a payment and returns a provider-specific payment ID
	// and a URL/identifier the user should complete payment through.
	CreatePayment(ctx context.Context, params PaymentParams) (*PaymentResult, error)

	// VerifyPayment checks whether the payment was successfully completed.
	VerifyPayment(ctx context.Context, providerPaymentID string) (bool, error)
}

// PaymentParams contains the information needed to create a payment.
type PaymentParams struct {
	SubscriptionID string
	AmountRubles   int
	Description    string
	UserEmail      string
}

// PaymentResult is returned by CreatePayment.
type PaymentResult struct {
	ProviderPaymentID string
	PaymentURL        string
	Status            string
}
