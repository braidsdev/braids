package extension

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// ExtensionProtocols lists protocols that require out-of-process extensions.
var ExtensionProtocols = map[string]bool{
	"postgres": true,
	"mysql":    true,
	"mongodb":  true,
	"grpc":     true,
}

// Manager coordinates extension process lifecycles.
type Manager struct {
	mu        sync.Mutex
	processes map[string]*Process
}

// NewManager creates a new extension manager.
func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*Process),
	}
}

// IsExtensionProtocol returns true if the protocol requires an extension binary.
func IsExtensionProtocol(protocol string) bool {
	return ExtensionProtocols[protocol]
}

// GetOrStart returns a running extension process, starting one if needed.
// Downloads the binary if it's missing (first use during serve).
// Never auto-updates a running extension.
func (m *Manager) GetOrStart(protocol string) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if proc, ok := m.processes[protocol]; ok && proc.IsAlive() {
		return proc, nil
	}

	bp, err := binPath(protocol)
	if err != nil {
		return nil, err
	}

	// Download if missing
	if _, err := os.Stat(bp); os.IsNotExist(err) {
		log.Printf("Extension %q not installed, downloading...", protocol)
		if err := Download(protocol, false); err != nil {
			return nil, fmt.Errorf("auto-downloading extension %q: %w", protocol, err)
		}
	}

	proc, err := StartProcess(bp, protocol)
	if err != nil {
		return nil, err
	}
	m.processes[protocol] = proc
	return proc, nil
}

// Download downloads an extension binary. Called by CLI commands (install/update).
func (m *Manager) Download(protocol string, force bool) error {
	return Download(protocol, force)
}

// ListInstalled returns metadata for all installed extensions.
func (m *Manager) ListInstalled() ([]InstalledExtension, error) {
	manifest, err := loadManifest()
	if err != nil {
		return nil, err
	}
	var result []InstalledExtension
	for proto, entry := range manifest.Extensions {
		result = append(result, InstalledExtension{
			Protocol:      proto,
			ManifestEntry: entry,
		})
	}
	return result, nil
}

// CheckForUpdates checks GitHub releases for newer versions of installed extensions.
// Returns a map of protocol → latest version for extensions that have updates available.
// Non-blocking: errors for individual extensions are logged but not returned.
func (m *Manager) CheckForUpdates(protocols []string) map[string]string {
	manifest, err := loadManifest()
	if err != nil {
		return nil
	}

	updates := make(map[string]string)
	for _, proto := range protocols {
		entry, ok := manifest.Extensions[proto]
		if !ok {
			continue
		}
		latest, err := CheckLatestVersion(proto)
		if err != nil {
			continue // network error or no releases — skip silently
		}
		if latest != "" && latest != entry.Version {
			updates[proto] = latest
		}
	}
	return updates
}

// Shutdown stops all running extension processes.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for proto, proc := range m.processes {
		log.Printf("Stopping extension %q...", proto)
		if err := proc.Stop(); err != nil {
			log.Printf("Warning: error stopping extension %q: %v", proto, err)
		}
	}
	m.processes = make(map[string]*Process)
}
