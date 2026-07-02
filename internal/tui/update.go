package tui

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"bubble-stream/internal/config"
	"bubble-stream/internal/player"
	"bubble-stream/internal/prowlarr"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(prowlarr.ErrMsg); ok {
		m.err = msg.Err
		m.state = StateModeSelect
		return m, nil
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		return m, cmd

	case LoadingTickMsg:
		m.loadingPhrase = LoadingPhrases[rand.Intn(len(LoadingPhrases))]
		return m, tickLoadingText()

	case player.PlayerFinishedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.state = StateModeSelect
		} else {
			if m.isAsian {
				if m.isTVShow {
					m.state = StateAsianEpSelect
				} else {
					m.state = StateAsianShowSelect
				}
			} else if m.isAnime {
				m.state = StateAnikotoEpSelect
			} else if m.isTVShow {
				m.state = StateTVFileSelect
			} else {
				m.state = StateList
			}
		}
		return m, nil

	case APIKeyStatusMsg:
		if msg.Found {
			m.state = StateFrontPage
			return m, nil
		}
		m.state = StateSetupAPIKey
		m.setupInput.Focus()
		return m, textinput.Blink

	case tea.WindowSizeMsg:
		m.terminalHeight = msg.Height
		m.terminalWidth = msg.Width

		spacing := 6
		wCell := (m.terminalWidth - (2 * spacing) - 4) / 3
		maxHeight := m.terminalHeight / 2
		if wCell > 2*maxHeight {
			wCell = 2 * maxHeight
		}
		if wCell > 65 {
			wCell = 65
		}
		if wCell < 25 {
			wCell = 25
		}

		bestDist := 9999
		bestW := 50
		for k := range m.cacheMovies {
			dist := k - wCell
			if dist < 0 {
				dist = -dist
			}
			if dist < bestDist {
				bestDist = dist
				bestW = k
			}
		}
		m.activeWCell = bestW

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "up":
			if m.state != StateModeSelect && m.cursor > 0 {
				m.cursor--
			}

		case "down":
			if m.state == StateAnimeTypeSelect && m.cursor < len(config.AnimeTypeOptions)-1 {
				m.cursor++
			}
			if m.state == StateMovieSelect && m.cursor < len(m.cinemetaMovies)-1 {
				m.cursor++
			}
			if m.state == StateTVShowSelect && m.cursor < len(m.tvShows)-1 {
				m.cursor++
			}
			if m.state == StateTVSeasonSelect && m.cursor < len(m.tvSeasons)-1 {
				m.cursor++
			}
			if m.state == StateQuality && m.cursor < len(config.QualityOptions)-1 {
				m.cursor++
			}
			if m.state == StateList && m.cursor < m.totalMovies-1 {
				m.cursor++
			}
			if m.state == StateTVFileSelect && m.cursor < len(m.tvFiles)-1 {
				m.cursor++
			}
			// Anime specific downs
			if m.state == StateAnimeSelect && m.cursor < len(m.animeList)-1 {
				m.cursor++
			}
			if m.state == StateAnikotoShowSelect && m.cursor < len(m.anikotoShows)-1 {
				m.cursor++
			}
			if m.state == StateAnikotoEpSelect && m.cursor < len(m.anikotoEpisodes)-1 {
				m.cursor++
			}
			if m.state == StateAnikotoModeSelect && m.cursor < 1 { // 0: Sub, 1: Dub
				m.cursor++
			}
			if m.state == StateAsianShowSelect && m.cursor < len(m.asianShows)-1 {
				m.cursor++
			}
			if m.state == StateOriginSelect && m.cursor < 1 { // 0: Western, 1: Asian
				m.cursor++
			}
			if m.state == StateAsianEpSelect && m.cursor < len(m.asianEpisodes)-1 {
				m.cursor++
			}

		case "left":
			if m.state == StateModeSelect {
				if m.cursor == 0 {
					m.cursor = 2
				} else {
					m.cursor--
				}
			}
		case "right":
			if m.state == StateModeSelect {
				if m.cursor == 2 {
					m.cursor = 0
				} else {
					m.cursor++
				}
			}
		case "enter":
			switch m.state {
			case StateSetupAPIKey:
				key := strings.TrimSpace(m.setupInput.Value())
				if key != "" {
					config.SaveConfig(key)
					m.state = StateFrontPage
				}
				return m, nil

			case StateFrontPage:
				m.state = StateModeSelect
				return m, nil

			case StateModeSelect:
				if m.cursor == 0 {
					m.isTVShow, m.isAnime, m.isAsian = false, false, false
					m.state = StateOriginSelect
				} else if m.cursor == 1 {
					m.isTVShow, m.isAnime, m.isAsian = true, false, false
					m.state = StateOriginSelect
				} else if m.cursor == 2 {
					m.isTVShow, m.isAnime, m.isAsian = false, true, false
					m.state = StateAnimeTypeSelect
				}
				m.cursor = 0
				return m, nil

			case StateOriginSelect:
				if m.cursor == 0 {
					m.isAsian = false
				} else {
					m.isAsian = true
				}
				m.cursor = 0
				m.state = StateSearch
				return m, nil

			case StateAnimeTypeSelect:
				m.animeTypeFilter = config.AnimeTypeOptions[m.cursor]
				m.state = StateSearch
				m.cursor = 0
				return m, nil

			case StateSearch:
				if m.textInput.Value() != "" {
					rawInput := strings.TrimSpace(m.textInput.Value())
					m.cursor = 0
					m.state = StateLoading
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
					if m.isAsian {
						return m, tea.Batch(m.loadingSpinner.Tick, player.FetchAsianShowsCmd(rawInput, m.isTVShow))
					} else if m.isAnime {
						return m, tea.Batch(m.loadingSpinner.Tick, prowlarr.FetchAnime(rawInput, m.animeTypeFilter))
					} else if m.isTVShow {
						return m, tea.Batch(m.loadingSpinner.Tick, prowlarr.FetchTVShows(rawInput))
					} else {
						return m, tea.Batch(m.loadingSpinner.Tick, prowlarr.FetchCinemetaMovies(rawInput))
					}
				}

			case StateAsianShowSelect:
				if len(m.asianShows) > 0 {
					m.selectedAsianShow = m.asianShows[m.cursor]
					m.cursor = 0
					m.state = StateLoading
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
					return m, tea.Batch(m.loadingSpinner.Tick, player.FetchAsianEpisodesCmd(m.selectedAsianShow))
				}

			case StateAsianEpSelect:
				if len(m.asianEpisodes) > 0 {
					chosenEp := m.asianEpisodes[m.cursor]
					m.cursor = 0
					m.state = StateLoading
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
					return m, tea.Batch(m.loadingSpinner.Tick, player.FetchAsianStreamCmd(chosenEp.Link))
				}

			// --- Movie Logic Flow (Cinemeta) ---
			case StateMovieSelect:
				if len(m.cinemetaMovies) > 0 {
					chosen := m.cinemetaMovies[m.cursor]
					year := chosen.Year
					if year == "" {
						year = chosen.ReleaseInfo
					}
					if len(year) > 4 {
						year = year[:4]
					}
					if year != "" {
						m.currentQuery = fmt.Sprintf("%s %s", chosen.Name, year)
					} else {
						m.currentQuery = chosen.Name
					}
					m.cursor = 0
					m.state = StateQuality
					return m, nil
				}

			// --- Anime Logic Flow (Anikoto Scraper) ---
			case StateAnimeSelect:
				if len(m.animeList) > 0 {
					chosen := m.animeList[m.cursor]
					m.cursor = 0
					m.state = StateLoading
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
					return m, tea.Batch(m.loadingSpinner.Tick, player.FetchAnikotoShowsCmd(chosen.Title))
				}

			case StateAnikotoShowSelect:
				if len(m.anikotoShows) > 0 {
					chosen := m.anikotoShows[m.cursor]
					m.cursor = 0
					m.state = StateLoading
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
					return m, tea.Batch(m.loadingSpinner.Tick, player.FetchAnikotoEpisodesCmd(chosen.Slug))
				}
			case StateAnikotoEpSelect:
				if len(m.anikotoEpisodes) > 0 {
					m.anikotoSelectedEp = m.anikotoEpisodes[m.cursor]
					m.cursor = 0
					m.state = StateAnikotoModeSelect
					return m, nil
				}
			case StateAnikotoModeSelect:
				if m.cursor == 0 {
					m.anikotoMode = "sub"
				} else {
					m.anikotoMode = "dub"
				}
				m.cursor = 0
				m.state = StateLoading
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
				return m, tea.Batch(m.loadingSpinner.Tick, player.RaceAnikotoStreamsCmd(m.anikotoSelectedEp.Token, m.anikotoMode, m.anikotoWatchURL))

			// --- Western TV Logic Flow ---
			case StateTVShowSelect:
				if len(m.tvShows) > 0 {
					chosen := m.tvShows[m.cursor]
					m.selectedShow = chosen.Name
					m.cursor = 0
					m.state = StateLoading
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
					return m, tea.Batch(m.loadingSpinner.Tick, prowlarr.FetchTVSeasons(chosen.ID))
				}
			case StateTVSeasonSelect:
				if len(m.tvSeasons) > 0 {
					chosen := m.tvSeasons[m.cursor]
					m.selectedSeason = chosen.Number
					m.selectedSeasonID = chosen.ID
					m.currentQuery = fmt.Sprintf("%s S%02d", m.selectedShow, m.selectedSeason)
					m.cursor = 0
					m.state = StateQuality
					return m, nil
				}
			case StateQuality:
				m.qualityFilter = config.QualityOptions[m.cursor]
				m.state = StateList
				m.groups = []prowlarr.IndexerGroup{}
				m.totalMovies = 0
				m.cursor = 0
				m.finishCounter = 0
				return m, prowlarr.FetchIndexers
			case StateList:
				if m.totalMovies == 0 {
					return m, nil
				}
				var chosen prowlarr.ProwlarrResult
				movieCounter := 0
				found := false
				for _, g := range m.groups {
					if g.Status == "No Matches" || (g.Status == "Complete" && len(g.Results) == 0) {
						continue
					}
					for _, res := range g.Results {
						if movieCounter == m.cursor {
							chosen = res
							found = true
							break
						}
						movieCounter++
					}
					if found {
						break
					}
				}

				resolvedMagnet := prowlarr.ResolveProxyLink(chosen, m.isAnime)
				if resolvedMagnet == "" {
					return m, func() tea.Msg { return prowlarr.ErrMsg{Err: fmt.Errorf("failed to unmask link.")} }
				}

				if m.isTVShow {
					m.selectedMagnet = resolvedMagnet
					m.state = StateLoading
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
					m.tvEpisodes = nil // Clear old episodes
					m.tvFiles = nil    // Clear old files
					return m, tea.Batch(m.loadingSpinner.Tick, prowlarr.FetchTVFiles(resolvedMagnet), prowlarr.FetchTVEpisodes(m.selectedSeasonID))
				}

				m.state = StateLoadingTorrent
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
				return m, tea.Batch(m.loadingSpinner.Tick, player.LaunchPlayer(resolvedMagnet, "", "", ""))

			case StateTVFileSelect:
				if len(m.tvFiles) > 0 {
					chosenLine := m.tvFiles[m.cursor]
					fields := strings.Fields(chosenLine)
					if len(fields) > 0 {
						targetIndex := fields[0]
						m.state = StateLoadingTorrent
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
						return m, tea.Batch(m.loadingSpinner.Tick, player.LaunchPlayer(m.selectedMagnet, targetIndex, "", ""))
					}
				}
			}
		case "backspace":
			if m.state == StateSearch {
				if len(m.textInput.Value()) == 0 {
					if m.isAnime {
						m.state = StateAnimeTypeSelect
						m.cursor = 0
					} else {
						m.state = StateOriginSelect
						m.cursor = 0
					}
				}
			} else if m.state == StateTVFileSelect && len(m.tvFileSearch) > 0 {
				m.tvFileSearch = m.tvFileSearch[:len(m.tvFileSearch)-1]
				m = m.updateTVFileCursor()
			} else if (m.state == StateMovieSelect || m.state == StateTVShowSelect || m.state == StateAnimeSelect || m.state == StateAnikotoShowSelect || m.state == StateAnikotoEpSelect || m.state == StateAsianShowSelect || m.state == StateAsianEpSelect) && len(m.dbMatchSearch) > 0 {
				m.dbMatchSearch = m.dbMatchSearch[:len(m.dbMatchSearch)-1]
				m = m.updateDBMatchCursor()
			} else {
				m.dbMatchSearch = ""
				m.tvFileSearch = ""
				switch m.state {
				case StateOriginSelect:
					m.state = StateModeSelect
					if m.isTVShow {
						m.cursor = 1
					} else {
						m.cursor = 0
					}
				case StateAnimeTypeSelect:
					m.state = StateModeSelect
					m.cursor = 2
				case StateMovieSelect, StateTVShowSelect, StateAnimeSelect:
					m.state = StateSearch
					m.cursor = 0
				case StateQuality:
					if m.isTVShow {
						m.state = StateTVSeasonSelect
					} else {
						m.state = StateMovieSelect
					}
					m.cursor = 0
				case StateList:
					m.state = StateQuality
					m.cursor = 0
				case StateTVSeasonSelect:
					m.state = StateTVShowSelect
					m.cursor = 0
				case StateTVFileSelect:
					m.state = StateList
					m.cursor = 0
				case StateAnikotoShowSelect:
					m.state = StateAnimeSelect
					m.cursor = 0
				case StateAnikotoEpSelect:
					m.state = StateAnikotoShowSelect
					m.cursor = 0
				case StateAnikotoModeSelect:
					m.state = StateAnikotoEpSelect
					m.cursor = 0
				case StateAsianShowSelect:
					m.state = StateSearch
					m.cursor = 0
				case StateAsianEpSelect:
					m.state = StateAsianShowSelect
					m.cursor = 0
				}
			}
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if m.state == StateTVFileSelect {
				if len(m.tvFileSearch) < 4 { // Prevent infinite typing
					m.tvFileSearch += msg.String()
					m = m.updateTVFileCursor()
				}
			} else if m.state == StateMovieSelect || m.state == StateTVShowSelect || m.state == StateAnimeSelect || m.state == StateAnikotoShowSelect || m.state == StateAnikotoEpSelect || m.state == StateAsianShowSelect || m.state == StateAsianEpSelect {
				if len(m.dbMatchSearch) < 4 {
					m.dbMatchSearch += msg.String()
					m = m.updateDBMatchCursor()
				}
			}
		}

	case prowlarr.CinemetaMsg:
		m.cinemetaMovies = msg
		m.state = StateMovieSelect
		m.cursor = 0
		return m, nil
	case prowlarr.AnimeMsg:
		m.animeList = msg
		m.state = StateAnimeSelect
		m.cursor = 0
		return m, nil
	case player.AnikotoShowsMsg:
		m.anikotoShows = msg
		m.state = StateAnikotoShowSelect
		m.cursor = 0
		m.dbMatchSearch = ""
		return m, nil
	case player.AnikotoEpsMsg:
		m.anikotoEpisodes = msg.Eps
		m.anikotoWatchURL = msg.WatchURL
		m.state = StateAnikotoEpSelect
		m.cursor = 0
		m.dbMatchSearch = ""
		return m, nil

	case player.AnikotoStreamMsg:
		proxyURL := player.GlobalProxy.Register(msg.M3u8URL, msg.Referer)
		m.state = StateLoadingTorrent
					m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
		return m, tea.Batch(m.loadingSpinner.Tick, player.LaunchPlayer(proxyURL, "", msg.Referer, msg.SubtitleURL))

	case player.AsianShowsMsg:
		m.asianShows = msg
		m.state = StateAsianShowSelect
		m.cursor = 0
		m.dbMatchSearch = ""
		return m, nil

	case player.AsianEpisodesMsg:
		if msg.Type == "movie" {
			m.state = StateLoading
			m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
			return m, tea.Batch(m.loadingSpinner.Tick, player.FetchAsianStreamCmd(msg.WatchURL))
		}
		m.asianEpisodes = msg.Eps
		m.asianWatchURL = msg.WatchURL
		m.state = StateAsianEpSelect
		m.cursor = 0
		m.dbMatchSearch = ""
		return m, nil

	case player.AsianStreamMsg:
		m.state = StateLoadingTorrent
		m.loadingSpinner = spinner.New(spinner.WithSpinner(m.loadingSpinner.Spinner), spinner.WithStyle(m.loadingSpinner.Style))
		return m, tea.Batch(m.loadingSpinner.Tick, player.LaunchPlayer(msg.StreamURL, "", msg.Referer, msg.SubtitleURL))

	case prowlarr.TvShowsMsg:
		m.tvShows = msg
		m.state = StateTVShowSelect
		m.cursor = 0
		return m, nil
	case prowlarr.TvSeasonsMsg:
		m.tvSeasons = msg
		m.state = StateTVSeasonSelect
		m.cursor = 0
		return m, nil
	case prowlarr.TvEpisodesMsg:
		m.tvEpisodes = msg
		if len(m.tvFiles) > 0 {
			m.tvFiles = simplifyTVFiles(m.tvFiles, m.tvEpisodes)
		}
		return m, nil
	case prowlarr.TvFilesMsg:
		m.tvFiles = msg
		if len(m.tvEpisodes) > 0 {
			m.tvFiles = simplifyTVFiles(m.tvFiles, m.tvEpisodes)
		}
		m.state = StateTVFileSelect
		m.cursor = 0
		return m, nil

	case []prowlarr.Indexer:
		m.state = StateList
		m.pendingSearch = len(msg)
		var cmds []tea.Cmd
		for _, idx := range msg {
			m.groups = append(m.groups, prowlarr.IndexerGroup{ID: idx.ID, Name: idx.Name, Status: "Loading...", Order: 999})
			cmds = append(cmds, prowlarr.SearchSingleIndexer(m.currentQuery, idx.ID, m.qualityFilter, m.isTVShow, m.isAnime))
		}
		return m, tea.Batch(cmds...)

	case prowlarr.SearchResultMsg:
		m.pendingSearch--
		for i := range m.groups {
			if m.groups[i].ID == msg.IndexerID {
				seen := make(map[string]bool)
				var clean []prowlarr.ProwlarrResult
				for _, existing := range m.groups[i].Results {
					seen[strings.ToLower(existing.Title)] = true
					clean = append(clean, existing)
				}
				for _, newRes := range msg.Results {
					if !seen[strings.ToLower(newRes.Title)] {
						clean = append(clean, newRes)
						seen[strings.ToLower(newRes.Title)] = true
					}
				}
				m.groups[i].Results = clean
				sort.Slice(m.groups[i].Results, func(x, y int) bool {
					return m.groups[i].Results[x].Seeders > m.groups[i].Results[y].Seeders
				})

				if len(m.groups[i].Results) > 0 {
					if m.groups[i].Status != "Complete" {
						m.finishCounter++
						m.groups[i].Order = m.finishCounter
					}
					m.groups[i].Status = "Complete"
				} else if m.groups[i].Status != "Complete" {
					m.groups[i].Status = "No Matches"
					m.groups[i].Order = 999
				}
				break
			}
		}

		m.totalMovies = 0
		for _, g := range m.groups {
			if g.Status == "Complete" {
				m.totalMovies += len(g.Results)
			}
		}
		sort.Slice(m.groups, func(i, j int) bool {
			score := func(status string) int {
				if status == "Complete" {
					return 1
				}
				if status == "Loading..." {
					return 2
				}
				return 3
			}
			sI, sJ := score(m.groups[i].Status), score(m.groups[j].Status)
			if sI != sJ {
				return sI < sJ
			}
			if sI == 1 {
				return m.groups[i].Order < m.groups[j].Order
			}
			return m.groups[i].Name < m.groups[j].Name
		})
		return m, nil
	}

	var cmd tea.Cmd
	if m.state == StateSearch {
		m.textInput, cmd = m.textInput.Update(msg)
	} else if m.state == StateSetupAPIKey {
		m.setupInput, cmd = m.setupInput.Update(msg)
	}
	return m, cmd
}

