---
type: adr-erratum
id: ADR-004-E9
title: Erratum E9 to ADR-004, the R2 action and row shape for a create refused with number_space_exhausted
status: ACCEPTED
accepted: 2026-11-14
date: 2026-11-14
issue: TICK-864
rfc: docs/adr/ADR-004/rfc/RFC-019-adr-004-e9-number-space-exhausted.md (final, rev 1 with the CTO tie-break folded in; archived with rev-0/ and rev-1/; thread under TICK-864 comment 01a1db27-da9e-7244-93b2-71a762188381)
supersedes: "the R2 `action` list in [[ADR-004-audit-log]] (as extended by [[ADR-004-E1-cascade-entries-and-diff-truncated]])"
amends: "[[ADR-004-audit-log]] R2, adds R2b and acceptance test CI-A18; ADR-005 §3.5 ('audited') by reference; docs/budgets.md (allocator-pool rule)"
source: "TICK-674 (P6-09), Principal Software Engineer comment 01a1db26: ADR-005 §3.5 requires the refusal to be audited, and R2 has no action for it."
deciders: Solution Architect (author, verdict); Technical Architect (round 1 CHANGES REQUESTED, comment 01a1db31-5d54; spike S1 FAIL, comment 01a1db42-ff0a); Security & Multi-Tenancy Reviewer (round 1 APPROVE, comment 01a1db2e); Postgres & Data Modeling Specialist (round 1 APPROVE with spike, comment 01a1db33-060c); CTO (tie-break on F1, comment 01a1db43-638a)
---

# ADR-004 erratum E9: the audit row for a create refused with `number_space_exhausted`

> **Status: ACCEPTED, 2026-11-14.** The Solution Architect accepted it on TICK-864 after the placement checks on PR #680:
> - the Technical Architect approved, conditional on N1 (comment `01a1db4a-5cf6`). N1 is folded in word for word; it is the CTO's ruling `01a1db44-4d21`;
> - the Security & Multi-Tenancy Reviewer approved and found no isolation gap (comment `01a1db4a-6690`). N1 also closes P3;
> - the Postgres & Data Modeling Specialist asked for one sentence on `docs/budgets.md`, which is folded in (comment `01a1db4a-6d52`).
>
> The contested point, F1, was settled by the CTO tie-break (`01a1db43-638a`, corrected by `01a1db44-4d21`). The Solution Architect did not self-approve a contested design. No human sign-off is required under the current policy.
>
> **History.**
> - Round 1 on rev 0:
>   - Security approved; S2–S4 were non-blocking.
>   - Postgres approved, with a spike; items 1–3 were non-blocking.
>   - The Technical Architect asked for changes (asks 1–4) and gated acceptance on spike S1 (ask 5).
> - Rev 1 resolved every round-1 item.
> - S1 failed for the rev 1 placement: the row on the allocator pool.
> - The CTO ruled F1 blocking and adopted option 1, the row on the caller's pool after the create's transaction ends. This was measured at 5 of 5. No dissent was recorded.
>
> **Invariants source.** The project description defines no invariants. As in ADR-004 and E1–E8, this erratum is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. Against any separate list of eight, these checks are incomplete.

## Context

- ADR-005 §3.5: under `overflow: refuse`, a create whose counter no longer fits the `{seq:N}` width fails with `number_space_exhausted` (409), naming the type and scope, and it is **audited**.
- ADR-004 R2 is a closed list of actions. No action on it fits a refused create.
- E1 R2a set the precedent: the `cascade_too_deep` row for a refused write, written in a transaction of its own.

## Decision

**R2, `action`, new text:** {create, update, publish, enable, disable, delete, promote, rollback, state_change, export, erase, reseed, cascade_cycle, cascade_too_deep, **number_space_exhausted**}.

There is no new subject type, no new envelope column and no new index.

**R2b · the refused-create entry (new rule).**

| Field | Value |
|---|---|
| Where it is written | Through `audit.Append`, in a transaction of its own, after the refused create's transaction has ended (see below). |
| `domain` | `record` |
| `subject_type` / `subject_id` | `entity_definition` / D, the type the create targeted, resolved under RLS in the prefetch |
| `entity_definition_id` | D |
| `version_before` / `version_after` | NULL / NULL |
| `causation_id` / `causation_depth` | the refused command's |
| actor, `source`, `source_ref`, `request_id` | the refused command's |
| `content_sha256` | NULL (record domain; R3 does not apply) |
| `diff` | one `add` of `/number_space`: `{epoch, scope_key, width, counter}` |

