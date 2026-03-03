package schema

import (
	"fmt"

	"github.com/braidsdev/braids/internal/config"
	"github.com/braidsdev/braids/internal/connector"
)

// Apply transforms raw upstream records using the field mapping and schema field types.
func Apply(records []connector.Record, mapping map[string]string, fields map[string]config.Field) ([]connector.Record, error) {
	result := make([]connector.Record, 0, len(records))
	for _, rec := range records {
		mapped := make(connector.Record, len(mapping))
		for fieldName, expr := range mapping {
			val, err := EvalExpr(expr, rec)
			if err != nil {
				return nil, fmt.Errorf("evaluating %q for field %q: %w", expr, fieldName, err)
			}
			if f, ok := fields[fieldName]; ok {
				val = Coerce(val, f.Type)
			}
			mapped[fieldName] = val
		}
		result = append(result, mapped)
	}
	return result, nil
}
