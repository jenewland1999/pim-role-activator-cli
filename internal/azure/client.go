// Package azure wraps the Azure SDK clients used by the PIM CLI.
package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
)

// Clients holds the three ARM authorization service clients needed by the CLI
// for a single Azure subscription.
type Clients struct {
	Eligible   *armauthorization.RoleEligibilityScheduleInstancesClient
	Active     *armauthorization.RoleAssignmentScheduleInstancesClient
	Activation *armauthorization.RoleAssignmentScheduleRequestsClient
}

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