**Placement [CTO ruling, spike S1]:**
- The entry is written in a transaction of its own **on the caller's (API) pool**, **after** the refused create's transaction has ended.
- It is never written on the allocator pool, and never while a create transaction is open.
- `Write` returns the refusal carrying the entry. The caller (`internal/server/records.go`, or any later create path) appends it with `tenancy.InTenantTx`, using the token's tenant. It then returns the refusal joined with any audit error.
- The caller has released its create connection before it asks for the audit connection. That rules out hold-and-wait, so the TICK-657 starvation cannot recur.
- The append is bounded by the request's own deadline and runs on `context.WithoutCancel` of the request context, so a client that disconnects after the refusal does not drop the entry (one entry per refused attempt, E1 RS-E1-2), and a stalled append never holds an API connection past the request.

**One entry per refused create.** Each refused create's entry is appended in its own transaction, which holds exactly that entry. It is never queued on any of these:
- the create's transaction;
- the allocation's `AllocTx`;
- any allocator-pool connection;
- a batch shared with another refusal.

**Two refusal sites, one shape** (PR #679):

| Site | Mode | `counter` | Caller transaction state at refusal |
|---|---|---|---|
| `internal/records/pipeline/pipeline.go`, after `allocate`, at the `Fits` check | `gaps` | `alloc.Number`, already committed on `AllocTx` | still usable |
| `internal/records/pipeline/batch.go` `writeCreate`, `gapless && numbered.N > 0 && !format.Fits(...)` | `gapless` | `numbered.N`, given back by the rollback | aborted; it holds the `record_counters` row lock until the rollback, and the append runs only after it |

- `ErrNumberSpaceExhausted` gains `Epoch` and `Counter`, next to `{EntityKey, ScopeKey, Width}`.
- `InsertAtCommit` (TICK-692) is reached only after `Fits`, so it is not a third site.

**Diff keys.**
- `epoch`: the format's epoch (ADR-005 §3.3).
- `scope_key`: the counter key, `e<epoch>|<parts>`, at most 128 characters by CHECK.
- `width`: the `{seq:N}` width.
- `counter`: the value that did not fit.

**What the entry never carries:** the attempted record data, the candidate field values, or the idempotency key. The refusal's log line carries the tenant and type, never `scope_key`.

**Failure.** If the append fails, the caller returns the refusal joined with the audit error. A refusal is never reported as a success, and the create never commits.

**Why the unique key holds.**
- `audit_log_commit_key` is `UNIQUE NULLS NOT DISTINCT (tenant_id, occurred_at, txid, subject_type, subject_id, version_after)`.
- In an audit transaction that holds one entry, the key reduces to (`entity_definition`, D, NULL).
- No config entry for D has a NULL `version_after` (R3).
- Today's create path is one create per HTTP transaction (`internal/server/records.go:149`), and workflow `ApplyStep` is update-only (`internal/workflow/interp/activities.go:212`).
- A future bulk or import path is safe only if it follows the one-entry-per-transaction rule. CI-A18 (g) pins that rule.

**`scope_key` in `diff` is safe because of ADR-005 §3.4, not because of the read path.**
- **The read path does not protect it.**
  - Field masks apply only to `subject_type = record` (`internal/audit/read/mask.go:253`), so these rows are never field-masked.
  - The audit timeline is not ACL-filtered (`internal/server/audit.go:82-85`).
- **ADR-005 §3.4 does protect it.**
  - Rule 3: `scope_key` holds only mapped enum codes (`^[A-Z0-9]{1,10}$`) and date parts computed in SQL.
  - Rule 4: no mask is live on those fields while the format is in use.
  - So `scope_key` shows no more than every issued number in that scope already prints.
- **A mask added later** is AR-TICK-653-01, whose text is widened.
- **Reference-code tokens** (deferred) reopen this decision; see RS-E9-2.

**Migration [PG].** There are two files. Each runs in its own goose transaction, and both set `SET LOCAL lock_timeout = '5s'`.
- **A** drops `audit_log_action_check` and re-adds it `NOT VALID`.
  - It changes the catalogue only: about 2 ms under `AccessExclusiveLock`.
  - It reaches every partition.
- **B** runs `VALIDATE`.
  - It holds only `ShareUpdateExclusiveLock`, at about 0.2 s per million rows.
  - It blocks neither reads nor audit writes.
- **The Go list.** `audit.ActionNumberSpaceExhausted` is added to `internal/audit/entry.go`, and the CHECK mirrors `audit.Actions()`.
- **`ATTACH PARTITION` must carry the current CHECK definition.**
  - Attaching a table with no action CHECK fails with `child table is missing constraint`. Attaching one with the old 14-value list fails with `different definition`.
  - Any future partition job uses `CREATE TABLE … PARTITION OF`, or copies the definition from `pg_get_constraintdef`. It never uses a hard-coded copy.

**Budget rule (`docs/budgets.md`).** Allocator-pool transactions are allocations only (`synchronous_commit = off`, about 2 ms). No synchronous-commit transaction runs on the allocator pool. Source: S1.

## Why this placement (spike S1)

- **What S1 tested.** Tenant A retried refused `gaps` creates at its full rate (50 rps, burst 100). The R2b row was on the 4-connection allocator pool, as in rev 1.
- **Result: rev 1 failed the agreed criterion in 1 of 7 reps.**
  - In that rep, tenant B's allocation p95 rose to **114 ms, 32× its baseline**.
  - A had 12 timeouts.
  - The audit-transaction p95 was 170 ms, during a stall, most likely a slow WAL flush (inferred: run 2 had no checkpoint counters).
- **Cause (known from the code).** A synchronously committed transaction (about 20 ms p95) held connections on a pool tuned for asynchronous commits of about 2 ms.
- **The measured alternative passed 5 of 5 reps.** It writes the row on the API pool after the create rolled back.
  - B's p95 was 2.99–3.71 ms, against a 3.15 ms baseline.
  - B's p99 was ≤ 5.11 ms.
  - B's max was 3.7–6.8 ms, against 16.8–30.6 ms with the row on the allocator pool.

## Allocator-pool cost per create

| Mode | Success | Refusal, rev 1 (rejected) | Refusal, this ADR |
|---|---|---|---|
| `gaps` | 1 | 2 | **1** |
| `gapless` | 0 | 1 | **0** |

The audit row costs one short API-pool transaction, taken after the create's connection is released.

## Rejected options

| Option | Why rejected |
|---|---|
| Subject = a scope key (UUIDv5 over T, D, epoch, `scope_key`), with a new subject type `number_scope` | It adds vocabulary and a namespace for a subject that is not a row, and `scope_key` would still be needed in `diff`. |
| Subject = `cascade_key` / K(T, O, nil, nil) | It ties the row to a cascade origin when most refusals are not cascades, and it leaves the row out of D's history. |
| Write the row inside the create's transaction | The create rolls back or never inserts, so the row would be lost. |
| Reuse `create` with a `refused` flag | The commit check pairs `create` entries with a record version. |
| Dedupe on the `record_counters` row, or a per-process throttle | The first puts audit state on the hottest row. The second is per replica and lossy, and it breaks one entry per attempt (E1 RS-E1-2). |
| Hash `scope_key`, or keep only `epoch` | The admin could not tell which scope is full, and the 409 already names it. |
| **The row on the allocator pool, inside `Write` (RFC-019 rev 1)** | S1 failed (32× p95 for another tenant). |
| A dedicated audit-runner pool (option 2) | Not measured. It adds a pool and a PgBouncer entry. |
| The row on the API pool while the create's transaction is open | Hold-and-wait on one pool: the TICK-657 starvation. |

## Accepted residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS-E9-1 | A client that retries a refused create writes one row per attempt until an admin publishes a wider format. Each refusal costs one short synchronous-commit API-pool transaction. | Technical Lead | Any one of: these rows exceed 10 % of a tenant's daily audit volume; seal lag (R16) shows them; API-pool acquire waits rise during a refusal burst. |
| RS-E9-2 | A field masked later still shows its code in earlier refusal rows, because these rows are never field-masked. This is covered by AR-TICK-653-01, whose text is widened. | Security & Multi-Tenancy Reviewer | As for AR-TICK-653-01, **or reference-code tokens ship**. |
| RS-E9-3 | `rejectTooDeep` (`internal/records/pipeline/pipeline.go:617` on the PR #679 head) still writes `cascade_too_deep` on the allocator pool, the placement S1 failed. E1 is not reopened: Milestone 1 scope stays narrow, and cascade depth (not client retries) bounds that row. The follow-up task may move it under the R2b placement if that costs one line. | Solution Architect | Cascade refusals are observed at a non-trivial rate. |

## Invariants

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. The tenant comes from the validated token through `tenancy.InTenantTx`, the same tenant `Write` checks against `tenancy.TenantOf(tx)`. D is read under RLS. The call site moves into `internal/server/records.go`; Security re-checks it before ACCEPTED. The placement and the budget rule close the noisy-neighbour risk on the shared allocator pool. |
| I4 audit in the same transaction | Holds. The create commits nothing, so the rule has nothing to cover. This is E1's argument for `cascade_too_deep`. |
| I5 masks | Holds through ADR-005 §3.4 rules 3 and 4. Later masks are covered by AR-TICK-653-01. |
| I7 idempotency | Holds. A refusal rolls back the claim on its idempotency key, so a retry is a new attempt with a new entry. |
| I8 platform-owned vocabulary | Holds. One action, added by a platform migration. |
| I2, I3, I6, I9 | Not touched. |

## Acceptance tests

| # | Proves |
|---|---|
| CI-A18 | (a) A full `refuse` counter: one create returns 409 `number_space_exhausted`, writes no record, and writes exactly one R2b entry (subject `entity_definition`/D, versions NULL, four diff keys, no record data). (b) In `gaps` mode, five refusals write five entries, and `counter` rises by one each time. (c) In `gapless` mode, five refusals write five entries with the same `counter` (max + 1), and the counter row is unchanged. (d) Tenant B's log has no entry from tenant A's refusals. (e) An `action = 'number_space_exhausted'` insert fails before the migration and succeeds after it. (f) A failing append returns the refusal joined with the audit error, and no record exists. (g) Two refused `gaps` creates of D in one caller transaction write two entries with distinct `txid` and no `ErrDuplicateEntry`. (h) An `audit.read` principal with a field mask on D reads exactly the four keys and no record payload path, and the log line has no `scope_key`. (i) No allocator-pool connection writes the row: with a `PoolRunner` whose pool is closed after allocation, the row is still written, and only after the create's transaction has ended. |

## Consequences

**Follow-up task (TICK-674's audit-row task, which waits on E9):**
- R2b at both refusal sites, with `Write` returning the entry and `records.go` appending it.
- `Epoch` and `Counter` on `ErrNumberSpaceExhausted`.
- Migrations A and B.
- The metric `records_number_space_exhausted_total`, with no tenant or definition label.
- CI-A18 (a)–(i).
- **[PG 1]** `TestP510bSubjectTypesMirrorRegistry` (`internal/db/dbtest/audit_log_test.go:182`) compares the action CHECK with `audit.Actions()`.
- **[PG 2]** Migration A's Down raises when `number_space_exhausted` rows exist.
- Optionally, if it costs one line, `rejectTooDeep` under the same placement.
- The append runs on `context.WithoutCancel(ctx)`, bounded by the request's deadline (N1).
- **[Sec P2]** CI-A18 (d) runs through `POST /api/v1/records/{entity}` with tenant A's token, then reads tenant B's log.
- **[Sec P3]** A test that a request cancelled right after the refusal still writes its R2b row.
- **[TA, (i)]** CI-A18 (i) also asserts that the row's `txid` differs from the create's `pg_current_xact_id()`, and that the refused request's API connection was released before the append acquired one. For `gapless`, close the allocator pool before the create.
- **[PG]** A compile-time or test guard for the allocator-pool rule: preferably split the allocator runner from `TxRunner` so `PoolRunner` on `allocPool` has no `InTenantTx`; otherwise a test that counts synchronous commits on `allocPool`. The Postgres & Data Modeling Specialist reviews it.

**This PR:**
- This file and its `.html` explainer.
- The RFC archived under `docs/adr/ADR-004/rfc/`, with `rev-0/` and `rev-1/`.
- The spikes under `docs/adr/ADR-004/spikes/`.
- The `docs/budgets.md` allocator-pool rule, with its `rejectTooDeep` exception; reviewed by the Postgres & Data Modeling Specialist.
- AR-TICK-653-01 widened in `docs/security/accepted-risks.md`.

**Handbook.** The Technical Lead adds the ADR-004 E9 page and its nav entry (`docs/handbook/src/adr/adr-004-e9.md`, `nav.yaml`).

**TICK-674 (P6-09)** merges without the row, as agreed. Story S18 (TICK-669) is not accepted until the follow-up task is done.

## Evidence

- `docs/adr/ADR-004/spikes/technical-architect/rfc-019-s1/`: `REPORT.md` and the spike code (`zz_spike_s1_integration_test.go.txt`). This is spike S1: PR #679 `111a3527`, `postgres:16.10`, durable settings, three runs including 5 reps comparing the rev 1 placement with the final one.
- `docs/adr/ADR-004/spikes/postgres/rfc-019/`: `spike.sql`, `concurrency.sh` and `out.txt`, from the Postgres & Data Modeling Specialist on `postgres:16.10` with 1M rows.
  - T1–T9: the `NOT VALID` / `VALIDATE` pair on the partitioned table, including `ATTACH PARTITION` (T6) and a bad-row `VALIDATE` (T8).
  - C1–C3: lock behaviour under concurrent writes.
- RFC-019 is archived at `docs/adr/ADR-004/rfc/RFC-019-adr-004-e9-number-space-exhausted.md`, with `rev-0/` and `rev-1/` beside it.
