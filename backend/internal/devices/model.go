package devices

import "time"

// Device represents a user's VPN device.
type Device struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	SubscriptionID *string    `json:"subscription_id,omitempty"`
	NodeID         *string    `json:"node_id,omitempty"`
	Name           string     `json:"name"`
	UUID           string     `json:"uuid"`
	Protocol       string     `json:"protocol"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}
