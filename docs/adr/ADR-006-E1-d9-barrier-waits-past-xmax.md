---
type: adr-erratum
id: ADR-006-E1
title: "Erratum E1 to ADR-006: the D9 barrier must also wait on running xids at or above the snapshot's xmax, found with a fence xid"
status: ACCEPTED
date: 2026-11-19
issue: TICK-745
rfc: none (erratum from an implementation finding; decision thread under TICK-745 comment 01a1f12d-3b8a-7251-a1ff-39304fc2dc46)
supersedes: none
amends: "[[ADR-006-filterable-tenant-fields]] D9 (barrier wait set; the isolation precondition made explicit)"
source: "TICK-745 (P7-05, PR #738): a snapshot `1869:1869:` with xid 1869 still running, so `pg_snapshot_xip` was empty and the barrier did not wait (Principal Software Engineer, comment 01a1f122-2a9e); routed to the spec owner by the Quality, Testing & CI Advisor (comment 01a1f12d-3b8a)"
deciders: Solution Architect (author; verdict after critique); Technical Architect (critique 2026-11-19, comment 01a1f149-9107: sound, four non-blocking points, all taken in rev 1; independent spike T1–T4); Technical Architect second critique on TICK-977 (comment 01a1f14b-c007: no blocking findings, three non-blocking ones, taken in rev 2; spikes S1–S5 and BEFORE/RR/RC); CTO (no tie-break needed: nothing contested)
---

# ADR-006 erratum E1: the barrier also waits on running xids at or above `xmax`

> **Status: ACCEPTED, rev 2** (2026-11-19). The Solution Architect gave the verdict on TICK-977 after two Technical Architect critiques. Both found the correction sound, and nothing was contested, so no CTO tie-break was needed. The E1-3 gate is met: `TestBarrierWaitsForTheNewestXID` merged in PR #743 (TICK-976). Rev 1 took the first critique and added E1-5. Rev 2 took the second critique.
>
> **Rev 2 changelog:** E1-1 records that the fence costs one xid per barrier attempt, and that on a hot standby it fails closed. E1-5 adds a catalogue test that pins the AFTER ROW timing and that no other `records` trigger reads the registry. AR-TICK-745-01 stops claiming a detector we do not have. Its revisit trigger no longer cites a runtime "recompute mismatch" (only the CI harness has one). It now names a change of trigger timing and the E1-5 guard error in logs. The TA spikes are archived under `ADR-006/spikes/technical-architect/e1-critique/`.
>
> **Rev 1 changelog:** E1-1 says the range is unbounded under subtransactions, and that the heartbeat may keep the bounds instead of the list. The D9 sentence matches. E1-2 names the AFTER ROW timing as part of the precondition. New E1-5: a trigger guard makes the READ COMMITTED precondition fail closed. AR-TICK-745-01 is updated to match.
>
> **Invariants source.** The project description defines no invariants, so the invariant check is incomplete. Checked against [[ADR-001-platform-blueprint]] §1, I1–I9, as ADR-006 is.

## What was wrong

D9 says the barrier "takes `pg_current_snapshot()` once, keeps `pg_snapshot_xip` in its heartbeat details, and polls `pg_xact_status`". The xip list is not the set of transactions in progress at the snapshot.

A snapshot's `xmax` is **one past the highest completed xid**, and xip lists only in-progress xids **below** `xmax`. When the transaction holding the newest assigned xid is still running, `xmax` equals that xid, and the transaction is not in xip. The PostgreSQL manual states this in its snapshot components table: "there might be active IDs higher than xmax".

So a records writer that took the newest xid before a flip committed, and whose trigger read the old registry state, can still be running when the barrier returns. Its record then has no side row after `ready`. That is the CI-5 R1 failure D9 exists to prevent.

