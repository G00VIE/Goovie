package config

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"runtime"
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

	QualityOptions   = []string{"720p", "1080p", "4K"}
	AnimeTypeOptions = []string{"All", "Series", "Movies", "OVA", "Special"}
)

type AppConfig struct {
	ProwlarrAPIKey  string `json:"prowlarrApiKey"`
	ProwlarrURL     string `json:"prowlarrUrl,omitempty"`
	FlareSolverrURL string `json:"flaresolverrUrl,omitempty"`
	MinimumSeeders  int    `json:"minimumSeeders,omitempty"`
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
}

func LoadConfig() bool {
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

func AutoDetectAPIKey() bool {
	var prowlarrConfigPaths []string

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		prowlarrConfigPaths = append(prowlarrConfigPaths, `C:\ProgramData\Prowlarr\config.xml`)
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
			var xmlCfg prowlarrXMLConfig
			if err := xml.Unmarshal(data, &xmlCfg); err == nil && xmlCfg.ApiKey != "" {
				ProwlarrAPIKey = xmlCfg.ApiKey
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
