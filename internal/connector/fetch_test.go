package connector

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	engine := NewHTTPFetcher(def, nil)

	records, err := engine.Fetch("users", nil, nil)
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
	engine := NewHTTPFetcher(def, nil)

	records, err := engine.Fetch("users", nil, nil)
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

func TestFetchWithParams(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": "cus_1", "email": "alice@test.com"},
		})
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
			"customers": {Path: "/customers", Method: "GET"},
		},
	}
	engine := NewHTTPFetcher(def, nil)

	params := map[string]any{
		"limit":      100,
		"expand[]":   "data.customer",
		"active":     "true",
	}

	records, err := engine.Fetch("customers", params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	// Verify query params were sent
	if receivedQuery == "" {
		t.Fatal("expected query parameters to be sent")
	}

	// Parse and check individual params
	parsed, err := url.ParseQuery(receivedQuery)
	if err != nil {
		t.Fatalf("failed to parse query: %v", err)
	}
	if parsed.Get("limit") != "100" {
		t.Errorf("expected limit=100, got %q", parsed.Get("limit"))
	}
	if parsed.Get("expand[]") != "data.customer" {
		t.Errorf("expected expand[]=data.customer, got %q", parsed.Get("expand[]"))
	}
	if parsed.Get("active") != "true" {
		t.Errorf("expected active=true, got %q", parsed.Get("active"))
	}
}

func TestFetchWithArrayParams(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": "1"}})
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
			"charges": {Path: "/charges", Method: "GET"},
		},
	}
	engine := NewHTTPFetcher(def, nil)

	params := map[string]any{
		"expand[]": []any{"data.customer", "data.invoice"},
	}

	_, err := engine.Fetch("charges", params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := url.ParseQuery(receivedQuery)
	if err != nil {
		t.Fatalf("failed to parse query: %v", err)
	}
	expandVals := parsed["expand[]"]
	if len(expandVals) != 2 {
		t.Fatalf("expected 2 expand[] values, got %d: %v", len(expandVals), expandVals)
	}
}

func TestFetchWithIndexedArrayParams(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": "1"}})
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
			"accounts": {Path: "/v2/core/accounts", Method: "GET"},
		},
	}
	engine := NewHTTPFetcher(def, nil)

	// Key without [] suffix -> indexed encoding: include[0]=a&include[1]=b
	params := map[string]any{
		"include": []any{"configuration.customer", "identity"},
	}

	_, err := engine.Fetch("accounts", params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := url.ParseQuery(receivedQuery)
	if err != nil {
		t.Fatalf("failed to parse query: %v", err)
	}
	if got := parsed.Get("include[0]"); got != "configuration.customer" {
		t.Errorf("expected include[0]=configuration.customer, got %q", got)
	}
	if got := parsed.Get("include[1]"); got != "identity" {
		t.Errorf("expected include[1]=identity, got %q", got)
	}
}

func TestFetchWithHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": "acct_1"}})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:    "test",
		BaseURL: server.URL,
		Auth:    config.AuthDef{Type: "bearer", TokenField: "api_key"},
		Pagination: config.PaginationDef{
			Type: "none",
		},
		Resources: map[string]config.ResourceDef{
			"accounts": {Path: "/v2/core/accounts", Method: "GET"},
		},
	}
	engine := NewHTTPFetcher(def, map[string]string{"api_key": "sk_test_123"})

	headers := map[string]string{
		"Stripe-Version": "2025-01-27.acacia",
	}

	records, err := engine.Fetch("accounts", nil, nil, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if got := receivedHeaders.Get("Stripe-Version"); got != "2025-01-27.acacia" {
		t.Errorf("expected Stripe-Version header '2025-01-27.acacia', got %q", got)
	}
	if got := receivedHeaders.Get("Authorization"); got != "Bearer sk_test_123" {
		t.Errorf("expected Authorization header, got %q", got)
	}
}

func TestFetchWithPathParams(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":           "acct_123",
			"display_name": "Test Account",
		})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:    "test",
		BaseURL: server.URL,
		Auth:    config.AuthDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"get_account": {Path: "/v2/core/accounts/{id}", Method: "GET"},
		},
	}
	engine := NewHTTPFetcher(def, nil)

	pathParams := map[string]string{"id": "acct_123"}
	records, err := engine.Fetch("get_account", nil, pathParams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if receivedPath != "/v2/core/accounts/acct_123" {
		t.Errorf("expected path /v2/core/accounts/acct_123, got %s", receivedPath)
	}
	if records[0]["id"] != "acct_123" {
		t.Errorf("expected id acct_123, got %v", records[0]["id"])
	}
}

func TestFetchResourceNotFound(t *testing.T) {
	def := &config.ConnectorDef{
		Name:      "test",
		Resources: map[string]config.ResourceDef{},
	}
	engine := NewHTTPFetcher(def, nil)

	_, err := engine.Fetch("nonexistent", nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent resource")
	}
}

