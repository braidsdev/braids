package config

import (
	"testing"
)

func TestValidateValid(t *testing.T) {
	cfg := &Config{
		Version: "1",
		Connectors: map[string]ConnectorRef{
			"stripe": {Type: "stripe", Config: map[string]string{"api_key": "sk_test"}},
		},
		Schemas: map[string]Schema{
			"customer": {
				MergeOn: "email",
				Fields:  map[string]Field{"id": {Type: "string"}, "email": {Type: "string"}},
			},
		},
		Endpoints: map[string]Endpoint{
			"/customers": {
				Schema: "customer",
				Sources: []Source{
					{Connector: "stripe", Resource: "customers", Mapping: map[string]string{"id": "id", "email": "email"}},
				},
			},
		},
		Server: Server{Port: 8080},
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("expected valid config, got: %v", err)
	}
}

func TestValidateMissingVersion(t *testing.T) {
	cfg := &Config{
		Connectors: map[string]ConnectorRef{"s": {Type: "stripe"}},
		Schemas:    map[string]Schema{"c": {Fields: map[string]Field{"id": {Type: "string"}}}},
		Endpoints:  map[string]Endpoint{"/c": {Schema: "c", Sources: []Source{{Connector: "s", Resource: "r", Mapping: map[string]string{"id": "id"}}}}},
	}
	if err := Validate(cfg); err == nil {
		t.Error("expected error for missing version")
	}
}

func TestValidateUnknownSchema(t *testing.T) {
	cfg := &Config{
		Version:    "1",
		Connectors: map[string]ConnectorRef{"s": {Type: "stripe"}},
		Schemas:    map[string]Schema{"customer": {Fields: map[string]Field{"id": {Type: "string"}}}},
		Endpoints:  map[string]Endpoint{"/c": {Schema: "nonexistent", Sources: []Source{{Connector: "s", Resource: "r", Mapping: map[string]string{"id": "id"}}}}},
	}
	if err := Validate(cfg); err == nil {
		t.Error("expected error for unknown schema reference")
	}
}

func TestValidatePathConnectorMissingPath(t *testing.T) {
	cfg := &Config{
		Version: "1",
		Connectors: map[string]ConnectorRef{
			"custom": {Type: "path"},
		},
		Schemas:   map[string]Schema{"c": {Fields: map[string]Field{"id": {Type: "string"}}}},
		Endpoints: map[string]Endpoint{"/c": {Schema: "c", Sources: []Source{{Connector: "custom", Resource: "r", Mapping: map[string]string{"id": "id"}}}}},
	}
	if err := Validate(cfg); err == nil {
		t.Error("expected error for path connector without path")
	}
}

func TestValidatePathConnectorWithPath(t *testing.T) {
	cfg := &Config{
		Version: "1",
		Connectors: map[string]ConnectorRef{
			"custom": {Type: "path", Path: "./my-connectors/custom"},
		},
		Schemas:   map[string]Schema{"c": {Fields: map[string]Field{"id": {Type: "string"}}}},
		Endpoints: map[string]Endpoint{"/c": {Schema: "c", Sources: []Source{{Connector: "custom", Resource: "r", Mapping: map[string]string{"id": "id"}}}}},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("expected valid config, got: %v", err)
	}
}

func TestValidateUnknownConnector(t *testing.T) {
	cfg := &Config{
		Version:    "1",
		Connectors: map[string]ConnectorRef{"stripe": {Type: "stripe"}},
		Schemas:    map[string]Schema{"c": {Fields: map[string]Field{"id": {Type: "string"}}}},
		Endpoints:  map[string]Endpoint{"/c": {Schema: "c", Sources: []Source{{Connector: "unknown", Resource: "r", Mapping: map[string]string{"id": "id"}}}}},
	}
	if err := Validate(cfg); err == nil {
		t.Error("expected error for unknown connector reference")
	}
}
