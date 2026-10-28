---
type: rfc
id: RFC-006
title: ADR-002 erratum E1, a line-scoped nolint replaces the api.go forbidigo exclusion
status: Draft (rev 0)
date: 2026-10-28
issue: TICK-272
target: ADR-002-E1 (amends ADR-002 §3 "Lint rule" row, §5 AC8 row, §7 R-b)
source: TICK-268 / PR #204, Security & Multi-Tenancy Reviewer finding 2 (Low)
author: Solution Architect
---

# RFC-006: ADR-002-E1, line-scoped `//nolint:forbidigo` instead of the `api.go` path exclusion

> **Invariants missing:** the Ticket Management System project description is empty, so the invariant check below is incomplete.

## Decision

1. Delete the `path: ^internal/server/api\.go$ / linters: [forbidigo]` exclusion from `.golangci.yml`.
2. Put `//nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller` on the single call site, `internal/server/api.go` (`raw, err := h.paths.GroupPaths(...)` in `GetMe`).
3. Enable `nolintlint` with `require-specific: true`, `require-explanation: true` and `allow-unused: false`.
4. Add an allow-list count gate to `make lint`: the repo may contain exactly **one** `//nolint:forbidigo` outside `_test.go` and `docs/`, and it must be in `internal/server/api.go`. A second one fails CI.

Effect: the raw `net/http` rule and the `otelpgx` rule apply to `api.go` again. The `GroupPaths` waiver covers one line, not one file.

## The problem

ADR-002 §3 turned `forbidigo` off for the whole of `api.go` so that `GetMe` could call `GroupPaths`. That also turned off the other two `forbidigo` rules for that file: the raw HTTP client rule and the `otelpgx.WithIncludeQueryParameters` rule. It is Low severity and not exploitable today, but `api.go` is where new handlers land.

## Options

| Option | Scope of waiver | Catches a 2nd `GroupPaths` caller in `api.go` | Cost | Verdict |
|---|---|---|---|---|
| A. Keep the file exclusion (status quo) | All `forbidigo` rules, whole file | No (R-b) | 0 | Rejected: the finding |
| B. Narrow exclusion: `path` + `text: GroupPaths` | `GroupPaths` rule only, whole file | No | 1 YAML line | Rejected: still per file, so R-b remains |
| C. Move `GetMe` to its own file and exclude that file | All rules, one small file | Only while the file holds one handler | file split | Rejected: same hole, smaller; relies on nobody adding to the file |
| **D. Line `//nolint:forbidigo` + `nolintlint` + count gate** | `forbidigo`, one line | **Yes** (lint error, or count gate) | ~10 YAML/shell lines | **Chosen** |
| E. Custom go/analysis check in `navalint` that allows only `(*handlers).GetMe` | Exact function | Yes | new analyzer + fixture | Rejected: right shape, too much for one read. Revisit if a 2nd waiver ever appears |

## Does the setup stop unused or unscoped directives? (spike, golangci-lint 2.14.0, S5 branch)

`nolintlint` is **not** enabled today (it is not in `default: standard`), so without step 3 a stale or bare directive would pass silently. With step 3:

| # | Case | Result |
|---|---|---|
| 1 | Exclusion removed, no directive | `forbidigo` at `api.go:94` (rule fires) |
| 2/3 | Exclusion removed + line directive (± `nolintlint`) | 0 issues |
| 4 | A stale `//nolint:forbidigo` on a clean line | `nolintlint`: directive is unused |
| 5 | Bare `//nolint` | `nolintlint`: should mention a specific linter |
| 10 | `//nolint:all // reason` | `nolintlint`: should mention a specific linter |
| 6 | `//nolint:forbidigo` with no `// reason` | `nolintlint`: should provide explanation |
| 9 | Directive added but the file exclusion left in | `nolintlint`: directive is unused, so the half-done change fails |
| 7 | Second `GroupPaths` caller in `api.go`, no directive | `forbidigo` at the new line |
| 8 | Second caller with the directive **copied** | **0 issues**: the gap step 4 closes |

Full output: `spike-forbidigo-nolint.md` (attached).

## How a second `GroupPaths` caller is caught

- Outside `api.go`: `forbidigo` type-qualified rules, unchanged (ADR-002 AC8).
- Inside `api.go`, no directive: `forbidigo` (case 7). This is new; R-b is closed.
- Inside `api.go`, directive copied: the count gate (step 4) fails because there are two `//nolint:forbidigo`. Adding a legitimate one means editing the gate's allow-list, which is a reviewed CI change.
- In a test file: `_test.go` stays excluded from `forbidigo`. That is unchanged and is not production reach.

## Acceptance criteria (one Delivery task, created by the CTO after sign-off)

1. `.golangci.yml`: the `api.go` exclusion is gone; `nolintlint` is enabled with the three settings above.
2. `api.go`: the directive is on the call-site line, word for word as above.
3. The gate script (e.g. `scripts/ci/nolint-allowlist.sh`, run from `make lint`) fails on a second `//nolint:forbidigo` or on one outside `api.go`. Show red to green in the PR.
4. `make lint` and `make lint-fixture` pass. The fixture count stays 6 `forbidigo` findings.
5. Before merging, run `make lint` on the whole repo with `nolintlint` on. Existing directives (`gosec`, `revive`, `gochecknoglobals`, `errorlint`; about 10) all have explanations, but any that are now unused must be removed in the same PR. **I did not get this data:** the whole-repo run timed out locally at 500 s.

## Residual risk

- R-b is closed. A new row, R-e: the waiver is per line, not per function. Someone could put a second call on the *same* line, or move the line to another handler. Review of `api.go` is the control, as before. It is much narrower than a whole file.

## Invariants

**The project description is empty, so the eight invariants could not be loaded.** The change only touches lint config, a comment and a CI script. It touches no runtime path, tenant scoping, schema or workflow, so **I expect no invariant to be affected, but the invariant check is incomplete** until the Technical Architect confirms it.

## Sequencing

The erratum PR (`docs/adr/ADR-002-E1-*.md`, `.html` and this archived RFC under `docs/adr/ADR-002/rfc/`) branches from `main` after PR #221 merges. PR #221 was still OPEN on 2026-10-28.