func TestBasicAuth(t *testing.T) {
	var receivedUser, receivedPass string
	var receivedOK bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUser, receivedPass, receivedOK = r.BasicAuth()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": "1"}})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:    "test",
		BaseURL: server.URL,
		Auth: config.AuthDef{
			Type:          "basic",
			UsernameField: "username",
			PasswordField: "password",
		},
		Pagination: config.PaginationDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"items": {Path: "/items", Method: "GET"},
		},
	}
	engine := NewHTTPFetcher(def, map[string]string{"username": "myuser", "password": "mypass"})

	_, err := engine.Fetch("items", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !receivedOK {
		t.Fatal("expected basic auth to be set")
	}
	if receivedUser != "myuser" {
		t.Errorf("expected username 'myuser', got %q", receivedUser)
	}
	if receivedPass != "mypass" {
		t.Errorf("expected password 'mypass', got %q", receivedPass)
	}
}

func TestQueryParamAuth(t *testing.T) {
	var receivedQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": "1"}})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:    "test",
		BaseURL: server.URL,
		Auth: config.AuthDef{
			Type:       "query_param",
			TokenField: "api_key",
			ParamName:  "key",
		},
		Pagination: config.PaginationDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"items": {Path: "/items", Method: "GET"},
		},
	}
	engine := NewHTTPFetcher(def, map[string]string{"api_key": "secret123"})

	_, err := engine.Fetch("items", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := receivedQuery.Get("key"); got != "secret123" {
		t.Errorf("expected query param key=secret123, got %q", got)
	}
}

func TestTokenPrefix(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": "1"}})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:    "test",
		BaseURL: server.URL,
		Auth: config.AuthDef{
			Type:        "header",
			HeaderName:  "Authorization",
			TokenField:  "token",
			TokenPrefix: "Bot ",
		},
		Pagination: config.PaginationDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"guilds": {Path: "/guilds", Method: "GET"},
		},
	}
	engine := NewHTTPFetcher(def, map[string]string{"token": "mytoken123"})

	_, err := engine.Fetch("guilds", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAuth != "Bot mytoken123" {
		t.Errorf("expected 'Bot mytoken123', got %q", receivedAuth)
	}
}

func TestExtraHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{{"id": "1"}})
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:    "test",
		BaseURL: server.URL,
		Auth: config.AuthDef{
			Type:       "header",
			HeaderName: "PLAID-CLIENT-ID",
			TokenField: "client_id",
			ExtraHeaders: map[string]string{
				"secret": "PLAID-SECRET",
			},
		},
		Pagination: config.PaginationDef{Type: "none"},
		Resources: map[string]config.ResourceDef{
			"accounts": {Path: "/accounts", Method: "GET"},
		},
	}
	engine := NewHTTPFetcher(def, map[string]string{
		"client_id": "my_client_id",
		"secret":    "my_secret",
	})

	_, err := engine.Fetch("accounts", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := receivedHeaders.Get("PLAID-CLIENT-ID"); got != "my_client_id" {
		t.Errorf("expected PLAID-CLIENT-ID 'my_client_id', got %q", got)
	}
	if got := receivedHeaders.Get("PLAID-SECRET"); got != "my_secret" {
		t.Errorf("expected PLAID-SECRET 'my_secret', got %q", got)
	}
}

func TestOffsetPagination(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		offset := r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")

		switch offset {
		case "", "0":
			json.NewEncoder(w).Encode(map[string]any{
				"results": []any{
					map[string]any{"id": 1},
					map[string]any{"id": 2},
				},
				"total": 3,
			})
		case "2":
			json.NewEncoder(w).Encode(map[string]any{
				"results": []any{
					map[string]any{"id": 3},
				},
				"total": 3,
			})
		default:
			json.NewEncoder(w).Encode(map[string]any{
				"results": []any{},
				"total":   3,
			})
		}
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:    "test",
		BaseURL: server.URL,
		Auth:    config.AuthDef{Type: "none"},
		Pagination: config.PaginationDef{
			Type:         "offset",
			OffsetParam:  "offset",
			LimitParam:   "limit",
			LimitDefault: 2,
			TotalField:   "total",
			DataField:    "results",
		},
		Resources: map[string]config.ResourceDef{
			"items": {Path: "/items", Method: "GET", DataField: "results"},
		},
	}
	engine := NewHTTPFetcher(def, nil)

	records, err := engine.Fetch("items", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestPagePagination(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")

		switch page {
		case "", "1":
			json.NewEncoder(w).Encode(map[string]any{
				"data":  []any{map[string]any{"id": 1}, map[string]any{"id": 2}},
				"total": 3,
			})
		case "2":
			json.NewEncoder(w).Encode(map[string]any{
				"data":  []any{map[string]any{"id": 3}},
				"total": 3,
			})
		default:
			json.NewEncoder(w).Encode(map[string]any{
				"data":  []any{},
				"total": 3,
			})
		}
	}))
	defer server.Close()

	def := &config.ConnectorDef{
		Name:    "test",
		BaseURL: server.URL,
		Auth:    config.AuthDef{Type: "none"},
		Pagination: config.PaginationDef{
			Type:         "page",
			PageParam:    "page",
			LimitParam:   "per_page",
			LimitDefault: 2,
			TotalField:   "total",
			DataField:    "data",
		},
		Resources: map[string]config.ResourceDef{
			"items": {Path: "/items", Method: "GET", DataField: "data"},
		},
	}
	engine := NewHTTPFetcher(def, nil)

	records, err := engine.Fetch("items", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}
