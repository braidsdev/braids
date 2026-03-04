package connector

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var configVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

// pathParamPattern matches {name} placeholders in resource paths.
var pathParamPattern = regexp.MustCompile(`\{(\w+)\}`)

// substitutePath replaces {name} placeholders in a path with values from pathParams.
func substitutePath(path string, pathParams map[string]string) string {
	if len(pathParams) == 0 {
		return path
	}
	return pathParamPattern.ReplaceAllStringFunc(path, func(match string) string {
		name := pathParamPattern.FindStringSubmatch(match)[1]
		if val, ok := pathParams[name]; ok {
			return val
		}
		return match
	})
}

// Fetch retrieves all records for a resource, handling auth and pagination.
// Static query parameters from params and headers are included on every request.
// pathParams substitutes {name} placeholders in the resource path.
func (c *ConnectorEngine) Fetch(resource string, params map[string]any, pathParams map[string]string, headers ...map[string]string) ([]Record, error) {
	res, ok := c.def.Resources[resource]
	if !ok {
		return nil, fmt.Errorf("resource %q not found in connector %q", resource, c.def.Name)
	}

	baseURL := c.substituteVars(c.def.BaseURL)
	resolvedPath := substitutePath(res.Path, pathParams)
	fetchURL := baseURL + resolvedPath

	// Build static query params from source config
	staticParams := buildQueryParams(params)

	if len(staticParams) > 0 {
		fetchURL += "?" + staticParams.Encode()
	}

	var allRecords []Record

	for {
		req, err := http.NewRequest(res.Method, fetchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		c.addAuth(req)

		// Apply static headers from source config
		if len(headers) > 0 && headers[0] != nil {
			for key, val := range headers[0] {
				req.Header.Set(key, val)
			}
		}

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", fetchURL, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("upstream %s returned %d: %s", fetchURL, resp.StatusCode, string(body))
		}

		var parsed any
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("parsing response JSON: %w", err)
		}

		// Handle bare JSON arrays (e.g. JSONPlaceholder returns [{...}, ...])
		if arr, ok := parsed.([]any); ok {
			records := arrayToRecords(arr)
			allRecords = append(allRecords, records...)
			return allRecords, nil
		}

		raw, ok := parsed.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unexpected JSON type in response")
		}

		// If no data_field is defined and the response is a plain object,
		// treat it as a single record (e.g. GET /resource/{id}).
		if res.DataField == "" && c.def.Pagination.DataField == "" {
			allRecords = append(allRecords, Record(raw))
			return allRecords, nil
		}

		records, err := c.extractRecords(raw, res.DataField)
		if err != nil {
			return nil, err
		}
		allRecords = append(allRecords, records...)

		// Handle pagination
		switch c.def.Pagination.Type {
		case "cursor":
			hasMore, _ := raw[c.def.Pagination.HasMoreField].(bool)
			if !hasMore || len(records) == 0 {
				return allRecords, nil
			}
			lastRecord := records[len(records)-1]
			cursor, _ := lastRecord[c.def.Pagination.CursorField].(string)
			if cursor == "" {
				return allRecords, nil
			}
			// Rebuild URL with cursor param, preserving static params
			paginatedParams := buildQueryParams(params)
			paginatedParams.Set(c.def.Pagination.CursorParam, cursor)
			fetchURL = baseURL + resolvedPath + "?" + paginatedParams.Encode()

		case "link_header":
			nextURL := parseLinkHeaderNext(resp.Header.Get("Link"))
			if nextURL == "" {
				return allRecords, nil
			}
			fetchURL = nextURL

		default:
			return allRecords, nil
		}
	}
}

// buildQueryParams converts a map[string]any to url.Values.
// String values become single params. Array values use the encoding style
// implied by the key:
//   - "expand[]": [a, b] → expand[]=a&expand[]=b  (repeated key, v1-style)
//   - "include":  [a, b] → include[0]=a&include[1]=b (indexed, v2-style)
func buildQueryParams(params map[string]any) url.Values {
	vals := url.Values{}
	for key, val := range params {
		switch v := val.(type) {
		case string:
			vals.Add(key, v)
		case int:
			vals.Add(key, fmt.Sprintf("%d", v))
		case float64:
			if v == float64(int(v)) {
				vals.Add(key, fmt.Sprintf("%d", int(v)))
			} else {
				vals.Add(key, fmt.Sprintf("%g", v))
			}
		case bool:
			vals.Add(key, fmt.Sprintf("%t", v))
		case []any:
			if strings.HasSuffix(key, "[]") {
				// Repeated key style: expand[]=a&expand[]=b
				for _, item := range v {
					vals.Add(key, fmt.Sprintf("%v", item))
				}
			} else {
				// Indexed style: include[0]=a&include[1]=b
				for i, item := range v {
					vals.Add(fmt.Sprintf("%s[%d]", key, i), fmt.Sprintf("%v", item))
				}
			}
		default:
			vals.Add(key, fmt.Sprintf("%v", v))
		}
	}
	return vals
}

func (c *ConnectorEngine) addAuth(req *http.Request) {
	switch c.def.Auth.Type {
	case "bearer":
		token := c.config[c.def.Auth.TokenField]
		req.Header.Set("Authorization", "Bearer "+token)
	case "header":
		token := c.config[c.def.Auth.TokenField]
		req.Header.Set(c.def.Auth.HeaderName, token)
	}
}

func (c *ConnectorEngine) extractRecords(raw map[string]any, resourceDataField string) ([]Record, error) {
	// Resource-level data_field takes precedence, then pagination-level
	dataField := resourceDataField
	if dataField == "" {
		dataField = c.def.Pagination.DataField
	}

	if dataField == "" {
		return nil, fmt.Errorf("no data_field defined for extracting records")
	}

	arr, ok := raw[dataField].([]any)
	if !ok {
		return nil, fmt.Errorf("expected array at %q in response", dataField)
	}

	records := make([]Record, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			records = append(records, Record(m))
		}
	}
	return records, nil
}

func (c *ConnectorEngine) substituteVars(s string) string {
	return configVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := configVarPattern.FindStringSubmatch(match)[1]
		if val, ok := c.config[varName]; ok {
			return val
		}
		return match
	})
}

// arrayToRecords converts a []any to []Record, skipping non-object elements.
func arrayToRecords(arr []any) []Record {
	records := make([]Record, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			records = append(records, Record(m))
		}
	}
	return records
}

// parseLinkHeaderNext extracts the URL for rel="next" from a Link header.
func parseLinkHeaderNext(header string) string {
	if header == "" {
		return ""
	}
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, `rel="next"`) {
			// Extract URL between < and >
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}
