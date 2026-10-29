package api

import "embed"

// SpecFile is the bundled contract in Files: openapi.yaml and paths/*.yaml
// are the source, `go generate ./api` bundles them.
const SpecFile = "openapi.bundled.yaml"

// Files holds the bundled contract (SpecFile) and the error-code registry
// (errors.yaml), so the binary serves exactly what CI checked.
//
//go:embed openapi.bundled.yaml errors.yaml
var Files embed.FS
