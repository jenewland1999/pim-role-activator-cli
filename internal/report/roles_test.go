package report

import (
	"testing"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

func TestSortEligibleRolesByExpiry(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	roles := []model.EligibleRole{
		{Role: model.Role{RoleName: "Reader", ScopeName: "RG-B", SubscriptionName: "Sub-B"}, ExpiresAt: now.Add(10 * 24 * time.Hour), ExpiresIn: 10 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Owner", ScopeName: "RG-A", SubscriptionName: "Sub-A"}, ExpiresAt: now.Add(7 * 24 * time.Hour), ExpiresIn: 7 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Contributor", ScopeName: "RG-C", SubscriptionName: "Sub-C"}, ExpiresAt: now.Add(2 * 24 * time.Hour), ExpiresIn: 2 * 24 * time.Hour},
	}

	SortEligibleRolesByExpiry(roles)

	got := []string{roles[0].RoleName, roles[1].RoleName, roles[2].RoleName}
	want := []string{"Contributor", "Owner", "Reader"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("roles[%d] = %q, want %q (full order: %v)", i, got[i], want[i], got)
		}
	}
}

func TestSortEligibleRolesByExpiry_PlacesNoExpiryLast(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	roles := []model.EligibleRole{
		{Role: model.Role{RoleName: "Never", ScopeName: "RG-Z", SubscriptionName: "Sub-Z"}},
		{Role: model.Role{RoleName: "Soon", ScopeName: "RG-A", SubscriptionName: "Sub-A"}, ExpiresAt: now.Add(24 * time.Hour), ExpiresIn: 24 * time.Hour},
	}

	SortEligibleRolesByExpiry(roles)

	if roles[0].RoleName != "Soon" {
		t.Fatalf("roles[0] = %q, want Soon", roles[0].RoleName)
	}
	if roles[1].RoleName != "Never" {
		t.Fatalf("roles[1] = %q, want Never", roles[1].RoleName)
	}
}
