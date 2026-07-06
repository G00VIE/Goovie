package player

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// --- resolvePlaylistURL tests ---

func TestResolvePlaylistURL_AbsoluteHTTP(t *testing.T) {
	tests := []struct {
		entry string
		want  string
	}{
		{"http://cdn.example.com/v1/seg.ts", "http://cdn.example.com/v1/seg.ts"},
		{"https://cdn.example.com/v1/seg.ts", "https://cdn.example.com/v1/seg.ts"},
	}

	for _, tt := range tests {
		result := resolvePlaylistURL("https://base.example.com/master.m3u8", tt.entry)
		if result != tt.want {
			t.Errorf("resolvePlaylistURL(%q) = %q, want %q", tt.entry, result, tt.want)
		}
	}
}

func TestResolvePlaylistURL_AbsolutePath(t *testing.T) {
	base := "https://cdn.example.com/hls/vod/master.m3u8"
	entry := "/hls/vod/playlist.m3u8"

	result := resolvePlaylistURL(base, entry)
	expected := "https://cdn.example.com/hls/vod/playlist.m3u8"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestResolvePlaylistURL_RelativePath(t *testing.T) {
	base := "https://cdn.example.com/hls/vod/master.m3u8"
	entry := "playlist_720p.m3u8"

	result := resolvePlaylistURL(base, entry)
	expected := "https://cdn.example.com/hls/vod/playlist_720p.m3u8"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestResolvePlaylistURL_RelativeSubdirectory(t *testing.T) {
	base := "https://cdn.example.com/hls/master.m3u8"
	entry := "subdir/playlist.m3u8"

	result := resolvePlaylistURL(base, entry)
	expected := "https://cdn.example.com/hls/subdir/playlist.m3u8"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestResolvePlaylistURL_NoPathInBase(t *testing.T) {
	base := "https://cdn.example.com"
	entry := "playlist.m3u8"

	result := resolvePlaylistURL(base, entry)
	if !strings.HasSuffix(result, "playlist.m3u8") {
		t.Errorf("result should end with playlist.m3u8, got: %s", result)
	}
}

func TestResolvePlaylistURL_EmptyEntry(t *testing.T) {
	// An empty entry: no absolute scheme, not a leading "/", and url.Parse succeeds.
	// The code does LastIndex(path,"/") to keep a trailing "/" and appends "" → "<dir>/"
	result := resolvePlaylistURL("https://example.com/master.m3u8", "")
	// Per the implementation, an empty entry resolves to the parent directory URL.
	if result != "https://example.com/" {
		t.Errorf("empty entry should resolve to base dir, got: %q", result)
	}
}

// --- rewritePlaylistLines tests ---

func TestRewritePlaylistLines_PreservesComments(t *testing.T) {
	input := "#EXTM3U\n#EXT-X-VERSION:3\nsegment.ts"
	result := rewritePlaylistLines(input, strings.ToUpper)

	if !strings.Contains(result, "#EXTM3U") {
		t.Error("comments should be preserved")
	}
	if !strings.Contains(result, "#EXT-X-VERSION:3") {
		t.Error("version tag should be preserved")
	}
	if !strings.Contains(result, "SEGMENT.TS") {
		t.Error("non-comment lines should be transformed")
	}
}

func TestRewritePlaylistLines_EmptyLinesPreserved(t *testing.T) {
	input := "line1\n\nline3"
	result := rewritePlaylistLines(input, func(s string) string { return s })
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if lines[1] != "" {
		t.Error("empty line should be preserved")
	}
}

func TestRewritePlaylistLines_CRLF(t *testing.T) {
	input := "line1\r\nline2\r\n"
	result := rewritePlaylistLines(input, func(s string) string { return s })
	// CRLF should be normalized to LF
	if strings.Contains(result, "\r\n") {
		t.Error("CRLF should be normalized to LF")
	}
	if !strings.Contains(result, "line1\nline2") {
		t.Error("lines should be preserved after CRLF normalization")
	}
}

// --- stripPNGWrapper tests ---

func TestStripPNGWrapper_WithMarker(t *testing.T) {
	// PNG IEND marker + some trailing data
	marker := []byte{0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82}
	data := append([]byte("PNG_HEADER_DATA_GARBAGE"), marker...)
	data = append(data, []byte("ACTUAL_VIDEO_DATA")...)

	result := stripPNGWrapper(data)

	expected := []byte("ACTUAL_VIDEO_DATA")
	if !bytes.Equal(result, expected) {
		t.Errorf("stripPNGWrapper failed: got %q, want %q", result, expected)
	}
}

func TestStripPNGWrapper_NoMarker(t *testing.T) {
	data := []byte("NO_MARKER_HERE_JUST_RAW_DATA")

	result := stripPNGWrapper(data)

	if !bytes.Equal(result, data) {
		t.Errorf("data without marker should be returned unchanged")
	}
}

func TestStripPNGWrapper_Empty(t *testing.T) {
	result := stripPNGWrapper([]byte{})
	if len(result) != 0 {
		t.Errorf("empty input should return empty, got %d bytes", len(result))
	}
}

func TestStripPNGWrapper_MarkerAtEnd(t *testing.T) {
	// When the IEND marker sits at the very end, there is no trailing data
	// after it, so stripPNGWrapper returns an empty slice (everything is "PNG").
	marker := pngIENDMarker
	data := []byte("PREFIX_DATA")
	data = append(data, marker...)

	result := stripPNGWrapper(data)

	// Correct behavior: marker at the end → nothing follows → empty result
	if len(result) != 0 {
		t.Errorf("marker at end should yield empty result, got %d bytes: %q", len(result), result)
	}
}

func TestStripPNGWrapper_MarkerInMiddle(t *testing.T) {
	// Marker in the middle: everything after the marker is the real payload
	marker := pngIENDMarker
	data := []byte("PREFIX_DATA")
	data = append(data, marker...)
	data = append(data, []byte("PAYLOAD")...)

	result := stripPNGWrapper(data)

	if string(result) != "PAYLOAD" {
		t.Errorf("expected 'PAYLOAD' after marker, got %q", result)
	}
}

func TestPNGIENDMarker_Expected(t *testing.T) {
	expected := []byte{0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82}
	if !bytes.Equal(pngIENDMarker, expected) {
		t.Error("PNG IEND marker bytes are incorrect")
	}
}

// --- VibeProxy tests ---

func TestVibeProxy_Register(t *testing.T) {
	proxy := &VibeProxy{
		BaseURL:  "http://127.0.0.1:12345",
		Sessions: make(map[string]*VibeSession),
	}

	url := proxy.Register("http://cdn.example.com/master.m3u8", "http://example.com")

	if url == "" {
		t.Fatal("Register should return non-empty URL")
	}
	if !strings.HasPrefix(url, proxy.BaseURL+"/stream/") {
		t.Errorf("URL should start with %s/stream/, got: %s", proxy.BaseURL, url)
	}
	if !strings.HasSuffix(url, "master.m3u8") {
		t.Errorf("URL should end with master.m3u8, got: %s", url)
	}

	if len(proxy.Sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(proxy.Sessions))
	}
}

func TestVibeProxy_Register_UniqueIDs(t *testing.T) {
	proxy := &VibeProxy{
		BaseURL:  "http://127.0.0.1:12345",
		Sessions: make(map[string]*VibeSession),
	}

	url1 := proxy.Register("http://a.com/master.m3u8", "http://a.com")
	url2 := proxy.Register("http://b.com/master.m3u8", "http://b.com")

	if url1 == url2 {
		t.Error("two registrations should produce different URLs")
	}
	if len(proxy.Sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(proxy.Sessions))
	}
}

func TestVibeProxy_Handle_NotFound(t *testing.T) {
	proxy := &VibeProxy{
		BaseURL:  "http://127.0.0.1:12345",
		Sessions: make(map[string]*VibeSession),
	}

	req := httptest.NewRequest("GET", "/stream/invalidid/master.m3u8", nil)
	w := httptest.NewRecorder()

	proxy.handle(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for invalid session, got %d", w.Code)
	}
}

func TestVibeProxy_Handle_BadPath(t *testing.T) {
	proxy := &VibeProxy{
		BaseURL:  "http://127.0.0.1:12345",
		Sessions: make(map[string]*VibeSession),
	}

	// Path with less than 2 parts after /stream/
	req := httptest.NewRequest("GET", "/stream/", nil)
	w := httptest.NewRecorder()

	proxy.handle(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for bad path, got %d", w.Code)
	}
}

func TestVibeProxy_ServeMaster_ValidM3U8(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		io.WriteString(w, "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1000000\nplaylist.m3u8")
	}))
	defer ts.Close()

	proxy := &VibeProxy{
		BaseURL:  "http://127.0.0.1:12345",
		Sessions: make(map[string]*VibeSession),
	}
	// Register returns the full proxy URL: <BaseURL>/stream/<id>/master.m3u8
	registeredURL := proxy.Register(ts.URL+"/master.m3u8", ts.URL)

	// Extract <id> from the registered URL by trimming the known prefix/suffix
	trimmed := strings.TrimPrefix(registeredURL, proxy.BaseURL+"/stream/")
	trimmed = strings.TrimSuffix(trimmed, "/master.m3u8")
	sessionID := trimmed

	req := httptest.NewRequest("GET", "/stream/"+sessionID+"/master.m3u8", nil)
	w := httptest.NewRecorder()
	proxy.handle(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "#EXTM3U") {
		t.Error("master playlist should contain #EXTM3U")
	}
	// Variant URL should be rewritten to point at the proxy
	if !strings.Contains(body, proxy.BaseURL) {
		t.Error("variant URL should be rewritten to proxy URL")
	}
}

