package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

// rowRender holds pre-built output lines for a role in each of its four possible
// display states. Building these once up-front means View() only needs a slice
// lookup per row — no fmt.Sprintf, truncate, or lipgloss calls in the hot path.
type rowRender struct {
	normalUnsel string // cursor elsewhere, role not selected
	normalSel   string // cursor elsewhere, role selected
	cursorUnsel string // cursor is on this row, role not selected
	cursorSel   string // cursor is on this row, role selected
}

// buildRowRender pre-renders all four display states for a single role.
// When showAppEnv is true, column order is: App · Env · Scope · Role · Sub.
// When false (no scope_pattern configured), the App and Env columns are omitted
// and the order is: Scope · Role · Sub.
// IMPORTANT: the selection marker and rest must be plain unstyled text so that
// a single lipgloss call covers the entire line uniformly. Pre-rendering
// a coloured checkmark inside the line breaks Reverse() for cursor rows.
func buildRowRender(r model.Role, showAppEnv bool) rowRender {
	var rest string
	if showAppEnv {
		rest = fmt.Sprintf("%-4s  %-4s  %-18s  %-30s  %-32s",
			r.AppCode,
			r.Environment,
			Truncate(r.ScopeName, 18),
			Truncate(r.RoleName, 30),
			Truncate(r.SubscriptionName, 32),
		)
	} else {
		rest = fmt.Sprintf("%-18s  %-30s  %-32s",
			Truncate(r.ScopeName, 18),
			Truncate(r.RoleName, 30),
			Truncate(r.SubscriptionName, 32),
		)
	}
	lineUnsel := "    " + rest // 4 spaces for marker column
	lineSel := "✔   " + rest   // checkmark + 3 spaces (same display width)
	return rowRender{
		normalUnsel: "  " + Dim(lineUnsel) + "\n",
		normalSel:   "  " + Green(lineSel) + "\n",
		cursorUnsel: "  " + Reverse(lineUnsel) + "\n",
		cursorSel:   "  " + Reverse(lineSel) + "\n",
	}
}

// selectorModel is the bubbletea model for the interactive role selector.
type selectorModel struct {
	roles         []model.Role
	renders       []rowRender // pre-rendered rows, indexed by role index
	cursor        int         // cursor position within visible slice
	visible       []int       // indices into roles that match the current search
	search        textinput.Model
	searching     bool
	quitting      bool
	cancelled     bool
	width         int
	height        int
	groupPatterns []string // substrings used by the 'g' shortcut
	showAppEnv    bool     // true when a scope_pattern is configured in config
}

func newSelectorModel(roles []model.Role, groupPatterns []string, showAppEnv bool) selectorModel {
	ti := textinput.New()
	ti.Placeholder = "filter roles…"
	ti.CharLimit = 60
	ti.Width = 50

	m := selectorModel{
		roles:         roles,
		renders:       make([]rowRender, len(roles)),
		search:        ti,
		width:         80,
		height:        24,
		groupPatterns: groupPatterns,
		showAppEnv:    showAppEnv,
	}
	m.rebuildVisible()
	m.buildAllRenders()
	return m
}

// buildAllRenders rebuilds the render cache for every role.
func (m *selectorModel) buildAllRenders() {
	if len(m.renders) != len(m.roles) {
		m.renders = make([]rowRender, len(m.roles))
	}
	for i := range m.roles {
		m.renders[i] = buildRowRender(m.roles[i], m.showAppEnv)
	}
}

// rebuildRenderAt refreshes the cache entry for a single role index.
func (m *selectorModel) rebuildRenderAt(i int) {
	if i >= 0 && i < len(m.renders) {
		m.renders[i] = buildRowRender(m.roles[i], m.showAppEnv)
	}
}

// rebuildVisible recomputes the visible index slice based on the current search term.
func (m *selectorModel) rebuildVisible() {
	term := strings.ToLower(m.search.Value())
	m.visible = m.visible[:0]
	for i, r := range m.roles {
		if term == "" {
			m.visible = append(m.visible, i)
			continue
		}
		line := strings.ToLower(fmt.Sprintf("%s %s %s %s %s", r.RoleName, r.ScopeName, r.Environment, r.AppCode, r.SubscriptionName))
		if strings.Contains(line, term) {
			m.visible = append(m.visible, i)
		}
	}
	// Clamp cursor
	if m.cursor >= len(m.visible) && len(m.visible) > 0 {
		m.cursor = len(m.visible) - 1
	}
}

func (m *selectorModel) selectedCount() int {
	n := 0
	for _, r := range m.roles {
		if r.Selected {
			n++
		}
	}
	return n
}

