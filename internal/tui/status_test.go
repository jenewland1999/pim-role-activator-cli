package tui

import (
	"strings"
	"testing"
	"time"
)

func TestExpirySeverity(t *testing.T) {
	tests := []struct {
		name      string
		expiresIn time.Duration
		want      int
	}{
		{name: "default", expiresIn: 30 * 24 * time.Hour, want: 0},
		{name: "warning threshold", expiresIn: 14 * 24 * time.Hour, want: 1},
		{name: "urgent threshold", expiresIn: 7 * 24 * time.Hour, want: 2},
		{name: "expired", expiresIn: 0, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expirySeverity(tt.expiresIn); got != tt.want {
				t.Fatalf("expirySeverity(%v) = %d, want %d", tt.expiresIn, got, tt.want)
			}
		})
	}
}

func TestFormatInfoExpiryTimestamp(t *testing.T) {
	now := time.Date(2026, time.March, 9, 10, 30, 0, 0, time.UTC)
	got := formatInfoExpiryTimestamp(now, 48*time.Hour)
	if !strings.Contains(got, "2026-03-11 10:30") {
		t.Fatalf("formatInfoExpiryTimestamp() = %q, want formatted local timestamp", got)
	}
}

func TestFormatInfoExpiryCell_AppliesColourThresholds(t *testing.T) {
	urgent := formatInfoExpiryCell("soon", 3*24*time.Hour, 8)
	if !strings.Contains(urgent, "soon") {
		t.Fatalf("urgent cell = %q, expected rendered content", urgent)
	}

	warning := formatInfoExpiryCell("later", 10*24*time.Hour, 8)
	if !strings.Contains(warning, "later") {
		t.Fatalf("warning cell = %q, expected rendered content", warning)
	}

	normal := formatInfoExpiryCell("safe", 30*24*time.Hour, 8)
	if normal != "safe    " {
		t.Fatalf("normal cell = %q, want uncoloured padded value", normal)
	}
}
