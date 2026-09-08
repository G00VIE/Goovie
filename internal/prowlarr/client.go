package prowlarr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"bubble-stream/internal/bittorrent"
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

func fetchWithRetry(targetUrl string) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	var resp *http.Response
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		var req *http.Request
		req, err = http.NewRequest("GET", targetUrl, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", config.UserAgent)
		req.Header.Set("Accept", "application/json")
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			return resp, nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		if attempt == 0 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if err != nil {
		return nil, err
	}
	if resp != nil {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	return nil, fmt.Errorf("request failed")
}

// --- Western Movies (Cinemeta) ---
func FetchCinemetaMovies(query string) tea.Cmd {
	return func() tea.Msg {
		safeQuery := url.QueryEscape(strings.ToLower(strings.TrimSpace(query)))
		targetUrl := fmt.Sprintf("https://v3-cinemeta.strem.io/catalog/movie/top/search=%s.json", safeQuery)

		resp, err := fetchWithRetry(targetUrl)
		if err != nil {
			return CinemetaMsg(nil)
		}
		defer resp.Body.Close()

		var catalog CinemetaCatalog
		json.NewDecoder(resp.Body).Decode(&catalog)
		return CinemetaMsg(catalog.Metas)
	}
}

// --- Western TV (TVMaze with Cinemeta fallback) ---
func FetchTVShows(query string) tea.Cmd {
	return func() tea.Msg {
		safeQuery := url.QueryEscape(strings.TrimSpace(query))
		targetUrl := fmt.Sprintf("https://api.tvmaze.com/search/shows?q=%s", safeQuery)
		resp, err := fetchWithRetry(targetUrl)
		if err == nil {
			defer resp.Body.Close()
			var results []TVMazeSearchResult
			if err := json.NewDecoder(resp.Body).Decode(&results); err == nil && len(results) > 0 {
				var shows []TVMazeShow
				for _, r := range results {
					shows = append(shows, r.Show)
				}
				return TvShowsMsg(shows)
			}
		}

		// Fallback to Cinemeta series catalog if TVMaze fails or returns no results
		cinemetaUrl := fmt.Sprintf("https://v3-cinemeta.strem.io/catalog/series/top/search=%s.json", safeQuery)
		cResp, cErr := fetchWithRetry(cinemetaUrl)
		if cErr == nil {
			defer cResp.Body.Close()
			var catalog CinemetaCatalog
			if err := json.NewDecoder(cResp.Body).Decode(&catalog); err == nil && len(catalog.Metas) > 0 {
				var shows []TVMazeShow
				for i, m := range catalog.Metas {
					year := m.Year
					if year == "" {
						year = m.ReleaseInfo
					}
					shows = append(shows, TVMazeShow{
						ID:        -(i + 1),
						Name:      m.Name,
						Premiered: year,
					})
				}
				return TvShowsMsg(shows)
			}
		}

		return TvShowsMsg([]TVMazeShow{})
	}
}

func FetchTVSeasons(showID int) tea.Cmd {
	return func() tea.Msg {
		if showID > 0 {
			targetUrl := fmt.Sprintf("https://api.tvmaze.com/shows/%d/seasons", showID)
			resp, err := fetchWithRetry(targetUrl)
			if err == nil {
				defer resp.Body.Close()
				var seasons []TVMazeSeason
				if err := json.NewDecoder(resp.Body).Decode(&seasons); err == nil && len(seasons) > 0 {
					return TvSeasonsMsg(seasons)
				}
			}
		}

		// Fallback: Default seasons 1-10 so user is never blocked
		var defaultSeasons []TVMazeSeason
		for s := 1; s <= 10; s++ {
			defaultSeasons = append(defaultSeasons, TVMazeSeason{ID: s, Number: s})
		}
		return TvSeasonsMsg(defaultSeasons)
	}
}

type TvEpisodesMsg []TVMazeEpisode

func FetchTVEpisodes(seasonID int) tea.Cmd {
	return func() tea.Msg {
		if seasonID > 0 {
			targetUrl := fmt.Sprintf("https://api.tvmaze.com/seasons/%d/episodes", seasonID)
			resp, err := fetchWithRetry(targetUrl)
			if err == nil {
				defer resp.Body.Close()
				var episodes []TVMazeEpisode
				if err := json.NewDecoder(resp.Body).Decode(&episodes); err == nil {
					return TvEpisodesMsg(episodes)
				}
			}
		}
		// Graceful degradation: never fail catastrophically for episode names
		return TvEpisodesMsg(nil)
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

// MatchesQuality checks whether the release title matches the requested quality filter.
func MatchesQuality(title string, quality string) bool {
	if quality == "" || strings.EqualFold(quality, "all") {
		return true
	}
	lower := strings.ToLower(title)
	switch strings.ToLower(quality) {
	case "720p", "720":
		return strings.Contains(lower, "720p") || strings.Contains(lower, "720")
	case "1080p", "1080":
		return strings.Contains(lower, "1080p") || strings.Contains(lower, "1080") || strings.Contains(lower, "fhd")
	case "4k", "2160p", "2160":
		return strings.Contains(lower, "4k") || strings.Contains(lower, "2160p") || strings.Contains(lower, "2160") || strings.Contains(lower, "uhd")
	default:
		return strings.Contains(lower, strings.ToLower(quality))
	}
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
				if !MatchesQuality(res.Title, quality) {
					continue
				}
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

func hasVideoExtension(line string) bool {
	clean := strings.TrimSpace(line)
	if strings.HasSuffix(clean, ")") {
		if idx := strings.LastIndex(clean, "("); idx > 0 {
			clean = strings.TrimSpace(clean[:idx])
		}
	}
	lower := strings.ToLower(clean)
	videoExts := []string{".mkv", ".mp4", ".avi", ".webm", ".m4v", ".ts", ".mov", ".wmv", ".flv"}
	for _, ext := range videoExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func IsJunkOrNonVideo(line string) bool {
	if !hasVideoExtension(line) {
		return true
	}

	lower := strings.ToLower(line)
	reJunk := regexp.MustCompile(`(?i)\b(?:ninite|codec|installer|updater|sample|featurettes?|extras?|bonus|trailers?|bloopers?|gag[._ -]*reels?|deleted[._ -]*scenes?)\b|\.(?:exe|bat|cmd|msi|vbs|sh|ps1|zip|rar|7z|txt|nfo|url|website|lnk|jpg|jpeg|png|srt)\b`)
	return reJunk.MatchString(lower)
}

func FetchTVFiles(magnet string) tea.Cmd {
	return func() tea.Msg {
		engine, err := bittorrent.GetGlobalEngine()
		if err != nil {
			return ErrMsg{fmt.Errorf("failed to start torrent engine: %w", err)}
		}

		items, err := engine.FetchFiles(magnet, 30*time.Second)
		if err != nil || len(items) == 0 {
			return ErrMsg{fmt.Errorf("failed to extract episodes: %v", err)}
		}

		var files []string
		for _, item := range items {
			line := item.FormatLine()
			if IsJunkOrNonVideo(line) {
				continue
			}
			files = append(files, line)
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

type kitsuAnimeResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			CanonicalTitle string `json:"canonicalTitle"`
			Titles         struct {
				En   string `json:"en"`
				EnJp string `json:"en_jp"`
			} `json:"titles"`
			Subtype      string `json:"subtype"`
			StartDate    string `json:"startDate"`
			EpisodeCount int    `json:"episodeCount"`
		} `json:"attributes"`
	} `json:"data"`
}

func fetchAnimeFromAniList(query string, animeType string) ([]JikanAnime, error) {
	queryStr := `
	query ($search: String, $formatIn: [MediaFormat]) {
		Page(page: 1, perPage: 10) {
			media(search: $search, type: ANIME, format_in: $formatIn) {
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
			variables["formatIn"] = []string{"TV", "TV_SHORT", "ONA"}
		} else if apiType == "MOVIES" {
			variables["formatIn"] = []string{"MOVIE"}
		} else {
			variables["formatIn"] = []string{apiType}
		}
	}

	requestBody := map[string]interface{}{
		"query":     queryStr,
		"variables": variables,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", config.UserAgent)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AniList HTTP error: %d", resp.StatusCode)
	}

	var results AniListResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	if len(results.Data.Page.Media) == 0 {
		return nil, fmt.Errorf("no AniList results")
	}

	var animes []JikanAnime
	for _, m := range results.Data.Page.Media {
		title := m.Title.English
		if title == "" {
			title = m.Title.Romaji
		}
		animes = append(animes, JikanAnime{
			MalID:    m.ID,
			Title:    title,
			Type:     m.Format,
			Year:     m.SeasonYear,
			Episodes: m.Episodes,
		})
	}
	return animes, nil
}

func fetchAnimeFromKitsu(query string, animeType string) ([]JikanAnime, error) {
	baseURL := "https://kitsu.io/api/edge/anime?filter[text]=" + url.QueryEscape(query) + "&page[limit]=10"
	switch animeType {
	case "Series":
		baseURL += "&filter[subtype]=TV,ONA"
	case "Movies":
		baseURL += "&filter[subtype]=movie"
	case "OVA":
		baseURL += "&filter[subtype]=OVA"
	case "Special":
		baseURL += "&filter[subtype]=special"
	}

	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("User-Agent", config.UserAgent)

	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Kitsu HTTP error: %d", resp.StatusCode)
	}

	var kRes kitsuAnimeResponse
	if err := json.NewDecoder(resp.Body).Decode(&kRes); err != nil {
		return nil, err
	}

	if len(kRes.Data) == 0 {
		return nil, fmt.Errorf("no Kitsu results")
	}

	var animes []JikanAnime
	for _, item := range kRes.Data {
		title := item.Attributes.Titles.En
		if title == "" {
			title = item.Attributes.CanonicalTitle
		}
		if title == "" {
			title = item.Attributes.Titles.EnJp
		}

		id, _ := strconv.Atoi(item.ID)
		year := 0
		if len(item.Attributes.StartDate) >= 4 {
			year, _ = strconv.Atoi(item.Attributes.StartDate[:4])
		}

		animes = append(animes, JikanAnime{
			MalID:    id,
			Title:    title,
			Type:     strings.ToUpper(item.Attributes.Subtype),
			Year:     year,
			Episodes: item.Attributes.EpisodeCount,
		})
	}
	return animes, nil
}

func fetchAnimeFromAnikoto(query string) ([]JikanAnime, error) {
	searchURL := fmt.Sprintf("%s/filter?keyword=%s", config.AnikotoBaseURL, url.QueryEscape(query))
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.UserAgent)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anikoto HTTP error: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	searchRE := regexp.MustCompile(`class="name d-title"\s+href="https://anikototv\.to/watch/([^/"]+)(?:/ep-\d+)?"[^>]*>([^<]+)</a>`)
	matches := searchRE.FindAllStringSubmatch(string(bodyBytes), -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches on Anikoto")
	}

	seenTitles := make(map[string]bool)
	var animes []JikanAnime
	for i, m := range matches {
		rawTitle := strings.TrimSpace(m[2])
		for strings.Contains(rawTitle, "&amp;") {
			rawTitle = strings.ReplaceAll(rawTitle, "&amp;", "&")
		}
		rawTitle = html.UnescapeString(rawTitle)

		if !seenTitles[strings.ToLower(rawTitle)] {
			seenTitles[strings.ToLower(rawTitle)] = true
			animes = append(animes, JikanAnime{
				MalID: -(i + 1),
				Title: rawTitle,
				Type:  "ANIME",
			})
		}
	}
	return animes, nil
}

func FetchAnime(query string, animeType string) tea.Cmd {
	return func() tea.Msg {
		// 1. Try AniList (Primary)
		if animes, err := fetchAnimeFromAniList(query, animeType); err == nil && len(animes) > 0 {
			return AnimeMsg(animes)
		}

		// 2. Fallback to Kitsu (Secondary)
		if animes, err := fetchAnimeFromKitsu(query, animeType); err == nil && len(animes) > 0 {
			return AnimeMsg(animes)
		}

		// 3. Fallback to Anikoto (Direct provider search)
		if animes, err := fetchAnimeFromAnikoto(query); err == nil && len(animes) > 0 {
			return AnimeMsg(animes)
		}

		return ErrMsg{fmt.Errorf("no matching anime found for '%s'", query)}
	}
}

func ResolveProxyLink(res ProwlarrResult, isAnime bool) string {
	var base string

	if res.InfoHash != "" {
		hash := strings.ToLower(strings.TrimSpace(res.InfoHash))
		if len(hash) == 40 {
			dn := url.QueryEscape(res.Title)
			base = fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", hash, dn)
		}
	}
	if base == "" {
		if res.MagnetUri != "" && strings.HasPrefix(res.MagnetUri, "magnet:") {
			base = res.MagnetUri
		} else if res.DownloadUrl != "" {
			if strings.HasPrefix(res.DownloadUrl, "magnet:") {
				base = res.DownloadUrl
			} else {
				if isAnime {
					localPath, err := downloadTorrentFile(res.DownloadUrl)
					if err != nil {
						return ""
					}
					return localPath
				} else {
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
							}
						}
					}
				}
			}
		}
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
	if err == nil {
		sysutil.RegisterTempDir(tmpFile)
	}
	return tmpFile, err
}

