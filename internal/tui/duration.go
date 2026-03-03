package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

// durationModel is the bubbletea model for the arrow-key duration picker.
type durationModel struct {
	options   []model.DurationOption
	cursor    int
	quitting  bool
	cancelled bool
}

func newDurationModel(options []model.DurationOption, defaultIdx int) durationModel {
	if defaultIdx < 0 || defaultIdx >= len(options) {
		defaultIdx = 0
	}
	return durationModel{
		options: options,
		cursor:  defaultIdx,
	}
}

// Init implements tea.Model.
func (m durationModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m durationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kMsg, ok := msg.(tea.KeyMsg); ok {
		n := len(m.options)
		switch kMsg.String() {
		case "up", "k":
			m.cursor = (m.cursor - 1 + n) % n
		case "down", "j":
			m.cursor = (m.cursor + 1) % n
		case "enter", " ":
			m.quitting = true
			return m, tea.Quit
		case "esc", "c", "C", "ctrl+c":
			m.cancelled = true
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m durationModel) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(Bold("Step 3: Activation duration") + "\n")
	sb.WriteString(Dim("  ↑/↓ Navigate  Enter Confirm  c Cancel") + "\n")
	sb.WriteString("\n")

	for i, opt := range m.options {
		if i == m.cursor {
			sb.WriteString(Reverse(fmt.Sprintf("  ► %-12s", opt.Label)) + "\n")
		} else {
			sb.WriteString(Dim(fmt.Sprintf("    %-12s", opt.Label)) + "\n")
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// RunDurationSelector launches the arrow-key duration picker.
// options is the list of durations to present; defaultIdx sets the initial
// cursor position. Returns the chosen index, a cancelled flag, and any error.
func RunDurationSelector(options []model.DurationOption, defaultIdx int) (int, bool, error) {
	m := newDurationModel(options, defaultIdx)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithFPS(60))
	finalModel, err := p.Run()
	if err != nil {
		return 0, false, err
	}
	fm, ok := finalModel.(durationModel)
	if !ok {
		return 0, false, fmt.Errorf("unexpected duration model type")
	}
	if fm.cancelled {
		return 0, true, nil
	}
	return fm.cursor, false, nil
}
