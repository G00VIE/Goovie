package prowlarr

import "fmt"

// Indexer represents a Prowlarr indexer from the API.
type Indexer struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
}

// TorrentResult holds a single parsed search result.
type TorrentResult struct {
	TorrentName string
	Size        int64
	Seeders     int
	MagnetURL   string
	Indexer     string
}

// Title implements the list.Item interface for bubbletea.
func (t TorrentResult) Title() string { return t.TorrentName }

// Description implements the list.Item interface for bubbletea.
func (t TorrentResult) Description() string {
	sizeGB := float64(t.Size) / (1024 * 1024 * 1024)
	return fmt.Sprintf("Seeders: %d | Size: %.2f GB | [%s]", t.Seeders, sizeGB, t.Indexer)
}

// FilterValue implements the list.Item interface for bubbletea.
func (t TorrentResult) FilterValue() string { return t.TorrentName }

// searchResult is the raw JSON shape returned by the Prowlarr search API.
type searchResult struct {
	Title       string `json:"title"`
	Size        int64  `json:"size"`
	Seeders     int    `json:"seeders"`
	MagnetURL   string `json:"magnetUrl"`
	DownloadURL string `json:"downloadUrl"`
	Indexer     string `json:"indexer"`
}
