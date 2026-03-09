package azure

import (
	"regexp"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
)

func TestMapEligibleExpiryInstance(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	end := now.Add(5 * 24 * time.Hour)
	re := regexp.MustCompile(`^(?P<env>PRD)-(?P<app>[A-Z0-9]+)$`)
	inst := &armauthorization.RoleEligibilityScheduleInstance{
		Properties: &armauthorization.RoleEligibilityScheduleInstanceProperties{
			Scope:            strPtr("/subscriptions/sub-1/resourceGroups/rg-1"),
			RoleDefinitionID: strPtr("/providers/Microsoft.Authorization/roleDefinitions/reader"),
			EndDateTime:      &end,
			ExpandedProperties: &armauthorization.ExpandedProperties{
				RoleDefinition: &armauthorization.ExpandedPropertiesRoleDefinition{
					DisplayName: strPtr("Reader"),
				},
				Scope: &armauthorization.ExpandedPropertiesScope{
					DisplayName: strPtr("prd-app1"),
					Type:        strPtr("resourcegroup"),
				},
			},
		},
	}

	role := mapEligibleExpiryInstance(inst, "Prod Sub", re, map[string]string{"PRD": "Prod"}, now)
	if role.RoleName != "Reader" {
		t.Fatalf("RoleName = %q, want Reader", role.RoleName)
	}
	if role.SubscriptionName != "Prod Sub" {
		t.Fatalf("SubscriptionName = %q, want Prod Sub", role.SubscriptionName)
	}
	if role.Environment != "Prod" {
		t.Fatalf("Environment = %q, want Prod", role.Environment)
	}
	if role.AppCode != "APP1" {
		t.Fatalf("AppCode = %q, want APP1", role.AppCode)
	}
	if !role.ExpiresAt.Equal(end) {
		t.Fatalf("ExpiresAt = %v, want %v", role.ExpiresAt, end)
	}
	if role.ExpiresIn != 5*24*time.Hour {
		t.Fatalf("ExpiresIn = %v, want %v", role.ExpiresIn, 5*24*time.Hour)
	}
}

func TestMapEligibleExpiryInstance_NoEndDate(t *testing.T) {
	inst := &armauthorization.RoleEligibilityScheduleInstance{
		Properties: &armauthorization.RoleEligibilityScheduleInstanceProperties{
			ExpandedProperties: &armauthorization.ExpandedProperties{
				RoleDefinition: &armauthorization.ExpandedPropertiesRoleDefinition{DisplayName: strPtr("Reader")},
			},
		},
	}

	role := mapEligibleExpiryInstance(inst, "Sub", nil, nil, time.Now())
	if !role.ExpiresAt.IsZero() {
		t.Fatalf("ExpiresAt = %v, want zero", role.ExpiresAt)
	}
	if role.ExpiresIn != 0 {
		t.Fatalf("ExpiresIn = %v, want 0", role.ExpiresIn)
	}
}

func strPtr(value string) *string { return &value }
