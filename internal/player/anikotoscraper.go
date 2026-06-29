package player

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"bubble-stream/internal/config"
	"bubble-stream/internal/prowlarr"
	tea "github.com/charmbracelet/bubbletea"
)

type ShowResult struct {
	Slug  string
	Title string
}
type EpisodeResult struct {
	ID    string
	Num   string
	Token string
}
type ServerResult struct {
	Name   string
	LinkID string
}
type AjaxResponse struct {
	Status int    `json:"status"`
	Result string `json:"result"`
}
type SourceResponse struct {
	Status int `json:"status"`
	Result struct {
		URL string `json:"url"`
	} `json:"result"`
}
type GetSourcesResponse struct {
	Sources struct {
		File string `json:"file"`
	} `json:"sources"`
}

type AnikotoShowsMsg []ShowResult
type AnikotoEpsMsg struct {
	Eps      []EpisodeResult
	WatchURL string
}
type AnikotoServersMsg []ServerResult
type AnikotoStreamMsg struct {
	M3u8URL string
	Referer string
}

func fetchHTTP(client *http.Client, reqURL string, referer string, xRequestedWith bool) ([]byte, error) {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.UserAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	if xRequestedWith {
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func FetchAnikotoShowsCmd(query string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Jar: GlobalJar, Timeout: 15 * time.Second}
		searchURL := fmt.Sprintf("%s/filter?keyword=%s", config.AnikotoBaseURL, url.QueryEscape(query))
		bodyBytes, err := fetchHTTP(client, searchURL, "", false)
		if err != nil {
			return prowlarr.ErrMsg{Err: fmt.Errorf("Anikoto search failed: %v", err)}
		}

		searchRE := regexp.MustCompile(`class="name d-title"\s+href="https://anikototv\.to/watch/([^/"]+)(?:/ep-\d+)?"[^>]*>([^<]+)</a>`)
		matches := searchRE.FindAllStringSubmatch(string(bodyBytes), -1)

		if len(matches) == 0 {
			return prowlarr.ErrMsg{Err: fmt.Errorf("no matching shows found on Anikoto for '%s'", query)}
		}

		seenSlugs := make(map[string]bool)
		var shows []ShowResult
		for _, m := range matches {
			slug := m[1]
			title := strings.TrimSpace(m[2])
			if !seenSlugs[slug] {
				seenSlugs[slug] = true
				shows = append(shows, ShowResult{Slug: slug, Title: title})
			}
		}
		return AnikotoShowsMsg(shows)
	}
}

func FetchAnikotoEpisodesCmd(slug string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Jar: GlobalJar, Timeout: 15 * time.Second}
		watchPageURL := fmt.Sprintf("%s/watch/%s", config.AnikotoBaseURL, slug)
		watchPageBytes, err := fetchHTTP(client, watchPageURL, "", false)
		if err != nil {
			return prowlarr.ErrMsg{Err: fmt.Errorf("failed to fetch Anikoto watch page")}
		}

		idRE := regexp.MustCompile(`id="watch-main"[^>]*data-id="(\d+)"`)
		idMatch := idRE.FindStringSubmatch(string(watchPageBytes))
		if len(idMatch) < 2 {
			return prowlarr.ErrMsg{Err: fmt.Errorf("could not resolve show ID on Anikoto")}
		}
		showID := idMatch[1]

		episodeListURL := fmt.Sprintf("%s/ajax/episode/list/%s", config.AnikotoBaseURL, showID)
		epListBytes, err := fetchHTTP(client, episodeListURL, watchPageURL, true)
		if err != nil {
			return prowlarr.ErrMsg{Err: fmt.Errorf("failed to fetch episode list")}
		}

		var ajaxRes AjaxResponse
		json.Unmarshal(epListBytes, &ajaxRes)

		epRE := regexp.MustCompile(`data-id="(\d+)"\s+data-num="(\d+)"[^>]*data-ids="([^"]+)"`)
		epMatches := epRE.FindAllStringSubmatch(ajaxRes.Result, -1)
		if len(epMatches) == 0 {
			return prowlarr.ErrMsg{Err: fmt.Errorf("no episodes found on Anikoto")}
		}

		var episodes []EpisodeResult
		for _, m := range epMatches {
			episodes = append(episodes, EpisodeResult{
				ID:    m[1],
				Num:   m[2],
				Token: m[3],
			})
		}
		return AnikotoEpsMsg{Eps: episodes, WatchURL: watchPageURL}
	}
}

