package subscription

import (
	"strings"
	"testing"

	"github.com/vpn-project/backend/internal/devices"
	"github.com/vpn-project/backend/internal/nodes"
)

func TestBuildVLESSURI(t *testing.T) {
	uri := BuildVLESSURI(ConnectionParams{
		DeviceName: "iPhone",
		UUID:       "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
		Address:    "203.0.113.10",
		Port:       443,
		SNI:        "www.microsoft.com",
		PublicKey:  "publicKeyBase64",
		ShortID:    "0123456789abcdef",
		Flow:       "xtls-rprx-vision",
	})

	if !strings.HasPrefix(uri, "vless://") {
		t.Fatalf("expected vless URI, got %s", uri)
	}
	if !strings.Contains(uri, "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11") {
		t.Errorf("expected uuid in URI")
	}
	if !strings.Contains(uri, "203.0.113.10:443") {
		t.Errorf("expected address in URI")
	}
	if !strings.Contains(uri, "security=reality") {
		t.Errorf("expected reality security in URI")
	}
	if !strings.Contains(uri, "pbk=publicKeyBase64") {
		t.Errorf("expected public key in URI")
	}
	if !strings.Contains(uri, "#iPhone") {
		t.Errorf("expected device name fragment")
	}
}

func TestBuildConnectionParams(t *testing.T) {
	dest := "www.microsoft.com"
	pubKey := "publicKeyBase64"
	ip := "203.0.113.10"
	node := nodes.Node{
		Hostname:         "node1.example.com",
		PublicIP:         &ip,
		RealityDest:      &dest,
		RealityPublicKey: &pubKey,
		RealityShortIDs:  []string{"", "0123456789abcdef"},
	}
	device := devices.Device{
		Name: "Test",
		UUID: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
	}

	params, err := BuildConnectionParams(device, node)
	if err != nil {
		t.Fatalf("BuildConnectionParams failed: %v", err)
	}
	if params.Address != ip {
		t.Errorf("expected address %s, got %s", ip, params.Address)
	}
	if params.SNI != dest {
		t.Errorf("expected SNI %s, got %s", dest, params.SNI)
	}
	if params.PublicKey != pubKey {
		t.Errorf("expected public key %s, got %s", pubKey, params.PublicKey)
	}
	if params.ShortID != "" {
		t.Errorf("expected first short id to be empty, got %s", params.ShortID)
	}
}

func TestBuildConnectionParams_MissingDest(t *testing.T) {
	node := nodes.Node{Hostname: "node1.example.com"}
	device := devices.Device{UUID: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"}
	_, err := BuildConnectionParams(device, node)
	if err == nil {
		t.Fatal("expected error for missing dest")
	}
}

func TestSingBoxConfig(t *testing.T) {
	cfg, err := SingBoxConfig(ConnectionParams{
		UUID:      "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
		Address:   "203.0.113.10",
		SNI:       "www.microsoft.com",
		PublicKey: "publicKeyBase64",
		ShortID:   "0123456789abcdef",
	})
	if err != nil {
		t.Fatalf("SingBoxConfig failed: %v", err)
	}
	if cfg["type"] != "vless" {
		t.Errorf("expected vless type")
	}
	tls, ok := cfg["tls"].(map[string]any)
	if !ok {
		t.Fatalf("missing tls section")
	}
	if tls["enabled"] != true {
		t.Errorf("expected tls enabled")
	}
}
