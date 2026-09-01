package subscription

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/skip2/go-qrcode"
	"github.com/vpn-project/backend/internal/devices"
	"github.com/vpn-project/backend/internal/middleware"
	"github.com/vpn-project/backend/internal/nodes"
)

// Handler provides config/QR/file delivery endpoints.
type Handler struct {
	deviceRepo *devices.Repository
	nodeRepo   *nodes.Repository
}

// NewHandler creates a new subscription handler.
func NewHandler(deviceRepo *devices.Repository, nodeRepo *nodes.Repository) *Handler {
	return &Handler{deviceRepo: deviceRepo, nodeRepo: nodeRepo}
}

// GetConfig returns the VLESS URI and sing-box JSON for a device.
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	device, node, ok := h.loadDeviceAndNode(w, r)
	if !ok {
		return
	}

	params, err := BuildConnectionParams(*device, *node)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	singBoxJSON, err := MarshalJSON(*params)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to build config")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"vless_uri":     BuildVLESSURI(*params),
		"sing_box_json": json.RawMessage(singBoxJSON),
	})
}

// GetQR returns a PNG QR code for the VLESS URI.
func (h *Handler) GetQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	device, node, ok := h.loadDeviceAndNode(w, r)
	if !ok {
		return
	}

	params, err := BuildConnectionParams(*device, *node)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	uri := BuildVLESSURI(*params)
	png, err := qrcode.Encode(uri, qrcode.Medium, 256)
	if err != nil {
		log.Printf("qr encode error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to generate qr")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s.png\"", device.Name))
	w.WriteHeader(http.StatusOK)
	w.Write(png)
}

// GetFile returns a downloadable config file based on the ?format query param.
// Supported formats: vless (plain text URI), sing-box (JSON).
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	device, node, ok := h.loadDeviceAndNode(w, r)
	if !ok {
		return
	}

	params, err := BuildConnectionParams(*device, *node)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	format := r.URL.Query().Get("format")
	switch format {
	case "", "vless":
		uri := BuildVLESSURI(*params)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.txt\"", device.Name))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(uri))
	case "sing-box":
		data, err := MarshalJSON(*params)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to build config")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.json\"", device.Name))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	default:
		respondError(w, http.StatusBadRequest, "unsupported format")
	}
}

func (h *Handler) loadDeviceAndNode(w http.ResponseWriter, r *http.Request) (*devices.Device, *nodes.Node, bool) {
	claims, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return nil, nil, false
	}

	id := r.PathValue("id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "device id required")
		return nil, nil, false
	}

	device, err := h.deviceRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, devices.ErrDeviceNotFound) {
			respondError(w, http.StatusNotFound, "device not found")
			return nil, nil, false
		}
		respondError(w, http.StatusInternalServerError, "failed to load device")
		return nil, nil, false
	}

	if device.UserID != claims.UserID {
		respondError(w, http.StatusForbidden, "access denied")
		return nil, nil, false
	}

	if device.NodeID == nil {
		respondError(w, http.StatusBadRequest, "device has no assigned node")
		return nil, nil, false
	}

	node, err := h.nodeRepo.GetByID(r.Context(), *device.NodeID)
	if err != nil {
		if errors.Is(err, nodes.ErrNodeNotFound) {
			respondError(w, http.StatusNotFound, "node not found")
			return nil, nil, false
		}
		respondError(w, http.StatusInternalServerError, "failed to load node")
		return nil, nil, false
	}

	return device, node, true
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
