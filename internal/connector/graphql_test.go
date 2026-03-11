package connector

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/braidsdev/braids/internal/config"
)

func TestGraphQLFetchBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)

		if _, ok := req["query"]; !ok {
			t.Error("expected query field in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"users": []any{
					map[string]any{"id": "1", "name": "Alice"},
					map[string]any{"id": "2", "name": "Bob"},
				},
			},
		})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:     "test-graphql",
		Protocol: "graphql",
		BaseURL:  server.URL,
		Auth:     config.AuthDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"users": {
				Query:     `query { users { id name } }`,
				DataField: "users",
			},
		},
	}

	fetcher := NewGraphQLFetcher(def, nil)
	records, err := fetcher.Fetch("users", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("expected first user 'Alice', got %v", records[0]["name"])
	}
}

func TestGraphQLFetchWithVariables(t *testing.T) {
	var receivedVars map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)

		if vars, ok := req["variables"].(map[string]any); ok {
			receivedVars = vars
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"user": map[string]any{"id": "1", "name": "Alice"},
			},
		})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:     "test-graphql",
		Protocol: "graphql",
		BaseURL:  server.URL,
		Auth:     config.AuthDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"user": {
				Query:     `query($id: ID!) { user(id: $id) { id name } }`,
				DataField: "user",
			},
		},
	}

	fetcher := NewGraphQLFetcher(def, nil)
	params := map[string]any{"id": "1"}
	records, err := fetcher.Fetch("user", params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("expected name 'Alice', got %v", records[0]["name"])
	}
	if receivedVars["id"] != "1" {
		t.Errorf("expected variable id='1', got %v", receivedVars["id"])
	}
}

func TestGraphQLFetchNestedDataField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"products": map[string]any{
					"edges": []any{
						map[string]any{"node": map[string]any{"id": "p1", "title": "Widget"}},
						map[string]any{"node": map[string]any{"id": "p2", "title": "Gadget"}},
					},
				},
			},
		})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:     "test-graphql",
		Protocol: "graphql",
		BaseURL:  server.URL,
		Auth:     config.AuthDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"products": {
				Query:     `query { products { edges { node { id title } } } }`,
				DataField: "products.edges",
			},
		},
	}

	fetcher := NewGraphQLFetcher(def, nil)
	records, err := fetcher.Fetch("products", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

func TestGraphQLFetchWithAuth(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"viewer": map[string]any{"login": "testuser"},
			},
		})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:     "test-graphql",
		Protocol: "graphql",
		BaseURL:  server.URL,
		Auth:     config.AuthDef{Type: "bearer", TokenField: "token"},
		Resources: map[string]config.ResourceDef{
			"viewer": {
				Query:     `query { viewer { login } }`,
				DataField: "viewer",
			},
		},
	}

	fetcher := NewGraphQLFetcher(def, map[string]string{"token": "ghp_test123"})
	records, err := fetcher.Fetch("viewer", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if receivedAuth != "Bearer ghp_test123" {
		t.Errorf("expected Bearer auth, got %q", receivedAuth)
	}
}

func TestGraphQLFetchErrorPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data":   nil,
			"errors": []map[string]any{{"message": "Field 'foo' not found"}},
		})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:     "test-graphql",
		Protocol: "graphql",
		BaseURL:  server.URL,
		Auth:     config.AuthDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"bad": {Query: `query { foo }`, DataField: "foo"},
		},
	}

	fetcher := NewGraphQLFetcher(def, nil)
	_, err := fetcher.Fetch("bad", nil, nil)
	if err == nil {
		t.Fatal("expected error for GraphQL errors response")
	}
	if got := err.Error(); got != "GraphQL errors: Field 'foo' not found" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestGraphQLFetchResourceNotFound(t *testing.T) {
	def := &config.ConnectorDef{
		Name:      "test-graphql",
		Protocol:  "graphql",
		Resources: map[string]config.ResourceDef{},
	}

	fetcher := NewGraphQLFetcher(def, nil)
	_, err := fetcher.Fetch("nonexistent", nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent resource")
	}
}

