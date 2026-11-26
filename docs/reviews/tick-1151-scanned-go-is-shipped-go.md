# Review: TICK-1151, scanned Go equals shipped Go; no tracked go.work (V-5)

PR #913 at `71ced7cb`. Reviewer: Quality, Testing & CI Advisor.
Read first: `docs/adr/ADR-001/E4/vulncheck-contract.md` rules 7 and 8, test V-5.

**VERDICT: APPROVED.** No Bug, no Spec drift. One non-blocking gap, below.

## Evidence

- CI on `71ced7cb` with `ci:run`: `lint` (runs `make check-pins`), `unit`, `integration`, `coverage`, `changes`, `commit-check-budget`, `runner`, `story-branches` pass. The others skip by path filter.
- Local: `go test -race -count=1 ./tools/ci/checkpins/` passes (95 s on Windows).

## Acceptance criteria

| # | Criterion | Test | Result |
|---|---|---|---|
| 1 | Green on the repo; red with `nava-relay` or `temporal-server` lowered | `TestScannedToolchainIsTheShippedToolchain/{repository,nava-relay lowered,temporal-server lowered}` | pass |
| 2 | Red on an extra unmapped `Dockerfile*` with `FROM golang:` | `.../unmapped {Dockerfile.build, cmd/tool/Dockerfile, Dockerfile.ci}` | pass |
| 3 | Red on a tracked `go.work`; green on the repo | `TestNoTrackedGoWork` (root, `go.work.sum`, nested, untracked ignored) | pass |

The tests also cover a raised toolchain, a missing toolchain line, a later build stage, `--platform` and a registry prefix, a `golang:${ARG}` tag, an untagged `golang`, a dropped mapped Dockerfile, and exit 2 outside a git work tree. Each one is a regression that could really happen.

## Findings (most severe first)

1. **Breaks under: the Go image is named through an ARG or a non-`golang` image** (`samego.go`, `fromGolangRe`). The rule only sees lines that literally say `golang`. If `deploy/compose/nava-relay/Dockerfile` changes to `ARG GO_IMAGE=golang:1.27.3-alpine` plus `FROM ${GO_IMAGE} AS build`, or to `FROM cgr.dev/chainguard/go:...`, the check passes, but the image ships a Go that was never scanned. The same holds for a new, unmapped Dockerfile. The contract text ("every `FROM golang:`") is met, but its intent is not. Suggested fix (follow-up, not this PR): a mapped Dockerfile with no literal `FROM golang:<version>` stage fails ("mapped Dockerfile has no golang build stage"), and a `FROM ${...}` whose ARG default contains `golang` gets the same check. Missing test: a copy with `nava-relay` switched to `FROM ${GO_IMAGE}`. An attacker would need a reviewed Dockerfile change, so this protects against accidents, not a deliberate edit (the same stance as RS3).
2. **Preference:** a file that is tracked but deleted from the work tree makes `check-pins` exit 2 with a bare ENOENT. That is right (it cannot check), but naming the file and "tracked but missing" would be clearer.
3. **Preference:** each subtest `git init`s a full copy, so the package takes about 95 s under `-race` on Windows. That is fine for now. If it grows, share a single read-only copy for the green cases.

## Handbook

If a handbook page describes `check-pins`, it needs a line about rules 7 and 8 and the `-same-go` flag (the PR description says so).
