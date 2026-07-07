package prowlarr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"bubble-stream/internal/config"
	"bubble-stream/internal/sysutil"
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

type TvEpisodesMsg []TVMazeEpisode

func FetchTVEpisodes(seasonID int) tea.Cmd {
	return func() tea.Msg {
		targetUrl := fmt.Sprintf("https://api.tvmaze.com/seasons/%d/episodes", seasonID)
		resp, err := http.Get(targetUrl)
		if err != nil || resp.StatusCode != 200 {
			return ErrMsg{fmt.Errorf("TVMaze API episode lookup failed")}
		}
		defer resp.Body.Close()
		var episodes []TVMazeEpisode
		json.NewDecoder(resp.Body).Decode(&episodes)
		return TvEpisodesMsg(episodes)
	}
}

// --- Torrent Indexing (Prowlarr) ---
func FetchIndexers() tea.Msg {
	apiUrl := fmt.Sprintf("%s/api/v1/indexer?apikey=%s", config.ProwlarrURL, config.ProwlarrAPIKey)
	client := &http.Client{Timeout: 15 * time.Second}

	var resp *http.Response
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err = client.Get(apiUrl)
		if err == nil && resp.StatusCode == 200 {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if attempt == 0 {
			time.Sleep(1 * time.Second)
		}
	}
	if err != nil {
		return ErrMsg{fmt.Errorf("unable to connect to local Prowlarr instance: %v", err)}
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return ErrMsg{fmt.Errorf("unable to connect to local Prowlarr instance (status %d)", resp.StatusCode)}
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
			sysutil.SetCmdLine(cmd, rawCmd)
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
					lowerLine := strings.ToLower(line)
					if strings.Contains(lowerLine, ".txt") || strings.Contains(lowerLine, ".jpg") || strings.Contains(lowerLine, ".png") || strings.Contains(lowerLine, ".srt") || strings.Contains(lowerLine, ".nfo") || strings.Contains(lowerLine, ".url") || strings.Contains(lowerLine, ".exe") {
						continue
					}
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

type AniListResponse struct {
	Data struct {
		Page struct {
			Media []struct {
				ID    int `json:"id"`
				Title struct {
					Romaji  string `json:"romaji"`
					English string `json:"english"`
					Native  string `json:"native"`
				} `json:"title"`
				Type       string `json:"type"`
				Format     string `json:"format"`
				SeasonYear int    `json:"seasonYear"`
				Episodes   int    `json:"episodes"`
			} `json:"media"`
		} `json:"Page"`
	} `json:"data"`
}

func FetchAnime(query string, animeType string) tea.Cmd {
	return func() tea.Msg {
		queryStr := `
		query ($search: String, $format: MediaFormat) {
			Page(page: 1, perPage: 10) {
				media(search: $search, type: ANIME, format: $format) {
					id
					title {
						romaji
						english
						native
					}
					type
					format
					seasonYear
					episodes
				}
			}
		}
		`

		variables := map[string]interface{}{
			"search": query,
		}

		if animeType != "All" && animeType != "" {
			apiType := strings.ToUpper(animeType)
			if apiType == "SERIES" {
				apiType = "TV"
			} else if apiType == "MOVIES" {
				apiType = "MOVIE"
			}
			variables["format"] = apiType
		}

		requestBody := map[string]interface{}{
			"query":     queryStr,
			"variables": variables,
		}
		
		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			return ErrMsg{fmt.Errorf("failed to prepare AniList request")}
		}

		req, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(jsonData))
		if err != nil {
			return ErrMsg{fmt.Errorf("failed to create AniList request")}
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			return ErrMsg{fmt.Errorf("AniList API lookup failed")}
		}
		defer resp.Body.Close()

		var results AniListResponse
		if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
			return ErrMsg{fmt.Errorf("failed to decode AniList response")}
		}

		var jikanAnimes []JikanAnime
		for _, m := range results.Data.Page.Media {
			title := m.Title.English
			if title == "" {
				title = m.Title.Romaji
			}
			jikanAnimes = append(jikanAnimes, JikanAnime{
				MalID:    m.ID, // Using AniList ID
				Title:    title,
				Type:     m.Format,
				Year:     m.SeasonYear,
				Episodes: m.Episodes,
			})
		}

		return AnimeMsg(jikanAnimes)
	}
}

