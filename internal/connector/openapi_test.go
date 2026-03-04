package connector

import (
	"testing"
)

func TestParseOpenAPIResourcesBasic(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"paths": {
			"/v1/widgets": {
				"get": {"operationId": "GetWidgets"},
				"post": {"operationId": "PostWidgets"}
			},
			"/v1/widgets/{id}": {
				"get": {"operationId": "GetWidgetsId"},
				"delete": {"operationId": "DeleteWidgetsId"}
			}
		}
	}`

	resources, err := ParseOpenAPIResources([]byte(spec), "/v1")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(resources) != 4 {
		t.Fatalf("expected 4 resources, got %d", len(resources))
	}

	tests := []struct {
		key    string
		path   string
		method string
	}{
		{"get_widgets", "/widgets", "GET"},
		{"post_widgets", "/widgets", "POST"},
		{"get_widgets_id", "/widgets/{id}", "GET"},
		{"delete_widgets_id", "/widgets/{id}", "DELETE"},
	}

	for _, tt := range tests {
		res, ok := resources[tt.key]
		if !ok {
			t.Errorf("expected resource %q", tt.key)
			continue
		}
		if res.Path != tt.path {
			t.Errorf("%q: expected path %q, got %q", tt.key, tt.path, res.Path)
		}
		if res.Method != tt.method {
			t.Errorf("%q: expected method %q, got %q", tt.key, tt.method, res.Method)
		}
	}
}

func TestParseOpenAPIResourcesNoBasePath(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"paths": {
			"/items": {
				"get": {"operationId": "ListItems"}
			}
		}
	}`

	resources, err := ParseOpenAPIResources([]byte(spec), "")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	res, ok := resources["list_items"]
	if !ok {
		t.Fatal("expected 'list_items' resource")
	}
	if res.Path != "/items" {
		t.Errorf("expected path '/items', got %q", res.Path)
	}
}

func TestParseOpenAPIResourcesSkipsNonHTTPMethods(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"paths": {
			"/things": {
				"get": {"operationId": "GetThings"},
				"parameters": [{"name": "limit"}]
			}
		}
	}`

	resources, err := ParseOpenAPIResources([]byte(spec), "")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(resources))
	}
	if _, ok := resources["get_things"]; !ok {
		t.Error("expected 'get_things' resource")
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GetCustomers", "get_customers"},
		{"PostCustomers", "post_customers"},
		{"GetCustomersCustomer", "get_customers_customer"},
		{"DeleteCustomersCustomer", "delete_customers_customer"},
		{"GetAccountsAccountBankAccounts", "get_accounts_account_bank_accounts"},
		{"ListItems", "list_items"},
		{"GetV1Things", "get_v1_things"},
	}

	for _, tt := range tests {
		got := camelToSnake(tt.input)
		if got != tt.want {
			t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractBasePath(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://api.stripe.com/v1", "/v1"},
		{"https://api.example.com/api/v2", "/api/v2"},
		{"https://api.example.com", ""},
		{"not-a-url", ""},
	}

	for _, tt := range tests {
		got := extractBasePath(tt.url)
		if got != tt.want {
			t.Errorf("extractBasePath(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
