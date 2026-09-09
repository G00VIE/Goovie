package sysutil

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"bubble-stream/internal/config"
	"github.com/go-rod/rod/lib/launcher"
)

// FindMPVExecutable locates the mpv player binary across common system install locations
func FindMPVExecutable() string {
	ensurePath := func(p string) string {
		if p != "" {
			dir := filepath.Dir(p)
			cur := os.Getenv("PATH")
			if !strings.Contains(cur, dir) {
				_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+cur)
			}
		}
		return p
	}

	if p, err := exec.LookPath("mpv"); err == nil {
		return ensurePath(p)
	}
	if p, err := exec.LookPath("mpv.exe"); err == nil {
		return ensurePath(p)
	}
	if p, err := exec.LookPath("mpv.com"); err == nil {
		return ensurePath(p)
	}

	if runtime.GOOS == "windows" {
		localApp := os.Getenv("LOCALAPPDATA")
		progFiles := os.Getenv("ProgramFiles")
		progFilesX86 := os.Getenv("ProgramFiles(x86)")
		userProf := os.Getenv("USERPROFILE")
		appData := os.Getenv("APPDATA")

		// 1. Check WinGet Packages directories (glob pattern)
		if localApp != "" {
			matches, _ := filepath.Glob(filepath.Join(localApp, "Microsoft", "WinGet", "Packages", "*mpv*", "mpv.exe"))
			if len(matches) > 0 {
				return ensurePath(matches[0])
			}
			matchesCom, _ := filepath.Glob(filepath.Join(localApp, "Microsoft", "WinGet", "Packages", "*mpv*", "mpv.com"))
			if len(matchesCom) > 0 {
				return ensurePath(matchesCom[0])
			}
		}

		candidates := []string{
			filepath.Join(localApp, "Microsoft", "WinGet", "Links", "mpv.exe"),
			filepath.Join(localApp, "Programs", "mpv", "mpv.exe"),
			filepath.Join(localApp, "Programs", "mpv", "mpv.com"),
			filepath.Join(localApp, "Programs", "MPV Player", "mpv.exe"),
			filepath.Join(progFiles, "MPV Player", "mpv.exe"),
			filepath.Join(progFiles, "MPV Player", "mpv.com"),
			filepath.Join(progFiles, "mpv", "mpv.exe"),
			filepath.Join(progFiles, "mpv", "mpv.com"),
			filepath.Join(progFilesX86, "mpv", "mpv.exe"),
			filepath.Join(progFilesX86, "mpv", "mpv.com"),
			filepath.Join(appData, "mpv", "mpv.exe"),
			filepath.Join(userProf, "scoop", "shims", "mpv.exe"),
			`C:\ProgramData\chocolatey\bin\mpv.exe`,
		}
		for _, c := range candidates {
			if c != "" {
				if _, err := os.Stat(c); err == nil {
					return ensurePath(c)
				}
			}
		}
	} else if runtime.GOOS == "darwin" {
		candidates := []string{
			"/opt/homebrew/bin/mpv",
			"/usr/local/bin/mpv",
			"/Applications/mpv.app/Contents/MacOS/mpv",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return ensurePath(c)
			}
		}
	} else { // Linux
		candidates := []string{
			"/usr/bin/mpv",
			"/usr/local/bin/mpv",
			"/var/lib/flatpak/exports/bin/io.mpv.Mpv",
			"/snap/bin/mpv",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return ensurePath(c)
			}
		}
	}

	return ""
}

func downloadAndExtractMPVWindows(onProgress func(string)) error {
	localApp := os.Getenv("LOCALAPPDATA")
	if localApp == "" {
		localApp = `C:\Users\Default\AppData\Local`
	}
	targetDir := filepath.Join(localApp, "Programs", "mpv")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	zipURL := "https://github.com/mpv-player/mpv/releases/download/v0.41.0/mpv-v0.41.0-x86_64-pc-windows-msvc.zip"
	if onProgress != nil {
		onProgress("Downloading portable MPV video player...")
	}

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Get(zipURL)
	if err != nil {
		return fmt.Errorf("failed to download MPV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status downloading MPV: %d", resp.StatusCode)
	}

	tempZip := filepath.Join(os.TempDir(), "mpv_portable.zip")
	out, err := os.Create(tempZip)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		_ = os.Remove(tempZip)
		return err
	}
	defer os.Remove(tempZip)

	if onProgress != nil {
		onProgress("Extracting MPV video player...")
	}
	r, err := zip.OpenReader(tempZip)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		destPath := filepath.Join(targetDir, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(destPath, 0755)
		} else {
			_ = os.MkdirAll(filepath.Dir(destPath), 0755)
			rc, err := f.Open()
			if err != nil {
				continue
			}
			destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err == nil {
				_, _ = io.Copy(destFile, rc)
				destFile.Close()
			}
			rc.Close()
		}
	}

	return nil
}

