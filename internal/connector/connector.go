// Package connector implements a generic connector engine that reads
// connector definitions (YAML) and executes API calls with auth and pagination.
package connector

import (
	"net/http"

	"github.com/braidsdev/braids/internal/config"
)

// Record is a single upstream API record.
type Record map[string]any

// ConnectorEngine executes API calls for a given connector definition.
type ConnectorEngine struct {
	def    *config.ConnectorDef
	config map[string]string
	client *http.Client
}

// New creates a ConnectorEngine from a connector definition and user config.
func New(def *config.ConnectorDef, cfg map[string]string) *ConnectorEngine {
	return &ConnectorEngine{
		def:    def,
		config: cfg,
		client: &http.Client{},
	}
}
