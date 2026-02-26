package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims holds the subset of JWT claims we care about.
type TokenClaims struct {
	jwt.RegisteredClaims

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
// payload claims.
//
// # Security: JWT Trust Boundary
//
// This function uses [jwt.Parser.ParseUnverified] — the token's cryptographic
// signature is NOT verified. This is acceptable here because:
//
//  1. The token is obtained directly from [azcore.TokenCredential.GetToken],
//     which retrieves it over TLS from Microsoft Entra ID (Azure AD). We are
//     the original requester, not a relying party receiving the token from an
//     external source.
//  2. Verifying the signature would require fetching Entra's OIDC discovery
//     document and JWKS keys, adding network round-trips and complexity with
//     no security benefit in this context.
//  3. The extracted claims are used only for display purposes (greeting the
//     user) and as a suggested principal ID during setup — never for
//     authorization decisions.
//
// WARNING: Do NOT reuse this function with tokens received from untrusted
// sources (e.g. user input, incoming HTTP requests, third-party services).
// In those scenarios full signature verification against Entra's JWKS
// endpoint is required.
func GetTokenClaims(ctx context.Context, cred azcore.TokenCredential) (*TokenClaims, error) {
	tok, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return nil, fmt.Errorf("acquiring token: %w", err)
	}

	var claims TokenClaims
	parser := jwt.NewParser()
	if _, _, err := parser.ParseUnverified(tok.Token, &claims); err != nil {
		return nil, fmt.Errorf("parsing token claims: %w", err)
	}
	return &claims, nil
}