// Init implements tea.Model.
func (m selectorModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			switch msg.Type {
			case tea.KeyEnter, tea.KeyEsc:
				m.searching = false
				m.search.Blur()
				m.cursor = 0
				m.rebuildVisible()
				return m, nil
			case tea.KeyBackspace:
				var cmd tea.Cmd
				m.search, cmd = m.search.Update(msg)
				m.cursor = 0
				m.rebuildVisible()
				return m, cmd
			}
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			m.cursor = 0
			m.rebuildVisible()
			return m, cmd
		}

		// Normal (navigation) mode
		total := len(m.visible)
		switch msg.String() {
		case "up", "k":
			if total > 0 {
				m.cursor = (m.cursor - 1 + total) % total
			}

		case "down", "j":
			if total > 0 {
				m.cursor = (m.cursor + 1) % total
			}

		case " ":
			if total > 0 {
				idx := m.visible[m.cursor]
				m.roles[idx].Selected = !m.roles[idx].Selected
				m.rebuildRenderAt(idx)
			}

		case "a", "A":
			for _, i := range m.visible {
				m.roles[i].Selected = true
			}
			m.buildAllRenders()

		case "n", "N":
			for _, i := range m.visible {
				m.roles[i].Selected = false
			}
			m.buildAllRenders()

		case "g", "G":
			for i, r := range m.roles {
				for _, p := range m.groupPatterns {
					if strings.Contains(r.ScopeName, p) {
						m.roles[i].Selected = true
						break
					}
				}
			}
			m.buildAllRenders()

		case "/":
			m.searching = true
			m.search.Focus()
			return m, textinput.Blink

		case "backspace", "ctrl+h":
			if m.search.Value() != "" {
				m.search.SetValue("")
				m.cursor = 0
				m.rebuildVisible()
			}

		case "c", "C", "ctrl+c", "esc":
			m.cancelled = true
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if m.selectedCount() > 0 {
				m.quitting = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m selectorModel) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(Bold("Step 1: Select PIM roles to activate") + "\n")
	sb.WriteString(Dim("  ↑/↓ Navigate  ⨍ Toggle  a Select all  n Deselect all") + "\n")
	if len(m.groupPatterns) > 0 {
		sb.WriteString(Dim(fmt.Sprintf("  g Group (%s)", strings.Join(m.groupPatterns, "/"))) + "\n")
	}
	sb.WriteString(Dim("  / Search  Backspace Clear  Enter Confirm  c Cancel") + "\n")
	sb.WriteString("\n")

	if m.searching || m.search.Value() != "" {
		if m.searching {
			sb.WriteString("  " + Cyan("🔍 Search: ") + m.search.View() + "\n")
		} else {
			total := len(m.visible)
			fmt.Fprintf(&sb, "  %s %s  %s\n",
				Cyan("🔍 Filter:"),
				m.search.Value(),
				Dim(fmt.Sprintf("(%d match(es))", total)),
			)
		}
		sb.WriteString("\n")
	}

	// Column header — App/Env columns shown only when a scope_pattern is set
	if m.showAppEnv {
		fmt.Fprintf(&sb, "  %s\n", Dim(fmt.Sprintf("    %-4s  %-4s  %-18s  %-30s  %-32s", "App", "Env", "Scope", "Role", "Subscription")))
		sb.WriteString("  " + Dim(strings.Repeat("─", 100)) + "\n")
	} else {
		fmt.Fprintf(&sb, "  %s\n", Dim(fmt.Sprintf("    %-18s  %-30s  %-32s", "Scope", "Role", "Subscription")))
		sb.WriteString("  " + Dim(strings.Repeat("─", 88)) + "\n")
	}

	total := len(m.visible)
	// Reserve lines: banner(4) + search(2) + header(2) + footer(3) + padding(2)
	maxVisible := m.height - 13
	if maxVisible < 5 {
		maxVisible = 5
	}

	scrollStart := 0
	if total > maxVisible {
		scrollStart = m.cursor - maxVisible/2
		if scrollStart < 0 {
			scrollStart = 0
		}
		if scrollStart > total-maxVisible {
			scrollStart = total - maxVisible
		}
	}
	scrollEnd := scrollStart + maxVisible
	if scrollEnd > total {
		scrollEnd = total
	}

	if scrollStart > 0 {
		sb.WriteString(fmt.Sprintf("  %s\n", Dim(fmt.Sprintf("  ↑ %d more above", scrollStart))))
	}

	for vi := scrollStart; vi < scrollEnd; vi++ {
		realIdx := m.visible[vi]
		rnd := m.renders[realIdx]
		if vi == m.cursor {
			if m.roles[realIdx].Selected {
				sb.WriteString(rnd.cursorSel)
			} else {
				sb.WriteString(rnd.cursorUnsel)
			}
		} else if m.roles[realIdx].Selected {
			sb.WriteString(rnd.normalSel)
		} else {
			sb.WriteString(rnd.normalUnsel)
		}
	}

	below := total - scrollEnd
	if below > 0 {
		sb.WriteString(fmt.Sprintf("  %s\n", Dim(fmt.Sprintf("  ↓ %d more below", below))))
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n", Dim(fmt.Sprintf("%d selected", m.selectedCount()))))

	return sb.String()
}

// RunSelector launches the interactive role selector and returns the chosen roles.
// cancelled is true when the user pressed 'c' or Ctrl+C without confirming.
// showAppEnv should be true when a scope_pattern is configured in UserConfig.
func RunSelector(roles []model.Role, groupPatterns []string, showAppEnv bool) (selected []model.Role, cancelled bool, err error) {
	m := newSelectorModel(roles, groupPatterns, showAppEnv)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, runErr := p.Run()
	if runErr != nil {
		return nil, false, runErr
	}

	fm, ok := finalModel.(selectorModel)
	if !ok {
		return nil, false, fmt.Errorf("unexpected model type")
	}
	if fm.cancelled {
		return nil, true, nil
	}

	for _, r := range fm.roles {
		if r.Selected {
			selected = append(selected, r)
		}
	}
	return selected, false, nil
}