| Run | Snapshot | xip-only waits on | E1 waits on |
|---|---|---|---|
| TICK-745 integration (PR #738) | `1869:1869:`, 1869 running | nothing | 1869 |
| SA spike, PG 16.10 (`ADR-006/spikes/solution-architect/e1-newest-xid/`) | `731:731:`, 731 running | nothing | 731 (fence M = 732) |

The RFC-015 spikes (`s3.sh`, TA round 2 R1–R3) passed with xip only because their held writer was never the newest xid: later xids kept completing, so `xmax` moved past it. The newest-xid case was not constructed.

## Correction

D9's first sentence reads, from this erratum on:

> **Snapshot barrier after every state flip** (`backfilling`, `ready`, `removing`). In one transaction, the activity takes `pg_current_snapshot()` and then a **fence xid** `M = pg_current_xact_id()`. The wait set is the snapshot's xip **plus** every xid in `[xmax, M)` whose `pg_xact_status` is `in progress`. It keeps the wait set **or its bounds** (`xip`, `xmax`, `M`) in its heartbeat details and polls `pg_xact_status` once per attempt. *(The rest of D9 is unchanged: timer between polls, 1 s backing off to 10 s, nothing sleeps in SQL, `field_index_barrier_slow` after 15 minutes, no extra grant.)*

| # | Decision |
|---|---|
| **E1-1** | **Wait set = xip ∪ in-progress xids in `[xmax, M)`.** The snapshot is taken first and the fence second, in the same transaction. The transaction is rolled back; it only consumes one xid. The range holds the running top-level xids above the latest completed xid, their in-progress subtransaction xids, and any xids assigned between the two statements. It is usually small, but subtransactions make it unbounded: the TA spike measured 100,001 xids, an 811 KB list and a 305 ms fence query for one writer holding 100k savepoints. So the activity keeps either the wait set or its bounds (`xip`, `xmax`, `M`) in its heartbeat details. With the bounds, each poll checks `xip ∪ [xmax, M)`; the upper bound stays `M`, because PostgreSQL refuses xids from the future. Waiting on xids assigned after the snapshot is conservative, never wrong. A retry that finds no heartbeat details takes a fresh snapshot and fence. That is still sound, because any snapshot taken after the flip is a valid barrier. This is what PR #738 implements (`internal/fieldindex/backfill/activities.go`, `snapshotSQL` and `fenceSQL`). **Cost and failure mode** (TA spike S2, S3, S5): exactly one xid per snapshot, which means once per Barrier attempt (three flips per field lifecycle), so there is no measurable wraparound pressure. The fence works inside `BEGIN READ ONLY`. On a hot standby, `pg_current_xact_id()` raises an error, so a misrouted barrier fails and retries instead of silently using a standby snapshot. |
| **E1-2** | **The precondition is now written down.** Every xid above `M` was assigned after the flip committed. That is safe only because records writers run at **READ COMMITTED**, and the D4 trigger is an **AFTER ROW** trigger (so the writer's xid exists before it runs) and reads the registry in a statement of a volatile plpgsql function (so the read takes a fresh snapshot). A writer at REPEATABLE READ or above could take its snapshot before the flip and its xid after the fence, and the barrier would miss it. D9 as written had the same hole. This was confirmed by the TA spike: in T2, a REPEATABLE READ writer got xid 736 above the fence and its trigger saw `off`. In T3, a READ COMMITTED writer saw `ready` in 8 of 8 runs. Code must not raise the isolation level of a `records` write path, and must not change the D4 trigger to BEFORE ROW: a transaction's first write would then read the registry before it has an xid. Operators must not set `default_transaction_isolation` above READ COMMITTED for `nava_app` or `nava_worker`. This is the same operator rule as ADR-005-E1, and it is recorded as AR-TICK-745-01. |
| **E1-3** | **Pinned regression test.** This is a direct test of the newest-xid case (QA, comment 01a1f12d-3b8a). (a) Begin a writer, and assert that `pg_snapshot_xmax(pg_current_snapshot())` equals its xid and that xip is empty. (b) Run `Barrier`, and assert that it does not return until the writer commits. (c) The test must fail with the fence removed. Owner: Principal Software Engineer. Landed as `TestBarrierWaitsForTheNewestXID` in PR #743, a follow-up into `TICK-739/filterable-fields`. It retries up to 20 times until the precondition holds, and fails rather than skips if it never does (TA point 7). It was red with the fence removed. **Merged 2026-11-19 (PR #743); this was the gate for ACCEPTED.** The test is its red-to-green evidence. |
| **E1-4** | **Same barrier for removal.** P7-10 (TICK-750) reuses `fieldindex.Barrier`, so D11 step 2 gets the same wait set. No separate change. |
| **E1-5** | **The precondition fails closed.** The first statement of `record_field_values_sync()` raises if `current_setting('transaction_isolation') <> 'read committed'`, with a SQLSTATE in class 25 (invalid transaction state) and a message naming ADR-006-E1. The check is unconditional: a stale "no registry rows" read is the same hole. It ships as a follow-up migration (`CREATE OR REPLACE FUNCTION`) with an integration test. That test shows an RR insert refused and an RC insert accepted. A catalogue test in the same change asserts from `pg_trigger.tgtype` that the D4 sync trigger is AFTER ROW, and that no other trigger on `records` reads `record_index_fields`. The purge trigger is BEFORE DELETE, but it reads no registry state, so it is unaffected (TA spike 2). TICK-747's write-latency run confirms the per-row cost (one setting read, expected to be negligible; not measured). Owner: a follow-up task the Technical Lead creates. E1's acceptance does not wait for it. |

**Not changed:** D4, D10, D11 and R11. I1, I4, I7 and I8 hold as in ADR-006. The barrier stays under `nava_worker` and needs no new grant: `pg_current_xact_id` and `pg_xact_status` are callable by any role. No tenant data enters the wait set, the heartbeat or the slow-barrier metric.

## Competitive Gate (ADR-008)

**Result:** N/A. This is a correctness erratum. It changes which transactions an internal barrier waits on. It moves no product axis and touches no competitive pain point. Parity: I1 to I9 are unchanged, and no grant or `SECURITY DEFINER` is added.

## Rejected

| Option | Why |
|---|---|
| Keep D9 as written (xip only) | Proven to miss the newest running writer (two independent reproductions above). |
| Wait on every in-progress xid in `[xmin, M)` | Correct, but the range scan grows with the oldest open transaction (`xmin`). xip already answers exactly below `xmax`, so only the range above it needs `pg_xact_status`. |
| Read running xids from `pg_stat_activity.backend_xid` | Other roles' sessions are hidden without `pg_read_all_stats`. That grant is broader than D9's "no extra grant", and it still needs a cut-off point, which is the same fence. |
| Make the trigger take a lock the flip waits on (row lock or advisory lock on the registry row) | It puts a lock on every records write to cover a rare flip. ADR-006 already rejected write-path coordination in favour of the barrier (D4: "the trigger is not the race fix; D9 is"). |

## Accepted risk

| ID | Sev | Risk | Trigger to revisit |
|---|---|---|---|
| AR-TICK-745-01 | 🟡 Low | The barrier's guarantee assumes two things (E1-2): records writers run at READ COMMITTED, and the D4 sync trigger is AFTER ROW. A write path, or a role or database default, at REPEATABLE READ or above, or a state-reading BEFORE ROW trigger on `records`, could write a record across a flip without a side row (TA spike 2 reproduces both). Its filters would then miss that record within the same tenant. There is no cross-tenant effect: the trigger is SECURITY INVOKER under the writer's RLS. Today no records path uses RR or SERIALIZABLE (`git grep` on the PR #738 head), and the trigger is AFTER ROW. **Until E1-5 ships, this is an operator and code-review rule with no runtime detector.** A miss would be silent. Only the CI-4/CI-5 harness checks side rows, so a miss would not be seen in production. After E1-5, a higher isolation level is a refused write, and the catalogue test pins the trigger timing. | A proposal to raise isolation on a records write path, a change of D4 trigger timing or a new state-reading trigger on `records`, or an E1-5 guard error in logs. A runtime side-row recompute check, if one is ever added, also qualifies. |

## Evidence

- Reproduction: `docs/adr/ADR-006/spikes/solution-architect/e1-newest-xid/` (script and README; PG 16.10, `731:731:`).
- Original finding: TICK-745 comment 01a1f122-2a9e (`1869:1869:`), and PR #738 `docs/data/field-index-backfill.md`, Barrier row.
- CI-5 R1 on PR #738 passes with the fence. The no-op barrier mutation fails it (bites). The direct newest-xid test (E1-3) is `TestBarrierWaitsForTheNewestXID` in PR #743. It is red with the fence removed ("the barrier returned while the newest-xid writer was still running") and green with it.
- Technical Architect spike T1–T4 (critique comment 01a1f149-9107): T1 runs the production SQL from #738 and gets `late={732}`. T2 shows the REPEATABLE READ hole. In T3, READ COMMITTED saw `ready` in 8 of 8 runs. T4 measures the subtransaction size.
- Technical Architect second critique (TICK-977 comment 01a1f14b-c007), archived in `docs/adr/ADR-006/spikes/technical-architect/e1-critique/`. S1: under synchronous replication, clog is set before the SyncRep wait, and the row stays invisible while the writer stays `in progress`. S2: the fence works under read-only. S3: on a hot standby it fails closed. S4: the top-level and sub xid of a subtransaction writer are both caught. S5: one xid per fence. Spike 2: BEFORE/RC and AFTER/RR writers escape, AFTER/RC does not.
- PostgreSQL 16 manual, "Snapshot Information Functions", table "Snapshot Components": `xmax` is one past the highest completed transaction ID. xip includes only active IDs between `xmin` and `xmax`, and there might be active IDs higher than `xmax`.
