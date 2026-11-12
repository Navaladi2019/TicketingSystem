---
type: adr-erratum
id: ADR-005-E1
title: "Erratum E1 to ADR-005: between M2 and M3, old and new binaries coexist only for existing counters; a type's first create can fail with 23505"
status: ACCEPTED
accepted: 2026-11-12 (Solution Architect verdict after the Technical Architect critique, TICK-665 comment 01a1cb58-39fe; no dissent, no tie-break)
date: 2026-11-11
issue: TICK-665
rfc: none (erratum from an implementation finding; decision thread under TICK-665 comment 01a1cb3b-dde2-7529-bb5c-204bf554c874)
supersedes: none
amends: "[[ADR-005-record-numbering]] §4.2 (Deploy row, 'old and new binaries coexist'), §8 (new accepted risk)"
source: "TICK-665 (P6-01, PR #581): `TestAllocatorPoolDoesNotStarveThePool` (as it stood before `48ae5047`) failed with 23505 on `record_counters_scoped_key` once M2 is applied (Postgres & Data Modeling Specialist, comment 01a1caee-0b64)"
deciders: Technical Architect (critique APPROVE with non-blocking items 1–4 and a retry spike, comment 01a1cb58-39fe); Postgres & Data Modeling Specialist (finding, options A–C, measurements); Technical Lead (decision: option A, B as runbook guidance, C rejected; comment 01a1cb3b-dde2; no dissent); Solution Architect (author)
---

# ADR-005 erratum E1: the first create of a type can fail while both unique indexes exist

> **Status: ACCEPTED** (2026-11-12, Solution Architect verdict on TICK-665). The Technical Architect approved with four non-blocking items, all folded in here (E1-1 `gapless` wording and retry lifetime, the isolation constraint, the reproducer test). No dissent; no CTO tie-break was needed.
>
> **Invariants source.** The project description defines no invariants; checked against [[ADR-001-platform-blueprint]] §1, I1–I9, as ADR-005 is.

## What was wrong

ADR-005 §4.2 says, for the window between M2 (the 3-column unique index `record_counters_scoped_key` is built) and M3 (it becomes the primary key), that **old and new binaries coexist**. That holds for **existing counter rows only**.

`INSERT … ON CONFLICT (cols)` absorbs a conflict only on the unique index its target names (the *arbiter*). Between M2 and M3 the table has two unique indexes on the same rows: `record_counters_pkey (tenant_id, entity_definition_id)` and `record_counters_scoped_key (tenant_id, entity_definition_id, scope_key)`. When two sessions make the **first** upsert for the same (tenant, type) at the same time, the loser waits for the winner, then hits the index that is *not* its arbiter and gets `23505` instead of taking the UPDATE arm. R0 (2-column target) fails on `record_counters_scoped_key`; R1 (3-column target) fails on `record_counters_pkey`. The spike ran sequentially, so it missed this.

