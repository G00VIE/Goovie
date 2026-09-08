package sysutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestFlareSolverrDownloadURL(t *testing.T) {
	url := FlareSolverrDownloadURL()
	if !strings.HasPrefix(url, "https://github.com/FlareSolverr/FlareSolverr/releases/latest/download/flaresolverr_") {
		t.Errorf("unexpected download URL format: %s", url)
	}

	switch runtime.GOOS {
	case "windows":
		if !strings.HasSuffix(url, ".zip") {
			t.Errorf("windows download should be zip: %s", url)
		}
	default:
		if !strings.HasSuffix(url, ".tar.gz") {
			t.Errorf("unix download should be tar.gz: %s", url)
		}
	}
}

func TestEnsureFlareSolverrInProwlarr_Mock(t *testing.T) {
	createdTag := false
	createdProxy := false

	mux := http.NewServeMux()

	// Handle tags
	mux.HandleFunc("/api/v1/tag", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[]`))
			return
		}
		if r.Method == http.MethodPost {
			createdTag = true
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id": 42, "label": "flaresolverr"}`))
			return
		}
	})

	// Handle indexer proxy schema
	mux.HandleFunc("/api/v1/indexerproxy/schema", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		schemas := []map[string]interface{}{
			{
				"definitionName": "FlareSolverr",
				"implementation": "FlareSolverr",
				"configContract": "FlareSolverrSettings",
				"fields": []map[string]interface{}{
					{"name": "host", "value": ""},
					{"name": "requestTimeout", "value": 30},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(schemas)
	})

	// Handle indexer proxies
	mux.HandleFunc("/api/v1/indexerproxy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[]`))
			return
		}
		if r.Method == http.MethodPost {
			createdProxy = true
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id": 1, "name": "FlareSolverr"}`))
			return
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tagID, err := EnsureFlareSolverrInProwlarr(server.URL, "dummy-key", "http://localhost:8191")
	if err != nil {
		t.Fatalf("EnsureFlareSolverrInProwlarr returned error: %v", err)
	}

	if tagID != 42 {
		t.Errorf("expected tagID 42, got %d", tagID)
	}
	if !createdTag {
		t.Error("expected tag to be created")
	}
	if !createdProxy {
		t.Error("expected proxy to be created")
	}
}
