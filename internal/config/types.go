package config

// braids.yaml types

type Config struct {
	Version    string                    `yaml:"version"`
	Connectors map[string]ConnectorRef   `yaml:"connectors"`
	Schemas    map[string]Schema         `yaml:"schemas"`
	Endpoints  map[string]Endpoint       `yaml:"endpoints"`
	Server     Server                    `yaml:"server"`
}

type ConnectorRef struct {
	Type      string                      `yaml:"type"`
	Path      string                      `yaml:"path"`
	Config    map[string]string           `yaml:"config"`
	Resources map[string]ResourceOverride `yaml:"resources"`
}

// ResourceOverride allows instance-level resource definitions in braids.yaml.
// Used for protocols where queries/collections are user-specific (databases, GraphQL).
type ResourceOverride struct {
	Query      string `yaml:"query"`       // SQL query or GraphQL query
	Collection string `yaml:"collection"`  // MongoDB collection
	Filter     string `yaml:"filter"`      // MongoDB filter (JSON)
	Method     string `yaml:"method"`      // HTTP method or gRPC method
	Service    string `yaml:"service"`     // gRPC service name
	DataField  string `yaml:"data_field"`
	Path       string `yaml:"path"`        // URL path override
}

type Schema struct {
	MergeOn            string           `yaml:"merge_on"`
	ConflictResolution string           `yaml:"conflict_resolution"`
	Fields             map[string]Field `yaml:"fields"`
}

type Field struct {
	Type string `yaml:"type"`
}

type Endpoint struct {
	Schema  string   `yaml:"schema"`
	Sources []Source `yaml:"sources"`
}

type Source struct {
	Connector string            `yaml:"connector"`
	Resource  string            `yaml:"resource"`
	Params    map[string]any    `yaml:"params"`
	Headers   map[string]string `yaml:"headers"`
	Mapping   map[string]string `yaml:"mapping"`
}

type Server struct {
	Port      int  `yaml:"port"`
	HotReload bool `yaml:"hot_reload"`
}

// connector.yaml types

type ConnectorDef struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Category    string                 `yaml:"category"`
	Tags        []string               `yaml:"tags"`
	Version     string                 `yaml:"version"`
	Protocol    string                 `yaml:"protocol"` // http (default), graphql, soap, postgres, mysql, mongodb, grpc
	BaseURL     string                 `yaml:"base_url"`
	OpenAPISpec string                 `yaml:"openapi_spec"`
	OpenAPIURL  string                 `yaml:"openapi_url"`
	Auth        AuthDef                `yaml:"auth"`
	Pagination  PaginationDef          `yaml:"pagination"`
	Resources   map[string]ResourceDef `yaml:"resources"`
}

type AuthDef struct {
	Type          string            `yaml:"type"`           // bearer, header, basic, query_param, none
	TokenField    string            `yaml:"token_field"`
	HeaderName    string            `yaml:"header_name"`
	TokenPrefix   string            `yaml:"token_prefix"`   // e.g. "Bot " for Discord
	UsernameField string            `yaml:"username_field"` // for basic auth
	PasswordField string            `yaml:"password_field"` // for basic auth
	ParamName     string            `yaml:"param_name"`     // for query_param auth
	ExtraHeaders  map[string]string `yaml:"extra_headers"`  // field_name -> header_name mapping
}

type PaginationDef struct {
	Type         string `yaml:"type"` // cursor, link_header, offset, page, none
	CursorParam  string `yaml:"cursor_param"`
	CursorField  string `yaml:"cursor_field"`
	HasMoreField string `yaml:"has_more_field"`
	DataField    string `yaml:"data_field"`
	LimitParam   string `yaml:"limit_param"`    // for offset/page pagination
	LimitDefault int    `yaml:"limit_default"`  // default page size
	OffsetParam  string `yaml:"offset_param"`   // for offset pagination
	TotalField   string `yaml:"total_field"`    // total count field in response
	PageParam    string `yaml:"page_param"`     // for page pagination
	ResultsField string `yaml:"results_field"`  // some APIs nest results differently per-page
}

type ResourceDef struct {
	Path       string `yaml:"path"`
	Method     string `yaml:"method"`
	DataField  string `yaml:"data_field"`
	Query      string `yaml:"query"`       // SQL query or GraphQL query
	Collection string `yaml:"collection"`  // MongoDB collection
	Filter     string `yaml:"filter"`      // MongoDB filter (JSON)
	Service    string `yaml:"service"`     // gRPC service name
}
