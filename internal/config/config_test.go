package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// --- getGoovieConfigPath ---

func TestGetGoovieConfigPath_ReturnsValidPath(t *testing.T) {
	path := getGoovieConfigPath()
	if path == "" {
		t.Fatal("expected non-empty config path")
	}

	// Should end with config.json
	if filepath.Base(path) != "config.json" {
		t.Errorf("expected basename config.json, got %s", filepath.Base(path))
	}
}

func TestGetGoovieConfigPath_CreatesDirectory(t *testing.T) {
	path := getGoovieConfigPath()
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected config dir to exist, got error: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", dir)
	}
}

// --- LoadConfig / SaveConfig round-trip ---

func TestSaveAndLoadConfig_RoundTrip(t *testing.T) {
	// Save a known API key
	testKey := "test-api-key-12345"
	err := SaveConfig(testKey)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify the global was updated
	if ProwlarrAPIKey != testKey {
		t.Errorf("expected ProwlarrAPIKey=%q, got %q", testKey, ProwlarrAPIKey)
	}

	// Reset global to test LoadConfig
	ProwlarrAPIKey = ""

	// Load config
	ok := LoadConfig()
	if !ok {
		t.Fatal("LoadConfig should return true after saving")
	}

	if ProwlarrAPIKey != testKey {
		t.Errorf("expected loaded key=%q, got %q", testKey, ProwlarrAPIKey)
	}

	// Clean up
	os.Remove(getGoovieConfigPath())
}

func TestSaveConfig_WritesValidJSON(t *testing.T) {
	testKey := "my-key"
	err := SaveConfig(testKey)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	data, err := os.ReadFile(getGoovieConfigPath())
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("config file is not valid JSON: %v", err)
	}

	if cfg.ProwlarrAPIKey != testKey {
		t.Errorf("expected key=%q in file, got %q", testKey, cfg.ProwlarrAPIKey)
	}

	// Clean up
	os.Remove(getGoovieConfigPath())
}

func TestLoadConfig_NoFile(t *testing.T) {
	// Use a non-existent path by manipulating state
	// We'll test the actual behavior: if the file doesn't exist, return false
	original := ProwlarrAPIKey
	ProwlarrAPIKey = "original"

	// The default config path was likely created. Remove it temporarily.
	path := getGoovieConfigPath()
	backup := path + ".bak"

	_, errR := os.ReadFile(path)
	if errR == nil {
		os.Rename(path, backup)
		defer os.Rename(backup, path)
	}

	ok := LoadConfig()
	if ok {
		t.Error("LoadConfig should return false when no config file exists")
	}

	// Key should remain unchanged since load failed
	if ProwlarrAPIKey != "original" {
		t.Errorf("ProwlarrAPIKey should not change on load failure, got %q", ProwlarrAPIKey)
	}
	_ = original
}

