package offerings

import "time"

// Offering represents an available VPN location/speed/price option.
type Offering struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	DurationMonths int       `json:"duration_months"`
	PriceRubles    int       `json:"price_rubles"`
	SpeedMbps      int       `json:"speed_mbps"`
	CountryCode    string    `json:"country_code"`
	CountryName    string    `json:"country_name"`
	Region         *string   `json:"region,omitempty"`
	IsActive       bool      `json:"is_active"`
	IsAvailable    bool      `json:"is_available"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
