package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

func TestEligibleExpirySeverity(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	tests := []struct {
		name      string
		expiresAt time.Time
		expiresIn time.Duration
		want      int
	}{
		{name: "no expiry", expiresAt: time.Time{}, expiresIn: 0, want: 0},
		{name: "default", expiresAt: now.Add(30 * 24 * time.Hour), expiresIn: 30 * 24 * time.Hour, want: 0},
		{name: "warning threshold", expiresAt: now.Add(14 * 24 * time.Hour), expiresIn: 14 * 24 * time.Hour, want: 1},
		{name: "urgent threshold", expiresAt: now.Add(7 * 24 * time.Hour), expiresIn: 7 * 24 * time.Hour, want: 2},
		{name: "expired", expiresAt: now.Add(-1 * time.Hour), expiresIn: 0, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eligibleExpirySeverity(tt.expiresAt, tt.expiresIn); got != tt.want {
				t.Fatalf("eligibleExpirySeverity(%v, %v) = %d, want %d", tt.expiresAt, tt.expiresIn, got, tt.want)
			}
		})
	}
}

func TestFormatEligibleExpiryTimestamp(t *testing.T) {
	expiresAt := time.Date(2026, time.March, 11, 10, 30, 0, 0, time.UTC)
	want := expiresAt.Local().Format("2006-01-02 15:04")
	got := formatEligibleExpiryTimestamp(expiresAt, 48*time.Hour)
	if !strings.Contains(got, want) {
		t.Fatalf("formatEligibleExpiryTimestamp() = %q, want formatted local timestamp containing %q", got, want)
	}
}

func TestFormatEligibleExpiryTimestamp_NoExpiry(t *testing.T) {
	if got := formatEligibleExpiryTimestamp(time.Time{}, 0); got != "never           " {
		t.Fatalf("formatEligibleExpiryTimestamp(no expiry) = %q, want padded never", got)
	}
}

func TestFormatEligibleExpiryCell_AppliesColourThresholds(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	urgent := formatEligibleExpiryCell("soon", now.Add(3*24*time.Hour), 3*24*time.Hour, 8)
	if want := Red("soon    "); urgent != want {
		t.Fatalf("urgent cell = %q, want %q", urgent, want)
	}

	warning := formatEligibleExpiryCell("later", now.Add(10*24*time.Hour), 10*24*time.Hour, 8)
	if want := Orange("later   "); warning != want {
		t.Fatalf("warning cell = %q, want %q", warning, want)
	}

	normal := formatEligibleExpiryCell("safe", time.Time{}, 0, 8)
	if normal != "safe    " {
		t.Fatalf("normal cell = %q, want uncoloured padded value", normal)
	}
}

func TestFormatEligibleExpiryDuration_FriendlyUnits(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	tests := []struct {
		name      string
		expiresAt time.Time
		expiresIn time.Duration
		want      string
	}{
		{name: "hours and minutes", expiresAt: now.Add(3*time.Hour + 15*time.Minute), expiresIn: 3*time.Hour + 15*time.Minute, want: "3h 15m"},
		{name: "days and hours", expiresAt: now.Add(2*24*time.Hour + 5*time.Hour), expiresIn: 2*24*time.Hour + 5*time.Hour, want: "2d 5h"},
		{name: "weeks and days", expiresAt: now.Add(5*7*24*time.Hour + 2*24*time.Hour), expiresIn: 5*7*24*time.Hour + 2*24*time.Hour, want: "5w 2d"},
		{name: "months and weeks", expiresAt: now.Add(4*30*24*time.Hour + 2*7*24*time.Hour), expiresIn: 4*30*24*time.Hour + 2*7*24*time.Hour, want: "4mo 2w"},
		{name: "no expiry", expiresAt: time.Time{}, expiresIn: 0, want: "never"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatEligibleExpiryDuration(tt.expiresAt, tt.expiresIn); got != tt.want {
				t.Fatalf("formatEligibleExpiryDuration() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGroupEligibleRoles(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	roles := []model.EligibleRole{
		{Role: model.Role{RoleName: "Soon"}, ExpiresAt: now.Add(3 * 24 * time.Hour), ExpiresIn: 3 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Quarter"}, ExpiresAt: now.Add(30 * 24 * time.Hour), ExpiresIn: 30 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Later"}, ExpiresAt: now.Add(120 * 24 * time.Hour), ExpiresIn: 120 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Never"}},
		{Role: model.Role{RoleName: "Expired"}, ExpiresAt: now.Add(-24 * time.Hour), ExpiresIn: 0},
	}

	groups := groupEligibleRoles(roles)
	if groups[2].title != "Expires After 1 Quarter or Never" {
		t.Fatalf("group[2].title = %q, want updated non-expiring label", groups[2].title)
	}
	if len(groups[0].roles) != 1 || groups[0].roles[0].RoleName != "Soon" {
		t.Fatalf("group[0] = %+v, want Soon", groups[0].roles)
	}
	if len(groups[1].roles) != 1 || groups[1].roles[0].RoleName != "Quarter" {
		t.Fatalf("group[1] = %+v, want Quarter", groups[1].roles)
	}
	if len(groups[2].roles) != 2 {
		t.Fatalf("group[2] len = %d, want 2", len(groups[2].roles))
	}
	seen := make(map[string]bool, len(groups[2].roles))
	for _, role := range groups[2].roles {
		seen[role.RoleName] = true
	}
	if !seen["Later"] || !seen["Never"] {
		t.Fatalf("group[2] = %+v, want Later and Never", groups[2].roles)
	}
	for groupIndex, group := range groups {
		for _, role := range group.roles {
			if role.RoleName == "Expired" {
				t.Fatalf("group[%d] unexpectedly contained expired role: %+v", groupIndex, group.roles)
			}
		}
	}
	if countEligibleRoles(groups) != 4 {
		t.Fatalf("countEligibleRoles(groups) = %d, want 4", countEligibleRoles(groups))
	}
}

func TestGroupEligibleRoles_SortsByExpiryStable(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	equalExpiry := now.Add(5 * 24 * time.Hour)
	roles := []model.EligibleRole{
		{Role: model.Role{RoleName: "Later"}, ExpiresAt: now.Add(120 * 24 * time.Hour), ExpiresIn: 120 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Soon-B"}, ExpiresAt: equalExpiry, ExpiresIn: 5 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Never"}},
		{Role: model.Role{RoleName: "Soon-A"}, ExpiresAt: equalExpiry, ExpiresIn: 5 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Quarter"}, ExpiresAt: now.Add(30 * 24 * time.Hour), ExpiresIn: 30 * 24 * time.Hour},
	}

	groups := groupEligibleRoles(roles)
	if got := groups[0].roles[0].RoleName; got != "Soon-B" {
		t.Fatalf("group[0].roles[0] = %q, want stable ordering for equal expiries", got)
	}
	if got := groups[0].roles[1].RoleName; got != "Soon-A" {
		t.Fatalf("group[0].roles[1] = %q, want stable ordering for equal expiries", got)
	}
	if got := groups[2].roles[0].RoleName; got != "Later" {
		t.Fatalf("group[2].roles[0] = %q, want dated expiry before never", got)
	}
	if got := groups[2].roles[1].RoleName; got != "Never" {
		t.Fatalf("group[2].roles[1] = %q, want never to sort last", got)
	}
	if got := groups[1].roles[0].RoleName; got != "Quarter" {
		t.Fatalf("group[1].roles[0] = %q, want Quarter", got)
	}
	if got := roles[0].RoleName; got != "Later" {
		t.Fatalf("input slice mutated, roles[0] = %q", got)
	}
}
