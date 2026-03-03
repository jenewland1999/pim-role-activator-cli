package model

import (
	"testing"
	"time"
)

func TestDurationOptions_Order(t *testing.T) {
	if len(DurationOptions) != 4 {
		t.Fatalf("expected 4 duration options, got %d", len(DurationOptions))
	}

	expectedLabels := []string{"30 minutes", "1 hour", "2 hours", "4 hours"}
	for i, opt := range DurationOptions {
		if opt.Label != expectedLabels[i] {
			t.Errorf("DurationOptions[%d].Label = %q, want %q", i, opt.Label, expectedLabels[i])
		}
	}
}

func TestDurationOptions_ISO8601Values(t *testing.T) {
	expectedISO := []string{"PT30M", "PT1H", "PT2H", "PT4H"}
	for i, opt := range DurationOptions {
		if opt.ISO8601 != expectedISO[i] {
			t.Errorf("DurationOptions[%d].ISO8601 = %q, want %q", i, opt.ISO8601, expectedISO[i])
		}
	}
}

func TestDurationOptions_Durations(t *testing.T) {
	expectedDurations := []time.Duration{
		30 * time.Minute,
		time.Hour,
		2 * time.Hour,
		4 * time.Hour,
	}
	for i, opt := range DurationOptions {
		if opt.Duration != expectedDurations[i] {
			t.Errorf("DurationOptions[%d].Duration = %v, want %v", i, opt.Duration, expectedDurations[i])
		}
	}
}

func TestDurationOptions_Ascending(t *testing.T) {
	for i := 1; i < len(DurationOptions); i++ {
		if DurationOptions[i].Duration <= DurationOptions[i-1].Duration {
			t.Errorf("DurationOptions not ascending at index %d: %v <= %v",
				i, DurationOptions[i].Duration, DurationOptions[i-1].Duration)
		}
	}
}

// ── CachedActiveRole tests ───────────────────────────────────────────────────

func TestActiveRole_ToCached(t *testing.T) {
	r := ActiveRole{
		RoleName:         "Contributor",
		ScopeName:        "TESTRG01",
		ScopeType:        "resourcegroup",
		SubscriptionName: "Sub1",
		Environment:      "Dev",
		AppCode:          "APP",
		ExpiresIn:        1 * time.Hour,
		Justification:    "Testing",
	}

	cached := r.ToCached()
	if cached.RoleName != "Contributor" {
		t.Errorf("RoleName = %q, want %q", cached.RoleName, "Contributor")
	}
	if cached.ScopeName != "TESTRG01" {
		t.Errorf("ScopeName = %q, want %q", cached.ScopeName, "TESTRG01")
	}
	if cached.Justification != "Testing" {
		t.Errorf("Justification = %q, want %q", cached.Justification, "Testing")
	}
	// ExpiresAt should be roughly 1 hour from now.
	remaining := time.Until(cached.ExpiresAt)
	if remaining < 59*time.Minute || remaining > 61*time.Minute {
		t.Errorf("ExpiresAt remaining = %v, expected ~1h", remaining)
	}
}

func TestCachedActiveRole_ToActive(t *testing.T) {
	cached := CachedActiveRole{
		RoleName:         "Reader",
		ScopeName:        "RG02",
		ScopeType:        "subscription",
		SubscriptionName: "Sub2",
		ExpiresAt:        time.Now().Add(30 * time.Minute),
		Justification:    "Debug",
	}

	active := cached.ToActive()
	if active.RoleName != "Reader" {
		t.Errorf("RoleName = %q, want %q", active.RoleName, "Reader")
	}
	if active.ExpiresIn < 29*time.Minute || active.ExpiresIn > 31*time.Minute {
		t.Errorf("ExpiresIn = %v, expected ~30m", active.ExpiresIn)
	}
}

func TestCachedActiveRole_ToActive_Expired(t *testing.T) {
	cached := CachedActiveRole{
		RoleName:  "Old",
		ExpiresAt: time.Now().Add(-10 * time.Minute),
	}

	active := cached.ToActive()
	if active.ExpiresIn != 0 {
		t.Errorf("ExpiresIn = %v, want 0 for expired role", active.ExpiresIn)
	}
}

func TestToCachedRoles(t *testing.T) {
	roles := []ActiveRole{
		{RoleName: "A", ExpiresIn: 1 * time.Hour},
		{RoleName: "B", ExpiresIn: 2 * time.Hour},
	}
	cached := ToCachedRoles(roles)
	if len(cached) != 2 {
		t.Fatalf("len(cached) = %d, want 2", len(cached))
	}
	if cached[0].RoleName != "A" || cached[1].RoleName != "B" {
		t.Errorf("cached names = [%q, %q], want [A, B]", cached[0].RoleName, cached[1].RoleName)
	}
}