func TestGraphQLFetchNoQuery(t *testing.T) {
	def := &config.ConnectorDef{
		Name:     "test-graphql",
		Protocol: "graphql",
		Resources: map[string]config.ResourceDef{
			"empty": {DataField: "data"},
		},
	}

	fetcher := NewGraphQLFetcher(def, nil)
	_, err := fetcher.Fetch("empty", nil, nil)
	if err == nil {
		t.Fatal("expected error for resource with no query")
	}
}

func TestGraphQLFetchWithCustomHeader(t *testing.T) {
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Shopify-Storefront-Access-Token")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"products": []any{
					map[string]any{"id": "1", "title": "Test"},
				},
			},
		})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:     "shopify-storefront",
		Protocol: "graphql",
		BaseURL:  server.URL,
		Auth: config.AuthDef{
			Type:       "header",
			HeaderName: "X-Shopify-Storefront-Access-Token",
			TokenField: "storefront_token",
		},
		Resources: map[string]config.ResourceDef{
			"products": {
				Query:     `query { products(first: 10) { id title } }`,
				DataField: "products",
			},
		},
	}

	fetcher := NewGraphQLFetcher(def, map[string]string{"storefront_token": "token_abc"})
	records, err := fetcher.Fetch("products", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if receivedToken != "token_abc" {
		t.Errorf("expected storefront token 'token_abc', got %q", receivedToken)
	}
}

func TestNewFetcherFactory(t *testing.T) {
	def := &config.ConnectorDef{
		Name:     "test",
		Protocol: "",
		BaseURL:  "http://localhost",
		Auth:     config.AuthDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"items": {Path: "/items", Method: "GET"},
		},
	}
	ref := config.ConnectorRef{Type: "test"}

	f, err := NewFetcher(def, ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := f.(*HTTPFetcher); !ok {
		t.Error("expected HTTPFetcher for empty protocol")
	}

	def2 := &config.ConnectorDef{
		Name:     "test-gql",
		Protocol: "graphql",
		BaseURL:  "http://localhost",
		Auth:     config.AuthDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"q": {Query: "query { x }"},
		},
	}
	ref2 := config.ConnectorRef{Type: "test-gql"}

	f2, err := NewFetcher(def2, ref2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := f2.(*GraphQLFetcher); !ok {
		t.Error("expected GraphQLFetcher for graphql protocol")
	}

	def3 := &config.ConnectorDef{
		Name:     "test-bad",
		Protocol: "ftp",
	}
	ref3 := config.ConnectorRef{Type: "test-bad"}
	_, err = NewFetcher(def3, ref3)
	if err == nil {
		t.Error("expected error for unsupported protocol")
	}
}

func TestMergeInstanceResources(t *testing.T) {
	def := &config.ConnectorDef{
		Name: "test",
		Resources: map[string]config.ResourceDef{
			"existing": {Path: "/existing", Method: "GET"},
		},
	}
	ref := config.ConnectorRef{
		Type: "test",
		Resources: map[string]config.ResourceOverride{
			"existing": {DataField: "results"},
			"new_resource": {
				Query:     "SELECT * FROM users",
				DataField: "rows",
			},
		},
	}

	mergeInstanceResources(def, ref)

	// Check existing resource got the data_field override
	existing := def.Resources["existing"]
	if existing.Path != "/existing" {
		t.Errorf("expected path '/existing', got %q", existing.Path)
	}
	if existing.DataField != "results" {
		t.Errorf("expected data_field 'results', got %q", existing.DataField)
	}

	// Check new resource was added
	newRes := def.Resources["new_resource"]
	if newRes.Query != "SELECT * FROM users" {
		t.Errorf("expected query, got %q", newRes.Query)
	}
	if newRes.DataField != "rows" {
		t.Errorf("expected data_field 'rows', got %q", newRes.DataField)
	}
}
