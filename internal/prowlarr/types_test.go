package prowlarr

import (
	"encoding/json"
	"testing"
)

// --- JSON Unmarshal tests for types ---

func TestProwlarrResult_JSONRoundTrip(t *testing.T) {
	original := ProwlarrResult{
		Title:       "Breaking Bad S01 Complete 1080p",
		MagnetUri:   "magnet:?xt=urn:btih:abcdef123456",
		InfoHash:    "ABCDEF0123456789ABCDEF0123456789ABCDEF01",
		Seeders:     150,
		Peers:       300,
		Size:        10737418240, // 10 GB
		Indexer:     "The Pirate Bay",
		DownloadUrl: "http://example.com/download",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded ProwlarrResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Title != original.Title {
		t.Errorf("Title mismatch: got %q, want %q", decoded.Title, original.Title)
	}
	if decoded.MagnetUri != original.MagnetUri {
		t.Errorf("MagnetUri mismatch: got %q, want %q", decoded.MagnetUri, original.MagnetUri)
	}
	if decoded.InfoHash != original.InfoHash {
		t.Errorf("InfoHash mismatch: got %q, want %q", decoded.InfoHash, original.InfoHash)
	}
	if decoded.Seeders != original.Seeders {
		t.Errorf("Seeders mismatch: got %d, want %d", decoded.Seeders, original.Seeders)
	}
	if decoded.Peers != original.Peers {
		t.Errorf("Peers mismatch: got %d, want %d", decoded.Peers, original.Peers)
	}
	if decoded.Size != original.Size {
		t.Errorf("Size mismatch: got %d, want %d", decoded.Size, original.Size)
	}
	if decoded.Indexer != original.Indexer {
		t.Errorf("Indexer mismatch: got %q, want %q", decoded.Indexer, original.Indexer)
	}
	if decoded.DownloadUrl != original.DownloadUrl {
		t.Errorf("DownloadUrl mismatch: got %q, want %q", decoded.DownloadUrl, original.DownloadUrl)
	}
}

func TestIndexer_JSONRoundTrip(t *testing.T) {
	original := Indexer{ID: 5, Name: "1337x"}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Indexer
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ID != 5 || decoded.Name != "1337x" {
		t.Errorf("Indexer mismatch: got %+v", decoded)
	}
}

func TestTVMazeShow_JSONRoundTrip(t *testing.T) {
	original := TVMazeShow{ID: 123, Name: "Breaking Bad", Premiered: "2008-01-20"}

	data, _ := json.Marshal(original)
	var decoded TVMazeShow
	json.Unmarshal(data, &decoded)

	if decoded.ID != 123 || decoded.Name != "Breaking Bad" || decoded.Premiered != "2008-01-20" {
		t.Errorf("TVMazeShow mismatch: got %+v", decoded)
	}
}

func TestTVMazeSearchResult_JSONUnmarshal(t *testing.T) {
	input := `{"score": 10, "show": {"id": 1, "name": "Test Show", "premiered": "2020-01-01"}}`

	var result TVMazeSearchResult
	if err := json.Unmarshal([]byte(input), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Show.ID != 1 || result.Show.Name != "Test Show" {
		t.Errorf("TVMazeSearchResult mismatch: got %+v", result)
	}
}

func TestTVMazeSeason_JSONRoundTrip(t *testing.T) {
	original := TVMazeSeason{ID: 999, Number: 3}

	data, _ := json.Marshal(original)
	var decoded TVMazeSeason
	json.Unmarshal(data, &decoded)

	if decoded.ID != 999 || decoded.Number != 3 {
		t.Errorf("TVMazeSeason mismatch: got %+v", decoded)
	}
}

func TestTVMazeEpisode_JSONRoundTrip(t *testing.T) {
	original := TVMazeEpisode{ID: 42, Name: "Pilot", Number: 1}

	data, _ := json.Marshal(original)
	var decoded TVMazeEpisode
	json.Unmarshal(data, &decoded)

	if decoded.ID != 42 || decoded.Name != "Pilot" || decoded.Number != 1 {
		t.Errorf("TVMazeEpisode mismatch: got %+v", decoded)
	}
}

func TestCinemetaMovie_JSONRoundTrip(t *testing.T) {
	original := CinemetaMovie{
		ID:          "tt123456",
		Name:        "Inception",
		Year:        "2010",
		ReleaseInfo: "2010-07-16",
	}

	data, _ := json.Marshal(original)
	var decoded CinemetaMovie
	json.Unmarshal(data, &decoded)

	if decoded.ID != "tt123456" || decoded.Name != "Inception" || decoded.Year != "2010" {
		t.Errorf("CinemetaMovie mismatch: got %+v", decoded)
	}
}

func TestCinemetaCatalog_JSONUnmarshal(t *testing.T) {
	input := `{"metas": [{"id": "tt1", "name": "Movie A", "year": "2024"}]}`

	var catalog CinemetaCatalog
	if err := json.Unmarshal([]byte(input), &catalog); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(catalog.Metas) != 1 {
		t.Fatalf("expected 1 meta, got %d", len(catalog.Metas))
	}
	if catalog.Metas[0].Name != "Movie A" {
		t.Errorf("expected Movie A, got %s", catalog.Metas[0].Name)
	}
}

func TestJikanAnime_JSONRoundTrip(t *testing.T) {
	original := JikanAnime{
		MalID:    21,
		Title:    "One Piece",
		Type:     "TV",
		Year:     1999,
		Episodes: 1000,
	}

	data, _ := json.Marshal(original)
	var decoded JikanAnime
	json.Unmarshal(data, &decoded)

	if decoded.MalID != 21 || decoded.Title != "One Piece" || decoded.Episodes != 1000 {
		t.Errorf("JikanAnime mismatch: got %+v", decoded)
	}
}

func TestJikanResponse_JSONUnmarshal(t *testing.T) {
	input := `{"data": [{"mal_id": 1, "title": "Cowboy Bebop", "type": "TV", "year": 1998, "episodes": 26}]}`

	var resp JikanResponse
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(resp.Data) != 1 || resp.Data[0].Title != "Cowboy Bebop" {
		t.Errorf("JikanResponse mismatch: got %+v", resp)
	}
}

func TestProwlarrResult_EmptyFields(t *testing.T) {
	// Ensure empty JSON doesn't cause panics
	input := `{}`
	var result ProwlarrResult
	if err := json.Unmarshal([]byte(input), &result); err != nil {
		t.Fatalf("Unmarshal empty object failed: %v", err)
	}

	if result.Title != "" || result.Seeders != 0 || result.Size != 0 {
		t.Error("empty JSON should produce zero-value struct")
	}
}

func TestIndexerGroup_Fields(t *testing.T) {
	g := IndexerGroup{
		ID:      1,
		Name:    "Test",
		Status:  "Complete",
		Results: []ProwlarrResult{{Title: "Movie1"}},
		Order:   1,
	}

	if g.ID != 1 || len(g.Results) != 1 || g.Results[0].Title != "Movie1" {
		t.Errorf("IndexerGroup field access failed: got %+v", g)
	}
}

// --- Message type tests ---

func TestErrMsg_Creation(t *testing.T) {
	msg := ErrMsg{Err: nil}
	if msg.Err != nil {
		t.Error("expected nil error")
	}
}

func TestSearchResultMsg_Creation(t *testing.T) {
	msg := SearchResultMsg{
		IndexerID: 42,
		Results:   []ProwlarrResult{{Title: "Test"}},
	}
	if msg.IndexerID != 42 || len(msg.Results) != 1 {
		t.Errorf("SearchResultMsg fields incorrect: %+v", msg)
	}
}

func TestCinemetaMsg_Creation(t *testing.T) {
	msg := CinemetaMsg([]CinemetaMovie{{Name: "Movie"}})
	if len(msg) != 1 || msg[0].Name != "Movie" {
		t.Errorf("CinemetaMsg incorrect: %+v", msg)
	}
}

func TestTvShowsMsg_Creation(t *testing.T) {
	msg := TvShowsMsg([]TVMazeShow{{Name: "Show"}})
	if len(msg) != 1 || msg[0].Name != "Show" {
		t.Errorf("TvShowsMsg incorrect: %+v", msg)
	}
}

func TestTvEpisodesMsg_Creation(t *testing.T) {
	msg := TvEpisodesMsg([]TVMazeEpisode{{Number: 5}})
	if len(msg) != 1 || msg[0].Number != 5 {
		t.Errorf("TvEpisodesMsg incorrect: %+v", msg)
	}
}

func TestAnimeMsg_Creation(t *testing.T) {
	msg := AnimeMsg([]JikanAnime{{Title: "Anime"}})
	if len(msg) != 1 || msg[0].Title != "Anime" {
		t.Errorf("AnimeMsg incorrect: %+v", msg)
	}
}
