package sysutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterAndPurgeTempDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "goovie_torrent_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	_ = os.WriteFile(testFile, []byte("hello"), 0644)

	RegisterTempDir(tempDir)
	PurgeAllTempData()

	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("expected tempDir to be purged, but it still exists")
	}
}

func TestPurgeAllTempData_CleansPrefixes(t *testing.T) {
	dir1, err1 := os.MkdirTemp("", "goovie_torrent_orphan_*")
	if err1 != nil {
		t.Fatalf("failed to create dir1: %v", err1)
	}
	dir2, err2 := os.MkdirTemp("", "goovie_meta_orphan_*")
	if err2 != nil {
		t.Fatalf("failed to create dir2: %v", err2)
	}

	PurgeAllTempData()

	if _, err := os.Stat(dir1); !os.IsNotExist(err) {
		t.Errorf("expected dir1 to be purged")
	}
	if _, err := os.Stat(dir2); !os.IsNotExist(err) {
		t.Errorf("expected dir2 to be purged")
	}
}
