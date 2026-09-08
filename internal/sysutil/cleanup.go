package sysutil

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	trackedTempMu sync.Mutex
	trackedTemps  []string
)

// RegisterTempDir tracks a temporary file or directory for purging on shutdown
func RegisterTempDir(path string) {
	if path == "" {
		return
	}
	trackedTempMu.Lock()
	defer trackedTempMu.Unlock()
	trackedTemps = append(trackedTemps, path)
}

// RemoveTempDir safely and permanently removes a file or directory with retries.
// Uses native os.RemoveAll which unlinks files at the filesystem level, completely bypassing the Recycle Bin.
func RemoveTempDir(path string) {
	if path == "" {
		return
	}
	for i := 0; i < 5; i++ {
		err := os.RemoveAll(path)
		if err == nil || os.IsNotExist(err) {
			return
		}
		// On Windows, read-only attributes can block deletion; clear them before retry
		_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err == nil {
				_ = os.Chmod(p, 0666)
			}
			return nil
		})
		time.Sleep(100 * time.Millisecond)
	}
}

// PurgeAllTempData permanently deletes all tracked files/dirs and scans os.TempDir() for ANY
// lingering goovie_* temporary files, directories, dumps, or chunks.
func PurgeAllTempData() {
	trackedTempMu.Lock()
	items := append([]string(nil), trackedTemps...)
	trackedTemps = nil
	trackedTempMu.Unlock()

	for _, item := range items {
		RemoveTempDir(item)
	}

	tempDir := os.TempDir()
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "goovie_") {
			fullPath := filepath.Join(tempDir, name)
			RemoveTempDir(fullPath)
		}
	}
}
