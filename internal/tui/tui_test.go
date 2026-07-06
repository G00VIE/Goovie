package tui

import (
	"image/color"
	"strings"
	"testing"

	"bubble-stream/internal/prowlarr"
)

// simpleRGBA implements color.Color for testing luminance
type simpleRGBA struct {
	r, g, b, a uint32
}

func (c simpleRGBA) RGBA() (uint32, uint32, uint32, uint32) {
	return c.r, c.g, c.b, c.a
}

func colorFromRGBA(r, g, b uint8) color.Color {
	return simpleRGBA{
		r: uint32(r) << 8,
		g: uint32(g) << 8,
		b: uint32(b) << 8,
		a: uint32(255) << 8,
	}
}

// --- simplifyTVFiles tests ---

func TestSimplifyTVFiles_BasicMatch(t *testing.T) {
	files := []string{
		"0  showname.s01e01.1080p.mkv",
		"1  showname.s01e02.1080p.mkv",
		"2  showname.s01e03.1080p.mkv",
	}
	episodes := []prowlarr.TVMazeEpisode{
		{Number: 1, Name: "Pilot"},
		{Number: 2, Name: "Chapter Two"},
		{Number: 3, Name: "Chapter Three"},
	}

	result := simplifyTVFiles(files, episodes)

	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	// Episode 1 should be renamed
	if !contains(result, "0 Episode 1: Pilot") {
		t.Errorf("expected episode 1 to be renamed, got files: %v", result)
	}
	if !contains(result, "1 Episode 2: Chapter Two") {
		t.Errorf("expected episode 2 to be renamed, got files: %v", result)
	}
	if !contains(result, "2 Episode 3: Chapter Three") {
		t.Errorf("expected episode 3 to be renamed, got files: %v", result)
	}
}

func TestSimplifyTVFiles_CaseInsensitive(t *testing.T) {
	files := []string{
		"0  Show.S01E01.mkv",
		"1  SHOW.S01E02.MKV",
	}
	episodes := []prowlarr.TVMazeEpisode{
		{Number: 1, Name: "First"},
		{Number: 2, Name: "Second"},
	}

	result := simplifyTVFiles(files, episodes)

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if !contains(result, "0 Episode 1: First") {
		t.Errorf("case insensitive match failed, got: %v", result)
	}
}

func TestSimplifyTVFiles_EpPrefix(t *testing.T) {
	files := []string{
		"0  show.ep01.1080p.mkv",
	}
	episodes := []prowlarr.TVMazeEpisode{
		{Number: 1, Name: "Beginnings"},
	}

	result := simplifyTVFiles(files, episodes)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if !contains(result, "0 Episode 1: Beginnings") {
		t.Errorf("ep prefix match failed, got: %v", result)
	}
}

func TestSimplifyTVFiles_ZeroPadded(t *testing.T) {
	files := []string{
		"0  show.s01e001.1080p.mkv",
	}
	episodes := []prowlarr.TVMazeEpisode{
		{Number: 1, Name: "Start"},
	}

	result := simplifyTVFiles(files, episodes)

	if !contains(result, "0 Episode 1: Start") {
		t.Errorf("zero-padded episode match failed, got: %v", result)
	}
}

func TestSimplifyTVFiles_NoMatch(t *testing.T) {
	files := []string{
		"0  somefile.mkv",
		"1  another.mp4",
	}
	episodes := []prowlarr.TVMazeEpisode{
		{Number: 1, Name: "Pilot"},
	}

	result := simplifyTVFiles(files, episodes)

	// Files without episode patterns should be returned as-is
	if len(result) != 2 {
		t.Fatalf("expected 2 results (unchanged), got %d", len(result))
	}
	if result[0] != "0  somefile.mkv" {
		t.Errorf("non-matching file should be unchanged, got: %s", result[0])
	}
}

func TestSimplifyTVFiles_NoEpisodes(t *testing.T) {
	files := []string{
		"0  show.s01e01.1080p.mkv",
	}

	result := simplifyTVFiles(files, nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	// With no episode data, file stays unchanged
	if result[0] != "0  show.s01e01.1080p.mkv" {
		t.Errorf("file should be unchanged with no episode data, got: %s", result[0])
	}
}

func TestSimplifyTVFiles_EmptyInput(t *testing.T) {
	result := simplifyTVFiles([]string{}, []prowlarr.TVMazeEpisode{})
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %d items", len(result))
	}
}

// --- view helper tests ---

