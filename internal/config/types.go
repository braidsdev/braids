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
	Type   string            `yaml:"type"`
	Path   string            `yaml:"path"`
	Config map[string]string `yaml:"config"`
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
	Version     string                 `yaml:"version"`
	BaseURL     string                 `yaml:"base_url"`
	OpenAPISpec string                 `yaml:"openapi_spec"`
	OpenAPIURL  string                 `yaml:"openapi_url"`
	Auth        AuthDef                `yaml:"auth"`
	Pagination  PaginationDef          `yaml:"pagination"`
	Resources   map[string]ResourceDef `yaml:"resources"`
}

type AuthDef struct {
	Type       string `yaml:"type"`
	TokenField string `yaml:"token_field"`
	HeaderName string `yaml:"header_name"`
}

type PaginationDef struct {
	Type         string `yaml:"type"`
	CursorParam  string `yaml:"cursor_param"`
	CursorField  string `yaml:"cursor_field"`
	HasMoreField string `yaml:"has_more_field"`
	DataField    string `yaml:"data_field"`
}

type ResourceDef struct {
	Path      string `yaml:"path"`
	Method    string `yaml:"method"`
	DataField string `yaml:"data_field"`
}
