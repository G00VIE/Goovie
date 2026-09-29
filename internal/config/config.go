package config

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

var (
	ProwlarrAPIKey  = ""
	ProwlarrURL     = "http://localhost:9696"
	FlareSolverrURL = "http://localhost:8191"
	MinimumSeeders  = 5
	AnikotoBaseURL  = "https://anikototv.to"
	UserAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	DefaultTrackers = []string{
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://open.stealth.si:80/announce",
		"udp://tracker.torrent.eu.org:451/announce",
	}

	QualityOptions       = []string{"720p", "1080p", "4K"}
	AnimeTypeOptions     = []string{"All", "Series", "Movies", "OVA", "Special"}
	DownloadCacheOptions = []int{150, 256, 512, 1024}
	SkipIntroOptions     = []string{"prompt", "auto", "off"}

	DefaultDownloadCacheMB = 256
	DownloadCacheMB        = DefaultDownloadCacheMB
	SkipIntroMode          = "prompt"
	AutoResume             = true
	WatchedEpisodes        = make(map[string]bool)
)

type AppConfig struct {
	ProwlarrAPIKey  string `json:"prowlarrApiKey"`
	ProwlarrURL     string `json:"prowlarrUrl,omitempty"`
	FlareSolverrURL string `json:"flaresolverrUrl,omitempty"`
	MinimumSeeders  int    `json:"minimumSeeders,omitempty"`
	DownloadCacheMB int    `json:"downloadCacheMB,omitempty"`
	SkipIntroMode   string `json:"skipIntroMode,omitempty"`
	AutoResume      bool   `json:"autoResume,omitempty"`
}

type prowlarrXMLConfig struct {
	XMLName xml.Name `xml:"Config"`
	ApiKey  string   `xml:"ApiKey"`
}

// GoovieDir returns the per-user data directory used by goovie.
func GoovieDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "goovie_data"
	}
	return filepath.Join(home, ".goovie")
}

// TorrentCacheDir returns the persistent directory where torrent chunks and files are stored.
func TorrentCacheDir() string {
	dir := filepath.Join(GoovieDir(), "torrent_cache")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func AppConfigPath() string {
	dir := GoovieDir()
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "config.json")
}

func getGoovieConfigPath() string {
	return AppConfigPath()
}

// ReadConfig loads the raw persisted config without touching globals.
func ReadConfig() (AppConfig, error) {
	var cfg AppConfig
	data, err := os.ReadFile(AppConfigPath())
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

// WriteConfig saves the full AppConfig formatted with indentation.
func WriteConfig(cfg AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(AppConfigPath(), data, 0644)
}

func applyConfigToGlobals(cfg AppConfig) {
	if cfg.ProwlarrURL != "" {
		ProwlarrURL = cfg.ProwlarrURL
	}
	if cfg.FlareSolverrURL != "" {
		FlareSolverrURL = cfg.FlareSolverrURL
	}
	if cfg.MinimumSeeders > 0 {
		MinimumSeeders = cfg.MinimumSeeders
	}
	if cfg.DownloadCacheMB > 0 {
		DownloadCacheMB = cfg.DownloadCacheMB
	}
	if cfg.SkipIntroMode != "" {
		SkipIntroMode = cfg.SkipIntroMode
	}
	AutoResume = cfg.AutoResume
}

func LoadConfig() bool {
	loadWatchedEpisodes()
	cfg, err := ReadConfig()
	if err != nil {
		return false
	}
	applyConfigToGlobals(cfg)
	if cfg.ProwlarrAPIKey != "" {
		ProwlarrAPIKey = cfg.ProwlarrAPIKey
		return true
	}
	return false
}

// SaveConfig persists a Prowlarr API key while preserving all other configuration fields.
func SaveConfig(key string) error {
	cfg, err := ReadConfig()
	if err != nil {
		cfg = AppConfig{}
	}
	cfg.ProwlarrAPIKey = key
	ProwlarrAPIKey = key
	return WriteConfig(cfg)
}

// SaveFlareSolverrURL updates and persists the FlareSolverr URL.
func SaveFlareSolverrURL(u string) error {
	cfg, err := ReadConfig()
	if err != nil {
		cfg = AppConfig{}
	}
	cfg.FlareSolverrURL = u
	FlareSolverrURL = u
	return WriteConfig(cfg)
}

// GetDownloadCacheMB returns the configured torrent download cache size in MB (defaults to 256).
func GetDownloadCacheMB() int {
	if DownloadCacheMB <= 0 {
		return DefaultDownloadCacheMB
	}
	return DownloadCacheMB
}

// SaveDownloadCacheMB persists the download cache size in MB.
func SaveDownloadCacheMB(mb int) error {
	cfg, err := ReadConfig()
	if err != nil {
		cfg = AppConfig{}
	}
	cfg.DownloadCacheMB = mb
	DownloadCacheMB = mb
	return WriteConfig(cfg)
}

// SaveSkipIntroMode persists the skip intro mode ("prompt", "auto", "off").
func SaveSkipIntroMode(mode string) error {
	cfg, err := ReadConfig()
	if err != nil {
		cfg = AppConfig{}
	}
	cfg.SkipIntroMode = mode
	SkipIntroMode = mode
	return WriteConfig(cfg)
}

// SaveAutoResume persists the auto resume toggle.
func SaveAutoResume(enable bool) error {
	cfg, err := ReadConfig()
	if err != nil {
		cfg = AppConfig{}
	}
	cfg.AutoResume = enable
	AutoResume = enable
	return WriteConfig(cfg)
}

func WatchedHistoryPath() string {
	dir := GoovieDir()
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "watched.json")
}

