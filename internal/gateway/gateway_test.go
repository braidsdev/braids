package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/braidsdev/braids/internal/config"
	"github.com/braidsdev/braids/internal/connector"
)

func TestHandleRequestEndToEnd(t *testing.T) {
	// Mock upstream API
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "1", "email": "alice@test.com", "name": "Alice", "created": float64(1704067200)},
				{"id": "2", "email": "bob@test.com", "name": "Bob", "created": float64(1704153600)},
			},
			"has_more": false,
		})
	}))
	defer mock.Close()

	// Create a test connector definition pointing at mock
	connDef := &config.ConnectorDef{
		Name:    "test",
		Version: "1.0",
		BaseURL: mock.URL,
		Auth:    config.AuthDef{Type: "bearer", TokenField: "api_key"},
		Pagination: config.PaginationDef{
			Type:         "cursor",
			CursorParam:  "starting_after",
			CursorField:  "id",
			HasMoreField: "has_more",
			DataField:    "data",
		},
		Resources: map[string]config.ResourceDef{
			"customers": {Path: "/customers", Method: "GET"},
		},
	}

	cfg := &config.Config{
		Version: "1",
		Connectors: map[string]config.ConnectorRef{
			"test": {Type: "test", Config: map[string]string{"api_key": "sk_test"}},
		},
		Schemas: map[string]config.Schema{
			"customer": {
				MergeOn:            "email",
				ConflictResolution: "prefer_latest",
				Fields: map[string]config.Field{
					"id":         {Type: "string"},
					"email":      {Type: "string"},
					"name":       {Type: "string"},
					"created_at": {Type: "datetime"},
				},
			},
		},
		Endpoints: map[string]config.Endpoint{
			"/customers": {
				Schema: "customer",
				Sources: []config.Source{
					{
						Connector: "test",
						Resource:  "customers",
						Mapping: map[string]string{
							"id":         "'test_' + id",
							"email":      "email",
							"name":       "name",
							"created_at": "created",
						},
					},
				},
			},
		},
		Server: config.Server{Port: 0},
	}

	g := &Gateway{
		cfg: cfg,
		engines: map[string]*connector.ConnectorEngine{
			"test": connector.New(connDef, map[string]string{"api_key": "sk_test"}),
		},
	}

	req := httptest.NewRequest("GET", "/customers", nil)
	w := httptest.NewRecorder()
	g.handleRequest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatal("expected data array in response")
	}
	if len(data) != 2 {
		t.Errorf("expected 2 records, got %d", len(data))
	}

	meta, ok := resp["meta"].(map[string]any)
	if !ok {
		t.Fatal("expected meta object in response")
	}
	if total, ok := meta["total"].(float64); !ok || total != 2 {
		t.Errorf("expected total 2, got %v", meta["total"])
	}

	// Check first record mapping
	first := data[0].(map[string]any)
	if first["id"] != "test_1" {
		t.Errorf("expected id 'test_1', got %v", first["id"])
	}
	if first["email"] != "alice@test.com" {
		t.Errorf("expected email 'alice@test.com', got %v", first["email"])
	}
}

func TestHandleRequestNotFound(t *testing.T) {
	g := &Gateway{
		cfg: &config.Config{
			Endpoints: map[string]config.Endpoint{},
		},
		engines: map[string]*connector.ConnectorEngine{},
	}

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	g.handleRequest(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestNewGateway(t *testing.T) {
	content := fmt.Sprintf(`
version: "1"
connectors:
  stripe:
    type: stripe
    config:
      api_key: sk_test_123
schemas:
  customer:
    merge_on: email
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
  port: %d
`, 0)

	dir := t.TempDir()
	path := filepath.Join(dir, "braids.yaml")
	os.WriteFile(path, []byte(content), 0644)

	gw, err := New(path)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if gw.cfg.Version != "1" {
		t.Errorf("expected version '1', got %q", gw.cfg.Version)
	}
	if len(gw.engines) != 1 {
		t.Errorf("expected 1 engine, got %d", len(gw.engines))
	}
}
