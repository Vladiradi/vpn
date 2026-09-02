// Package xray generates XRay-core configuration payloads.
package xray

import (
	"fmt"
)

// Client represents a VLESS client entry.
type Client struct {
	ID   string `json:"id"`
	Flow string `json:"flow,omitempty"`
}

// NodeConfig holds the XRay/REALITY parameters for a single node.
type NodeConfig struct {
	RealityDest       string
	RealityPrivateKey string
	RealityShortIDs   []string
	DecoyHost         string
}

// ConfigBuilder builds an XRay config.json compatible map.
type ConfigBuilder struct {
	Node    NodeConfig
	Clients []Client
}

// NewConfigBuilder creates a builder for the given node and clients.
func NewConfigBuilder(node NodeConfig, clients []Client) *ConfigBuilder {
	return &ConfigBuilder{Node: node, Clients: clients}
}

// Build returns the complete XRay config as a map ready for JSON encoding.
func (b *ConfigBuilder) Build() (map[string]any, error) {
	if b.Node.RealityDest == "" {
		return nil, fmt.Errorf("reality dest is required")
	}
	if b.Node.RealityPrivateKey == "" {
		return nil, fmt.Errorf("reality private key is required")
	}
	if len(b.Node.RealityShortIDs) == 0 {
		return nil, fmt.Errorf("at least one reality short id is required")
	}

	clients := make([]map[string]any, 0, len(b.Clients))
	for _, c := range b.Clients {
		flow := c.Flow
		if flow == "" {
			flow = "xtls-rprx-vision"
		}
		clients = append(clients, map[string]any{
			"id":    c.ID,
			"email": c.ID,
			"flow":  flow,
		})
	}
	if len(clients) == 0 {
		// XRay requires at least one client; use a placeholder that will be
		// rejected by the client but keeps the config valid until devices are added.
		clients = append(clients, map[string]any{
			"id":   "00000000-0000-0000-0000-000000000000",
			"flow": "xtls-rprx-vision",
		})
	}

	inbounds := []map[string]any{
		{
			"port":     443,
			"protocol": "vless",
			"settings": map[string]any{
				"clients":    clients,
				"decryption": "none",
				"fallbacks": []map[string]any{
					{
						"dest": b.decoyAddress(),
					},
				},
			},
			"streamSettings": map[string]any{
				"network":  "tcp",
				"security": "reality",
				"realitySettings": map[string]any{
					"show":        false,
					"dest":        fmt.Sprintf("%s:443", b.Node.RealityDest),
					"xver":        0,
					"serverNames": []string{b.Node.RealityDest},
					"privateKey":  b.Node.RealityPrivateKey,
					"shortIds":    b.Node.RealityShortIDs,
					"maxTimeDiff": 0,
				},
			},
			"sniffing": map[string]any{
				"enabled":      true,
				"destOverride": []string{"http", "tls", "quic"},
			},
		},
		{
			"listen":   "127.0.0.1",
			"port":     10085,
			"protocol": "dokodemo-door",
			"settings": map[string]any{
				"address": "127.0.0.1",
			},
			"tag": "api",
		},
	}

	cfg := map[string]any{
		"log": map[string]any{
			"loglevel": "warning",
		},
		"api": map[string]any{
			"tag":      "api",
			"services": []string{"StatsService"},
		},
		"inbounds": inbounds,
		"outbounds": []map[string]any{
			{
				"protocol": "freedom",
				"tag":      "direct",
			},
		},
		"routing": map[string]any{
			"rules": []map[string]any{
				{
					"inboundTag":  []string{"api"},
					"outboundTag": "api",
				},
			},
		},
		"stats": map[string]any{},
		"policy": map[string]any{
			"levels": map[string]any{
				"0": map[string]any{
					"statsUserUplink":   true,
					"statsUserDownlink": true,
				},
			},
			"system": map[string]any{
				"statsInboundUplink":   true,
				"statsInboundDownlink": true,
			},
		},
	}

	return cfg, nil
}

func (b *ConfigBuilder) decoyAddress() string {
	if b.Node.DecoyHost != "" {
		return fmt.Sprintf("%s:80", b.Node.DecoyHost)
	}
	return "127.0.0.1:8080"
}
