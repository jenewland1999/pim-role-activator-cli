package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jenewland1999/pim-role-activator-cli/internal/azure"
	"github.com/jenewland1999/pim-role-activator-cli/internal/cache"
	"github.com/jenewland1999/pim-role-activator-cli/internal/config"
	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
	"github.com/jenewland1999/pim-role-activator-cli/internal/setup"
	"github.com/jenewland1999/pim-role-activator-cli/internal/state"
	"github.com/jenewland1999/pim-role-activator-cli/internal/tui"
)

// version is set at build time via:
//
//	go build -ldflags "-X main.version=1.2.3"
//
// When not set (i.e. during local development) it defaults to "dev".
var version = "dev"

var (
	dryRun     bool
	noCache    bool
	apiTimeout time.Duration
)

const defaultAPITimeout = 2 * time.Minute

// maxJustificationLen is the maximum number of runes allowed in a justification
// string. Azure PIM does not formally document a limit, but 500 characters is a
// reasonable upper bound that covers any realistic audit justification.
const maxJustificationLen = 500

// validateJustification checks that the justification text is non-empty, within
// the length limit, and free of control characters (tabs and newlines are not
// expected in a single-line input and could corrupt audit logs).
func validateJustification(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("justification cannot be empty")
	}
	if len([]rune(s)) > maxJustificationLen {
		return fmt.Errorf("justification must be %d characters or fewer", maxJustificationLen)
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return fmt.Errorf("justification must not contain control characters")
		}
	}
	return nil
}

// pimDir returns ~/.pim/ and ensures the directory exists.
func pimDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("could not determine home directory", "err", err)
		os.Exit(1)
	}
	dir := filepath.Join(home, ".pim")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		slog.Warn("could not create config directory", "path", dir, "err", err)
	}
	return dir
}

// ── Status mode ───────────────────────────────────────────────────────────────

func runStatus(cmd *cobra.Command, _ []string) error {
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

	// ── Fast path: show cached active roles immediately ──────────────────────
	cachedRoles, cacheHit := cache.LoadActiveRoles(dir, cfg.CacheTTL())
	if cacheHit {
		tui.PrintCachedStatus(cachedRoles, showAppEnv)
	}

	// ── Refresh from API ─────────────────────────────────────────────────────
	spinnerMsg := "Checking active PIM roles…"
	if cacheHit {
		spinnerMsg = "Refreshing active roles…"
	}

	var allRoles []model.ActiveRole
	if err := tui.RunWithSpinner(spinnerMsg, func() error {
		g, gctx := errgroup.WithContext(ctx)
		var mu sync.Mutex
		for _, sub := range cfg.Subscriptions {
			g.Go(func() error {
				clients, e := azure.NewClients(sub.ID, cred)
				if e != nil {
					return e
				}
				roles, e := azure.FetchActiveRoles(gctx, clients.Active, "/subscriptions/"+sub.ID, sub.Name, justMap, re, cfg.EnvLabels)
				if e != nil {
					return e
				}
				mu.Lock()
				allRoles = append(allRoles, roles...)
				mu.Unlock()
				return nil
			})
		}
		return g.Wait()
	}); err != nil {
		if cacheHit {
			// We already showed cached results — warn but don't fail hard.
			slog.Warn("could not refresh active roles", "err", err)
			return nil
		}
		return fmt.Errorf("%s Failed to fetch active role assignments: %w", tui.Cross, err)
	}

	// Persist fresh results to cache.
	if len(allRoles) > 0 {
		if saveErr := cache.SaveActiveRoles(dir, cfg.CacheTTL(), allRoles); saveErr != nil {
			slog.Warn("could not cache active roles", "err", saveErr)
		}
	}

	// If we showed cached data, only reprint if the fresh data differs.
	if cacheHit {
		if !model.ActiveRolesEqual(cachedRoles, allRoles) {
			tui.PrintStatus(allRoles, showAppEnv)
		} else {
			fmt.Println("  " + tui.Dim("Up to date."))
			fmt.Println()
		}
	} else {
		tui.PrintStatus(allRoles, showAppEnv)
	}
	return nil
}

