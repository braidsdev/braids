package connector

import (
	"fmt"

	"github.com/braidsdev/braids/internal/config"
	"github.com/braidsdev/braids/internal/extension"
)

// ExtensionFetcher bridges the Fetcher interface to an out-of-process extension.
type ExtensionFetcher struct {
	def            *config.ConnectorDef
	cfg            map[string]string
	manager        *extension.Manager
	connDefPayload *extension.ConnDefPayload
}

// NewExtensionFetcher creates an ExtensionFetcher for the given connector definition.
func NewExtensionFetcher(def *config.ConnectorDef, cfg map[string]string, mgr *extension.Manager) *ExtensionFetcher {
	// Build the wire-format ConnDefPayload once at construction time
	resources := make(map[string]extension.ResourcePayload, len(def.Resources))
	for name, res := range def.Resources {
		resources[name] = extension.ResourcePayload{
			Path:       res.Path,
			Method:     res.Method,
			DataField:  res.DataField,
			Query:      res.Query,
			Collection: res.Collection,
			Filter:     res.Filter,
			Service:    res.Service,
		}
	}

	return &ExtensionFetcher{
		def:     def,
		cfg:     cfg,
		manager: mgr,
		connDefPayload: &extension.ConnDefPayload{
			Name:     def.Name,
			Protocol: def.Protocol,
			BaseURL:  def.BaseURL,
			Auth: extension.ConnDefAuth{
				Type:          def.Auth.Type,
				TokenField:    def.Auth.TokenField,
				UsernameField: def.Auth.UsernameField,
				PasswordField: def.Auth.PasswordField,
			},
			Resources: resources,
		},
	}
}

// Fetch sends a fetch request to the extension process and returns records.
func (f *ExtensionFetcher) Fetch(resource string, params map[string]any, pathParams map[string]string, headers ...map[string]string) ([]Record, error) {
	proc, err := f.manager.GetOrStart(f.def.Protocol)
	if err != nil {
		return nil, fmt.Errorf("starting extension %q: %w", f.def.Protocol, err)
	}

	req := extension.Request{
		Action:     extension.ActionFetch,
		Resource:   resource,
		Params:     params,
		PathParams: pathParams,
		Config:     f.cfg,
		ConnDef:    f.connDefPayload,
	}
	if len(headers) > 0 {
		req.Headers = headers[0]
	}

	resp, err := proc.Send(req)
	if err != nil {
		return nil, fmt.Errorf("extension %q fetch error: %w", f.def.Protocol, err)
	}
	if !resp.OK {
		return nil, fmt.Errorf("extension %q returned error: %s", f.def.Protocol, resp.Error)
	}

	records := make([]Record, len(resp.Records))
	for i, r := range resp.Records {
		records[i] = Record(r)
	}
	return records, nil
}
