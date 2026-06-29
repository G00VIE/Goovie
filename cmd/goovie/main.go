package main

import (
	"fmt"
	"os"
	"strings"

	"bubble-stream/internal/assets"
	"bubble-stream/internal/player"
	"bubble-stream/internal/tui"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

func main() {
	player.InitProxy()
	ti := textinput.New()
	ti.Placeholder = "Search query..."
	ti.Focus()

	imgMovies, errM := tui.LoadPNG("logos/deadpool.png")
	if errM != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load deadpool.png: %v\n", errM)
	}
	imgTVShows, errT := tui.LoadPNG("logos/hisenberg.png")
	if errT != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load hisenberg.png: %v\n", errT)
	}
	imgAnime, errA := tui.LoadPNG("logos/itadori.png")
	if errA != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load itadori.png: %v\n", errA)
	}

	commonWidths := []int{30, 40, 50, 60}

	var cachedTitleLines []string
	if fANSI, errANSI := assets.EmbeddedFiles.Open("font/ANSI Compact.flf"); errANSI == nil {
		cachedTitleLines = strings.Split(figure.NewFigureWithFont("SELECT GENRE", fANSI, true).String(), "\n")
		fANSI.Close()
	} else {
		cachedTitleLines = strings.Split(figure.NewFigure("SELECT GENRE", "rowancap", true).String(), "\n")
	}
	var sb strings.Builder
	for _, line := range cachedTitleLines {
		if strings.TrimRight(line, " \t") != "" {
			sb.WriteString(line + "\n")
		}
	}
	cachedTitle := tui.TitleStyle.Render(sb.String())

	cachedFrontTitleLines := strings.Split(figure.NewFigure("GOOVIE", "graffiti", true).String(), "\n")
	var sbFront strings.Builder
	for _, line := range cachedFrontTitleLines {
		if strings.TrimRight(line, " \t") != "" {
			sbFront.WriteString(line + "\n")
		}
	}
	cachedFrontTitle := tui.TitleStyle.Render(sbFront.String())

	cacheMovies := tui.PreRenderCache(imgMovies, commonWidths)
	cacheTV := tui.PreRenderCache(imgTVShows, commonWidths)
	cacheAnime := tui.PreRenderCache(imgAnime, commonWidths)

	setupTi := textinput.New()
	setupTi.Placeholder = "Paste API Key here..."

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	m := tui.NewModel(ti, setupTi, s, imgMovies, imgTVShows, imgAnime, cachedTitle, cachedFrontTitle, cacheMovies, cacheTV, cacheAnime)

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime structural panic: %v\n", err)
		os.Exit(1)
	}
}
