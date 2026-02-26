package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// TokenClaims holds the subset of JWT claims we care about.
type TokenClaims struct {
	// UPN / login name (present for interactive user sessions)
	UPN              string `json:"upn"`
	UniqueName       string `json:"unique_name"`
	PreferredUsername string `json:"preferred_username"`
	Name             string `json:"name"`
	// OID is the Entra Object ID — useful as a suggested PrincipalID during setup.
	OID string `json:"oid"`
}

// DisplayName returns the best available human-readable identifier for the
// authenticated principal (UPN preferred, falling back through other claims).
func (c *TokenClaims) DisplayName() string {
	for _, candidate := range []string{c.UPN, c.UniqueName, c.PreferredUsername, c.Name} {
		if candidate != "" {
			return candidate
		}
	}
	return "unknown"
}

// GetTokenClaims obtains an ARM access token via cred and decodes the JWT
// payload claims without signature verification (we trust our own token).
func GetTokenClaims(ctx context.Context, cred azcore.TokenCredential) (*TokenClaims, error) {
	tok, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return nil, fmt.Errorf("acquiring token: %w", err)
	}

	// JWT is three base64url sections separated by '.'
	parts := strings.SplitN(tok.Token, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("unexpected token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding token payload: %w", err)
	}

	var claims TokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parsing token claims: %w", err)
	}
	return &claims, nil
}
