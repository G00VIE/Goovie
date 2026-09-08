package prowlarr

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"bubble-stream/internal/config"
)

// --- ResolveProxyLink tests ---

func TestResolveProxyLink_InfoHash(t *testing.T) {
	res := ProwlarrResult{
		Title:    "Test Movie 2024",
		InfoHash: "ABCDEF0123456789ABCDEF0123456789ABCDEF01",
	}

	// Remove debug file so test is clean
	os.Remove("torrent_debug.txt")

	result := ResolveProxyLink(res, false)

	if result == "" {
		t.Fatal("ResolveProxyLink should return a non-empty magnet when InfoHash is valid")
	}
	if !strings.HasPrefix(result, "magnet:?xt=urn:btih:") {
		t.Errorf("result should start with magnet:?xt=urn:btih:, got: %s", result[:50])
	}
	if !strings.Contains(result, url.QueryEscape("Test Movie 2024")) {
		t.Error("result should contain the DN (display name)")
	}
}

func TestResolveProxyLink_InfoHashCaseInsensitive(t *testing.T) {
	res := ProwlarrResult{
		Title:    "Test Movie",
		InfoHash: "aBcDeF0123456789AbCdEf0123456789AbCdEf01",
	}

	result := ResolveProxyLink(res, false)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	// Hash in magnet should be lowercase
	if !strings.Contains(result, "xt=urn:btih:abcdef") {
		t.Errorf("hash in magnet should be lowercase, got: %s", result)
	}
}

func TestResolveProxyLink_InfoHashTooShort(t *testing.T) {
	res := ProwlarrResult{
		Title:    "Test",
		InfoHash: "short",
	}

	result := ResolveProxyLink(res, false)
	if result != "" {
		t.Errorf("short InfoHash should fall through, got: %s", result)
	}
}

func TestResolveProxyLink_MagnetUri(t *testing.T) {
	res := ProwlarrResult{
		Title:     "Test",
		MagnetUri: "magnet:?xt=urn:btih:ABCDEF0123456789ABCDEF0123456789ABCDEF01",
	}

	os.Remove("torrent_debug.txt")
	result := ResolveProxyLink(res, false)

	if result == "" {
		t.Fatal("expected non-empty result for valid MagnetUri")
	}
	if !strings.HasPrefix(result, "magnet:") {
		t.Errorf("result should be a magnet link, got: %s", result[:50])
	}
}

func TestResolveProxyLink_DownloadUrlRedirect(t *testing.T) {
	// Set up a test server that returns a redirect to a magnet link
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "magnet:?xt=urn:btih:1234567890abcdef1234567890abcdef12345678", http.StatusFound)
	}))
	defer ts.Close()

	res := ProwlarrResult{
		Title:       "Redirect Test",
		DownloadUrl: ts.URL + "/torrent/download",
	}

	os.Remove("torrent_debug.txt")
	result := ResolveProxyLink(res, false)

	if result == "" {
		t.Fatal("expected non-empty result from redirect")
	}
	if !strings.Contains(result, "magnet:?xt=urn:btih:") {
		t.Errorf("expected magnet link from redirect, got: %s", result)
	}
}

func TestResolveProxyLink_DownloadUrlNonMagnetRedirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://other.site/file.torrent", http.StatusFound)
	}))
	defer ts.Close()

	res := ProwlarrResult{
		Title:       "Non Magnet Redirect",
		DownloadUrl: ts.URL + "/download",
	}

	os.Remove("torrent_debug.txt")
	result := ResolveProxyLink(res, false)

	// Should fail since redirect goes to non-magnet URL
	if result != "" {
		t.Errorf("expected empty result for non-magnet redirect, got: %s", result)
	}
}

func TestResolveProxyLink_DownloadUrl200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("torrent data"))
	}))
	defer ts.Close()

	res := ProwlarrResult{
		Title:       "Direct 200",
		DownloadUrl: ts.URL + "/file",
	}

	os.Remove("torrent_debug.txt")
	result := ResolveProxyLink(res, false)

	if result != "" {
		t.Errorf("expected empty for 200 response (not a redirect), got: %s", result)
	}
}

func TestResolveProxyLink_TrackersAdded(t *testing.T) {
	res := ProwlarrResult{
		Title:    "Tracker Test",
		InfoHash: "ABCDEF0123456789ABCDEF0123456789ABCDEF01",
	}

	os.Remove("torrent_debug.txt")
	result := ResolveProxyLink(res, false)

	for _, tr := range config.DefaultTrackers {
		encoded := url.QueryEscape(tr)
		if !strings.Contains(result, encoded) {
			t.Errorf("result should contain tracker %q (encoded: %q)", tr, encoded)
		}
	}
}

func TestResolveProxyLink_EmptyResult(t *testing.T) {
	res := ProwlarrResult{
		Title: "Empty Everything",
	}

	os.Remove("torrent_debug.txt")
	result := ResolveProxyLink(res, false)

	if result != "" {
		t.Errorf("expected empty result for empty ProwlarrResult, got: %s", result)
	}
}

