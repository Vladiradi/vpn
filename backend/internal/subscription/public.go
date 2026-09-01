package subscription

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/vpn-project/backend/internal/billing"
	"github.com/vpn-project/backend/internal/devices"
	"github.com/vpn-project/backend/internal/nodes"
)

// PublicHandler provides the public subscription endpoint /sub/{token}.
type PublicHandler struct {
	subRepo    *billing.Repository
	deviceRepo *devices.Repository
	nodeRepo   *nodes.Repository
}

// NewPublicHandler creates a new public subscription handler.
func NewPublicHandler(subRepo *billing.Repository, deviceRepo *devices.Repository, nodeRepo *nodes.Repository) *PublicHandler {
	return &PublicHandler{
		subRepo:    subRepo,
		deviceRepo: deviceRepo,
		nodeRepo:   nodeRepo,
	}
}

// ServeHTTP handles GET /sub/{token} and responds based on User-Agent.
func (h *PublicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := strings.TrimPrefix(r.URL.Path, "/sub/")
	if token == "" {
		respondError(w, http.StatusBadRequest, "subscription token required")
		return
	}

	ctx := r.Context()
	sub, err := h.subRepo.GetByToken(ctx, token)
	if err != nil {
		if err == billing.ErrSubscriptionNotFound {
			respondError(w, http.StatusNotFound, "subscription not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load subscription")
		return
	}

	if sub.Status != billing.StatusActive {
		respondError(w, http.StatusForbidden, "subscription is not active")
		return
	}

	device, err := h.deviceRepo.GetBySubscription(ctx, sub.ID)
	if err != nil {
		if err == devices.ErrDeviceNotFound {
			respondError(w, http.StatusNotFound, "device not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to load device")
		return
	}

	if device.NodeID == nil {
		respondError(w, http.StatusBadRequest, "device has no assigned node")
		return
	}

	node, err := h.nodeRepo.GetByID(ctx, *device.NodeID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load node")
		return
	}

	params, err := BuildConnectionParams(*device, *node)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	ua := strings.ToLower(r.UserAgent())
	isSingBox := strings.Contains(ua, "sing-box") || strings.Contains(ua, "hiddify") || strings.Contains(ua, "streisand")

	if isSingBox {
		data, err := MarshalJSON(*params)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to build config")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.json\"", sub.ID))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	uri := BuildVLESSURI(*params)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.txt\"", sub.ID))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(uri))
}