func TestCenterStr(t *testing.T) {
	tests := []struct {
		input string
		width int
	}{
		{"hello", 10},
		{"hi", 5},
		{"longer than width", 5},
		{"exact", 5},
	}

	for _, tt := range tests {
		result := centerStr(tt.input, tt.width)
		// centerStr left-pads with (width-len(s))/2 spaces, prepending them to s
		if len(tt.input) >= tt.width {
			// Input is at least as wide as width → returned unchanged
			if result != tt.input {
				t.Errorf("centerStr(%q, %d) = %q, want %q (unchanged)", tt.input, tt.width, result, tt.input)
			}
			continue
		}
		// Otherwise, it should be the input prefixed with some padding spaces
		pad := (tt.width - len(tt.input)) / 2
		expected := strings.Repeat(" ", pad) + tt.input
		if result != expected {
			t.Errorf("centerStr(%q, %d) = %q, want %q", tt.input, tt.width, result, expected)
		}
	}
}

func TestCenterStr_ShorterThanInput(t *testing.T) {
	result := centerStr("hello", 3)
	if result != "hello" {
		t.Errorf("centerStr should return input unchanged when shorter, got %q", result)
	}
}

func TestCenterStr_EqualLength(t *testing.T) {
	result := centerStr("abc", 3)
	if result != "abc" {
		t.Errorf("centerStr should return input unchanged when equal, got %q", result)
	}
}

func TestCenterStr_ExactPadding(t *testing.T) {
	// centerStr prepends (width-len(s))/2 spaces to the input.
	// For "a" (len 1) at width 5: pad = (5-1)/2 = 2 → "  a" (len 3)
	result := centerStr("a", 5)
	expected := "  a"
	if result != expected {
		t.Errorf("centerStr(\"a\", 5) = %q, want %q", result, expected)
	}
}

// --- renderDatabaseMatchCamera ---

func TestRenderDatabaseMatchCamera_Basic(t *testing.T) {
	items := [][2]string{
		{"Movie One", "2024"},
		{"Movie Two", "2023"},
		{"Movie Three", "2022"},
	}

	result := renderDatabaseMatchCamera("SELECT DATABASE MATCH", items, 0, "")

		if result == "" {
			t.Error("renderDatabaseMatchCamera should return non-empty string")
		}
		// Should contain cursor position indicator
		if !strings.Contains(result, "1 / 3") {
		t.Error("should contain '1 / 3' counter")
	}
}

func TestRenderDatabaseMatchCamera_WithSearch(t *testing.T) {
	items := [][2]string{
		{"Movie One", "2024"},
	}

	result := renderDatabaseMatchCamera("SELECT DATABASE MATCH", items, 0, "1")

	if result == "" {
		t.Error("should return non-empty with search string")
	}
}

func TestRenderDatabaseMatchCamera_Empty(t *testing.T) {
	items := [][2]string{}

	result := renderDatabaseMatchCamera("TITLE", items, 0, "")
	// Should still produce header at minimum
	if result == "" {
		t.Error("should return non-empty even with empty items")
	}
}

// --- renderMenuCamera ---

func TestRenderMenuCamera_Basic(t *testing.T) {
	items := []string{"Option A", "Option B", "Option C"}

	result := renderMenuCamera(items, 1, 30, "Choose:")

	if result == "" {
		t.Error("renderMenuCamera should return non-empty string")
	}
}

func TestRenderMenuCamera_SingleItem(t *testing.T) {
	items := []string{"Only Option"}

	result := renderMenuCamera(items, 0, 20, "Pick:")

	if result == "" {
		t.Error("should work with single item")
	}
}

// --- LoadingPhrases ---

func TestLoadingPhrases_NotEmpty(t *testing.T) {
	if len(LoadingPhrases) == 0 {
		t.Error("LoadingPhrases should not be empty")
	}
}

// --- State constants ---

func TestStateConstants_Unique(t *testing.T) {
	states := map[int]bool{
		StateFrontPage:       true,
		StateModeSelect:     true,
		StateOriginSelect:   true,
		StateAnimeTypeSelect: true,
		StateSearch:          true,
		StateLoading:         true,
		StateMovieSelect:     true,
		StateTVShowSelect:    true,
		StateTVSeasonSelect:  true,
		StateQuality:         true,
		StateList:            true,
		StateTVFileSelect:    true,
		StateAnimeSelect:     true,
		StateAnikotoShowSelect: true,
		StateAnikotoEpSelect: true,
		StateAnikotoModeSelect: true,
		StateAsianShowSelect: true,
		StateAsianEpSelect:   true,
		StateCheckingAPIKey:  true,
		StateSetupAPIKey:     true,
		StateLoadingTorrent:  true,
	}

	if len(states) != 21 {
		t.Errorf("expected 21 unique state constants, got %d", len(states))
	}
}

// --- updateDBMatchCursor tests ---

func TestUpdateDBMatchCursor_MovieSelect(t *testing.T) {
	m := Model{
		state:          StateMovieSelect,
		cursor:         0,
		cinemetaMovies: make([]prowlarr.CinemetaMovie, 10),
		dbMatchSearch:  "5",
	}

	m = m.updateDBMatchCursor()
	// "5" => target = 5-1 = 4
	if m.cursor != 4 {
		t.Errorf("expected cursor=4, got %d", m.cursor)
	}
}

