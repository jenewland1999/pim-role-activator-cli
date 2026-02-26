// Package model defines all data types shared across packages.
package model

import "time"

// Role represents one eligible PIM role assignment returned by the API.
type Role struct {
	// Display fields (from expandedProperties)
	RoleName         string // e.g. "Contributor"
	ScopeName        string // e.g. "RG-PRD-APP1-001"
	ScopeType        string // "resourcegroup", "subscription", etc.
	SubscriptionName string // Friendly display name of the parent subscription

	// API fields required for the activation PUT request
	RoleDefinitionID string // Full ARM resource ID of the role definition
	Scope            string // Full ARM resource ID of the scope

	// Derived fields decoded from the scope name via the user-configured regexp.
	// Both are empty string when no scope_pattern is configured (columns hidden).
	Environment string // from the "env" named capture group, or "—" on no match
	AppCode     string // from the "app" named capture group, or "—" on no match

	// UI state — set by the interactive selector
	Selected bool
}

// DurationOption maps a human label to its ISO 8601 and time.Duration equivalents.
type DurationOption struct {
	Label    string        // e.g. "1 hour"
	ISO8601  string        // e.g. "PT1H"
	Duration time.Duration // e.g. time.Hour
}

// DurationOptions is the ordered list presented during activation.
var DurationOptions = []DurationOption{
	{Label: "30 minutes", ISO8601: "PT30M", Duration: 30 * time.Minute},
	{Label: "1 hour", ISO8601: "PT1H", Duration: time.Hour},
	{Label: "2 hours", ISO8601: "PT2H", Duration: 2 * time.Hour},
	{Label: "4 hours", ISO8601: "PT4H", Duration: 4 * time.Hour},
}
