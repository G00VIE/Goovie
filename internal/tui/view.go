package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true).MarginBottom(1)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// View implements tea.Model.
func (m Model) View() string {
	if m.Err != nil {
		return errorStyle.Render(fmt.Sprintf("\nError: %v\n\nPress Esc to quit.", m.Err))
	}

	var s string
	switch m.State {
	case StateSearchInput:
		s = titleStyle.Render("Search Prowlarr") + "\n" + m.TextInput.View()
	case StateLoading:
		s = fmt.Sprintf("\n %s Reaching out to indexers...", m.Spinner.View())
	case StateList:
		s = m.List.View()
		if m.PendingIndexers > 0 {
			s += fmt.Sprintf("\n %s Loading %d remaining indexers...", m.Spinner.View(), m.PendingIndexers)
		} else {
			s += "\n ✓ All indexers loaded."
		}
	}

	return appStyle.Render(s)
}
