---
type: adr-erratum
id: ADR-002-E1
title: Erratum E1 to ADR-002, a line-scoped nolint pinned by a CI gate replaces the api.go forbidigo exclusion
status: ACCEPTED
date: 2026-10-28
accepted: 2026-10-28 (nava, TICK-272 comment 01a18105-fb21-7b3f-a337-c42df39e03d8)
issue: TICK-272
rfc: docs/adr/ADR-002/rfc/RFC-006-adr-002-e1-line-scoped-nolint.md (rev 1, archived; thread under TICK-272 comment 01a18067-6c25-7b6e-9480-792ee59b7f17)
supersedes: "[[ADR-002-me-group-paths]] §3 'Lint rule' row (the per-file exclusion for internal/server/api.go) and §7 residual R-b"
amends: "[[ADR-002-me-group-paths]] §3 'Lint rule' row, §5 AC8 row, §7 R-b"
source: TICK-268 / PR #204, Security & Multi-Tenancy Reviewer finding 2 (Low)
deciders: Solution Architect (author), Technical Architect (APPROVE, round 2), Security & Multi-Tenancy Reviewer (APPROVE, round 2), CTO
signs: nava (signed 2026-10-28: "APPROVED")
---

# ADR-002 erratum E1: one line waives `forbidigo` for `GroupPaths`, not the whole of `api.go`

> **Status: ACCEPTED, 2026-10-28.** nava approved it on TICK-272 (comment `01a18105`: "APPROVED"), accepting residual R-e (AR-039). Delivery plans the implementation in TICK-301. This erratum comes from RFC-006 rev 1. Both reviewers asked for changes in round 1, to the same bypass of the step-4 gate. Both approved rev 1 in round 2: the Technical Architect in comment `01a180a3` and the Security & Multi-Tenancy Reviewer in comment `01a180a0`. The CTO accepted it for write-up in comment `01a180a4` and folded the generated-file check (F3) into AC3. The plain-language explainer is `ADR-002-E1-explained.html`.

> **Invariants source.** The project description defines no invariants. As in ADR-002 and ADR-001 A1, A2 and E3, this erratum is checked against [[ADR-001-platform-blueprint]] §1 I1–I9.

## Decision

ADR-002 §3 "Lint rule" row, which says "plus a per-file exclusion for `api.go`", is replaced by:

1. **No `forbidigo` exclusion for `internal/server/api.go`.** Delete `path: ^internal/server/api\.go$ / linters: [forbidigo]` from `.golangci.yml`. The raw `net/http` client rule and the `otelpgx.WithIncludeQueryParameters` rule apply to `api.go` again.
2. **One line-scoped waiver.** The `GroupPaths` call in `GetMe` carries a trailing directive, word for word:
   `raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller`
3. **`nolintlint` enabled**, with `require-specific: true`, `require-explanation: true` and `allow-unused: false`. Before this change it was not enabled at all, because it is not part of `default: standard`.
4. **`scripts/ci/nolint-allowlist.sh`, run from `make lint`**, with three checks. Each one fails closed:
   - **Directives.** `git grep --untracked -i -E 'nolint[^/]*forbidigo'` over non-test `.go` files outside `docs/` must return exactly the one line in step 2. It matches any spelling golangci-lint honours: any case, extra spaces, comma lists. It also pins the waiver to the call line, so moving the directive above `package` (whole file) or above `func` (whole function) fails.
   - **Config.** `.golangci.yml` fails the gate if it has a `forbidigo` exclusion rule for any path other than `_test\.go` or `^internal/platform/telemetry/`, a `forbidigo` rule with no `path`, or any global `linters.exclusions.paths` entry.
   - **Generated files (F3).** The set of `.go` files outside `docs/` whose header matches `^// Code generated .* DO NOT EDIT\.$` must be exactly `api/api.gen.go` and `internal/apitest/api.gen.go`. golangci-lint skips *every* linter on a generated file, `nolintlint` included, so a fake header on `api.go` would otherwise turn off the whole file with CI green.

Adding a second waiver, a new `forbidigo` exclusion or a new generated file means editing the gate. That is a reviewed CI change.

ADR-002 §5, AC8 row, now reads: "AC8, the `forbidigo` rules, plus the ADR-002-E1 gate | a second caller of `GroupPaths` anywhere outside the one `GetMe` line".

ADR-002 §7 R-b is **closed** and replaced by R-e (below).

## Context

- Under ADR-002 §3, `GetMe` may call `GroupPaths` and no other production code may. `forbidigo` cannot scope a rule to one function. The accepted design therefore excluded the whole file, and that exclusion also turned off the file's other two `forbidigo` rules. The Security & Multi-Tenancy Reviewer raised this on TICK-268 (PR #204) as finding 2, Low severity, not exploitable today.
- Round 1 showed that a line directive alone is not enough. golangci-lint 2.14.0 applies a `//nolint` on its own line above `package` to the whole file, and one above `func` to the whole function (cases A and G). A count-only gate also misses `//nolint:FORBIDIGO` and `//nolint: forbidigo` (cases D and E). Round 2 found that a `Code generated ... DO NOT EDIT.` header silences every linter on the file (F3).

## Consequences

