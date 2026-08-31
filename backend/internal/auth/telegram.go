package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// TelegramUser represents verified Telegram user info.
type TelegramUser struct {
	ID        int64
	FirstName string
	Username  string
	Email     string // provided by frontend if needed
}

// VerifyTelegramAuth checks the hash from Telegram Login Widget.
func VerifyTelegramAuth(botToken string, data map[string]string, hash string) (*TelegramUser, error) {
	if botToken == "" {
		return nil, fmt.Errorf("telegram bot token not configured")
	}

	// Build data_check_string from sorted keys, excluding hash.
	var keys []string
	for k := range data {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, data[k]))
	}
	dataCheckString := strings.Join(pairs, "\n")

	// secret_key = HMAC-SHA256(bot_token, b"WebAppData")
	hmacWebApp := hmac.New(sha256.New, []byte("WebAppData"))
	hmacWebApp.Write([]byte(botToken))
	secretKey := hmacWebApp.Sum(nil)

	// expected_hash = HMAC-SHA256(data_check_string, secret_key)
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(expectedHash), []byte(hash)) {
		return nil, fmt.Errorf("invalid telegram hash")
	}

	id, err := strconv.ParseInt(data["id"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid telegram id: %w", err)
	}

	return &TelegramUser{
		ID:        id,
		FirstName: data["first_name"],
		Username:  data["username"],
		Email:     data["email"],
	}, nil
}
