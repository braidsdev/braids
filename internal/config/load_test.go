package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	content := `
version: "1"
connectors:
  stripe:
    type: stripe
    config:
      api_key: sk_test_123
schemas:
  customer:
    merge_on: email
    conflict_resolution: prefer_latest
    fields:
      id:
        type: string
      email:
        type: string
endpoints:
  /customers:
    schema: customer
    sources:
      - connector: stripe
        resource: customers
        mapping:
          id: id
          email: email
server:
  port: 9090
  hot_reload: true
`
	dir := t.TempDir()
	path := filepath.Join(dir, "braids.yaml")
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Version != "1" {
		t.Errorf("expected version '1', got %q", cfg.Version)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if !cfg.Server.HotReload {
		t.Error("expected hot_reload true")
	}
	if len(cfg.Connectors) != 1 {
		t.Errorf("expected 1 connector, got %d", len(cfg.Connectors))
	}
	if cfg.Connectors["stripe"].Config["api_key"] != "sk_test_123" {
		t.Errorf("expected api_key sk_test_123, got %q", cfg.Connectors["stripe"].Config["api_key"])
	}
}

func TestLoadEnvSubstitution(t *testing.T) {
	t.Setenv("TEST_API_KEY", "sk_live_abc")

	content := `
version: "1"
connectors:
  stripe:
    type: stripe
    config:
      api_key: ${TEST_API_KEY}
schemas:
  customer:
    fields:
      id:
        type: string
endpoints:
  /customers:
    schema: customer
    sources:
      - connector: stripe
        resource: customers
        mapping:
          id: id
server:
  port: 8080
`
	dir := t.TempDir()
	path := filepath.Join(dir, "braids.yaml")
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Connectors["stripe"].Config["api_key"] != "sk_live_abc" {
		t.Errorf("expected substituted key, got %q", cfg.Connectors["stripe"].Config["api_key"])
	}
}