func TestFromCachedRoles_PrunesExpired(t *testing.T) {
	cached := []CachedActiveRole{
		{RoleName: "Alive", ExpiresAt: time.Now().Add(1 * time.Hour)},
		{RoleName: "Dead", ExpiresAt: time.Now().Add(-1 * time.Hour)},
		{RoleName: "AlsoAlive", ExpiresAt: time.Now().Add(30 * time.Minute)},
	}
	active := FromCachedRoles(cached)
	if len(active) != 2 {
		t.Fatalf("len(active) = %d, want 2 (expired role pruned)", len(active))
	}
	if active[0].RoleName != "Alive" {
		t.Errorf("active[0].RoleName = %q, want %q", active[0].RoleName, "Alive")
	}
	if active[1].RoleName != "AlsoAlive" {
		t.Errorf("active[1].RoleName = %q, want %q", active[1].RoleName, "AlsoAlive")
	}
}

func TestPruneCachedRoles(t *testing.T) {
	cached := []CachedActiveRole{
		{RoleName: "Live", ExpiresAt: time.Now().Add(1 * time.Hour)},
		{RoleName: "Expired", ExpiresAt: time.Now().Add(-5 * time.Minute)},
	}
	live := PruneCachedRoles(cached)
	if len(live) != 1 {
		t.Fatalf("len(live) = %d, want 1", len(live))
	}
	if live[0].RoleName != "Live" {
		t.Errorf("live[0].RoleName = %q, want %q", live[0].RoleName, "Live")
	}
}

func TestActiveRolesEqual_SameRoles(t *testing.T) {
	a := []ActiveRole{
		{RoleName: "Contributor", ScopeName: "RG1", SubscriptionName: "Sub1"},
		{RoleName: "Reader", ScopeName: "RG2", SubscriptionName: "Sub1"},
	}
	b := []ActiveRole{
		{RoleName: "Contributor", ScopeName: "RG1", SubscriptionName: "Sub1"},
		{RoleName: "Reader", ScopeName: "RG2", SubscriptionName: "Sub1"},
	}
	if !ActiveRolesEqual(a, b) {
		t.Error("ActiveRolesEqual() = false, want true for identical slices")
	}
}

func TestActiveRolesEqual_DifferentOrder(t *testing.T) {
	a := []ActiveRole{
		{RoleName: "Reader", ScopeName: "RG2", SubscriptionName: "Sub1"},
		{RoleName: "Contributor", ScopeName: "RG1", SubscriptionName: "Sub1"},
	}
	b := []ActiveRole{
		{RoleName: "Contributor", ScopeName: "RG1", SubscriptionName: "Sub1"},
		{RoleName: "Reader", ScopeName: "RG2", SubscriptionName: "Sub1"},
	}
	if !ActiveRolesEqual(a, b) {
		t.Error("ActiveRolesEqual() = false, want true for same roles in different order")
	}
}

func TestActiveRolesEqual_DifferentLength(t *testing.T) {
	a := []ActiveRole{
		{RoleName: "Contributor", ScopeName: "RG1", SubscriptionName: "Sub1"},
	}
	b := []ActiveRole{
		{RoleName: "Contributor", ScopeName: "RG1", SubscriptionName: "Sub1"},
		{RoleName: "Reader", ScopeName: "RG2", SubscriptionName: "Sub1"},
	}
	if ActiveRolesEqual(a, b) {
		t.Error("ActiveRolesEqual() = true, want false for different lengths")
	}
}

func TestActiveRolesEqual_DifferentRoles(t *testing.T) {
	a := []ActiveRole{
		{RoleName: "Contributor", ScopeName: "RG1", SubscriptionName: "Sub1"},
	}
	b := []ActiveRole{
		{RoleName: "Owner", ScopeName: "RG1", SubscriptionName: "Sub1"},
	}
	if ActiveRolesEqual(a, b) {
		t.Error("ActiveRolesEqual() = true, want false for different role names")
	}
}

func TestActiveRolesEqual_BothEmpty(t *testing.T) {
	if !ActiveRolesEqual(nil, nil) {
		t.Error("ActiveRolesEqual(nil, nil) = false, want true")
	}
	if !ActiveRolesEqual([]ActiveRole{}, []ActiveRole{}) {
		t.Error("ActiveRolesEqual([], []) = false, want true")
	}
}

func TestActiveRolesEqual_IgnoresExpiresIn(t *testing.T) {
	a := []ActiveRole{
		{RoleName: "Contributor", ScopeName: "RG1", SubscriptionName: "Sub1", ExpiresIn: 1 * time.Hour},
	}
	b := []ActiveRole{
		{RoleName: "Contributor", ScopeName: "RG1", SubscriptionName: "Sub1", ExpiresIn: 30 * time.Minute},
	}
	if !ActiveRolesEqual(a, b) {
		t.Error("ActiveRolesEqual() = false, want true (ExpiresIn drift should be ignored)")
	}
}
