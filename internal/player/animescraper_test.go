package player

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Anime scraper type JSON tests ---

func TestShowResult_Fields(t *testing.T) {
	s := ShowResult{Slug: "one-piece", Title: "One Piece"}
	if s.Slug != "one-piece" || s.Title != "One Piece" {
		t.Error("ShowResult fields incorrect")
	}
}

func TestEpisodeResult_Fields(t *testing.T) {
	e := EpisodeResult{ID: "123", Num: "5", Token: "abc123token"}
	if e.ID != "123" || e.Num != "5" || e.Token != "abc123token" {
		t.Error("EpisodeResult fields incorrect")
	}
}

func TestServerResult_Fields(t *testing.T) {
	s := ServerResult{Name: "Server A", LinkID: "link-xyz"}
	if s.Name != "Server A" || s.LinkID != "link-xyz" {
		t.Error("ServerResult fields incorrect")
	}
}

func TestAjaxResponse_JSONUnmarshal(t *testing.T) {
	input := `{"status": 200, "result": "<div>html content</div>"}`

	var res AjaxResponse
	if err := json.Unmarshal([]byte(input), &res); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if res.Status != 200 || res.Result != "<div>html content</div>" {
		t.Errorf("AjaxResponse mismatch: %+v", res)
	}
}

func TestSourceResponse_JSONUnmarshal(t *testing.T) {
	input := `{"status": 200, "result": {"url": "https://example.com/embed/123"}}`

	var res SourceResponse
	if err := json.Unmarshal([]byte(input), &res); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if res.Result.URL != "https://example.com/embed/123" {
		t.Errorf("SourceResponse URL mismatch: %s", res.Result.URL)
	}
}

func TestGetSourcesResponse_JSONUnmarshal(t *testing.T) {
	input := `{
		"sources": {"file": "https://m3u8.example.com/stream.m3u8"},
		"tracks": [
			{"file": "https://sub.example.com/en.vtt", "label": "English", "kind": "captions"},
			{"file": "https://sub.example.com/es.vtt", "label": "Spanish", "kind": "captions"}
		]
	}`

	var res GetSourcesResponse
	if err := json.Unmarshal([]byte(input), &res); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if res.Sources.File != "https://m3u8.example.com/stream.m3u8" {
		t.Errorf("Sources file mismatch: %s", res.Sources.File)
	}
	if len(res.Tracks) != 2 {
		t.Errorf("expected 2 tracks, got %d", len(res.Tracks))
	}
}

func TestAnikotoShowsMsg_Type(t *testing.T) {
	msg := AnikotoShowsMsg{
		{Slug: "test", Title: "Test Show"},
	}
	if len(msg) != 1 || msg[0].Slug != "test" {
		t.Errorf("AnikotoShowsMsg incorrect: %+v", msg)
	}
}

func TestAnikotoEpsMsg_Type(t *testing.T) {
	msg := AnikotoEpsMsg{
		Eps:      []EpisodeResult{{ID: "1", Num: "1", Token: "t"}},
		WatchURL: "https://example.com/watch/show",
	}
	if msg.WatchURL != "https://example.com/watch/show" || len(msg.Eps) != 1 {
		t.Errorf("AnikotoEpsMsg incorrect: %+v", msg)
	}
}

func TestAnikotoServersMsg_Type(t *testing.T) {
	msg := AnikotoServersMsg{{Name: "Vidstream", LinkID: "123"}}
	if len(msg) != 1 || msg[0].Name != "Vidstream" {
		t.Errorf("AnikotoServersMsg incorrect: %+v", msg)
	}
}

func TestAnikotoStreamMsg_Fields(t *testing.T) {
	msg := AnikotoStreamMsg{
		M3u8URL:     "https://stream.example.com/playlist.m3u8",
		Referer:     "https://embed.example.com/",
		SubtitleURL: "https://sub.example.com/en.vtt",
	}
	if msg.M3u8URL == "" || msg.Referer == "" {
		t.Error("AnikotoStreamMsg fields should be non-empty")
	}
}

// --- fetchHTTP tests ---

func TestFetchHTTP_InvalidURL(t *testing.T) {
	client := &http.Client{}
	_, err := fetchHTTP(client, "://invalid-url", "", false)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestFetchHTTP_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := &http.Client{}
	_, err := fetchHTTP(client, ts.URL+"/fail", "", false)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestFetchHTTP_MissingScheme(t *testing.T) {
	client := &http.Client{}
	_, err := fetchHTTP(client, "not-a-url", "", false)
	if err == nil {
		t.Error("expected error for missing scheme")
	}
}

func TestDecryptSourcesEnc_Success(t *testing.T) {
	// Sample encrypted string generated with the standard key and IV
	enc := "wdeBruh3qqn_i5wUNnyaPcXqidp1UWP84FfPHzGyKXD33V0jwSoOHPTijvJ6ZqQumcZd_ZfU0icvRA_1o2Ty7gVUSYvlMmrk09FoaXRXJ57blHmT9cZHuQ8Qxv_T2U7GTEmtdIIIVZVgy5sNeRSSkqY1VPqlBtMjz6xf0LrNy-g"
	dec, err := decryptSourcesEnc(enc)
	if err != nil {
		t.Fatalf("decryptSourcesEnc failed: %v", err)
	}
	var res struct {
		File string `json:"file"`
	}
	if err := json.Unmarshal([]byte(dec), &res); err != nil {
		t.Fatalf("unmarshal decrypted text failed: %v", err)
	}
	if res.File == "" {
		t.Error("expected non-empty file in decrypted payload")
	}
}

func TestAttachCdnToken(t *testing.T) {
	input := "https://fetch.example.com/anime/44119006254708ef096f25a96700dfb1/98345dd250f1043a41f4ee111c071011/master.m3u8"
	signed := attachCdnToken(input)
	if signed == input {
		t.Error("attachCdnToken should have appended token")
	}
	if !contains(signed, "token=") {
		t.Error("expected token query parameter in signed URL")
	}

	// Should not re-sign if token already exists
	alreadySigned := attachCdnToken(signed)
	if alreadySigned != signed {
		t.Error("should not re-sign already signed URL")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && (func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()))
}

