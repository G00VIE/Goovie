package sysutil

import (
	"testing"

	"github.com/go-rod/rod/lib/launcher"
)

func TestCheckSystemHealth(t *testing.T) {
	path, has := launcher.LookPath()
	t.Logf("launcher.LookPath() has=%v, path=%s", has, path)

	mpv := FindMPVExecutable()
	t.Logf("FindMPVExecutable() = %s", mpv)

	health := CheckSystemHealth()
	t.Logf("Health: HasMPV=%v, HasBrowser=%v, HasProwlarr=%v, BrowserPath=%s", health.HasMPV, health.HasBrowser, health.HasProwlarr, health.BrowserPath)

	_ = health.AllReady()
	_ = health.CanWatchAnime()
	_ = health.CanWatchKDrama()
	_ = health.CanWatchWesternMedia()
}

func TestPreseedProwlarrConfig(t *testing.T) {
	key, err := PreseedProwlarrConfig()
	if err != nil {
		t.Fatalf("PreseedProwlarrConfig returned error: %v", err)
	}
	if key == "" {
		t.Error("expected non-empty API key from PreseedProwlarrConfig")
	}
}
