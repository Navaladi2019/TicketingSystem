---
type: spike
title: Technical Architect spike: apidiff and oapi-codegen on the RFC-005 contract edit (round 1)
issue: TICK-266
source: TICK-266 comment 01a17cc4-6419-7874-8f96-d00ba6f39cac (Technical Architect)
---

# Technical Architect spike: apidiff and oapi-codegen on the RFC-005 contract edit (round 1)

> Archived verbatim from the Technical Architect's critique comment. Spike: setup, raw output, rerun.

Environment: go1.26.10 windows/amd64, repo `main` @ `da31944`, kin-openapi and oapi-codegen v2.8.0 as pinned in `go.mod`. Hypothesis: the RFC's `Me` edit passes `openapi-compat` (pass = exit 0) and leaves `go test ./internal/platform/apidiff/` green (fail = any FAIL).

The edit: append `group_paths` to `Me.required`, and add the property after `idp_alias`:
```yaml
        group_paths:
          type: array
          items:
            type: object
            required: [id, path]
            properties:
              id:   {type: string, format: uuid}
              path: {type: string, pattern: "^/"}
```
Run:
```
cp api/openapi.yaml /tmp/base.yaml   # before the edit
# apply the edit to api/openapi.yaml
go run ./cmd/openapi-compat /tmp/base.yaml api/openapi.yaml; echo $?
go test ./internal/platform/apidiff/
(cd api && go generate ./...) && git diff api/api.gen.go
git checkout -- api/
```
Raw output:
```
openapi-compat exit=0
--- FAIL: TestBreakingChanges (0.00s)
    --- FAIL: TestBreakingChanges/required_dropped (0.00s)
        apidiff_test.go:68: test edit did not change the spec
FAIL	github.com/Navaladi2019/TicketingSystem/internal/platform/apidiff	5.749s

+	GroupPaths []struct {
+		Id   openapi_types.UUID `json:"id"`
+		Path string             `json:"path"`
+	} `json:"group_paths"`
```
Result: the CLI half **passes**. The test-suite half **fails** (O2), and the fix is a two-line fixture update.
