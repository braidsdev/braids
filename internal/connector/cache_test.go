package connector

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/braidsdev/braids/internal/config"
)

func TestCacheKeyForURLDeterminism(t *testing.T) {
	url := "https://example.com/spec.json"
	key1 := cacheKeyForURL(url)
	key2 := cacheKeyForURL(url)
	if key1 != key2 {
		t.Errorf("expected deterministic key, got %q and %q", key1, key2)
	}

	// Different URLs produce different keys
	other := cacheKeyForURL("https://example.com/other.json")
	if key1 == other {
		t.Error("expected different keys for different URLs")
	}

	// Key ends with .json
	if filepath.Ext(key1) != ".json" {
		t.Errorf("expected .json extension, got %q", key1)
	}
}

func TestDownloadAndCacheSpec(t *testing.T) {
	// Override HOME for cache isolation
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	specJSON := `{"openapi":"3.0.0","paths":{}}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(specJSON))
	}))
	defer ts.Close()

	data, err := downloadAndCacheSpec(ts.URL + "/spec.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != specJSON {
		t.Errorf("expected %q, got %q", specJSON, string(data))
	}

	// Verify it was cached
	cached, ok := loadCachedSpec(ts.URL + "/spec.json")
	if !ok {
		t.Fatal("expected cache hit after download")
	}
	if string(cached) != specJSON {
		t.Errorf("cached data mismatch: expected %q, got %q", specJSON, string(cached))
	}
}

func TestLoadCachedSpecMiss(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	_, ok := loadCachedSpec("https://example.com/nonexistent.json")
	if ok {
		t.Error("expected cache miss for uncached URL")
	}
}

func TestDownloadAndCacheSpecHTTPError(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := downloadAndCacheSpec(ts.URL + "/missing.json")
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}

func TestResolveOpenAPISpecPrefersLocalFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create a local spec file
	tmpDir := t.TempDir()
	localSpec := `{"openapi":"3.0.0","paths":{"/local":{"get":{}}}}`
	if err := os.WriteFile(filepath.Join(tmpDir, "spec.json"), []byte(localSpec), 0644); err != nil {
		t.Fatal(err)
	}

	// Set up a server that would return a different spec
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"openapi":"3.0.0","paths":{"/remote":{"get":{}}}}`))
	}))
	defer ts.Close()

	def := &config.ConnectorDef{
		OpenAPISpec: "spec.json",
		OpenAPIURL:  ts.URL + "/spec.json",
	}

	data, err := resolveOpenAPISpec(def, tmpDir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != localSpec {
		t.Errorf("expected local spec to take priority, got %q", string(data))
	}
}

func TestResolveOpenAPISpecFallsBackToURL(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	remoteSpec := `{"openapi":"3.0.0","paths":{"/remote":{"get":{}}}}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(remoteSpec))
	}))
	defer ts.Close()

	def := &config.ConnectorDef{
		OpenAPIURL: ts.URL + "/spec.json",
	}

	data, err := resolveOpenAPISpec(def, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != remoteSpec {
		t.Errorf("expected remote spec, got %q", string(data))
	}
}

func TestResolveOpenAPISpecNeitherConfigured(t *testing.T) {
	def := &config.ConnectorDef{}

	data, err := resolveOpenAPISpec(def, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Error("expected nil data when neither spec nor URL configured")
	}
}
