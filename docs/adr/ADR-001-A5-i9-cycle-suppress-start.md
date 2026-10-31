---
type: adr-amendment
id: ADR-001-A5
title: Amendment A5 to ADR-001 I9 and CI-10, a cascade cycle suppresses the re-start (one start per tenant, origin, record and definition) instead of rejecting the write
status: ACCEPTED
date: 2026-10-31
issue: TICK-361
rfc: docs/adr/ADR-001/rfc/RFC-005-a5-i9-cycle-suppress-start.md (rev 1, archived with rev-0/ and rev-1/; thread under TICK-361 comment 01a1926d-5f4f-776e-a5b2-6adbecfa0f3d)
supersedes: "the I9 clause \"and rejects a cycle (the origin seen again on the same record)\" and the Annex B CI-10 row in [[ADR-001-platform-blueprint]]"
amends: "[[ADR-001-platform-blueprint]] §1 I9, Annex B CI-10"
source: PLAN-003 D-3-16 (TICK-332), grooming TICK-363, CTO ruling on TICK-361 comment 01a1926e-0124-7493-a681-56c4c4f2bbbe, Technical Architect critique round 1 (TICK-361 comment 01a19279-481f-707d-b585-9b69a7cb42df)
deciders: Solution Architect (author), Technical Architect (round 1, one blocking finding, closed in rev 1: TICK-361 comment 01a192db-3753-7c57-a074-6fcb8d8c9144), CTO (ruled 2026-10-31)
signs: Technical Lead, delegated by nava for this amendment only (TICK-361 comment 01a192d3-bc6d-76f6-8604-a20844e89bb5); accepted by the Technical Lead under nava's delegation, 2026-10-31
---

# ADR-001 amendment A5: a cascade cycle suppresses the re-start

> **Status: ACCEPTED (rev 1), 2026-10-31.** Accepted by the Technical Lead under the sign-off nava (vknavaladi) delegated for this amendment only (comment `01a192d3`), after the Technical Architect closed the blocking finding (comment `01a192db`).
>
> **History.** The CTO ruled on 2026-10-31 (TICK-361, comment `01a1926e`). The Technical Architect's round-1 critique (comment `01a19279`) had one blocking finding (1 and 1b, causation through signals). Rev 1 adds A5-7 and CI-10 (g) for it and applies fixes 2–6 and 9. nava has delegated sign-off for this amendment to the Technical Lead (comment `01a192d3`).

> **Invariants source.** The project description defines no invariants. As in A1, A2 and E3, this is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. If the project is meant to have its own list of eight, these checks are incomplete against it.

> **Numbering.** A1 and A2 exist. A3 and A4 are skipped because they would read as Annex A.3 and A.4, the same collision E3 avoided. This is **A5**, from **RFC-005**.

## Decision

**A workflow definition D starts at most once per (tenant, origin, record).** The check runs when the pipeline is about to start a workflow, not when it writes. On a repeat, the pipeline suppresses the start, commits the write and records one audit row. **Every interpreter write is one level deeper than the delivery that caused it, so signals between running workflows hit the depth limit too.** The depth limit is unchanged.

