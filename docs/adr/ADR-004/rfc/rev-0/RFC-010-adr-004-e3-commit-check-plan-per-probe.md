---
type: rfc
id: RFC-010
title: ADR-004 erratum E3, the commit-check probe is planned per call (no cached generic plan), and Finding 5 tests the plan that runs
status: Draft (rev 0)
date: 2026-11-03
issue: TICK-502
target: ADR-004-E3 (erratum to [[ADR-004-audit-log]] decision 2), plus a revised Finding 5 (TICK-499) pass condition for TICK-428
author: Solution Architect
reviewers: Technical Architect, Postgres & Data Modeling Specialist (critique); CTO (tie-break)
---

# RFC-010: stop the commit check from reusing a plan built on a tiny partition

> **Invariants source.** The project description of *Ticket Management System* is empty: it defines no invariants, and not the eight my role refers to. As in ADR-004, RFC-009 and E2, this RFC is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. **If the project has its own list of eight, these checks are incomplete against it.**
>
> **Numbering.** The highest RFC in the repo or any branch is RFC-009, and ADR-004 has E1 and E2. This is **RFC-010**, targeting **ADR-004-E3**.

## 1. Recommendation

**Add one clause to the commit-check function: `SET plan_cache_mode = force_custom_plan`.** The probe is then planned at every call, with its real values and the partition's *current* size. No plan built while a partition held 2 pages can be reused after an import has grown it. The function already carries a `SET search_path` clause, so this is a second clause on the same line, not a new mechanism.

Pair it with **a revised Finding 5 pass condition** (section 6). Item 2 (b) stops testing the generic plan, which the trigger will never run, and tests the plan that *does* run. Item 3 gains the pooled-session case, with a negative control that must go red without the clause.

**Reject route 3 as a fix.** Pre-creating, pre-analysing or sizing partitions cannot make a young partition large, and the remaining variants (scheduled ANALYZE, a short PgBouncer `server_lifetime`) only shorten the window. **That moves the risk to operational discipline; it does not remove it.** Pre-analysing an empty partition probably makes it worse (section 4, Inferred).

The decision depends on one measurement I don't have: **the per-call cost of planning the probe.** Spike S2 (section 7) gates it against the existing budget of ≤ 100 µs per call. If S2 fails, this RFC goes back to the triad with the number. It does not fall back to route 2 or 3.

## 2. The problem in one paragraph

On a partition of 60 rows (2 pages), a Seq Scan costs 3.35 and every index path costs more. The planner is right: scanning 2 pages *is* the cheapest way to answer one probe. The fault is reuse. Under `plan_cache_mode = auto` (the production default), PL/pgSQL plans its first five calls with the real values, then switches to a cached generic plan when that is no costlier. On a 2-page partition it is not costlier, so the backend caches the Seq Scan. Under PgBouncer transaction pooling, that backend lives on and serves other tenants' transactions. When one of them imports 4,000 subjects, each of the 4,000 deferred probes at COMMIT scans the whole partition, which now holds this import's rows: **172.8 ms → 2,923 ms for 1,000 → 4,000 subjects, ratio 16.9** (TICK-428, `scratch/audit-commit-check/out/cached-plan-reuse.txt`). The plan only goes away when the partition is analysed again, or the backend is recycled.

```mermaid
sequenceDiagram
    participant B as Pooled backend
    participant P as Young partition
    Note over P: month starts, first ANALYZE at ~60 rows (2 pages)
    B->>P: tx 1 (1 subject): probe planned, Seq Scan cached (cost 3.35)
    Note over B: backend returns to the pool, plan stays
    B->>P: tx 2 (import, 4,000 subjects) inserts 4,000 rows
    B->>P: COMMIT: 4,000 probes x Seq Scan over the grown partition
    Note over B,P: O(n²): 2.9 s at 4k, about 18 s at 10k (extrapolated)
```

This recurs every month, on every pooled backend, for every audited table. Item 3 missed it because it opens a fresh session per import, so its plan is first built at COMMIT, after the partition has grown.

## 3. Why the fix has to be "no reuse", not "a better plan"

Any fix that still caches one plan per backend has to pick a plan that is good **both** at 2 pages and at 4,000+ rows. For a single probe on 2 pages, the planner will always prefer a Seq Scan, and it is right to. The candidates that try to force an index shape fail on measured costs:

