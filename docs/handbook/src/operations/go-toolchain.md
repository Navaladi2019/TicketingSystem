---
title: "Go toolchain and setup"
section: operations
status: in-progress
sources: [go.mod, temporal-server/go.mod, tools/navalint/go.mod, tools/workflowcheck/go.mod, docs/pins.md, .github/workflows/ci.yml, .golangci.yml, deploy/compose/nava-core/Dockerfile, temporal-server/Dockerfile, Taskfile.yml, make/lint.mk, docs/delivery/decisions/2026-11-15-tick-961-go-1-27-upgrade.md]
stories: [GO127]
packages: [tools/navalint, tools/workflowcheck]
---

The Go version every module builds with, what a developer installs to build and test without the stack, and where the version is pinned. Status: on the story branch `TICK-961/go-1-27-upgrade` until the Go 1.27 story merges to `main`; `main` is still on Go 1.26.11.

## The version

| Module | `go` line | `toolchain` | Image that builds it |
|---|---|---|---|
| Platform (`go.mod`, root) | `go 1.27.0` | `go1.27.4` | `golang:1.27.4-alpine` (every `deploy/compose/*/Dockerfile`) |
| `temporal-server` | `go 1.27.0` | `go1.27.4` | `golang:1.27.4-bookworm` |
| `tools/navalint`, `tools/workflowcheck` | follow the root floor | | not built into an image; `task lint` runs them |

`go 1.27.0` is the floor: an older Go refuses to build the module. The `toolchain` line makes Go 1.21 or later download and use go1.27.4 when the installed Go is older (`GOTOOLCHAIN=auto`, the default). The reason for every raise is in `docs/pins.md` ("Toolchain and CI tools").

CI never names a version. Every Go job reads it from `go-version-file: go.mod` (or `temporal-server/go.mod`), and the pinned-tools cache key ends in `-go<version>`, so a toolchain bump rebuilds `golangci-lint` and `govulncheck` with the new Go. `.golangci.yml` sets `run.go: "1.27"`.

## Set up a machine

| Tool | Version | Used by |
|---|---|---|
| Go | 1.27.4 (or any Go ≥ 1.21 with `GOTOOLCHAIN=auto`) | everything |
| go-task | v3 | `task build`, `task test`, `task lint` |
| golangci-lint | v2.14.0 (`docs/pins.md`) | `task lint` |
| A C compiler (cgo) | any | `task test` runs with `-race`; use `task test:short` on a host without cgo |
| Docker | | only `task test:integration` and the [local stack](local-stack.md) |
| Python 3 | standard library only | `task handbook` |

```sh
go version                 # go1.27.4, or older with the toolchain downloaded on first use
task build                 # bin/nava-core and bin/nava
task test                  # every test with the race detector, navalint's included
task lint                  # golangci-lint, navalint (uuidv7 among them), workflowcheck, adrgate, compat
```

Never run `make ci*` or `make stack-*` to verify a change: open a PR and read the GitHub `ci` run (see `docs/quality/ci-pipeline.md`).

## What changed with Go 1.27

- **`go fix` modernizers** ran once over the code (TICK-962): mechanical rewrites across 125 files, mostly `strings.SplitSeq`, `maps.Copy`, `slices.Contains` and `new(value)` for pointers to values. No hand edits, no behaviour change.
- **`encoding/json` v1 now runs on the v2 engine.** Output is the same except for one case the audit export hit: an `Encoder` with `SetEscapeHTML(false)` now escapes U+2028 and U+2029. The export writes canonical envelope bytes directly so the chain hash does not move (TICK-970); see [Audit chain and offline verify](../security/10-audit-chain.md).
- **`encoding/json/v2` is used at the request boundary**: a body with a duplicate member name or invalid UTF-8 is refused with a 422. See [API errors and strict JSON bodies](../system/api-errors.md). Hashed and stored JSON is still written with v1 (recorded deferral, below).
- **`testing/synctest`**: timing-dependent unit tests run in a synctest bubble with a fake clock instead of sleeping (TICK-968).
- **Ids**: every persisted id is a UUIDv7 and a lint rule stops new v4 ids. See [Id generation](../system/id-generation.md).

## Deferred

| What | Why | Trigger to revisit |
|---|---|---|
| Move from `github.com/google/uuid` to the stdlib `uuid` package (TICK-966) | 340 files; overlaps every open story; `google/uuid` already mints v7 | No open story branch changes Go files under `internal/` or `cmd/`, or `google/uuid` is deprecated or stops building |
| Hashed and stored JSON (idempotency request hash, record `data`, outbox payloads, export framing) to `encoding/json/v2` | v2's default marshalling differs (map order, nil slices, `omitempty`), so a move risks changing hashed bytes | The next change to one of those sites, or v1 leaving the standard library |

Both are in `docs/delivery/decisions/2026-11-15-tick-961-go-1-27-upgrade.md` and on the Go 1.27 story.