func TestUpdateDBMatchCursor_OutOfBounds(t *testing.T) {
	m := Model{
		state:          StateMovieSelect,
		cursor:         0,
		cinemetaMovies: make([]prowlarr.CinemetaMovie, 3),
		dbMatchSearch:  "99",
	}

	m = m.updateDBMatchCursor()
	// Should clamp to maxIdx = 2
	if m.cursor != 2 {
		t.Errorf("expected cursor clamped to 2, got %d", m.cursor)
	}
}

func TestUpdateDBMatchCursor_Zero(t *testing.T) {
	m := Model{
		state:          StateMovieSelect,
		cursor:         5,
		cinemetaMovies: make([]prowlarr.CinemetaMovie, 10),
		dbMatchSearch:  "0",
	}

	m = m.updateDBMatchCursor()
	// "0" => target = -1, clamped to 0
	if m.cursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.cursor)
	}
}

func TestUpdateDBMatchCursor_EmptySearch(t *testing.T) {
	m := Model{
		cursor:        5,
		dbMatchSearch: "",
	}

	m = m.updateDBMatchCursor()
	if m.cursor != 0 {
		t.Errorf("empty search should reset cursor to 0, got %d", m.cursor)
	}
}

func TestUpdateDBMatchCursor_NonNumeric(t *testing.T) {
	m := Model{
		cursor:        2,
		dbMatchSearch: "abc",
		cinemetaMovies: make([]prowlarr.CinemetaMovie, 5),
		state:         StateMovieSelect,
	}

	m = m.updateDBMatchCursor()
	// Non-numeric should leave cursor unchanged (strconv.Atoi fails)
	if m.cursor != 2 {
		t.Errorf("non-numeric search should not change cursor, got %d", m.cursor)
	}
}

// --- updateTVFileCursor tests ---

func TestUpdateTVFileCursor_Basic(t *testing.T) {
	m := Model{
		tvFiles: []string{
			"0 Episode 1: Pilot",
			"1 Episode 2: Chapter Two",
			"2 Episode 3: Chapter Three",
		},
		tvFileSearch: "2",
	}

	m = m.updateTVFileCursor()
	// "2" builds "Episode 2:" which matches the SECOND entry (index 1)
	if m.cursor != 1 {
		t.Errorf("expected cursor=1 (the 'Episode 2:' entry), got %d", m.cursor)
	}
}

func TestUpdateTVFileCursor_MatchesEpisode3(t *testing.T) {
	m := Model{
		tvFiles: []string{
			"0 Episode 1: Pilot",
			"1 Episode 2: Chapter Two",
			"2 Episode 3: Chapter Three",
		},
		tvFileSearch: "3",
	}

	m = m.updateTVFileCursor()
	// "3" builds "Episode 3:" which matches the THIRD entry (index 2)
	if m.cursor != 2 {
		t.Errorf("expected cursor=2 (the 'Episode 3:' entry), got %d", m.cursor)
	}
}

func TestUpdateTVFileCursor_NoMatch(t *testing.T) {
	m := Model{
		tvFiles:      []string{"0 Episode 1: Pilot"},
		tvFileSearch: "9",
		cursor:       0,
	}

	m = m.updateTVFileCursor()
	// Should remain unchanged when no match
	if m.cursor != 0 {
		t.Errorf("expected cursor=0 when no match, got %d", m.cursor)
	}
}

func TestUpdateTVFileCursor_EmptySearch(t *testing.T) {
	m := Model{
		tvFiles:      []string{"0 Episode 1: Pilot"},
		tvFileSearch: "",
		cursor:       5,
	}

	m = m.updateTVFileCursor()
	if m.cursor != 5 {
		t.Errorf("empty search should not change cursor, got %d", m.cursor)
	}
}

// --- helper ---

func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// --- luminance ---

func TestLuminance_Black(t *testing.T) {
	// Black = 0 luminance
	l := luminance(colorFromRGBA(0, 0, 0))
	if l != 0 {
		t.Errorf("black luminance should be 0, got %f", l)
	}
}

func TestLuminance_White(t *testing.T) {
	l := luminance(colorFromRGBA(255, 255, 255))
	if l != 255 {
		t.Errorf("white luminance should be 255, got %f", l)
	}
}

func TestLuminance_Red(t *testing.T) {
	l := luminance(colorFromRGBA(255, 0, 0))
	if l < 75 || l > 77 {
		t.Errorf("red luminance should be ~76.245, got %f", l)
	}
}

func TestLuminance_GreenDominant(t *testing.T) {
	// Green channel has highest weight (0.587)
	l := luminance(colorFromRGBA(0, 255, 0))
	if l < 149 || l > 150 {
		t.Errorf("green luminance should be ~149.685, got %f", l)
	}
}
