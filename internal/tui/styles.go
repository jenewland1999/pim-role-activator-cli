// Package tui contains terminal UI components for the PIM CLI.
package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleBold   = lipgloss.NewStyle().Bold(true)
	styleDim    = lipgloss.NewStyle().Faint(true)
	styleCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	styleGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	styleOrange = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	styleRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	styleBoldCyan   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	styleBoldYellow = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	styleReverse    = lipgloss.NewStyle().Reverse(true)

	// Exported symbols used by other tui files and main.
	Check = styleGreen.Render("✔")
	Cross = styleRed.Render("✖")
	Arrow = styleCyan.Render("▸")

	BannerTop    = styleBoldCyan.Render("╔══════════════════════════════════════╗")
	BannerMiddle = styleBoldCyan.Render("║        PIM Role Activator CLI        ║")
	BannerBottom = styleBoldCyan.Render("╚══════════════════════════════════════╝")
)

// Bold wraps text in bold styling.
func Bold(s string) string { return styleBold.Render(s) }

// Dim wraps text in faint styling.
func Dim(s string) string { return styleDim.Render(s) }

// Cyan wraps text in cyan.
func Cyan(s string) string { return styleCyan.Render(s) }

// Green wraps text in green.
func Green(s string) string { return styleGreen.Render(s) }

// Yellow wraps text in yellow.
func Yellow(s string) string { return styleYellow.Render(s) }

// Orange wraps text in orange.
func Orange(s string) string { return styleOrange.Render(s) }

// Red wraps text in red.
func Red(s string) string { return styleRed.Render(s) }

// BoldCyan wraps text bold+cyan.
func BoldCyan(s string) string { return styleBoldCyan.Render(s) }

// BoldYellow wraps text bold+yellow.
func BoldYellow(s string) string { return styleBoldYellow.Render(s) }

// Reverse applies reverse-video styling (cursor highlight).
func Reverse(s string) string { return styleReverse.Render(s) }

// SelectionMarker returns a coloured checkmark when selected, otherwise spaces.
func SelectionMarker(selected bool) string {
	if selected {
		return styleGreen.Render("✔ ")
	}
	return "  "
}

// Truncate shortens s to max runes, appending "…" when truncated.
// It operates on runes so multi-byte characters are handled correctly.
func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
