# api/

The OpenAPI 3.1 contract for `/api/v1` and the error-code registry.

- `openapi.yaml`: the contract. `api.gen.go` is generated from it by
  oapi-codegen v2.8.0 (`std-http-server`, `strict-server`, `models`); run
  `task generate` (or `make generate`) after editing it and commit the result.
  The server serves it at `/api/v1/openapi.yaml` and a page at `/api/v1/docs`.
- `errors.yaml`: every `code` an error response may carry. Every error response
  is `application/problem+json` (RFC 9457). Add a code here and a constant in
  `internal/platform/problem`; a test fails if the two differ.
- Additive only within v1: `task check-openapi` diffs `openapi.yaml` against
  `origin/main` and fails on a removed or narrowed field.
