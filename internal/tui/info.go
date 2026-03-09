package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

const (
	infoExpiryWarningThreshold = 14 * 24 * time.Hour
	infoExpiryUrgentThreshold  = 7 * 24 * time.Hour
)

// PrintInfo renders eligible roles ordered by eligibility expiry.
func PrintInfo(roles []model.EligibleRole, showAppEnv bool) {
	if len(roles) == 0 {
		fmt.Println()
		fmt.Println("  " + Dim("No eligible PIM role assignments were found."))
		fmt.Println()
		return
	}

	fmt.Println()
	fmt.Printf("  %s %d eligible PIM role(s), ordered by eligibility expiry:\n", Check, len(roles))
	fmt.Println()

	if showAppEnv {
		hdr := fmt.Sprintf("  %-4s │ %-4s │ %-18s │ %-24s │ %-16s │ %-10s │ %-24s",
			"App", "Env", "Scope", "Role", "Expires", "In", "Subscription")
		fmt.Println(Bold(hdr))
		fmt.Println("  " + Dim(strings.Repeat("─", 122)))
		for _, r := range roles {
			expiresAt := formatInfoExpiryTimestamp(r.ExpiresAt, r.ExpiresIn)
			expiresIn := formatInfoExpiryCell(formatInfoExpiryDuration(r.ExpiresAt, r.ExpiresIn), r.ExpiresAt, r.ExpiresIn, 10)
			fmt.Printf("  %-4s │ %-4s │ %-18s │ %-24s │ %s │ %s │ %-24s\n",
				displayOrDash(r.AppCode),
				displayOrDash(r.Environment),
				Truncate(r.ScopeName, 18),
				Truncate(r.RoleName, 24),
				expiresAt,
				expiresIn,
				Truncate(r.SubscriptionName, 24),
			)
		}
	} else {
		hdr := fmt.Sprintf("  %-18s │ %-24s │ %-16s │ %-10s │ %-24s",
			"Scope", "Role", "Expires", "In", "Subscription")
		fmt.Println(Bold(hdr))
		fmt.Println("  " + Dim(strings.Repeat("─", 102)))
		for _, r := range roles {
			expiresAt := formatInfoExpiryTimestamp(r.ExpiresAt, r.ExpiresIn)
			expiresIn := formatInfoExpiryCell(formatInfoExpiryDuration(r.ExpiresAt, r.ExpiresIn), r.ExpiresAt, r.ExpiresIn, 10)
			fmt.Printf("  %-18s │ %-24s │ %s │ %s │ %-24s\n",
				Truncate(r.ScopeName, 18),
				Truncate(r.RoleName, 24),
				expiresAt,
				expiresIn,
				Truncate(r.SubscriptionName, 24),
			)
		}
	}

	fmt.Println()
	fmt.Println("  " + Dim("Red: eligibility expires within 7 days. Orange: eligibility expires within 14 days."))
	fmt.Println()
}

func infoExpirySeverity(expiresAt time.Time, expiresIn time.Duration) int {
	if expiresAt.IsZero() {
		return 0
	}
	switch {
	case expiresIn <= infoExpiryUrgentThreshold:
		return 2
	case expiresIn <= infoExpiryWarningThreshold:
		return 1
	default:
		return 0
	}
}

func formatInfoExpiryCell(value string, expiresAt time.Time, expiresIn time.Duration, width int) string {
	cell := fmt.Sprintf("%-*s", width, value)
	switch infoExpirySeverity(expiresAt, expiresIn) {
	case 2:
		return Red(cell)
	case 1:
		return Orange(cell)
	default:
		return cell
	}
}

func formatInfoExpiryTimestamp(expiresAt time.Time, expiresIn time.Duration) string {
	if expiresAt.IsZero() {
		return formatInfoExpiryCell("never", expiresAt, expiresIn, 16)
	}
	if expiresIn <= 0 {
		return formatInfoExpiryCell("expired", expiresAt, expiresIn, 16)
	}
	return formatInfoExpiryCell(expiresAt.Local().Format("2006-01-02 15:04"), expiresAt, expiresIn, 16)
}

func formatInfoExpiryDuration(expiresAt time.Time, expiresIn time.Duration) string {
	if expiresAt.IsZero() {
		return "never"
	}
	return FormatExpiryDuration(expiresIn)
}
