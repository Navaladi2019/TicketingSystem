---
type: adr-erratum
id: ADR-010-E1
title: "Erratum E1 to ADR-010: CI-6 compares ReadBranches SQL text and bind arguments up to two named rewrites (constant-true folding, scalar caller: self)"
status: ACCEPTED
date: 2026-11-19
accepted: 2026-11-19 (Solution Architect verdict, TICK-993 thread 01a1f442-03e8)
issue: TICK-993
rfc: none (erratum from an implementation finding; option A framed in the TICK-993 description, critique in thread 01a1f442-03e8)
supersedes: none
amends: "[[ADR-010-composable-access-rules]] Consequences, test bullet 'format-1 equivalence, with the same branch count and SQL as today'; RFC-022 §5 CI-6 (rev 1, B3 sentence)"
source: "TICK-938 (P11-06, PR #765): 8 of 48 golden cases differ in text with identical branch counts (Technical Lead, TICK-993 description)"
evidence: "docs/adr/ADR-010/E1/technical-architect-spike/ (critique.md, seed.sql, spike_test.go.txt)"
deciders: "Solution Architect (author; verdict); Technical Architect (critique round 1, comment 01a1f49c-1893: VERDICT APPROVE, findings 2 and 3 taken into the text, finding 4 routed to PR #765); Technical Lead (option A recommendation); CTO (no tie-break needed, comment 01a1f49c-5348)"
---

# ADR-010 erratum E1: CI-6 text equality, up to two named rewrites

> **Status: ACCEPTED** on 2026-11-19 by the Solution Architect (TICK-993), under the merge policy set by vknavaladi. The Technical Architect's critique approved option A and supplied the starting wording. Nothing was contested, so the CTO confirmed no tie-break was needed. This erratum blocks only the merge of PR #765 (TICK-938), and E1-3 is a condition on that merge.
>
> **Invariants source.** As in ADR-010: ADR-001 I1 to I9; the project description is empty, so the invariant check is against ADR-001 only. This erratum touches I5 (the SQL filter and the Go check agree). I1 to I4 and I6 to I9 are unchanged.

## In plain words

- **What was wrong:** ADR-010 promised that old-format access rules, once converted, produce exactly the same SQL text as before. In 8 of 48 test cases they produce SQL that means the same thing but is written differently.
- **What changes:** the test now allows exactly two named rewrites and still fails on anything else. The design admits that one of them can change which index the database uses; measured, it was the same speed or faster.
- **What you'll see:** nothing. Who can see which ticket is unchanged.

## What was wrong

ADR-010's test list and RFC-022 §5 CI-6 (rev 1, B3) say `ReadBranches` on a converted format-1 policy returns "the same number of branches and the same SQL text" as `main`. PR #765 (TICK-938) measured 6 fixtures × 8 callers = 48 golden cases. The branch count matches in **48 of 48**. Text and arguments match in **40 of 40** of the rest, but differ in 8:

| Fixture × caller | Difference | Arguments |
|---|---|---|
| `every_subject.json` × 7 callers | `assignee_id = ANY($9::uuid[])` becomes `assignee_id = $9::uuid` (a `where` condition with `caller: self`) | `[]string [<id>]` becomes `uuid.UUID <id>` |
| `every_subject.json` × `listed_user`, `listed_user_group` | also `(true AND …)` becomes `(…)` | as above |
| `onboard_default_incident.json` × `service_desk_group` | `(true AND …)` becomes `(…)` | unchanged |
| `seed_allow_all_d_3_27.json` (D-3-27) × 8 | byte-identical | unchanged |

Two things in the original framing were also wrong, and this erratum corrects them:

1. **"No plan change" is false for `caller: self`.** The Technical Architect's spike found that the scalar rendering *does* change index choice in `every_subject`'s 3-branch cases (40 of 40 statement variants). It was neutral or faster in every measurement (see E1-2).
2. **The golden compares bind arguments too**, not only text. The `caller: self` argument changes Go type. Both encode to the same `uuid` through pgx.

## Correction

CI-6's B3 sentence, and the ADR-010 test bullet "format-1 equivalence, with the same branch count and SQL as today", read from this erratum on:

> `ReadBranches` on a converted format-1 policy produces the same branch count as `main`, and the same SQL text and bind arguments, up to two rewrites: **(a)** a constant `true` conjunct is removed; **(b)** a `where` condition with `caller: self` renders as `= $n::uuid` with a `uuid.UUID` argument, where `main` rendered `= ANY($n::uuid[])` with a one-element `[]string` argument. Rewrite (b) can change the planner's index choice within a branch: beside a subject equality on the same column, the two now form one equivalence class instead of being double-counted. It was measured as neutral or faster on 800k records (TICK-993). The golden test compares text and arguments after normalizing (a) and (b), not by skipping cases. The rest of CI-6 (Go and SQL agreement on every generated case, the explainer's rule id, format 1 published as format 2) is unchanged, and so is the SP-2 threshold.

