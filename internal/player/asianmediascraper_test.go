package player

import (
	"encoding/json"
	"testing"
)

// --- Asian media type JSON tests ---

func TestAsianShow_Fields(t *testing.T) {
	s := AsianShow{
		Title: "Crash Landing on You",
		Type:  "tv",
		ID:    "42",
		Link:  "42",
	}
	if s.Title != "Crash Landing on You" || s.Type != "tv" || s.ID != "42" {
		t.Error("AsianShow fields incorrect")
	}
}

func TestAsianEpisode_Fields(t *testing.T) {
	e := AsianEpisode{
		Title: "1",
		Link:  "https://kisskh.do/Drama/Show/Episode-1?id=42&ep=1",
	}
	if e.Title != "1" || e.Link == "" {
		t.Error("AsianEpisode fields incorrect")
	}
}

func TestAsianShowsMsg_Type(t *testing.T) {
	msg := AsianShowsMsg{{Title: "Test", Type: "tv", ID: "1", Link: "1"}}
	if len(msg) != 1 || msg[0].Title != "Test" {
		t.Errorf("AsianShowsMsg incorrect: %+v", msg)
	}
}

func TestAsianEpisodesMsg_Type(t *testing.T) {
	msg := AsianEpisodesMsg{
		Type:     "tv",
		Eps:      []AsianEpisode{{Title: "1", Link: "url1"}},
		WatchURL: "https://example.com/watch",
	}
	if msg.Type != "tv" || len(msg.Eps) != 1 {
		t.Errorf("AsianEpisodesMsg incorrect: %+v", msg)
	}
}

func TestAsianStreamMsg_Fields(t *testing.T) {
	msg := AsianStreamMsg{
		StreamURL:   "https://stream.example.com/video.mp4",
		Referer:     "https://kisskh.do/",
		SubtitleURL: "https://sub.example.com/en.srt",
	}
	if msg.StreamURL == "" || msg.Referer == "" {
		t.Error("AsianStreamMsg should have non-empty fields")
	}
}

func TestSearchResponse_JSONUnmarshal(t *testing.T) {
	input := `[{"id": 1, "title": "Show A", "episodesCount": 16}, {"id": 2, "title": "Show B", "episodesCount": 1}]`

	var res SearchResponse
	if err := json.Unmarshal([]byte(input), &res); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].Title != "Show A" || res[0].EpisodesCount != 16 {
		t.Errorf("first result mismatch: %+v", res[0])
	}
	if res[1].Title != "Show B" || res[1].EpisodesCount != 1 {
		t.Errorf("second result mismatch: %+v", res[1])
	}
}

func TestDramaResponse_JSONUnmarshal(t *testing.T) {
	input := `{"episodes": [{"id": 101, "number": 16}, {"id": 100, "number": 15}]}`

	var res DramaResponse
	if err := json.Unmarshal([]byte(input), &res); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(res.Episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(res.Episodes))
	}
	if res.Episodes[0].Number != 16 {
		t.Errorf("expected first episode number 16, got %g", res.Episodes[0].Number)
	}
}

func TestSearchResponse_Empty(t *testing.T) {
	input := `[]`

	var res SearchResponse
	if err := json.Unmarshal([]byte(input), &res); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(res) != 0 {
		t.Errorf("expected empty array, got %d items", len(res))
	}
}
