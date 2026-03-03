package schema

import (
	"testing"

	"github.com/braidsdev/braids/internal/connector"
)

func TestMergeNoKey(t *testing.T) {
	g1 := []connector.Record{{"id": "1"}, {"id": "2"}}
	g2 := []connector.Record{{"id": "3"}}
	result := Merge([][]connector.Record{g1, g2}, "", "")
	if len(result) != 3 {
		t.Errorf("expected 3 records, got %d", len(result))
	}
}

func TestMergeOnEmail(t *testing.T) {
	g1 := []connector.Record{
		{"id": "stripe_1", "email": "a@b.com", "name": "Alice", "created_at": "2024-01-01T00:00:00Z"},
		{"id": "stripe_2", "email": "c@d.com", "name": "Charlie", "created_at": "2024-01-01T00:00:00Z"},
	}
	g2 := []connector.Record{
		{"id": "shopify_1", "email": "a@b.com", "name": "Alice Smith", "created_at": "2024-06-01T00:00:00Z"},
	}
	result := Merge([][]connector.Record{g1, g2}, "email", "prefer_latest")

	if len(result) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result))
	}

	// The merged record for a@b.com should prefer the later created_at (shopify)
	for _, rec := range result {
		if rec["email"] == "a@b.com" {
			if rec["name"] != "Alice Smith" {
				t.Errorf("expected merged name 'Alice Smith' (prefer_latest), got %v", rec["name"])
			}
			return
		}
	}
	t.Error("did not find merged record for a@b.com")
}

func TestMergeSingleSourcePassthrough(t *testing.T) {
	g1 := []connector.Record{
		{"id": "1", "email": "unique@test.com"},
	}
	result := Merge([][]connector.Record{g1}, "email", "prefer_latest")
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}
