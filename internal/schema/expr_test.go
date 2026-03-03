package schema

import (
	"testing"

	"github.com/braidsdev/braids/internal/connector"
)

func TestEvalExprDirectField(t *testing.T) {
	rec := connector.Record{"email": "test@example.com"}
	val, err := EvalExpr("email", rec)
	if err != nil {
		t.Fatal(err)
	}
	if val != "test@example.com" {
		t.Errorf("expected test@example.com, got %v", val)
	}
}

func TestEvalExprLiteralPlusField(t *testing.T) {
	rec := connector.Record{"id": "123"}
	val, err := EvalExpr("'stripe_' + id", rec)
	if err != nil {
		t.Fatal(err)
	}
	if val != "stripe_123" {
		t.Errorf("expected stripe_123, got %v", val)
	}
}

func TestEvalExprConcatenation(t *testing.T) {
	rec := connector.Record{"first_name": "Jane", "last_name": "Doe"}
	val, err := EvalExpr("first_name + ' ' + last_name", rec)
	if err != nil {
		t.Fatal(err)
	}
	if val != "Jane Doe" {
		t.Errorf("expected 'Jane Doe', got %v", val)
	}
}

func TestEvalExprMissingField(t *testing.T) {
	rec := connector.Record{}
	val, err := EvalExpr("missing", rec)
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Errorf("expected empty string for missing field, got %v", val)
	}
}
