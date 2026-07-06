package player

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"bubble-stream/internal/prowlarr"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

type AsianShow struct {
	Title string
	Type  string // "tv" or "movie"
	ID    string
	Link  string
}

type AsianEpisode struct {
	Title string
	Link  string
}

type AsianShowsMsg []AsianShow

type AsianEpisodesMsg struct {
	Type     string // "tv" or "movie"
	Eps      []AsianEpisode
	WatchURL string
}

type AsianStreamMsg struct {
	StreamURL   string
	Referer     string
	SubtitleURL string
}

// SearchResponse from Kisskh
type SearchResponse []struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	EpisodesCount int    `json:"episodesCount"`
}

func cleanSlug(title string) string {
	var sb strings.Builder
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		} else {
			sb.WriteByte('-')
		}
	}
	return sb.String()
}

func fetchJSON(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

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

func FetchAsianShowsCmd(query string, isTVShow bool) tea.Cmd {
	return func() tea.Msg {
		body, err := fetchJSON("https://kisskh.do/api/DramaList/Search?q=" + strings.ReplaceAll(query, " ", "+"))
		if err != nil {
			return prowlarr.ErrMsg{Err: fmt.Errorf("kisskh search failed: %v", err)}
		}
		var searchRes SearchResponse
		err = json.Unmarshal(body, &searchRes)
		if err != nil {
			return prowlarr.ErrMsg{Err: fmt.Errorf("kisskh search decode failed: %v", err)}
		}

		if len(searchRes) == 0 {
			typeStr := "movies"
			if isTVShow {
				typeStr = "tv shows"
			}
			return prowlarr.ErrMsg{Err: fmt.Errorf("no matching %s found on kisskh for '%s'", typeStr, query)}
		}

		var shows []AsianShow
		
		queryWords := strings.Fields(strings.ToLower(query))

		for _, res := range searchRes {
			titleLower := strings.ToLower(res.Title)
			matched := false
			for _, w := range queryWords {
				if strings.Contains(titleLower, w) {
					matched = true
					break
				}
			}
			
			if !matched {
				continue
			}
			// If the user explicitly selected MOVIE, filter out results with > 1 episodes
			if !isTVShow && res.EpisodesCount > 1 {
				continue
			}

			showType := "drama"
			if !isTVShow {
				showType = "movie"
			}

			shows = append(shows, AsianShow{
				Title: res.Title,
				Type:  showType,
				ID:    fmt.Sprintf("%d", res.ID),
				Link:  fmt.Sprintf("%d", res.ID), // Passing ID in link field for convenience
			})
		}

		return AsianShowsMsg(shows)
	}
}

type DramaResponse struct {
	Episodes []struct {
		ID     int     `json:"id"`
		Number float64 `json:"number"`
	} `json:"episodes"`
}

func FetchAsianEpisodesCmd(show AsianShow) tea.Cmd {
	return func() tea.Msg {
		body, err := fetchJSON(fmt.Sprintf("https://kisskh.do/api/DramaList/Drama/%s?isq=false", show.ID))
		if err != nil {
			return prowlarr.ErrMsg{Err: fmt.Errorf("failed to fetch details from kisskh")}
		}
		var dramaRes DramaResponse
		err = json.Unmarshal(body, &dramaRes)
		if err != nil {
			return prowlarr.ErrMsg{Err: fmt.Errorf("failed to decode drama details")}
		}

		if len(dramaRes.Episodes) == 0 {
			return prowlarr.ErrMsg{Err: fmt.Errorf("no episodes found for this show")}
		}

		var episodes []AsianEpisode
		// In Kisskh, episodes are usually sorted latest first, so let's check
		// To show Episode 1 first, we might want to iterate in reverse or just depend on what API returns.
		// API returns descending (ep 16 ... ep 1)
		for i := len(dramaRes.Episodes) - 1; i >= 0; i-- {
			ep := dramaRes.Episodes[i]
			titleStr := fmt.Sprintf("%g", ep.Number)
			episodes = append(episodes, AsianEpisode{
				Title: titleStr,
				Link:  fmt.Sprintf("https://kisskh.do/Drama/%s/Episode-%g?id=%s&ep=%d", cleanSlug(show.Title), ep.Number, show.ID, ep.ID),
			})
		}

		return AsianEpisodesMsg{
			Type: "tv",
			Eps:  episodes,
		}
	}
}

func FetchAsianStreamCmd(watchURL string) tea.Cmd {
	return func() tea.Msg {
		l := launcher.New().Leakless(false)
		u, err := l.Launch()
		if err != nil {
			return prowlarr.ErrMsg{Err: fmt.Errorf("failed to launch browser: %v", err)}
		}
		browser := rod.New().ControlURL(u).MustConnect()
		defer browser.MustClose()

		page := stealth.MustPage(browser)

		videoUrl := ""
		subUrl := ""

		router := page.HijackRequests()
		defer router.MustStop()

		router.MustAdd("*/api/DramaList/Episode/*", func(ctx *rod.Hijack) {
			_ = ctx.LoadResponse(http.DefaultClient, true)

			var epData struct {
				Video string `json:"Video"`
			}

			bodyStr := ctx.Response.Body()
			if len(bodyStr) > 0 {
				if err := json.Unmarshal([]byte(bodyStr), &epData); err == nil && epData.Video != "" {
					videoUrl = epData.Video
				}
			}
		})

		router.MustAdd("*/api/Sub/*", func(ctx *rod.Hijack) {
			_ = ctx.LoadResponse(http.DefaultClient, true)

			type SubItem struct {
				Src  string `json:"src"`
				Land string `json:"land"`
			}
			var subData []SubItem

			bodyStr := ctx.Response.Body()
			if len(bodyStr) > 0 {
				if err := json.Unmarshal([]byte(bodyStr), &subData); err == nil {
					for _, sub := range subData {
						if sub.Land == "en" || strings.EqualFold(sub.Land, "English") {
							subUrl = sub.Src
							break
						}
					}
				}
			}
		})

		go router.Run()

		page.MustNavigate(watchURL)

		// Wait dynamically for up to 30 seconds for the API calls to complete
		for i := 0; i < 30; i++ {
			if videoUrl != "" && subUrl != "" {
				break // Got both!
			}
			time.Sleep(1 * time.Second)
		}

		if videoUrl == "" {
			return prowlarr.ErrMsg{Err: fmt.Errorf("failed to extract video URL")}
		}

		return AsianStreamMsg{
			StreamURL:   videoUrl,
			Referer:     "https://kisskh.do/",
			SubtitleURL: subUrl,
		}
	}
}