Measured (Postgres 16.10, 8 workers × 20,000 upserts, every one a first insert; `docs/adr/ADR-005/spikes/postgres/e1-first-create-race/race-stress.sh`, copied from PR #581 `scratch/tick-665/`):

| Indexes | Statement | Failures |
|---|---|---|
| pkey only (today) | R0 | 0 / 160,000 |
| pkey + scoped_key (M2 applied) | R0 | 15,533 on `scoped_key` |
| pkey + scoped_key (M2 applied) | R1 | 19,829 on `pkey` |
| 3-column pkey (M3 applied) | R1 | 0 / 160,000 |

Nothing is duplicated or corrupted: the failure is a refused create (a 500 to the caller), never a wrong number.

## Correction

**§4.2, Deploy row, last column** reads, from this erratum on: *old and new binaries coexist for existing counters; a type's first create can fail with 23505 until M3 (E1). R1 retries once.*

| # | Decision |
|---|---|
| **E1-1** | **R1 retries once.** The allocation retries exactly once on `23505` whose constraint is `record_counters_pkey` or `record_counters_scoped_key`. `gaps`: the retry is a fresh `tenancy.AllocTx`. `gapless`: the batch opens with `SAVEPOINT`, queued in the same `pgx.Batch` with no extra round trip. On `23505` from the counter statement, the pipeline runs `ROLLBACK TO SAVEPOINT` and rebuilds and re-sends the **whole** batch (counter, record, write key, audit), because the failed statement skips the rest of the batch. The retry and the savepoint exist only in R1: S18 (R2) removes both with M3 (TODO-S17-02), after which the predicate cannot fire. Owner: TICK-666 (P6-02). |
| **E1-2** | **Pinned test.** A concurrent first-create test lands on PR #581 skipped, with the skip naming TICK-666; P6-02 removes the skip. It runs both modes. |
| **E1-3** | **Merge order.** PR #581 (M1, M2) merges into `TICK-664/counter-scope-key` together with the P6-02 PR and immediately before it, so the story branch never runs M2 without the retry. |
| **E1-4** | **Runbook (option B, guidance only).** `docs/data/record-counters-scope-key.md` says: run M3 soon after R1 is fully rolled out; the window is the exposure. |

**Why the retry is sound.** The loser only raises `23505` after the winner committed (if the winner aborts, the loser's insert proceeds). The create transaction runs at PostgreSQL's default READ COMMITTED (no isolation override in `internal/`), so the retried statement sees the committed row in both modes. Only `scope_key = ''` is written before M3 (the publish gate refuses non-default formats until then), so the retry never meets the deterministic pkey failure a non-empty `scope_key` would cause.

**This holds only at READ COMMITTED.** At REPEATABLE READ or above, the same race raises `40001`, not `23505` (Technical Architect spike, TICK-665), and the retry does not fire. Neither code nor `ALTER ROLE/DATABASE … SET default_transaction_isolation` may raise it for `nava_app`. A code-path change is caught by the pinned test once unskipped; a role- or database-level default is not, so it is an operator rule in the runbook.

**Not changed:** I1, I4, I7 and I8 hold as in ADR-005 §7. The retry stays in the pipeline's allocation step under `nava_app`; it adds no write path and no grant.

## Rejected

| Option | Why |
|---|---|
| C: create counter rows in advance (backfill plus a write at type creation) | Conflicts with D-3-9 ("no step creates the row in advance") and `CHECK (next >= 2)`; would need its own erratum for a window that M3 closes anyway. |
| B alone: shorten the window | Narrows the exposure, does not remove it. Kept as runbook guidance under A. |
| Retry until success | One retry is enough once the row exists; an unbounded loop would hide a real constraint fault. |

## Accepted risk

| ID | Sev | Risk | Trigger to revisit |
|---|---|---|---|
| AR-S17-01 | 🟡 Low | While R1 rolls out (after M2, before every replica runs R1), an **R0** binary has no retry, so a type's **first** create can fail with 23505 when it races another first create. Existing counters are unaffected; the caller sees a 500 and can retry; no number is duplicated. Closed by M3. | The R1 rollout or the gap between R1 and M3 is longer than one deploy cycle, or a tenant reports a failed first create |

## Evidence

- Race reproduction and table above: `docs/adr/ADR-005/spikes/postgres/e1-first-create-race/` (script and README); finding in TICK-665 comment 01a1caee-0b64.
- Reproducer: `TestConcurrentFirstCreatesOfATypeAllSucceed` (`internal/records/pipeline`, PR #581; skipped until TICK-666); with the skip removed it fails in both modes with 23505 on `record_counters_scoped_key`. (`TestAllocatorPoolDoesNotStarveThePool` first showed the race; since `48ae5047` it runs the first create alone and no longer races.)
- One-retry spike (Technical Architect): READ COMMITTED, 0 retry failures out of 13,136 retries, numbers = upserts (40,000); REPEATABLE READ fails with `40001`. `docs/adr/ADR-005/spikes/technical-architect/e1-retry-once/`.
- PostgreSQL documentation, `INSERT … ON CONFLICT`: the conflict target names the arbiter index; violations of other unique indexes are raised.
