package schema

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// Coerce converts a value to the specified type.
func Coerce(val any, targetType string) any {
	if val == nil {
		return nil
	}

	switch targetType {
	case "string":
		return fmt.Sprintf("%v", val)

	case "int":
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
		return val

	case "float":
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
		return val

	case "datetime":
		return coerceDatetime(val)

	default:
		return val
	}
}

// coerceDatetime converts unix timestamps and ISO 8601 strings to RFC 3339.
func coerceDatetime(val any) any {
	switch v := val.(type) {
	case float64:
		// Unix timestamp
		sec := int64(v)
		nsec := int64(math.Round((v - float64(sec)) * 1e9))
		return time.Unix(sec, nsec).UTC().Format(time.RFC3339)
	case int:
		return time.Unix(int64(v), 0).UTC().Format(time.RFC3339)
	case string:
		// Try RFC 3339 first, then common ISO 8601 variants
		for _, layout := range []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02T15:04:05",
			"2006-01-02",
		} {
			if t, err := time.Parse(layout, v); err == nil {
				return t.UTC().Format(time.RFC3339)
			}
		}
		return v
	default:
		return val
	}
}
