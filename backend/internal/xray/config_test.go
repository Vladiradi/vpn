package xray

import (
	"encoding/json"
	"testing"
)

func TestConfigBuilder_Build(t *testing.T) {
	cfg, err := NewConfigBuilder(NodeConfig{
		RealityDest:       "www.microsoft.com",
		RealityPrivateKey: "test-private-key",
		RealityShortIDs:   []string{"", "0123456789abcdef"},
		DecoyHost:         "decoy.example.com",
	}, []Client{
		{ID: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", Flow: "xtls-rprx-vision"},
	}).Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify top-level keys exist.
	for _, key := range []string{"log", "inbounds", "outbounds", "routing"} {
		if _, ok := cfg[key]; !ok {
			t.Errorf("missing top-level key %q", key)
		}
	}

	// Verify inbound structure.
	inbounds, ok := cfg["inbounds"].([]map[string]any)
	if !ok || len(inbounds) != 1 {
		t.Fatalf("expected one inbound, got %v", cfg["inbounds"])
	}
	inbound := inbounds[0]
	if inbound["port"] != 443 {
		t.Errorf("expected port 443, got %v", inbound["port"])
	}
	if inbound["protocol"] != "vless" {
		t.Errorf("expected vless protocol, got %v", inbound["protocol"])
	}

	streamSettings, ok := inbound["streamSettings"].(map[string]any)
	if !ok {
		t.Fatalf("missing streamSettings")
	}
	if streamSettings["security"] != "reality" {
		t.Errorf("expected reality security, got %v", streamSettings["security"])
	}

	realitySettings, ok := streamSettings["realitySettings"].(map[string]any)
	if !ok {
		t.Fatalf("missing realitySettings")
	}
	if realitySettings["dest"] != "www.microsoft.com:443" {
		t.Errorf("unexpected dest: %v", realitySettings["dest"])
	}

	// Verify config is JSON-serializable.
	if _, err := json.Marshal(cfg); err != nil {
		t.Fatalf("config not JSON-serializable: %v", err)
	}
}

func TestConfigBuilder_Build_MissingDest(t *testing.T) {
	_, err := NewConfigBuilder(NodeConfig{
		RealityPrivateKey: "test-private-key",
		RealityShortIDs:   []string{""},
	}, nil).Build()
	if err == nil {
		t.Fatal("expected error for missing dest")
	}
}

func TestConfigBuilder_Build_MissingPrivateKey(t *testing.T) {
	_, err := NewConfigBuilder(NodeConfig{
		RealityDest:     "www.microsoft.com",
		RealityShortIDs: []string{""},
	}, nil).Build()
	if err == nil {
		t.Fatal("expected error for missing private key")
	}
}

func TestConfigBuilder_Build_MissingShortIDs(t *testing.T) {
	_, err := NewConfigBuilder(NodeConfig{
		RealityDest:       "www.microsoft.com",
		RealityPrivateKey: "test-private-key",
	}, nil).Build()
	if err == nil {
		t.Fatal("expected error for missing short ids")
	}
}
