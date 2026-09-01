package payments

import (
	"context"
	"fmt"
)

// PlategaProvider is a skeleton for platega.io integration.
// It is not yet fully wired to the real API; implement CreatePayment/VerifyPayment
// once API credentials and documentation are available.
type PlategaProvider struct {
	apiKey   string
	endpoint string
}

// NewPlategaProvider creates a new platega.io payment provider.
func NewPlategaProvider(apiKey, endpoint string) *PlategaProvider {
	if endpoint == "" {
		endpoint = "https://api.platega.io/v1"
	}
	return &PlategaProvider{
		apiKey:   apiKey,
		endpoint: endpoint,
	}
}

// CreatePayment should call platega.io API to create an invoice and return the payment URL.
// TODO: implement real HTTP call to platega.io /invoices or similar endpoint.
func (p *PlategaProvider) CreatePayment(ctx context.Context, params PaymentParams) (*PaymentResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("platega API key is not configured")
	}
	// Placeholder: replace with actual API request.
	return &PaymentResult{
		ProviderPaymentID: fmt.Sprintf("platega-%s", params.SubscriptionID),
		PaymentURL:        fmt.Sprintf("https://pay.platega.io/invoice/%s", params.SubscriptionID),
		Status:            "pending",
	}, nil
}

// VerifyPayment should call platega.io API to check invoice status.
// TODO: implement real HTTP call to platega.io /invoices/{id}/status or webhook handler.
func (p *PlategaProvider) VerifyPayment(ctx context.Context, providerPaymentID string) (bool, error) {
	if p.apiKey == "" {
		return false, fmt.Errorf("platega API key is not configured")
	}
	// Placeholder: replace with actual status check.
	return false, fmt.Errorf("platega verification not yet implemented")
}