func ResolveProxyLink(res ProwlarrResult, isAnime bool) string {
	var base string
	var debugLog string
	
	debugLog += fmt.Sprintf("=== Resolving Torrent ===\n")
	debugLog += fmt.Sprintf("Title: %s\n", res.Title)
	debugLog += fmt.Sprintf("Indexer: %s\n", res.Indexer)
	debugLog += fmt.Sprintf("InfoHash: %s\n", res.InfoHash)
	debugLog += fmt.Sprintf("MagnetUri: %s\n", res.MagnetUri)
	debugLog += fmt.Sprintf("DownloadUrl: %s\n", res.DownloadUrl)

	if res.InfoHash != "" {
		hash := strings.ToLower(strings.TrimSpace(res.InfoHash))
		if len(hash) == 40 {
			dn := url.QueryEscape(res.Title)
			base = fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", hash, dn)
			debugLog += fmt.Sprintf("-> Extracted magnet from InfoHash\n")
		}
	}
	if base == "" {
		if res.MagnetUri != "" && strings.HasPrefix(res.MagnetUri, "magnet:") {
			base = res.MagnetUri
			debugLog += fmt.Sprintf("-> Used MagnetUri\n")
		} else if res.DownloadUrl != "" {
			if strings.HasPrefix(res.DownloadUrl, "magnet:") {
				base = res.DownloadUrl
				debugLog += fmt.Sprintf("-> Used DownloadUrl (magnet)\n")
			} else {
				if isAnime {
					debugLog += fmt.Sprintf("-> Returning DownloadUrl (raw HTTP). Attempting to download locally...\n")
					localPath, err := downloadTorrentFile(res.DownloadUrl)
					if err != nil {
						debugLog += fmt.Sprintf("-> Failed to download locally: %v\n\n", err)
						writeDebugLog(debugLog)
						return ""
					}
					debugLog += fmt.Sprintf("-> Successfully downloaded to: %s\n\n", localPath)
					writeDebugLog(debugLog)
					return localPath
				} else {
					debugLog += fmt.Sprintf("-> Resolving HTTP redirect for magnet (Movie/TV Show)\n")
					client := &http.Client{
						CheckRedirect: func(req *http.Request, via []*http.Request) error {
							return http.ErrUseLastResponse
						},
						Timeout: 5 * time.Second,
					}
					resp, err := client.Get(res.DownloadUrl)
					if err == nil {
						defer resp.Body.Close()
						if resp.StatusCode >= 300 && resp.StatusCode < 400 {
							loc := resp.Header.Get("Location")
							if strings.HasPrefix(loc, "magnet:") {
								base = loc
								debugLog += fmt.Sprintf("-> Successfully extracted magnet from redirect Location\n")
							} else {
								debugLog += fmt.Sprintf("-> Redirect Location is not a magnet link\n")
							}
						} else {
							debugLog += fmt.Sprintf("-> Did not receive a redirect status code (Got %d)\n", resp.StatusCode)
						}
					} else {
						debugLog += fmt.Sprintf("-> Error during HTTP GET: %v\n", err)
					}
				}
			}
		}
	}
	if base == "" || !strings.HasPrefix(base, "magnet:") {
		debugLog += fmt.Sprintf("-> FAILED: base is empty or not a magnet link\n\n")
		writeDebugLog(debugLog)
		return ""
	}
	for _, tr := range config.DefaultTrackers {
		encodedTr := url.QueryEscape(tr)
		if !strings.Contains(base, encodedTr) {
			base += "&tr=" + encodedTr
		}
	}
	debugLog += fmt.Sprintf("-> FINAL URI: %s\n\n", base)
	writeDebugLog(debugLog)
	return base
}

func writeDebugLog(content string) {
	f, err := os.OpenFile("torrent_debug.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(content)
	}
}

func downloadTorrentFile(dlUrl string) (string, error) {
	resp, err := http.Get(dlUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("goovie_%d.torrent", time.Now().UnixNano()))
	out, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return tmpFile, err
}

