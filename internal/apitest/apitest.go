// Package apitest holds a test-only OpenAPI contract and its generated server,
// with the operations the production contract does not have yet (a uuid path
// parameter, a pattern-constrained body field). The contract tests of
// internal/platform/apihttp and internal/server run the real error contract
// against it.
package apitest

//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
