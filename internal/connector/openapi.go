package connector

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/braidsdev/braids/internal/config"
)

// openAPISpec is a minimal representation of an OpenAPI 3.x document,
// containing only the fields we need for resource discovery.
type openAPISpec struct {
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

type openAPIOperation struct {
	OperationID string `json:"operationId"`
}

var camelToSnakeRe = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// ParseOpenAPIResources parses an OpenAPI 3.x JSON spec and returns a map of
// resources keyed by the snake_case operationId. The basePath prefix (e.g. "/v1")
// is stripped from paths since the connector's base_url already includes it.
func ParseOpenAPIResources(data []byte, basePath string) (map[string]config.ResourceDef, error) {
	var spec openAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI spec: %w", err)
	}

	resources := make(map[string]config.ResourceDef, len(spec.Paths)*2)
	httpMethods := map[string]bool{
		"get": true, "post": true, "put": true, "patch": true, "delete": true,
	}

	for path, methods := range spec.Paths {
		for method, raw := range methods {
			if !httpMethods[strings.ToLower(method)] {
				continue
			}

			var op openAPIOperation
			if err := json.Unmarshal(raw, &op); err != nil || op.OperationID == "" {
				continue
			}

			key := camelToSnake(op.OperationID)
			trimmedPath := strings.TrimPrefix(path, basePath)
			if trimmedPath == "" {
				trimmedPath = "/"
			}

			resources[key] = config.ResourceDef{
				Path:   trimmedPath,
				Method: strings.ToUpper(method),
			}
		}
	}

	return resources, nil
}

// camelToSnake converts a CamelCase string to snake_case.
// e.g. "GetCustomers" -> "get_customers", "GetCustomersCustomer" -> "get_customers_customer"
func camelToSnake(s string) string {
	snake := camelToSnakeRe.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snake)
}
