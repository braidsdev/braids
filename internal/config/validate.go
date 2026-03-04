package config

import (
	"fmt"
	"strings"
)

// Validate checks that a Config is well-formed.
func Validate(cfg *Config) error {
	var errs []string

	if cfg.Version == "" {
		errs = append(errs, "version is required")
	}

	if len(cfg.Connectors) == 0 {
		errs = append(errs, "at least one connector is required")
	}
	for name, conn := range cfg.Connectors {
		if conn.Type == "" {
			errs = append(errs, fmt.Sprintf("connector %q: type is required", name))
		}
		if conn.Type == "path" && conn.Path == "" {
			errs = append(errs, fmt.Sprintf("connector %q: path is required when type is \"path\"", name))
		}
	}

	if len(cfg.Schemas) == 0 {
		errs = append(errs, "at least one schema is required")
	}
	for name, s := range cfg.Schemas {
		if len(s.Fields) == 0 {
			errs = append(errs, fmt.Sprintf("schema %q: at least one field is required", name))
		}
		for fname, f := range s.Fields {
			if f.Type == "" {
				errs = append(errs, fmt.Sprintf("schema %q field %q: type is required", name, fname))
			}
		}
	}

	if len(cfg.Endpoints) == 0 {
		errs = append(errs, "at least one endpoint is required")
	}
	for path, ep := range cfg.Endpoints {
		if ep.Schema == "" {
			errs = append(errs, fmt.Sprintf("endpoint %q: schema is required", path))
		} else if _, ok := cfg.Schemas[ep.Schema]; !ok {
			errs = append(errs, fmt.Sprintf("endpoint %q: references unknown schema %q", path, ep.Schema))
		}
		if len(ep.Sources) == 0 {
			errs = append(errs, fmt.Sprintf("endpoint %q: at least one source is required", path))
		}
		for i, src := range ep.Sources {
			if src.Connector == "" {
				errs = append(errs, fmt.Sprintf("endpoint %q source %d: connector is required", path, i))
			} else if _, ok := cfg.Connectors[src.Connector]; !ok {
				errs = append(errs, fmt.Sprintf("endpoint %q source %d: references unknown connector %q", path, i, src.Connector))
			}
			if src.Resource == "" {
				errs = append(errs, fmt.Sprintf("endpoint %q source %d: resource is required", path, i))
			}
			if len(src.Mapping) == 0 {
				errs = append(errs, fmt.Sprintf("endpoint %q source %d: mapping is required", path, i))
			}
		}
	}

	if cfg.Server.Port < 0 || cfg.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("server.port must be between 0 and 65535, got %d", cfg.Server.Port))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}
