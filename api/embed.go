package api

import "embed"

// Files holds the contract (openapi.yaml) and the error-code registry
// (errors.yaml), so the binary serves exactly what CI checked.
//
//go:embed openapi.yaml errors.yaml
var Files embed.FS