// FindProwlarrExecutable checks common installation paths on Windows, macOS, and Linux
func FindProwlarrExecutable() string {
	if p, err := exec.LookPath("Prowlarr.Console"); err == nil {
		return p
	}
	if p, err := exec.LookPath("prowlarr-console"); err == nil {
		return p
	}
	if p, err := exec.LookPath("Prowlarr"); err == nil {
		return p
	}
	if p, err := exec.LookPath("prowlarr"); err == nil {
		return p
	}

	localApp := os.Getenv("LOCALAPPDATA")
	progFiles := os.Getenv("ProgramFiles")
	progFilesX86 := os.Getenv("ProgramFiles(x86)")
	progData := os.Getenv("ProgramData")
	if progData == "" {
		progData = `C:\ProgramData`
	}

	candidates := []string{
		filepath.Join(progData, "Prowlarr", "bin", "Prowlarr.Console.exe"),
		filepath.Join(progData, "Prowlarr", "bin", "Prowlarr.exe"),
		filepath.Join(progFiles, "Prowlarr", "bin", "Prowlarr.Console.exe"),
		filepath.Join(progFiles, "Prowlarr", "bin", "Prowlarr.exe"),
		filepath.Join(progFiles, "Prowlarr", "Prowlarr.Console.exe"),
		filepath.Join(progFiles, "Prowlarr", "Prowlarr.exe"),
		filepath.Join(progFilesX86, "Prowlarr", "bin", "Prowlarr.Console.exe"),
		filepath.Join(progFilesX86, "Prowlarr", "bin", "Prowlarr.exe"),
		filepath.Join(localApp, "Programs", "Prowlarr", "Prowlarr.Console.exe"),
		filepath.Join(localApp, "Programs", "Prowlarr", "Prowlarr.exe"),
		filepath.Join(localApp, "Prowlarr", "bin", "Prowlarr.Console.exe"),
		filepath.Join(localApp, "Prowlarr", "bin", "Prowlarr.exe"),
		// macOS
		"/Applications/Prowlarr.app/Contents/MacOS/Prowlarr",
		"/opt/homebrew/bin/prowlarr",
		"/usr/local/bin/prowlarr",
		// Linux
		"/usr/bin/prowlarr",
		"/usr/local/bin/prowlarr",
		"/opt/Prowlarr/prowlarr",
		"/opt/Prowlarr/Prowlarr",
	}

	for _, c := range candidates {
		if c != "" {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}
	return ""
}

// ProwlarrConfigXML models the structure of Prowlarr's config.xml for validation and parsing
type ProwlarrConfigXML struct {
	XMLName                xml.Name `xml:"Config"`
	Port                   string   `xml:"Port"`
	BindAddress            string   `xml:"BindAddress"`
	ApiKey                 string   `xml:"ApiKey"`
	AuthenticationMethod   string   `xml:"AuthenticationMethod"`
	AuthenticationRequired string   `xml:"AuthenticationRequired"`
	LaunchBrowser          string   `xml:"LaunchBrowser"`
	Branch                 string   `xml:"Branch"`
	LogLevel               string   `xml:"LogLevel"`
}

// GetProwlarrConfigPaths returns candidate paths for Prowlarr config.xml across all supported platforms
func GetProwlarrConfigPaths() []string {
	var paths []string
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		progData := os.Getenv("ProgramData")
		if progData == "" {
			progData = `C:\ProgramData`
		}
		localApp := os.Getenv("LOCALAPPDATA")
		appData := os.Getenv("APPDATA")

		// C:\ProgramData\Prowlarr is the primary location used by Prowlarr on Windows
		paths = append(paths, filepath.Join(progData, "Prowlarr", "config.xml"))
		if localApp != "" {
			paths = append(paths, filepath.Join(localApp, "Prowlarr", "config.xml"))
		}
		if appData != "" {
			paths = append(paths, filepath.Join(appData, "Prowlarr", "config.xml"))
		}
		if home != "" {
			paths = append(paths,
				filepath.Join(home, "AppData", "Local", "Prowlarr", "config.xml"),
				filepath.Join(home, "AppData", "Roaming", "Prowlarr", "config.xml"),
			)
		}
	} else if runtime.GOOS == "darwin" {
		if home != "" {
			paths = append(paths,
				filepath.Join(home, ".config", "Prowlarr", "config.xml"),
				filepath.Join(home, "Library", "Application Support", "Prowlarr", "config.xml"),
			)
		}
	} else { // linux
		if home != "" {
			paths = append(paths, filepath.Join(home, ".config", "Prowlarr", "config.xml"))
		}
		paths = append(paths, "/var/lib/prowlarr/config.xml")
	}
	return paths
}

