// Package extension implements the out-of-process extension protocol.
// Extensions are separate binaries that communicate via JSON lines over stdin/stdout.
// This keeps driver dependencies (e.g. libpq, mongo driver) out of the core binary.
package extension

// IPC action constants.
const (
	ActionPing     = "ping"
	ActionFetch    = "fetch"
	ActionShutdown = "shutdown"
)

// Request is the JSON-line envelope sent to an extension process.
type Request struct {
	Action     string            `json:"action"`
	Resource   string            `json:"resource,omitempty"`
	Params     map[string]any    `json:"params,omitempty"`
	PathParams map[string]string `json:"path_params,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Config     map[string]string `json:"config,omitempty"`
	ConnDef    *ConnDefPayload   `json:"conn_def,omitempty"`
}

// ConnDefPayload is a self-contained subset of config.ConnectorDef,
// kept separate to avoid importing config types into the wire format.
type ConnDefPayload struct {
	Name      string                     `json:"name"`
	Protocol  string                     `json:"protocol"`
	BaseURL   string                     `json:"base_url,omitempty"`
	Auth      ConnDefAuth                `json:"auth,omitempty"`
	Resources map[string]ResourcePayload `json:"resources,omitempty"`
}

// ConnDefAuth mirrors the auth fields an extension needs.
type ConnDefAuth struct {
	Type          string `json:"type,omitempty"`
	TokenField    string `json:"token_field,omitempty"`
	UsernameField string `json:"username_field,omitempty"`
	PasswordField string `json:"password_field,omitempty"`
}

// ResourcePayload mirrors config.ResourceDef for the wire format.
type ResourcePayload struct {
	Path       string `json:"path,omitempty"`
	Method     string `json:"method,omitempty"`
	DataField  string `json:"data_field,omitempty"`
	Query      string `json:"query,omitempty"`
	Collection string `json:"collection,omitempty"`
	Filter     string `json:"filter,omitempty"`
	Service    string `json:"service,omitempty"`
}

// Response is the JSON-line envelope received from an extension process.
type Response struct {
	OK      bool               `json:"ok"`
	Records []map[string]any   `json:"records,omitempty"`
	Error   string             `json:"error,omitempty"`
	Version string             `json:"version,omitempty"`
}
