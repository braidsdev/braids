package schema

import (
	"fmt"
	"strings"

	"github.com/braidsdev/braids/internal/connector"
)

// EvalExpr evaluates a mapping expression against a record.
// Supported forms:
//   - Direct field: "email" → record["email"]
//   - Literal + field: "'stripe_' + id" → "stripe_" + record["id"]
//   - Concatenation: "first_name + ' ' + last_name"
func EvalExpr(expr string, record connector.Record) (any, error) {
	parts := strings.Split(expr, "+")
	if len(parts) == 1 {
		return resolveToken(strings.TrimSpace(expr), record), nil
	}

	var result strings.Builder
	for _, part := range parts {
		token := strings.TrimSpace(part)
		val := resolveToken(token, record)
		result.WriteString(fmt.Sprintf("%v", val))
	}
	return result.String(), nil
}

func resolveToken(token string, record connector.Record) any {
	// Quoted string literal
	if len(token) >= 2 && token[0] == '\'' && token[len(token)-1] == '\'' {
		return token[1 : len(token)-1]
	}
	// Field reference
	if val, ok := record[token]; ok {
		return val
	}
	return ""
}