// ValidateProwlarrConfigFile checks whether the given config.xml is valid XML, non-empty,
// has no corruption (e.g. null bytes from crashes), and contains a non-empty ApiKey.
func ValidateProwlarrConfigFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "", false
	}
	// Detect null-byte padding from system crashes or uncommitted filesystem flushes
	if bytes.ContainsRune(trimmed, 0) {
		return "", false
	}
	var cfg ProwlarrConfigXML
	if err := xml.Unmarshal(trimmed, &cfg); err != nil {
		return "", false
	}
	apiKey := strings.TrimSpace(cfg.ApiKey)
	if apiKey == "" || cfg.XMLName.Local != "Config" {
		return "", false
	}
	return apiKey, true
}

// KillHungProwlarrProcesses terminates any orphaned or dialog-locked Prowlarr processes
// when Prowlarr is not responding on its ping endpoint.
func KillHungProwlarrProcesses() {
	// If Prowlarr is already responding healthy, do not kill it
	client := &http.Client{Timeout: 1 * time.Second}
	if resp, err := client.Get("http://localhost:9696/ping"); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return
		}
	}

	if runtime.GOOS == "windows" {
		cmd1 := exec.Command("taskkill", "/F", "/IM", "Prowlarr.exe", "/T")
		HideConsoleWindow(cmd1)
		cmd1.Stdout = io.Discard
		cmd1.Stderr = io.Discard
		_ = cmd1.Run()

		cmd2 := exec.Command("taskkill", "/F", "/IM", "Prowlarr.Console.exe", "/T")
		HideConsoleWindow(cmd2)
		cmd2.Stdout = io.Discard
		cmd2.Stderr = io.Discard
		_ = cmd2.Run()
	} else {
		cmd := exec.Command("pkill", "-9", "-f", "prowlarr")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		_ = cmd.Run()
	}
	time.Sleep(200 * time.Millisecond)
}

func disableBrowserInExistingConfig() {
	paths := GetProwlarrConfigPaths()

	reBrowser := regexp.MustCompile(`(?i)<LaunchBrowser>\s*(True|true|1)\s*</LaunchBrowser>`)
	reAuthReq := regexp.MustCompile(`(?i)<AuthenticationRequired>\s*(Enabled|enabled)\s*</AuthenticationRequired>`)
	reAuthMeth := regexp.MustCompile(`(?i)<AuthenticationMethod>\s*(Basic|Forms|External|basic|forms|external)\s*</AuthenticationMethod>`)

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := string(data)
		changed := false

		if reBrowser.MatchString(content) {
			content = reBrowser.ReplaceAllString(content, "<LaunchBrowser>False</LaunchBrowser>")
			changed = true
		}
		if reAuthReq.MatchString(content) {
			content = reAuthReq.ReplaceAllString(content, "<AuthenticationRequired>DisabledForLocalAddresses</AuthenticationRequired>")
			changed = true
		}
		if reAuthMeth.MatchString(content) {
			content = reAuthMeth.ReplaceAllString(content, "<AuthenticationMethod>None</AuthenticationMethod>")
			changed = true
		}

		if changed {
			_ = os.Chmod(p, 0666)
			_ = os.WriteFile(p, []byte(content), 0644)
		}
	}
}

