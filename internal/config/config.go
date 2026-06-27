package config

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"runtime"
)

var (
	ProwlarrAPIKey = ""
	ProwlarrURL    = "http://localhost:9696"
	MinimumSeeders = 5
	AnikotoBaseURL = "https://anikototv.to"
	UserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	DefaultTrackers = []string{
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://open.stealth.si:80/announce",
		"udp://tracker.torrent.eu.org:451/announce",
	}

	QualityOptions   = []string{"All", "720p", "1080p", "4K"}
	AnimeTypeOptions = []string{"All", "TV", "Movie", "OVA", "Special"}
)

type AppConfig struct {
	ProwlarrAPIKey string `json:"prowlarrApiKey"`
}

type prowlarrXMLConfig struct {
	XMLName xml.Name `xml:"Config"`
	ApiKey  string   `xml:"ApiKey"`
}

func getGoovieConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "goovie_config.json" // fallback to current dir
	}
	dir := filepath.Join(home, ".goovie")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "config.json")
}

func LoadConfig() bool {
	path := getGoovieConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false
	}
	if cfg.ProwlarrAPIKey != "" {
		ProwlarrAPIKey = cfg.ProwlarrAPIKey
		return true
	}
	return false
}

func SaveConfig(key string) error {
	ProwlarrAPIKey = key
	path := getGoovieConfigPath()
	cfg := AppConfig{ProwlarrAPIKey: key}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
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
