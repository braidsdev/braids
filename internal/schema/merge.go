package schema

import (
	"fmt"
	"time"

	"github.com/braidsdev/braids/internal/connector"
)

// Merge combines multiple record sets by grouping on mergeOn field.
// Records with the same mergeOn value are merged using the conflict resolution strategy.
func Merge(groups [][]connector.Record, mergeOn, conflictRes string) []connector.Record {
	if mergeOn == "" {
		// No merge key — just concatenate
		var all []connector.Record
		for _, g := range groups {
			all = append(all, g...)
		}
		return all
	}

	// Index records by merge key
	index := make(map[string][]entry)
	var order []string

	for srcIdx, records := range groups {
		for _, rec := range records {
			key := fmt.Sprintf("%v", rec[mergeOn])
			if _, seen := index[key]; !seen {
				order = append(order, key)
			}
			index[key] = append(index[key], entry{record: rec, source: srcIdx})
		}
	}

	result := make([]connector.Record, 0, len(order))
	for _, key := range order {
		entries := index[key]
		if len(entries) == 1 {
			result = append(result, entries[0].record)
			continue
		}
		merged := mergeRecords(entries, conflictRes)
		result = append(result, merged)
	}
	return result
}

func mergeRecords(entries []entry, conflictRes string) connector.Record {
	if conflictRes == "prefer_latest" {
		return mergePreferLatest(entries)
	}
	// Default: last source wins
	merged := make(connector.Record)
	for _, e := range entries {
		for k, v := range e.record {
			merged[k] = v
		}
	}
	return merged
}

func mergePreferLatest(entries []entry) connector.Record {
	// Find the entry with the latest created_at
	best := 0
	bestTime := time.Time{}

	for i, e := range entries {
		if t := parseTime(e.record["created_at"]); !t.IsZero() && t.After(bestTime) {
			best = i
			bestTime = t
		}
	}

	// Start with oldest, layer newer on top
	merged := make(connector.Record)
	for i := range entries {
		if i == best {
			continue
		}
		for k, v := range entries[i].record {
			if v != nil && v != "" {
				merged[k] = v
			}
		}
	}
	// Overlay the "best" (latest) record on top
	for k, v := range entries[best].record {
		if v != nil && v != "" {
			merged[k] = v
		}
	}
	return merged
}

type entry struct {
	record connector.Record
	source int
}

func parseTime(val any) time.Time {
	if val == nil {
		return time.Time{}
	}
	s, ok := val.(string)
	if !ok {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
