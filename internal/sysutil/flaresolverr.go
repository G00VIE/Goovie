package sysutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"bubble-stream/internal/config"
)

var (
	flareCmd   *exec.Cmd
	flareCmdMu sync.Mutex
)

// IsFlareSolverrRunning checks whether FlareSolverr is actively responding on its configured URL.
func IsFlareSolverrRunning() bool {
	u := config.FlareSolverrURL
	if u == "" {
		u = "http://localhost:8191"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// FindFlareSolverrExecutable searches for flaresolverr on PATH and within the goovie data directory.
func FindFlareSolverrExecutable() string {
	names := []string{"flaresolverr"}
	if runtime.GOOS == "windows" {
		names = []string{"flaresolverr.exe"}
	}

	for _, name := range names {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}

	candidates := []string{
		filepath.Join(config.GoovieDir(), "flaresolverr", "flaresolverr"),
		filepath.Join(config.GoovieDir(), "flaresolverr", "flaresolverr.exe"),
		filepath.Join(config.GoovieDir(), "flaresolverr", "FlareSolverr", "flaresolverr"),
		filepath.Join(config.GoovieDir(), "flaresolverr", "FlareSolverr", "flaresolverr.exe"),
		filepath.Join(config.GoovieDir(), "bin", "flaresolverr"),
		filepath.Join(config.GoovieDir(), "bin", "flaresolverr.exe"),
	}

	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}

	return ""
}

// FlareSolverrDownloadURL constructs the GitHub release download URL for the current OS and architecture.
func FlareSolverrDownloadURL() string {
	osName := "linux"
	ext := ".tar.gz"
	switch runtime.GOOS {
	case "windows":
		osName = "windows"
		ext = ".zip"
	case "darwin":
		osName = "macos"
		ext = ".tar.gz"
	}

	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}

	return fmt.Sprintf("https://github.com/FlareSolverr/FlareSolverr/releases/latest/download/flaresolverr_%s_%s%s", osName, arch, ext)
}

type progressReader struct {
	reader     io.Reader
	total      int64
	current    int64
	lastPct    int
	onProgress func(string)
	prefix     string
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)
	if pr.total > 0 && pr.onProgress != nil {
		pct := int(float64(pr.current) / float64(pr.total) * 100)
		if pct > pr.lastPct && (pct-pr.lastPct >= 2 || pct == 100) {
			pr.lastPct = pct
			pr.onProgress(fmt.Sprintf("%s%d%%", pr.prefix, pct))
		}
	}
	return n, err
}

