package connector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/braidsdev/braids/internal/config"
)

// GraphQLFetcher executes GraphQL queries for a given connector definition.
type GraphQLFetcher struct {
	httpHelper
	client *http.Client
}

// NewGraphQLFetcher creates a GraphQLFetcher from a connector definition and user config.
func NewGraphQLFetcher(def *config.ConnectorDef, cfg map[string]string) *GraphQLFetcher {
	return &GraphQLFetcher{
		httpHelper: httpHelper{def: def, config: cfg},
		client:     &http.Client{},
	}
}

// Fetch executes a GraphQL query for the named resource.
// The resource's Query field provides the GraphQL query string.
// params become GraphQL variables. pathParams are unused but accepted for interface compatibility.
func (g *GraphQLFetcher) Fetch(resource string, params map[string]any, pathParams map[string]string, headers ...map[string]string) ([]Record, error) {
	res, ok := g.def.Resources[resource]
	if !ok {
		return nil, fmt.Errorf("resource %q not found in connector %q", resource, g.def.Name)
	}

	if res.Query == "" {
		return nil, fmt.Errorf("resource %q in connector %q has no query defined", resource, g.def.Name)
	}

	// Build the GraphQL request body
	gqlBody := map[string]any{
		"query": res.Query,
	}
	if len(params) > 0 {
		gqlBody["variables"] = params
	}

	bodyBytes, err := json.Marshal(gqlBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling GraphQL request: %w", err)
	}

	// Build the URL — use baseURL, optionally append resource path
	endpoint := g.substituteVars(g.def.BaseURL)
	if res.Path != "" {
		endpoint += res.Path
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating GraphQL request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	g.addAuth(req)

	// Apply additional headers
	if len(headers) > 0 && headers[0] != nil {
		for key, val := range headers[0] {
			req.Header.Set(key, val)
		}
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing GraphQL request: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("reading GraphQL response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GraphQL endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, fmt.Errorf("parsing GraphQL response: %w", err)
	}

	// Check for GraphQL-level errors
	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %s", gqlResp.Errors[0].Message)
	}

	// Navigate data_field path to extract records
	if res.DataField != "" {
		records, err := extractPath(gqlResp.Data, res.DataField)
		if err != nil {
			return nil, fmt.Errorf("extracting data_field %q: %w", res.DataField, err)
		}
		if records != nil {
			return records, nil
		}
	}

	// No data_field — return the entire data map as a single record
	if gqlResp.Data != nil {
		return []Record{Record(gqlResp.Data)}, nil
	}

	return nil, nil
}

type graphQLResponse struct {
	Data   map[string]any  `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

type graphQLError struct {
	Message string `json:"message"`
}
