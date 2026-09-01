package payments

import (
	"context"
	"fmt"
)

// StubProvider is a no-op payment provider that always succeeds.
// It is used for local development and MVP testing.
type StubProvider struct{}

// NewStubProvider creates a new stub payment provider.
func NewStubProvider() *StubProvider {
	return &StubProvider{}
}

// CreatePayment returns a fake payment ID and URL.
func (p *StubProvider) CreatePayment(ctx context.Context, params PaymentParams) (*PaymentResult, error) {
	return &PaymentResult{
		ProviderPaymentID: fmt.Sprintf("stub-%s", params.SubscriptionID),
		PaymentURL:        "",
		Status:            "pending",
	}, nil
}

// VerifyPayment always returns true for the stub provider.
func (p *StubProvider) VerifyPayment(ctx context.Context, providerPaymentID string) (bool, error) {
	return true, nil
}
