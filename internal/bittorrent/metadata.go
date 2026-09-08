package bittorrent

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"bubble-stream/internal/sysutil"
)

// FileItem represents a file inside a multi-file or single-file torrent.
type FileItem struct {
	Index  int
	Path   string
	Length int64
}

// FormatLine formats the file item to match the format expected by Goovie's episode parser:
// e.g. "0  Friends.S04E01.mkv (245.3 MB)"
func (f FileItem) FormatLine() string {
	return fmt.Sprintf("%d  %s (%s)", f.Index, f.Path, FormatBytes(f.Length))
}

// FormatBytes formats byte count into human-readable size (e.g. 245.3 MB, 1.2 GB).
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// FetchFiles downloads torrent metadata in memory and returns all files within the torrent.
func (e *Engine) FetchFiles(magnet string, timeout time.Duration) ([]FileItem, error) {
	cl := e.Client()
	if cl == nil {
		return nil, fmt.Errorf("engine is closed")
	}

	t, err := cl.AddMagnet(magnet)
	if err != nil {
		return nil, fmt.Errorf("invalid magnet link: %w", err)
	}

	var trackerTiers [][]string
	for _, tr := range AllTrackers() {
		trackerTiers = append(trackerTiers, []string{tr})
	}
	t.AddTrackers(trackerTiers)

	defer func() {
		info := t.Info()
		t.Drop()
		if info != nil && e.DataDir() != "" {
			sysutil.RemoveTempDir(filepath.Join(e.DataDir(), info.Name))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		return nil, fmt.Errorf("timed out waiting for torrent metadata: %w", ctx.Err())
	}

	files := t.Files()
	items := make([]FileItem, 0, len(files))
	for i, f := range files {
		items = append(items, FileItem{
			Index:  i,
			Path:   f.DisplayPath(),
			Length: f.Length(),
		})
	}
	return items, nil
}
