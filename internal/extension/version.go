package extension

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ManifestEntry records metadata for one installed extension.
type ManifestEntry struct {
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
	SHA256      string    `json:"sha256"`
}

// InstalledExtension pairs a protocol name with its manifest entry.
type InstalledExtension struct {
	Protocol string
	ManifestEntry
}

type manifest struct {
	Extensions map[string]ManifestEntry `json:"extensions"`
}

// extensionsDir returns ~/.braids/extensions/, creating it if needed.
func extensionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	dir := filepath.Join(home, ".braids", "extensions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating extensions directory: %w", err)
	}
	return dir, nil
}

func manifestPath() (string, error) {
	dir, err := extensionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "manifest.json"), nil
}

func loadManifest() (*manifest, error) {
	p, err := manifestPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &manifest{Extensions: make(map[string]ManifestEntry)}, nil
		}
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	if m.Extensions == nil {
		m.Extensions = make(map[string]ManifestEntry)
	}
	return &m, nil
}

func saveManifest(m *manifest) error {
	p, err := manifestPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	// Atomic write via temp + rename
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	return os.Rename(tmp, p)
}

func saveManifestEntry(protocol string, entry ManifestEntry) error {
	m, err := loadManifest()
	if err != nil {
		return err
	}
	m.Extensions[protocol] = entry
	return saveManifest(m)
}

// binPath returns the expected binary path for a protocol extension.
func binPath(protocol string) (string, error) {
	dir, err := extensionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "braids-ext-"+protocol), nil
}
