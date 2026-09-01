package xray

import (
	"encoding/base64"
	"testing"
)

func TestGenerateRealityKeypair(t *testing.T) {
	priv, pub, err := GenerateRealityKeypair()
	if err != nil {
		t.Fatalf("GenerateRealityKeypair failed: %v", err)
	}
	if priv == "" || pub == "" {
		t.Fatal("expected non-empty keys")
	}

	// Keys should be valid base64 (RawURLEncoding) and decode to 32 bytes.
	privBytes, err := base64.RawURLEncoding.DecodeString(priv)
	if err != nil {
		t.Fatalf("private key is not valid base64: %v", err)
	}
	if len(privBytes) != 32 {
		t.Errorf("expected private key length 32, got %d", len(privBytes))
	}

	pubBytes, err := base64.RawURLEncoding.DecodeString(pub)
	if err != nil {
		t.Fatalf("public key is not valid base64: %v", err)
	}
	if len(pubBytes) != 32 {
		t.Errorf("expected public key length 32, got %d", len(pubBytes))
	}
}

func TestGenerateShortIDs(t *testing.T) {
	ids, err := GenerateShortIDs(3)
	if err != nil {
		t.Fatalf("GenerateShortIDs failed: %v", err)
	}
	if len(ids) != 4 {
		t.Fatalf("expected 4 ids (including empty), got %d", len(ids))
	}
	if ids[0] != "" {
		t.Errorf("expected first short id to be empty string, got %q", ids[0])
	}
	for i, id := range ids[1:] {
		if len(id) != 16 {
			t.Errorf("short id %d has unexpected length %d", i+1, len(id))
		}
	}
}

func TestGenerateShortIDs_InvalidCount(t *testing.T) {
	_, err := GenerateShortIDs(0)
	if err == nil {
		t.Fatal("expected error for n=0")
	}
}
