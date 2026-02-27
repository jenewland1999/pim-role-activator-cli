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
