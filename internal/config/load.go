package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// Load reads a braids.yaml file, substitutes environment variables, and unmarshals it.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	expanded := substituteEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// LoadConnectorDef reads a connector.yaml file and unmarshals it.
func LoadConnectorDef(data []byte) (*ConnectorDef, error) {
	var def ConnectorDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parsing connector definition: %w", err)
	}
	return &def, nil
}

func substituteEnvVars(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match
	})
}
