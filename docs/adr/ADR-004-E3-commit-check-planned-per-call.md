---
type: adr-erratum
id: ADR-004-E3
title: "Erratum E3 to ADR-004 decision 2: the commit-check probe is planned on every call (force_custom_plan), Finding 5 tests the plan that runs, and the budget is rebased"
status: PROPOSED
date: 2026-11-03
issue: TICK-502
rfc: docs/adr/ADR-004/rfc/RFC-010-adr-004-e3-commit-check-plan-per-probe.md (rev 1, archived with rev-0/ and rev-1/; thread under TICK-502 comment 01a19fc5-0441)
supersedes: "the `docs/budgets.md` row 'Commit-check trigger … ≤ 100 µs per call' (42.2 µs); Finding 5 (TICK-499) item 2 (b) pass condition"
amends: "[[ADR-004-audit-log]] decision 2, R28 (budget shape note), acceptance test CI-A3"
source: "TICK-428 (P5-11) RESULT: BLOCKED: under a cached generic plan, the probe on a young partition is a Seq Scan, and a later 4,000-subject import in a pooled session is O(n²) at COMMIT (172.8 ms → 2,923 ms)"
deciders: Solution Architect (author); Postgres & Data Modeling Specialist (round 1, Q2, comment 01a19fcf-b245); Technical Architect (round 1 REVISE with S2, comment 01a19fdc-c359; final re-check ACCEPT, comment 01a1a00e-cb7b); CTO (§7 tie-break, comment 01a19fdd-2ed2; approval and text change, comment 01a1a00f-1cdf)
---

# ADR-004 erratum E3: plan the commit-check probe on every call

> **Status: PROPOSED, 2026-11-03.** All reviewers approved RFC-010 rev 1. The CTO decided the one tradeoff (cost against determinism) under RFC §7: accept 1c and rebase the budget. There is no recorded dissent. The Solution Architect gives the verdict on this PR.
>
> **Invariants source.** The project description defines no invariants. As in ADR-004, E1 and E2, this erratum is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. The check is incomplete against any separate list of eight.

## Decision

1. **The commit-check function carries `SET plan_cache_mode = force_custom_plan`.** It sits next to its existing `SET search_path`. Every probe is planned with its own values and the partition's current size, and no plan outlives the call. A session-level or `SET LOCAL` `plan_cache_mode` cannot override it (PG specialist, confirmed).
2. **The probe reads only plpgsql locals.** That covers `now()` and `pg_current_xact_id()` read into locals (S1 F1), and never `NEW.<col>`. With locals, the bound values appear as constants in the plan, and that is what test 2 (b′) checks. P5-10b's `n_distinct = 1` on `occurred_at` stays.
3. **CI-A3 adds a catalog check.** `pg_proc.proconfig` of `audit_commit_check` must contain `plan_cache_mode=force_custom_plan`. Negative control: a test function without it is reported.
4. **Finding 5 tests the plan the trigger runs** (replacing item 2 (b), adding item 3 (c)); see below.
5. **The cost budget is rebased.** Two rows replace the ≤ 100 µs / 42.2 µs row in `docs/budgets.md`; see below. The Postgres specialist commits them after acceptance.
6. **Rollout is a normal migration:** `ALTER FUNCTION audit_commit_check() SET plan_cache_mode = force_custom_plan` in TICK-428's migration, before any table attaches the trigger. A pooled session that already holds the cached Seq Scan picks the change up at its next transaction. No reconnect, ANALYZE or PgBouncer recycle is needed.

Scope: one function, one migration, one catalog check. No schema, grant, policy, partition, role or database GUC changes.

## Context

On a young partition (60 rows, 2 pages), a Seq Scan is the cheapest plan for one probe, and the planner is right to pick it. The fault is reuse. Under `plan_cache_mode = auto`, plpgsql switches to the cached generic plan after five custom plans. A pooled backend then keeps that Seq Scan and serves other tenants' transactions with it. When one of them imports *n* subjects, each of the *n* probes at COMMIT scans the grown partition: **O(n²), 2.9 s at 4,000 subjects** (TICK-428; the PG specialist reproduced it at 12.3–20.0× for 4k against 1k, in both `auto` and `force_generic_plan`). The single-subject path decays too: the shipped function went from 50 µs to 234 µs per call after 5,000 writes in one backend (S2 F4). The old 42.2 µs row was measured in a state that hides this decay.

## Revised Finding 5 pass condition (TICK-428 tests)