func simplifyTVFiles(files []string, episodes []prowlarr.TVMazeEpisode) []string {
	var simplified []string
	epMap := make(map[int]string)
	for _, ep := range episodes {
		epMap[ep.Number] = ep.Name
	}

	re := regexp.MustCompile(`(?i)(?:s\d{1,2}e|e|ep)\s*0*(\d{1,3})\b`)

	for _, f := range files {
		fields := strings.SplitN(f, " ", 2)
		if len(fields) != 2 {
			simplified = append(simplified, f)
			continue
		}
		idxStr := fields[0]
		filename := fields[1]

		match := re.FindStringSubmatch(filename)
		if len(match) > 1 {
			epNum, _ := strconv.Atoi(match[1])
			if title, ok := epMap[epNum]; ok {
				newTitle := fmt.Sprintf("%s Episode %d: %s", idxStr, epNum, title)
				simplified = append(simplified, newTitle)
				continue
			}
		}
		simplified = append(simplified, f)
	}
	return simplified
}

func (m Model) updateTVFileCursor() Model {
	if m.tvFileSearch == "" {
		return m
	}
	searchStr := fmt.Sprintf("Episode %s:", m.tvFileSearch)
	for i, f := range m.tvFiles {
		if strings.Contains(f, searchStr) {
			m.cursor = i
			return m
		}
	}
	return m
}

func (m Model) updateDBMatchCursor() Model {
	if m.dbMatchSearch == "" {
		m.cursor = 0
		return m
	}
	idx, err := strconv.Atoi(m.dbMatchSearch)
	if err == nil {
		target := idx - 1 // 1-based index
		maxIdx := 0
		if m.state == StateMovieSelect {
			maxIdx = len(m.cinemetaMovies) - 1
		} else if m.state == StateTVShowSelect {
			maxIdx = len(m.tvShows) - 1
		} else if m.state == StateAnimeSelect {
			maxIdx = len(m.animeList) - 1
		} else if m.state == StateAnikotoShowSelect {
			maxIdx = len(m.anikotoShows) - 1
		} else if m.state == StateAnikotoEpSelect {
			maxIdx = len(m.anikotoEpisodes) - 1
		} else if m.state == StateAsianShowSelect {
			maxIdx = len(m.asianShows) - 1
		} else if m.state == StateAsianEpSelect {
			maxIdx = len(m.asianEpisodes) - 1
		}
		
		if target < 0 {
			target = 0
		}
		if target > maxIdx {
			target = maxIdx
		}
		m.cursor = target
	}
	return m
}
