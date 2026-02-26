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

// FetchActiveRoles retrieves currently active PIM role assignments and filters
// for assignmentType == "Activated" (excludes permanent/direct assignments).
//
// subscriptionName is the friendly display name attached to each returned role.
// The justificationMap argument is keyed by "scope|roleDefinitionID" and
// is populated from the local state file by the caller.
// re is the compiled scope_pattern regexp from config; pass nil to suppress App/Env columns.
// envLabels maps raw decoded env values to friendly labels (may be nil).
func FetchActiveRoles(ctx context.Context, client *armauthorization.RoleAssignmentScheduleInstancesClient, scope string, subscriptionName string, justificationMap map[string]string, re *regexp.Regexp, envLabels map[string]string) ([]model.ActiveRole, error) {
	filter := "asTarget()"
	pager := client.NewListForScopePager(scope, &armauthorization.RoleAssignmentScheduleInstancesClientListForScopeOptions{
		Filter: &filter,
	})

	var roles []model.ActiveRole
	now := time.Now()

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching active roles: %w", err)
		}
		for _, inst := range page.Value {
			if inst == nil || inst.Properties == nil {
				continue
			}
			// Only include PIM JIT activations, not permanent assignments.
			if inst.Properties.AssignmentType == nil ||
				*inst.Properties.AssignmentType != armauthorization.AssignmentTypeActivated {
				continue
			}

			r := mapActiveInstance(inst, now, justificationMap, subscriptionName, re, envLabels)
			roles = append(roles, r)
		}
	}
	return roles, nil
}

func mapActiveInstance(inst *armauthorization.RoleAssignmentScheduleInstance, now time.Time, justificationMap map[string]string, subscriptionName string, re *regexp.Regexp, envLabels map[string]string) model.ActiveRole {
	props := inst.Properties
	var (
		roleName  string
		scopeName string
		scopeType string
		scopeID   = safeStr(props.Scope)
		roleDefID = safeStr(props.RoleDefinitionID)
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
		roleName = "Unknown"
	}
	if scopeName == "" {
		scopeName = "—"
	}

	var expiresIn time.Duration
	if props.EndDateTime != nil {
		expiresIn = props.EndDateTime.Sub(now)
		if expiresIn < 0 {
			expiresIn = 0
		}
	}

	env, app := model.DecodeScopeFields(scopeName, re, envLabels)

	key := scopeID + "|" + roleDefID
	justification := "—"
	if j, ok := justificationMap[key]; ok && j != "" {
		justification = j
	}

	return model.ActiveRole{
		RoleName:         roleName,
		ScopeName:        scopeName,
		ScopeType:        scopeType,
		SubscriptionName: subscriptionName,
		Environment:      env,
		AppCode:          app,
		ExpiresIn:        expiresIn,
		Justification:    justification,
	}
}
