---
type: adr-amendment
id: ADR-001-A5
title: Amendment A5 to ADR-001 I9 and CI-10, a cascade cycle suppresses the re-start (one start per tenant, origin, record and definition) instead of rejecting the write
status: PROPOSED
date: 2026-10-31
issue: TICK-361
rfc: docs/adr/ADR-001/rfc/RFC-005-a5-i9-cycle-suppress-start.md (rev 0, archived; thread under TICK-361 comment 01a1926d-5f4f-776e-a5b2-6adbecfa0f3d)
supersedes: "the I9 clause \"and rejects a cycle (the origin seen again on the same record)\" and the Annex B CI-10 row in [[ADR-001-platform-blueprint]]"
amends: "[[ADR-001-platform-blueprint]] §1 I9, Annex B CI-10"
source: PLAN-003 D-3-16 (TICK-332), grooming TICK-363, CTO ruling on TICK-361 comment 01a1926e-0124-7493-a681-56c4c4f2bbbe
deciders: Solution Architect (author), Technical Architect (critique pending), CTO (ruled 2026-10-31)
signs: nava (pending)
---

# ADR-001 amendment A5: a cascade cycle suppresses the re-start

> **Status: PROPOSED.** The CTO ruled on 2026-10-31 (TICK-361, comment `01a1926e`). The Technical Architect's critique is pending. nava must sign off, because the normative text of I9 changes.

> **Invariants source.** The project description defines no invariants. As in A1, A2 and E3, this is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. If the project is meant to have its own list of eight, these checks are incomplete against it.

> **Numbering.** A1 and A2 exist. A3 and A4 are skipped because they would read as Annex A.3 and A.4, the same collision E3 avoided. This is **A5**, from **RFC-005**.

## Decision

**A workflow definition D starts at most once per (tenant, origin, record).** The check runs when the pipeline is about to start a workflow, not when it writes. On a repeat, the pipeline suppresses the start, commits the write and records one audit row. The depth limit is unchanged.

**I9, new text** (replaces the whole I9 cell; ADR-001's signed text is not edited, as with A1, A2 and E3):

> **Cascades are bounded.** Every write and every outbox row carries a causation record: an origin event id plus a depth. A write at depth 0 mints a new, non-NULL origin id. The pipeline rejects any write beyond a hard platform depth limit with `ErrCascadeTooDeep` and an audit entry. **A workflow definition starts at most once per (tenant, origin, record).** When a write would start definition D on record R a second time under origin O, the start is suppressed, the write commits, and one `cascade_cycle` audit entry records O, R and D. The writer gets no error. The cycle key and the depth come only from causation stored under the writing tenant's RLS, never from an outbox payload. The limit is platform configuration, not tenant configuration (8, PLAN-001 D-P7).

**Annex B CI-10, new row:**

| # | Proves | Invariant |
|---|---|---|
| CI-10 | (a) D writing back to its triggering record R commits (the S11b demo). (b) D with two `update_record` steps on R: both commit, and the second start of D is suppressed. (c) D re-triggering itself on R gives exactly one start and exactly one `cascade_cycle` audit row. (d) A retried activity creates no second start. (e) A write beyond the depth limit (depth 9) is rejected with `ErrCascadeTooDeep` and audited, as today. (f) The same origin, record and definition uuids in two tenants never collide, and tenant B cannot claim a key in tenant A. | I9 |

| # | Rule | Why |
|---|---|---|
| **A5-1** | **Where:** the check runs in the `EventSink` (PLAN-003 D-3-14, P3-42), inside the triggering write's transaction, before it enqueues a `nava.interpreter` start of D on R. It does not run on writes. | A write-time check flags two `update_record` steps, or a retried activity, as a cycle. |
| **A5-2** | **Key:** a new table `cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id, created_at)`, with `PRIMARY KEY (tenant_id, causation_origin_id, record_id, definition_id)`, every key column `NOT NULL`, `ENABLE` and `FORCE ROW LEVEL SECURITY`, tenant `SELECT` and `INSERT` policies for `nava_app` and `nava_worker`, and no `UPDATE` or `DELETE` grant. The claim is `INSERT … ON CONFLICT ON CONSTRAINT cascade_start_pkey DO NOTHING RETURNING …`. If it returns one row, enqueue. If it returns none, it is a cycle. Composite FK `(tenant_id, record_id) → records`. | It is a dedicated table, not an index on `outbox`, because `outbox` rows are purged 7 days after dispatch (`internal/purge`). A long-running workflow that writes R after that would start D again. `outbox` also has no `record_id` or `definition_id` column. |
| **A5-3** | **Origin:** the claim takes O from the triggering write's own causation, which the pipeline holds in the transaction (`records.causation_origin_id`, P3-09). It never reads O from an outbox payload or a workflow input. A depth-0 write mints O (UUIDv7) before the sink runs. | This is Security's condition. A NULL origin would bypass the key, and the `NOT NULL` column turns that bug into an error. |
| **A5-4** | **On a hit:** the sink enqueues nothing for D, writes one `audit_log` row **in the write's transaction** (`action = cascade_cycle`, reason `ErrCascadeCycle`, origin O, record R, definition D, the writer's actor), and returns `nil` to the pipeline. Other definitions whose trigger matches the same write are claimed independently. | The write commits, so the audit row commits with it. D-3-7a's separate-runner audit stays for `ErrCascadeTooDeep` only, where the write rolls back. If the write rolls back for any other reason, the claim and the audit row roll back with it. That is correct, because nothing was suppressed. |
| **A5-5** | **No trigger on writes.** `Causation` stays `{OriginID, Depth}`. There is no `Causation.TriggerID` and no `audit_log.causation_trigger_id`. P3-09 (TICK-353) is unchanged. | The definition id is already on the start row. |
| **A5-6** | **Ownership:** TICK-88 (P3-12) ships the migration, the claim function `cascade.ClaimStart(ctx, tx, Key) (claimed bool, err error)`, tests for (a)–(f) against a test sink, and an `arch_test` that fails if any code enqueues a `nava.interpreter` start without calling `ClaimStart`. P3-42 (S11b) calls it from the real `EventSink`. P3-40 adds the composite FK `(tenant_id, definition_id) → workflow_definitions` once that table exists. Postgres reviews the migration. | The ruling lands before TICK-88 starts, so the D-3-16 fallback (move the cycle check to S11b) does not apply, unless TICK-88 starts before nava signs. |

