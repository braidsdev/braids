package gateway

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/braidsdev/braids/internal/config"
	"github.com/braidsdev/braids/internal/connector"
	"github.com/braidsdev/braids/internal/schema"
)

func (g *Gateway) handleRequest(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	cfg := g.cfg
	engines := g.engines
	g.mu.RUnlock()

	ep, ok := cfg.Endpoints[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}

	schemaDef, ok := cfg.Schemas[ep.Schema]
	if !ok {
		http.Error(w, "schema not found", http.StatusInternalServerError)
		return
	}

	// Fetch from all sources in parallel
	type sourceResult struct {
		index   int
		records []connector.Record
		err     error
		name    string
	}

	results := make([]sourceResult, len(ep.Sources))
	var wg sync.WaitGroup

	for i, src := range ep.Sources {
		wg.Add(1)
		go func(idx int, src config.Source) {
			defer wg.Done()
			engine, ok := engines[src.Connector]
			if !ok {
				results[idx] = sourceResult{index: idx, name: src.Connector, err: errConnectorNotFound(src.Connector)}
				return
			}
			records, err := engine.Fetch(src.Resource)
			results[idx] = sourceResult{index: idx, name: src.Connector, records: records, err: err}
		}(i, src)
	}
	wg.Wait()

	// Apply mappings and collect
	var groups [][]connector.Record
	var sources []string
	var fetchErrors []map[string]string

	for i, res := range results {
		if res.err != nil {
			log.Printf("Error fetching from %s: %v", res.name, res.err)
			fetchErrors = append(fetchErrors, map[string]string{
				"source": res.name,
				"error":  res.err.Error(),
			})
			continue
		}

		mapped, err := schema.Apply(res.records, ep.Sources[i].Mapping, schemaDef.Fields)
		if err != nil {
			log.Printf("Error mapping %s: %v", res.name, err)
			fetchErrors = append(fetchErrors, map[string]string{
				"source": res.name,
				"error":  err.Error(),
			})
			continue
		}

		groups = append(groups, mapped)
		sources = append(sources, res.name)
	}

	// Merge
	merged := schema.Merge(groups, schemaDef.MergeOn, schemaDef.ConflictResolution)

	// Build response
	meta := map[string]any{
		"total":   len(merged),
		"sources": sources,
	}
	if len(fetchErrors) > 0 {
		meta["errors"] = fetchErrors
	}

	response := map[string]any{
		"data": merged,
		"meta": meta,
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

type connectorNotFoundError struct {
	name string
}

func (e *connectorNotFoundError) Error() string {
	return "connector not found: " + e.name
}

func errConnectorNotFound(name string) error {
	return &connectorNotFoundError{name: name}
}
