package prowlarr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"bubble-stream/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

type ErrMsg struct{ Err error }
type SearchResultMsg struct {
	IndexerID int
	Results   []ProwlarrResult
}
type CinemetaMsg []CinemetaMovie
type TvShowsMsg []TVMazeShow
type TvSeasonsMsg []TVMazeSeason
type TvFilesMsg []string
type AnimeMsg []JikanAnime

// --- Western Movies (Cinemeta) ---
func FetchCinemetaMovies(query string) tea.Cmd {
	return func() tea.Msg {
		safeQuery := url.QueryEscape(strings.ToLower(query))
		targetUrl := fmt.Sprintf("https://v3-cinemeta.strem.io/catalog/movie/top/search=%s.json", safeQuery)

		client := http.Client{Timeout: 8 * time.Second}
		resp, err := client.Get(targetUrl)
		if err != nil || resp.StatusCode != 200 {
			return ErrMsg{fmt.Errorf("Cinemeta API movie lookup failed")}
		}
		defer resp.Body.Close()

		var catalog CinemetaCatalog
		json.NewDecoder(resp.Body).Decode(&catalog)
		return CinemetaMsg(catalog.Metas)
	}
}

// --- Western TV (TVMaze) ---
func FetchTVShows(query string) tea.Cmd {
	return func() tea.Msg {
		targetUrl := fmt.Sprintf("https://api.tvmaze.com/search/shows?q=%s", url.QueryEscape(query))
		resp, err := http.Get(targetUrl)
		if err != nil || resp.StatusCode != 200 {
			return ErrMsg{fmt.Errorf("TVMaze API show lookup failed")}
		}
		defer resp.Body.Close()
		var results []TVMazeSearchResult
		json.NewDecoder(resp.Body).Decode(&results)
		var shows []TVMazeShow
		for _, r := range results {
			shows = append(shows, r.Show)
		}
		return TvShowsMsg(shows)
	}
}

func FetchTVSeasons(showID int) tea.Cmd {
	return func() tea.Msg {
		targetUrl := fmt.Sprintf("https://api.tvmaze.com/shows/%d/seasons", showID)
		resp, err := http.Get(targetUrl)
		if err != nil || resp.StatusCode != 200 {
			return ErrMsg{fmt.Errorf("TVMaze API season lookup failed")}
		}
		defer resp.Body.Close()
		var seasons []TVMazeSeason
		json.NewDecoder(resp.Body).Decode(&seasons)
		return TvSeasonsMsg(seasons)
	}
}

// --- Torrent Indexing (Prowlarr) ---
func FetchIndexers() tea.Msg {
	apiUrl := fmt.Sprintf("%s/api/v1/indexer?apikey=%s", config.ProwlarrURL, config.ProwlarrAPIKey)
	resp, err := http.Get(apiUrl)
	if err != nil || resp.StatusCode != 200 {
		return ErrMsg{fmt.Errorf("unable to connect to local Prowlarr instance")}
	}
	defer resp.Body.Close()
	var indexers []Indexer
	json.NewDecoder(resp.Body).Decode(&indexers)
	return indexers
}

