package config

import "os"

// Config holds the application configuration for connecting to Prowlarr.
type Config struct {
	ProwlarrURL    string
	ProwlarrAPIKey string
}

// Load returns a Config populated from environment variables,
// falling back to sensible defaults for local development.
func Load() Config {
	cfg := Config{
		ProwlarrURL:    "http://127.0.0.1:9696",
		ProwlarrAPIKey: "", // Left blank intentionally so it doesn't leak on GitHub
	}

	if v := os.Getenv("PROWLARR_URL"); v != "" {
		cfg.ProwlarrURL = v
	}
	if v := os.Getenv("PROWLARR_API_KEY"); v != "" {
		cfg.ProwlarrAPIKey = v
	}

	return cfg
}