- One Delivery task, which the CTO creates after sign-off. Its acceptance criteria:
  1. `.golangci.yml`: the `api.go` exclusion is removed and `nolintlint` is enabled with the three settings.
  2. `api.go`: the directive from Decision step 2 is on the call line, word for word.
  3. `scripts/ci/nolint-allowlist.sh`, as in `ADR-002/spikes/solution-architect/rfc-006/gate-cases.md`, runs from `make lint`, and the Taskfile mirrors it so `make check-taskfile` passes. The PR shows red to green. The head before the change (R0) fails and the change (P) passes. Each of these cases also fails: **A** (above `package`), **G** (above `func GetMe`), **D**, **E**, **B** (`//nolint:gosec,forbidigo`), **8** (a copied directive), **N1a** (the `api.go` exclusion put back) and **F3** (a generated-file header on `api.go`).
  4. `make lint` and `make lint-fixture` pass. The fixture count stays at 6 `forbidigo` findings.
  5. Remove the one directive that `nolintlint` reports as unused: `tools/ci/suiteran/main_test.go:235` (`//nolint:gosec // the repository's own files`). With that one line removed, a whole-repo run is otherwise clean (Technical Architect's 278 s run, round 1).
- No runtime code changes. ADR-002's signed text is not edited: this file sits beside it, as E3 does beside ADR-001.

## How a second `GroupPaths` caller is caught

| Where | Caught by |
|---|---|
| Another file or package | `forbidigo` type-qualified rules (ADR-002 AC8, unchanged) |
| `api.go`, no directive | `forbidigo` (R-b closed) |
| `api.go`, directive copied, respelled or in a list | gate, directive check (cases 8, D, E, B) |
| `api.go`, the one directive moved above `package` or `func` | gate, directive check (A, G) |
| A new or wider exclusion in `.golangci.yml` | gate, config check (N1a–N1d). Forms the `awk` misses (F1, F2) still fail, because `nolintlint` then reports the pinned directive as unused. |
| `forbidigo` removed from `linters.enable` | `make lint-fixture` (it expects 6 `forbidigo` findings) |
| A fake generated-file header | gate, generated-file check (F3, F3b) |
| A `_test.go` file | not covered. `forbidigo` stays excluded for tests (unchanged, and not production reach) |

## Invariants

| Invariant | Effect |
|---|---|
| I5 (one authorization layer) | Strengthened. The display-only rule that keeps group paths out of authorization is now enforced on one line, not across a whole file. |
| I1–I4, I6–I9 | Not touched. The change is lint config, one comment and one CI script. It touches no runtime path, RLS, role, grant, write pipeline, workflow, outbox or contract. |

## Rejected options

| Option | Why rejected |
|---|---|
| Keep the file exclusion | It is the finding: every `forbidigo` rule stays off for `api.go`. |
| Narrow the exclusion with `text: GroupPaths` | Still per file, so any handler in `api.go` could call `GroupPaths` (R-b stays). |
| Move `GetMe` to its own excluded file | The same hole, smaller. It holds only while nobody adds to that file. |
| Line directive plus a count gate (RFC-006 rev 0) | Bypassable. The one allowed directive can move above `package` or `func` and silence the whole file or function, and `//nolint:FORBIDIGO` and `//nolint: forbidigo` escape a case-sensitive count (round 1, cases A, G, D, E). |
| A function-scoped `grouppaths` analyzer in `navalint` | The right shape. It would cost about 60 lines plus testdata in the existing `tools/navalint` module, would replace both `forbidigo` `GroupPaths` rules, and would need no directive. It is deferred until a second waiver is needed (trigger on R-e). |

## Accepted residual

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| R-e (AR-039) | The gate pins the line *text*, not its location. The same line could be moved to another `api.go` function, with the original removed. Review of `api.go` is the control. Severity Low, not an isolation gap. | Solution Architect | A second `forbidigo` waiver is needed, or this happens once. Then adopt the `navalint` analyzer. |

**Inferred, not run.** `make lint` runs without build tags, so `nolintlint` does not see `integration`- or `temporalstack`-tagged non-test files. There are none today. The gate greps text, so it sees them anyway.

## Recorded dissent

None. Both reviewers approved in round 2. The CTO put F3 into this erratum's Delivery task, not a follow-up (comment `01a180a4`). The CTO also ruled the optional `awk` change for F1 out of scope, because `nolintlint` plus the exact-line gate already makes F1 fail.

## Evidence

All runs used golangci-lint 2.14.0 (go1.26.10) on the PR #221 head (`origin/TICK-47/group-changes-within-10-minutes`).

- `ADR-002/spikes/solution-architect/rfc-006/forbidigo-nolint.md` (rev 0): lint behaviour for the exclusion, the line directive and `nolintlint`, cases 0–10.
- `ADR-002/spikes/solution-architect/rfc-006/gate-cases.md`: the gate as specified here, 17 out of 17 cases as intended. That covers R0 red, P green, then 8, A, G, D, E, B, U, N1a–N1d and F3/F3b failing, and T and N1e passing.
- Technical Architect, round 1 (comment `01a18097-33e3`): reproduced A, D, E and G, and ran whole-repo lint (278 s, one unused directive). Round 2 (comment `01a180a3`): re-checked the gate at `027a9f4`, found F1–F4, and confirmed that mawk and gawk give the same output.
- Security & Multi-Tenancy Reviewer, round 1 (comment `01a18091`): B1, and found 43 directives outside tests. Round 2 (comment `01a180a0`): B1 resolved, and reproduced F3.
- The archived RFC: `ADR-002/rfc/RFC-006-adr-002-e1-line-scoped-nolint.md` (as approved), plus `rfc/rev-0/` (Draft) and `rfc/rev-1/` (Revised).
