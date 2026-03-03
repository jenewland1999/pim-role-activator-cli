package azure

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

func TestActivateRoles_EmptyInput(t *testing.T) {
	originalCreate := createRoleAssignmentScheduleRequest
	defer func() { createRoleAssignmentScheduleRequest = originalCreate }()

	called := 0
	createRoleAssignmentScheduleRequest = func(
		_ context.Context,
		_ *armauthorization.RoleAssignmentScheduleRequestsClient,
		_ string,
		_ string,
		_ armauthorization.RoleAssignmentScheduleRequest,
	) error {
		called++
		return nil
	}

	results := ActivateRoles(context.Background(), nil, nil, "pid", "just", model.DurationOption{Label: "1h", ISO8601: "PT1H", Duration: time.Hour})
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(results))
	}
	if called != 0 {
		t.Fatalf("create called %d times, want 0", called)
	}
}

func TestActivateRoles_AttemptsAllRoles_AndPreservesResultIndex(t *testing.T) {
	originalCreate := createRoleAssignmentScheduleRequest
	originalNow := nowUTC
	originalID := newRequestID
	defer func() {
		createRoleAssignmentScheduleRequest = originalCreate
		nowUTC = originalNow
		newRequestID = originalID
	}()

	fixedNow := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)
	nowUTC = func() time.Time { return fixedNow }

	var idMu sync.Mutex
	nextID := 0
	newRequestID = func() string {
		idMu.Lock()
		defer idMu.Unlock()
		nextID++
		return fmt.Sprintf("req-%d", nextID)
	}

	roles := []model.Role{
		{RoleName: "Reader", Scope: "/subscriptions/s1/resourceGroups/rg1", RoleDefinitionID: "rd1"},
		{RoleName: "Contributor", Scope: "/subscriptions/s1/resourceGroups/rg2", RoleDefinitionID: "rd2"},
		{RoleName: "Owner", Scope: "/subscriptions/s1/resourceGroups/rg3", RoleDefinitionID: "rd3"},
	}
	duration := model.DurationOption{Label: "1 hour", ISO8601: "PT1H", Duration: time.Hour}

	failScope := roles[1].Scope
	var callMu sync.Mutex
	calls := make(map[string]armauthorization.RoleAssignmentScheduleRequest)

	createRoleAssignmentScheduleRequest = func(
		_ context.Context,
		_ *armauthorization.RoleAssignmentScheduleRequestsClient,
		scope string,
		_ string,
		req armauthorization.RoleAssignmentScheduleRequest,
	) error {
		callMu.Lock()
		calls[scope] = req
		callMu.Unlock()

		if scope == failScope {
			return errors.New("simulated activation failure")
		}
		return nil
	}

	results := ActivateRoles(context.Background(), nil, roles, "principal-123", "deploying fix", duration)
	if len(results) != len(roles) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(roles))
	}

	for i := range roles {
		if results[i].Role != roles[i] {
			t.Fatalf("results[%d].Role mismatch", i)
		}
	}

	if results[0].Err != nil {
		t.Fatalf("results[0].Err = %v, want nil", results[0].Err)
	}
	if results[1].Err == nil {
		t.Fatalf("results[1].Err = nil, want non-nil")
	}
	if results[2].Err != nil {
		t.Fatalf("results[2].Err = %v, want nil", results[2].Err)
	}

	if len(calls) != len(roles) {
		t.Fatalf("calls = %d, want %d", len(calls), len(roles))
	}

	for _, r := range roles {
		req, ok := calls[r.Scope]
		if !ok {
			t.Fatalf("missing create call for scope %q", r.Scope)
		}
		if req.Properties == nil {
			t.Fatalf("request properties nil for scope %q", r.Scope)
		}
		if got := deref(req.Properties.PrincipalID); got != "principal-123" {
			t.Fatalf("principalID = %q, want %q", got, "principal-123")
		}
		if got := deref(req.Properties.RoleDefinitionID); got != r.RoleDefinitionID {
			t.Fatalf("roleDefinitionID = %q, want %q", got, r.RoleDefinitionID)
		}
		if got := deref(req.Properties.Justification); got != "deploying fix" {
			t.Fatalf("justification = %q, want %q", got, "deploying fix")
		}
		if req.Properties.ScheduleInfo == nil || req.Properties.ScheduleInfo.Expiration == nil {
			t.Fatalf("schedule info missing for scope %q", r.Scope)
		}
		if got := deref(req.Properties.ScheduleInfo.Expiration.Duration); got != duration.ISO8601 {
			t.Fatalf("duration = %q, want %q", got, duration.ISO8601)
		}
		if req.Properties.ScheduleInfo.StartDateTime == nil || !req.Properties.ScheduleInfo.StartDateTime.Equal(fixedNow) {
			t.Fatalf("startDateTime = %v, want %v", req.Properties.ScheduleInfo.StartDateTime, fixedNow)
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
