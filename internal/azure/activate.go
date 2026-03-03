package azure

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/google/uuid"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

var (
	newRequestID = func() string { return uuid.New().String() }
	nowUTC       = func() time.Time { return time.Now().UTC() }
	createRoleAssignmentScheduleRequest = func(
		ctx context.Context,
		client *armauthorization.RoleAssignmentScheduleRequestsClient,
		scope string,
		requestID string,
		req armauthorization.RoleAssignmentScheduleRequest,
	) error {
		_, err := client.Create(ctx, scope, requestID, req, nil)
		return err
	}
)

// ActivateRoles sends parallel SelfActivate PUT requests for all selected roles.
// Individual role failures are captured in ActivationResult.Err rather than
// aborting the whole group — every role gets an attempt.
func ActivateRoles(
	ctx context.Context,
	client *armauthorization.RoleAssignmentScheduleRequestsClient,
	roles []model.Role,
	principalID string,
	justification string,
	duration model.DurationOption,
) []model.ActivationResult {
	results := make([]model.ActivationResult, len(roles))
	var wg sync.WaitGroup

	for i, role := range roles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			requestID := newRequestID()
			startTime := nowUTC()

			req := armauthorization.RoleAssignmentScheduleRequest{
				Properties: &armauthorization.RoleAssignmentScheduleRequestProperties{
					PrincipalID:      to.Ptr(principalID),
					RoleDefinitionID: to.Ptr(role.RoleDefinitionID),
					RequestType:      to.Ptr(armauthorization.RequestTypeSelfActivate),
					Justification:    to.Ptr(justification),
					ScheduleInfo: &armauthorization.RoleAssignmentScheduleRequestPropertiesScheduleInfo{
						StartDateTime: to.Ptr(startTime),
						Expiration: &armauthorization.RoleAssignmentScheduleRequestPropertiesScheduleInfoExpiration{
							Type:     to.Ptr(armauthorization.TypeAfterDuration),
							Duration: to.Ptr(duration.ISO8601),
						},
					},
				},
			}

			err := createRoleAssignmentScheduleRequest(ctx, client, role.Scope, requestID, req)
			results[i] = model.ActivationResult{Role: role, Err: err}
		}()
	}

	wg.Wait()
	return results
}
