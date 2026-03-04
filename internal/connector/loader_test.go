package connector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefBuiltinStripe(t *testing.T) {
	def, err := LoadDef("stripe", "/nonexistent", "")
	if err != nil {
		t.Fatalf("failed to load built-in stripe connector: %v", err)
	}
	if def.Name != "stripe" {
		t.Errorf("expected name 'stripe', got %q", def.Name)
	}
	if def.Auth.Type != "bearer" {
		t.Errorf("expected auth type 'bearer', got %q", def.Auth.Type)
	}
	if def.OpenAPISpec != "openapi.spec3.json" {
		t.Errorf("expected openapi_spec 'openapi.spec3.json', got %q", def.OpenAPISpec)
	}

	// Verify OpenAPI-derived resources exist
	expectedResources := []struct {
		key    string
		path   string
		method string
	}{
		{"get_customers", "/v1/customers", "GET"},
		{"post_customers", "/v1/customers", "POST"},
		{"get_customers_customer", "/v1/customers/{customer}", "GET"},
		{"delete_customers_customer", "/v1/customers/{customer}", "DELETE"},
		{"get_charges", "/v1/charges", "GET"},
		{"post_charges", "/v1/charges", "POST"},
		{"get_accounts", "/v1/accounts", "GET"},
		{"post_payment_intents", "/v1/payment_intents", "POST"},
		{"get_v2_core_accounts_id", "/v2/core/accounts/{id}", "GET"},
	}
	for _, exp := range expectedResources {
		res, ok := def.Resources[exp.key]
		if !ok {
			t.Errorf("expected resource %q to exist", exp.key)
			continue
		}
		if res.Path != exp.path {
			t.Errorf("resource %q: expected path %q, got %q", exp.key, exp.path, res.Path)
		}
		if res.Method != exp.method {
			t.Errorf("resource %q: expected method %q, got %q", exp.key, exp.method, res.Method)
		}
	}

	// Verify we got a substantial number of resources (spec has 616 operations)
	if len(def.Resources) < 500 {
		t.Errorf("expected at least 500 resources from OpenAPI spec, got %d", len(def.Resources))
	}
}

func TestLoadDefBuiltinShopify(t *testing.T) {
	def, err := LoadDef("shopify", "/nonexistent", "")
	if err != nil {
		t.Fatalf("failed to load built-in shopify connector: %v", err)
	}
	if def.Name != "shopify" {
		t.Errorf("expected name 'shopify', got %q", def.Name)
	}
	if def.Auth.Type != "header" {
		t.Errorf("expected auth type 'header', got %q", def.Auth.Type)
	}
	if def.Auth.HeaderName != "X-Shopify-Access-Token" {
		t.Errorf("expected header name 'X-Shopify-Access-Token', got %q", def.Auth.HeaderName)
	}
}

func TestLoadDefBuiltinJSONPlaceholder(t *testing.T) {
	def, err := LoadDef("jsonplaceholder", "/nonexistent", "")
	if err != nil {
		t.Fatalf("failed to load built-in jsonplaceholder connector: %v", err)
	}
	if def.Name != "jsonplaceholder" {
		t.Errorf("expected name 'jsonplaceholder', got %q", def.Name)
	}
	if def.Auth.Type != "none" {
		t.Errorf("expected auth type 'none', got %q", def.Auth.Type)
	}
	if _, ok := def.Resources["users"]; !ok {
		t.Error("expected 'users' resource")
	}
	if _, ok := def.Resources["posts"]; !ok {
		t.Error("expected 'posts' resource")
	}
	if _, ok := def.Resources["todos"]; !ok {
		t.Error("expected 'todos' resource")
	}
}

func TestLoadDefBuiltinDummyJSON(t *testing.T) {
	def, err := LoadDef("dummyjson", "/nonexistent", "")
	if err != nil {
		t.Fatalf("failed to load built-in dummyjson connector: %v", err)
	}
	if def.Name != "dummyjson" {
		t.Errorf("expected name 'dummyjson', got %q", def.Name)
	}
	if def.Auth.Type != "none" {
		t.Errorf("expected auth type 'none', got %q", def.Auth.Type)
	}
	if res, ok := def.Resources["users"]; !ok {
		t.Error("expected 'users' resource")
	} else if res.DataField != "users" {
		t.Errorf("expected users data_field 'users', got %q", res.DataField)
	}
	if res, ok := def.Resources["products"]; !ok {
		t.Error("expected 'products' resource")
	} else if res.DataField != "products" {
		t.Errorf("expected products data_field 'products', got %q", res.DataField)
	}
}

func TestLoadDefNotFound(t *testing.T) {
	_, err := LoadDef("nonexistent", "/tmp", "")
	if err == nil {
		t.Error("expected error for nonexistent connector")
	}
}

func TestLoadDefExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	connectorYAML := `name: custom
version: "1.0"
base_url: https://api.example.com
auth:
  type: bearer
  token_field: api_key
resources:
  users:
    path: /users
    method: GET
`
	if err := os.WriteFile(filepath.Join(tmpDir, "connector.yaml"), []byte(connectorYAML), 0644); err != nil {
		t.Fatal(err)
	}

	def, err := LoadDef("path", "/unused", tmpDir)
	if err != nil {
		t.Fatalf("failed to load connector from explicit path: %v", err)
	}
	if def.Name != "custom" {
		t.Errorf("expected name 'custom', got %q", def.Name)
	}
	if def.Auth.Type != "bearer" {
		t.Errorf("expected auth type 'bearer', got %q", def.Auth.Type)
	}
	if _, ok := def.Resources["users"]; !ok {
		t.Error("expected 'users' resource")
	}
}

func TestLoadDefExplicitPathRelative(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "my-connectors", "custom")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	connectorYAML := `name: relative-custom
version: "1.0"
base_url: https://api.example.com
auth:
  type: none
resources:
  items:
    path: /items
    method: GET
`
	if err := os.WriteFile(filepath.Join(subDir, "connector.yaml"), []byte(connectorYAML), 0644); err != nil {
		t.Fatal(err)
	}

	def, err := LoadDef("path", tmpDir, "my-connectors/custom")
	if err != nil {
		t.Fatalf("failed to load connector from relative explicit path: %v", err)
	}
	if def.Name != "relative-custom" {
		t.Errorf("expected name 'relative-custom', got %q", def.Name)
	}
}

func TestLoadDefExplicitPathNotFound(t *testing.T) {
	_, err := LoadDef("path", "/tmp", "/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent explicit path")
	}
}
