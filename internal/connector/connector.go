// Package connector implements a generic connector engine that reads
// connector definitions (YAML) and executes API calls with auth and pagination.
package connector

import (
	"fmt"
	"net/http"

	"github.com/braidsdev/braids/internal/config"
	"github.com/braidsdev/braids/internal/extension"
)

// Record is a single upstream API record.
type Record map[string]any

// HTTPFetcher executes REST API calls for a given connector definition.
type HTTPFetcher struct {
	httpHelper
	client *http.Client
}

// NewHTTPFetcher creates an HTTPFetcher from a connector definition and user config.
func NewHTTPFetcher(def *config.ConnectorDef, cfg map[string]string) *HTTPFetcher {
	return &HTTPFetcher{
		httpHelper: httpHelper{def: def, config: cfg},
		client:     &http.Client{},
	}
}

// NewFetcher creates the appropriate Fetcher for a connector based on its protocol.
// The optional extMgr parameter provides the extension manager for protocols that
// require out-of-process extensions (postgres, mysql, mongodb, grpc).
func NewFetcher(def *config.ConnectorDef, ref config.ConnectorRef, extMgr ...*extension.Manager) (Fetcher, error) {
	mergeInstanceResources(def, ref)

	protocol := def.Protocol
	if protocol == "" {
		protocol = "http"
	}

	switch protocol {
	case "http":
		return NewHTTPFetcher(def, ref.Config), nil
	case "graphql":
		return NewGraphQLFetcher(def, ref.Config), nil
	case "postgres", "mysql", "mongodb", "grpc":
		if len(extMgr) == 0 || extMgr[0] == nil {
			return nil, fmt.Errorf("protocol %q requires the extension system", protocol)
		}
		return NewExtensionFetcher(def, ref.Config, extMgr[0]), nil
	default:
		return nil, fmt.Errorf("unsupported protocol %q", protocol)
	}
}

// mergeInstanceResources merges instance-level resource overrides from ConnectorRef
// into the ConnectorDef's resources. Instance resources override connector-def resources.
func mergeInstanceResources(def *config.ConnectorDef, ref config.ConnectorRef) {
	if len(ref.Resources) == 0 {
		return
	}
	if def.Resources == nil {
		def.Resources = make(map[string]config.ResourceDef)
	}
	for name, override := range ref.Resources {
		res := def.Resources[name] // start from existing def if present
		if override.Path != "" {
			res.Path = override.Path
		}
		if override.Method != "" {
			res.Method = override.Method
		}
		if override.DataField != "" {
			res.DataField = override.DataField
		}
		if override.Query != "" {
			res.Query = override.Query
		}
		if override.Collection != "" {
			res.Collection = override.Collection
		}
		if override.Filter != "" {
			res.Filter = override.Filter
		}
		if override.Service != "" {
			res.Service = override.Service
		}
		def.Resources[name] = res
	}
}
