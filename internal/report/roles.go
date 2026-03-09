package report

import (
	"context"
	"sort"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"golang.org/x/sync/errgroup"

	"github.com/jenewland1999/pim-role-activator-cli/internal/azure"
	"github.com/jenewland1999/pim-role-activator-cli/internal/config"
	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

func LoadActiveRoles(ctx context.Context, cfg *config.UserConfig, cred azcore.TokenCredential, justificationMap map[string]string) ([]model.ActiveRole, error) {
	re, err := cfg.ParsedScopePattern()
	if err != nil {
		return nil, err
	}

	rolesBySubscription := make([][]model.ActiveRole, len(cfg.Subscriptions))
	g, gctx := errgroup.WithContext(ctx)

	for index, sub := range cfg.Subscriptions {
		index, sub := index, sub
		g.Go(func() error {
			clients, clientErr := azure.NewClients(sub.ID, cred)
			if clientErr != nil {
				return clientErr
			}

			roles, fetchErr := azure.FetchActiveRoles(
				gctx,
				clients.Active,
				"/subscriptions/"+sub.ID,
				sub.Name,
				justificationMap,
				re,
				cfg.EnvLabels,
			)
			if fetchErr != nil {
				return fetchErr
			}

			rolesBySubscription[index] = roles
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var allRoles []model.ActiveRole
	for _, roles := range rolesBySubscription {
		allRoles = append(allRoles, roles...)
	}

	return allRoles, nil
}

func LoadEligibleRoleExpiries(ctx context.Context, cfg *config.UserConfig, cred azcore.TokenCredential) ([]model.EligibleRole, error) {
	re, err := cfg.ParsedScopePattern()
	if err != nil {
		return nil, err
	}

	rolesBySubscription := make([][]model.EligibleRole, len(cfg.Subscriptions))
	g, gctx := errgroup.WithContext(ctx)

	for index, sub := range cfg.Subscriptions {
		index, sub := index, sub
		g.Go(func() error {
			clients, clientErr := azure.NewClients(sub.ID, cred)
			if clientErr != nil {
				return clientErr
			}

			roles, fetchErr := azure.FetchEligibleRoleExpiries(
				gctx,
				clients.Eligible,
				"/subscriptions/"+sub.ID,
				sub.Name,
				re,
				cfg.EnvLabels,
			)
			if fetchErr != nil {
				return fetchErr
			}

			rolesBySubscription[index] = roles
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var allRoles []model.EligibleRole
	for _, roles := range rolesBySubscription {
		allRoles = append(allRoles, roles...)
	}

	return allRoles, nil
}

func SortEligibleRolesByExpiry(roles []model.EligibleRole) {
	sort.SliceStable(roles, func(i, j int) bool {
		iHasExpiry := !roles[i].ExpiresAt.IsZero()
		jHasExpiry := !roles[j].ExpiresAt.IsZero()
		if iHasExpiry != jHasExpiry {
			return iHasExpiry
		}
		if iHasExpiry && !roles[i].ExpiresAt.Equal(roles[j].ExpiresAt) {
			return roles[i].ExpiresAt.Before(roles[j].ExpiresAt)
		}
		if roles[i].SubscriptionName != roles[j].SubscriptionName {
			return roles[i].SubscriptionName < roles[j].SubscriptionName
		}
		if roles[i].ScopeName != roles[j].ScopeName {
			return roles[i].ScopeName < roles[j].ScopeName
		}
		return roles[i].RoleName < roles[j].RoleName
	})
}
