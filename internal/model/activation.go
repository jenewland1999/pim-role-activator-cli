package model

import "time"

// ActivationRecord is persisted to ~/.pim/activations.json after a successful
// activation. It is read back by status mode for justification display.
type ActivationRecord struct {
	Scope            string `json:"scope"`
	RoleDefinitionID string `json:"roleDefinitionId"`
	RoleName         string `json:"roleName"`
	ScopeName        string `json:"scopeName"`
	Justification    string `json:"justification"`
	Duration         string `json:"duration"`
	ActivatedAt      string `json:"activatedAt"`
	ExpiresEpoch     int64  `json:"expiresEpoch"`
}

// LookupKey returns the composite key used to match activations to API results.
func (a ActivationRecord) LookupKey() string {
	return a.Scope + "|" + a.RoleDefinitionID
}

// ActiveRole combines live API data with locally stored information for display.
type ActiveRole struct {
	RoleName         string
	ScopeName        string
	ScopeType        string
	SubscriptionName string
	Environment      string
	AppCode          string
	ExpiresIn        time.Duration
	Justification    string // From local state, or "—"
}

// ActivationResult captures the outcome of one activation request.
type ActivationResult struct {
	Role Role
	Err  error
}
