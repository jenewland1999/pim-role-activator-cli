package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

// --- Load ---

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := &UserConfig{
		PrincipalID: "abc-123",
		Subscriptions: []Subscription{
			{ID: "sub-1", Name: "Dev"},
			{ID: "sub-2", Name: "Prod"},
		},
		CacheTTLHours:      12,
		ScopePattern:       `^.(?P<env>[PQTD])`,
		GroupSelectPatterns: []string{"rg-*"},
		EnvLabels:          map[string]string{"P": "Prod", "D": "Dev"},
	}
	writeConfig(t, dir, cfg)

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got.PrincipalID != cfg.PrincipalID {
		t.Errorf("PrincipalID = %q, want %q", got.PrincipalID, cfg.PrincipalID)
	}
	if len(got.Subscriptions) != 2 {
		t.Fatalf("Subscriptions length = %d, want 2", len(got.Subscriptions))
	}
	if got.Subscriptions[0].ID != "sub-1" {
		t.Errorf("Subscriptions[0].ID = %q, want %q", got.Subscriptions[0].ID, "sub-1")
	}
	if got.Subscriptions[1].Name != "Prod" {
		t.Errorf("Subscriptions[1].Name = %q, want %q", got.Subscriptions[1].Name, "Prod")
	}
	if got.CacheTTLHours != 12 {
		t.Errorf("CacheTTLHours = %d, want 12", got.CacheTTLHours)
	}
	if got.ScopePattern != cfg.ScopePattern {
		t.Errorf("ScopePattern = %q, want %q", got.ScopePattern, cfg.ScopePattern)
	}
	if len(got.EnvLabels) != 2 {
		t.Fatalf("EnvLabels length = %d, want 2", len(got.EnvLabels))
	}
	if got.EnvLabels["P"] != "Prod" {
		t.Errorf("EnvLabels[P] = %q, want %q", got.EnvLabels["P"], "Prod")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()

	_, err := Load(dir)
	if err == nil {
		t.Error("Load() with missing file returned nil error, want error")
	}
}

func TestLoad_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(`{not json`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Error("Load() with corrupt JSON returned nil error, want error")
	}
}

func TestLoad_EmptyObject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load() with empty object returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), "principal_id is empty") {
		t.Errorf("error = %q, want mention of principal_id", err)
	}
	if !strings.Contains(err.Error(), "no subscriptions configured") {
		t.Errorf("error = %q, want mention of subscriptions", err)
	}
	if !strings.Contains(err.Error(), "pim setup") {
		t.Errorf("error = %q, want mention of 'pim setup'", err)
	}
}

// --- Save ---

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &UserConfig{
		PrincipalID: "principal-xyz",
		Subscriptions: []Subscription{
			{ID: "sub-a", Name: "Alpha"},
		},
		CacheTTLHours: 48,
		ScopePattern:  `^test`,
		EnvLabels:     map[string]string{"T": "Test"},
	}

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}
	if got.PrincipalID != cfg.PrincipalID {
		t.Errorf("PrincipalID = %q, want %q", got.PrincipalID, cfg.PrincipalID)
	}
	if len(got.Subscriptions) != 1 || got.Subscriptions[0].ID != "sub-a" {
		t.Errorf("Subscriptions = %+v, want [{ID:sub-a Name:Alpha}]", got.Subscriptions)
	}
	if got.CacheTTLHours != 48 {
		t.Errorf("CacheTTLHours = %d, want 48", got.CacheTTLHours)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	cfg := &UserConfig{PrincipalID: "test"}

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if !Exists(dir) {
		t.Error("Exists() = false after Save(), want true")
	}
}

func TestSave_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	cfg := &UserConfig{PrincipalID: "perm-test"}

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, configFile))
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}
}

func TestSave_DirectoryPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "newdir")
	cfg := &UserConfig{PrincipalID: "dir-perm-test"}

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o700 {
		t.Errorf("directory permissions = %o, want 0700", perm)
	}
}

func TestSave_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()

	cfg1 := &UserConfig{PrincipalID: "first", Subscriptions: []Subscription{{ID: "s1", Name: "S1"}}}
	if err := Save(dir, cfg1); err != nil {
		t.Fatalf("Save(first) error: %v", err)
	}

	cfg2 := &UserConfig{PrincipalID: "second", Subscriptions: []Subscription{{ID: "s2", Name: "S2"}}}
	if err := Save(dir, cfg2); err != nil {
		t.Fatalf("Save(second) error: %v", err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got.PrincipalID != "second" {
		t.Errorf("PrincipalID = %q, want %q", got.PrincipalID, "second")
	}
}

func TestSave_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	cfg := &UserConfig{
		PrincipalID:   "json-test",
		Subscriptions: []Subscription{{ID: "s1", Name: "Sub One"}},
	}

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Errorf("saved file is not valid JSON: %v", err)
	}
}

// --- Exists ---

func TestExists_True(t *testing.T) {
	dir := t.TempDir()
	cfg := &UserConfig{PrincipalID: "exists-test"}
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if !Exists(dir) {
		t.Error("Exists() = false, want true")
	}
}