**I9, new text** (replaces the whole I9 cell; ADR-001's signed text is not edited, as with A1, A2 and E3):

> **Cascades are bounded.** Every write and every outbox row carries a causation record: an origin event id plus a depth. A write at depth 0 mints a new, non-NULL origin id. A workflow's write is one deeper than the start or signal that caused it. The pipeline rejects any write beyond a hard platform depth limit with `ErrCascadeTooDeep` and an audit entry, and the workflow run fails without retrying. **A workflow definition starts at most once per (tenant, origin, record).** When a write would start definition D on record R a second time under origin O, the start is suppressed, the write commits, and a `cascade_cycle` audit entry records O, R and D. The writer gets no error. The cycle key and the depth come only from causation stored by the platform under the writing tenant's RLS, never from an outbox payload or tenant-authored definition data. The limit is platform configuration, not tenant configuration (8, PLAN-001 D-P7).

**Annex B CI-10, new row:**

| # | Proves | Invariant |
|---|---|---|
| CI-10 | (a) D writing back to its triggering record R commits (the S11b demo). (b) D with two `update_record` steps on R: both writes commit, D has exactly one start, and there are exactly 2 `cascade_cycle` rows (one per write that fires D again). (c) D re-triggering itself on R: exactly one start, and one `cascade_cycle` row per write D makes to R. (d) A retried activity creates no second start. (e) A write beyond the depth limit (depth 9) is rejected with `ErrCascadeTooDeep` and audited, as today. (f) The same origin, record and definition uuids in two tenants never collide, and tenant B cannot claim a key in tenant A. (g) Runs A and B both wait on `record_changed` on R, and each writes R when signalled: the first write at depth 9 or more is rejected with `ErrCascadeTooDeep`, that run fails without retrying, and R has at most 8 writes under O. | I9 |

| # | Rule | Why |
|---|---|---|
| **A5-1** | **Where and in what order:** the check runs in the `EventSink` (PLAN-003 D-3-14, P3-42), inside the triggering write's transaction, before it enqueues a `nava.interpreter` start of D on R. It does not run on writes. `ClaimStart` runs **after** the triggering write holds R's row lock. | A write-time check flags two `update_record` steps, or a retried activity, as a cycle. Every key a transaction claims is for the R it has already locked, so concurrent claims on one key are serialized on R's row, and claims for several definitions in one transaction cannot deadlock (Technical Architect, round 1, finding 7). |
| **A5-2** | **Key:** a new table `cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id, created_at)`, with `PRIMARY KEY (tenant_id, causation_origin_id, record_id, definition_id)`, every column `NOT NULL`, `ENABLE` and `FORCE ROW LEVEL SECURITY`, tenant `SELECT` and `INSERT` policies for `nava_app` and `nava_worker`, and no `UPDATE` or `DELETE` grant. Composite FK `(tenant_id, record_id) → records`. The claim is `INSERT … ON CONFLICT ON CONSTRAINT cascade_start_pkey DO NOTHING RETURNING …`. If it returns one row, enqueue. If it returns none, it is a cycle. | It is a dedicated table, not an index on `outbox` or `workflow_ref`. `outbox` rows are purged 7 days after dispatch and `outbox` has no `record_id` or `definition_id`. `workflow_ref.kind` is the Temporal type, not the definition, and it is purged 30 days after close. Confirmed by the Technical Architect. |
| **A5-3** | **Origin at depth 0:** a depth-0 write mints O (UUIDv7) before the sink runs. `outbox.Enqueue` rejects `uuid.Nil` as `Causation.OriginID` with `ErrInvalidEntry`, and the pipeline rejects a write without an origin. TICK-88 removes the "may have no origin" comment on `outbox.Causation` (`internal/outbox/outbox.go:53-56`) and the NULL branch in `Enqueue`. | Security's condition. A NULL origin would bypass the key. The `NOT NULL` column catches it only on starts, so signals and P3-09 record writes need the same rule (finding 5). |
| **A5-4** | **On a hit:** the sink enqueues nothing for D, writes one `audit_log` row **in the write's transaction** (`action = cascade_cycle`, reason `ErrCascadeCycle`, origin O, record R, definition D, the writer's actor), and returns `nil` to the pipeline. The row is written per suppressed attempt, not per key. Other definitions whose trigger matches the same write are claimed independently. | The write commits, so the audit row commits with it. D-3-7a's separate-runner audit stays for `ErrCascadeTooDeep` only, where the write rolls back. If the write rolls back for any other reason, the claim and the audit row roll back with it. Every "on update, set X" run writes one such row on its normal path, so it is routine, not an alarm. |
| **A5-5** | **No trigger on writes.** `Causation` stays `{OriginID, Depth}`. There is no `Causation.TriggerID` and no `audit_log.causation_trigger_id`. P3-09 (TICK-353) is unchanged. | The definition id is already on the start row. |
| **A5-6** | **Claim token, enforced by the compiler.** `package cascade`: `type Claim struct{ outboxID uuid.UUID }`, with the field unexported so only `ClaimStart` can build one, and `func ClaimStart(ctx, tx pgx.Tx, k Key) (c Claim, claimed bool, err error)`, which mints the UUIDv7 outbox id and inserts it into `cascade_start`. `package outbox`: `func EnqueueStart(ctx, tx pgx.Tx, c cascade.Claim, e Entry) (uuid.UUID, error)` is the only way to enqueue a `nava.interpreter` start, and it uses `c`'s id as the outbox row id. `Enqueue` rejects `TargetStart{Kind: "nava.interpreter"}` with `ErrInvalidEntry`. The `arch_test` shrinks to "nothing outside `EnqueueStart` calls the internal insert". **Ownership:** TICK-88 (P3-12) ships the migration, `ClaimStart`, `EnqueueStart`, and the CI-10 tests (a)–(f) against a test sink. P3-42 (S11b) calls them from the real `EventSink` and owns (g) once the interpreter exists. P3-40 adds the composite FK `(tenant_id, definition_id) → workflow_definitions`. Postgres reviews the migration. | Claiming first fills `outbox_id NOT NULL` without a savepoint. Enqueuing first would force a hit to undo the outbox row, the `records.version` bump and the `workflow_ref` row (finding 2). A type token enforces the order at compile time instead of by parsing (finding 3). The ruling lands before TICK-88 starts, so the D-3-16 fallback does not apply, unless TICK-88 starts before sign-off. |
| **A5-7** | **Causation through starts and signals.** Each interpreter write carries the causation of the most recent delivery the run handled: after the start, `(start.origin, start.depth + 1)`; after a `record_changed` signal s, `(s.origin, s.depth + 1)`. Every write is exactly one deeper than the delivery that caused it, so no causal chain exceeds the limit. The interpreter takes these values from the `Delivery` fields `causation_id` and `causation_depth`, which the relay copies from the outbox row's stored columns, not from `payload` (`internal/outbox/outbox.go:88-96`). It holds them in interpreter state that no step input or definition expression can set. The pipeline rejects a `source = workflow` write at depth 0 with `ErrInvalidEntry`. `update_record` returns `ErrCascadeTooDeep` as a non-retryable `temporal.ApplicationError` (`temporal.NewNonRetryableApplicationError`, type `nava.CascadeTooDeep`), and the run fails. The stop relies on the non-retryable flag, not on the type name: the worker guard may retype the error `nava.Opaque`, but it keeps the flag (`internal/workerguard/results.go:170-174`, AR-043). Return the `ApplicationError` unwrapped, with `ErrCascadeTooDeep` as its cause: the worker guard reads the flag by top-level type assertion (`results.go:163`), so wrapping it with `%w` loses the flag. `ErrCascadeCycle` never reaches a writer (A5-4), so it needs no classification. | Without the rule, the natural implementation stamps every write with the start's causation. Two runs signalling each other through R would then write at a constant depth forever, and no start would happen to stop them (finding 1). The Temporal Go SDK's default activity `RetryPolicy` retries without limit (`MaximumAttempts: 0`), so a retryable `ErrCascadeTooDeep` would become a poison run (finding 1b). Rejecting depth-0 workflow writes stops a run from restarting the chain. |