func loadWatchedEpisodes() {
	data, err := os.ReadFile(WatchedHistoryPath())
	if err == nil {
		_ = json.Unmarshal(data, &WatchedEpisodes)
	}
	if WatchedEpisodes == nil {
		WatchedEpisodes = make(map[string]bool)
	}
}

func saveWatchedEpisodes() error {
	if WatchedEpisodes == nil {
		WatchedEpisodes = make(map[string]bool)
	}
	data, err := json.MarshalIndent(WatchedEpisodes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(WatchedHistoryPath(), data, 0644)
}

// IsEpisodeWatched checks if an episode key was marked as watched.
func IsEpisodeWatched(key string) bool {
	if len(WatchedEpisodes) == 0 {
		loadWatchedEpisodes()
	}
	return WatchedEpisodes[key]
}

// MarkEpisodeWatched records that an episode has been watched and persists it.
func MarkEpisodeWatched(key string) error {
	if WatchedEpisodes == nil {
		loadWatchedEpisodes()
	}
	WatchedEpisodes[key] = true
	return saveWatchedEpisodes()
}

func AutoDetectAPIKey() bool {
	var prowlarrConfigPaths []string

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		progData := os.Getenv("ProgramData")
		if progData == "" {
			progData = `C:\ProgramData`
		}
		prowlarrConfigPaths = append(prowlarrConfigPaths,
			filepath.Join(progData, "Prowlarr", "config.xml"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Prowlarr", "config.xml"),
			filepath.Join(os.Getenv("APPDATA"), "Prowlarr", "config.xml"),
		)
		if home != "" {
			prowlarrConfigPaths = append(prowlarrConfigPaths,
				filepath.Join(home, "AppData", "Local", "Prowlarr", "config.xml"),
				filepath.Join(home, "AppData", "Roaming", "Prowlarr", "config.xml"),
			)
		}
	} else if runtime.GOOS == "darwin" {
		if home != "" {
			prowlarrConfigPaths = append(prowlarrConfigPaths, filepath.Join(home, ".config", "Prowlarr", "config.xml"))
			prowlarrConfigPaths = append(prowlarrConfigPaths, filepath.Join(home, "Library", "Application Support", "Prowlarr", "config.xml"))
		}
	} else { // linux
		if home != "" {
			prowlarrConfigPaths = append(prowlarrConfigPaths, filepath.Join(home, ".config", "Prowlarr", "config.xml"))
		}
		prowlarrConfigPaths = append(prowlarrConfigPaths, "/var/lib/prowlarr/config.xml")
	}

	for _, p := range prowlarrConfigPaths {
		data, err := os.ReadFile(p)
		if err == nil {
			trimmed := bytes.TrimSpace(data)
			if len(trimmed) == 0 || bytes.ContainsRune(trimmed, 0) {
				continue
			}
			var xmlCfg prowlarrXMLConfig
			if err := xml.Unmarshal(trimmed, &xmlCfg); err == nil && strings.TrimSpace(xmlCfg.ApiKey) != "" {
				ProwlarrAPIKey = strings.TrimSpace(xmlCfg.ApiKey)
				return true
			}
		}
	}
	return false
}

func InitConfig() bool {
	if envURL := os.Getenv("PROWLARR_URL"); envURL != "" {
		ProwlarrURL = envURL
	}
	if envKey := os.Getenv("PROWLARR_API_KEY"); envKey != "" {
		ProwlarrAPIKey = envKey
		return true
	}

	if LoadConfig() {
		return true
	}
	return AutoDetectAPIKey()
}

// ResumeState represents the last watched stream session for 1-click continuation.
type ResumeState struct {
	MediaType    string  `json:"mediaType"` // "movie", "tv", "anime", "asian"
	Title        string  `json:"title"`
	Season       int     `json:"season,omitempty"`
	Episode      int     `json:"episode,omitempty"`
	EpisodeTitle string  `json:"episodeTitle,omitempty"`
	Target       string  `json:"target"` // Magnet or Stream URL
	FileIndex    string  `json:"fileIndex,omitempty"`
	Referer      string  `json:"referer,omitempty"`
	SubtitleURL  string  `json:"subtitleURL,omitempty"`
	TimePos      float64 `json:"timePos,omitempty"`
	Duration     float64 `json:"duration,omitempty"`
	UpdatedAt    string  `json:"updatedAt"`
}

func ResumeStatePath() string {
	return filepath.Join(GoovieDir(), "resume.json")
}

// LoadResumeState reads the persistent resume state, syncing any recent timestamp from last_pos.json.
func LoadResumeState() (*ResumeState, error) {
	data, err := os.ReadFile(ResumeStatePath())
	if err != nil {
		return nil, err
	}
	var state ResumeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	lastPosPath := filepath.Join(GoovieDir(), "last_pos.json")
	if posData, err := os.ReadFile(lastPosPath); err == nil {
		var pos struct {
			TimePos  float64 `json:"timePos"`
			Duration float64 `json:"duration"`
		}
		if json.Unmarshal(posData, &pos) == nil && pos.TimePos > 0 {
			state.TimePos = pos.TimePos
			if pos.Duration > 0 {
				state.Duration = pos.Duration
			}
		}
	}
	return &state, nil
}

// SaveResumeState persists the current stream state for 1-click continuation.
func SaveResumeState(state ResumeState) error {
	if state.UpdatedAt == "" {
		state.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ResumeStatePath(), data, 0644)
}

// ClearResumeState removes the saved resume state.
func ClearResumeState() error {
	_ = os.Remove(filepath.Join(GoovieDir(), "last_pos.json"))
	return os.Remove(ResumeStatePath())
}

// GetTorrentCacheSize calculates the total bytes used by persistent torrent cache.
func GetTorrentCacheSize() (int64, error) {
	var totalSize int64
	dir := TorrentCacheDir()
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize, err
}

// ClearTorrentCache purges all files in the torrent cache directory.
func ClearTorrentCache() error {
	dir := TorrentCacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}
	return nil
}