// GenerateProwlarrConfigXML creates a clean, well-formed config.xml content
func GenerateProwlarrConfigXML(apiKey string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<Config>
  <BindAddress>*</BindAddress>
  <Port>9696</Port>
  <SslPort>6969</SslPort>
  <EnableSsl>False</EnableSsl>
  <LaunchBrowser>False</LaunchBrowser>
  <ApiKey>%s</ApiKey>
  <AuthenticationMethod>None</AuthenticationMethod>
  <AuthenticationRequired>DisabledForLocalAddresses</AuthenticationRequired>
  <Branch>master</Branch>
  <LogLevel>info</LogLevel>
  <UrlBase></UrlBase>
  <InstanceName>Prowlarr</InstanceName>
</Config>
`, apiKey)
}

// PreseedProwlarrConfig ensures config.xml exists with AuthenticationMethod=None
// and a valid ApiKey, completely skipping the first-run browser credential wizard.
// Crucially, it detects and wipes any corrupted config files so Prowlarr doesn't crash on startup.
func PreseedProwlarrConfig() (string, error) {
	configPaths := GetProwlarrConfigPaths()

	var existingApiKey string

	// 1. Check existing config files: delete any corrupted ones, extract API key from valid ones
	for _, p := range configPaths {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			if key, valid := ValidateProwlarrConfigFile(p); valid {
				if existingApiKey == "" {
					existingApiKey = key
				}
			} else {
				// Corrupted / 0-byte / invalid XML file: remove it to prevent ProwlarrStartupException
				_ = os.Chmod(p, 0666)
				_ = os.Remove(p)
			}
		}
	}

	apiKey := existingApiKey
	if apiKey == "" {
		// Generate a random 32-character hex API key
		keyBytes := make([]byte, 16)
		if _, err := rand.Read(keyBytes); err != nil {
			return "", fmt.Errorf("failed to generate random API key: %w", err)
		}
		apiKey = hex.EncodeToString(keyBytes)
	}

	xmlContent := GenerateProwlarrConfigXML(apiKey)

	// Choose directories to seed based on OS
	home, _ := os.UserHomeDir()
	var targetDirs []string
	if runtime.GOOS == "windows" {
		progData := os.Getenv("ProgramData")
		if progData == "" {
			progData = `C:\ProgramData`
		}
		targetDirs = append(targetDirs, filepath.Join(progData, "Prowlarr"))
		if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
			targetDirs = append(targetDirs, filepath.Join(localApp, "Prowlarr"))
		}
	} else if runtime.GOOS == "darwin" {
		if home != "" {
			targetDirs = append(targetDirs,
				filepath.Join(home, ".config", "Prowlarr"),
				filepath.Join(home, "Library", "Application Support", "Prowlarr"),
			)
		}
	} else { // linux
		if home != "" {
			targetDirs = append(targetDirs, filepath.Join(home, ".config", "Prowlarr"))
		}
		targetDirs = append(targetDirs, "/var/lib/prowlarr")
	}

	for _, dir := range targetDirs {
		_ = os.MkdirAll(dir, 0755)
		xmlPath := filepath.Join(dir, "config.xml")

		// If missing or invalid, write fresh config
		if _, valid := ValidateProwlarrConfigFile(xmlPath); !valid {
			_ = os.Chmod(xmlPath, 0666)
			_ = os.Remove(xmlPath)
			_ = os.WriteFile(xmlPath, []byte(xmlContent), 0644)
		}
	}

	// Update existing configs to ensure headless / disabled auth
	disableBrowserInExistingConfig()

	config.ProwlarrAPIKey = apiKey
	_ = config.SaveConfig(apiKey)

	return apiKey, nil
}

// StartProwlarr launches Prowlarr in the background without opening a browser and waits for port 9696.
// It auto-heals corrupted configs, kills hung/zombie instances, and retries if Prowlarr fails to start.
func StartProwlarr() error {
	// Check if already responsive
	resp, err := http.Get("http://localhost:9696/ping")
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
	}

	// Kill any hung/zombie Prowlarr process (including modal error dialogs)
	KillHungProwlarrProcesses()

	// Guarantee config.xml is pre-seeded, valid, and uncorrupted
	_, _ = PreseedProwlarrConfig()

	exe := FindProwlarrExecutable()
	if exe == "" {
		return fmt.Errorf("Prowlarr executable not found")
	}

	startAttempt := func() error {
		flag := "-nobrowser"
		if runtime.GOOS == "windows" {
			flag = "/nobrowser"
		}
		cmd := exec.Command(exe, flag)
		HideConsoleWindow(cmd)
		var errBuf bytes.Buffer
		cmd.Stdout = io.Discard
		cmd.Stderr = &errBuf
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start Prowlarr: %w", err)
		}

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		// Poll until localhost:9696 is ready (up to 35 seconds)
		for i := 0; i < 70; i++ {
			time.Sleep(500 * time.Millisecond)

			// Check if process crashed prematurely
			select {
			case exitErr := <-done:
				stderrMsg := strings.TrimSpace(errBuf.String())
				if stderrMsg != "" {
					return fmt.Errorf("Prowlarr process exited prematurely (%v): %s", exitErr, stderrMsg)
				}
				return fmt.Errorf("Prowlarr process exited prematurely (%v)", exitErr)
			default:
			}

			resp, err := http.Get("http://localhost:9696/ping")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}

		return fmt.Errorf("timed out waiting for Prowlarr to initialize")
	}

	err = startAttempt()
	if err != nil {
		// Auto-heal on startup failure:
		// Kill any stuck/crashed instance, purge all config files, re-seed, and retry once
		KillHungProwlarrProcesses()
		for _, p := range GetProwlarrConfigPaths() {
			_ = os.Chmod(p, 0666)
			_ = os.Remove(p)
		}
		_, _ = PreseedProwlarrConfig()

		if retryErr := startAttempt(); retryErr == nil {
			return nil
		}
		return err
	}

	return nil
}

// ConfigureTopIndexers adds recommended public indexers (YTS, The Pirate Bay, LimeTorrents, EZTV, TorrentGalaxy) via Prowlarr REST API,
// routing Cloudflare-protected indexers through FlareSolverr if it is running.
func ConfigureTopIndexers(apiKey string) error {
	client := &http.Client{Timeout: 15 * time.Second}

	prowlarrURL := config.ProwlarrURL
	if prowlarrURL == "" {
		prowlarrURL = "http://localhost:9696"
	}
	flareURL := config.FlareSolverrURL
	if flareURL == "" {
		flareURL = "http://localhost:8191"
	}

	indexerEndpoint := fmt.Sprintf("%s/api/v1/indexer", prowlarrURL)

	// 0. If FlareSolverr is running, ensure tag and indexerproxy are registered in Prowlarr
	proxyTagID := 0
	if IsFlareSolverrRunning() {
		if id, err := EnsureFlareSolverrInProwlarr(prowlarrURL, apiKey, flareURL); err == nil {
			proxyTagID = id
		}
	}

	// 1. Get existing indexers to avoid duplicates
	req, err := http.NewRequest("GET", indexerEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var existing []map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&existing)
	existingNames := make(map[string]bool)
	for _, idx := range existing {
		if name, ok := idx["name"].(string); ok {
			existingNames[strings.ToLower(name)] = true
		}
		if def, ok := idx["definitionName"].(string); ok {
			existingNames[strings.ToLower(def)] = true
		}
	}

	// 2. Fetch available indexer schemas, retrying if Prowlarr is still loading them on startup
	schemaURL := fmt.Sprintf("%s/api/v1/indexer/schema", prowlarrURL)
	var schemas []map[string]interface{}
	for attempt := 0; attempt < 30; attempt++ {
		sReq, err := http.NewRequest("GET", schemaURL, nil)
		if err == nil {
			sReq.Header.Set("X-Api-Key", apiKey)
			sResp, err := client.Do(sReq)
			if err == nil {
				if sResp.StatusCode == http.StatusOK {
					_ = json.NewDecoder(sResp.Body).Decode(&schemas)
					sResp.Body.Close()
					if len(schemas) > 0 {
						break
					}
				} else {
					sResp.Body.Close()
				}
			}
		}
		time.Sleep(1 * time.Second)
	}

	if len(schemas) == 0 {
		return fmt.Errorf("no indexer schemas available from Prowlarr")
	}

	// Target top public indexers: Cloudflare-protected ones route through FlareSolverr
	targets := []struct {
		name      string
		needProxy bool
	}{
		{name: "yts", needProxy: false},
		{name: "thepiratebay", needProxy: false},
		{name: "limetorrents", needProxy: true},
		{name: "eztv", needProxy: true},
		{name: "torrentgalaxy", needProxy: false},
	}

	for _, target := range targets {
		if existingNames[target.name] {
			continue
		}

		for _, schema := range schemas {
			defName, _ := schema["definitionName"].(string)
			name, _ := schema["name"].(string)

			if strings.EqualFold(defName, target.name) || strings.EqualFold(name, target.name) {
				// Must set AppProfileId=1 (default profile) and remove any null/0 id
				schema["appProfileId"] = 1
				schema["enable"] = true
				delete(schema, "id")

				if target.needProxy && proxyTagID > 0 {
					schema["tags"] = []int{proxyTagID}
				}

				bodyBytes, err := json.Marshal(schema)
				if err != nil {
					continue
				}

				// Attempt 1: Add with enable=true
				addReq, err := http.NewRequest("POST", indexerEndpoint, bytes.NewBuffer(bodyBytes))
				if err != nil {
					continue
				}
				addReq.Header.Set("Content-Type", "application/json")
				addReq.Header.Set("X-Api-Key", apiKey)

				addResp, err := client.Do(addReq)
				if err == nil {
					defer addResp.Body.Close()
					if addResp.StatusCode == http.StatusOK || addResp.StatusCode == http.StatusCreated {
						existingNames[target.name] = true
						break
					}
				}

				// Attempt 2: If live connection test failed (e.g. Cloudflare / ISP block),
				// add with enable=false so it is successfully saved and listed in Prowlarr library
				schema["enable"] = false
				fbBytes, err := json.Marshal(schema)
				if err != nil {
					continue
				}

				fbReq, err := http.NewRequest("POST", indexerEndpoint, bytes.NewBuffer(fbBytes))
				if err != nil {
					continue
				}
				fbReq.Header.Set("Content-Type", "application/json")
				fbReq.Header.Set("X-Api-Key", apiKey)

				fbResp, err := client.Do(fbReq)
				if err == nil {
					defer fbResp.Body.Close()
					if fbResp.StatusCode == http.StatusOK || fbResp.StatusCode == http.StatusCreated {
						existingNames[target.name] = true
					}
				}
				break
			}
		}
	}

	return nil
}

type rodProgressLogger struct {
	onProgress func(string)
}

func (l *rodProgressLogger) Println(vs ...interface{}) {
	msg := fmt.Sprint(vs...)
	if strings.Contains(msg, "Progress:") {
		parts := strings.Split(msg, "Progress:")
		if len(parts) > 1 {
			pct := strings.TrimSpace(parts[1])
			if l.onProgress != nil {
				l.onProgress(fmt.Sprintf("Downloading Chromium Browser: %s", pct))
			}
			return
		}
	}
	if l.onProgress != nil && strings.TrimSpace(msg) != "" {
		l.onProgress(fmt.Sprintf("Chromium Setup: %s", strings.TrimSpace(msg)))
	}
}

// AutoInstallDependencies checks for and installs MPV, Prowlarr, and FlareSolverr, pre-seeding config and adding top indexers
func AutoInstallDependencies(onProgress func(step string)) error {
	// 0. Terminate any hung or modal-dialog locked Prowlarr processes from previous runs
	KillHungProwlarrProcesses()

	// 1. Install MPV if missing
	if FindMPVExecutable() == "" {
		switch runtime.GOOS {
		case "windows":
			// Try portable MSVC winget package first (silent, zero admin/UAC prompt required)
			cmd := exec.Command("winget", "install", "--id", "mpv-player.mpv-CI.MSVC", "-e", "--accept-source-agreements", "--accept-package-agreements", "--silent")
			HideConsoleWindow(cmd)
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			_ = cmd.Run()

			// If winget didn't locate or install it, download official portable build directly
			if FindMPVExecutable() == "" {
				_ = downloadAndExtractMPVWindows(onProgress)
			}
		case "darwin":
			onProgress("Installing Video Player (MPV) via Homebrew...")
			cmd := exec.Command("brew", "install", "mpv")
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			_ = cmd.Run()
		default: // linux
			onProgress("Installing Video Player (MPV) via package manager...")
			var cmd *exec.Cmd
			if _, err := exec.LookPath("dnf"); err == nil {
				cmd = exec.Command("sudo", "dnf", "install", "-y", "mpv")
			} else if _, err := exec.LookPath("pacman"); err == nil {
				cmd = exec.Command("sudo", "pacman", "-S", "--noconfirm", "mpv")
			} else {
				cmd = exec.Command("sudo", "apt-get", "install", "-y", "mpv")
			}
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			_ = cmd.Run()
		}
	}

	// 2. Pre-seed Prowlarr config and repair any corrupt config.xml to bypass credential setup
	onProgress("Verifying and pre-seeding Prowlarr security config...")
	apiKey, _ := PreseedProwlarrConfig()

	// 3. Install Prowlarr if not installed
	if FindProwlarrExecutable() == "" {
		switch runtime.GOOS {
		case "windows":
			onProgress("Installing Prowlarr via winget...")
			cmd := exec.Command("winget", "install", "--id", "TeamProwlarr.Prowlarr", "-e", "--accept-source-agreements", "--accept-package-agreements", "--silent")
			HideConsoleWindow(cmd)
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			_ = cmd.Run()
			// After winget install, guarantee config.xml exists and is clean in newly created directory
			time.Sleep(1 * time.Second)
			if key, err := PreseedProwlarrConfig(); err == nil && key != "" {
				apiKey = key
			}
		case "darwin":
			onProgress("Installing Prowlarr via Homebrew Cask...")
			cmd := exec.Command("brew", "install", "--cask", "prowlarr")
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			_ = cmd.Run()
			time.Sleep(1 * time.Second)
			if key, err := PreseedProwlarrConfig(); err == nil && key != "" {
				apiKey = key
			}
		default: // linux
			onProgress("On Linux, install Prowlarr via package manager or Docker...")
		}
	}

	// 4. Start Prowlarr process and wait for ready state
	onProgress("Starting Prowlarr engine on localhost:9696...")
	if err := StartProwlarr(); err != nil {
		return fmt.Errorf("failed to start Prowlarr: %w", err)
	}

	// Give Prowlarr a few seconds to load database and schema definitions
	time.Sleep(3 * time.Second)

	// Re-check detected API key if needed
	if apiKey == "" {
		for attempt := 0; attempt < 10; attempt++ {
			if config.AutoDetectAPIKey() && config.ProwlarrAPIKey != "" {
				apiKey = config.ProwlarrAPIKey
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	} else {
		_ = config.AutoDetectAPIKey()
		if config.ProwlarrAPIKey != "" {
			apiKey = config.ProwlarrAPIKey
		}
	}

	// 4b. Ensure FlareSolverr is downloaded, running, and ready to solve Cloudflare challenges
	if !IsFlareSolverrRunning() {
		onProgress("Ensuring FlareSolverr (Cloudflare bypass service)...")
		if err := StartFlareSolverr(onProgress); err != nil {
			// Non-fatal: if FlareSolverr fails to auto-download, log progress and continue
			onProgress("FlareSolverr auto-start skipped (direct indexer queries enabled)")
		}
	}

	// 4c. Ensure Browser Engine (Chromium for K-Drama scraper)
	if _, has := FindBrowserExecutable(); !has {
		onProgress("Installing Chromium browser engine for K-Drama...")
		b := launcher.NewBrowser()
		b.Logger = &rodProgressLogger{onProgress: onProgress}
		if err := b.Download(); err != nil {
			onProgress("Browser engine install skipped (K-Drama only)")
		}
	}

	// 5. Configure top indexers via API (attaching FlareSolverr proxy to Cloudflare-protected indexers)
	if apiKey != "" {
		onProgress("Configuring top indexers (YTS, The Pirate Bay, LimeTorrents, EZTV, TorrentGalaxy)...")
		if err := ConfigureTopIndexers(apiKey); err != nil {
			time.Sleep(2 * time.Second)
			_ = ConfigureTopIndexers(apiKey)
		}
		config.ProwlarrAPIKey = apiKey
		_ = config.SaveConfig(apiKey)
	}

	onProgress("System setup complete!")
	return nil
}
