package gateway

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/braidsdev/braids/internal/config"
	"github.com/braidsdev/braids/internal/connector"
)

// PrintBanner writes a formatted startup banner to stdout.
func PrintBanner(cfg *config.Config, engines map[string]*connector.ConnectorEngine, version string, configPath string) {
	printBanner(os.Stdout, cfg, engines, version, configPath)
}

func printBanner(w io.Writer, cfg *config.Config, engines map[string]*connector.ConnectorEngine, version string, configPath string) {
	port := cfg.Server.Port
	if port == 0 {
		port = 8080
	}

	fmt.Fprintf(w, "\n■ Braids Gateway %s\n", version)
	fmt.Fprintln(w, "────────────────────────────────────")
	fmt.Fprintln(w)

	// Config loaded
	fmt.Fprintf(w, "✓ Config loaded            %s\n", filepath.Base(configPath))

	// Connectors (sorted)
	connectorNames := make([]string, 0, len(engines))
	for name := range engines {
		connectorNames = append(connectorNames, name)
	}
	sort.Strings(connectorNames)
	for _, name := range connectorNames {
		fmt.Fprintf(w, "✓ Connector ready          %s\n", name)
	}

	// Schema summary
	endpointCount := len(cfg.Endpoints)
	sourceSet := make(map[string]struct{})
	for _, ep := range cfg.Endpoints {
		for _, src := range ep.Sources {
			sourceSet[src.Connector] = struct{}{}
		}
	}
	fmt.Fprintf(w, "✓ Schema validated         %d endpoint%s, %d source%s\n",
		endpointCount, plural(endpointCount),
		len(sourceSet), plural(len(sourceSet)))

	// Per-schema merge strategy
	schemaNames := make([]string, 0, len(cfg.Schemas))
	for name := range cfg.Schemas {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)
	for _, name := range schemaNames {
		s := cfg.Schemas[name]
		if s.ConflictResolution != "" && s.MergeOn != "" {
			fmt.Fprintf(w, "✓ Merge strategy           %s on %q\n", s.ConflictResolution, s.MergeOn)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "■ Gateway listening on     http://localhost:%d\n", port)
	fmt.Fprintln(w)

	// Endpoints
	fmt.Fprintln(w, "  Endpoints:")
	paths := make([]string, 0, len(cfg.Endpoints))
	for path := range cfg.Endpoints {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		ep := cfg.Endpoints[path]
		names := make([]string, 0, len(ep.Sources))
		for _, src := range ep.Sources {
			names = append(names, src.Connector)
		}
		fmt.Fprintf(w, "  GET  %-20s %s\n", path, strings.Join(names, " + "))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Press Ctrl+C to stop")
	fmt.Fprintln(w)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
