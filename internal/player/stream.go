package player

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Stream resolves the given URL (which may be a magnet link, an HTTP URL that
// redirects to a magnet, or an HTTP URL that serves a .torrent file) and
// launches webtorrent with mpv for playback.
func Stream(targetPath string) error {
	if strings.HasPrefix(targetPath, "http") {
		resolved, err := resolveURL(targetPath)
		if err != nil {
			return err
		}
		targetPath = resolved
	}

	fmt.Printf("Buffering Torrent in MPV...\n")
	cmd := exec.Command("webtorrent", targetPath, "--mpv")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("playback error: %w", err)
	}
	return nil
}

// resolveURL follows HTTP redirects manually so it can intercept magnet:
// redirects that Go's HTTP client would otherwise reject.
func resolveURL(currentURL string) (string, error) {
	for {
		client := &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Block auto redirect follow to catch magnet handoffs
				return http.ErrUseLastResponse
			},
		}
		resp, err := client.Get(currentURL)
		if err != nil {
			return "", fmt.Errorf("failed to reach download endpoint: %w", err)
		}

		// Catch 3xx redirect locations before Go errors out on raw magnets
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := resp.Header.Get("Location")
			resp.Body.Close()
			if location == "" {
				return "", fmt.Errorf("redirect missing Location header")
			}
			if strings.HasPrefix(location, "magnet:") {
				return location, nil
			}
			currentURL = location
			continue
		}

		// If 200 OK, it's a binary .torrent file payload
		if resp.StatusCode == http.StatusOK {
			tmpFile, err := os.Create("/tmp/stream.torrent")
			if err != nil {
				resp.Body.Close()
				return "", fmt.Errorf("failed to build temporary file: %w", err)
			}
			_, err = io.Copy(tmpFile, resp.Body)
			resp.Body.Close()
			tmpFile.Close()
			if err != nil {
				return "", fmt.Errorf("failed to serialize binary payload: %w", err)
			}
			return "/tmp/stream.torrent", nil
		}

		resp.Body.Close()
		return "", fmt.Errorf("Prowlarr backend rejected request with status: %d", resp.StatusCode)
	}
}
