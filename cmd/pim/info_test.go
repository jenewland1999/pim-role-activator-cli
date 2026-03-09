package main

import (
	"testing"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

func TestSortActiveRolesByExpiry(t *testing.T) {
	roles := []model.ActiveRole{
		{RoleName: "Reader", ScopeName: "RG-B", SubscriptionName: "Sub-B", ExpiresIn: 10 * 24 * time.Hour},
		{RoleName: "Owner", ScopeName: "RG-A", SubscriptionName: "Sub-A", ExpiresIn: 7 * 24 * time.Hour},
		{RoleName: "Contributor", ScopeName: "RG-C", SubscriptionName: "Sub-C", ExpiresIn: 2 * 24 * time.Hour},
	}

	sortActiveRolesByExpiry(roles)

	got := []string{roles[0].RoleName, roles[1].RoleName, roles[2].RoleName}
	want := []string{"Contributor", "Owner", "Reader"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("roles[%d] = %q, want %q (full order: %v)", i, got[i], want[i], got)
		}
	}
}

func TestSortActiveRolesByExpiry_UsesStableTieBreakers(t *testing.T) {
	roles := []model.ActiveRole{
		{RoleName: "Reader", ScopeName: "RG-B", SubscriptionName: "Sub-B", ExpiresIn: 24 * time.Hour},
		{RoleName: "Owner", ScopeName: "RG-A", SubscriptionName: "Sub-B", ExpiresIn: 24 * time.Hour},
		{RoleName: "Contributor", ScopeName: "RG-A", SubscriptionName: "Sub-A", ExpiresIn: 24 * time.Hour},
	}

	sortActiveRolesByExpiry(roles)

	got := []string{roles[0].SubscriptionName + "/" + roles[0].ScopeName + "/" + roles[0].RoleName, roles[1].SubscriptionName + "/" + roles[1].ScopeName + "/" + roles[1].RoleName, roles[2].SubscriptionName + "/" + roles[2].ScopeName + "/" + roles[2].RoleName}
	want := []string{"Sub-A/RG-A/Contributor", "Sub-B/RG-A/Owner", "Sub-B/RG-B/Reader"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("roles[%d] = %q, want %q (full order: %v)", i, got[i], want[i], got)
		}
	}
}
