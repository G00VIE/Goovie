package main

import (
	"fmt"
	"os"

	"bubble-stream/internal/config"
	"bubble-stream/internal/player"
	"bubble-stream/internal/prowlarr"
	"bubble-stream/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg := config.Load()
	client := prowlarr.NewClient(cfg.ProwlarrURL, cfg.ProwlarrAPIKey)

	p := tea.NewProgram(tui.NewModel(client), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	finalModel := m.(tui.Model)
	if finalModel.StreamURL != "" {
		if err := player.Stream(finalModel.StreamURL); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	}
}
