package schema

import (
	"testing"
)

func TestCoerceString(t *testing.T) {
	if v := Coerce(123, "string"); v != "123" {
		t.Errorf("expected '123', got %v", v)
	}
}

func TestCoerceInt(t *testing.T) {
	if v := Coerce(3.14, "int"); v != 3 {
		t.Errorf("expected 3, got %v", v)
	}
	if v := Coerce("42", "int"); v != 42 {
		t.Errorf("expected 42, got %v", v)
	}
}

func TestCoerceDatetimeUnix(t *testing.T) {
	v := Coerce(float64(1704067200), "datetime")
	if v != "2024-01-01T00:00:00Z" {
		t.Errorf("expected 2024-01-01T00:00:00Z, got %v", v)
	}
}

func TestCoerceDatetimeISO(t *testing.T) {
	v := Coerce("2024-01-01T12:00:00+05:00", "datetime")
	if v != "2024-01-01T07:00:00Z" {
		t.Errorf("expected 2024-01-01T07:00:00Z, got %v", v)
	}
}

func TestCoerceNil(t *testing.T) {
	if v := Coerce(nil, "string"); v != nil {
		t.Errorf("expected nil, got %v", v)
	}
}