func TestExists_False(t *testing.T) {
	dir := t.TempDir()

	if Exists(dir) {
		t.Error("Exists() = true with no config file, want false")
	}
}

func TestExists_NonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")

	if Exists(dir) {
		t.Error("Exists() = true with non-existent dir, want false")
	}
}

// --- CacheTTL ---

func TestCacheTTL_CustomValue(t *testing.T) {
	cfg := &UserConfig{CacheTTLHours: 12}
	got := cfg.CacheTTL()
	want := 12 * time.Hour
	if got != want {
		t.Errorf("CacheTTL() = %v, want %v", got, want)
	}
}

func TestCacheTTL_DefaultWhenZero(t *testing.T) {
	cfg := &UserConfig{CacheTTLHours: 0}
	got := cfg.CacheTTL()
	want := 24 * time.Hour
	if got != want {
		t.Errorf("CacheTTL() = %v, want %v (default)", got, want)
	}
}

func TestCacheTTL_DefaultWhenNegative(t *testing.T) {
	cfg := &UserConfig{CacheTTLHours: -5}
	got := cfg.CacheTTL()
	want := 24 * time.Hour
	if got != want {
		t.Errorf("CacheTTL() = %v, want %v (default)", got, want)
	}
}

// --- ParsedScopePattern ---

func TestParsedScopePattern_EmptyPattern(t *testing.T) {
	cfg := &UserConfig{ScopePattern: ""}
	re, err := cfg.ParsedScopePattern()
	if err != nil {
		t.Fatalf("ParsedScopePattern() error: %v", err)
	}
	if re != nil {
		t.Errorf("ParsedScopePattern() = %v, want nil for empty pattern", re)
	}
}

func TestParsedScopePattern_ValidPattern(t *testing.T) {
	cfg := &UserConfig{ScopePattern: `^.(?P<env>[PQTD]).{5}(?P<app>.{4})`}
	re, err := cfg.ParsedScopePattern()
	if err != nil {
		t.Fatalf("ParsedScopePattern() error: %v", err)
	}
	if re == nil {
		t.Fatal("ParsedScopePattern() = nil, want non-nil regexp")
	}

	names := re.SubexpNames()
	found := map[string]bool{}
	for _, n := range names {
		if n != "" {
			found[n] = true
		}
	}
	if !found["env"] {
		t.Error("compiled regexp missing named group 'env'")
	}
	if !found["app"] {
		t.Error("compiled regexp missing named group 'app'")
	}
}

func TestParsedScopePattern_InvalidPattern(t *testing.T) {
	cfg := &UserConfig{ScopePattern: `[invalid`}
	re, err := cfg.ParsedScopePattern()
	if err == nil {
		t.Errorf("ParsedScopePattern() returned nil error for invalid regex, got re = %v", re)
	}
}

func TestParsedScopePattern_SimpleMatch(t *testing.T) {
	cfg := &UserConfig{ScopePattern: `^rg-(?P<env>\w+)-(?P<app>\w+)$`}
	re, err := cfg.ParsedScopePattern()
	if err != nil {
		t.Fatalf("ParsedScopePattern() error: %v", err)
	}

	match := re.FindStringSubmatch("rg-prod-myapp")
	if match == nil {
		t.Fatal("expected pattern to match 'rg-prod-myapp'")
	}

	envIdx := re.SubexpIndex("env")
	appIdx := re.SubexpIndex("app")
	if match[envIdx] != "prod" {
		t.Errorf("env capture = %q, want %q", match[envIdx], "prod")
	}
	if match[appIdx] != "myapp" {
		t.Errorf("app capture = %q, want %q", match[appIdx], "myapp")
	}
}

// --- Scopes ---

func TestScopes_MultipleSubscriptions(t *testing.T) {
	cfg := &UserConfig{
		Subscriptions: []Subscription{
			{ID: "aaaa-bbbb", Name: "Dev"},
			{ID: "cccc-dddd", Name: "Prod"},
		},
	}
	got := cfg.Scopes()
	if len(got) != 2 {
		t.Fatalf("Scopes() length = %d, want 2", len(got))
	}
	want0 := "/subscriptions/aaaa-bbbb"
	want1 := "/subscriptions/cccc-dddd"
	if got[0] != want0 {
		t.Errorf("Scopes()[0] = %q, want %q", got[0], want0)
	}
	if got[1] != want1 {
		t.Errorf("Scopes()[1] = %q, want %q", got[1], want1)
	}
}

func TestScopes_Empty(t *testing.T) {
	cfg := &UserConfig{}
	got := cfg.Scopes()
	if len(got) != 0 {
		t.Errorf("Scopes() length = %d, want 0", len(got))
	}
}

func TestScopes_SingleSubscription(t *testing.T) {
	cfg := &UserConfig{
		Subscriptions: []Subscription{
			{ID: "one-sub", Name: "Only"},
		},
	}
	got := cfg.Scopes()
	if len(got) != 1 {
		t.Fatalf("Scopes() length = %d, want 1", len(got))
	}
	want := "/subscriptions/one-sub"
	if got[0] != want {
		t.Errorf("Scopes()[0] = %q, want %q", got[0], want)
	}
}

