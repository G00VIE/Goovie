package tui

import (
	"image"

	"bubble-stream/internal/config"
	"bubble-stream/internal/player"
	"bubble-stream/internal/prowlarr"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	StateFrontPage       = iota
	StateModeSelect
	StateAnimeTypeSelect
	StateSearch
	StateLoading
	StateMovieSelect
	StateTVShowSelect
	StateTVSeasonSelect
	StateQuality
	StateList
	StateTVFileSelect
	StateAnimeSelect
	StateAnikotoShowSelect
	StateAnikotoEpSelect
	StateAnikotoModeSelect

	StateCheckingAPIKey
	StateSetupAPIKey
	StateLoadingTorrent
)

type APIKeyStatusMsg struct {
	Found bool
}

type renderRow struct {
	text     string
	isTarget bool
}

type Model struct {
	state           int
	terminalHeight  int
	terminalWidth   int
	err             error
	isTVShow        bool
	isAnime         bool
	textInput       textinput.Model
	currentQuery    string
	dbMatchSearch   string
	cursor          int
	animeTypeFilter string
	imgMovies       image.Image
	imgTVShows      image.Image
	imgAnime        image.Image

	cinemetaMovies []prowlarr.CinemetaMovie
	tvShows        []prowlarr.TVMazeShow
	tvSeasons      []prowlarr.TVMazeSeason
	tvEpisodes       []prowlarr.TVMazeEpisode
	tvFiles          []string
	tvFileSearch     string
	selectedShow     string
	selectedSeason   int
	selectedSeasonID int
	selectedMagnet   string
	groups         []prowlarr.IndexerGroup
	totalMovies    int
	qualityFilter  string
	pendingSearch  int
	finishCounter  int

	animeList         []prowlarr.JikanAnime
	anikotoShows      []player.ShowResult
	anikotoEpisodes   []player.EpisodeResult

	anikotoWatchURL   string
	anikotoSelectedEp player.EpisodeResult
	anikotoMode       string

	setupInput     textinput.Model
	loadingSpinner spinner.Model

	cachedTitle      string
	cachedFrontTitle string
	cacheMovies      map[int]map[bool]string
	cacheTV          map[int]map[bool]string
	cacheAnime       map[int]map[bool]string
	activeWCell      int
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.loadingSpinner.Tick, checkAPIKeyCmd())
}

func checkAPIKeyCmd() tea.Cmd {
	return func() tea.Msg {
		found := config.InitConfig()
		return APIKeyStatusMsg{Found: found}
	}
}

// NewModel handles creation with existing state. It will be initialized from main.go
func NewModel(ti textinput.Model, setupInput textinput.Model, s spinner.Model, imgMovies image.Image, imgTVShows image.Image, imgAnime image.Image, cachedTitle string, cachedFrontTitle string, cacheMovies map[int]map[bool]string, cacheTV map[int]map[bool]string, cacheAnime map[int]map[bool]string) Model {
	return Model{
		state:            StateCheckingAPIKey,
		cursor:           1, // Default to TV SHOWS
		textInput:        ti,
		setupInput:       setupInput,
		loadingSpinner:   s,
		imgMovies:        imgMovies,
		imgTVShows:       imgTVShows,
		imgAnime:         imgAnime,
		cachedTitle:      cachedTitle,
		cachedFrontTitle: cachedFrontTitle,
		cacheMovies:      cacheMovies,
		cacheTV:          cacheTV,
		cacheAnime:       cacheAnime,
		activeWCell:      50,
	}
}
