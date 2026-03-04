package connector

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/braidsdev/braids/internal/config"
)

//go:embed connectors/*
var builtinConnectors embed.FS

// LoadDef resolves a connector type to its definition.
// It checks embedded built-in connectors first, then a local connectors/ directory.
// If explicitPath is non-empty, it loads directly from that directory (for type: path connectors).
// If the connector specifies an OpenAPI spec, it parses the spec and merges
// discovered resources with any explicitly declared resources.
func LoadDef(connectorType string, configDir string, explicitPath string) (*config.ConnectorDef, error) {
	// When an explicit path is provided (type: path), load directly from that directory
	if explicitPath != "" {
		dir := explicitPath
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(configDir, dir)
		}
		data, err := os.ReadFile(filepath.Join(dir, "connector.yaml"))
		if err != nil {
			return nil, fmt.Errorf("reading connector at path %q: %w", dir, err)
		}
		def, err := config.LoadConnectorDef(data)
		if err != nil {
			return nil, err
		}
		return mergeOpenAPIResources(def, dir, false)
	}

	connectorDir := filepath.Join("connectors", connectorType)

	// Try embedded built-in connectors
	data, err := builtinConnectors.ReadFile(filepath.Join(connectorDir, "connector.yaml"))
	if err == nil {
		def, err := config.LoadConnectorDef(data)
		if err != nil {
			return nil, err
		}
		return mergeOpenAPIResources(def, connectorDir, true)
	}

	// Try local connectors/ directory relative to braids.yaml
	localDir := filepath.Join(configDir, "connectors", connectorType)
	localPath := filepath.Join(localDir, "connector.yaml")
	data, err = os.ReadFile(localPath)
	if err == nil {
		def, err := config.LoadConnectorDef(data)
		if err != nil {
			return nil, err
		}
		return mergeOpenAPIResources(def, localDir, false)
	}

	return nil, fmt.Errorf("connector %q not found (checked built-in and %s)", connectorType, localPath)
}

// mergeOpenAPIResources reads the OpenAPI spec (if configured) and merges
// discovered resources into the connector definition. Explicit resources
// in connector.yaml take precedence over spec-derived ones.
func mergeOpenAPIResources(def *config.ConnectorDef, dir string, embedded bool) (*config.ConnectorDef, error) {
	if def.OpenAPISpec == "" {
		return def, nil
	}

	var specData []byte
	var err error
	if embedded {
		specData, err = builtinConnectors.ReadFile(filepath.Join(dir, def.OpenAPISpec))
	} else {
		specData, err = os.ReadFile(filepath.Join(dir, def.OpenAPISpec))
	}
	if err != nil {
		return nil, fmt.Errorf("reading OpenAPI spec %q: %w", def.OpenAPISpec, err)
	}

	// Derive the base path from the base_url (e.g. "https://api.stripe.com/v1" -> "/v1")
	basePath := extractBasePath(def.BaseURL)

	specResources, err := ParseOpenAPIResources(specData, basePath)
	if err != nil {
		return nil, err
	}

	// Merge: spec resources are the base, explicit YAML resources override
	explicit := def.Resources
	def.Resources = specResources
	for key, res := range explicit {
		def.Resources[key] = res
	}

	return def, nil
}

// extractBasePath extracts the path portion from a URL.
// e.g. "https://api.stripe.com/v1" -> "/v1"
func extractBasePath(baseURL string) string {
	// Find the third slash (after scheme://)
	idx := strings.Index(baseURL, "://")
	if idx < 0 {
		return ""
	}
	rest := baseURL[idx+3:]
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		return ""
	}
	return rest[slashIdx:]
}