| # | Decision |
|---|---|
| **E1-1** | **Rewrite (a), constant folding, is allowed.** The D6 planner keeps constants only at the root; `Allows` and `CompileRead` share that invariant, so `ReadBranches` does not reintroduce `true` inside a distributed `all`. PostgreSQL removes a `true` conjunct during preprocessing, so the planner never sees it. Measured: identical plan shape in 8 of 8 statement variants for the one seeded ACL it touches (onboard default `incident`, `service_desk_group`): page and count, `updated_at` and `created_at`, custom and generic plans. |
| **E1-2** | **Rewrite (b), scalar `caller: self`, is allowed, and it can change the plan.** After D11 conversion, a `where` condition `assignee_id = caller: self` is the same node as the `assignee` subject, which `main` already rendered as scalar `=`. **Mechanism:** on `main`, `= ANY(...)` is not an equality the planner can merge with the subject's `= $4` on the same column, so it multiplied two selectivities, estimated `rows=1` on `records_list_assignee`, and then discarded 1,280 rows by filter. With both scalar, they form one equivalence class; the estimate is honest and the planner picks `records_list_requester` (40 rows removed). **Measured** (contended host, so only the main-vs-PR difference carries meaning): uniform data, custom plans, main 1.5–12.9 ms, PR 0.32–0.97 ms (one 4.5 ms outlier). Skewed data (requester owns 100,040 rows, caller assigned 20): the planner returns to the assignee index, everything under 2 ms, PR ≤ main within noise, worst case 0.26 → 0.59 ms. **Not proven** for every distribution; the inference is that more accurate estimates should not choose worse. No seeded ACL contains `caller: self` (nava and nava-s18: onboard default, S18 `permit`/`visa` group-only rules), so no running tenant's plan changes. |
| **E1-3** | **The golden test enforces the two rewrites; it does not skip.** In PR #765, `ci6TextPending` in `branches_golden_test.go` skips all text comparison for `every_subject.json`. That fixture is the only one that combines requester, assignee, share, users, groups and `where`, so a permanent skip would hide any future regression there. It is replaced, before PR #765 merges, by a comparison after normalizing `main`'s golden: (a) `(true AND ` to `((`; (b) only for the argument that came from `caller: self` (one-element array equal to the caller's `PrincipalID`), `= ANY($n::uuid[])` to `= $n::uuid` and `[]string [<id>]` to `uuid.UUID <id>`. A blind rewrite of every one-element `ANY` is not allowed: it would hide a real regression of an `in` list to a scalar. Owner: the PR #765 engineer, inside TICK-938 (Technical Lead). |
| **E1-4** | **No code change in the planner or converter.** Option B (make the text identical) is rejected below. |

**Not changed:** D5, D6 (branch distribution, the SP-2 threshold, the fallback), D11 (the conversion). The CI-6 semantic checks (Go and SQL agree on every case; random-tree OR-of-branches equals `Allows`; branches pairwise disjoint, as PR #765 already tests). I1 (tenant wall) is unaffected: neither rewrite touches the `tenant_id` predicate or RLS. I5 holds: both rewrites are semantically equal, and CI-1 continues to compare SQL against Go.

## Competitive Gate (ADR-008)

**Result:** N/A. This corrects a test's equality rule. It moves no product axis and changes no access decision.

## Rejected

| Option | Why |
|---|---|
| **B: make the text identical** (converter emits `caller: self` as `op: in`; planner keeps a literal `true` inside a distributed `all`) | Changes every converted tree and its conversion goldens, and breaks the "constants only at the root" invariant that `Allows` and `CompileRead` share, to preserve text that the database discards (a) or plans worse (b). |
| Keep the CI-6 wording and leave the 8 cases skipped (`ci6TextPending`) | Turns CI-6 off for the one fixture that exercises every subject kind together. |
| Amend to "same text up to semantic equality" without naming the rewrites | Not testable: a reviewer could not tell an allowed rewrite from a regression. Naming exactly (a) and (b) keeps the golden byte-exact everywhere else. |
| Amend with the issue's original claim that neither rewrite changes a plan | Contradicted by the Technical Architect's measurement for (b). |

## Accepted risk

| ID | Sev | Risk | Trigger to revisit |
|---|---|---|---|
| AR-TICK-993-01 | 🟢 Low | Rewrite (b) gives the planner a more accurate estimate, measured on two data shapes as neutral or faster. A data distribution we did not measure could make the new choice slower for policies that use `caller: self` in a `where` beside a same-column subject. No seeded tenant has such a policy today. Access decisions are unaffected; only latency could move. | A tenant publishes a policy with `caller: self` beside a same-column subject and its list p95 regresses against `docs/quality/acl-perf-800k` (CI-7), or a slow-query report names such a branch. |

## Evidence

- Technical Architect critique and spike: `docs/adr/ADR-010/E1/technical-architect-spike/` (`critique.md`, `seed.sql`, `spike_test.go.txt`). PostgreSQL 16.10 (the pinned image), schema and roles dumped from the `nava-s18` stack, RLS active as `nava_app`, statements built with `pipeline.PageStatement` and `CountStatement`, 800,000 records. Control: 248 statements with identical text gave 0 plan differences.
- Golden diff: `TestReadBranchesFormat1Golden` on PR #765 head `55c5cb8f` against main's golden at `2277d6b0`.
- PR #765 random-tree tests: the OR of the branches equals `Allows`, and branches are pairwise disjoint.
