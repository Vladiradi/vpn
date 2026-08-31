package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

// GoogleUser represents verified Google user info.
type GoogleUser struct {
	ID    string
	Email string
	Name  string
}

// VerifyGoogleIDToken verifies a Google ID token and returns user info.
func VerifyGoogleIDToken(ctx context.Context, clientID, idToken string) (*GoogleUser, error) {
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("create google provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	token, err := verifier.Verify(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("verify id token: %w", err)
	}

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	return &GoogleUser{
		ID:    token.Subject,
		Email: claims.Email,
		Name:  claims.Name,
	}, nil
}
