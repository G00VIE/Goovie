package bittorrent

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"bubble-stream/internal/sysutil"
	"github.com/anacrolix/torrent"
)

// StreamSession represents an active local streaming session.
type StreamSession struct {
	StreamURL     string
	Torrent       *torrent.Torrent
	File          *torrent.File
	server        *http.Server
	listener      net.Listener
	engineDataDir string
	closed        bool
	mu            sync.Mutex
}

// Close shuts down the local HTTP streaming server, closes ports, and drops the torrent from the engine,
// wiping its downloaded chunks immediately from disk.
func (s *StreamSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true

	if s.server != nil {
		s.server.SetKeepAlivesEnabled(false)
		_ = s.server.Close()
		s.server = nil
	}
	if s.listener != nil {
		_ = s.listener.Close()
		s.listener = nil
	}
	if s.Torrent != nil {
		info := s.Torrent.Info()
		s.Torrent.Drop()
		if info != nil && s.engineDataDir != "" {
			sysutil.RemoveTempDir(filepath.Join(s.engineDataDir, info.Name))
		}
		s.Torrent = nil
	}
}

// StartStreamServer creates a local HTTP server streaming the requested file from the torrent.
// If fileIndex < 0, it automatically selects the largest video file in the torrent.
func (e *Engine) StartStreamServer(ctx context.Context, magnet string, fileIndex int) (*StreamSession, error) {
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

	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		t.Drop()
		return nil, fmt.Errorf("timed out waiting for torrent metadata: %w", ctx.Err())
	}

	files := t.Files()
	if len(files) == 0 {
		t.Drop()
		return nil, fmt.Errorf("no files found in torrent")
	}

	var targetFile *torrent.File
	if fileIndex >= 0 && fileIndex < len(files) {
		targetFile = files[fileIndex]
	} else {
		// Pick largest file (typical for movie releases)
		var maxLen int64 = -1
		for _, f := range files {
			if f.Length() > maxLen {
				maxLen = f.Length()
				targetFile = f
			}
		}
	}

	if targetFile == nil {
		t.Drop()
		return nil, fmt.Errorf("target file not found")
	}

	// Tell the torrent client to prioritize pieces for this file
	targetFile.Download()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Drop()
		return nil, fmt.Errorf("failed to bind local stream port: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	streamURL := fmt.Sprintf("http://127.0.0.1:%d/stream", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		// Create an independent reader with aggressive readahead per HTTP connection
		reader := targetFile.NewReader()
		reader.SetReadahead(64 * 1024 * 1024) // 64MB aggressive readahead
		reader.SetResponsive()                 // Immediate streaming of chunks without waiting for full piece verification
		defer reader.Close()

		w.Header().Set("Content-Type", "video/octet-stream")
		w.Header().Set("Accept-Ranges", "bytes")
		http.ServeContent(w, r, targetFile.DisplayPath(), time.Time{}, reader)
	})

	server := &http.Server{Handler: mux}

	session := &StreamSession{
		StreamURL:     streamURL,
		Torrent:       t,
		File:          targetFile,
		server:        server,
		listener:      listener,
		engineDataDir: e.DataDir(),
	}

	go func() {
		_ = server.Serve(listener)
	}()

	return session, nil
}
