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
	infoExpiryQuarterThreshold = 90 * 24 * time.Hour
)

type infoGroup struct {
	title string
	roles []model.EligibleRole
}

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

	groups := groupInfoRoles(roles)
	for _, group := range groups {
		if len(group.roles) == 0 {
			continue
		}

		fmt.Println("  " + Bold(group.title))
		if showAppEnv {
			hdr := fmt.Sprintf("  %-4s │ %-4s │ %-18s │ %-24s │ %-16s │ %-12s │ %-24s",
				"App", "Env", "Scope", "Role", "Expires", "In", "Subscription")
			fmt.Println(Bold(hdr))
			fmt.Println("  " + Dim(strings.Repeat("─", 124)))
			for _, r := range group.roles {
				expiresAt := formatInfoExpiryTimestamp(r.ExpiresAt, r.ExpiresIn)
				expiresIn := formatInfoExpiryCell(formatInfoExpiryDuration(r.ExpiresAt, r.ExpiresIn), r.ExpiresAt, r.ExpiresIn, 12)
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
			hdr := fmt.Sprintf("  %-18s │ %-24s │ %-16s │ %-12s │ %-24s",
				"Scope", "Role", "Expires", "In", "Subscription")
			fmt.Println(Bold(hdr))
			fmt.Println("  " + Dim(strings.Repeat("─", 104)))
			for _, r := range group.roles {
				expiresAt := formatInfoExpiryTimestamp(r.ExpiresAt, r.ExpiresIn)
				expiresIn := formatInfoExpiryCell(formatInfoExpiryDuration(r.ExpiresAt, r.ExpiresIn), r.ExpiresAt, r.ExpiresIn, 12)
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
	}

	fmt.Println("  " + Dim("Red: eligibility expires within 7 days. Orange: eligibility expires within 14 days."))
	fmt.Println()
}

func groupInfoRoles(roles []model.EligibleRole) []infoGroup {
	groups := []infoGroup{
		{title: "Expiring in the next 14 days"},
		{title: "Expiring from 14 days to 1 quarter"},
		{title: "Expiring after 1 quarter"},
	}

	for _, role := range roles {
		switch {
		case role.ExpiresAt.IsZero() || role.ExpiresIn > infoExpiryQuarterThreshold:
			groups[2].roles = append(groups[2].roles, role)
		case role.ExpiresIn > infoExpiryWarningThreshold:
			groups[1].roles = append(groups[1].roles, role)
		default:
			groups[0].roles = append(groups[0].roles, role)
		}
	}

	return groups
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
	if expiresIn <= 0 {
		return "expired"
	}

	if expiresIn < 24*time.Hour {
		return FormatExpiryDuration(expiresIn)
	}

	totalDays := int(expiresIn / (24 * time.Hour))
	hours := int((expiresIn % (24 * time.Hour)) / time.Hour)

	if totalDays < 14 {
		if hours == 0 {
			return fmt.Sprintf("%dd", totalDays)
		}
		return fmt.Sprintf("%dd %dh", totalDays, hours)
	}

	if totalDays < 90 {
		weeks := totalDays / 7
		days := totalDays % 7
		if days == 0 {
			return fmt.Sprintf("%dw", weeks)
		}
		return fmt.Sprintf("%dw %dd", weeks, days)
	}

	months := totalDays / 30
	remainingDays := totalDays % 30
	weeks := remainingDays / 7
	if weeks == 0 {
		return fmt.Sprintf("%dmo", months)
	}
	return fmt.Sprintf("%dmo %dw", months, weeks)
}
