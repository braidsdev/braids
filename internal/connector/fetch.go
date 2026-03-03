package connector

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var configVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

// Fetch retrieves all records for a resource, handling auth and pagination.
func (c *ConnectorEngine) Fetch(resource string) ([]Record, error) {
	res, ok := c.def.Resources[resource]
	if !ok {
		return nil, fmt.Errorf("resource %q not found in connector %q", resource, c.def.Name)
	}

	baseURL := c.substituteVars(c.def.BaseURL)
	url := baseURL + res.Path

	var allRecords []Record

	for {
		req, err := http.NewRequest(res.Method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		c.addAuth(req)

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", url, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("upstream %s returned %d: %s", url, resp.StatusCode, string(body))
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
			// Rebuild URL with cursor param
			url = baseURL + res.Path + "?" + c.def.Pagination.CursorParam + "=" + cursor

		case "link_header":
			nextURL := parseLinkHeaderNext(resp.Header.Get("Link"))
			if nextURL == "" {
				return allRecords, nil
			}
			url = nextURL

		default:
			return allRecords, nil
		}
	}
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
