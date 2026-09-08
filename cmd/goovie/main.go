package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bubble-stream/internal/assets"
	"bubble-stream/internal/bittorrent"
	"bubble-stream/internal/player"
	"bubble-stream/internal/sysutil"
	"bubble-stream/internal/tui"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

func main() {
	sysutil.PurgeAllTempData()
	defer func() {
		bittorrent.CloseGlobalEngine()
		player.CloseProxy()
		sysutil.StopFlareSolverr()
		sysutil.PurgeAllTempData()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		bittorrent.CloseGlobalEngine()
		player.CloseProxy()
		sysutil.StopFlareSolverr()
		sysutil.PurgeAllTempData()
		os.Exit(0)
	}()

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

	commonWidths := []int{18, 22, 26, 30, 40, 50, 60}

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
	s.Spinner = spinner.Spinner{
		Frames: []string{
			"⠄                       ",
			"⠤                       ",
			"⠴                       ",
			"⠼                       ",
			"⠼⠁                      ",
			"⠼⠉                      ",
			"⠼⠙                      ",
			"⠼⠹                      ",
			"⠼⠹⠄                     ",
			"⠼⠹⠤                     ",
			"⠼⠹⠴                     ",
			"⠼⠹⠼                     ",
			"⠼⠹⠼⠁                    ",
			"⠼⠹⠼⠉                    ",
			"⠼⠹⠼⠙                    ",
			"⠼⠹⠼⠹                    ",
			"⠼⠹⠼⠹⠄                   ",
			"⠸⠹⠼⠹⠤                   ",
			"⠘⠹⠼⠹⠴                   ",
			"⠈⠹⠼⠹⠼                   ",
			" ⠹⠼⠹⠼⠁                  ",
			" ⠸⠼⠹⠼⠉                  ",
			" ⠰⠼⠹⠼⠙                  ",
			" ⠠⠼⠹⠼⠹                  ",
			"  ⠼⠹⠼⠹⠄                 ",
			"  ⠸⠹⠼⠹⠤                 ",
			"  ⠘⠹⠼⠹⠴                 ",
			"  ⠈⠹⠼⠹⠼                 ",
			"   ⠹⠼⠹⠼⠁                ",
			"   ⠸⠼⠹⠼⠉                ",
			"   ⠰⠼⠹⠼⠙                ",
			"   ⠠⠼⠹⠼⠹                ",
			"    ⠼⠹⠼⠹⠄               ",
			"    ⠸⠹⠼⠹⠤               ",
			"    ⠘⠹⠼⠹⠴               ",
			"    ⠈⠹⠼⠹⠼               ",
			"     ⠹⠼⠹⠼⠁              ",
			"     ⠸⠼⠹⠼⠉              ",
			"     ⠰⠼⠹⠼⠙              ",
			"     ⠠⠼⠹⠼⠹              ",
			"      ⠼⠹⠼⠹⠄             ",
			"      ⠸⠹⠼⠹⠤             ",
			"      ⠘⠹⠼⠹⠴             ",
			"      ⠈⠹⠼⠹⠼             ",
			"       ⠹⠼⠹⠼⠁            ",
			"       ⠸⠼⠹⠼⠉            ",
			"       ⠰⠼⠹⠼⠙            ",
			"       ⠠⠼⠹⠼⠹            ",
			"        ⠼⠹⠼⠹⠄           ",
			"        ⠸⠹⠼⠹⠤           ",
			"        ⠘⠹⠼⠹⠴           ",
			"        ⠈⠹⠼⠹⠼           ",
			"         ⠹⠼⠹⠼⠁          ",
			"         ⠸⠼⠹⠼⠉          ",
			"         ⠰⠼⠹⠼⠙          ",
			"         ⠠⠼⠹⠼⠹          ",
			"          ⠼⠹⠼⠹⠄         ",
			"          ⠸⠹⠼⠹⠤         ",
			"          ⠘⠹⠼⠹⠴         ",
			"          ⠈⠹⠼⠹⠼         ",
			"           ⠹⠼⠹⠼⠁        ",
			"           ⠸⠼⠹⠼⠉        ",
			"           ⠰⠼⠹⠼⠙        ",
			"           ⠠⠼⠹⠼⠹        ",
			"            ⠼⠹⠼⠹⠄       ",
			"            ⠸⠹⠼⠹⠤       ",
			"            ⠘⠹⠼⠹⠴       ",
			"            ⠈⠹⠼⠹⠼       ",
			"             ⠹⠼⠹⠼⠁      ",
			"             ⠸⠼⠹⠼⠉      ",
			"             ⠰⠼⠹⠼⠙      ",
			"             ⠠⠼⠹⠼⠹      ",
			"              ⠼⠹⠼⠹⠄     ",
			"              ⠸⠹⠼⠹⠤     ",
			"              ⠘⠹⠼⠹⠴     ",
			"              ⠈⠹⠼⠹⠼     ",
			"               ⠹⠼⠹⠼⠁    ",
			"               ⠸⠼⠹⠼⠉    ",
			"               ⠰⠼⠹⠼⠙    ",
			"               ⠠⠼⠹⠼⠹    ",
			"                ⠼⠹⠼⠹⠄   ",
			"                ⠸⠹⠼⠹⠤   ",
			"                ⠘⠹⠼⠹⠴   ",
			"                ⠈⠹⠼⠹⠼   ",
			"                 ⠹⠼⠹⠼⠁  ",
			"                 ⠸⠼⠹⠼⠉  ",
			"                 ⠰⠼⠹⠼⠙  ",
			"                 ⠠⠼⠹⠼⠹  ",
			"                  ⠼⠹⠼⠹⠄ ",
			"                  ⠸⠹⠼⠹⠤ ",
			"                  ⠘⠹⠼⠹⠴ ",
			"                  ⠈⠹⠼⠹⠼ ",
			"                   ⠹⠼⠹⠼⠁",
			"                   ⠸⠼⠹⠼⠉",
			"                   ⠰⠼⠹⠼⠙",
			"                   ⠠⠼⠹⠼⠹",
			"                    ⠼⠹⠼⠹",
			"                    ⠸⠹⠼⠹",
			"                    ⠘⠹⠼⠹",
			"                    ⠈⠹⠼⠹",
			"                     ⠹⠼⠹",
			"                     ⠸⠼⠹",
			"                     ⠰⠼⠹",
			"                     ⠠⠼⠹",
			"                      ⠼⠹",
			"                      ⠸⠹",
			"                      ⠘⠹",
			"                      ⠈⠹",
			"                       ⠹",
			"                       ⠸",
			"                       ⠰",
			"                       ⠠",
		},
		FPS: time.Second / 15,
	}
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	m := tui.NewModel(ti, setupTi, s, imgMovies, imgTVShows, imgAnime, cachedTitle, cachedFrontTitle, cacheMovies, cacheTV, cacheAnime)

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime structural panic: %v\n", err)
		bittorrent.CloseGlobalEngine()
		player.CloseProxy()
		sysutil.PurgeAllTempData()
		os.Exit(1)
	}
}
