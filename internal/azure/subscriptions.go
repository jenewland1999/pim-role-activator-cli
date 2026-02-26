package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// SubscriptionInfo holds the ID and display name of an Azure subscription
// as returned by the ARM subscriptions list API.
type SubscriptionInfo struct {
	ID   string
	Name string
}

// FetchSubscriptions lists all Azure subscriptions visible to the authenticated
// principal by calling the ARM management API directly.
func FetchSubscriptions(ctx context.Context, cred azcore.TokenCredential) ([]SubscriptionInfo, error) {
	tok, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return nil, fmt.Errorf("acquiring token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://management.azure.com/subscriptions?api-version=2022-12-01", nil)
	if err != nil {
		return nil, fmt.Errorf("building subscriptions request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.Token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing subscriptions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d listing subscriptions", resp.StatusCode)
	}

	var result struct {
		Value []struct {
			SubscriptionID string `json:"subscriptionId"`
			DisplayName    string `json:"displayName"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding subscriptions response: %w", err)
	}

	subs := make([]SubscriptionInfo, 0, len(result.Value))
	for _, v := range result.Value {
		subs = append(subs, SubscriptionInfo{
			ID:   v.SubscriptionID,
			Name: v.DisplayName,
		})
	}
	return subs, nil
}
