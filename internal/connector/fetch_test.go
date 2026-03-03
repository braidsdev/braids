package connector

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/braidsdev/braids/internal/config"
)

func TestFetchBareArrayResponse(t *testing.T) {
	// Simulate an API that returns a bare JSON array (like JSONPlaceholder)
	users := []map[string]any{
		{"id": 1, "name": "Alice", "email": "alice@example.com"},
		{"id": 2, "name": "Bob", "email": "bob@example.com"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:    "test",
		BaseURL: server.URL,
		Auth:    config.AuthDef{Type: "none"},
		Pagination: config.PaginationDef{
			Type: "none",
		},
		Resources: map[string]config.ResourceDef{
			"users": {Path: "/users", Method: "GET"},
		},
	}
	engine := New(def, nil)

	records, err := engine.Fetch("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("expected first user name 'Alice', got %v", records[0]["name"])
	}
	if records[1]["email"] != "bob@example.com" {
		t.Errorf("expected second user email 'bob@example.com', got %v", records[1]["email"])
	}
}

func TestFetchObjectWithDataField(t *testing.T) {
	// Simulate an API that returns {"users": [...]} (like DummyJSON)
	response := map[string]any{
		"users": []any{
			map[string]any{"id": 1, "firstName": "Charlie"},
			map[string]any{"id": 2, "firstName": "Diana"},
		},
		"total": 2,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:    "test",
		BaseURL: server.URL,
		Auth:    config.AuthDef{Type: "none"},
		Pagination: config.PaginationDef{
			Type: "none",
		},
		Resources: map[string]config.ResourceDef{
			"users": {Path: "/users", Method: "GET", DataField: "users"},
		},
	}
	engine := New(def, nil)

	records, err := engine.Fetch("users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["firstName"] != "Charlie" {
		t.Errorf("expected first user firstName 'Charlie', got %v", records[0]["firstName"])
	}
}

func TestFetchResourceNotFound(t *testing.T) {
	def := &config.ConnectorDef{
		Name:      "test",
		Resources: map[string]config.ResourceDef{},
	}
	engine := New(def, nil)

	_, err := engine.Fetch("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent resource")
	}
}
