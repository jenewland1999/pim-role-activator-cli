package cache

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

func TestLoadActiveRoles_EmptyCache(t *testing.T) {
	dir := t.TempDir()
	roles, ok := LoadActiveRoles(dir, 5*time.Minute)
	if ok {
		t.Errorf("LoadActiveRoles() returned ok=true with empty cache, roles = %v", roles)
	}
	if roles != nil {
		t.Errorf("LoadActiveRoles() roles = %v, want nil", roles)
	}
}

func TestSaveAndLoadActiveRoles_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	roles := []model.ActiveRole{
		{
			RoleName:         "Contributor",
			ScopeName:        "TESTRG01",
			ScopeType:        "resourcegroup",
			SubscriptionName: "Test Sub",
			Environment:      "Dev",
			AppCode:          "TST",
			ExpiresIn:        30 * time.Minute,
			Justification:    "Testing",
		},
		{
			RoleName:         "Reader",
			ScopeName:        "TESTRG02",
			ScopeType:        "resourcegroup",
			SubscriptionName: "Test Sub",
			Environment:      "Prod",
			AppCode:          "APP",
			ExpiresIn:        2 * time.Hour,
			Justification:    "Debugging",
		},
	}

	if err := SaveActiveRoles(dir, 24*time.Hour, roles); err != nil {
		t.Fatalf("SaveActiveRoles() error: %v", err)
	}

	loaded, ok := LoadActiveRoles(dir, 24*time.Hour)
	if !ok {
		t.Fatal("LoadActiveRoles() returned ok=false after save")
	}
	if len(loaded) != 2 {
		t.Fatalf("LoadActiveRoles() returned %d roles, want 2", len(loaded))
	}

	// Check fields are preserved (ExpiresIn will be slightly less due to time passing).
	if loaded[0].RoleName != "Contributor" {
		t.Errorf("loaded[0].RoleName = %q, want %q", loaded[0].RoleName, "Contributor")
	}
	if loaded[0].ScopeName != "TESTRG01" {
		t.Errorf("loaded[0].ScopeName = %q, want %q", loaded[0].ScopeName, "TESTRG01")
	}
	if loaded[0].Justification != "Testing" {
		t.Errorf("loaded[0].Justification = %q, want %q", loaded[0].Justification, "Testing")
	}
	if loaded[1].RoleName != "Reader" {
		t.Errorf("loaded[1].RoleName = %q, want %q", loaded[1].RoleName, "Reader")
	}
}

func TestLoadActiveRoles_PrunesExpired(t *testing.T) {
	dir := t.TempDir()

	// Write a cache with one expired and one still-live role.
	cached := []model.CachedActiveRole{
		{
			RoleName:  "Expired",
			ScopeName: "RG01",
			ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
		},
		{
			RoleName:  "Active",
			ScopeName: "RG02",
			ExpiresAt: time.Now().Add(1 * time.Hour), // still live
		},
	}
	data, _ := json.Marshal(cached)
	c := New(dir, 24*time.Hour, activeRolesPrefix)
	if err := c.Set(data); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	loaded, ok := LoadActiveRoles(dir, 24*time.Hour)
	if !ok {
		t.Fatal("LoadActiveRoles() returned ok=false")
	}
	if len(loaded) != 1 {
		t.Fatalf("LoadActiveRoles() returned %d roles, want 1 (expired pruned)", len(loaded))
	}
	if loaded[0].RoleName != "Active" {
		t.Errorf("loaded[0].RoleName = %q, want %q", loaded[0].RoleName, "Active")
	}
}

func TestLoadActiveRoles_AllExpired(t *testing.T) {
	dir := t.TempDir()

	cached := []model.CachedActiveRole{
		{RoleName: "Old1", ExpiresAt: time.Now().Add(-2 * time.Hour)},
		{RoleName: "Old2", ExpiresAt: time.Now().Add(-1 * time.Hour)},
	}
	data, _ := json.Marshal(cached)
	c := New(dir, 24*time.Hour, activeRolesPrefix)
	if err := c.Set(data); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	roles, ok := LoadActiveRoles(dir, 24*time.Hour)
	if ok {
		t.Errorf("LoadActiveRoles() returned ok=true when all roles expired, roles = %v", roles)
	}
}

func TestLoadActiveRoles_CorruptData(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 24*time.Hour, activeRolesPrefix)
	if err := c.Set([]byte(`not valid json`)); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	roles, ok := LoadActiveRoles(dir, 24*time.Hour)
	if ok {
		t.Errorf("LoadActiveRoles() returned ok=true with corrupt data, roles = %v", roles)
	}
}

func TestSaveActiveRoles_DynamicTTL(t *testing.T) {
	dir := t.TempDir()
	roles := []model.ActiveRole{
		{RoleName: "Short", ExpiresIn: 10 * time.Minute},
		{RoleName: "Long", ExpiresIn: 4 * time.Hour},
	}

	if err := SaveActiveRoles(dir, 24*time.Hour, roles); err != nil {
		t.Fatalf("SaveActiveRoles() error: %v", err)
	}

	// The cache should have been written with the active-roles prefix.
	c := New(dir, 24*time.Hour, activeRolesPrefix)
	age, err := c.Age()
	if err != nil {
		t.Fatalf("Age() error: %v", err)
	}
	if age > 1*time.Second {
		t.Errorf("Age() = %v, expected < 1s (just written)", age)
	}
}

func TestSaveActiveRoles_DoesNotColideWithEligible(t *testing.T) {
	dir := t.TempDir()

	// Write to eligible-roles cache.
	eligible := New(dir, 24*time.Hour, "eligible-roles")
	if err := eligible.Set([]byte(`[{"eligible":"data"}]`)); err != nil {
		t.Fatalf("eligible Set() error: %v", err)
	}

	// Write to active-roles cache.
	roles := []model.ActiveRole{
		{RoleName: "Contributor", ExpiresIn: 1 * time.Hour},
	}
	if err := SaveActiveRoles(dir, 24*time.Hour, roles); err != nil {
		t.Fatalf("SaveActiveRoles() error: %v", err)
	}

	// Verify eligible cache is untouched.
	got, ok := eligible.Get()
	if !ok {
		t.Fatal("eligible.Get() returned ok=false after active-roles save")
	}
	if string(got) != `[{"eligible":"data"}]` {
		t.Errorf("eligible cache corrupted: %q", got)
	}
}