func TestLoadConfig_EmptyKeyInFile(t *testing.T) {
	path := getGoovieConfigPath()
	backup := path + ".bak"
	data, _ := os.ReadFile(path)
	if data != nil {
		os.Rename(path, backup)
		defer os.Rename(backup, path)
	}

	// Write a config with empty key
	os.WriteFile(path, []byte(`{"prowlarrApiKey": ""}`), 0644)

	ok := LoadConfig()
	if ok {
		t.Error("LoadConfig should return false when API key is empty")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	path := getGoovieConfigPath()
	backup := path + ".bak"
	data, _ := os.ReadFile(path)
	if data != nil {
		os.Rename(path, backup)
		defer os.Rename(backup, path)
	}

	os.WriteFile(path, []byte(`not json at all`), 0644)

	ok := LoadConfig()
	if ok {
		t.Error("LoadConfig should return false for invalid JSON")
	}
}

func TestLoadConfig_ValidJSONButMissingKey(t *testing.T) {
	path := getGoovieConfigPath()
	backup := path + ".bak"
	data, _ := os.ReadFile(path)
	if data != nil {
		os.Rename(path, backup)
		defer os.Rename(backup, path)
	}

	os.WriteFile(path, []byte(`{"someOtherField": "value"}`), 0644)

	ok := LoadConfig()
	if ok {
		t.Error("LoadConfig should return false when prowlarrApiKey field is missing")
	}
}

// --- AutoDetectAPIKey ---

func TestAutoDetectAPIKey_WithFile(t *testing.T) {
	// Create a temp XML file to test parsing
	xmlContent := `<?xml version="1.0" encoding="utf-8"?>
<Config>
  <ApiKey>detected-key-abc123</ApiKey>
</Config>`

	// Write to platform-appropriate location
	home, _ := os.UserHomeDir()
	var testPath string
	switch runtime.GOOS {
	case "linux":
		testPath = filepath.Join(home, ".config", "Prowlarr", "config.xml")
	case "darwin":
		testPath = filepath.Join(home, ".config", "Prowlarr", "config.xml")
	default:
		t.Skip("AutoDetect not fully testable on this OS")
	}

	// Create parent dirs
	os.MkdirAll(filepath.Dir(testPath), 0755)

	// Backup existing
	backup := testPath + ".bak"
	data, _ := os.ReadFile(testPath)
	if data != nil {
		os.Rename(testPath, backup)
		defer os.Rename(backup, testPath)
	}

	os.WriteFile(testPath, []byte(xmlContent), 0644)

	ok := AutoDetectAPIKey()
	if !ok {
		t.Error("AutoDetectAPIKey should return true when valid XML exists")
	}
	if ProwlarrAPIKey != "detected-key-abc123" {
		t.Errorf("expected detected key, got %q", ProwlarrAPIKey)
	}
}

func TestAutoDetectAPIKey_WithEmptyApiKey(t *testing.T) {
	home, _ := os.UserHomeDir()
	var testPath string
	switch runtime.GOOS {
	case "linux":
		testPath = filepath.Join(home, ".config", "Prowlarr", "config.xml")
	case "darwin":
		testPath = filepath.Join(home, ".config", "Prowlarr", "config.xml")
	default:
		t.Skip("Skipping on non-linux/darwin")
	}

	os.MkdirAll(filepath.Dir(testPath), 0755)

	backup := testPath + ".bak"
	data, _ := os.ReadFile(testPath)
	if data != nil {
		os.Rename(testPath, backup)
		defer os.Rename(backup, testPath)
	}

	os.WriteFile(testPath, []byte(`<Config><ApiKey></ApiKey></Config>`), 0644)

	ok := AutoDetectAPIKey()
	if ok {
		t.Error("AutoDetectAPIKey should return false when ApiKey is empty")
	}
}

func TestAutoDetectAPIKey_NoFile(t *testing.T) {
	// This won't find any file unless Prowlarr is installed
	// Reset so we can verify behavior
	ProwlarrAPIKey = ""

	ok := AutoDetectAPIKey()
	// This should return false in CI (no Prowlarr installed)
	if ok {
		t.Log("AutoDetectAPIKey found a key - Prowlarr may be installed on this machine")
	}
}

// --- InitConfig ---

func TestInitConfig_WithEnvVars(t *testing.T) {
	t.Setenv("PROWLARR_URL", "http://custom:1234")
	t.Setenv("PROWLARR_API_KEY", "env-key")

	// Reset globals
	ProwlarrURL = "http://localhost:9696"
	ProwlarrAPIKey = ""

	ok := InitConfig()
	if !ok {
		t.Error("InitConfig should return true when env vars are set")
	}
	if ProwlarrURL != "http://custom:1234" {
		t.Errorf("expected ProwlarrURL=http://custom:1234, got %s", ProwlarrURL)
	}
	if ProwlarrAPIKey != "env-key" {
		t.Errorf("expected ProwlarrAPIKey=env-key, got %s", ProwlarrAPIKey)
	}

	// Clean up env for other tests
	os.Unsetenv("PROWLARR_URL")
	os.Unsetenv("PROWLARR_API_KEY")
	// Reset to defaults
	ProwlarrURL = "http://localhost:9696"
	ProwlarrAPIKey = ""
}

// --- Default variable tests ---

func TestDefaultVariables(t *testing.T) {
	if ProwlarrURL != "http://localhost:9696" {
		t.Errorf("default ProwlarrURL should be http://localhost:9696, got %s", ProwlarrURL)
	}
	if MinimumSeeders != 5 {
		t.Errorf("default MinimumSeeders should be 5, got %d", MinimumSeeders)
	}
	if UserAgent == "" {
		t.Error("UserAgent should not be empty")
	}
	if len(DefaultTrackers) != 3 {
		t.Errorf("expected 3 default trackers, got %d", len(DefaultTrackers))
	}
	if len(QualityOptions) != 3 {
		t.Errorf("expected 3 quality options, got %d", len(QualityOptions))
	}
	if len(AnimeTypeOptions) != 5 {
		t.Errorf("expected 5 anime type options, got %d", len(AnimeTypeOptions))
	}
}

func TestDefaultTrackers_UdpScheme(t *testing.T) {
	for _, tr := range DefaultTrackers {
		if tr[:3] != "udp" {
			t.Errorf("tracker %q should use udp scheme", tr)
		}
	}
}
