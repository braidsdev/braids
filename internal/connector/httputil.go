package connector

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/braidsdev/braids/internal/config"
)

var configVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

// httpHelper provides shared auth and variable substitution for HTTP-based fetchers.
type httpHelper struct {
	def    *config.ConnectorDef
	config map[string]string
}

func (h *httpHelper) addAuth(req *http.Request) {
	switch h.def.Auth.Type {
	case "bearer":
		token := h.config[h.def.Auth.TokenField]
		req.Header.Set("Authorization", "Bearer "+token)
	case "header":
		token := h.config[h.def.Auth.TokenField]
		prefix := h.def.Auth.TokenPrefix
		req.Header.Set(h.def.Auth.HeaderName, prefix+token)
	case "basic":
		username := h.config[h.def.Auth.UsernameField]
		password := h.config[h.def.Auth.PasswordField]
		req.SetBasicAuth(username, password)
	case "query_param":
		// handled in Fetch() when building URL params
	}
	// Apply extra headers (e.g. Plaid's dual-header auth)
	for configKey, headerName := range h.def.Auth.ExtraHeaders {
		if val, ok := h.config[configKey]; ok {
			req.Header.Set(headerName, val)
		}
	}
}

func (h *httpHelper) substituteVars(s string) string {
	return configVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := configVarPattern.FindStringSubmatch(match)[1]
		if val, ok := h.config[varName]; ok {
			return val
		}
		return match
	})
}

// extractPath navigates a dot-separated field path in a nested map and returns
// the value as a slice of Records. Works for both JSON-decoded maps and arrays.
func extractPath(data map[string]any, dotPath string) ([]Record, error) {
	if dotPath == "" {
		return nil, nil
	}

	parts := strings.Split(dotPath, ".")
	var current any = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			current = v[part]
		default:
			return nil, nil
		}
	}

	switch v := current.(type) {
	case []any:
		return arrayToRecords(v), nil
	case map[string]any:
		return []Record{Record(v)}, nil
	default:
		return nil, nil
	}
}
