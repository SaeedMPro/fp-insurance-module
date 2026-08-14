// Package api holds the machine-readable OpenAPI contract (openapi.yaml).
// The YAML is embedded so the running binary can serve Swagger UI without
// shipping a loose file next to the executable.
package api

import _ "embed"

// OpenAPIYAML is the source-of-truth contract (ADR-0002).
//
//go:embed openapi.yaml
var OpenAPIYAML []byte
