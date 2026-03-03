// Package azure wraps the Azure SDK clients used by the PIM CLI.
package azure

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

// ---------------------------------------------------------------------------
// Interfaces — enable dependency injection and unit testing without live
// Azure credentials.
// ---------------------------------------------------------------------------

// Authenticator abstracts credential acquisition and JWT claim extraction.
// The default implementation ([DefaultAuthenticator]) delegates to the Azure
// SDK; tests can substitute a stub that returns crafted tokens.
type Authenticator interface {
	// NewCredential returns an Azure token credential for ARM API calls.
	NewCredential() (azcore.TokenCredential, error)
	// GetTokenClaims obtains an ARM access token and decodes its JWT claims.
	GetTokenClaims(ctx context.Context, cred azcore.TokenCredential) (*TokenClaims, error)
}

// RoleFetcher abstracts retrieval of PIM role assignments for a single
// subscription. [*Clients] satisfies this interface.
type RoleFetcher interface {
	// FetchEligibleRoles returns all PIM-eligible role assignments visible to the
	// authenticated principal within scope.
	FetchEligibleRoles(ctx context.Context, scope, subscriptionName string, re *regexp.Regexp, envLabels map[string]string) ([]model.Role, error)
	// FetchActiveRoles returns currently active (JIT-activated) PIM role
	// assignments within scope.
	FetchActiveRoles(ctx context.Context, scope, subscriptionName string, justificationMap map[string]string, re *regexp.Regexp, envLabels map[string]string) ([]model.ActiveRole, error)
}

// RoleActivator abstracts PIM role activation requests. [*Clients] satisfies
// this interface.
type RoleActivator interface {
	// ActivateRoles sends parallel SelfActivate requests for the given roles.
	ActivateRoles(ctx context.Context, roles []model.Role, principalID, justification string, duration model.DurationOption) []model.ActivationResult
}

// Compile-time interface satisfaction checks.
var (
	_ Authenticator = DefaultAuthenticator{}
	_ RoleFetcher   = (*Clients)(nil)
	_ RoleActivator = (*Clients)(nil)
)

// ---------------------------------------------------------------------------
// DefaultAuthenticator — production Authenticator backed by the Azure SDK.
// ---------------------------------------------------------------------------

// DefaultAuthenticator implements [Authenticator] using the Azure SDK's
// DefaultAzureCredential and the package-level [GetTokenClaims] function.
type DefaultAuthenticator struct{}

// NewCredential delegates to the package-level [NewCredential] function.
func (DefaultAuthenticator) NewCredential() (azcore.TokenCredential, error) {
	return NewCredential()
}

// GetTokenClaims delegates to the package-level [GetTokenClaims] function.
func (DefaultAuthenticator) GetTokenClaims(ctx context.Context, cred azcore.TokenCredential) (*TokenClaims, error) {
	return GetTokenClaims(ctx, cred)
}

// ---------------------------------------------------------------------------
// Clients — ARM service clients for a single subscription.
// ---------------------------------------------------------------------------

// Clients holds the three ARM authorization service clients needed by the CLI
// for a single Azure subscription.
type Clients struct {
	Eligible   *armauthorization.RoleEligibilityScheduleInstancesClient
	Active     *armauthorization.RoleAssignmentScheduleInstancesClient
	Activation *armauthorization.RoleAssignmentScheduleRequestsClient
}

// FetchEligibleRoles satisfies [RoleFetcher] by delegating to the package-level
// function of the same name using the embedded Eligible client.
func (c *Clients) FetchEligibleRoles(ctx context.Context, scope, subscriptionName string, re *regexp.Regexp, envLabels map[string]string) ([]model.Role, error) {
	return FetchEligibleRoles(ctx, c.Eligible, scope, subscriptionName, re, envLabels)
}

// FetchActiveRoles satisfies [RoleFetcher] by delegating to the package-level
// function of the same name using the embedded Active client.
func (c *Clients) FetchActiveRoles(ctx context.Context, scope, subscriptionName string, justificationMap map[string]string, re *regexp.Regexp, envLabels map[string]string) ([]model.ActiveRole, error) {
	return FetchActiveRoles(ctx, c.Active, scope, subscriptionName, justificationMap, re, envLabels)
}

// ActivateRoles satisfies [RoleActivator] by delegating to the package-level
// function of the same name using the embedded Activation client.
func (c *Clients) ActivateRoles(ctx context.Context, roles []model.Role, principalID, justification string, duration model.DurationOption) []model.ActivationResult {
	return ActivateRoles(ctx, c.Activation, roles, principalID, justification, duration)
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// NewCredential authenticates using DefaultAzureCredential (environment
// variables → managed identity → Azure Developer CLI → Azure CLI →
// Azure PowerShell). Returns the credential for reuse across subscriptions.
func NewCredential() (azcore.TokenCredential, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("not authenticated — run 'az login' (or set AZURE_CLIENT_ID/SECRET/TENANT_ID environment variables): %w", err)
	}
	return cred, nil
}

// NewClients constructs the ARM clients for the given subscription using the
// supplied credential. Call NewCredential() once and pass it here for each
// subscription to avoid re-authenticating.
func NewClients(subscriptionID string, cred azcore.TokenCredential) (*Clients, error) {
	factory, err := armauthorization.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ARM client factory: %w", err)
	}

	return &Clients{
		Eligible:   factory.NewRoleEligibilityScheduleInstancesClient(),
		Active:     factory.NewRoleAssignmentScheduleInstancesClient(),
		Activation: factory.NewRoleAssignmentScheduleRequestsClient(),
	}, nil
}