## Context

- The old I9 text is "rejects a cycle (the origin seen again on the same record)". It rejects Epic 09's own demo (S11b, TICK-337). A user creates R (origin O, depth 0), D starts, and D's `update_record` writes R at (O, 1), so O is seen again on R.
- With the check on starts, rejecting the write would still break every "on update, set X on this record" workflow, because its own write always fires it again. Suppressing the second start keeps those workflows and still ends the loop.
- `outbox.Causation` is `{OriginID, Depth}` today (`internal/outbox/outbox.go:56-59`), and its comment allows a depth-0 write with no origin. A5-3 changes that.
- A signal row already carries its writer's causation, and `Delivery` hands it to the run (`internal/outbox/outbox.go:88-96`, `internal/relay/dispatch.go:242`). No rule said the run's next write must use it. A5-7 adds that rule.

## Consequences

- **Termination.** Under one origin, the number of starts is at most records written × matching definitions, because each pair starts at most once. Every write, including one caused by a signal, is one deeper than its cause, so the depth limit bounds every chain of writes. If D1 and D2 fire each other through R, each runs once and then stops. If running workflows A and B signal each other through R, the ninth write fails and that run stops.
- **Idempotent starts.** A retried activity that writes R again cannot start D again. This adds to D-3-13's per-step idempotency key, which already stops the write itself from being applied twice.
- **Concurrency.** If two transactions claim the same key, the second waits on the first, on R's row lock (A5-1) and on the uncommitted key. If the first commits, the second gets no row and the start is suppressed. If the first rolls back, the second claims the key. Shown by spike.
- **Growth.** One `cascade_start` row per workflow start, about 200 B with its PK entry (estimated, not measured). Rows are kept indefinitely in M1, because records are never deleted (`records` has no `DELETE` grant or policy, and `internal/purge` does not touch it). A future record purge must delete a record's `cascade_start` rows first, through a `nava_purge` `DELETE` policy, because the FK is `NO ACTION`.
- **Determinism.** The claim runs in the Go core's write transaction, not in workflow code. A5-7's causation is computed from deliveries already in the workflow's history, so replay yields the same values.

