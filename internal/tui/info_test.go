package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

func TestInfoExpirySeverity(t *testing.T) {
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
			if got := infoExpirySeverity(tt.expiresAt, tt.expiresIn); got != tt.want {
				t.Fatalf("infoExpirySeverity(%v, %v) = %d, want %d", tt.expiresAt, tt.expiresIn, got, tt.want)
			}
		})
	}
}

func TestFormatInfoExpiryTimestamp(t *testing.T) {
	expiresAt := time.Date(2026, time.March, 11, 10, 30, 0, 0, time.UTC)
	got := formatInfoExpiryTimestamp(expiresAt, 48*time.Hour)
	if !strings.Contains(got, "2026-03-11 10:30") {
		t.Fatalf("formatInfoExpiryTimestamp() = %q, want formatted local timestamp", got)
	}
}

func TestFormatInfoExpiryTimestamp_NoExpiry(t *testing.T) {
	if got := formatInfoExpiryTimestamp(time.Time{}, 0); got != "never           " {
		t.Fatalf("formatInfoExpiryTimestamp(no expiry) = %q, want padded never", got)
	}
}

func TestFormatInfoExpiryCell_AppliesColourThresholds(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	urgent := formatInfoExpiryCell("soon", now.Add(3*24*time.Hour), 3*24*time.Hour, 8)
	if !strings.Contains(urgent, "soon") {
		t.Fatalf("urgent cell = %q, expected rendered content", urgent)
	}

	warning := formatInfoExpiryCell("later", now.Add(10*24*time.Hour), 10*24*time.Hour, 8)
	if !strings.Contains(warning, "later") {
		t.Fatalf("warning cell = %q, expected rendered content", warning)
	}

	normal := formatInfoExpiryCell("safe", time.Time{}, 0, 8)
	if normal != "safe    " {
		t.Fatalf("normal cell = %q, want uncoloured padded value", normal)
	}
}

func TestFormatInfoExpiryDuration_FriendlyUnits(t *testing.T) {
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
			if got := formatInfoExpiryDuration(tt.expiresAt, tt.expiresIn); got != tt.want {
				t.Fatalf("formatInfoExpiryDuration() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGroupInfoRoles(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	roles := []model.EligibleRole{
		{Role: model.Role{RoleName: "Soon"}, ExpiresAt: now.Add(3 * 24 * time.Hour), ExpiresIn: 3 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Quarter"}, ExpiresAt: now.Add(30 * 24 * time.Hour), ExpiresIn: 30 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Later"}, ExpiresAt: now.Add(120 * 24 * time.Hour), ExpiresIn: 120 * 24 * time.Hour},
		{Role: model.Role{RoleName: "Never"}},
	}

	groups := groupInfoRoles(roles)
	if len(groups[0].roles) != 1 || groups[0].roles[0].RoleName != "Soon" {
		t.Fatalf("group[0] = %+v, want Soon", groups[0].roles)
	}
	if len(groups[1].roles) != 1 || groups[1].roles[0].RoleName != "Quarter" {
		t.Fatalf("group[1] = %+v, want Quarter", groups[1].roles)
	}
	if len(groups[2].roles) != 2 {
		t.Fatalf("group[2] len = %d, want 2", len(groups[2].roles))
	}
}
