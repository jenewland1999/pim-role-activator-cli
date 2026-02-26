// Package config handles user configuration stored at ~/.pim/config.json.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	// APIVersion is the ARM API version for all PIM endpoints.
	APIVersion = "2020-10-01"
	configFile = "config.json"
)

// Subscription holds the ID and display name for one Azure subscription.
type Subscription struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

// Scopes returns an ARM subscription scope string for each configured subscription.
func (c *UserConfig) Scopes() []string {
	scopes := make([]string, len(c.Subscriptions))
	for i, s := range c.Subscriptions {
		scopes[i] = "/subscriptions/" + s.ID
	}
	return scopes
}

// Load reads and deserialises the config file from dir.
func Load(dir string) (*UserConfig, error) {
	data, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		return nil, err
	}
	var cfg UserConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save serialises cfg to disk inside dir, creating the directory if needed.
func Save(dir string, cfg *UserConfig) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, configFile), data, 0o644)
}

// Exists reports whether a config file is present in dir.
func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, configFile))
	return err == nil
}