// --- Validate ---

func TestValidate_Valid(t *testing.T) {
	cfg := &UserConfig{
		PrincipalID:   "abc-123",
		Subscriptions: []Subscription{{ID: "sub-1", Name: "Dev"}},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidate_MissingPrincipalID(t *testing.T) {
	cfg := &UserConfig{
		Subscriptions: []Subscription{{ID: "sub-1", Name: "Dev"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want error for missing principal_id")
	}
	if !strings.Contains(err.Error(), "principal_id is empty") {
		t.Errorf("error = %q, want mention of principal_id", err)
	}
	if !strings.Contains(err.Error(), "pim setup") {
		t.Errorf("error = %q, want mention of 'pim setup'", err)
	}
}

func TestValidate_MissingSubscriptions(t *testing.T) {
	cfg := &UserConfig{
		PrincipalID: "abc-123",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want error for missing subscriptions")
	}
	if !strings.Contains(err.Error(), "no subscriptions configured") {
		t.Errorf("error = %q, want mention of subscriptions", err)
	}
}

func TestValidate_BothMissing(t *testing.T) {
	cfg := &UserConfig{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want error for empty config")
	}
	if !strings.Contains(err.Error(), "principal_id is empty") {
		t.Errorf("error = %q, want mention of principal_id", err)
	}
	if !strings.Contains(err.Error(), "no subscriptions configured") {
		t.Errorf("error = %q, want mention of subscriptions", err)
	}
}

func TestValidate_EmptySubscriptionsSlice(t *testing.T) {
	cfg := &UserConfig{
		PrincipalID:   "abc-123",
		Subscriptions: []Subscription{},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want error for empty subscriptions slice")
	}
	if !strings.Contains(err.Error(), "no subscriptions configured") {
		t.Errorf("error = %q, want mention of subscriptions", err)
	}
}

// --- DurationOptions ---

func TestDurationOptions_DefaultWhenEmpty(t *testing.T) {
	cfg := &UserConfig{PrincipalID: "x", Subscriptions: []Subscription{{ID: "s", Name: "n"}}}
	got := cfg.DurationOptions()
	if len(got) != len(model.DurationOptions) {
		t.Fatalf("len = %d, want %d (built-in defaults)", len(got), len(model.DurationOptions))
	}
	for i, opt := range got {
		if opt != model.DurationOptions[i] {
			t.Errorf("[%d] = %+v, want %+v", i, opt, model.DurationOptions[i])
		}
	}
}

func TestDurationOptions_CustomOverride(t *testing.T) {
	cfg := &UserConfig{
		PrincipalID:   "x",
		Subscriptions: []Subscription{{ID: "s", Name: "n"}},
		Durations: []DurationConfig{
			{Label: "15 minutes", ISO8601: "PT15M", Minutes: 15},
			{Label: "8 hours", ISO8601: "PT8H", Minutes: 480},
		},
	}
	got := cfg.DurationOptions()
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Label != "15 minutes" || got[0].ISO8601 != "PT15M" || got[0].Duration != 15*time.Minute {
		t.Errorf("[0] = %+v", got[0])
	}
	if got[1].Label != "8 hours" || got[1].ISO8601 != "PT8H" || got[1].Duration != 480*time.Minute {
		t.Errorf("[1] = %+v", got[1])
	}
}

func TestDurationOptions_RoundTripThroughJSON(t *testing.T) {
	dir := t.TempDir()
	cfg := &UserConfig{
		PrincipalID:   "abc",
		Subscriptions: []Subscription{{ID: "sub-1", Name: "Dev"}},
		Durations: []DurationConfig{
			{Label: "1 hour", ISO8601: "PT1H", Minutes: 60},
			{Label: "24 hours", ISO8601: "PT24H", Minutes: 1440},
		},
	}
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	opts := loaded.DurationOptions()
	if len(opts) != 2 {
		t.Fatalf("len = %d, want 2", len(opts))
	}
	if opts[0].Label != "1 hour" || opts[0].Duration != time.Hour {
		t.Errorf("[0] = %+v", opts[0])
	}
	if opts[1].Label != "24 hours" || opts[1].Duration != 24*time.Hour {
		t.Errorf("[1] = %+v", opts[1])
	}
}

func TestDurationOptions_NilDurationsFieldUsesDefaults(t *testing.T) {
	cfg := &UserConfig{
		PrincipalID:   "x",
		Subscriptions: []Subscription{{ID: "s", Name: "n"}},
		Durations:     nil,
	}
	got := cfg.DurationOptions()
	if len(got) != len(model.DurationOptions) {
		t.Fatalf("len = %d, want defaults (%d)", len(got), len(model.DurationOptions))
	}
}

// --- helpers ---

func writeConfig(t *testing.T, dir string, cfg *UserConfig) {
	t.Helper()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, configFile), data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
