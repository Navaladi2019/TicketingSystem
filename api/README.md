# api/

The OpenAPI 3.1 contract for `/api/v1` and the error-code registry.

## Layout

- `openapi.yaml`: the root. Info, servers, security and the components shared
  by several groups (`Problem`, `FieldError`, `responses.Problem`). Each entry
  under `paths` is only a `$ref` to a group file.
- `paths/<group>.yaml`: one file per path group, the group being the first path
  segment after `/api/v1` (`me.yaml`, `operator.yaml`). A group file holds its
  `paths` and the `components` only that group uses. Refs to a shared component
  are written `../openapi.yaml#/components/...`.
- `openapi.bundled.yaml`: **generated**, never edited or hand-merged. It is what
  the server embeds (`embed.go`, served at `/api/v1/openapi.yaml`, page at
  `/api/v1/docs`) and what oapi-codegen reads.
- `api.gen.go`: **generated** from the bundle by oapi-codegen v2.8.0
  (`std-http-server`, `strict-server`, `models`). It stays one file:
  `include-tags` would split the server and types without a clean seam.
- `errors.yaml`: every `code` an error response may carry. Every error response
  is `application/problem+json` (RFC 9457). Add a code here and a constant in
  `internal/platform/problem`; a test fails if the two differ.

## Adding an endpoint

1. Put the operation in `paths/<group>.yaml` (create the file for a new group)
   and its group-only schemas under that file's `components`.
2. Add one line to `openapi.yaml` under `paths`:
   `/api/v1/<group>/...: {$ref: "paths/<group>.yaml#/paths/~1api~1v1~1<group>~1..."}`
   (`/` is `~1` in a JSON Pointer).
3. Run `task generate` (or `make generate`, or `go generate ./api`) and commit
   `openapi.bundled.yaml` and `api.gen.go` with the source.

Two stories that add endpoints in different groups touch different files. If
git conflicts on a generated file, resolve the source files, then regenerate;
never pick a side (`merge=ours` is not used).

The bundler (`tools/openapibundle`) fails on a path in the wrong group file, a
component name defined twice, a group file the root does not reference, or a
bundle kin-openapi rejects.

## Gates

- `make check-generate` (in `ci-lint`) runs `scripts/ci/openapi-generated.sh`: regenerate
  and fail on any `git diff` under `api/`, so a source edit without
  regenerating fails.
- Additive only within v1: `task check-openapi` diffs `openapi.bundled.yaml`
  against `origin/main` and fails on a removed or narrowed field.
