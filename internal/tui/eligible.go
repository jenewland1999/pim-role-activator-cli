package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

const (
	eligibleExpiryWarningThreshold = 14 * 24 * time.Hour
	eligibleExpiryUrgentThreshold  = 7 * 24 * time.Hour
	eligibleExpiryQuarterThreshold = 90 * 24 * time.Hour
)

type eligibleGroup struct {
	title string
	roles []model.EligibleRole
}

// PrintEligible renders eligible roles ordered by eligibility expiry.
func PrintEligible(roles []model.EligibleRole, showAppEnv bool) {
	roles = sortEligibleRolesByExpiry(roles)

	if len(roles) == 0 {
		fmt.Println()
		fmt.Println("  " + Dim("No eligible PIM role assignments were found."))
		fmt.Println()
		return
	}

	groups := groupEligibleRoles(roles)
	visibleRoles := countEligibleRoles(groups)
	if visibleRoles == 0 {
		fmt.Println()
		fmt.Println("  " + Dim("No eligible PIM role assignments were found."))
		fmt.Println()
		return
	}

	fmt.Println()
	fmt.Printf("  %s %d eligible PIM role(s), ordered by eligibility expiry:\n", Check, visibleRoles)
	fmt.Println()

	for _, group := range groups {
		if len(group.roles) == 0 {
			continue
		}

		fmt.Println("  " + Bold(group.title))
		if showAppEnv {
			hdr := fmt.Sprintf("  %-4s │ %-4s │ %-18s │ %-24s │ %-16s │ %-12s │ %-24s",
				"App", "Env", "Scope", "Role", "Expires", "In", "Subscription")
			fmt.Println(Bold(hdr))
			fmt.Println("  " + Dim(strings.Repeat("─", 120)))
			for _, r := range group.roles {
				expiresAt := formatEligibleExpiryTimestamp(r.ExpiresAt, r.ExpiresIn)
				expiresIn := formatEligibleExpiryCell(formatEligibleExpiryDuration(r.ExpiresAt, r.ExpiresIn), r.ExpiresAt, r.ExpiresIn, 12)
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
			fmt.Println("  " + Dim(strings.Repeat("─", 106)))
			for _, r := range group.roles {
				expiresAt := formatEligibleExpiryTimestamp(r.ExpiresAt, r.ExpiresIn)
				expiresIn := formatEligibleExpiryCell(formatEligibleExpiryDuration(r.ExpiresAt, r.ExpiresIn), r.ExpiresAt, r.ExpiresIn, 12)
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

func groupEligibleRoles(roles []model.EligibleRole) []eligibleGroup {
	roles = sortEligibleRolesByExpiry(roles)

	groups := []eligibleGroup{
		{title: "Expires Within 14 Days"},
		{title: "Expires Within 1 Quarter"},
		{title: "Expires After 1 Quarter or Never"},
	}

	for _, role := range roles {
		switch {
		case role.ExpiresIn <= 0 && !role.ExpiresAt.IsZero():
			continue
		case role.ExpiresAt.IsZero() || role.ExpiresIn > eligibleExpiryQuarterThreshold:
			groups[2].roles = append(groups[2].roles, role)
		case role.ExpiresIn > eligibleExpiryWarningThreshold:
			groups[1].roles = append(groups[1].roles, role)
		default:
			groups[0].roles = append(groups[0].roles, role)
		}
	}

	return groups
}

func sortEligibleRolesByExpiry(roles []model.EligibleRole) []model.EligibleRole {
	sorted := append([]model.EligibleRole(nil), roles...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i].ExpiresAt
		right := sorted[j].ExpiresAt

		switch {
		case left.IsZero() && right.IsZero():
			return false
		case left.IsZero():
			return false
		case right.IsZero():
			return true
		default:
			return left.Before(right)
		}
	})
	return sorted
}

func countEligibleRoles(groups []eligibleGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.roles)
	}
	return total
}

func eligibleExpirySeverity(expiresAt time.Time, expiresIn time.Duration) int {
	if expiresAt.IsZero() {
		return 0
	}
	switch {
	case expiresIn <= eligibleExpiryUrgentThreshold:
		return 2
	case expiresIn <= eligibleExpiryWarningThreshold:
		return 1
	default:
		return 0
	}
}

func formatEligibleExpiryCell(value string, expiresAt time.Time, expiresIn time.Duration, width int) string {
	cell := fmt.Sprintf("%-*s", width, value)
	switch eligibleExpirySeverity(expiresAt, expiresIn) {
	case 2:
		return Red(cell)
	case 1:
		return Orange(cell)
	default:
		return cell
	}
}

func formatEligibleExpiryTimestamp(expiresAt time.Time, expiresIn time.Duration) string {
	if expiresAt.IsZero() {
		return formatEligibleExpiryCell("never", expiresAt, expiresIn, 16)
	}
	if expiresIn <= 0 {
		return formatEligibleExpiryCell("expired", expiresAt, expiresIn, 16)
	}
	return formatEligibleExpiryCell(expiresAt.Local().Format("2006-01-02 15:04"), expiresAt, expiresIn, 16)
}

func formatEligibleExpiryDuration(expiresAt time.Time, expiresIn time.Duration) string {
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
