---
type: rfc
id: RFC-005
title: ADR-001 I9 cycle rule, check on interpreter starts and suppress the re-start (amendment A5)
status: IN CRITIQUE (rev 0, framed after the CTO ruling of 2026-10-31; Technical Architect critique pending)
date: 2026-10-31
issue: TICK-361
source: PLAN-003 D-3-16 (TICK-332); grooming TICK-363 (Technical Architect, Principal Software Engineer, Postgres, Security); CTO ruling, TICK-361 comment 01a1926e-0124-7493-a681-56c4c4f2bbbe
amends: "[[ADR-001-platform-blueprint]] §1 I9, Annex B CI-10"
author: Solution Architect
---

# RFC-005: I9 rejects its own demo; check on starts and suppress the re-start

> **Invariants source.** The project description defines no invariants. As with RFC-004, this is checked against [[ADR-001-platform-blueprint]] §1 **I1–I9**. If the project is meant to have its own list of eight, the invariant checks are incomplete against it.

> **Numbering.** RFC-001 to RFC-004 exist, so this is **RFC-005**. The record it produces is **ADR-001-A5**. A3 and A4 are skipped because they would read as Annex A.3 and A.4.

## 1. Summary

- **Recommendation (the CTO's ruling): a workflow definition D starts at most once per (tenant, origin, record).** The check runs in the `EventSink`, on interpreter starts. On a repeat, the start is suppressed, the write commits, and one `cascade_cycle` audit row is written.
- **Why the change is needed:** I9 rejects "the origin seen again on the same record". Epic 09's demo, where D writes back to the record R that started it, is exactly that case, so the demo, and every "on create, change this record" workflow, cannot work.
- **Why an amendment and not an erratum:** I9's intent (no unbounded cascades) holds, but its normative text moves from "rejects the write" to "suppresses the start". nava signs that explicitly.
- **What the human accepts:** a definition cannot run twice on the same record within one origin (RS-A5-1). Signal ping-pong stays bounded by depth only (RS-A5-2).

## 2. Decisions

| # | Decision | Why / what it prevents |
|---|---|---|
| D1 | The check runs on interpreter starts, in the `EventSink` (P3-42), inside the triggering write's transaction. | A write-time check flags two `update_record` steps, or a retried activity, as a cycle. |
| D2 | Key: `cascade_start` with PK `(tenant_id, causation_origin_id, record_id, definition_id)`, all `NOT NULL`, under `FORCE` RLS. Claimed with `INSERT … ON CONFLICT DO NOTHING`. | Durable for the record's lifetime, unlike `outbox` (purged 7 days after dispatch). Tenant-leading key, so no cross-tenant collision. |
| D3 | O comes from the write's own stored causation. A depth-0 write mints O. | Security's condition. A NULL origin would bypass the key. |
| D4 | On a hit: no enqueue, one `cascade_cycle` audit row in the write's transaction, no error to the writer. | Keeps "set X on this record" workflows working, and still ends the loop. |
| D5 | No `Causation.TriggerID` and no `audit_log.causation_trigger_id`. | The definition id is on the start row. P3-09 is unaffected. |
| D6 | TICK-88 owns the migration, `cascade.ClaimStart` and the CI-10 tests (a)–(f). P3-42 calls `ClaimStart`. An `arch_test` forbids enqueuing a start without it. | TICK-88 runs in S9, before the real sink exists in S11b, so the rule has to be testable on its own. |

## 3. Options considered

| Option | Lost because |
|---|---|
| O1 Keep the I9 text | Rejects the S11b demo. |
| O2 Check on writes, key `(O, R, trigger)` with `Causation.TriggerID` | False positives on repeated steps and retries, and a new column on every write. This is the Principal Software Engineer's proposal, declined by the CTO. |
| O3 Check on starts, reject the write | Every "on update" workflow fires itself again on its own write, so that write always fails. |
| O4 Unique index on `outbox` | Rows are purged after 7 days, and `outbox` has no record or definition column. |
| O5 Advisory lock plus `SELECT` | Leaves no durable record across transactions. |
| **O6 Check on starts, suppress (chosen)** | — |

## 4. Risks and accepted residuals

| # | Risk | Owner | Revisit |
|---|---|---|---|
| RS-A5-1 | No re-entry of one definition on one record within an origin. | Solution Architect | A tenant use case needs it. Candidate: a `reentrant` flag. |
| RS-A5-2 | Signals between open runs are bounded by depth only. | Solution Architect | Signal storms in P3-36 or production. |

## 5. Test and acceptance spec (CI-10, new)

| Case | Catches |
|---|---|
| (a) D writes back to R, and the write commits | The old rule rejecting the demo |
| (b) Two `update_record` steps on R both commit, and the second start is suppressed | Write-time false positives |
| (c) D re-triggering itself on R gives one start and one `cascade_cycle` row | An unbounded loop, or duplicate audit rows |
| (d) A retried activity gives no second start | Non-idempotent starts |
| (e) Depth 9 is rejected with `ErrCascadeTooDeep`, as today | A regression of the depth bound |
| (f) A cross-tenant definition id or origin never collides, and tenant B cannot claim in tenant A | A tenant leak through the key |

Spike evidence: `docs/adr/ADR-001/spikes/solution-architect/rfc-005/`, PostgreSQL 16.10.

## 6. Open questions

- **Technical Architect:** is a dedicated `cascade_start` table (D2) the right home, or would you accept the outbox purge window as the key's lifetime?
- **Postgres & Data Modeling Specialist (at TICK-88):** retention and the purge path for `cascade_start`, and the FK to `workflow_definitions` once P3-40 creates it.
