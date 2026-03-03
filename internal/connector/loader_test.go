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

func TestLoadDefNotFound(t *testing.T) {
	_, err := LoadDef("nonexistent", "/tmp")
	if err == nil {
		t.Error("expected error for nonexistent connector")
	}
}
