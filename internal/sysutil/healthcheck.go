package sysutil

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"bubble-stream/internal/config"
	"github.com/go-rod/rod/lib/launcher"
)

// SystemHealth tracks availability of runtime dependencies
type SystemHealth struct {
	HasMPV      bool
	HasBrowser  bool
	HasProwlarr bool
	MPVPath     string
	BrowserPath string
	ProwlarrURL string
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

	// 2. Check Browser for Rod (Edge / Chrome / Chromium)
	if browserPath, has := launcher.LookPath(); has {
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

	// 4. Auto-heal: If Prowlarr is installed locally on the system but not running, automatically start it!
	if !health.HasProwlarr && FindProwlarrExecutable() != "" {
		if err := StartProwlarr(); err == nil {
			health.HasProwlarr = true
			if config.ProwlarrAPIKey == "" {
				_ = config.AutoDetectAPIKey()
			}
		}
	}

	return health
}