Unchanged: TICK-460 (a) and (b), the dense ≥ 10× ratio and its `RESET` control, item 2 (a), and items 3 (a) and (b).

- **Item 2 (b′), young partition, 60 rows, analysed, 10 fresh ANALYZE samples.** The probe plan executed at COMMIT is captured with `auto_explain` (`log_nested_statements = on`, `log_min_duration = 0`). It is not a `PREPARE`/`EXPLAIN` generic plan. After a 1-subject write, any plan passes. After an in-transaction import of 1,000 subjects, every sampled plan has `subject_id` and `occurred_at` in `Index Cond`, in 10 of 10 samples, and either index may supply them. Every captured plan shows bound values as constants, not `$n`.
- **Item 3 (c), pooled session.** It runs in two session `plan_cache_mode` modes, `auto` and `force_generic_plan`. Each mode uses one session: 10 warm-up 1-subject writes while the partition is 2 pages, then imports of N = 1,000 and 4,000 subjects into a tenant absent from the statistics. It runs at least 3 times, on fresh fixtures, and the median of t(4N)/t(N) is used. **Pass: the clause arm's median is < 8 in both modes.**
- **Negative control:** the same script, in both modes, against a copy of the function without the clause. **Its median must be ≥ 8.** Both arms run in the same CI job. Measured: 3.1–5.4 with the clause, 12.3–20.0 without it.

If any item is red or flaky, the TICK-460 rule applies: stop and tell the Technical Lead. Until this erratum is accepted and the clause is in, 2 (b′) and 3 (c) stay skipped with TICK-502's id. P5-13b (TICK-452) needs them green and unskipped.

## Budget (replaces the 42.2 µs row)

**Production partition count: 38.** That is DEFAULT, the current month and 36 retained months. R28 leaves partition management to Epic 30, and M1 has no audit purge (ADR-001 D23), so partitions only accumulate. M1 stays below 38.

| Row | Method | Ceiling | Measured (PG 16.10, laptop, provenance only) |
|---|---|---|---|
| Commit-check trigger, single subject | `pg_stat_user_functions` self_time / calls, `track_functions = pl`. One connection (pooled-backend state), 5,000 single-subject transactions as `nava_worker`. Young partition analysed at 60 rows, autovacuum off on it. 38 partitions. Reported at N = 200, 1,000 and 5,000 | **≤ 300 µs per call, and flat: max/min < 1.5 across the three N** | Round 1: 248–251 µs with the clause, flat (shipped 50 → 234 µs, not flat). Re-check: max/min 1.06–1.30 with the clause, 2.05 shipped |
| Commit-check trigger, import | COMMIT time / N for one 10,000-subject import, fresh session, 38 partitions, median of 3 | **≤ 175 µs per probe** | 120–142 µs at 2 partitions. The 38-partition cost equals the 2-partition cost (ratio 0.80–1.08), so about 96–153 µs at 38 |

- **Where the rows are measured.** Both rows are measured in the CI integration job (pinned `postgres:16.10` image, the same runner class every time), not on a developer laptop. The baselines are re-taken there before the Postgres specialist commits the rows; the S2 laptop numbers are provenance, not the baseline. On one laptop, the same fixture varied 1.8–2.3× within an hour (TICK-502 TA re-check).
- **Same-run reference.** Each run also measures the shipped function (no clause) on the same fixture in the same job. An over-ceiling result fires the S3 trigger only if the same-run ratio is also out of range: import clause/shipped > 15 (measured 9.5–13.1 across two host states), or per call at N = 200 clause/shipped > 5 (measured 1.7–4.1). An over-ceiling result with an in-range ratio is a runner problem: re-run it, and do not fire S3.
- The "flat" condition catches plan decay, which a single N cannot show and the old row missed.

## Options considered

