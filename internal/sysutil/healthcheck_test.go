package sysutil

import (
	"testing"
)

func TestCheckSystemHealth(t *testing.T) {
	health := CheckSystemHealth()
	// Should detect HasBrowser on Windows (Edge exists)
	if !health.HasBrowser {
		t.Log("Note: No browser detected on host")
	}
	// Verify helper methods don't panic
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
