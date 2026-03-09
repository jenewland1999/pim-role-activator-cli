package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/spf13/cobra"

	"github.com/jenewland1999/pim-role-activator-cli/internal/azure"
	"github.com/jenewland1999/pim-role-activator-cli/internal/config"
	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
	"github.com/jenewland1999/pim-role-activator-cli/internal/state"
	"github.com/jenewland1999/pim-role-activator-cli/internal/tui"
)

func runInfo(cmd *cobra.Command, _ []string) error {
	cmdStart := time.Now()
	ctx, cancel := context.WithTimeout(cmd.Context(), apiTimeout)
	defer cancel()
	dir := pimDir()

	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Subscriptions) == 0 {
		return fmt.Errorf("no subscriptions configured — run 'pim setup' to configure your subscriptions")
	}

	re, reErr := cfg.ParsedScopePattern()
	if reErr != nil {
		return fmt.Errorf("invalid scope_pattern in config: %w", reErr)
	}
	showAppEnv := re != nil

	tui.PrintBanner(false)

	var cred azcore.TokenCredential
	authStart := time.Now()
	if err := tui.RunWithSpinner("Authenticating with Azure…", func() error {
		var credErr error
		cred, credErr = azure.NewCredential()
		return credErr
	}); err != nil {
		logPhaseTiming("info_auth", authStart, "ok", false)
		return fmt.Errorf("%s %w", tui.Cross, err)
	}
	logPhaseTiming("info_auth", authStart, "ok", true)

	maybePrintIdentity(ctx, cred, "info")

	storeFile := filepath.Join(dir, "activations.json")
	justificationMap := state.New(storeFile).LookupJustification()

	var roles []model.ActiveRole
	fetchStart := time.Now()
	if err := tui.RunWithSpinner("Fetching active PIM roles…", func() error {
		var loadErr error
		roles, loadErr = loadActiveRoles(ctx, cfg, cred, justificationMap)
		return loadErr
	}); err != nil {
		logPhaseTiming("info_fetch_active_roles", fetchStart, "ok", false, "subscription_count", len(cfg.Subscriptions))
		return fmt.Errorf("%s Failed to fetch active role assignments: %w", tui.Cross, err)
	}
	logPhaseTiming("info_fetch_active_roles", fetchStart, "ok", true, "subscription_count", len(cfg.Subscriptions), "active_roles", len(roles))

	sortActiveRolesByExpiry(roles)
	tui.PrintInfo(roles, showAppEnv, time.Now())

	logPhaseTiming("info_total", cmdStart, "subscriptions", len(cfg.Subscriptions), "active_roles", len(roles))
	return nil
}
