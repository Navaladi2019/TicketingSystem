---
type: rfc
id: RFC-006
title: ADR-002 erratum E1, a line-scoped nolint replaces the api.go forbidigo exclusion
status: Revised (rev 1)
date: 2026-10-28
issue: TICK-272
target: ADR-002-E1 (amends ADR-002 §3 "Lint rule" row, §5 AC8 row, §7 R-b)
source: TICK-268 / PR #204, Security & Multi-Tenancy Reviewer finding 2 (Low)
author: Solution Architect
round-1: Technical Architect REVISE (comment 01a18097-33e3); Security CHANGES B1 + N1/N2 (comment 01a18091-07f3); CTO direction (comment 01a18097-7399)
---

# RFC-006 rev 1: ADR-002-E1, line-scoped `//nolint:forbidigo` instead of the `api.go` path exclusion

## Decision

1. Delete the `path: ^internal/server/api\.go$ / linters: [forbidigo]` exclusion from `.golangci.yml`.
2. Put `//nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller` as a trailing comment on the single call line in `GetMe` (`raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID)`).
3. Enable `nolintlint` with `require-specific: true`, `require-explanation: true` and `allow-unused: false`.
4. Add `scripts/ci/nolint-allowlist.sh` and run it from `make lint`. It is an **exact-line gate** with two checks:
   - **Directives.** It matches every spelling of a `forbidigo` waiver that golangci-lint honours, outside `_test.go` and `docs/`. The result must be exactly the one call line above, word for word. This is the Technical Architect's spiked gate.
   - **Config** (Security N1). `.golangci.yml` fails the gate if it contains any of these:
     - a `forbidigo` exclusion rule for a path other than `_test\.go` or `^internal/platform/telemetry/`;
     - a `forbidigo` rule with no `path`;
     - any global `linters.exclusions.paths` entry, which would turn off every linter for that path.

Effect: the raw `net/http` and `otelpgx` rules apply to `api.go` again. The `GroupPaths` waiver covers one line of text, not a file or a function. The waiver cannot be moved, copied or respelled, and the file exclusion cannot come back, without editing the gate, which is a reviewed CI change.

## The problem

ADR-002 §3 turned `forbidigo` off for the whole of `api.go` so that `GetMe` could call `GroupPaths`. That also turned off the raw HTTP client rule and the `otelpgx.WithIncludeQueryParameters` rule for that file. It is Low severity and not exploitable today, but `api.go` is where new handlers land.

## Options

| Option | Scope of waiver | Catches a 2nd `GroupPaths` caller in `api.go` | Cost | Verdict |
|---|---|---|---|---|
| A. Keep the file exclusion (status quo) | All `forbidigo` rules, whole file | No (R-b) | 0 | Rejected: the finding |
| B. Narrow the exclusion: `path` + `text: GroupPaths` | `GroupPaths` rule only, whole file | No | 1 YAML line | Rejected: still per file |
| C. Move `GetMe` to its own file and exclude that file | All rules, one file | Only while the file holds one handler | file split | Rejected: same hole, smaller |
| D0. Line directive + `nolintlint` + count gate (rev 0) | one line, in principle | Bypassable: the directive can move above `package` or `func`, or be respelled | ~10 lines | Rejected in round 1 (TA finding 1, Security B1) |
| **D. Line directive + `nolintlint` + exact-line gate with config check** | **one line, pinned** | **Yes** (`forbidigo`, or the gate) | ~35-line shell script | **Chosen** |
| E. A function-scoped `grouppaths` analyzer in `navalint` | the exact function, no directive | Yes | about 60 lines plus testdata in the existing `tools/navalint` module (unitchecker and analysistest already in place). It would replace both `forbidigo` `GroupPaths` rules. | Deferred: the right move **when a second waiver appears**; not needed for one |

## Evidence

