package connector

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/braidsdev/braids/internal/config"
)

//go:embed connectors/*
var builtinConnectors embed.FS

// LoadDef resolves a connector type to its definition.
// It checks embedded built-in connectors first, then a local connectors/ directory.
func LoadDef(connectorType string, configDir string) (*config.ConnectorDef, error) {
	// Try embedded built-in connectors
	data, err := builtinConnectors.ReadFile(filepath.Join("connectors", connectorType, "connector.yaml"))
	if err == nil {
		return config.LoadConnectorDef(data)
	}

	// Try local connectors/ directory relative to braids.yaml
	localPath := filepath.Join(configDir, "connectors", connectorType, "connector.yaml")
	data, err = os.ReadFile(localPath)
	if err == nil {
		return config.LoadConnectorDef(data)
	}

	return nil, fmt.Errorf("connector %q not found (checked built-in and %s)", connectorType, localPath)
}
