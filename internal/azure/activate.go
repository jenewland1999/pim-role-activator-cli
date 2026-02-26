package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
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
) ([]model.ActivationResult, error) {
	results := make([]model.ActivationResult, len(roles))
	g, ctx := errgroup.WithContext(ctx)

	for i, role := range roles {
		i, role := i, role // capture loop variables for goroutine
		g.Go(func() error {
			requestID := uuid.New().String()
			startTime := time.Now().UTC()

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

			_, err := client.Create(ctx, role.Scope, requestID, req, nil)
			results[i] = model.ActivationResult{Role: role, Err: err}
			// Return nil so the errgroup doesn't cancel sibling requests on failure.
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("activation group error: %w", err)
	}
	return results, nil
}
