package tui

import (
	"bubble-stream/internal/prowlarr"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// sessionState tracks which screen the TUI is currently showing.
type sessionState int

const (
	StateSearchInput sessionState = iota
	StateLoading
	StateList
)

// Model is the top-level bubbletea model for the application.
type Model struct {
	State           sessionState
	TextInput       textinput.Model
	List            list.Model
	Spinner         spinner.Model
	Err             error
	StreamURL       string
	AllResults      []prowlarr.TorrentResult
	PendingIndexers int
	Client          *prowlarr.Client
}

// NewModel creates and returns the initial TUI model.
func NewModel(client *prowlarr.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter Movie or Show (e.g., Breaking Bad S01E01)..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Prowlarr Results"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return Model{
		State:      StateSearchInput,
		TextInput:  ti,
		List:       l,
		Spinner:    sp,
		AllResults: []prowlarr.TorrentResult{},
		Client:     client,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}