func resolveStream(linkID string, watchURL string, mode string) (AnikotoStreamMsg, error) {
	client := &http.Client{Jar: GlobalJar, Timeout: 15 * time.Second}
	getSourceURL := fmt.Sprintf("%s/ajax/server?get=%s", config.AnikotoBaseURL, url.QueryEscape(linkID))
	sourceBytes, err := fetchHTTP(client, getSourceURL, watchURL, true)
	if err != nil {
		return AnikotoStreamMsg{}, fmt.Errorf("failed to get stream embed JSON")
	}

	var sourceRes SourceResponse
	json.Unmarshal(sourceBytes, &sourceRes)
	embedURL := sourceRes.Result.URL
	if embedURL == "" {
		return AnikotoStreamMsg{}, fmt.Errorf("resolved embed URL is empty")
	}

	parsedEmbedURL, err := url.Parse(embedURL)
	if err != nil {
		return AnikotoStreamMsg{}, err
	}
	embedHost := parsedEmbedURL.Host

	playerPageBytes, err := fetchHTTP(client, embedURL, config.AnikotoBaseURL+"/", false)
	if err != nil {
		return AnikotoStreamMsg{}, fmt.Errorf("failed to fetch player webpage")
	}

	mediaIDRE := regexp.MustCompile(`data-id="(\d+)"`)
	mediaIDMatch := mediaIDRE.FindStringSubmatch(string(playerPageBytes))
	if len(mediaIDMatch) < 2 {
		return AnikotoStreamMsg{}, fmt.Errorf("could not find media ID in player HTML")
	}
	mediaID := mediaIDMatch[1]

	finalSourcesURL := fmt.Sprintf("https://%s/stream/getSourcesNew?id=%s&type=%s", embedHost, mediaID, mode)
	finalSourcesBytes, err := fetchHTTP(client, finalSourcesURL, embedURL, true)
	if err != nil {
		return AnikotoStreamMsg{}, fmt.Errorf("failed to fetch final m3u8 sources")
	}

	var finalRes GetSourcesResponse
	json.Unmarshal(finalSourcesBytes, &finalRes)

	if finalRes.Sources.File == "" {
		return AnikotoStreamMsg{}, fmt.Errorf("no streaming link resolved from provider")
	}

	referrerBase := parsedEmbedURL.Scheme + "://" + parsedEmbedURL.Host + "/"
	return AnikotoStreamMsg{M3u8URL: finalRes.Sources.File, Referer: referrerBase}, nil
}

func RaceAnikotoStreamsCmd(epToken string, mode string, watchURL string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Jar: GlobalJar, Timeout: 15 * time.Second}
		serverListURL := fmt.Sprintf("%s/ajax/server/list?servers=%s", config.AnikotoBaseURL, url.QueryEscape(epToken))
		serverListBytes, err := fetchHTTP(client, serverListURL, watchURL, true)
		if err != nil {
			return prowlarr.ErrMsg{Err: fmt.Errorf("failed to fetch server list")}
		}

		var serverAjax AjaxResponse
		json.Unmarshal(serverListBytes, &serverAjax)

		typeBlockStart := fmt.Sprintf(`data-type="%s"`, mode)
		startIdx := strings.Index(serverAjax.Result, typeBlockStart)
		if startIdx == -1 {
			return prowlarr.ErrMsg{Err: fmt.Errorf("mode %s is not available for this episode", strings.ToUpper(mode))}
		}

		blockHTML := serverAjax.Result[startIdx:]
		endIdx := strings.Index(blockHTML, "</ul>")
		if endIdx != -1 {
			blockHTML = blockHTML[:endIdx]
		}

		serverRE := regexp.MustCompile(`<li[^>]*?data-link-id="([^"]+)"[^>]*>([^<]+)</li>`)
		serverMatches := serverRE.FindAllStringSubmatch(blockHTML, -1)
		if len(serverMatches) == 0 {
			return prowlarr.ErrMsg{Err: fmt.Errorf("no servers found for %s mode", mode)}
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		resultChan := make(chan AnikotoStreamMsg, len(serverMatches))
		
		var wg sync.WaitGroup
		for _, m := range serverMatches {
			wg.Add(1)
			go func(linkID string) {
				defer wg.Done()
				if ctx.Err() != nil {
					return
				}
				res, err := resolveStream(linkID, watchURL, mode)
				if err == nil {
					select {
					case resultChan <- res:
					case <-ctx.Done():
					}
				}
			}(m[1])
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		res, ok := <-resultChan
		if !ok {
			return prowlarr.ErrMsg{Err: fmt.Errorf("all providers failed to resolve a stream")}
		}
		
		cancel()
		
		return res
	}
}
