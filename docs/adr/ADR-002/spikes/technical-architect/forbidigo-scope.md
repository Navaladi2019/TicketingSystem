---
type: spike
title: Technical Architect spike: forbidigo scope for GroupPaths (round 2)
issue: TICK-266
source: TICK-266 comment 01a17cd8-9caa-79a9-a021-ed8f069f40e4 (Technical Architect)
---

# Technical Architect spike: forbidigo scope for GroupPaths (round 2)

> Archived verbatim from the Technical Architect's critique comment. Spike: forbidigo scope (setup, raw output, rerun).

**Hypothesis:** with the repo's `forbidigo` settings, pattern `\.GroupPaths\(` flags a `GroupPaths` call made outside `internal/server/api.go`. **Pass:** at least one issue on each disallowed call. **Fail:** `0 issues`.
**Environment:** golangci-lint 2.14.0 (built with go1.26.10, the version pinned in the `Makefile`), windows/amd64, a stand-alone module.
**Fixture:**
- `internal/identity`: `GroupPathReader` interface.
- `internal/identity/pgstore`: `Store.GroupPaths`.
- `internal/server/api.go`: `GetMe` calls `r.GroupPaths(...)` and returns `Me{GroupPaths: gp}`.
- `internal/other/other.go`: a call through the interface, a call on `*pgstore.Store`, a method value `return r.GroupPaths`, and a field read `len(m.GroupPaths)` on `server.Me`.

Config: `linters.default: none`, `enable: [forbidigo]`, `analyze-types: true`, with exclusions for `^internal/server/api\.go$` and `^internal/identity/pgstore/`.

Raw output:
```
=== A: \.GroupPaths\( (RFC as written)
0 issues.
=== B: \.GroupPaths$
internal\other\other.go:14:11: use of `r.GroupPaths` forbidden ...
internal\other\other.go:20:11: use of `s.GroupPaths` forbidden ...
internal\other\other.go:25:62: use of `r.GroupPaths` forbidden ...
internal\other\other.go:28:46: use of `m.GroupPaths` forbidden ...   <- response field, false positive
4 issues
=== C: type-qualified with pkg
internal\other\other.go:14:11: use of `r.GroupPaths` forbidden ...
internal\other\other.go:20:11: use of `s.GroupPaths` forbidden ...
internal\other\other.go:25:62: use of `r.GroupPaths` forbidden ...
3 issues
=== C without exclusions
... the same 3, plus internal\server\api.go:15:13: use of `r.GroupPaths` forbidden ...
4 issues
```
**Rerun:** build the fixture as described (module `example.com/spk`, `go 1.26.10`; for C, replace the module path in `pkg:`), write each config to `.golangci.yml`, and run `golangci-lint run ./...`. In the real PR, AC8's red/green check is the same test: add a call in a disallowed file, `task lint` must fail, remove it, and lint must pass.
