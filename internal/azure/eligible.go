package azure

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

// safeStr dereferences a *string; returns "" when nil.
func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// FetchEligibleRoles retrieves all PIM-eligible role assignments for the current
// user (including group-inherited eligibility via the asTarget() OData filter).
// subscriptionName is the friendly display name stored on each returned Role.
// re is the compiled scope_pattern regexp from config; pass nil to suppress the
// App/Env columns (i.e. when no pattern is configured).
// envLabels maps raw decoded env values to friendly labels (may be nil).
func FetchEligibleRoles(ctx context.Context, client *armauthorization.RoleEligibilityScheduleInstancesClient, scope string, subscriptionName string, re *regexp.Regexp, envLabels map[string]string) ([]model.Role, error) {
	filter := "asTarget()"
	pager := client.NewListForScopePager(scope, &armauthorization.RoleEligibilityScheduleInstancesClientListForScopeOptions{
		Filter: &filter,
	})

	var roles []model.Role
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching eligible roles: %w", err)
		}
		for _, inst := range page.Value {
			if inst == nil || inst.Properties == nil {
				continue
			}
			roles = append(roles, mapEligibleInstance(inst, subscriptionName, re, envLabels))
		}
	}
	return roles, nil
}

// FetchEligibleRoleExpiries retrieves all PIM-eligible role assignments for the
// current user together with the time remaining before each eligibility window
// expires. Eligibilities with no end date are returned with a zero ExpiresAt.
func FetchEligibleRoleExpiries(ctx context.Context, client *armauthorization.RoleEligibilityScheduleInstancesClient, scope string, subscriptionName string, re *regexp.Regexp, envLabels map[string]string) ([]model.EligibleRole, error) {
	filter := "asTarget()"
	pager := client.NewListForScopePager(scope, &armauthorization.RoleEligibilityScheduleInstancesClientListForScopeOptions{
		Filter: &filter,
	})

	var roles []model.EligibleRole
	now := time.Now()
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching eligible roles: %w", err)
		}
		for _, inst := range page.Value {
			if inst == nil || inst.Properties == nil {
				continue
			}
			roles = append(roles, mapEligibleExpiryInstance(inst, subscriptionName, re, envLabels, now))
		}
	}
	return roles, nil
}

func mapEligibleInstance(inst *armauthorization.RoleEligibilityScheduleInstance, subscriptionName string, re *regexp.Regexp, envLabels map[string]string) model.Role {
	props := inst.Properties
	var (
		roleName  string
		scopeName string
		scopeType string
	)

	if exp := props.ExpandedProperties; exp != nil {
		if exp.RoleDefinition != nil {
			roleName = safeStr(exp.RoleDefinition.DisplayName)
		}
		if exp.Scope != nil {
			scopeName = strings.ToUpper(safeStr(exp.Scope.DisplayName))
			scopeType = safeStr(exp.Scope.Type)
		}
	}

	if roleName == "" {
		roleName = "Unknown Role"
	}
	if scopeName == "" {
		scopeName = "—"
	}

	env, app := model.DecodeScopeFields(scopeName, re, envLabels)

	r := model.Role{
		RoleName:         roleName,
		ScopeName:        scopeName,
		ScopeType:        scopeType,
		SubscriptionName: subscriptionName,
		RoleDefinitionID: safeStr(props.RoleDefinitionID),
		Scope:            safeStr(props.Scope),
		Environment:      env,
		AppCode:          app,
	}

	return r
}

func mapEligibleExpiryInstance(inst *armauthorization.RoleEligibilityScheduleInstance, subscriptionName string, re *regexp.Regexp, envLabels map[string]string, now time.Time) model.EligibleRole {
	role := model.EligibleRole{Role: mapEligibleInstance(inst, subscriptionName, re, envLabels)}
	if inst == nil || inst.Properties == nil || inst.Properties.EndDateTime == nil {
		return role
	}

	role.ExpiresAt = *inst.Properties.EndDateTime
	role.ExpiresIn = role.ExpiresAt.Sub(now)
	if role.ExpiresIn < 0 {
		role.ExpiresIn = 0
	}
	return role
}
