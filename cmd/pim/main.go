package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/jenewland1999/pim-role-activator-cli/internal/azure"
	"github.com/jenewland1999/pim-role-activator-cli/internal/cache"
	"github.com/jenewland1999/pim-role-activator-cli/internal/config"
	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
	"github.com/jenewland1999/pim-role-activator-cli/internal/setup"
	"github.com/jenewland1999/pim-role-activator-cli/internal/state"
	"github.com/jenewland1999/pim-role-activator-cli/internal/tui"
)

var (
	dryRun  bool
	noCache bool
)

// pimDir returns ~/.pim/ and ensures the directory exists.
func pimDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not determine home directory: %v\n", err)
		os.Exit(1)
	}
	dir := filepath.Join(home, ".pim")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create %s: %v\n", dir, err)
	}
	return dir
}

// ── Status mode ───────────────────────────────────────────────────────────────

func runStatus(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
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
	if err := tui.RunWithSpinner("Authenticating with Azure…", func() error {
		var e error
		cred, e = azure.NewCredential()
		return e
	}); err != nil {
		return fmt.Errorf("%s %w", tui.Cross, err)
	}

	if claims, claimErr := azure.GetTokenClaims(ctx, cred); claimErr == nil {
		fmt.Printf("  %s Signed in as %s\n\n", tui.Check, tui.Cyan(claims.DisplayName()))
	}

	storeFile := filepath.Join(dir, "activations.json")
	justMap := state.New(storeFile).LookupJustification()

	var allRoles []model.ActiveRole
	if err := tui.RunWithSpinner("Checking active PIM roles…", func() error {
		for _, sub := range cfg.Subscriptions {
			clients, e := azure.NewClients(sub.ID, cred)
			if e != nil {
				return e
			}
			roles, e := azure.FetchActiveRoles(ctx, clients.Active, "/subscriptions/"+sub.ID, sub.Name, justMap, re, cfg.EnvLabels)
			if e != nil {
				return e
			}
			allRoles = append(allRoles, roles...)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("%s Failed to fetch active role assignments: %w", tui.Cross, err)
	}

	tui.PrintStatus(allRoles, showAppEnv)
	return nil
}

// ── Activate mode ─────────────────────────────────────────────────────────────

func runActivate(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	dir := pimDir()

	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Subscriptions) == 0 {
		return fmt.Errorf("no subscriptions configured — run 'pim setup' to configure your subscriptions")
	}

	// Compile the optional scope_pattern regexp once; nil means no App/Env columns.
	re, reErr := cfg.ParsedScopePattern()
	if reErr != nil {
		return fmt.Errorf("invalid scope_pattern in config: %w", reErr)
	}
	showAppEnv := re != nil

	tui.PrintBanner(dryRun)

	var cred azcore.TokenCredential
	if err := tui.RunWithSpinner("Authenticating with Azure…", func() error {
		var e error
		cred, e = azure.NewCredential()
		return e
	}); err != nil {
		return fmt.Errorf("%s %w", tui.Cross, err)
	}

	if claims, claimErr := azure.GetTokenClaims(ctx, cred); claimErr == nil {
		fmt.Printf("  %s Signed in as %s\n\n", tui.Check, tui.Cyan(claims.DisplayName()))
	}

	// Build an activation client from the first subscription. The scope is
	// embedded per-request so this single client works across all subscriptions.
	activationClients, aErr := azure.NewClients(cfg.Subscriptions[0].ID, cred)
	if aErr != nil {
		return fmt.Errorf("%s creating activation client: %w", tui.Cross, aErr)
	}

	// ── Step 0: Eligible roles with 24h cache ────────────────────────────────
	roleCache := cache.New(dir, cfg.CacheTTL())
	var eligibleRoles []model.Role

	if !noCache {
		if data, ok := roleCache.Get(); ok {
			if jsonErr := json.Unmarshal(data, &eligibleRoles); jsonErr == nil {
				// Build a quick lookup from subscription UUID → display name so that
				// SubscriptionName is always current even with a stale cache.
				subNameByID := make(map[string]string, len(cfg.Subscriptions))
				for _, s := range cfg.Subscriptions {
					subNameByID[strings.ToLower(s.ID)] = s.Name
				}
				// Re-derive fields that may have been stale when the cache was written.
				for i := range eligibleRoles {
					eligibleRoles[i].ScopeName = strings.ToUpper(eligibleRoles[i].ScopeName)
					env, app := model.DecodeScopeFields(eligibleRoles[i].ScopeName, re, cfg.EnvLabels)
					eligibleRoles[i].Environment = env
					eligibleRoles[i].AppCode = app
					// Extract subscription UUID from the ARM scope path (/subscriptions/<uuid>/...)
					if parts := strings.SplitN(eligibleRoles[i].Scope, "/", 4); len(parts) >= 3 {
						if name, ok := subNameByID[strings.ToLower(parts[2])]; ok {
							eligibleRoles[i].SubscriptionName = name
						}
					}
				}
				age, _ := roleCache.Age()
				remaining := cfg.CacheTTL() - age
				refreshAt := time.Now().Add(remaining)
				relative := tui.FormatExpiryDuration(remaining)
				exact := refreshAt.Format("Mon 2 Jan 2006 at 15:04")
				fmt.Println(tui.Dim(fmt.Sprintf(
					"Using cached roles (refreshes in %s, at %s). Use --no-cache to bypass.",
					relative, exact,
				)))
			} else {
				eligibleRoles = nil // corrupt cache — fall through to API fetch
			}
		}
	}

	if eligibleRoles == nil {
		if err := tui.RunWithSpinner("Fetching eligible PIM roles…", func() error {
			for _, sub := range cfg.Subscriptions {
				clients, e := azure.NewClients(sub.ID, cred)
				if e != nil {
					return e
				}
				roles, e := azure.FetchEligibleRoles(ctx, clients.Eligible, "/subscriptions/"+sub.ID, sub.Name, re, cfg.EnvLabels)
				if e != nil {
					return e
				}
				eligibleRoles = append(eligibleRoles, roles...)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("%s Failed to fetch eligible roles: %w", tui.Cross, err)
		}
		if data, mErr := json.Marshal(eligibleRoles); mErr == nil {
			_ = roleCache.Set(data)
		}
	}

	if len(eligibleRoles) == 0 {
		fmt.Println(tui.Yellow("No eligible PIM role assignments found."))
		return nil
	}

	fmt.Println(tui.Dim(fmt.Sprintf("Found %d eligible role(s).", len(eligibleRoles))))

	// ── Step 1: Interactive role selector ────────────────────────────────────
	selectedRoles, cancelled, err := tui.RunSelector(eligibleRoles, cfg.GroupSelectPatterns, showAppEnv)
	if err != nil {
		return fmt.Errorf("selector error: %w", err)
	}
	if cancelled {
		fmt.Println(tui.Yellow("Cancelled."))
		fmt.Println()
		return nil
	}
	if len(selectedRoles) == 0 {
		fmt.Println(tui.Yellow("No roles selected."))
		return nil
	}

	// Print selected roles
	fmt.Println()
	fmt.Printf("  %s Selected %d role(s):\n", tui.Check, len(selectedRoles))
	if showAppEnv {
		fmt.Printf("  %s\n", tui.Dim(fmt.Sprintf("    %-4s  %-4s  %-18s  %-30s  %-32s", "App", "Env", "Scope", "Role", "Subscription")))
		fmt.Println("  " + tui.Dim(strings.Repeat("─", 100)))
		for _, r := range selectedRoles {
			fmt.Printf("  %s %-4s  %-4s  %-18s  %-30s  %-32s\n",
				tui.Arrow,
				r.AppCode,
				r.Environment,
				truncate(r.ScopeName, 18),
				truncate(r.RoleName, 30),
				truncate(r.SubscriptionName, 32),
			)
		}
	} else {
		fmt.Printf("  %s\n", tui.Dim(fmt.Sprintf("    %-18s  %-30s  %-32s", "Scope", "Role", "Subscription")))
		fmt.Println("  " + tui.Dim(strings.Repeat("─", 88)))
		for _, r := range selectedRoles {
			fmt.Printf("  %s %-18s  %-30s  %-32s\n",
				tui.Arrow,
				truncate(r.ScopeName, 18),
				truncate(r.RoleName, 30),
				truncate(r.SubscriptionName, 32),
			)
		}
	}

	// ── Step 2: Justification ─────────────────────────────────────────────────
	var justification string

	justForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Step 2: Justification").
				Description("Appears verbatim in Azure audit logs.").
				Placeholder("e.g. Deploying hotfix to production").
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("justification cannot be empty")
					}
					return nil
				}).
				Value(&justification),
		),
	)

	if err := justForm.Run(); err != nil {
		if err.Error() == "user aborted" {
			fmt.Println(tui.Yellow("Cancelled."))
			return nil
		}
		return fmt.Errorf("form error: %w", err)
	}

	// ── Step 3: Duration (arrow-key picker, defaults to 1 hour) ──────────────
	durationIdx, cancelled, err := tui.RunDurationSelector(1)
	if err != nil {
		return fmt.Errorf("duration selector error: %w", err)
	}
	if cancelled {
		fmt.Println(tui.Yellow("Cancelled."))
		return nil
	}

	duration := model.DurationOptions[durationIdx]

	// ── Summary ───────────────────────────────────────────────────────────────
	tui.PrintSummary(selectedRoles, justification, duration.Label, dryRun, showAppEnv)

	if dryRun {
		fmt.Println("  " + tui.BoldYellow("Dry run complete.") + " " + tui.Dim("No roles were activated."))
		fmt.Println()
		return nil
	}

	// ── Confirmation ──────────────────────────────────────────────────────────
	fmt.Print("  Proceed with activation? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" {
		fmt.Println(tui.Yellow("Aborted."))
		return nil
	}

	// ── Parallel activation ───────────────────────────────────────────────────
	fmt.Println()
	results, err := azure.ActivateRoles(ctx, activationClients.Activation, selectedRoles, cfg.PrincipalID, justification, duration)
	if err != nil {
		return fmt.Errorf("activation error: %w", err)
	}

	// ── Persist state ─────────────────────────────────────────────────────────
	storeFile := filepath.Join(dir, "activations.json")
	stateStore := state.New(storeFile)

	now := time.Now()
	var newRecords []model.ActivationRecord
	for _, res := range results {
		if res.Err != nil {
			continue
		}
		newRecords = append(newRecords, model.ActivationRecord{
			Scope:            res.Role.Scope,
			RoleDefinitionID: res.Role.RoleDefinitionID,
			RoleName:         res.Role.RoleName,
			ScopeName:        res.Role.ScopeName,
			Justification:    justification,
			Duration:         duration.Label,
			ActivatedAt:      now.UTC().Format(time.RFC3339),
			ExpiresEpoch:     now.Add(duration.Duration).Unix(),
		})
	}

	if len(newRecords) > 0 {
		if appendErr := stateStore.Append(newRecords); appendErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not persist activation state: %v\n", appendErr)
		}
	}

	tui.PrintResults(results)
	return nil
}

