// Package config handles user configuration stored at ~/.pim/config.json.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

const (
	configFile = "config.json"
)

// Subscription holds the ID and display name for one Azure subscription.
type Subscription struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DurationConfig represents a user-configured activation duration option.
// All three fields are required when present.
type DurationConfig struct {
	Label   string `json:"label"`   // e.g. "8 hours"
	ISO8601 string `json:"iso8601"` // e.g. "PT8H" (sent to Azure API)
	Minutes int    `json:"minutes"` // e.g. 480 (used to compute expiry locally)
}

// UserConfig is the persistent user configuration stored in ~/.pim/config.json.
type UserConfig struct {
	Subscriptions       []Subscription `json:"subscriptions"`
	PrincipalID         string         `json:"principal_id"`
	GroupSelectPatterns []string       `json:"quick_select_patterns"`
	CacheTTLHours       int            `json:"cache_ttl_hours,omitempty"`
	// ScopePattern is an optional Go regexp with named capture groups used to
	// extract "env" and "app" labels from scope (resource-group) names.
	// Example: `^.(?P<env>[PQTD]).{5}(?P<app>.{4})`
	// When empty the App/Env columns are hidden in the role selector.
	ScopePattern string `json:"scope_pattern,omitempty"`
	// EnvLabels maps raw decoded environment values to friendly display labels.
	// For example: {"P": "Prod", "D": "Dev", "Q": "QA"}.
	// When empty, the raw decoded value from the regexp capture group is used.
	EnvLabels map[string]string `json:"env_labels,omitempty"`
	// Durations overrides the default activation duration options (30m/1h/2h/4h).
	// When empty or omitted, the built-in defaults are used.
	Durations []DurationConfig `json:"durations,omitempty"`
}

// ParsedScopePattern compiles and returns the configured ScopePattern regexp,
// or nil when no pattern is set. Returns an error if the pattern is invalid.
func (c *UserConfig) ParsedScopePattern() (*regexp.Regexp, error) {
	if c.ScopePattern == "" {
		return nil, nil
	}
	return regexp.Compile(c.ScopePattern)
}

// CacheTTL returns the configured cache duration, defaulting to 24 hours.
func (c *UserConfig) CacheTTL() time.Duration {
	h := c.CacheTTLHours
	if h <= 0 {
		h = 24
	}
	return time.Duration(h) * time.Hour
}

// DurationOptions returns the list of activation duration choices to present in
// the TUI. When the user has configured custom durations in config.json, those
// are returned; otherwise the built-in defaults from model.DurationOptions are
// used.
func (c *UserConfig) DurationOptions() []model.DurationOption {
	if len(c.Durations) == 0 {
		return model.DurationOptions
	}
	opts := make([]model.DurationOption, len(c.Durations))
	for i, d := range c.Durations {
		opts[i] = model.DurationOption{
			Label:    d.Label,
			ISO8601:  d.ISO8601,
			Duration: time.Duration(d.Minutes) * time.Minute,
		}
	}
	return opts
}

// Scopes returns an ARM subscription scope string for each configured subscription.
func (c *UserConfig) Scopes() []string {
	scopes := make([]string, len(c.Subscriptions))
	for i, s := range c.Subscriptions {
		scopes[i] = "/subscriptions/" + s.ID
	}
	return scopes
}

// Validate checks that the config contains the minimum required fields for
// normal operation. Returns a descriptive error directing the user to run
// 'pim setup' when essential fields are missing.
func (c *UserConfig) Validate() error {
	var errs []error
	if c.PrincipalID == "" {
		errs = append(errs, fmt.Errorf("principal_id is empty"))
	}
	if len(c.Subscriptions) == 0 {
		errs = append(errs, fmt.Errorf("no subscriptions configured"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("config: invalid configuration (%w); run 'pim setup' to reconfigure", errors.Join(errs...))
	}
	return nil
}

// Load reads, deserialises, and validates the config file from dir.
// Returns an error if the file cannot be read, contains invalid JSON, or
// is missing required fields (principal_id, subscriptions).
func Load(dir string) (*UserConfig, error) {
	p := filepath.Join(dir, configFile)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", p, err)
	}
	var cfg UserConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", p, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save serialises cfg to disk inside dir, creating the directory if needed.
func Save(dir string, cfg *UserConfig) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: create directory %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	p := filepath.Join(dir, configFile)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("config: write %s: %w", p, err)
	}
	return nil
}

// Exists reports whether a config file is present in dir.
func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, configFile))
	return err == nil
}
