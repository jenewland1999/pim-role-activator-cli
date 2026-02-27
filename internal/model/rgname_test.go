package model

import (
	"regexp"
	"testing"
)

func TestDecodeScopeFields_NilRegexp(t *testing.T) {
	env, app := DecodeScopeFields("RG-PRD-APP1-001", nil, nil)
	if env != "" || app != "" {
		t.Errorf("expected empty strings for nil regexp, got env=%q app=%q", env, app)
	}
}

func TestDecodeScopeFields_NoMatch(t *testing.T) {
	re := regexp.MustCompile(`^(?P<app>[A-Z]{4})(?P<env>[DPQT])`)
	env, app := DecodeScopeFields("ZZZZZ", re, nil)
	if env != "\u2014" || app != "\u2014" {
		t.Errorf("expected dashes for no match, got env=%q app=%q", env, app)
	}
}

func TestDecodeScopeFields_FullMatch(t *testing.T) {
	re := regexp.MustCompile(`^[A-Z]{4}[A-Z]\d(?P<app>[A-Z]{4})(?P<env>[DPQT])`)
	env, app := DecodeScopeFields("XPABCDEAPP1", re, nil)
	if env != "D" || app != "ECOF" {
		t.Errorf("expected env=D app=ECOF, got env=%q app=%q", env, app)
	}
}

func TestDecodeScopeFields_EnvLabels(t *testing.T) {
	re := regexp.MustCompile(`^[A-Z]{4}[A-Z]\d(?P<app>[A-Z]{4})(?P<env>[DPQT])`)
	labels := map[string]string{"D": "Dev", "P": "Prod", "Q": "QA", "T": "Test"}
	env, app := DecodeScopeFields("XPABCDEAPP1", re, labels)
	if env != "Dev" || app != "ECOF" {
		t.Errorf("expected env=Dev app=ECOF, got env=%q app=%q", env, app)
	}
}

func TestDecodeScopeFields_EnvLabelNotInMap(t *testing.T) {
	re := regexp.MustCompile(`^[A-Z]{4}[A-Z]\d(?P<app>[A-Z]{4})(?P<env>[A-Z])`)
	labels := map[string]string{"D": "Dev"}
	env, app := DecodeScopeFields("XPABCDEAPPX", re, labels)
	if env != "X" || app != "ECOF" {
		t.Errorf("expected env=X app=ECOF, got env=%q app=%q", env, app)
	}
}

func TestDecodeScopeFields_MissingEnvGroup(t *testing.T) {
	re := regexp.MustCompile(`^(?P<app>[A-Z]{4})`)
	env, app := DecodeScopeFields("ECOF", re, nil)
	if env != "\u2014" {
		t.Errorf("expected env=\u2014 for missing env group, got env=%q", env)
	}
	if app != "ECOF" {
		t.Errorf("expected app=ECOF, got app=%q", app)
	}
}

func TestDecodeScopeFields_MissingAppGroup(t *testing.T) {
	re := regexp.MustCompile(`^(?P<env>[DPQT])`)
	env, app := DecodeScopeFields("D", re, nil)
	if env != "D" {
		t.Errorf("expected env=D, got env=%q", env)
	}
	if app != "\u2014" {
		t.Errorf("expected app=\u2014 for missing app group, got app=%q", app)
	}
}

func TestNamedGroup_Exists(t *testing.T) {
	re := regexp.MustCompile(`(?P<name>\w+)`)
	match := re.FindStringSubmatch("hello")
	got := namedGroup(re, match, "name")
	if got != "hello" {
		t.Errorf("expected hello, got %q", got)
	}
}

func TestNamedGroup_Missing(t *testing.T) {
	re := regexp.MustCompile(`(?P<name>\w+)`)
	match := re.FindStringSubmatch("hello")
	got := namedGroup(re, match, "nonexistent")
	if got != "" {
		t.Errorf("expected empty string for missing group, got %q", got)
	}
}