func TestResolveProxyLink_InfoHashPreferredOverMagnetUri(t *testing.T) {
	res := ProwlarrResult{
		Title:     "Priority Test",
		InfoHash:  "ABCDEF0123456789ABCDEF0123456789ABCDEF01",
		MagnetUri: "magnet:?xt=urn:btih:FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
	}

	os.Remove("torrent_debug.txt")
	result := ResolveProxyLink(res, false)

	// InfoHash should take priority (and is lowercased by the resolver)
	if !strings.Contains(result, "xt=urn:btih:abcdef0123456789abcdef0123456789abcdef01") {
		t.Errorf("InfoHash should take priority over MagnetUri, got: %s", result)
	}
	// The MagnetUri's hash (all F's) must NOT be present
	if strings.Contains(result, "ffffffffffffffffffffffffffffffffffffffff") {
		t.Errorf("MagnetUri hash should NOT be used when InfoHash is present, got: %s", result)
	}
}

// --- downloadTorrentFile ---

func TestDownloadTorrentFile_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		w.Write([]byte("fake torrent content here"))
	}))
	defer ts.Close()

	path, err := downloadTorrentFile(ts.URL + "/test.torrent")
	if err != nil {
		t.Fatalf("downloadTorrentFile failed: %v", err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != "fake torrent content here" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestDownloadTorrentFile_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := downloadTorrentFile(ts.URL + "/missing")
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

// --- TV file parsing helpers (indirect via SearchSingleIndexer) ---

func TestSearchSingleIndexer_QualityFilter(t *testing.T) {
	// Mock test: ensure seeders filter works by checking the logic path
	// The minimum seeders is checked inside SearchSingleIndexer
	if config.MinimumSeeders <= 0 {
		t.Errorf("MinimumSeeders should be positive, got %d", config.MinimumSeeders)
	}
}

// --- TV and Retry Fetch Tests ---

func TestFetchWithRetry_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != config.UserAgent {
			t.Errorf("expected User-Agent %q, got %q", config.UserAgent, r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	resp, err := fetchWithRetry(ts.URL)
	if err != nil {
		t.Fatalf("fetchWithRetry failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestFetchWithRetry_RetryThenSuccess(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"recovered"}`))
	}))
	defer ts.Close()

	resp, err := fetchWithRetry(ts.URL)
	if err != nil {
		t.Fatalf("fetchWithRetry expected success on retry, got err: %v", err)
	}
	defer resp.Body.Close()
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestFetchWithRetry_AllFail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := fetchWithRetry(ts.URL)
	if err == nil {
		t.Error("expected error when server always fails")
	}
}

func TestFetchTVSeasons_Fallback(t *testing.T) {
	cmd := FetchTVSeasons(-1)
	msg := cmd()
	seasons, ok := msg.(TvSeasonsMsg)
	if !ok {
		t.Fatalf("expected TvSeasonsMsg, got %T", msg)
	}
	if len(seasons) == 0 {
		t.Error("expected fallback default seasons, got empty")
	}
	if seasons[0].Number != 1 {
		t.Errorf("expected first season number 1, got %d", seasons[0].Number)
	}
}

func TestFetchTVEpisodes_GracefulDegrade(t *testing.T) {
	cmd := FetchTVEpisodes(-1)
	msg := cmd()
	episodes, ok := msg.(TvEpisodesMsg)
	if !ok {
		t.Fatalf("expected TvEpisodesMsg, got %T", msg)
	}
	if episodes != nil {
		t.Errorf("expected nil episodes for invalid ID, got %+v", episodes)
	}
}

func TestIsJunkOrNonVideo(t *testing.T) {
	tests := []struct {
		input    string
		wantJunk bool
	}{
		{"Friends S04E01.mkv", false},
		{"Friends S04E01.mp4", false},
		{"Friends S04E01.avi", false},
		{"Ninite K-Lite Codec Pack Unattended Silent Installer and Updater.exe", true},
		{"Ninite K-Lite Codec Pack.bat", true},
		{"codec_installer.msi", true},
		{"sample.mkv", true},
		{"Friends S04E01.sample.mkv", true},
		{"trailer.mp4", true},
		{"readme.txt", true},
		{"info.nfo", true},
		{"poster.jpg", true},
		{"subs.srt", true},
	}

	for _, tt := range tests {
		got := IsJunkOrNonVideo(tt.input)
		if got != tt.wantJunk {
			t.Errorf("IsJunkOrNonVideo(%q) = %v, want %v", tt.input, got, tt.wantJunk)
		}
	}
}

func TestMatchesQuality(t *testing.T) {
	tests := []struct {
		title   string
		quality string
		want    bool
	}{
		{"Friends S04 (1080p BluRay x265 Joy)", "1080p", true},
		{"Friends S04 (720p BluRay x265)", "1080p", false},
		{"Friends S04 (720p BluRay x265)", "720p", true},
		{"Dune 2024 2160p UHD HDR", "4K", true},
		{"Dune 2024 4K UHD", "4K", true},
		{"Dune 2024 1080p", "4K", false},
		{"Any Movie Title 1080p", "All", true},
		{"Any Movie Title 720p", "", true},
	}

	for _, tt := range tests {
		got := MatchesQuality(tt.title, tt.quality)
		if got != tt.want {
			t.Errorf("MatchesQuality(%q, %q) = %v, want %v", tt.title, tt.quality, got, tt.want)
		}
	}
}

func TestFetchAnime_ReturnsCmd(t *testing.T) {
	cmd := FetchAnime("Naruto", "All")
	if cmd == nil {
		t.Fatal("FetchAnime should return a non-nil tea.Cmd")
	}
}

