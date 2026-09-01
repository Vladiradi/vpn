package billing

import "time"

// Subscription represents a user's VPN subscription.
type Subscription struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	OfferingID        string     `json:"offering_id"`
	SubscriptionToken *string    `json:"subscription_token,omitempty"`
	Status            string     `json:"status"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	TotalTrafficBytes int64      `json:"total_traffic_bytes"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Status values for subscriptions.
const (
	StatusPending   = "pending"
	StatusActive    = "active"
	StatusExpired   = "expired"
	StatusCancelled = "cancelled"
)
