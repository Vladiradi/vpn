package clientapps

import (
	"encoding/json"
	"net/http"
)

// AppLink represents a link to a client application.
type AppLink struct {
	Platform string `json:"platform"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Notes    string `json:"notes,omitempty"`
}

// Handler provides the client applications list endpoint.
type Handler struct{}

// NewHandler creates a new client apps handler.
func NewHandler() *Handler {
	return &Handler{}
}

// List returns recommended client applications by platform.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	links := []AppLink{
		{
			Platform: "ios",
			Name:     "Streisand",
			URL:      "https://apps.apple.com/app/streisand/id6451274064",
			Notes:    "Бесплатно, sing-box, сканируйте QR",
		},
		{
			Platform: "ios",
			Name:     "FoXray",
			URL:      "https://apps.apple.com/app/foxray/id6446811860",
			Notes:    "Бесплатно, VLESS+REALITY",
		},
		{
			Platform: "android",
			Name:     "v2rayNG",
			URL:      "https://github.com/2dust/v2rayNG/releases",
			Notes:    "Бесплатно, open-source",
		},
		{
			Platform: "android",
			Name:     "Hiddify",
			URL:      "https://github.com/hiddify/hiddify-next/releases",
			Notes:    "Бесплатно, sing-box",
		},
		{
			Platform: "macos",
			Name:     "V2RayXS",
			URL:      "https://github.com/tzmax/V2RayXS/releases",
			Notes:    "Бесплатно, open-source",
		},
		{
			Platform: "macos",
			Name:     "Hiddify",
			URL:      "https://github.com/hiddify/hiddify-next/releases",
			Notes:    "Бесплатно, sing-box",
		},
		{
			Platform: "windows",
			Name:     "v2rayN",
			URL:      "https://github.com/2dust/v2rayN/releases",
			Notes:    "Бесплатно, open-source",
		},
		{
			Platform: "windows",
			Name:     "Hiddify",
			URL:      "https://github.com/hiddify/hiddify-next/releases",
			Notes:    "Бесплатно, sing-box",
		},
		{
			Platform: "linux",
			Name:     "Hiddify",
			URL:      "https://github.com/hiddify/hiddify-next/releases",
			Notes:    "Бесплатно, sing-box",
		},
	}

	respondJSON(w, http.StatusOK, map[string]any{"apps": links})
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
