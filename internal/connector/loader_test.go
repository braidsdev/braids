package connector

import (
	"testing"
)

func TestLoadDefBuiltinStripe(t *testing.T) {
	def, err := LoadDef("stripe", "/nonexistent")
	if err != nil {
		t.Fatalf("failed to load built-in stripe connector: %v", err)
	}
	if def.Name != "stripe" {
		t.Errorf("expected name 'stripe', got %q", def.Name)
	}
	if def.Auth.Type != "bearer" {
		t.Errorf("expected auth type 'bearer', got %q", def.Auth.Type)
	}
	if _, ok := def.Resources["customers"]; !ok {
		t.Error("expected 'customers' resource")
	}
}

func TestLoadDefBuiltinShopify(t *testing.T) {
	def, err := LoadDef("shopify", "/nonexistent")
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
	def, err := LoadDef("jsonplaceholder", "/nonexistent")
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
	def, err := LoadDef("dummyjson", "/nonexistent")
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
	_, err := LoadDef("nonexistent", "/tmp")
	if err == nil {
		t.Error("expected error for nonexistent connector")
	}
}