// ── Activate mode ─────────────────────────────────────────────────────────────

func runActivate(cmd *cobra.Command, _ []string) error {
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
	roleCache := cache.New(dir, cfg.CacheTTL(), "eligible-roles")
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
			g, gctx := errgroup.WithContext(ctx)
			var mu sync.Mutex
			for _, sub := range cfg.Subscriptions {
				g.Go(func() error {
					clients, e := azure.NewClients(sub.ID, cred)
					if e != nil {
						return e
					}
					roles, e := azure.FetchEligibleRoles(gctx, clients.Eligible, "/subscriptions/"+sub.ID, sub.Name, re, cfg.EnvLabels)
					if e != nil {
						return e
					}
					mu.Lock()
					eligibleRoles = append(eligibleRoles, roles...)
					mu.Unlock()
					return nil
				})
			}
			return g.Wait()
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
				tui.Truncate(r.ScopeName, 18),
				tui.Truncate(r.RoleName, 30),
				tui.Truncate(r.SubscriptionName, 32),
			)
		}
	} else {
		fmt.Printf("  %s\n", tui.Dim(fmt.Sprintf("    %-18s  %-30s  %-32s", "Scope", "Role", "Subscription")))
		fmt.Println("  " + tui.Dim(strings.Repeat("─", 88)))
		for _, r := range selectedRoles {
			fmt.Printf("  %s %-18s  %-30s  %-32s\n",
				tui.Arrow,
				tui.Truncate(r.ScopeName, 18),
				tui.Truncate(r.RoleName, 30),
				tui.Truncate(r.SubscriptionName, 32),
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
				Validate(validateJustification).
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
	durationOpts := cfg.DurationOptions()
	// Default to index 1 ("1 hour") when using 4+ built-in options, else 0.
	defaultDurIdx := 0
	if len(durationOpts) > 1 {
		defaultDurIdx = 1
	}
	durationIdx, cancelled, err := tui.RunDurationSelector(durationOpts, defaultDurIdx)
	if err != nil {
		return fmt.Errorf("duration selector error: %w", err)
	}
	if cancelled {
		fmt.Println(tui.Yellow("Cancelled."))
		return nil
	}

	duration := durationOpts[durationIdx]

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
	results := azure.ActivateRoles(ctx, activationClients.Activation, selectedRoles, cfg.PrincipalID, justification, duration)

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
			slog.Warn("could not persist activation state", "err", appendErr)
		}
	}

	tui.PrintResults(results)
	return nil
}



// ── CLI wiring ────────────────────────────────────────────────────────────────

// runSetup runs the interactive configuration wizard.
func runSetup(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
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
	// Register a signal-aware root context so that SIGINT (Ctrl+C) and
	// SIGTERM cancel in-flight API calls cleanly instead of killing the
	// process mid-request.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rootCmd := &cobra.Command{
		Use:     "pim",
		Short:   "PIM Role Activator CLI",
		Version: version,
		Long: `An interactive CLI for activating Azure PIM (Privileged Identity Management)
eligible role assignments via the Azure Resource Manager REST API.`,
		RunE:          runStatus,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// First-run setup: if no config exists, run the wizard before any command.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if cmd.Name() == "setup" || cmd.Name() == "version" {
			return nil // these commands don't need config
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

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version, Go runtime, and platform information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("pim %s\n", version)
			fmt.Printf("go  %s\n", runtime.Version())
			fmt.Printf("os  %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}

	rootCmd.PersistentFlags().DurationVar(&apiTimeout, "timeout", defaultAPITimeout, "Timeout for Azure API calls (e.g. 30s, 2m, 5m)")

	rootCmd.SetContext(ctx)
	rootCmd.AddCommand(activateCmd, setupCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "\nInterrupted.")
			os.Exit(130) // 128 + SIGINT
		}
		slog.Error("command failed", "err", err)
		os.Exit(1)
	}
}
