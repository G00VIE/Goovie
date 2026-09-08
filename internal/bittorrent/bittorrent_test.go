package bittorrent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestEngineLifecycle(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	dataDir := engine.dataDir
	if dataDir == "" {
		t.Fatal("expected non-empty dataDir")
	}

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Fatalf("dataDir should exist while engine is active: %s", dataDir)
	}

	cl := engine.Client()
	if cl == nil {
		t.Fatal("expected client to be non-nil")
	}

	engine.Close()

	if engine.Client() != nil {
		t.Error("client should be nil after Close()")
	}

	// Verify temp directory was cleaned up
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Errorf("dataDir should be purged after Close(): %s", dataDir)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024 * 245, "245.0 MB"},
		{1024 * 1024 * 1024 * 3, "3.0 GB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.bytes)
		if got != tt.expected {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestFileItemFormatLine(t *testing.T) {
	item := FileItem{
		Index:  0,
		Path:   "Friends Season 4/Friends S04E01 The One with the Jellyfish.mkv",
		Length: 256901120,
	}

	line := item.FormatLine()
	if !strings.HasPrefix(line, "0  Friends Season 4") {
		t.Errorf("FormatLine() expected prefix '0  Friends Season 4', got: %s", line)
	}
	if !strings.Contains(line, "245.0 MB") {
		t.Errorf("FormatLine() expected size '245.0 MB', got: %s", line)
	}
}

func TestEngineClosedErrors(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	engine.Close()

	_, err = engine.FetchFiles("magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567", 1*time.Second)
	if err == nil {
		t.Error("expected error when FetchFiles called on closed engine")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = engine.StartStreamServer(ctx, "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567", 0)
	if err == nil {
		t.Error("expected error when StartStreamServer called on closed engine")
	}
}

func TestAllTrackers(t *testing.T) {
	trackers := AllTrackers()
	if len(trackers) < 5 {
		t.Errorf("expected at least 5 trackers, got %d", len(trackers))
	}
}
