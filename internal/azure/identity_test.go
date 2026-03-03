package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// --- mock TokenCredential ---------------------------------------------------

// mockCredential implements azcore.TokenCredential for testing.
type mockCredential struct {
	token string
	err   error
}

func (m *mockCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if m.err != nil {
		return azcore.AccessToken{}, m.err
	}
	return azcore.AccessToken{Token: m.token, ExpiresOn: time.Now().Add(time.Hour)}, nil
}

// craftJWT builds a minimal unsigned JWT (header.payload.signature) from an
// arbitrary claims map. The signature segment is left empty — this mirrors how
// ParseUnverified consumes the token.
func craftJWT(t *testing.T, claims map[string]any) string {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))

	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshalling claims: %v", err)
	}
	body := base64.RawURLEncoding.EncodeToString(payload)

	return header + "." + body + "."
}

// --- DisplayName tests -------------------------------------------------------

func TestDisplayName_UPN(t *testing.T) {
	c := &TokenClaims{UPN: "alice@contoso.com", UniqueName: "Alice U", PreferredUsername: "alice_pref", Name: "Alice"}
	if got := c.DisplayName(); got != "alice@contoso.com" {
		t.Errorf("DisplayName() = %q, want %q", got, "alice@contoso.com")
	}
}

func TestDisplayName_UniqueName(t *testing.T) {
	c := &TokenClaims{UniqueName: "unique_alice", PreferredUsername: "pref_alice", Name: "Alice"}
	if got := c.DisplayName(); got != "unique_alice" {
		t.Errorf("DisplayName() = %q, want %q", got, "unique_alice")
	}
}

func TestDisplayName_PreferredUsername(t *testing.T) {
	c := &TokenClaims{PreferredUsername: "pref_alice", Name: "Alice"}
	if got := c.DisplayName(); got != "pref_alice" {
		t.Errorf("DisplayName() = %q, want %q", got, "pref_alice")
	}
}

func TestDisplayName_Name(t *testing.T) {
	c := &TokenClaims{Name: "Alice"}
	if got := c.DisplayName(); got != "Alice" {
		t.Errorf("DisplayName() = %q, want %q", got, "Alice")
	}
}

func TestDisplayName_AllEmpty(t *testing.T) {
	c := &TokenClaims{}
	if got := c.DisplayName(); got != "unknown" {
		t.Errorf("DisplayName() = %q, want %q", got, "unknown")
	}
}

// --- GetTokenClaims tests ----------------------------------------------------

func TestGetTokenClaims_FullClaims(t *testing.T) {
	token := craftJWT(t, map[string]any{
		"upn":                "bob@contoso.com",
		"unique_name":        "Bob U",
		"preferred_username": "bob_pref",
		"name":               "Bob Builder",
		"oid":                "00000000-0000-0000-0000-000000000001",
	})

	cred := &mockCredential{token: token}
	claims, err := GetTokenClaims(context.Background(), cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claims.UPN != "bob@contoso.com" {
		t.Errorf("UPN = %q, want %q", claims.UPN, "bob@contoso.com")
	}
	if claims.UniqueName != "Bob U" {
		t.Errorf("UniqueName = %q, want %q", claims.UniqueName, "Bob U")
	}
	if claims.PreferredUsername != "bob_pref" {
		t.Errorf("PreferredUsername = %q, want %q", claims.PreferredUsername, "bob_pref")
	}
	if claims.Name != "Bob Builder" {
		t.Errorf("Name = %q, want %q", claims.Name, "Bob Builder")
	}
	if claims.OID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("OID = %q, want %q", claims.OID, "00000000-0000-0000-0000-000000000001")
	}
	if got := claims.DisplayName(); got != "bob@contoso.com" {
		t.Errorf("DisplayName() = %q, want %q", got, "bob@contoso.com")
	}
}

func TestGetTokenClaims_OnlyOID(t *testing.T) {
	token := craftJWT(t, map[string]any{
		"oid": "oid-only-value",
	})

	cred := &mockCredential{token: token}
	claims, err := GetTokenClaims(context.Background(), cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claims.OID != "oid-only-value" {
		t.Errorf("OID = %q, want %q", claims.OID, "oid-only-value")
	}
	if got := claims.DisplayName(); got != "unknown" {
		t.Errorf("DisplayName() = %q, want %q (no display-name claims set)", got, "unknown")
	}
}

func TestGetTokenClaims_DisplayNameFallback(t *testing.T) {
	// Only name claim set — DisplayName should return it.
	token := craftJWT(t, map[string]any{
		"name": "Fallback Name",
	})

	cred := &mockCredential{token: token}
	claims, err := GetTokenClaims(context.Background(), cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := claims.DisplayName(); got != "Fallback Name" {
		t.Errorf("DisplayName() = %q, want %q", got, "Fallback Name")
	}
}

func TestGetTokenClaims_CredentialError(t *testing.T) {
	cred := &mockCredential{err: errors.New("auth failed")}
	_, err := GetTokenClaims(context.Background(), cred)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "acquiring token: auth failed" {
		t.Errorf("error = %q, want %q", got, "acquiring token: auth failed")
	}
}

func TestGetTokenClaims_MalformedToken(t *testing.T) {
	cred := &mockCredential{token: "not-a-jwt"}
	_, err := GetTokenClaims(context.Background(), cred)
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestGetTokenClaims_EmptyPayload(t *testing.T) {
	token := craftJWT(t, map[string]any{})

	cred := &mockCredential{token: token}
	claims, err := GetTokenClaims(context.Background(), cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := claims.DisplayName(); got != "unknown" {
		t.Errorf("DisplayName() = %q, want %q", got, "unknown")
	}
	if claims.OID != "" {
		t.Errorf("OID = %q, want empty", claims.OID)
	}
}

func TestGetTokenClaims_RegisteredClaims(t *testing.T) {
	// Verify that standard JWT registered claims (sub, iss) are also parsed
	// via the embedded jwt.RegisteredClaims struct.
	token := craftJWT(t, map[string]any{
		"sub": "subject-id",
		"iss": "https://sts.windows.net/tenant-id/",
		"upn": "user@contoso.com",
	})

	cred := &mockCredential{token: token}
	claims, err := GetTokenClaims(context.Background(), cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, _ := claims.GetSubject(); got != "subject-id" {
		t.Errorf("Subject = %q, want %q", got, "subject-id")
	}
	if got, _ := claims.GetIssuer(); got != "https://sts.windows.net/tenant-id/" {
		t.Errorf("Issuer = %q, want %q", got, "https://sts.windows.net/tenant-id/")
	}
}
