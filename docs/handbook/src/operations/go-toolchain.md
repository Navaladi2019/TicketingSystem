---
title: "Go toolchain and setup"
section: operations
status: in-progress
sources: [go.mod, temporal-server/go.mod, tools/ci/checkpins/samego.go, tools/ci/checkpins/adversarial_toolchain_test.go, tools/navalint/go.mod, tools/workflowcheck/go.mod, docs/pins.md, .github/workflows/ci.yml, .golangci.yml, deploy/compose/nava-core/Dockerfile, temporal-server/Dockerfile, Taskfile.yml, make/lint.mk, docs/delivery/decisions/2026-11-15-tick-961-go-1-27-upgrade.md]
stories: [GO127]
packages: [tools/navalint, tools/workflowcheck]
---

The Go version every module builds with, what a developer installs to build and test without the stack, and where the version is pinned. Go 1.27.4 is on `main` (TICK-961).

## The version

| Module | `go` line | `toolchain` | Image that builds it |
|---|---|---|---|
| Platform (`go.mod`, root) | `go 1.27.0` | `go1.27.4` | `golang:1.27.4-alpine` (every `deploy/compose/*/Dockerfile`) |
| `temporal-server` | `go 1.27.0` | `go1.27.4` | `golang:1.27.4-bookworm` |
| `tools/navalint`, `tools/workflowcheck` | `go 1.27.0` (each in its own `go.mod`) | `go1.27.4` | not built into an image; `task lint` runs them. A toolchain bump edits these two `go.mod` files too |

`go 1.27.0` is the floor: an older Go refuses to build the module. The `toolchain` line makes Go 1.21 or later download and use go1.27.4 when the installed Go is older (`GOTOOLCHAIN=auto`, the default). The reason for every raise is in `docs/pins.md` ("Toolchain and CI tools").

CI never names a version. Every Go job reads it from `go-version-file: go.mod` (or `temporal-server/go.mod`), and the pinned-tools cache key ends in `-go<version>`, so a toolchain bump rebuilds `golangci-lint` and `govulncheck` with the new Go. `.golangci.yml` sets `run.go: "1.27"`.

## Scanned Go is shipped Go

The `toolchain` line of a module's `go.mod` must equal the version in every `FROM golang:` of that module: `temporal-server/Dockerfile` for `temporal-server`, the seven `deploy/compose/nava-*/Dockerfile` for the root module (ADR-001-E4 contract rule 8). govulncheck judges the standard library of the `toolchain` Go; the image ships the Go of its `FROM` line (the `golang` images set `GOTOOLCHAIN=local`). `make check-pins` fails when they differ, and `TestScannedToolchainIsTheShippedToolchain` (`tools/ci/checkpins`, V-5) is red on a lowered Dockerfile. A toolchain bump therefore edits `go.mod` and every Dockerfile of the module in one pull request.

No `go.work` (or `go.work.sum`) is tracked: one changes the scanned versions without changing `go.mod` or `go.sum`, and `make check-pins` fails on it (`TestNoTrackedGoWork`). An untracked `go.work` on a laptop does not count, so the vulnerability recipes (`make vulncheck`, `make vulncheck-temporal-server`, `task vulncheck`) run with `GOWORK=off`. In CI the scan steps set `GOWORK=off` in their `env:` and run govulncheck directly ([ADR-001-A7](../adr/adr-001-a7.md)).

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
- **`encoding/json` v1 now runs on the v2 engine.** Output is the same except for one case the audit export hit: a `json.Encoder` now escapes U+2028 and U+2029 inside the bytes a `MarshalJSON` method returns (Go 1.26 left them raw). RFC 8785 keeps them raw, so a 1.27-built exporter wrote envelopes that `nava audit verify` rejected as non-canonical. The export now writes the canonical envelope bytes directly, so exported envelopes stay RFC 8785 canonical and pass `nava audit verify`; the hashes were never affected (TICK-970). See [Audit chain and offline verify](../security/10-audit-chain.md).
- **`encoding/json/v2` is used at the request boundary**: a body with a duplicate member name or invalid UTF-8 is refused with a 422. See [API errors and strict JSON bodies](../system/api-errors.md). Hashed and stored JSON is still written with v1 (recorded deferral, below).
- **`testing/synctest`**: timing-dependent unit tests run in a synctest bubble with a fake clock instead of sleeping (TICK-968).
- **Ids**: every persisted id is a UUIDv7 and a lint rule stops new v4 ids. See [Id generation](../system/id-generation.md).

## Deferred

| What | Why | Trigger to revisit |
|---|---|---|
| Move from `github.com/google/uuid` to the stdlib `uuid` package (TICK-966) | 340 files; overlaps every open story; `google/uuid` already mints v7 | No open story branch changes Go files under `internal/` or `cmd/`, or `google/uuid` is deprecated or stops building |
| Hashed and stored JSON (idempotency request hash, record `data`, outbox payloads, export framing) to `encoding/json/v2` | v2's default marshalling differs (map order, nil slices, `omitempty`), so a move risks changing hashed bytes | The next change to one of those sites, or v1 leaving the standard library |

Both are in `docs/delivery/decisions/2026-11-15-tick-961-go-1-27-upgrade.md` and on the Go 1.27 story.
