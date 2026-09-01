package xray

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// GenerateRealityKeypair generates a fresh X25519 keypair for REALITY.
// Returns base64-encoded private and public keys.
func GenerateRealityKeypair() (privateKey string, publicKey string, err error) {
	curve := ecdh.X25519()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate x25519 key: %w", err)
	}
	pub := priv.PublicKey()
	return base64.RawURLEncoding.EncodeToString(priv.Bytes()),
		base64.RawURLEncoding.EncodeToString(pub.Bytes()),
		nil
}

// GenerateShortIDs generates n random REALITY short IDs as lowercase hex strings.
// Each short ID is between 1 and 8 bytes (2..16 hex chars). The first entry is
// always the empty string to maximize compatibility.
func GenerateShortIDs(n int) ([]string, error) {
	if n < 1 {
		return nil, fmt.Errorf("n must be at least 1")
	}
	ids := make([]string, 0, n+1)
	ids = append(ids, "") // empty short id is valid and commonly used
	for i := 0; i < n; i++ {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("generate short id: %w", err)
		}
		ids = append(ids, hex.EncodeToString(b))
	}
	return ids, nil
}
