// Package gateway implements the HTTP server that handles routing
// and response composition.
package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/braidsdev/braids/internal/config"
	"github.com/braidsdev/braids/internal/connector"
	"github.com/fsnotify/fsnotify"
)

// Gateway is the HTTP server that serves unified API endpoints.
type Gateway struct {
	configPath string
	configDir  string
	cfg        *config.Config
	engines    map[string]*connector.ConnectorEngine
	mu         sync.RWMutex
	server     *http.Server
}

// New creates a Gateway from a config file path.
func New(configPath string) (*Gateway, error) {
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	g := &Gateway{
		configPath: absPath,
		configDir:  filepath.Dir(absPath),
	}

	if err := g.loadConfig(); err != nil {
		return nil, err
	}

	return g, nil
}

func (g *Gateway) loadConfig() error {
	cfg, err := config.Load(g.configPath)
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	engines := make(map[string]*connector.ConnectorEngine, len(cfg.Connectors))
	for name, ref := range cfg.Connectors {
		def, err := connector.LoadDef(ref.Type, g.configDir)
		if err != nil {
			return fmt.Errorf("loading connector %q: %w", name, err)
		}
		engines[name] = connector.New(def, ref.Config)
	}

	g.mu.Lock()
	g.cfg = cfg
	g.engines = engines
	g.mu.Unlock()

	return nil
}

// Start begins serving HTTP and optionally watches for config changes.
func (g *Gateway) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", g.handleRequest)

	port := g.cfg.Server.Port
	if port == 0 {
		port = 8080
	}

	g.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	if g.cfg.Server.HotReload {
		go g.watchConfig()
	}

	log.Printf("Braids gateway listening on :%d", port)
	return g.server.ListenAndServe()
}

// Shutdown gracefully stops the gateway.
func (g *Gateway) Shutdown(ctx context.Context) error {
	return g.server.Shutdown(ctx)
}

func (g *Gateway) watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Warning: could not start file watcher: %v", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(g.configPath); err != nil {
		log.Printf("Warning: could not watch %s: %v", g.configPath, err)
		return
	}

	log.Printf("Watching %s for changes", g.configPath)

	var debounce *time.Timer
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(500*time.Millisecond, func() {
					log.Println("Config changed, reloading...")
					if err := g.loadConfig(); err != nil {
						log.Printf("Reload failed: %v", err)
					} else {
						log.Println("Config reloaded successfully")
					}
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
