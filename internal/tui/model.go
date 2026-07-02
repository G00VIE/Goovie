package tui

import (
	"image"
	"math/rand"
	"time"

	"bubble-stream/internal/config"
	"bubble-stream/internal/player"
	"bubble-stream/internal/prowlarr"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var LoadingPhrases = []string{
	"Brewing coffee...",
	"Making a sandwich...",
	"Petting the kittens...",
	"Watering the plants...",
	"Flipping the cassette...",
	"Dusting off the terminal...",
	"Warming up the server...",
	"I have a plan...",
	"You're a good man aurthur morgan...",
	"Seeking paleblood...",
	"Synchronizing...",
	"Calibrating the Animus...",
	"Praising the sun...",
	"Hey, you're finally awake...",
	"Loading graphical mods...",
	"I'm Pickle Rick!",
	"Wubba lubba dub dub...",
	"Bird up!",
	"Time to deliver a pizza ball...",
	"Who killed Hannibal?",
	"Bombing Israel?!",
	"Show me what you got...",
	"Legalize nuclear bombs...",
	"Waking up Manager Kim...",
	"Regressing to chapter 1...",
	"Hiding my power level...",
	"Awakening S-Class skills...",
	"Drawing a magic seal...",
	"Limit release...",
	"Domain expansion...",
}

type LoadingTickMsg time.Time

func tickLoadingText() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return LoadingTickMsg(t)
	})
}

const (
	StateFrontPage       = iota
	StateModeSelect
	StateOriginSelect
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
	StateAsianShowSelect
	StateAsianEpSelect

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
	imgDexter       image.Image
	imgAsian        image.Image

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

	isAsian           bool
	asianShows        []player.AsianShow
	asianEpisodes     []player.AsianEpisode
	selectedAsianShow player.AsianShow
	asianWatchURL     string

	setupInput     textinput.Model
	loadingSpinner spinner.Model

	cachedTitle      string
	cachedFrontTitle string
	cacheMovies      map[int]map[bool]string
	cacheTV          map[int]map[bool]string
	cacheAnime       map[int]map[bool]string
	activeWCell      int
	loadingPhrase    string
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.loadingSpinner.Tick, checkAPIKeyCmd(), tickLoadingText())
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
		loadingPhrase:    LoadingPhrases[rand.Intn(len(LoadingPhrases))],
	}
}
