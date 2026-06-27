package tui

import (
	"image"

	"bubble-stream/internal/player"
	"bubble-stream/internal/prowlarr"
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
	StateAnikotoServerSelect
)

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
	cursor          int
	animeTypeFilter string
	imgMovies       image.Image
	imgTVShows      image.Image
	imgAnime        image.Image

	cinemetaMovies []prowlarr.CinemetaMovie
	tvShows        []prowlarr.TVMazeShow
	tvSeasons      []prowlarr.TVMazeSeason
	tvFiles        []string
	selectedShow   string
	selectedSeason int
	selectedMagnet string
	groups         []prowlarr.IndexerGroup
	totalMovies    int
	qualityFilter  string
	pendingSearch  int
	finishCounter  int

	animeList         []prowlarr.JikanAnime
	anikotoShows      []player.ShowResult
	anikotoEpisodes   []player.EpisodeResult
	anikotoServers    []player.ServerResult
	anikotoWatchURL   string
	anikotoSelectedEp player.EpisodeResult
	anikotoMode       string

	cachedTitle      string
	cachedFrontTitle string
	cacheMovies      map[int]map[bool]string
	cacheTV          map[int]map[bool]string
	cacheAnime       map[int]map[bool]string
	activeWCell      int
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

// NewModel handles creation with existing state. It will be initialized from main.go
func NewModel(ti textinput.Model, imgMovies image.Image, imgTVShows image.Image, imgAnime image.Image, cachedTitle string, cachedFrontTitle string, cacheMovies map[int]map[bool]string, cacheTV map[int]map[bool]string, cacheAnime map[int]map[bool]string) Model {
	return Model{
		state:            StateFrontPage,
		cursor:           1, // Default to TV SHOWS
		textInput:        ti,
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
