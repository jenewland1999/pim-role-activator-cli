package azure

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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