// truncate shortens s to max runes, appending "…" when truncated.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// ── CLI wiring ────────────────────────────────────────────────────────────────

// runSetup runs the interactive configuration wizard.
func runSetup(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	dir := pimDir()

	var suggestedOID string
	var availableSubs []setup.AvailableSubscription

	if cred, credErr := azure.NewCredential(); credErr == nil {
		if claims, claimErr := azure.GetTokenClaims(ctx, cred); claimErr == nil {
			suggestedOID = claims.OID
			fmt.Printf("  %s Detected identity: %s\n", tui.Check, tui.Cyan(claims.DisplayName()))
			fmt.Printf("  %s Object ID: %s\n\n", tui.Arrow, tui.Dim(claims.OID))
		}
		// Attempt to discover subscriptions; non-fatal if it fails.
		if subs, subErr := azure.FetchSubscriptions(ctx, cred); subErr == nil && len(subs) > 0 {
			for _, s := range subs {
				availableSubs = append(availableSubs, setup.AvailableSubscription{
					ID:   s.ID,
					Name: s.Name,
				})
			}
			fmt.Printf("  %s Detected %d subscription(s).\n\n", tui.Check, len(availableSubs))
		}
	}

	_, err := setup.Run(dir, suggestedOID, availableSubs)
	return err
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "pim",
		Short: "PIM Role Activator CLI",
		Long: `An interactive CLI for activating Azure PIM (Privileged Identity Management)
eligible role assignments via the Azure Resource Manager REST API.`,
		RunE:          runStatus,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// First-run setup: if no config exists, run the wizard before any command.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if cmd.Name() == "setup" {
			return nil // setup manages its own config
		}
		dir := pimDir()
		if !config.Exists(dir) {
			fmt.Println(tui.BoldCyan("  Welcome to PIM Role Activator CLI!"))
			fmt.Println(tui.Dim("  No configuration found. Running first-time setup…"))
			fmt.Println()
			if err := runSetup(cmd, nil); err != nil {
				return err
			}
		}
		return nil
	}

	activateCmd := &cobra.Command{
		Use:   "activate",
		Short: "Activate eligible PIM roles interactively",
		Long: `Presents a scrollable list of eligible role assignments and activates
the selected roles after prompting for justification and duration.`,
		RunE:          runActivate,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	activateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Walk through prompts without sending activation requests")
	activateCmd.Flags().BoolVar(&noCache, "no-cache", false, "Bypass the 24-hour eligible roles cache")

	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure the CLI (subscriptions, principal ID, group patterns)",
		Long: `Launches an interactive wizard to update ~/.pim/config.json.
Run this to add subscriptions, change your principal ID, or update group-select patterns.`,
		RunE:          runSetup,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(activateCmd, setupCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", tui.Cross, err)
		os.Exit(1)
	}
}
