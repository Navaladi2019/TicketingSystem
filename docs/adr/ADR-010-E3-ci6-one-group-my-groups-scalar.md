---
type: adr-erratum
id: ADR-010-E3
title: "Erratum E3 to ADR-010: CI-6 allows a third rewrite (c), a caller: my_groups condition for a caller with exactly one group renders as scalar = $n::uuid"
status: PROPOSED
date: 2026-11-22
issue: TICK-1085
rfc: none (erratum from an implementation finding; framed in the TICK-1085 description, decided by the Technical Lead on TICK-1084 thread 01a200ff-f803)
supersedes: none
amends: "[[ADR-010-E1-ci6-text-equal-up-to-two-rewrites]] Correction paragraph and E1-3 (the list of allowed rewrites grows from two to three); [[ADR-010-composable-access-rules]] Consequences, test bullet 'format-1 equivalence'; RFC-022 §5 CI-6 (rev 1, B3 sentence)"
source: "TICK-1049 (T-A2-5, QA): the ≥ 40% ACL cell misses 350 ms p95 (361 to 429 ms) because the page plan under = ANY(array) reads ~20k rows and sorts; TICK-1084 (S21 fix): the one-line prototype fails only TestReadBranchesFormat1Golden"
evidence: "docs/adr/ADR-010/E3/evidence/ (page-plan-any.txt, page-plan-scalar.txt, probe-after-run.txt, a2-red-3-p95.txt), copied from the TICK-1049 result comment 01a200fd-af6e"
deciders: "Solution Architect (author; verdict); Technical Architect (critique, pending); Technical Lead (chose the erratum path on TICK-1084); CTO (tie-break only if contested)"
---

# ADR-010 erratum E3: CI-6 rewrite (c), one-group `caller: my_groups` renders scalar

> **Status: PROPOSED** on 2026-11-22 by the Solution Architect (TICK-1085). Awaiting the Technical Architect's critique, one revision, then the Solution Architect's verdict. It blocks TICK-1084 (S21 fix), which blocks TICK-1049 (T-A2-5) and story TICK-739.
>
> **Invariants source.** As in ADR-010 and E1: the project description is empty, so the invariant check is against ADR-001 I1 to I9 only and is marked incomplete against the project. This erratum touches I5 (the SQL filter and the Go check agree). I1 (tenant wall) and the rest are unchanged.

## In plain words

- **What was wrong:** for the most common access rule ("service desk agents see tickets assigned to their group"), the database query asks "is the ticket's group *in this list*?" even when the list holds one group. PostgreSQL 16 then cannot read the tickets already in date order: it reads all ~20,000 of the group's tickets and sorts them, 61 ms per page instead of 1 ms. The test that guards the old rules (CI-6) forbids changing that SQL text, except for the two rewrites E1 named.
- **What changes:** CI-6 names a third allowed rewrite, **(c)**: when the caller is in exactly one group, the query says "is the ticket's group *equal to this group*?". The test still fails on any other text change.
- **What you'll see:** nothing in who sees which ticket. The slow list (the ≥ 40% ACL cell) gets its page back from about 61 ms to about 1 ms.

## What was wrong

ADR-010-E1 lets a converted format-1 policy differ from `main`'s `ReadBranches` SQL in exactly two ways: (a) constant-`true` folding and (b) scalar `caller: self`. E1-3 forbids rewriting every one-element `ANY`.

The seeded onboard default `incident` ACL has a rule `default-service-desk` with `where: assignment_group_id in caller: my_groups`. `plan.go` (`CallerMyGroups` case of the condition planner) binds the caller's groups as a set, and `plannedCond.sql` renders it as `(r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($n::uuid[]))` whatever the set's size.

On a fresh stack (TICK-1049, red-3, quiet host), the ≥ 40% ACL cell (`filter[priority]=3` as a single-group agent) measured 361 ms p95 against 350 ms; the other red runs measured 429 and 277 ms. The count statement takes about 5 ms. The page statement is the cost:

| Form of the group branch | Plan of the branch under the `LIMIT` | Rows read from `records_list_group` | Execution |
|---|---|---|---|
| `assignment_group_id = ANY('{g}'::uuid[])` (today) | Index Scan, then `rfv_num` probe for each row, then **Sort** (top-N heapsort) | 20,137 | **61.3 ms** |
| `assignment_group_id = 'g'::uuid` (rewrite c) | Index Scan **in index order**, `rfv_num` probe, stops at the `LIMIT` | 115 (51 returned) | **0.99 ms** |

Both plans: `docs/adr/ADR-010/E3/evidence/page-plan-any.txt`, `page-plan-scalar.txt`. Single-client probe and the 8-client paced run (p50 227 ms): `probe-after-run.txt`.

