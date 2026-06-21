package prowlarr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Client communicates with the Prowlarr API.
type Client struct {
	BaseURL string
	APIKey  string
}

// NewClient creates a new Prowlarr API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{BaseURL: baseURL, APIKey: apiKey}
}

// --- Bubbletea messages ---

// IndexersFetchedMsg is sent when the list of active indexers has been retrieved.
type IndexersFetchedMsg struct {
	Indexers []Indexer
	Query    string
}

// SearchResultMsg is sent when a single indexer finishes its search.
type SearchResultMsg struct {
	Items []TorrentResult
}

// ErrMsg wraps an error for the bubbletea message loop.
type ErrMsg struct{ Err error }

func (e ErrMsg) Error() string { return e.Err.Error() }

// --- Bubbletea commands ---

// FetchIndexersCmd returns a tea.Cmd that fetches active indexers from Prowlarr.
func (c *Client) FetchIndexersCmd(query string) tea.Cmd {
	return func() tea.Msg {
		httpClient := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequest("GET", c.BaseURL+"/api/v1/indexer", nil)
		req.Header.Add("X-Api-Key", c.APIKey)

		resp, err := httpClient.Do(req)
		if err != nil {
			return ErrMsg{fmt.Errorf("failed to contact Prowlarr: %v", err)}
		}
		defer resp.Body.Close()

		var indexers []Indexer
		if err := json.NewDecoder(resp.Body).Decode(&indexers); err != nil {
			return ErrMsg{fmt.Errorf("failed to parse indexers")}
		}

		var active []Indexer
		for _, idx := range indexers {
			if idx.Enable {
				active = append(active, idx)
			}
		}

		if len(active) == 0 {
			return ErrMsg{fmt.Errorf("no indexers enabled in Prowlarr")}
		}

		return IndexersFetchedMsg{Indexers: active, Query: query}
	}
}

// FetchSingleIndexerCmd returns a tea.Cmd that searches a single indexer.
func (c *Client) FetchSingleIndexerCmd(query string, indexerID int) tea.Cmd {
	return func() tea.Msg {
		httpClient := &http.Client{Timeout: 15 * time.Second}
		apiURL := fmt.Sprintf("%s/api/v1/search?query=%s&type=search&indexerIds=%d", c.BaseURL, url.QueryEscape(query), indexerID)

		req, _ := http.NewRequest("GET", apiURL, nil)
		req.Header.Add("X-Api-Key", c.APIKey)

		resp, err := httpClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			return SearchResultMsg{Items: nil}
		}
		defer resp.Body.Close()

		var res []searchResult
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return SearchResultMsg{Items: nil}
		}

		var items []TorrentResult
		for _, v := range res {
			targetURL := v.MagnetURL

			if targetURL == "" && v.DownloadURL != "" {
				if strings.Contains(v.DownloadURL, "?") {
					targetURL = v.DownloadURL + "&apikey=" + c.APIKey
				} else {
					targetURL = v.DownloadURL + "?apikey=" + c.APIKey
				}
			}

			if targetURL != "" && v.Seeders > 0 {
				items = append(items, TorrentResult{
					TorrentName: v.Title,
					Size:        v.Size,
					Seeders:     v.Seeders,
					MagnetURL:   targetURL,
					Indexer:     v.Indexer,
				})
			}
		}

		return SearchResultMsg{Items: items}
	}
}