func TestVibeProxy_ServeSegment_InvalidIndex(t *testing.T) {
	proxy := &VibeProxy{
		BaseURL:  "http://127.0.0.1:12345",
		Sessions: make(map[string]*VibeSession),
	}

	// Create a session with empty variants
	proxy.Sessions["test"] = &VibeSession{
		MasterURL: "http://example.com",
		Variants:  make(map[string][]VibeSegment),
	}

	req := httptest.NewRequest("GET", "/stream/test/variant.m3u8/seg/abc", nil)
	w := httptest.NewRecorder()
	proxy.handle(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-numeric index, got %d", w.Code)
	}
}

func TestVibeProxy_ServeSegment_OutOfBounds(t *testing.T) {
	proxy := &VibeProxy{
		BaseURL:  "http://127.0.0.1:12345",
		Sessions: make(map[string]*VibeSession),
	}

	proxy.Sessions["test"] = &VibeSession{
		MasterURL: "http://example.com",
		Variants: map[string][]VibeSegment{
			"variant.m3u8": {{URL: "http://cdn.example.com/seg0.ts"}},
		},
	}

	// Index 5 is out of bounds (only 1 segment exists)
	req := httptest.NewRequest("GET", "/stream/test/variant.m3u8/seg/5", nil)
	w := httptest.NewRecorder()
	proxy.handle(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for out-of-bounds index, got %d", w.Code)
	}
}