| Route | Determinism under pooled sessions | Verdict |
|---|---|---|
| **1c. `force_custom_plan` on the function (chosen)** | **Complete by construction:** the executed plan depends only on this probe's values and the relation's current size | Chosen. Costs about +0.15–0.2 ms per audited write and about 1.3 s on a 10k import COMMIT, linear |
| 1a. Change the R2 key order or the probe text | None gained: a Seq Scan still beats every index at 2 pages, and it is still cached | Rejected |
| 1b. `enable_seqscan` / `enable_bitmapscan` off on the function | Worse: the `(tenant_id, occurred_at)` prefix plan gets cached (O(n²)), or a 3.36 / 3.36 tie decides it | Rejected (measured) |
| Plan shape by cost, `random_page_cost = 1.0` (TA Q1) | Moves the size at which the plan flips without removing the flip. An analysed empty leaf that gains rows during the transaction goes back to a Seq Scan, and the margins are tie-sized | Rejected (measured) |
| 1d. Dynamic `EXECUTE format()` | Same as 1c | Rejected: it adds a parse per call and string-built SQL in a trigger |
| 2. Revise the pass condition only | None: the O(n²) path ships and the test only documents it | Rejected alone. Adopted as 1c's companion (Finding 5 above) |
| 3a. Pre-create and ANALYZE partitions | Worse: an analysed empty leaf gets `reltuples = 0`, which turns off PG 16's 10-page floor (`tableam.c:705`), so the first plans are Seq Scans | Rejected (PG specialist Q2a) |
| 3b. Minimum partition size | Needs filler rows in a hash-chained, exported audit log | Rejected |
| 3c. Scheduled ANALYZE, or a short PgBouncer `server_lifetime` | Partial: it only shortens the window. Autoanalyze takes every leaf through the "analysed, 60 rows" state, and nothing can analyse during an import | **Rejected: it moves the risk to operational discipline and does not remove it** |
| `pg_hint_plan`; a set-based check | A new component; constraint triggers are row-level only and a statement trigger cannot be deferred | Rejected (RFC-010 §3) |
| S3. Custom plan on the leaf partition | Same as 1c, at about 21–22 µs of planning instead of 90–154 µs | **Deferred** (CTO): a follow-up, not an M1 condition. See risks |

## Accepted risks and follow-up

These are indexed in `docs/security/accepted-risks.md` as AR-TICK-502-01 (E3-R1) and AR-TICK-502-02 (the partition-count part of E3-R2).

- **E3-R1 · planning cost.** Every audited write pays one probe plan per audited row at COMMIT. This is bounded and linear, and it is the price of determinism. In a pooled backend, today's shipped path already costs 234 µs per call after 5,000 writes.
- **E3-R2 · S3 trigger** (the CTO holds it in the risk register). S3 fires when either budget row fails at the production partition count **and** the same-run ratio is out of range. That includes an Epic 30 retention that keeps more than 38 partitions attached, and Epic 30's design must re-run both rows at its count. Before S3 can ship, it must settle a deterministic leaf name (an Epic 30 naming rule from `occurred_at`, or a `tableoid` handoff) and a leaf grant the I3 model allows, and it must not use dynamic SQL.
- **E3-R3 · PG upgrades.** A future PG version could change how `plan_cache_mode` applies inside plpgsql. The catalog check cannot see that, but item 3 (c) can, and it runs on every Postgres bump through pinned-version CI.

## Invariant check (ADR-001 I1–I9)

| Invariant | Effect |
|---|---|
| I1 tenant wall | None. The probe keeps `tenant_id = v_tenant` under the writer's RLS, and a custom plan is planned under the same policies |
| I3 least privilege | None. The function stays `SECURITY INVOKER`, no grant is added, and a function `SET` needs no privilege at call time |
| I4 one write pipeline, audit in the same transaction | **Strengthened.** The database guarantee no longer degrades into a multi-second COMMIT that a caller might time out and retry |
| I2, I5–I9 | Not touched |

## Evidence

- `docs/adr/ADR-004/spikes/postgres/rfc-010/tick502-q2-pg-evidence.txt`: Postgres specialist. Covers the 10-page rule, the plancache source path, and the linearity of both arms in both modes.
- `docs/adr/ADR-004/spikes/technical-architect/rfc-010/s2-round-1.md`: Technical Architect S2. Covers per-call cost, planning split (parent against leaf), the 10k import, and the Q1 plan-shape grid, with scripts.
- `docs/adr/ADR-004/spikes/technical-architect/rfc-010/recheck-38-partitions.md`: final re-check at 38 partitions, the host-variance finding, and the partition builder.
- Original failure: TICK-428 branch `TICK-428/audited-tables-manifest` at `7b5e388`, `scratch/audit-commit-check/RESULT.md`.

## Follow-up (the Technical Lead creates the tasks once this is accepted)

1. TICK-428 migration: add the clause; add the CI-A3 `proconfig` check with its negative control.
2. TICK-428 tests: replace the skipped item 2 (b) and pooled-session tests with 2 (b′) and 3 (c), plus the negative control, all unskipped.
3. Postgres specialist: re-take both budget rows in the CI job at 38 partitions with the same-run reference, then commit them to `docs/budgets.md`, replacing the 42.2 µs row.
4. TICK-452 (P5-13b) starts once 1–3 are green.
