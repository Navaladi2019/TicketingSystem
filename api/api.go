// Package api holds the OpenAPI 3.1 contract for /api/v1, the error-code
// registry, and the server code generated from the contract.
package api

//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
