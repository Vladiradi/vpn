// Package subscription generates client-facing configs and VLESS URIs.
package subscription

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/vpn-project/backend/internal/devices"
	"github.com/vpn-project/backend/internal/nodes"
)

// ConnectionParams holds everything needed to build a client config.
type ConnectionParams struct {
	DeviceName string
	UUID       string
	Address    string
	Port       int
	SNI        string
	PublicKey  string
	ShortID    string
	Flow       string
}

// BuildVLESSURI returns a VLESS share link.
func BuildVLESSURI(p ConnectionParams) string {
	if p.Port == 0 {
		p.Port = 443
	}
	if p.Flow == "" {
		p.Flow = "xtls-rprx-vision"
	}
	if p.ShortID == "" && p.DeviceName == "" {
		p.DeviceName = "VPN"
	}

	q := url.Values{}
	q.Set("security", "reality")
	q.Set("flow", p.Flow)
	q.Set("type", "tcp")
	q.Set("sni", p.SNI)
	q.Set("fp", "chrome")
	q.Set("pbk", p.PublicKey)
	q.Set("sid", p.ShortID)

	u := url.URL{
		Scheme:   "vless",
		User:     url.User(p.UUID),
		Host:     fmt.Sprintf("%s:%d", p.Address, p.Port),
		RawQuery: q.Encode(),
		Fragment: p.DeviceName,
	}
	return u.String()
}

// SingBoxConfig returns a sing-box outbound JSON fragment for the connection.
func SingBoxConfig(p ConnectionParams) (map[string]any, error) {
	if p.Port == 0 {
		p.Port = 443
	}
	if p.Flow == "" {
		p.Flow = "xtls-rprx-vision"
	}

	cfg := map[string]any{
		"type":        "vless",
		"tag":         "proxy",
		"server":      p.Address,
		"server_port": p.Port,
		"uuid":        p.UUID,
		"flow":        p.Flow,
		"tls": map[string]any{
			"enabled":     true,
			"server_name": p.SNI,
			"utls": map[string]any{
				"enabled":     true,
				"fingerprint": "chrome",
			},
			"reality": map[string]any{
				"enabled":    true,
				"public_key": p.PublicKey,
				"short_id":   p.ShortID,
			},
		},
	}
	return cfg, nil
}

// BuildConnectionParams combines a device and its node into connection params.
func BuildConnectionParams(device devices.Device, node nodes.Node) (*ConnectionParams, error) {
	if node.RealityDest == nil || *node.RealityDest == "" {
		return nil, fmt.Errorf("node reality dest not configured")
	}
	if node.RealityPublicKey == nil || *node.RealityPublicKey == "" {
		return nil, fmt.Errorf("node reality public key not configured")
	}

	address := node.Hostname
	if node.PublicIP != nil && *node.PublicIP != "" {
		address = *node.PublicIP
	}

	shortID := ""
	if len(node.RealityShortIDs) > 0 {
		shortID = node.RealityShortIDs[0]
	}

	return &ConnectionParams{
		DeviceName: device.Name,
		UUID:       device.UUID,
		Address:    address,
		Port:       443,
		SNI:        *node.RealityDest,
		PublicKey:  *node.RealityPublicKey,
		ShortID:    shortID,
		Flow:       "xtls-rprx-vision",
	}, nil
}

// MarshalJSON returns the sing-box config as indented JSON bytes.
func MarshalJSON(p ConnectionParams) ([]byte, error) {
	cfg, err := SingBoxConfig(p)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(cfg, "", "  ")
}
