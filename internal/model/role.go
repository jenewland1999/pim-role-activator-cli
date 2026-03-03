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

// ActiveRole holds the display data for a currently active PIM role.
type ActiveRole struct {
	RoleName         string
	ScopeName        string
	ScopeType        string
	SubscriptionName string
	Environment      string
	AppCode          string
	ExpiresIn        time.Duration
	Justification    string
}

// CachedActiveRole is the JSON-serialisable representation of an active role
// stored in the file cache. It uses an absolute ExpiresAt timestamp rather than
// a relative ExpiresIn duration so that the remaining time can be recomputed
// accurately when the cache is read back later.
type CachedActiveRole struct {
	RoleName         string    `json:"role_name"`
	ScopeName        string    `json:"scope_name"`
	ScopeType        string    `json:"scope_type"`
	SubscriptionName string    `json:"subscription_name"`
	Environment      string    `json:"environment"`
	AppCode          string    `json:"app_code"`
	ExpiresAt        time.Time `json:"expires_at"`
	Justification    string    `json:"justification"`
}

// ToCached converts an ActiveRole to a CachedActiveRole by computing an
// absolute expiry time from now + ExpiresIn.
func (r ActiveRole) ToCached() CachedActiveRole {
	return CachedActiveRole{
		RoleName:         r.RoleName,
		ScopeName:        r.ScopeName,
		ScopeType:        r.ScopeType,
		SubscriptionName: r.SubscriptionName,
		Environment:      r.Environment,
		AppCode:          r.AppCode,
		ExpiresAt:        time.Now().Add(r.ExpiresIn),
		Justification:    r.Justification,
	}
}

// ToActive converts a CachedActiveRole back to an ActiveRole by recomputing
// ExpiresIn from the absolute ExpiresAt timestamp. If the role has expired
// (ExpiresAt is in the past), ExpiresIn is set to zero.
func (c CachedActiveRole) ToActive() ActiveRole {
	remaining := time.Until(c.ExpiresAt)
	if remaining < 0 {
		remaining = 0
	}
	return ActiveRole{
		RoleName:         c.RoleName,
		ScopeName:        c.ScopeName,
		ScopeType:        c.ScopeType,
		SubscriptionName: c.SubscriptionName,
		Environment:      c.Environment,
		AppCode:          c.AppCode,
		ExpiresIn:        remaining,
		Justification:    c.Justification,
	}
}

// ToCachedRoles converts a slice of ActiveRole to CachedActiveRole.
func ToCachedRoles(roles []ActiveRole) []CachedActiveRole {
	cached := make([]CachedActiveRole, len(roles))
	for i, r := range roles {
		cached[i] = r.ToCached()
	}
	return cached
}

// FromCachedRoles converts cached roles back to active roles, pruning any that
// have already expired (ExpiresAt in the past).
func FromCachedRoles(cached []CachedActiveRole) []ActiveRole {
	now := time.Now()
	var roles []ActiveRole
	for _, c := range cached {
		if c.ExpiresAt.After(now) {
			roles = append(roles, c.ToActive())
		}
	}
	return roles
}

// PruneCachedRoles returns only the cached roles whose ExpiresAt is still in
// the future.
func PruneCachedRoles(cached []CachedActiveRole) []CachedActiveRole {
	now := time.Now()
	var live []CachedActiveRole
	for _, c := range cached {
		if c.ExpiresAt.After(now) {
			live = append(live, c)
		}
	}
	return live
}

// ActiveRolesEqual reports whether two slices of ActiveRole are semantically
// equal by comparing role name, scope name, and subscription name. ExpiresIn
// drift is intentionally ignored.
func ActiveRolesEqual(a, b []ActiveRole) bool {
	if len(a) != len(b) {
		return false
	}
	type key struct{ role, scope, sub string }
	set := make(map[key]struct{}, len(a))
	for _, r := range a {
		set[key{r.RoleName, r.ScopeName, r.SubscriptionName}] = struct{}{}
	}
	for _, r := range b {
		if _, ok := set[key{r.RoleName, r.ScopeName, r.SubscriptionName}]; !ok {
			return false
		}
	}
	return true
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
