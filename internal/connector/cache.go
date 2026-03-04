package connector

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/braidsdev/braids/internal/config"
)

// cacheDir returns the path to ~/.braids/cache/openapi/, creating it if needed.
func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	dir := filepath.Join(home, ".braids", "cache", "openapi")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating cache directory: %w", err)
	}
	return dir, nil
}

// cacheKeyForURL returns a deterministic filesystem-safe filename for a URL.
func cacheKeyForURL(url string) string {
	h := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x.json", h)
}

// loadCachedSpec reads a cached OpenAPI spec for the given URL.
// Returns the data and true on hit, nil and false on miss.
func loadCachedSpec(specURL string) ([]byte, bool) {
	dir, err := cacheDir()
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(filepath.Join(dir, cacheKeyForURL(specURL)))
	if err != nil {
		return nil, false
	}
	return data, true
}

// downloadAndCacheSpec fetches an OpenAPI spec from the given URL and caches it.
// Cache write failures are non-fatal — the data is still returned for this session.
func downloadAndCacheSpec(specURL string) ([]byte, error) {
	resp, err := http.Get(specURL)
	if err != nil {
		return nil, fmt.Errorf("downloading OpenAPI spec from %s: %w", specURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading OpenAPI spec from %s: HTTP %d", specURL, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAPI spec response from %s: %w", specURL, err)
	}

	// Write to cache (non-fatal on failure)
	if dir, err := cacheDir(); err == nil {
		cachePath := filepath.Join(dir, cacheKeyForURL(specURL))
		if err := os.WriteFile(cachePath, data, 0644); err != nil {
			log.Printf("Warning: failed to cache OpenAPI spec: %v", err)
		} else {
			log.Printf("Cached OpenAPI spec (%d bytes) to %s", len(data), cachePath)
		}
	}

	return data, nil
}

// resolveOpenAPISpec resolves an OpenAPI spec for a connector definition.
// Priority: openapi_spec (local file) > openapi_url (cache hit > download).
// Returns nil, nil if neither is configured.
func resolveOpenAPISpec(def *config.ConnectorDef, dir string, embedded bool) ([]byte, error) {
	// Local file takes priority
	if def.OpenAPISpec != "" {
		log.Printf("Loading OpenAPI spec from local file: %s", def.OpenAPISpec)
		if embedded {
			return builtinConnectors.ReadFile(filepath.Join(dir, def.OpenAPISpec))
		}
		return os.ReadFile(filepath.Join(dir, def.OpenAPISpec))
	}

	// Try URL: cached first, then download
	if def.OpenAPIURL != "" {
		if data, ok := loadCachedSpec(def.OpenAPIURL); ok {
			log.Printf("Using cached OpenAPI spec for %s", def.OpenAPIURL)
			return data, nil
		}
		log.Printf("Downloading OpenAPI spec from %s ...", def.OpenAPIURL)
		return downloadAndCacheSpec(def.OpenAPIURL)
	}

	return nil, nil
}

// RefreshCachedSpec downloads and caches the OpenAPI spec for the given URL,
// replacing any existing cached version. Intended for the CLI update command.
func RefreshCachedSpec(name, specURL string) error {
	_, err := downloadAndCacheSpec(specURL)
	if err != nil {
		return fmt.Errorf("refreshing spec for connector %q: %w", name, err)
	}
	return nil
}
