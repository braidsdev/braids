package schema

import (
	"testing"

	"github.com/braidsdev/braids/internal/config"
	"github.com/braidsdev/braids/internal/connector"
)

func TestApply(t *testing.T) {
	records := []connector.Record{
		{"id": "123", "email": "test@example.com", "first_name": "Jane", "last_name": "Doe", "created_at": "2024-01-15"},
	}
	mapping := map[string]string{
		"id":         "'shopify_' + id",
		"email":      "email",
		"name":       "first_name + ' ' + last_name",
		"created_at": "created_at",
	}
	fields := map[string]config.Field{
		"id":         {Type: "string"},
		"email":      {Type: "string"},
		"name":       {Type: "string"},
		"created_at": {Type: "datetime"},
	}

	result, err := Apply(records, mapping, fields)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}

	rec := result[0]
	if rec["id"] != "shopify_123" {
		t.Errorf("expected shopify_123, got %v", rec["id"])
	}
	if rec["email"] != "test@example.com" {
		t.Errorf("expected test@example.com, got %v", rec["email"])
	}
	if rec["name"] != "Jane Doe" {
		t.Errorf("expected Jane Doe, got %v", rec["name"])
	}
	if rec["created_at"] != "2024-01-15T00:00:00Z" {
		t.Errorf("expected 2024-01-15T00:00:00Z, got %v", rec["created_at"])
	}
}