**Linter behaviour** (golangci-lint 2.14.0, PR #221 head, linting `./internal/server/` with steps 1–3; SA spike rev 0 plus TA and Security round 1):

| Case | Result |
|---|---|
| Exclusion removed, no directive | `forbidigo` at `api.go:94` |
| Step 1 + step 2 | 0 issues |
| Stale directive; bare `//nolint`; `//nolint:all`; no explanation; `// nolint:` with a leading space | each one is a `nolintlint` error |
| Directive added but the exclusion left in | `nolintlint`: directive unused (a half-done change fails) |
| Trailing directive on the `func GetMe` line | unused; `forbidigo` still fires (no widening) |
| **A** directive on its own line above `package server` | **0 issues for the whole file** (lint alone is bypassed) |
| **G** directive on its own line above `func GetMe` | **0 issues for the whole function** |
| **D** `//nolint:FORBIDIGO`, **E** `//nolint: forbidigo` on a 2nd caller | 0 issues |
| **8** directive copied onto a 2nd caller | 0 issues |

A, G, D, E and 8 are the cases lint alone cannot catch. The gate exists to catch them.

**Gate red to green** (`spike-gate-cases.md`, attached; it runs the gate script only, against the same tree):

| # | Case | Gate |
|---|---|---|
| R0 | PR #221 head as it is (exclusion present, no directive) | **fail** (red) |
| P | Steps 1 + 2 applied | **pass** (green) |
| 8 | Directive copied onto a 2nd caller | fail |
| A | Directive moved above `package` + an uncovered 2nd caller | fail |
| G | Directive moved above `func GetMe` | fail |
| D | `//nolint:FORBIDIGO` | fail |
| E | `//nolint: forbidigo` | fail |
| B | `//nolint:gosec,forbidigo` (list form) | fail |
| U | Directive in a new, untracked file | fail (`git grep --untracked`) |
| N1a | `api.go` file exclusion put back | fail |
| N1b | `forbidigo` exclusion in block-list YAML form | fail |
| N1c | `forbidigo` exclusion rule with no `path` | fail |
| N1d | Global `linters.exclusions.paths` entry | fail |
| N1e | Unrelated `gosec`-only exclusion | pass |
| T | Directive in a `_test.go` | pass (out of scope; `nolintlint` reports it unused, because `forbidigo` is excluded for tests) |

**Whole repo** (TA, round 1): `golangci-lint run ./...` with steps 1–3 took **278 s** and found exactly one issue: `tools/ci/suiteran/main_test.go:235` `//nolint:gosec` is unused, because `gosec` is excluded for `_test.go`. There are 77 directives in all, 43 outside tests. All name a linter and give a reason.

## How a second `GroupPaths` caller is caught

| Where | Caught by |
|---|---|
| Another package or file | the `forbidigo` type-qualified rules (ADR-002 AC8, unchanged) |
| `api.go`, no directive | `forbidigo` (R-b closed) |
| `api.go`, with a directive copied, respelled or listed | the directive check in the gate (cases 8, D, E, B) |
| `api.go`, with the one directive moved above `package` or `func` | the directive check in the gate (cases A, G) |
| Anywhere, through a new or wider exclusion in `.golangci.yml` | the config check in the gate (N1a–N1d) |
| A `_test.go` | not covered: `forbidigo` stays excluded for tests (unchanged; not production reach) |

## Acceptance criteria (one Delivery task, which the CTO creates after sign-off)

1. `.golangci.yml`: the `api.go` exclusion is gone, and `nolintlint` is enabled with the three settings.
2. `api.go`: the directive is on the call line, word for word.
3. `scripts/ci/nolint-allowlist.sh` (as in `spike-gate-cases.md`) runs from `make lint`. The Taskfile mirrors it, so `make check-taskfile` passes. The PR shows red to green: R0 fails, P passes, and cases **A, G, D, E, B, 8** and **N1a** each fail.
4. `make lint` and `make lint-fixture` pass. The fixture count stays at 6 `forbidigo` findings.
5. Remove the one unused directive, `tools/ci/suiteran/main_test.go:235` (`//nolint:gosec // the repository's own files`). A whole-repo `make lint` with steps 1–3 otherwise reports nothing (278 s run, TA round 1).

## Residual risk

- **R-b: closed.**
- **R-e (new):** the gate pins the line *text*, not its location. The same line text could appear in a different `api.go` function, with the old line removed. Review of `api.go` is the control. The trigger to adopt option E is a second waiver, or this happening once.
- **Inferred, not run:** `make lint` runs without build tags, so `nolintlint` does not see directives in `integration`- or `temporalstack`-tagged files. The gate does see them, because it greps text. There are no tagged non-test files today.
- **Inferred, not run:** the gate's `awk` was run with Git Bash's gawk. CI's Ubuntu `/bin/sh` and mawk should behave the same, because it uses only POSIX features. AC3's CI run confirms it.

## Invariants (ADR-001 §1, I1–I9)

Checked against `docs/adr/ADR-001-platform-blueprint.md` §1. That section lists **nine** invariants, not the eight the role brief states, and the project description is empty. **No invariant is affected.** The change touches `.golangci.yml`, one comment and one CI script: no runtime path, RLS, role, write pipeline, workflow, outbox or contract. **I5 (one authorization layer):** enforcement narrows from a whole file to one line. The display-only rule that stops group paths becoming a second authorization input now has no file-wide gap.

## Sequencing

The erratum PR adds `docs/adr/ADR-002-E1-*.md`, its `.html`, and this RFC archived under `docs/adr/ADR-002/rfc/` with the spikes. It branches from `main` after PR #221 merges. PR #221 was still OPEN at 2026-10-28 rev 1.

## Changelog (rev 0 → rev 1)

- Step 4: the count gate is replaced by the TA exact-line gate, plus the Security N1 config check (TA 1, Security B1, N1).
- AC3: red to green for R0/P, A, G, D, E, B, 8 and N1a. Spike attached.
- AC5: names the one unused directive and cites the 278 s run; the "about 10, no data" note is removed (TA 2, Security N2).
- Invariants: cited as ADR-001 §1 I1–I9, with I5 narrowed from file to line (TA 3).
- R-e reworded, and option E's cost corrected (TA 4).