## Invariants

| Invariant | Effect |
|---|---|
| I1 tenant wall | `cascade_start` is a tenant table: `tenant_id` leads the key, RLS is `FORCE`d, and the policies use the CI-1 predicate. The composite FK `(tenant_id, record_id)` blocks a cross-tenant reference, because FK checks bypass RLS. CI-1 and CI-4 cover the table once it is in `db/grants.yaml`. |
| I7 idempotency | Strengthened: a start is idempotent per (tenant, origin, record, definition). |
| I9 | Amended as above. Its intent (no unbounded cascades) is unchanged, and A5-7 closes the signal path. |
| I2–I6, I8 | Not touched. `cascade_start` uses no sequence (I8). |

## Rejected options

| Option | Why rejected |
|---|---|
| Keep the I9 text: reject the write when O is seen again on R | Rejects the S11b demo and every "on create or update, change this record" workflow. |
| Check on writes, keyed on `(O, R, trigger)` with `Causation.TriggerID` (Principal Software Engineer) | Two `update_record` steps on R, or a retried activity, write `(O, R, D)` twice and read as a cycle. It also adds a causation column to every write and to `audit_log`, which start-time keying does not need. |
| Check on starts, but reject the write on a hit | Every "on update, set X on this record" workflow fires itself again on its own write, so that write would always fail. |
| A unique index on `outbox` or `workflow_ref` | Both are purged, and neither has the definition id. The key must outlive the longest workflow. |
| An advisory lock plus a `SELECT` instead of a unique key | It works only while the lock is held. It leaves no durable record, so a later write under the same origin cannot see the earlier start. |
| `ClaimStart` returning only `(claimed bool, err error)`, after `Enqueue` | `outbox_id NOT NULL` cannot be filled first, and enqueuing first needs a savepoint to undo a hit (finding 2). |
| Stamp every interpreter write with the start's causation | Signal ping-pong never increases depth, so it never ends (finding 1). |

## Accepted residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS-A5-1 | A definition that legitimately needs to run twice on the same record within one origin cannot. Example: D1 and D2 each fire the other through R, and the second run of each is suppressed and audited. The workaround is a new user-originated write, which mints a new origin. The CTO accepted this for M1. | Solution Architect | A tenant use case needs re-entry within one origin. Candidate: a per-definition `reentrant` flag, still bounded by depth. |
| RS-A5-2 | Signal ping-pong between open runs is bounded by the depth limit (8, through A5-7), not by the start key. Up to 8 writes per origin can happen before the run fails. The Technical Architect accepts this with A5-7 in place. | Solution Architect | P3-36's performance run, or production, shows signal storms. |
| RS-A5-3 | A5-7's causation travels through Temporal history and the worker, so a compromised `nava-worker` could forge a lower depth. That is already inside AR-001 (R8): a stolen worker certificate can read and write any tenant. It adds no new reach. | Security & Multi-Tenancy Reviewer | AR-001 is revisited, or tenant-supplied code runs in the worker. |

## Recorded dissent

- **Principal Software Engineer** proposed carrying the trigger on writes (`Causation.TriggerID`, `audit_log.causation_trigger_id`). The CTO declined it as unnecessary under start-time keying (TICK-361, comment `01a1926e`).

## Evidence

- Spike in `docs/adr/ADR-001/spikes/solution-architect/rfc-005/` on `postgres:16.10`, 2026-10-31, run as a non-owner role under `FORCE ROW LEVEL SECURITY`. `spike.sql` and `output.txt` cover cases (a)–(c), D1 and D2 firing each other, a new origin, cross-tenant uuids (f), tenant B claiming in tenant A (rejected by the RLS policy), a NULL origin (not-null violation) and a rollback releasing the key. `concurrent.sh` and `concurrent-output.txt` cover two concurrent claims, ending in COMMIT and in ROLLBACK.
- Cases (d), (e) and (g) need the pipeline or the interpreter, so they are TICK-88's and P3-42's tests. The outbox side of (e) exists today at `internal/outbox/outbox_integration_test.go:357`. No spike is needed for (g), which settles finding 1 mechanically (Technical Architect).
- Technical Architect critique round 1, TICK-361 comment `01a19279`: A5-2, lock ordering, determinism and RLS confirmed sound.
- RFC-005 rev 1 is archived at `docs/adr/ADR-001/rfc/RFC-005-a5-i9-cycle-suppress-start.md`, with `rev-0/` and `rev-1/` beside it.
