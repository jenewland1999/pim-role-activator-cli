// Package setup provides the first-run interactive configuration wizard.
package setup

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jenewland1999/pim-role-activator-cli/internal/config"
	"github.com/jenewland1999/pim-role-activator-cli/internal/tui"
)

// AvailableSubscription represents a subscription discovered from the Azure API.
type AvailableSubscription struct {
	ID   string
	Name string
}

// Run launches the interactive setup wizard and persists the result to dir.
// suggestedPrincipalID can be pre-populated from a decoded auth token (pass ""
// to leave the field blank for the user to fill in).
// availableSubs is an optional list of subscriptions discovered from the Azure
// API; when non-empty a multi-select is shown instead of manual entry.
func Run(dir string, suggestedPrincipalID string, availableSubs []AvailableSubscription) (*config.UserConfig, error) {
	fmt.Println()
	fmt.Println(tui.BoldCyan("  Setup — PIM Role Activator CLI"))
	fmt.Println(tui.Dim("  Configuration will be saved to ~/.pim/config.json"))
	fmt.Println()

	cfg := &config.UserConfig{}

	// ── Principal ID ──────────────────────────────────────────────────────────
	principalID := suggestedPrincipalID
	principalForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Your Entra ID Object ID (Principal ID)").
				Description("Found in Azure Portal → Entra ID → Users → your profile → Object ID.\nThis is used to scope PIM eligible-role queries to your identity.").
				Placeholder("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("principal ID cannot be empty")
					}
					return nil
				}).
				Value(&principalID),
		),
	)
	if err := principalForm.Run(); err != nil {
		if err.Error() == "user aborted" {
			return nil, fmt.Errorf("setup cancelled")
		}
		return nil, err
	}
	cfg.PrincipalID = strings.TrimSpace(principalID)

	// ── Subscriptions ─────────────────────────────────────────────────────────
	fmt.Println()

	if len(availableSubs) > 0 {
		// Build options for the multi-select from the discovered list.
		options := make([]huh.Option[string], len(availableSubs))
		for i, s := range availableSubs {
			label := fmt.Sprintf("%s (%s)", s.Name, s.ID)
			options[i] = huh.NewOption(label, s.ID)
		}

		var selectedIDs []string
		selectForm := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select Azure subscriptions to manage").
					Description("Use space to toggle, enter to confirm. At least one is required.").
					Options(options...).
					Validate(func(vals []string) error {
						if len(vals) == 0 {
							return fmt.Errorf("at least one subscription must be selected")
						}
						return nil
					}).
					Value(&selectedIDs),
			),
		)
		if err := selectForm.Run(); err != nil {
			if err.Error() == "user aborted" {
				return nil, fmt.Errorf("setup cancelled")
			}
			return nil, err
		}

		// Build a lookup from ID → name for the selected subscriptions.
		nameByID := make(map[string]string, len(availableSubs))
		for _, s := range availableSubs {
			nameByID[s.ID] = s.Name
		}
		for _, id := range selectedIDs {
			cfg.Subscriptions = append(cfg.Subscriptions, config.Subscription{
				ID:   id,
				Name: nameByID[id],
			})
		}
	} else {
		// Fall back to manual entry when no subscriptions were discovered.
		fmt.Println(tui.Dim("  Add one or more Azure subscriptions to manage."))
		fmt.Println()

		for {
			var subID, subName string
			subForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title(fmt.Sprintf("Subscription %d — ID", len(cfg.Subscriptions)+1)).
						Description("The UUID found under Subscriptions in the Azure Portal.").
						Placeholder("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("subscription ID cannot be empty")
							}
							return nil
						}).
						Value(&subID),
					huh.NewInput().
						Title(fmt.Sprintf("Subscription %d — Display name", len(cfg.Subscriptions)+1)).
						Description("A friendly label shown in the role selector (e.g. \"Production\").").
						Placeholder("e.g. Production").
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("subscription name cannot be empty")
							}
							return nil
						}).
						Value(&subName),
				),
			)
			if err := subForm.Run(); err != nil {
				if err.Error() == "user aborted" {
					if len(cfg.Subscriptions) > 0 {
						break // allow exit after at least one subscription
					}
					return nil, fmt.Errorf("setup cancelled")
				}
				return nil, err
			}

			cfg.Subscriptions = append(cfg.Subscriptions, config.Subscription{
				ID:   strings.TrimSpace(subID),
				Name: strings.TrimSpace(subName),
			})

			// Offer to add another subscription
			var addAnother bool
			moreForm := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Add another subscription?").
						Value(&addAnother),
				),
			)
			if err := moreForm.Run(); err != nil || !addAnother {
				break
			}
		}
	}

	// ── Group select patterns ─────────────────────────────────────────────────
	fmt.Println()
	var rawPatterns string
	patternsForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Group-select patterns (optional)").
				Description("Comma-separated substrings. Pressing 'g' in the role selector will\nselect all roles whose scope name contains any of these strings.\nLeave blank to skip.").
				Placeholder("e.g. APP1,APP2,APP3").
				Value(&rawPatterns),
		),
	)
	if err := patternsForm.Run(); err != nil && err.Error() != "user aborted" {
		return nil, err
	}

	for _, p := range strings.Split(rawPatterns, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			cfg.GroupSelectPatterns = append(cfg.GroupSelectPatterns, p)
		}
	}

	// ── Scope pattern (optional) ──────────────────────────────────────────────
	fmt.Println()
	var scopePattern string
	scopeForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Scope naming pattern (optional)").
				Description(
					"A Go regexp with named capture groups used to decode App/Env columns.\n" +
						"Groups: (?P<env>…) and (?P<app>…) — both are optional.\n" +
						"Example: ^.(?P<env>[PQTD]).{5}(?P<app>.{4})\n" +
						"Leave blank to hide the App/Env columns.",
				).
				Placeholder("e.g. ^.(?P<env>[PQTD]).{5}(?P<app>.{4})").
				Validate(func(s string) error {
					s = strings.TrimSpace(s)
					if s == "" {
						return nil // optional
					}
					_, err := regexp.Compile(s)
					return err
				}).
				Value(&scopePattern),
		),
	)
	if err := scopeForm.Run(); err != nil && err.Error() != "user aborted" {
		return nil, err
	}
	cfg.ScopePattern = strings.TrimSpace(scopePattern)

	// ── Environment label mappings (optional) ─────────────────────────────────
	// Only prompt when a scope_pattern with an env group was configured.
	if cfg.ScopePattern != "" {
		fmt.Println()
		var rawEnvLabels string
		envLabelsForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Environment label mappings (optional)").
					Description(
						"Map raw decoded env values to friendly display names.\n" +
							"Format: comma-separated KEY=VALUE pairs.\n" +
							"Example: P=Prod,D=Dev,Q=QA,T=Test\n" +
							"Leave blank to display the raw value as-is.",
					).
					Placeholder("e.g. P=Prod,D=Dev,Q=QA").
					Value(&rawEnvLabels),
			),
		)
		if err := envLabelsForm.Run(); err != nil && err.Error() != "user aborted" {
			return nil, err
		}

		rawEnvLabels = strings.TrimSpace(rawEnvLabels)
		if rawEnvLabels != "" {
			cfg.EnvLabels = make(map[string]string)
			for _, pair := range strings.Split(rawEnvLabels, ",") {
				pair = strings.TrimSpace(pair)
				if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
					k := strings.TrimSpace(kv[0])
					v := strings.TrimSpace(kv[1])
					if k != "" && v != "" {
						cfg.EnvLabels[k] = v
					}
				}
			}
			if len(cfg.EnvLabels) == 0 {
				cfg.EnvLabels = nil
			}
		}
	}

	// ── Persist ───────────────────────────────────────────────────────────────
	if err := config.Save(dir, cfg); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s Configuration saved to ~/.pim/config.json\n", tui.Check)
	fmt.Println(tui.Dim("  Run 'pim setup' at any time to update your settings."))
	fmt.Println()

	return cfg, nil
}
