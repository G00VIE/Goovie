package sysutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateProwlarrConfigFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "prowlarr_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Non-existent file
	if _, ok := ValidateProwlarrConfigFile(filepath.Join(tempDir, "nonexistent.xml")); ok {
		t.Error("expected non-existent file to be invalid")
	}

	// 2. 0-byte empty file
	emptyPath := filepath.Join(tempDir, "empty.xml")
	if err := os.WriteFile(emptyPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ValidateProwlarrConfigFile(emptyPath); ok {
		t.Error("expected 0-byte file to be invalid/corrupt")
	}

	// 3. Whitespace only file
	spacePath := filepath.Join(tempDir, "space.xml")
	if err := os.WriteFile(spacePath, []byte("   \n\t  "), 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ValidateProwlarrConfigFile(spacePath); ok {
		t.Error("expected whitespace-only file to be invalid/corrupt")
	}

	// 4. File containing null bytes (common crash symptom)
	nullPath := filepath.Join(tempDir, "nulls.xml")
	if err := os.WriteFile(nullPath, []byte("<Config>\x00\x00\x00</Config>"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ValidateProwlarrConfigFile(nullPath); ok {
		t.Error("expected null-byte padded file to be invalid/corrupt")
	}

	// 5. Malformed XML syntax
	badXMLPath := filepath.Join(tempDir, "bad.xml")
	if err := os.WriteFile(badXMLPath, []byte("<Config><ApiKey>123</Config>"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ValidateProwlarrConfigFile(badXMLPath); ok {
		t.Error("expected malformed XML syntax to be invalid/corrupt")
	}

	// 6. Wrong root element
	wrongRootPath := filepath.Join(tempDir, "wrongroot.xml")
	if err := os.WriteFile(wrongRootPath, []byte("<Settings><ApiKey>123</ApiKey></Settings>"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ValidateProwlarrConfigFile(wrongRootPath); ok {
		t.Error("expected wrong root element to be invalid")
	}

	// 7. Missing ApiKey
	missingKeyPath := filepath.Join(tempDir, "missingkey.xml")
	if err := os.WriteFile(missingKeyPath, []byte("<Config><Port>9696</Port></Config>"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ValidateProwlarrConfigFile(missingKeyPath); ok {
		t.Error("expected missing ApiKey to be invalid")
	}

	// 8. Empty ApiKey
	emptyKeyPath := filepath.Join(tempDir, "emptykey.xml")
	if err := os.WriteFile(emptyKeyPath, []byte("<Config><ApiKey>   </ApiKey></Config>"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ValidateProwlarrConfigFile(emptyKeyPath); ok {
		t.Error("expected empty ApiKey to be invalid")
	}

	// 9. Valid Prowlarr config
	validPath := filepath.Join(tempDir, "valid.xml")
	validXML := GenerateProwlarrConfigXML("abcdef1234567890abcdef1234567890")
	if err := os.WriteFile(validPath, []byte(validXML), 0644); err != nil {
		t.Fatal(err)
	}
	key, ok := ValidateProwlarrConfigFile(validPath)
	if !ok {
		t.Error("expected valid config to pass validation")
	}
	if key != "abcdef1234567890abcdef1234567890" {
		t.Errorf("expected apiKey 'abcdef1234567890abcdef1234567890', got %q", key)
	}
}

func TestGenerateProwlarrConfigXML(t *testing.T) {
	key := "testkey123456"
	xmlStr := GenerateProwlarrConfigXML(key)

	if !strings.Contains(xmlStr, "<ApiKey>"+key+"</ApiKey>") {
		t.Errorf("expected ApiKey tag with %s", key)
	}
	if !strings.Contains(xmlStr, "<LaunchBrowser>False</LaunchBrowser>") {
		t.Error("expected LaunchBrowser=False")
	}
	if !strings.Contains(xmlStr, "<AuthenticationMethod>None</AuthenticationMethod>") {
		t.Error("expected AuthenticationMethod=None")
	}
	if !strings.Contains(xmlStr, "<AuthenticationRequired>DisabledForLocalAddresses</AuthenticationRequired>") {
		t.Error("expected AuthenticationRequired=DisabledForLocalAddresses")
	}
	if !strings.Contains(xmlStr, "<Port>9696</Port>") {
		t.Error("expected Port=9696")
	}
}