func SearchSingleIndexer(query string, indexerID int, quality string, isTVShow bool, isAnime bool) tea.Cmd {
	return func() tea.Msg {
		safeQuery := url.QueryEscape(query)
		apiUrl := fmt.Sprintf("%s/api/v1/search?query=%s&type=search&indexerIds=%d&apikey=%s",
			config.ProwlarrURL, safeQuery, indexerID, config.ProwlarrAPIKey)

		targetTag := ""
		words := strings.Fields(strings.ToLower(query))
		if isTVShow && len(words) > 0 {
			targetTag = words[len(words)-1]
		}

		client := http.Client{Timeout: 20 * time.Second}
		resp, err := client.Get(apiUrl)
		if err != nil || resp.StatusCode != 200 {
			return SearchResultMsg{IndexerID: indexerID, Results: []ProwlarrResult{}}
		}
		defer resp.Body.Close()

		var rawResults []ProwlarrResult
		json.NewDecoder(resp.Body).Decode(&rawResults)

		var cleanResults []ProwlarrResult
		for _, res := range rawResults {
			if res.Seeders >= config.MinimumSeeders {
				title := strings.ToLower(res.Title)
				if isTVShow && !isAnime && targetTag != "" {
					seasonNum := strings.TrimPrefix(targetTag, "s")
					seasonNumTrimmed := strings.TrimLeft(seasonNum, "0")
					if seasonNumTrimmed == "" {
						seasonNumTrimmed = "0"
					}

					seasonInt, _ := strconv.Atoi(seasonNum)
					nextSeasonTag := fmt.Sprintf("s%02d", seasonInt+1)
					nextSeasonStr := fmt.Sprintf("season %d", seasonInt+1)
					if strings.Contains(title, nextSeasonTag) || strings.Contains(title, nextSeasonStr) || strings.Contains(title, fmt.Sprintf("season %02d", seasonInt+1)) {
						continue
					}

					hasS01 := strings.Contains(title, targetTag)
					hasSeason1 := strings.Contains(title, "season "+seasonNumTrimmed)
					hasSeason01 := strings.Contains(title, "season "+seasonNum)
					if !hasS01 && !hasSeason1 && !hasSeason01 {
						continue
					}
					if strings.Contains(title, targetTag+"e") || strings.Contains(title, "episode") {
						continue
					}
				}
				cleanResults = append(cleanResults, res)
			}
		}
		sort.Slice(cleanResults, func(i, j int) bool { return cleanResults[i].Seeders > cleanResults[j].Seeders })
		return SearchResultMsg{IndexerID: indexerID, Results: cleanResults}
	}
}

func FetchTVFiles(magnet string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd")
			rawCmd := fmt.Sprintf(`cmd /c webtorrent.cmd "%s" --select`, magnet)
			cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: rawCmd}
		} else {
			cmd = exec.Command("webtorrent", magnet, "--select")
		}

		out, _ := cmd.CombinedOutput()
		outputStr := string(out)
		var files []string
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 0 && line[0] >= '0' && line[0] <= '9' {
				if !strings.HasPrefix(line, "To select") && !strings.Contains(line, "Example:") {
					files = append(files, line)
				}
			}
		}
		if len(files) == 0 {
			return ErrMsg{fmt.Errorf("failed to extract episodes")}
		}
		return TvFilesMsg(files)
	}
}

func FetchAnime(query string, animeType string) tea.Cmd {
	return func() tea.Msg {
		safeQuery := url.QueryEscape(query)
		targetUrl := fmt.Sprintf("https://api.jikan.moe/v4/anime?q=%s&sfw=true", safeQuery)

		if animeType != "All" && animeType != "" {
			targetUrl += fmt.Sprintf("&type=%s", strings.ToLower(animeType))
		}

		resp, err := http.Get(targetUrl)
		if err != nil || resp.StatusCode != 200 {
			return ErrMsg{fmt.Errorf("Jikan API anime lookup failed")}
		}
		defer resp.Body.Close()

		var results JikanResponse
		json.NewDecoder(resp.Body).Decode(&results)

		return AnimeMsg(results.Data)
	}
}

func ResolveProxyLink(res ProwlarrResult) string {
	var base string
	if res.InfoHash != "" {
		hash := strings.ToLower(strings.TrimSpace(res.InfoHash))
		if len(hash) == 40 {
			dn := url.QueryEscape(res.Title)
			base = fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", hash, dn)
		}
	}
	if base == "" && res.MagnetUri != "" && strings.HasPrefix(res.MagnetUri, "magnet:") {
		base = res.MagnetUri
	}
	if base == "" || !strings.HasPrefix(base, "magnet:") {
		return ""
	}
	for _, tr := range config.DefaultTrackers {
		encodedTr := url.QueryEscape(tr)
		if !strings.Contains(base, encodedTr) {
			base += "&tr=" + encodedTr
		}
	}
	return base
}