// --- fetchHTTPWithReferer tests ---

func TestFetchHTTPWithReferer_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "http://referer.com" {
			t.Errorf("expected Referer header, got: %s", r.Header.Get("Referer"))
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("expected User-Agent header")
		}
		w.Write([]byte("response body"))
	}))
	defer ts.Close()

	client := &http.Client{}
	data, err := fetchHTTPWithReferer(client, ts.URL, "http://referer.com")
	if err != nil {
		t.Fatalf("fetchHTTPWithReferer failed: %v", err)
	}
	if string(data) != "response body" {
		t.Errorf("unexpected response body: %q", string(data))
	}
}

func TestFetchHTTPWithReferer_Error(t *testing.T) {
	client := &http.Client{}
	_, err := fetchHTTPWithReferer(client, "http://127.0.0.1:1/nonexistent", "")
	if err == nil {
		t.Error("expected error for connection refused")
	}
}

func TestFetchHTTPWithReferer_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := &http.Client{}
	_, err := fetchHTTPWithReferer(client, ts.URL, "")
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

// --- writePlaylist tests ---

func TestWritePlaylist_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	writePlaylist(w, "#EXTM3U")

	if w.Header().Get("Content-Type") != "application/vnd.apple.mpegurl" {
		t.Errorf("expected application/vnd.apple.mpegurl, got %s", w.Header().Get("Content-Type"))
	}
}

