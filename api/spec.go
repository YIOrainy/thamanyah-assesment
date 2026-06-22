// Package api holds the embedded OpenAPI specification served at /openapi.yaml.
package api

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
