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

func TestHandleRequestParameterizedRoute(t *testing.T) {
	// Mock upstream that echoes back the requested path
	var upstreamPath string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":           "acct_123",
			"display_name": "Test Account",
			"country":      "US",
			"email":        "test@example.com",
		})
	}))
	defer mock.Close()

	connDef := &config.ConnectorDef{
		Name:    "test",
		Version: "1.0",
		BaseURL: mock.URL,
		Auth:    config.AuthDef{Type: "bearer", TokenField: "api_key"},
		Resources: map[string]config.ResourceDef{
			"get_account": {Path: "/v2/core/accounts/{id}", Method: "GET"},
		},
	}

	cfg := &config.Config{
		Version: "1",
		Connectors: map[string]config.ConnectorRef{
			"test": {Type: "test", Config: map[string]string{"api_key": "sk_test"}},
		},
		Schemas: map[string]config.Schema{
			"account": {
				Fields: map[string]config.Field{
					"id":           {Type: "string"},
					"display_name": {Type: "string"},
					"country":      {Type: "string"},
					"email":        {Type: "string"},
				},
			},
		},
		Endpoints: map[string]config.Endpoint{
			"/accounts/{id}": {
				Schema: "account",
				Sources: []config.Source{
					{
						Connector: "test",
						Resource:  "get_account",
						Mapping: map[string]string{
							"id":           "id",
							"display_name": "display_name",
							"country":      "country",
							"email":        "email",
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

	req := httptest.NewRequest("GET", "/accounts/acct_123", nil)
	w := httptest.NewRecorder()
	g.handleRequest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify path param was substituted in upstream URL
	if upstreamPath != "/v2/core/accounts/acct_123" {
		t.Errorf("expected upstream path /v2/core/accounts/acct_123, got %s", upstreamPath)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatal("expected data array in response")
	}
	if len(data) != 1 {
		t.Errorf("expected 1 record, got %d", len(data))
	}

	first := data[0].(map[string]any)
	if first["id"] != "acct_123" {
		t.Errorf("expected id 'acct_123', got %v", first["id"])
	}
	if first["display_name"] != "Test Account" {
		t.Errorf("expected display_name 'Test Account', got %v", first["display_name"])
	}
}

func TestMatchEndpoint(t *testing.T) {
	endpoints := map[string]config.Endpoint{
		"/customers":     {Schema: "customer"},
		"/accounts/{id}": {Schema: "account"},
	}

	// Static match
	ep, params, ok := matchEndpoint("/customers", endpoints)
	if !ok {
		t.Fatal("expected /customers to match")
	}
	if ep.Schema != "customer" {
		t.Errorf("expected schema 'customer', got %q", ep.Schema)
	}
	if len(params) != 0 {
		t.Errorf("expected no params for static match, got %v", params)
	}

	// Parameterized match
	ep, params, ok = matchEndpoint("/accounts/acct_456", endpoints)
	if !ok {
		t.Fatal("expected /accounts/acct_456 to match")
	}
	if ep.Schema != "account" {
		t.Errorf("expected schema 'account', got %q", ep.Schema)
	}
	if params["id"] != "acct_456" {
		t.Errorf("expected id=acct_456, got %v", params)
	}

	// No match
	_, _, ok = matchEndpoint("/nonexistent", endpoints)
	if ok {
		t.Error("expected /nonexistent to not match")
	}

	// Wrong segment count
	_, _, ok = matchEndpoint("/accounts/acct_1/extra", endpoints)
	if ok {
		t.Error("expected /accounts/acct_1/extra to not match")
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