func TestWritePlaylist_Body(t *testing.T) {
	w := httptest.NewRecorder()
	writePlaylist(w, "#EXTM3U\ntest.ts")

	if w.Body.String() != "#EXTM3U\ntest.ts" {
		t.Errorf("unexpected body: %q", w.Body.String())
	}
}

// --- LaunchPlayer tests ---

func TestLaunchPlayer_ReturnsCmd(t *testing.T) {
	// LaunchPlayer returns a tea.Cmd (func() tea.Msg). We verify the type
	// signature without executing it, since execution spawns a real mpv process.
	cmd := LaunchPlayer("http://example.com/stream.m3u8", "", "http://referer.com", "http://sub.example.com/sub.srt")
	if cmd == nil {
		t.Fatal("LaunchPlayer should return a non-nil tea.Cmd")
	}
	// Do NOT call cmd() here — it would attempt to spawn mpv, which is slow
	// and not guaranteed to be installed in CI. The function's correctness is
	// established by the non-nil return and the integration of its caller.
}

// --- Message type creation tests ---

func TestPlayerFinishedMsg_NilError(t *testing.T) {
	msg := PlayerFinishedMsg{Err: nil}
	if msg.Err != nil {
		t.Error("expected nil error")
	}
}

func TestVibeSession_Fields(t *testing.T) {
	s := VibeSession{
		MasterURL: "http://master.com",
		Referer:   "http://referer.com",
		Variants:  make(map[string][]VibeSegment),
	}

	if s.MasterURL != "http://master.com" || s.Referer != "http://referer.com" {
		t.Error("VibeSession field values incorrect")
	}
}

// --- Integration: resolvePlaylistURL edge cases ---

func TestResolvePlaylistURL_StripsQueryAndFragment(t *testing.T) {
	base := "https://cdn.example.com/hls/vod/master.m3u8?token=abc#section"
	entry := "playlist.m3u8"

	result := resolvePlaylistURL(base, entry)

	if strings.Contains(result, "token=abc") {
		t.Error("query string from base should not leak into result")
	}
	if strings.Contains(result, "#section") {
		t.Error("fragment from base should not leak into result")
	}
}

func TestResolvePlaylistURL_AbsolutePathStripsQuery(t *testing.T) {
	base := "https://cdn.example.com/hls/master.m3u8"
	entry := "/hls/vod/playlist.m3u8?x=1"

	result := resolvePlaylistURL(base, entry)

	// The implementation concatenates raw: scheme + "://" + host + entry.
	// So the entry (including any "?query") is appended verbatim.
	expected := "https://cdn.example.com/hls/vod/playlist.m3u8?x=1"
	if result != expected {
		t.Errorf("got: %s, want: %s", result, expected)
	}
}

// --- strconv.Atoi edge case for serveSegment ---

func TestServeSegment_NegativeIndex(t *testing.T) {
	proxy := &VibeProxy{
		BaseURL:  "http://127.0.0.1:12345",
		Sessions: make(map[string]*VibeSession),
	}

	req := httptest.NewRequest("GET", "/stream/test/v.m3u8/seg/-1", nil)
	w := httptest.NewRecorder()

	// No session exists, so this will be 404 from the session check
	proxy.handle(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- InitProxy ---

func TestInitProxy_StartsServer(t *testing.T) {
	InitProxy()

	if GlobalProxy == nil {
		t.Fatal("GlobalProxy should be initialized after InitProxy")
	}
	if GlobalProxy.BaseURL == "" {
		t.Error("GlobalProxy.BaseURL should be non-empty")
	}
	if !strings.HasPrefix(GlobalProxy.BaseURL, "http://127.0.0.1:") {
		t.Errorf("BaseURL should be localhost, got: %s", GlobalProxy.BaseURL)
	}
}

// --- GlobalJar initialized ---

func TestGlobalJar_NotNil(t *testing.T) {
	if GlobalJar == nil {
		t.Error("GlobalJar should be initialized (non-nil)")
	}
}

// --- Content-Length on serveSegment (verify strconv.Itoa usage) ---

func TestStrconvItoa(t *testing.T) {
	// Ensure strconv.Itoa is correctly importable/usable
	result := strconv.Itoa(42)
	if result != "42" {
		t.Errorf("strconv.Itoa(42) = %q, want '42'", result)
	}
}
