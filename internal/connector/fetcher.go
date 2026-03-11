package connector

// Fetcher retrieves records from an upstream data source.
// Each protocol (HTTP, GraphQL, SOAP, database, gRPC) implements this interface.
type Fetcher interface {
	Fetch(resource string, params map[string]any, pathParams map[string]string, headers ...map[string]string) ([]Record, error)
}

// Compile-time interface checks
var _ Fetcher = (*HTTPFetcher)(nil)
var _ Fetcher = (*GraphQLFetcher)(nil)
var _ Fetcher = (*ExtensionFetcher)(nil)
