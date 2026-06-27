package tui

import (
	"fmt"
	"sort"
	"strings"

	"bubble-stream/internal/config"
	"bubble-stream/internal/player"
	"bubble-stream/internal/prowlarr"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(prowlarr.ErrMsg); ok {
		m.err = msg.Err
		m.state = StateModeSelect
		return m, nil
	}

	switch msg := msg.(type) {
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
			if m.state == StateAnikotoServerSelect && m.cursor < len(m.anikotoServers)-1 {
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
			case StateFrontPage:
				m.state = StateModeSelect
				return m, nil

			case StateModeSelect:
				if m.cursor == 0 {
					m.isTVShow, m.isAnime = false, false
					m.state = StateSearch
				} else if m.cursor == 1 {
					m.isTVShow, m.isAnime = true, false
					m.state = StateSearch
				} else if m.cursor == 2 {
					m.isAnime = true
					m.state = StateAnimeTypeSelect
				}
				m.cursor = 0
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
					if m.isAnime {
						return m, prowlarr.FetchAnime(rawInput, m.animeTypeFilter)
					} else if m.isTVShow {
						return m, prowlarr.FetchTVShows(rawInput)
					} else {
						return m, prowlarr.FetchCinemetaMovies(rawInput)
					}
				}

			// --- Movie Logic Flow (Cinemeta) ---
			case StateMovieSelect:
				if len(m.cinemetaMovies) > 0 {
					chosen := m.cinemetaMovies[m.cursor]
					year := chosen.Year
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
					return m, player.FetchAnikotoShowsCmd(chosen.Title)
				}

			case StateAnikotoShowSelect:
				if len(m.anikotoShows) > 0 {
					chosen := m.anikotoShows[m.cursor]
					m.cursor = 0
					m.state = StateLoading
					return m, player.FetchAnikotoEpisodesCmd(chosen.Slug)
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
				return m, player.FetchAnikotoServersCmd(m.anikotoSelectedEp.Token, m.anikotoMode, m.anikotoWatchURL)
			case StateAnikotoServerSelect:
				if len(m.anikotoServers) > 0 {
					chosen := m.anikotoServers[m.cursor]
					m.cursor = 0
					m.state = StateLoading
					return m, player.ResolveAnikotoStreamCmd(chosen.LinkID, m.anikotoWatchURL, m.anikotoMode)
				}

			// --- Western TV Logic Flow ---
			case StateTVShowSelect:
				if len(m.tvShows) > 0 {
					chosen := m.tvShows[m.cursor]
					m.selectedShow = chosen.Name
					m.cursor = 0
					m.state = StateLoading
					return m, prowlarr.FetchTVSeasons(chosen.ID)
				}
			case StateTVSeasonSelect:
				if len(m.tvSeasons) > 0 {
					chosen := m.tvSeasons[m.cursor]
					m.selectedSeason = chosen.Number
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

				resolvedMagnet := prowlarr.ResolveProxyLink(chosen)
				if resolvedMagnet == "" {
					return m, func() tea.Msg { return prowlarr.ErrMsg{Err: fmt.Errorf("failed to unmask link.")} }
				}

				if m.isTVShow {
					m.selectedMagnet = resolvedMagnet
					m.state = StateLoading
					return m, prowlarr.FetchTVFiles(resolvedMagnet)
				}

				return m, player.LaunchPlayer(resolvedMagnet, "", "")

			case StateTVFileSelect:
				if len(m.tvFiles) > 0 {
					chosenLine := m.tvFiles[m.cursor]
					fields := strings.Fields(chosenLine)
					if len(fields) > 0 {
						targetIndex := fields[0]
						return m, player.LaunchPlayer(m.selectedMagnet, targetIndex, "")
					}
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
		return m, nil
	case player.AnikotoEpsMsg:
		m.anikotoEpisodes = msg.Eps
		m.anikotoWatchURL = msg.WatchURL
		m.state = StateAnikotoEpSelect
		m.cursor = 0
		return m, nil
	case player.AnikotoServersMsg:
		m.anikotoServers = msg
		m.state = StateAnikotoServerSelect
		m.cursor = 0
		return m, nil
	case player.AnikotoStreamMsg:
		proxyURL := player.GlobalProxy.Register(msg.M3u8URL, msg.Referer)
		return m, player.LaunchPlayer(proxyURL, "", msg.Referer)

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
	case prowlarr.TvFilesMsg:
		m.tvFiles = msg
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
	}
	return m, cmd
}
