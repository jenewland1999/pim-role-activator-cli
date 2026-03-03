package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

// PrintBanner prints the PIM CLI header banner.
func PrintBanner(dryRun bool) {
	fmt.Println()
	fmt.Println(BannerTop)
	fmt.Println(BannerMiddle)
	fmt.Println(BannerBottom)
	if dryRun {
		fmt.Println("  " + BoldYellow("▸ DRY RUN MODE"))
	}
	fmt.Println()
}

// PrintStatus renders the active-roles table to stdout.
// showAppEnv should be true when a scope_pattern is configured in UserConfig.
func PrintStatus(roles []model.ActiveRole, showAppEnv bool) {
	printStatusTable(roles, showAppEnv, false)
}

// PrintCachedStatus renders the active-roles table with a "(cached)" indicator.
func PrintCachedStatus(roles []model.ActiveRole, showAppEnv bool) {
	printStatusTable(roles, showAppEnv, true)
}

func printStatusTable(roles []model.ActiveRole, showAppEnv bool, cached bool) {
	if len(roles) == 0 {
		fmt.Println()
		fmt.Println("  " + Dim("No PIM roles are currently active."))
		fmt.Println("  " + Dim("Run ")+Bold("pim activate")+Dim(" to activate roles."))
		fmt.Println()
		return
	}

	suffix := ""
	if cached {
		suffix = " " + Dim("(cached)")
	}

	fmt.Println()
	fmt.Printf("  %s %d active PIM role(s):%s\n", Check, len(roles), suffix)
	fmt.Println()

	if showAppEnv {
		hdr := fmt.Sprintf("  %-4s │ %-4s │ %-20s │ %-30s │ %-12s │ %-20s │ %-32s",
			"App", "Env", "Scope", "Role", "Expires In", "Justification", "Subscription")
		fmt.Println(Bold(hdr))
		fmt.Println("  " + Dim(strings.Repeat("─", 140)))
		for _, r := range roles {
			exp := FormatExpiryDuration(r.ExpiresIn)
			fmt.Printf("  %-4s │ %-4s │ %-20s │ %-30s │ %-12s │ %-20s │ %-32s\n",
				r.AppCode, r.Environment, r.ScopeName, r.RoleName, exp, r.Justification, Truncate(r.SubscriptionName, 32))
		}
	} else {
		hdr := fmt.Sprintf("  %-20s │ %-30s │ %-12s │ %-20s │ %-32s",
			"Scope", "Role", "Expires In", "Justification", "Subscription")
		fmt.Println(Bold(hdr))
		fmt.Println("  " + Dim(strings.Repeat("─", 126)))
		for _, r := range roles {
			exp := FormatExpiryDuration(r.ExpiresIn)
			fmt.Printf("  %-20s │ %-30s │ %-12s │ %-20s │ %-32s\n",
				r.ScopeName, r.RoleName, exp, r.Justification, Truncate(r.SubscriptionName, 32))
		}
	}

	fmt.Println()
	fmt.Println("  " + Dim("Run ")+Bold("pim activate")+Dim(" to activate more roles."))
	fmt.Println()
}

// PrintSummary displays the pre-activation confirmation table.
// showAppEnv should be true when a scope_pattern is configured in UserConfig.
func PrintSummary(roles []model.Role, justification, durationLabel string, dryRun bool, showAppEnv bool) {
	// Match the overall width of the role selector table:
	//   showAppEnv=true  → "  " + 100 dashes = 102 chars
	//   showAppEnv=false → "  " + 88 dashes  =  90 chars
	totalWidth := 90
	if showAppEnv {
		totalWidth = 102
	}
	const titlePrefix = "─── Summary " // 12 chars
	topBorder := BoldCyan(titlePrefix + strings.Repeat("─", totalWidth-len(titlePrefix)+6))
	bottomBorder := BoldCyan(strings.Repeat("─", totalWidth))

	fmt.Println()
	fmt.Println(topBorder)
	fmt.Println("  " + Bold("Roles:"))
	for _, r := range roles {
		if showAppEnv {
			fmt.Printf("    %s %-4s  %-4s  %-18s  %-30s  %-32s\n",
				Arrow, r.AppCode, r.Environment, r.ScopeName, r.RoleName, Truncate(r.SubscriptionName, 32))
		} else {
			fmt.Printf("    %s %-18s  %-30s  %-32s\n",
				Arrow, r.ScopeName, r.RoleName, Truncate(r.SubscriptionName, 32))
		}
	}
	fmt.Println("  " + Bold("Justification: ") + justification)
	fmt.Println("  " + Bold("Duration:      ") + durationLabel)
	if dryRun {
		fmt.Println("  " + Bold("Mode:          ") + BoldYellow("DRY RUN"))
	}
	fmt.Println(bottomBorder)
	fmt.Println()
}

// PrintResults prints the final activation outcome.
func PrintResults(results []model.ActivationResult) {
	successes := 0
	failures := 0
	for _, r := range results {
		if r.Err == nil {
			successes++
		} else {
			failures++
		}
	}

	fmt.Println()
	fmt.Println(BoldCyan("─── Results ───────────────────────────"))
	if successes > 0 {
		fmt.Printf("  %s %d role(s) activated successfully.\n", Check, successes)
	}
	if failures > 0 {
		fmt.Printf("  %s %d role(s) failed to activate:\n", Cross, failures)
		for _, r := range results {
			if r.Err != nil {
				fmt.Printf("    • %s (%s): %v\n", r.Role.RoleName, r.Role.ScopeName, r.Err)
			}
		}
	}
	fmt.Println(BoldCyan("───────────────────────────────────────"))
	fmt.Println()
}

// FormatExpiryDuration converts a time.Duration to a concise human string.
func FormatExpiryDuration(d time.Duration) string {
	if d <= 0 {
		return "expired"
	}
	secs := int64(d.Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%dm", secs/60)
	}
	hrs := secs / 3600
	mins := (secs % 3600) / 60
	return fmt.Sprintf("%dh %dm", hrs, mins)
}
