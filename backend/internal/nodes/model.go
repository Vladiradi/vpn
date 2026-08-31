package nodes

import "time"

// Status represents a node connection status.
type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
	StatusSyncing Status = "syncing"
)

// Node represents a VPN node.
type Node struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Hostname           string    `json:"hostname"`
	PublicIP           *string   `json:"public_ip,omitempty"`
	Status             Status    `json:"status"`
	TokenHash          string    `json:"-"`
	APIToken           string    `json:"api_token,omitempty"`
	RealityDest        *string   `json:"reality_dest,omitempty"`
	RealityPrivateKey  *string   `json:"-"`
	RealityPublicKey   *string   `json:"reality_public_key,omitempty"`
	RealityShortIDs    []string  `json:"reality_short_ids,omitempty"`
	DecoyHost          *string   `json:"decoy_host,omitempty"`
	LastSeenAt         *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
