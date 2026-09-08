package sysutil

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"bubble-stream/internal/config"
	"github.com/go-rod/rod/lib/launcher"
)

// FindBrowserExecutable checks system PATH, standard system directories,
// and rod's downloaded browser cache directory (~/.cache/rod/browser/ or %APPDATA%/rod/browser).
func FindBrowserExecutable() (string, bool) {
	// 1. Check system PATH and standard locations (Edge / Chrome / Chromium)
	if p, has := launcher.LookPath(); has {
		return p, true
	}

	// 2. Check Rod's downloaded browser cache
	b := launcher.NewBrowser()
	binPath := b.BinPath()
	if fi, err := os.Stat(binPath); err == nil && !fi.IsDir() {
		return binPath, true
	}

	altPath := filepath.Join(b.Dir(), "chrome-linux", "chrome")
	if fi, err := os.Stat(altPath); err == nil && !fi.IsDir() {
		return altPath, true
	}

	return "", false
}

// SystemHealth tracks availability of runtime dependencies
type SystemHealth struct {
	HasMPV          bool
	HasBrowser      bool
	HasProwlarr     bool
	HasFlareSolverr bool
	MPVPath         string
	BrowserPath     string
	ProwlarrURL     string
	FlareSolverrURL string
}

// AllReady returns true if all tools needed for every media type are present
func (h SystemHealth) AllReady() bool {
	return h.HasMPV && h.HasBrowser && h.HasProwlarr
}

// CanWatchAnime returns true if the components needed for Anime are ready
func (h SystemHealth) CanWatchAnime() bool {
	return h.HasMPV
}

// CanWatchKDrama returns true if the components needed for K-Drama are ready
func (h SystemHealth) CanWatchKDrama() bool {
	return h.HasMPV && h.HasBrowser
}

// CanWatchWesternMedia returns true if components needed for Western Movies/TV are ready
func (h SystemHealth) CanWatchWesternMedia() bool {
	return h.HasMPV && h.HasProwlarr
}

// CheckSystemHealth inspects the local machine for MPV, browser engine, and Prowlarr
func CheckSystemHealth() SystemHealth {
	var health SystemHealth

	// 1. Check MPV
	if mpvPath, err := exec.LookPath("mpv"); err == nil {
		health.HasMPV = true
		health.MPVPath = mpvPath
	}

	// 2. Check Browser for Rod (Edge / Chrome / Chromium / Rod-downloaded Chromium)
	if browserPath, has := FindBrowserExecutable(); has {
		health.HasBrowser = true
		health.BrowserPath = browserPath
	}

	// 3. Check Prowlarr
	prowlarrURL := config.ProwlarrURL
	if prowlarrURL == "" {
		prowlarrURL = "http://localhost:9696"
	}
	health.ProwlarrURL = prowlarrURL

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	// Try ping endpoint first (doesn't require api key)
	pingReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/ping", prowlarrURL), nil)
	if err == nil {
		resp, err := http.DefaultClient.Do(pingReq)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				health.HasProwlarr = true
			}
		}
	}

	// If ping didn't succeed, check if API key exists and system/status responds
	if !health.HasProwlarr {
		if config.ProwlarrAPIKey == "" {
			_ = config.AutoDetectAPIKey()
		}
		apiKey := config.ProwlarrAPIKey
		if apiKey != "" {
			statusReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/system/status", prowlarrURL), nil)
			if err == nil {
				statusReq.Header.Set("X-Api-Key", apiKey)
				resp, err := http.DefaultClient.Do(statusReq)
				if err == nil {
					resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						health.HasProwlarr = true
					}
				}
			}
		}
	}

	// 4. Check FlareSolverr (Cloudflare bypass proxy)
	health.FlareSolverrURL = config.FlareSolverrURL
	health.HasFlareSolverr = IsFlareSolverrRunning()

	return health
}