**Mechanism.** `records_list_group` is `(tenant_id, entity_definition_id, assignment_group_id DESC, updated_at DESC, id DESC)` (`db/migrations/20261109140300_records_list_group.sql`). With a scalar `=` on every prefix column, the index already delivers rows in `updated_at DESC, id DESC` order, so the planner walks it and stops at the `LIMIT`. With `= ANY(array)`, PG 16 treats the column as a set membership test and does not assume a single value, so it cannot use the index order for the sort key and must read every matching row and sort. (PG 17's B-tree `ScalarArrayOp` improvements change parts of this; we pin 16.10, so this is about PG 16. Worth a spike only if we move versions.)

This also breaks the A2-2 input assumption that the page costs at most 20.6 ms, which the 200 ms count budget relies on. The Technical Lead records that drift in `docs/delivery/decision-log.md` (TICK-1049 thread); this erratum fixes the cause.

The Principal Software Engineer's prototype in TICK-1084 (one line under `CallerMyGroups`: mark the condition scalar when the set has one element) fails exactly one test, `TestReadBranchesFormat1Golden` on `onboard_default_incident.json service_desk_group`. All other authz tests pass.

## Correction

CI-6's B3 sentence, as amended by E1, reads from this erratum on:

> `ReadBranches` on a converted format-1 policy produces the same branch count as `main`, and the same SQL text and bind arguments, up to three rewrites: **(a)** a constant `true` conjunct is removed; **(b)** a `where` condition with `caller: self` renders as `= $n::uuid` with a `uuid.UUID` argument, where `main` rendered `= ANY($n::uuid[])` with a one-element `[]string` argument; **(c)** a `where` condition with `caller: my_groups`, for a caller with exactly one group, renders as `= $n::uuid` with a `uuid.UUID` argument equal to that group, where `main` rendered `= ANY($n::uuid[])` with a one-element `[]string` argument. The golden test compares text and arguments after normalizing (a), (b) and (c), not by skipping cases. The rest of CI-6 and the SP-2 threshold are unchanged.

| # | Decision |
|---|---|
| **E3-1** | **Rewrite (c) is allowed.** In the condition planner, a `caller: my_groups` condition whose bound set has exactly one element is planned as a scalar equality, and `plannedCond.sql` renders it through the same `colEq` as (b): `(r.assignment_group_id IS NOT NULL AND r.assignment_group_id = $n::uuid)`, argument `uuid.UUID`. The change is in the shared renderer, so it applies everywhere that condition is rendered: format 1 and format 2, `ReadBranches` and `CompileRead`, the page and the count statement, and both the positive occurrence and every `NOT (...)` occurrence in a later branch's subtraction (one argument, as today: the same `$n` is referenced from every occurrence). |
| **E3-2** | **Scope is narrow, and each boundary is a test.** (i) Only `caller: my_groups` conditions. A literal `in` or `eq` list on any column keeps `= ANY(...)` even with one value (E1-3's reason: a one-element literal list regressing to scalar must stay visible). (ii) Only when the caller has exactly one group. **Zero groups** stays fail-closed exactly as today (`condEmpty`: the condition plans to constant `false` and binds no argument). **Two or more groups** stay `= ANY($n::uuid[])`. (iii) The group-share test inside `record_shares` (`acl_share.principal_id = ANY(...)`) is not a `where` condition and is unchanged. (iv) Nothing else in `plan.go` or the converter changes. Naming: the planner flag that today reads `self` should be renamed to say what it means (scalar rendering), since it no longer implies `caller: self`; this is the implementer's call. |
| **E3-3** | **The golden enforces (c) narrowly; the golden is not regenerated.** `e1Normalize` in `internal/authz/branches_golden_test.go` gains (c): apply it only on a column the fixture compares with `caller: my_groups` in a `where`; only when the caller has exactly one group; only to the argument equal to `[]string [<that group>]`, which becomes `uuid.UUID <that group>`, and only to `= ANY($n::uuid[])` occurrences that reference that `$n`. If a literal `where` on the same column contains the caller's group, the test fails as ambiguous, as (b) does. The allowed-cases list must record **which** rewrites apply to each case, and the test fails if a rewrite applies to a case not listed for it, or to none of its listed cases. After E3 the list is: `onboard_default_incident.json service_desk_group` → (a) and (c); `every_subject.json` × 7 callers → (b), with (a) also on `listed_user` and `listed_user_group`, as E1 lists. The golden at `testdata/format1/read_branches.golden.json` stays `main`'s output and is not regenerated on a story branch. |
| **E3-4** | **Visible-row equivalence holds and is tested.** For a non-null single group `g`, `col IS NOT NULL AND col = ANY(ARRAY[g])` and `col IS NOT NULL AND col = g` are the same two-valued predicate (null `col` gives false in both), so they and their negations select the same rows; the Go check (`plannedCond.allows`, `slices.Contains` over the one id) is unchanged. I5 holds. Required tests, owned by TICK-1084: (1) unit: scalar for one group, `ANY` for two or more, fail-closed for zero; (2) equivalence on real PostgreSQL: the scalar and the array form return identical rows and keyset cursors for the same caller, filter and page (TICK-1084 AC 3); (3) the I5 generator's coverage check (`internal/tenancy/isolation/i5_gen_test.go`, the `has[...]` list) gains two keys, "my_groups with a one-group caller" and "my_groups with a multi-group caller", so the Go-versus-SQL oracle is proven to exercise both renderings; (4) the T-A2-3 and P7-08 isolation tests still pass, and cross-tenant and cross-group reads return nothing. |
| **E3-5** | **The plan test asserts the branch, not the whole statement.** The scalar plan QA measured still has a `Sort` at the top, over the 51 rows the `LIMIT` returns, after the `entity_definitions` and `groups` joins (0.85 ms, quicksort, 43 kB). TICK-1084 AC 2 ("no Sort node") must therefore assert: the `records_list_group` Index Scan for the group branch sits under that branch's `Limit` with **no Sort between them**, and its actual rows are bounded by the page (not the group's size). A test that rejects every Sort node in the plan would fail on the correct plan. |
| **E3-6** | **Not changed:** D5, D6, D11, the SP-2 threshold, the fallback, the CI-6 semantic checks, E1's (a) and (b) and their scope, the A2 count budget. |

## Competitive Gate (ADR-008)

**Result:** N/A. This corrects a test's equality rule and a SQL rendering. It changes no access decision and moves no product axis; it restores the latency the A2 design already assumed.

## Rejected

| Option | Why |
|---|---|
| **Keep `= ANY` and fix the plan elsewhere** (a new index, a planner hint, `ORDER BY` rewrite) | The cause is the `= ANY` shape on PG 16; no index ordered on `updated_at` after `assignment_group_id` helps while the planner cannot treat the column as one value. The Technical Lead ruled this out on TICK-1084. |
| **Blind rewrite: every one-element `ANY` renders scalar** (literal lists included) | Breaks E1-3: it would hide a one-element literal `in` list regressing to a scalar, and it changes goldens no measured problem asks to change. |
| **Regenerate the golden on the story branch** | Turns CI-6 into "whatever the branch does". The golden must stay `main`'s output. |
| **Skip `onboard_default_incident.json service_desk_group` in the golden** | Skips the one seeded policy every tenant runs, the exact case this change touches. |
| **Fix 2 and 3 groups here too** (one ordered branch per group, merged under the `LIMIT`) | Out of scope by the issue: TICK-1084 measures it, and the Technical Lead decides follow-up versus in-PR fix if a page exceeds 20.6 ms. See AR-TICK-1085-01. |

## Accepted risk

| ID | Sev | Risk | Trigger to revisit |
|---|---|---|---|
| AR-TICK-1085-01 | 🟡 Medium | A caller in two or more groups keeps `= ANY($n::uuid[])` under a `caller: my_groups` rule, so the band (ii) page for that caller may still read all matching rows and sort (the 61 ms shape, growing with the groups' size). Latency only: access decisions are unaffected. Not yet measured; TICK-1084 measures 2 and 3 groups. | TICK-1084 reports a 2- or 3-group page above 20.6 ms (then the Technical Lead decides follow-up task versus in-PR fix), or a slow-query report names a multi-group `my_groups` branch. |

## Evidence

- `docs/adr/ADR-010/E3/evidence/page-plan-any.txt` and `page-plan-scalar.txt`: `EXPLAIN (ANALYZE, BUFFERS)` of the same band (ii) page statement on the TICK-1049 red-3 fresh stack (PG 16.10), `filter[priority]=3`, single-group agent, array versus scalar group predicate. 61.270 ms versus 0.992 ms; 20,137 versus 115 rows read from `records_list_group`.
- `probe-after-run.txt` and `a2-red-3-p95.txt`: the single-client probe and the red-3 kit p95 (≥ 40% ACL cell 361 ms against 350).
- TICK-1084 prototype: `go test ./internal/authz/` fails only `TestReadBranchesFormat1Golden` on `onboard_default_incident.json service_desk_group`; nothing committed.
- Golden coverage today: of the six format-1 fixtures, only `onboard_default_incident.json` has a `caller: my_groups` condition, and every golden caller has zero or one group. The golden therefore cannot show that two groups stay `ANY`; E3-4 (1) and (3) cover that.
