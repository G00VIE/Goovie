package tui

import (
	"fmt"
	"sort"

	"bubble-stream/internal/prowlarr"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			switch m.State {
			case StateSearchInput:
				if m.TextInput.Value() != "" {
					m.State = StateLoading
					cmds = append(cmds, m.Spinner.Tick, m.Client.FetchIndexersCmd(m.TextInput.Value()))
				}
			case StateList:
				if i, ok := m.List.SelectedItem().(prowlarr.TorrentResult); ok {
					m.StreamURL = i.MagnetURL
					return m, tea.Quit
				}
			}
		}

	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.List.SetSize(msg.Width-h, msg.Height-v)

	case spinner.TickMsg:
		if m.State == StateLoading || (m.State == StateList && m.PendingIndexers > 0) {
			m.Spinner, cmd = m.Spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case prowlarr.IndexersFetchedMsg:
		m.PendingIndexers = len(msg.Indexers)
		m.AllResults = []prowlarr.TorrentResult{}
		for _, idx := range msg.Indexers {
			cmds = append(cmds, m.Client.FetchSingleIndexerCmd(msg.Query, idx.ID))
		}

	case prowlarr.SearchResultMsg:
		m.PendingIndexers--

		if len(msg.Items) > 0 {
			m.AllResults = append(m.AllResults, msg.Items...)

			sort.Slice(m.AllResults, func(i, j int) bool {
				return m.AllResults[i].Seeders > m.AllResults[j].Seeders
			})

			var listItems []list.Item
			for _, r := range m.AllResults {
				listItems = append(listItems, r)
			}
			m.List.SetItems(listItems)

			if m.State == StateLoading {
				m.State = StateList
			}
		} else if m.PendingIndexers == 0 && len(m.AllResults) == 0 {
			m.Err = fmt.Errorf("no seeded torrents found across any indexers")
		}

	case prowlarr.ErrMsg:
		m.Err = msg.Err
	}

	switch m.State {
	case StateSearchInput:
		m.TextInput, cmd = m.TextInput.Update(msg)
		cmds = append(cmds, cmd)
	case StateList:
		m.List, cmd = m.List.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