- **Key order or probe-form changes (route 1 as listed).** Changing the R2 key order changes which index wins *among indexes*. It does not make any index cheaper than reading 2 pages. Rejected.
- **Forcing an index shape with `SET enable_seqscan = off` on the function** (the Technical Architect's first candidate). Measured: the planner then picks a **Bitmap Heap Scan on `(tenant_id, occurred_at)`**, cost 6.24, with `subject_id` in `Filter` (`out/young-partition-plans.txt`). Every entry of one transaction shares that prefix, so this is the O(n²) shape again. Also turning off bitmap scans leaves a plain index-scan tie, prefix against subject index, at 3.36 / 3.36 on the young partition (TICK-499 spike table). That is the S1 F1 tie, which broke the bad way in 9 to 10 of 10 trials. Rejected.
- **`pg_hint_plan` or a similar extension.** A new dependency to install, upgrade and secure in every environment, to force a plan that is wrong for small partitions anyway. Rejected (and would need a CTO decision as a new component).
- **A set-based check (one anti-join per transaction instead of one probe per row).** O(n) by construction, but constraint triggers are row-level only, and a statement-level trigger with transition tables cannot be deferred to COMMIT, so it cannot see audit entries written later in the transaction. This would replace ADR-004 decision 2's mechanism and is out of proportion for this fault. Rejected.

The only lever that removes the fault is the one the Technical Architect named second: **don't cache a generic plan in the trigger.** Two ways to do that:

| Way | How | Verdict |
|---|---|---|
| **`SET plan_cache_mode = force_custom_plan` on the function** | One clause; PL/pgSQL plans each call with the bound values and the partition's current page count. The setting is restored on function exit, so the caller's session (and the next pooled transaction) is untouched. | **Recommended** |
| `EXECUTE format(...) USING ...` (dynamic SQL) | Same effect (one-shot plan per call), plus a parse per call and SQL built from strings inside a `SECURITY INVOKER` trigger. | Rejected: same determinism, more cost, worse to review |

## 4. Decision table

"Determinism under pooled sessions" asks: can the plan a probe executes in transaction *k* depend on what an *earlier, unrelated* transaction on the same backend did? That is what failed.

| Route | What it changes | Determinism under pooled sessions | Worst case at COMMIT | Who carries the risk | Verdict |
|---|---|---|---|---|---|
| **1a. Erratum: R2 key order / probe text** | Index choice among indexes | **None gained.** Seq Scan still beats every index at 2 pages, and is still cached | O(n²), unchanged (2.9 s at 4k) | Code, silently | Reject |
| **1b. Erratum: function `SET enable_seqscan = off` (+ bitmap off)** ("Index Only Scan forced by plan shape") | Cost penalties on scan types | **Worse.** The cached plan becomes the `(tenant_id, occurred_at)` prefix, or a 3.36 / 3.36 cost tie decided by ANALYZE samples | O(n²) by the prefix path, in every session, not only young partitions | Code, silently | Reject (measured) |
| **1c. Erratum: function `SET plan_cache_mode = force_custom_plan`** | No plan survives a call; each probe is planned with real values and current relation size | **Full, by construction.** No plan crosses transactions or backends' histories. A session-level `force_generic_plan` cannot override it (function SET wins), which the test exercises | Linear: n probes, each against the partition as it is at COMMIT. Small partition → cheap Seq Scan; grown partition → subject index (measured on a freshly built plan after growth, cost 8.30) | Code, tested | **Recommend**, gated by S2 planning cost |
| 1d. Erratum: dynamic `EXECUTE` | Same as 1c | Same as 1c | Same as 1c, plus a parse per call | Code | Reject (1c is the same with less) |
| **2. Revised pass condition only** (accept the residual, add the pooled case as a known-red or skipped test) | Nothing in production | **None.** The fault ships; the test only documents it | O(n²), measured 2.9 s at 4k; a 100k import would run for minutes inside one COMMIT holding locks | Tenants who import | Reject alone. **Adopt as the companion of 1c** (section 6) |
| **3a. Pre-create and ANALYZE partitions before they go live** | Stats on an empty partition | **Worse, probably.** An analysed empty partition has `reltuples = 0`, so PG 16 no longer applies the "assume 10 pages" rule for never-analysed relations, and the very first plans are Seq Scans (Inferred from `estimate_rel_size`; TICK-460 (b) passed *because* the partition was never analysed) | O(n²) from the first day instead of the first ANALYZE | Ops | Reject |
| **3b. Minimum size before a partition goes live** | Pad partitions with rows | Would work only with filler entries in an append-only, hash-chained, exported audit log. Not acceptable data | n/a | Auditors | Reject |
| **3c. Scheduled ANALYZE of the current partition, or PgBouncer `server_lifetime`** | How long a stale plan can live | **Partial.** Shrinks the window from "until autovacuum" to "until the next job / recycle". A plan cached at 09:00:00 still serves an import at 09:00:05 | O(n²) inside the window; depends on a cron job and a pooler setting staying right in every environment | **Operational discipline** | **Reject as a fix: it moves the risk to operational discipline.** Keep as nothing; not needed with 1c |

**Determinism in one line:** 1c makes the executed plan a function of *this* probe's values and *this* relation's size only. Every other route leaves it a function of the backend's history.

## 5. What ADR-004-E3 would say

> **Decision 2, added sentence.** The commit-check function carries `SET plan_cache_mode = force_custom_plan`, so no probe plan is reused across calls. The probe keeps the pinned predicate, with `now()` and `pg_current_xact_id()` read into plpgsql locals (S1 F1), and P5-10b's `n_distinct = 1` on `occurred_at` stays. CI-A3 rejects an `audit_commit_check` whose `proconfig` lacks the setting.

- **Why an erratum and not a code comment:** the setting is what makes the commit check's cost linear, which is part of the decision-2 contract that P5-13b and P5-32 rely on. Removing it in a later refactor must fail CI, not pass review by accident.
- **F1 stays.** A custom plan uses the same estimator, so the S1 tie on dense partitions (prefix vs R2 key, 8.46 / 8.46 without the override) would reappear without `n_distinct = 1`. S2 re-runs the TICK-460 checks to confirm.
- **Scope.** One function, one migration (`ALTER FUNCTION audit_commit_check() SET plan_cache_mode = force_custom_plan` in TICK-428's migration, before anything attaches it), one CI-A3 check. No schema, grant, policy or partition change. No role-level or database-level GUC change: nothing outside this function plans differently.

### Invariant check (against ADR-001 I1–I9; incomplete against an eight-item list, see the note at the top)

| Invariant | Effect |
|---|---|
| I1 tenant wall | None. The probe keeps `tenant_id = v_tenant` under the writer's RLS; a custom plan is planned under the same policies |
| I3 least privilege | None. Still `SECURITY INVOKER`, owned by `nava_migrator`, no new grant. A function `SET` clause needs no privilege at call time |
| I4 one write pipeline, audit in the same transaction | **Strengthened.** The database-side guarantee no longer degrades into a multi-second COMMIT that a caller might time out and retry |
| I6 / I7 | None (no workflow or side-effect path touched) |
| I2, I5, I8, I9 | Not touched |

## 6. Revised Finding 5 pass condition (replaces item 2 (b), adds item 3 (c))

Unchanged: TICK-460 (a) and (b), the dense ≥ 10× cost ratio and its `RESET` negative control, item 2 (a), item 3 (a) and (b). All of them run with the function as shipped (clause on).

**Item 2 (b′), young partition, 60 rows, analysed, 10 fresh ANALYZE samples.** Test the plan the trigger executes, captured with `auto_explain` (`log_nested_statements = on`, `log_min_duration = 0`) on probes fired at COMMIT, not a `PREPARE`/`EXPLAIN` generic plan:
- after a 1-subject write: any plan passes (a Seq Scan of 2 pages is the right answer);
- after an in-transaction import of 1,000 subjects into the same partition: **every sampled probe plan has `subject_id` and `occurred_at` in `Index Cond`, in 10 of 10 samples.** Subject index or R2 key both count; don't pin the name;
- every captured plan shows the bound values as constants, not `$n` (proves the plan was custom).

**Item 3 (c), pooled session, new.** One session, with `SET plan_cache_mode = force_generic_plan` at **session** level (the adversarial setting: the function's clause must win). Warm with ten 1-subject writes while the partition is 2 pages, then in that same session import N = 1,000 and then 4,000 subjects into a tenant absent from the statistics. **Pass: t(4N)/t(N) < 8.** This is `reuse.sql` from TICK-428 with the warm-up extended past the five custom-plan calls.

**Item 3 (c) negative control.** The same script against a copy of the function without the clause **must** give t(4N)/t(N) ≥ 8 (measured today: 16.9). If the control is green, the test cannot see the fault, and the gate is not met.

**Catalog check (CI-A3).** `pg_proc.proconfig` of `audit_commit_check` contains `plan_cache_mode=force_custom_plan`. Negative control: a test function without it is reported.

**Budget.** Self time per call (`pg_stat_user_functions`), single subject, 5,000 calls as `nava_worker`, young partition and DEFAULT partition: **≤ 100 µs** (the existing `docs/budgets.md` row; today 42.2 µs with a cached plan).

**Rule unchanged:** if any of these is red or flaky, the TICK-460 rule applies: stop and tell the Technical Lead.

The two skipped tests TICK-428 lands with (item 2 (b) and the pooled-session case) are replaced by 2 (b′) and 3 (c) above, still skipped with TICK-502's id until E3 is accepted and the clause is in.

## 7. Spike S2 (gate, before ADR-004-E3)

- **Question.** Does `force_custom_plan` on the function keep the commit check linear in a pooled session, and what does per-call planning cost?
- **Setup.** postgres:16.10, the TICK-428 harness on `7b5e388` (`internal/db/dbtest/audit_commit_check_test.go` and `scratch/audit-commit-check/`), with `ALTER FUNCTION audit_commit_check() SET plan_cache_mode = force_custom_plan`.
- **Pass.** Everything in section 6 green, negative control red, **and** per-call self time ≤ 100 µs. Report also: 10k-subject commit time (S1 item 2 shape; today 172–198 ms), and the planning share of the call (`EXPLAIN (ANALYZE, SUMMARY)` planning time on the probe).
- **Also confirm (version-specific, I am not taking it from memory):** that PL/pgSQL's SPI plans honour a function-level `plan_cache_mode` in PG 16 (the "constants, not `$n`" check above shows it), and section 4's 3a claim about an analysed empty partition (one `EXPLAIN` on a pre-analysed empty partition).
- **Owner.** Postgres & Data Modeling Specialist, or the TICK-428 implementer through the Technical Lead. Timebox: half a day.
- **If the budget fails** (planning dominates), the RFC returns to the triad with the number. Candidates then are a per-call cost trade (accepting > 100 µs for single-subject writes against linear imports), which is a CTO tradeoff, not a silent downgrade to route 2 or 3.

## 8. Tradeoffs, plainly

- **Cost.** Every audited write now plans one small query per audited row at COMMIT. I don't have the number; S2 gets it. Expect tens of µs on a 5-predicate probe that prunes to one partition (Inferred).
- **Gain.** The commit check becomes linear in every session state, not only in the states a test happened to construct. No cron job, pooler setting or partition procedure has to stay right for it to hold.
- **Lock-in or migration pain.** None: one reversible `ALTER FUNCTION`, before any table attaches the trigger.
- **Where it could still break.** A future PG version that changes how `plan_cache_mode` applies inside PL/pgSQL. The catalog check cannot see that; item 3 (c) can, and runs on every Postgres bump through the pinned-version CI.

## 9. Open questions for the critique

1. **Technical Architect:** do you accept the measured rejection of the "forced Index Only Scan" candidate (bitmap prefix under `enable_seqscan = off`, then the 3.36 / 3.36 tie)? Is there a plan shape I missed that beats a 2-page Seq Scan on cost without disabling scan types?
2. **Postgres specialist:** is the 3a claim right in PG 16.10 (analysed empty partition → no 10-page assumption → Seq Scan from the first plan)? And is there any PL/pgSQL or plancache path where a function-level `plan_cache_mode` is ignored for the probe?
3. **Both:** is ≤ 100 µs per call still the right budget once planning is per call, or should S2 report against p95 overhead as S1 item 1 did?