## Context

- The old I9 text is "rejects a cycle (the origin seen again on the same record)". It rejects Epic 09's own demo (S11b, TICK-337). A user creates R (origin O, depth 0), D starts, and D's `update_record` writes R at (O, 1), so O is seen again on R.
- With the check on starts, rejecting the write would still break every "on update, set X on this record" workflow, because its own write always fires it again. Suppressing the second start keeps those workflows and still ends the loop.
- `outbox.Causation` is `{OriginID, Depth}` today (`internal/outbox/outbox.go:56-59`), and its comment allows a depth-0 write with no origin. Rule A5-3 changes that.

## Consequences

- **Termination.** Under one origin, the number of starts is at most records written × matching definitions, because each pair starts at most once. Depth bounds the writes. If D1 and D2 fire each other through R, each runs once and then stops.
- **Idempotent starts.** A retried activity that writes R again cannot start D again. This adds to D-3-13's per-step idempotency key, which already stops the write itself from being applied twice.
- **Concurrency.** If two transactions claim the same key, the second waits on the first's uncommitted key. If the first commits, the second gets no row and the start is suppressed. If the first rolls back, the second claims the key. Shown by spike.
- **Signals are bounded by depth only.** A write that signals open runs waiting on `record_changed` (D-3-14) is not a start, so the key does not apply. Two runs signalling each other through R stop at the depth limit with `ErrCascadeTooDeep`, as today.
- **Growth.** One `cascade_start` row per workflow start, kept for the record's lifetime and removed when the record is purged. The table grows at the same rate as the start history.
- **Determinism.** The claim runs in the Go core's write transaction, not in workflow code. The interpreter's history does not change.

## Invariants

| Invariant | Effect |
|---|---|
| I1 tenant wall | `cascade_start` is a tenant table: `tenant_id` leads the key, RLS is `FORCE`d, and the policies use the CI-1 predicate. CI-1 and CI-4 cover it once it is in `db/grants.yaml`. |
| I7 idempotency | Strengthened: a start is idempotent per (tenant, origin, record, definition). |
| I9 | Amended as above. Its intent (no unbounded cascades) is unchanged. |
| I2–I6, I8 | Not touched. `cascade_start` uses no sequence (I8). |

## Rejected options

| Option | Why rejected |
|---|---|
| Keep the I9 text: reject the write when O is seen again on R | Rejects the S11b demo and every "on create or update, change this record" workflow. |
| Check on writes, keyed on `(O, R, trigger)` with `Causation.TriggerID` (Principal Software Engineer) | Two `update_record` steps on R, or a retried activity, write `(O, R, D)` twice and read as a cycle. It also adds a causation column to every write and to `audit_log`, which start-time keying does not need. |
| Check on starts, but reject the write on a hit | Every "on update, set X on this record" workflow fires itself again on its own write, so that write would always fail. |
| A unique index on `outbox` | `outbox` is purged 7 days after dispatch and has no `record_id` or `definition_id`. The key must outlive the longest workflow. |
| An advisory lock plus a `SELECT` instead of a unique key | It works only while the lock is held. It leaves no durable record, so a later write under the same origin cannot see the earlier start. |

## Accepted residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS-A5-1 | A definition that legitimately needs to run twice on the same record within one origin cannot. Example: D1 and D2 each fire the other through R, and the second run of each is suppressed and audited. The workaround is a new user-originated write, which mints a new origin. The CTO accepted this for M1. | Solution Architect | A tenant use case needs re-entry within one origin. Candidate: a per-definition `reentrant` flag, still bounded by depth. |
| RS-A5-2 | Signal ping-pong between open runs is bounded by the depth limit (8), not by the start key. | Solution Architect | P3-36's performance run, or production, shows signal storms. |

## Recorded dissent

- **Principal Software Engineer** proposed carrying the trigger on writes (`Causation.TriggerID`, `audit_log.causation_trigger_id`). The CTO declined it as unnecessary under start-time keying (TICK-361, comment `01a1926e`).

## Evidence

- Spike in `docs/adr/ADR-001/spikes/solution-architect/rfc-005/` on `postgres:16.10`, 2026-10-31, run as a non-owner role under `FORCE ROW LEVEL SECURITY`. `spike.sql` and `output.txt` cover cases (a)–(c), D1 and D2 firing each other, a new origin, cross-tenant uuids (f), tenant B claiming in tenant A (rejected by the RLS policy), a NULL origin (not-null violation) and a rollback releasing the key. `concurrent.sh` and `concurrent-output.txt` cover two concurrent claims, ending in COMMIT and in ROLLBACK.
- Cases (d) and (e) need the pipeline, so they are TICK-88's tests. The outbox side of (e) exists today at `internal/outbox/outbox_integration_test.go:357`.
- RFC-005 rev 0 is archived at `docs/adr/ADR-001/rfc/RFC-005-a5-i9-cycle-suppress-start.md`, with a copy in `rev-0/`.
