// Package api holds the OpenAPI 3.1 contract for /api/v1, the error-code
// registry, and the server code generated from the contract.
package api

//go:generate go run ../tools/openapibundle -root openapi.yaml -out openapi.bundled.yaml
//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.bundled.yaml