// DownloadAndExtractFlareSolverr downloads FlareSolverr from GitHub releases and extracts it.
func DownloadAndExtractFlareSolverr(destDir string, onProgress func(string)) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	dlURL := FlareSolverrDownloadURL()
	if onProgress != nil {
		onProgress("Downloading FlareSolverr binary from GitHub...")
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(dlURL)
	if err != nil {
		return fmt.Errorf("failed to download FlareSolverr from %s: %w", dlURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FlareSolverr download returned status %d from %s", resp.StatusCode, dlURL)
	}

	tmpFile, err := os.CreateTemp("", "flaresolverr_dl_*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	var reader io.Reader = resp.Body
	if resp.ContentLength > 0 && onProgress != nil {
		reader = &progressReader{
			reader:     resp.Body,
			total:      resp.ContentLength,
			onProgress: onProgress,
			prefix:     "Downloading FlareSolverr: ",
		}
	}

	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	if onProgress != nil {
		onProgress("Extracting FlareSolverr archive...")
	}

	if strings.HasSuffix(dlURL, ".zip") {
		return extractZipSafe(tmpName, destDir)
	}
	return extractTarGzSafe(tmpName, destDir)
}

func extractTarGzSafe(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, filepath.Clean(hdr.Name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(filepath.Separator)) && target != dest {
			continue // skip illegal path traversal
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, 0755)
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			mode := os.FileMode(hdr.Mode) & 0777
			if mode == 0 {
				mode = 0755
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			_, _ = io.Copy(out, tr)
			out.Close()
		}
	}
	return nil
}

func extractZipSafe(archivePath, dest string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		target := filepath.Join(dest, filepath.Clean(f.Name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(filepath.Separator)) && target != dest {
			continue
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0755)
			continue
		}

		_ = os.MkdirAll(filepath.Dir(target), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, _ = io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	return nil
}

// StartFlareSolverr checks if FlareSolverr is running, downloads it if missing, and starts it in the background.
func StartFlareSolverr(onProgress func(string)) error {
	if IsFlareSolverrRunning() {
		return nil
	}

	exe := FindFlareSolverrExecutable()
	if exe == "" {
		destDir := filepath.Join(config.GoovieDir(), "flaresolverr")
		if err := DownloadAndExtractFlareSolverr(destDir, onProgress); err != nil {
			return fmt.Errorf("failed to auto-download FlareSolverr: %w", err)
		}
		exe = FindFlareSolverrExecutable()
		if exe == "" {
			return fmt.Errorf("could not find flaresolverr executable after extraction")
		}
	}

	if onProgress != nil {
		onProgress("Starting FlareSolverr background service on port 8191...")
	}

	cmd := exec.Command(exe)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.Env = append(os.Environ(), "LOG_LEVEL=info", "PORT=8191")

	flareCmdMu.Lock()
	if err := cmd.Start(); err != nil {
		flareCmdMu.Unlock()
		return fmt.Errorf("failed to start FlareSolverr: %w", err)
	}
	flareCmd = cmd
	flareCmdMu.Unlock()

	// Poll until port 8191 is responsive (up to 30s)
	for i := 0; i < 60; i++ {
		time.Sleep(500 * time.Millisecond)
		if IsFlareSolverrRunning() {
			return nil
		}
	}

	return fmt.Errorf("timed out waiting for FlareSolverr to initialize on port 8191")
}

// StopFlareSolverr terminates the background FlareSolverr instance if one was launched by this process.
func StopFlareSolverr() {
	flareCmdMu.Lock()
	defer flareCmdMu.Unlock()
	if flareCmd != nil && flareCmd.Process != nil {
		_ = flareCmd.Process.Kill()
		_ = flareCmd.Wait()
		flareCmd = nil
	}
}

// EnsureFlareSolverrInProwlarr configures the 'flaresolverr' tag and FlareSolverr indexer proxy in Prowlarr.
func EnsureFlareSolverrInProwlarr(prowlarrURL, apiKey, flareURL string) (int, error) {
	if prowlarrURL == "" {
		prowlarrURL = "http://localhost:9696"
	}
	if flareURL == "" {
		flareURL = "http://localhost:8191"
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Ensure 'flaresolverr' tag exists
	tagID := 0
	tagsURL := fmt.Sprintf("%s/api/v1/tag?apikey=%s", prowlarrURL, apiKey)
	if resp, err := client.Get(tagsURL); err == nil {
		var tags []struct {
			ID    int    `json:"id"`
			Label string `json:"label"`
		}
		if json.NewDecoder(resp.Body).Decode(&tags) == nil {
			for _, t := range tags {
				if strings.EqualFold(t.Label, "flaresolverr") {
					tagID = t.ID
					break
				}
			}
		}
		resp.Body.Close()
	}

	if tagID == 0 {
		tagBody, _ := json.Marshal(map[string]string{"label": "flaresolverr"})
		req, err := http.NewRequest("POST", tagsURL, bytes.NewBuffer(tagBody))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			if resp, err := client.Do(req); err == nil {
				var created struct {
					ID int `json:"id"`
				}
				_ = json.NewDecoder(resp.Body).Decode(&created)
				tagID = created.ID
				resp.Body.Close()
			}
		}
	}

	// 2. Ensure FlareSolverr indexer proxy exists in Prowlarr
	proxiesURL := fmt.Sprintf("%s/api/v1/indexerproxy?apikey=%s", prowlarrURL, apiKey)
	hasProxy := false
	if resp, err := client.Get(proxiesURL); err == nil {
		var proxies []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if json.NewDecoder(resp.Body).Decode(&proxies) == nil {
			for _, p := range proxies {
				if strings.EqualFold(p.Name, "flaresolverr") {
					hasProxy = true
					break
				}
			}
		}
		resp.Body.Close()
	}

	if !hasProxy {
		// Fetch FlareSolverr proxy schema from Prowlarr
		schemaURL := fmt.Sprintf("%s/api/v1/indexerproxy/schema?apikey=%s", prowlarrURL, apiKey)
		if resp, err := client.Get(schemaURL); err == nil {
			var schemas []map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&schemas) == nil {
				for _, s := range schemas {
					def, _ := s["definitionName"].(string)
					if strings.EqualFold(def, "flaresolverr") {
						s["name"] = "FlareSolverr"
						if tagID > 0 {
							s["tags"] = []int{tagID}
						}

						// Configure host and requestTimeout fields
						if fields, ok := s["fields"].([]interface{}); ok {
							for _, f := range fields {
								if fm, ok := f.(map[string]interface{}); ok {
									if fm["name"] == "host" {
										fm["value"] = flareURL
									} else if fm["name"] == "requestTimeout" {
										fm["value"] = 60
									}
								}
							}
						}

						payloadBytes, _ := json.Marshal(s)
						req, err := http.NewRequest("POST", proxiesURL, bytes.NewBuffer(payloadBytes))
						if err == nil {
							req.Header.Set("Content-Type", "application/json")
							if pResp, err := client.Do(req); err == nil {
								pResp.Body.Close()
							}
						}
						break
					}
				}
			}
			resp.Body.Close()
		}
	}

	return tagID, nil
}
