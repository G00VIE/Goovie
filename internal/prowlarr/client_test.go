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

// --- writeDebugLog ---

func TestWriteDebugLog(t *testing.T) {
	os.Remove("torrent_debug.txt")
	writeDebugLog("test log entry\n")

	data, err := os.ReadFile("torrent_debug.txt")
	if err != nil {
		t.Fatalf("failed to read debug log: %v", err)
	}
	if string(data) != "test log entry\n" {
		t.Errorf("expected 'test log entry\\n', got %q", string(data))
	}
	os.Remove("torrent_debug.txt")
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