// PruneTorrentCache removes older torrent files if the cache size exceeds maxSizeGB, preserving active items.
func PruneTorrentCache(maxSizeGB int, keepActiveName string) error {
	if maxSizeGB <= 0 {
		maxSizeGB = 10
	}
	maxBytes := int64(maxSizeGB) * 1024 * 1024 * 1024
	dir := TorrentCacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	type itemInfo struct {
		path    string
		size    int64
		modTime time.Time
	}
	var items []itemInfo
	var totalSize int64

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		var itemSize int64
		_ = filepath.Walk(fullPath, func(_ string, f os.FileInfo, err error) error {
			if err == nil && !f.IsDir() {
				itemSize += f.Size()
			}
			return nil
		})
		totalSize += itemSize
		items = append(items, itemInfo{
			path:    fullPath,
			size:    itemSize,
			modTime: info.ModTime(),
		})
	}

	if totalSize <= maxBytes {
		return nil
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].modTime.Before(items[j].modTime)
	})

	for _, item := range items {
		if keepActiveName != "" && strings.Contains(item.path, keepActiveName) {
			continue
		}
		_ = os.RemoveAll(item.path)
		totalSize -= item.size
		if totalSize <= maxBytes {
			break
		}
	}
	return nil
}

// FormatBytes formats byte count into human-readable size (e.g. 245.3 MB, 1.2 GB).
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

