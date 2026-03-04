package gateway

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/braidsdev/braids/internal/config"
	"github.com/braidsdev/braids/internal/connector"
	"github.com/braidsdev/braids/internal/schema"
)

// matchEndpoint finds the endpoint matching the request path.
// It tries an exact match first (fast path), then falls back to
// segment-by-segment pattern matching where {name} segments are wildcards.
func matchEndpoint(path string, endpoints map[string]config.Endpoint) (config.Endpoint, map[string]string, bool) {
	// Fast path: exact match for static endpoints
	if ep, ok := endpoints[path]; ok {
		return ep, nil, true
	}

	reqSegments := strings.Split(strings.Trim(path, "/"), "/")

	for pattern, ep := range endpoints {
		patSegments := strings.Split(strings.Trim(pattern, "/"), "/")
		if len(patSegments) != len(reqSegments) {
			continue
		}

		params := map[string]string{}
		matched := true
		for i, pat := range patSegments {
			if strings.HasPrefix(pat, "{") && strings.HasSuffix(pat, "}") {
				name := pat[1 : len(pat)-1]
				params[name] = reqSegments[i]
			} else if pat != reqSegments[i] {
				matched = false
				break
			}
		}
		if matched && len(params) > 0 {
			return ep, params, true
		}
	}

	return config.Endpoint{}, nil, false
}

func (g *Gateway) handleRequest(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	cfg := g.cfg
	engines := g.engines
	debug := g.Debug
	g.mu.RUnlock()

	ep, pathParams, ok := matchEndpoint(r.URL.Path, cfg.Endpoints)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if debug {
		log.Printf("[DEBUG] Incoming: %s %s → matched endpoint, pathParams=%v", r.Method, r.URL.Path, pathParams)
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
			start := time.Now()
			records, err := engine.Fetch(src.Resource, src.Params, pathParams, src.Headers)
			if debug {
				log.Printf("[DEBUG] Source %s/%s: %d records, %v elapsed, err=%v",
					src.Connector, src.Resource, len(records), time.Since(start), err)
			}
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
