---
type: rfc
id: RFC-019
title: ADR-004 erratum E9, the audit row for a create refused with number_space_exhausted
status: FINAL (rev 1 with the CTO tie-break folded in) → ADR-004-E9
date: 2026-11-14
issue: TICK-864
source: TICK-674 (P6-09), Principal Software Engineer comment 01a1db26; ADR-005 §3.5
amends: "[[ADR-004-audit-log]] R2 (`action`), adds R2b; acceptance test CI-A18"
precedent: "[[ADR-004-E1-cascade-entries-and-diff-truncated]] R2a, `cascade_too_deep`"
gate: "Spike S1 FAILED for the rev 1 placement; CTO tie-break (comment 01a1db43-638a) adopted option 1, measured 5/5"
---

# RFC-019: ADR-004 E9, the `number_space_exhausted` audit row

> **CTO tie-break, folded in after rev 1 (comment `01a1db43-638a`).** Spike S1 failed for the rev 1 placement (row on the allocator pool). F1 is blocking, and option 1 is adopted. **The R2b row is written after the refused create's transaction has ended, in a transaction of its own on the caller's (API) pool, and never on the allocator pool.**
> - `Write` returns the refusal carrying the entry; the caller appends it.
> - The `DefaultAllocTimeout` bound (F2) is replaced by the request's own deadline, and the append runs on `context.WithoutCancel` (CTO correction `01a1db44-4d21`, N1 on PR #680).
> - CI-A18 (i) is added: no allocator-pool connection writes the row.
> - `docs/budgets.md` gains the rule "no synchronous-commit transaction runs on the allocator pool", citing S1.
> - `rejectTooDeep` keeps its current placement; that is accepted residual RS-E9-3.
>
> This was a ruling, not a new debate round. The sections below are rev 1 with the ruling applied. The rev 1 text as reviewed is archived in `rev-1/`.

> **Rev 1 status.** This revision resolved every round-1 item. Round 1 results:
> - Security & Multi-Tenancy Reviewer: APPROVE (comment `01a1db2e`).
> - Postgres & Data Modeling Specialist: APPROVE (comment `01a1db33-060c`).
> - Technical Architect: CHANGES REQUESTED (comment `01a1db31-5d54`). Asks 1 to 4 are non-blocking; ask 5 gates acceptance on S1.
>
> **Acceptance gate (rev 1).** Rev 1 gated the ADR on spike S1. S1 failed for the rev 1 placement, and the CTO ruling above resolves F1. See "Spike S1 result".
>
> **Numbering.** RFC-018 is the highest number in `docs/`; this is RFC-019. ADR-004 has E1 to E8, so this is E9.
>
> **Invariants source.** The project description does not list the eight invariants. As in ADR-004 and E1, this RFC is checked against [[ADR-001-platform-blueprint]] §1 I1 to I9. Checks against a separate list of eight are incomplete.

## Changelog, rev 0 → rev 1

| Item | From | Change |
|---|---|---|
| TA ask 1 / F1 | Technical Architect | Q2's "no amplification" bullet is replaced by the per-mode cost on the shared allocator pool. "Bounded upstream" now says it bounds the tenant's volume, not the shared pool. |
| TA ask 2 / Q1, F2, F3 | Technical Architect | R2b gains the one-entry-per-runner-transaction sentence, the `DefaultAllocTimeout` bound, and both refusal sites with their `counter` source. |
| TA ask 3 / F4 | Technical Architect | CI-A18 (c) asserts that every gapless entry has the same `counter`. New CI-A18 (g): two refusals in one caller transaction. |
| TA ask 4 | Technical Architect | "Non-HTTP writers" removed. No create path outside HTTP exists today; any future one must follow R2b as written. |
| TA ask 5 | Technical Architect | ADR acceptance depends on S1 (header and the "Gate" section). |
| S2 | Security | The reason `scope_key` is safe is corrected: the row is never field-masked, and it is safe only because of §3.4 rule 4 and because it shows no more than the issued number does. AR-TICK-653-01 wording updated. |
| S3 | Security | "Reference-code tokens ship" added as a revisit trigger to AR-TICK-653-01 and RS-E9-2. |
| S4 | Security | New CI-A18 (h): an `audit.read` reader with a mask on D sees exactly four keys. The log line carries no `scope_key`. |
| PG 1, PG 2 | Postgres | The mirror test against `audit.Actions()` and the guarded Down go into the follow-up task (Consequences). |
| PG 3 | Postgres | `ATTACH PARTITION` note added to the Migration section. |

## Plain-language summary

ADR-005 §3.5 says that a create refused because the number's width is full (`overflow: refuse`) is **audited**. ADR-004 R2 is a closed list of audit actions, and no action on that list fits a refused create. E1 closed the same kind of gap for `cascade_too_deep`. This RFC adds one action and one row shape, built the same way. Nothing else in R2 changes.

## Decision

**R2, `action`, new text:** {create, update, publish, enable, disable, delete, promote, rollback, state_change, export, erase, reseed, cascade_cycle, cascade_too_deep, **number_space_exhausted**}.

No new subject type, envelope column or index.

**R2b · the refused-create entry (new rule).**

| Field | Value |
|---|---|
| Where it is written | Through `audit.Append`, in a transaction of its own on the **caller's (API) pool**, **after** the refused create's transaction has ended. It is never written on the allocator pool, and never while a create transaction is open. `Write` returns the refusal carrying the entry. The caller (`internal/server/records.go`, or any later create path) appends it in `tenancy.InTenantTx` with the token's tenant, then returns the refusal joined with any audit error. [CTO ruling, S1] |
| `domain` | `record` |
| `subject_type` / `subject_id` | `entity_definition` / D, the type the create targeted, already resolved under RLS in the prefetch |
| `entity_definition_id` | D |
| `version_before` / `version_after` | NULL / NULL |
| `causation_id` / `causation_depth` | the refused command's |
| actor, `source`, `source_ref`, `request_id` | the refused command's |
| `content_sha256` | NULL (record domain; R3 does not apply) |
| `diff` | one `add` of `/number_space`: `{epoch, scope_key, width, counter}` |

**R2b also requires the following [TA Q1, F2, F3]:**
- **One entry per refused create, in its own transaction.** Each refused create's entry is appended in its own `InTenantTx` on the caller's pool, holding exactly that one entry. The entry is never queued on:
  - the create's transaction;
  - the allocation's `AllocTx`;
  - any allocator-pool connection;
  - a batch shared with another refusal.
- **No hold-and-wait.** The caller has released its create connection before it asks for the audit connection, so the TICK-657 starvation cannot recur. *(The rev 1 `DefaultAllocTimeout` bound is replaced:)*
- The append is bounded by the request's own deadline and runs on `context.WithoutCancel` of the request context, so a client that disconnects after the refusal does not drop the entry (one entry per refused attempt, E1 RS-E1-2), and a stalled append never holds an API connection past the request.
- **Two refusal sites, one shape.**

  | Site (PR #679) | Mode | `counter` source | Caller transaction state |
  |---|---|---|---|
  | `pipeline.go` after `allocate`, at the `Fits` check | `gaps` | `alloc.Number` (already committed on `AllocTx`) | still usable |
  | `batch.go` `writeCreate`, `gapless && numbered.N > 0 && !format.Fits(...)` | `gapless` | `numbered.N` | aborted. It holds the `record_counters` row lock until the caller rolls back. The audit append runs only after that rollback, so it never overlaps the lock. |

  `ErrNumberSpaceExhausted` today carries `{EntityKey, ScopeKey, Width}`. The follow-up adds `Epoch` and `Counter`, or passes them alongside. `InsertAtCommit` (TICK-692) is reached only after `Fits`, so it is not a third site.
- **The diff keys.**
  - `epoch` is the format's epoch (ADR-005 §3.3).
  - `scope_key` is the counter key (`e<epoch>|<parts>`, at most 128 characters by CHECK).
  - `width` is the `{seq:N}` width.
  - `counter` is the value that did not fit.
- **What the entry leaves out.** It never carries the attempted record data, the candidate field values, or the idempotency key. It names the type and the scope only.
- **The log line** for a refusal carries the tenant and the type, never `scope_key` [Sec S4].
- **Audit write failures.** If the caller's append fails, the caller returns the refusal joined with the audit error. The refusal is never reported as a success, and the create never commits.
- **Why the unique key holds.** `audit_log_commit_key` is `UNIQUE NULLS NOT DISTINCT (tenant_id, occurred_at, txid, subject_type, subject_id, version_after)`.
  - Within one audit transaction it reduces to (`entity_definition`, D, NULL), and that transaction holds one entry.
  - No config entry for D has a NULL `version_after` (R3), so none can collide.
  - Today's `Write` callers are one-create-per-HTTP-transaction (`internal/server/records.go:149`). The workflow `ApplyStep` path is update-only (`internal/workflow/interp/activities.go:212`).
  - A future bulk or import path is safe only if it follows the per-refusal rule above. CI-A18 (g) pins that rule.

**Migration [PG].**
- **Two files, each in its own goose transaction, both with `SET LOCAL lock_timeout = '5s'`.** This is the `20261109130000/130100` pattern.
  - Migration A drops `audit_log_action_check` and re-adds it `NOT VALID`. It is catalogue-only: about 2 ms under `AccessExclusiveLock`, and it reaches every partition.
  - Migration B runs `VALIDATE` under `ShareUpdateExclusiveLock` only. It took about 0.2 s per million rows, and it blocks neither reads nor audit writes. The spike is in `docs/adr/ADR-004/spikes/postgres/rfc-019/`.
- **Mirroring.** `audit.ActionNumberSpaceExhausted` is added to `internal/audit/entry.go`, and the CHECK mirrors `audit.Actions()`.
- **`ATTACH PARTITION` must carry the current CHECK definition** [PG 3].
  - Attaching a table with no action CHECK fails with `child table is missing constraint`. Attaching one with the old 14-value CHECK fails with `different definition`.
  - Nothing breaks today: production creates only the DEFAULT partition.
  - Any future partition-maintenance job must use `CREATE TABLE … PARTITION OF`, which inherits the constraint, or copy the definition from `pg_get_constraintdef`. It must never use a hard-coded copy of the list.

## Triad question 1: the subject, and `scope_key` in `diff`

**Recommendation (unchanged): subject = `entity_definition` / D, and keep `scope_key` in `diff` without hashing it.**

Why D:
- **The fix lives on D.** The only remedy is to publish a wider format on D, which starts a new epoch. Refusal rows sit in D's history next to the publish that fixes them, and `domain = record` keeps them apart from D's config entries.
- **No new vocabulary (I8).** A scope subject would need a new subject type and a UUIDv5 namespace for something that is not a row. `scope_key` would still have to be in `diff`.
- **No record exists, and D is a checked id.** E1 used the `cascade_key` fallback because it had no checked id. Here D is resolved under RLS before allocation [TA F6].

Why `scope_key` is safe in `diff`. This reason is corrected in rev 1 [Sec S2]:
- **The row is never field-masked.**
  - Field masks apply only to `subject_type = record` (`internal/audit/read/mask.go:253`), and the path `/number_space/scope_key` matches no `/<field>` anyway.
  - The audit timeline is not ACL-filtered (`internal/server/audit.go:82-85`), so every `audit.read` holder in the tenant sees the row, whatever types they can read.
  - The safety therefore does **not** come from the read path.
- **It is safe because of ADR-005 §3.4.**
  - Rule 3: `scope_key` holds only mapped enum codes (`^[A-Z0-9]{1,10}$`) and date parts computed in SQL, never raw field values or free text.
  - Rule 4: no mask is live on those fields while the format is in use.
  - So `scope_key` shows no more than every issued number of that scope already prints in `create` entries, outbox events and emails.
- **A mask added later.** Old `create` entries then get the field masked, but old R2b rows keep the code, as old numbers do. That is AR-TICK-653-01. Its text gains: *"and `number_space_exhausted` audit entries, which are never field-masked"*.
- **Reference-code tokens [Sec S3].** Today `scope_key` holds only `e<epoch>|` and date parts: field tokens have not shipped (`internal/records/numbering/numbering.go:183-220`). Reference-code tokens (§3.4 rule 3, deferred) would put another record's code into `scope_key` without the reference-read check. **"Reference-code tokens ship" is a revisit trigger** for AR-TICK-653-01 and RS-E9-2. That change must re-decide whether `scope_key` stays in `diff`.

Rejected: hashing `scope_key`, or keeping only `epoch`. An admin could then not tell which branch or month is full, and the 409 already names the scope to the caller.

## Triad question 2: flood cap or one row per refusal

**Recommendation (unchanged): one row per refused create, with no cap and no dedupe.** The CTO ruling places the row on the caller's pool.

- **Where a refusal uses connections** [TA F1, S1].
  - The allocator pool (`internal/server/api.go:115`, `allocatorMaxConns = 4`) is shared by all tenants. Its transactions commit asynchronously in about 2 ms (`internal/tenancy/alloc.go:31`).
  - The R2b row commits synchronously, at about 20 ms p95 in S1, roughly 10× as long.
  - Allocator-pool transactions per create:

    | Mode | Success | Refusal, rev 1 placement (rejected) | Refusal, final placement |
    |---|---|---|---|
    | `gaps` | 1 | 2 | **1** (the allocation only) |
    | `gapless` | 0 | 1 | **0** |

  - With the final placement, a refusal costs the allocator pool exactly what a success costs.
  - The audit row costs one short API-pool transaction. It is taken after the create's connection is released, the same pool and order as any later request.
- **Upstream limits bound the tenant's volume.** HTTP creates pass the per-tenant token bucket (50 rps, burst 100, per replica; `internal/platform/ratelimit/ratelimit.go:38-39`). The rows stay in one tenant (I1).
- **Each row is a real refused attempt.** This keeps E1's rule for `cascade_too_deep` (RS-E1-2).
  - In `gaps` mode, `counter` rises on every attempt, so the rows show the retry count.
  - In `gapless` mode, every row of a scope shows the same `counter` [TA F4].
- **A dedupe needs state we do not want.**
  - A "last audited" column on `record_counters` puts audit bookkeeping on the hottest row.
  - A per-process throttle is per replica, loses its state on restart, and breaks one-entry-per-attempt.
  - S1 confirmed that the problem was the pool, not the number of rows [TA F7].
- **Visibility.**
  - A metric, `records_number_space_exhausted_total`, with no tenant or definition label.
  - The existing 90 % alert (ADR-005 §3.5).
  - The tenant and type go in the log line only.

## Spike S1 result (Technical Architect, comment `01a1db42-ff0a`)

**Hypothesis.** Tenant A retries refused `gaps` creates at its full rate (50 rps, burst 100, one replica), with the R2b row on the allocator pool. Tenant B's allocations still meet ADR-005 A5.

**Criterion, agreed in advance.**
- Pass: B's p95 ≤ 200 ms, 0 B timeouts, and B's p95 ≤ 1.5× the A-idle baseline.
- Fail: anything else.

**Result: FAIL for the rev 1 placement.** The P2 phase met the criterion in 6 of 7 reps. In the failing rep (run 2):
- B's p95 was **114 ms, 32× its baseline**;
- A had 12 allocation timeouts;
- the audit-transaction p95 was 170 ms, from a slow WAL flush.

B had 0 timeouts in every phase.

**The measured alternative passed 5 of 5 reps.** P2-alt writes the row on the API pool, after the create rolled back.
- B's p95 was 2.99 to 3.71 ms, against a baseline p95 of 3.15 ms.
- B's p99 was ≤ 5.11 ms.
- B's max was 3.7 to 6.8 ms, against 16.8 to 30.6 ms with the rev 1 placement.

**Setup.**
- PR #679 head `111a3527`, with the real `PoolRunner.Allocate` and the real `audit.Append`.
- `postgres:16.10` with `fsync=on` and `synchronous_commit=on`, limited to 4 CPUs and 2 GB.
- Allocator pool of 4 connections, API pool of 20, no PgBouncer.

The report and the spike code are archived in `docs/adr/ADR-004/spikes/technical-architect/rfc-019-s1/`.

**Ruling.** CTO tie-break, comment `01a1db43-638a`:
- F1 is blocking.
- Option 1 (write on the API pool, after the create's transaction ends) is adopted.
- Option 2 (a dedicated audit pool) is rejected: it was not measured, and it adds a component.

**Budget rule (new, `docs/budgets.md`).** Allocator-pool transactions are allocations only (`synchronous_commit = off`, about 2 ms). No synchronous-commit transaction runs on the allocator pool. The source is S1. The Postgres & Data Modeling Specialist reviews this line.

## Rejected options

| Option | Why rejected |
|---|---|
| Subject = a scope key: UUIDv5 over (T, D, epoch, `scope_key`), new subject type `number_scope` | New vocabulary and a new namespace for a subject that is not a row. The `scope_key` would still be needed in `diff`. |
| Subject = `cascade_key` / K(T, O, nil, nil) | It ties the row to a cascade origin when most refused creates are not cascades. It also leaves the row out of D's history. |
| Write the row inside the create's transaction | The create rolls back or never inserts, so the row would be lost. |
| Reuse `create`, with a `refused` flag in `diff` | The commit check pairs `create` entries with a record version. A `create` entry with no record breaks that meaning. |
| Dedupe on the counter row, or a per-process throttle | See question 2. |
| Hash or drop `scope_key` | See question 1. |
| **The row on the allocator pool, inside `Write` (rev 1)** | S1 failed. A synchronously committed transaction runs on a 4-connection pool tuned for 2 ms asynchronous commits, so one WAL stall gave other tenants a 32× p95. |
| The row on a dedicated audit-runner pool (option 2) | Not measured. It adds a pool and a PgBouncer entry, and it would need its own spike. Option 1 has fewer parts and passed 5 of 5. |
| The row on the API pool while the create's transaction is still open | Hold-and-wait on one pool: the TICK-657 starvation. The final placement asks for a connection only after the create's connection is released. |

## Accepted residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS-E9-1 | A client that retries a refused create writes one row per attempt, until an admin publishes a wider format. Each refusal costs one short synchronous-commit transaction on the API pool. | Technical Lead | Any of: these rows exceed 10 % of a tenant's daily audit volume; a tenant's seal lag (R16) shows them; API-pool acquire waits rise during a refusal burst. |
| RS-E9-2 | A field masked later still shows its code in earlier refusal rows, because these rows are never field-masked. Covered by AR-TICK-653-01, whose text is widened. | Security & Multi-Tenancy Reviewer | As for AR-TICK-653-01, **or reference-code tokens ship**. |
| RS-E9-3 | `rejectTooDeep` (`internal/records/pipeline/pipeline.go:617` on the PR #679 head) still writes `cascade_too_deep` on the allocator pool, the placement S1 failed. E1 is **not** reopened. The row is rare, because cascade depth bounds it, not client retries. The follow-up task may move it under the R2b rule if that costs one line. | Solution Architect | Cascade refusals are observed at a non-trivial rate. |

## Invariants

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. The row is written with `audit.Append` in `tenancy.InTenantTx(ctx, pool, tenant)`, with the tenant from the validated token, the same tenant `Write` checks against `tenancy.TenantOf(tx)`. D is read under RLS [Sec S1]. **The call site moves into `internal/server/records.go`**, and the Security & Multi-Tenancy Reviewer re-checks that placement on the ADR. The placement and the budget rule close the cross-tenant pool contention, which is a noisy-neighbour risk. |
| I4 audit in the same transaction | Holds. The create commits nothing, so the same-transaction rule has nothing to cover. This is E1's argument for `cascade_too_deep`. |
| I5 masks | Holds, through ADR-005 §3.4 rules 3 and 4. The row is not field-masked, and later masks fall under AR-TICK-653-01. |
| I7 idempotency | Holds. A refusal rolls back the claim on its idempotency key, so a retry is a new attempt with a new entry [TA F5]. |
| I8 platform-owned vocabulary | Holds. One action, added by platform migration. |
| I2, I3, I6, I9 | Not touched. |

## Acceptance test

**CI-A18:**
- (a) A `refuse` format whose counter is full. One create:
  - returns 409 `number_space_exhausted`;
  - writes no record;
  - writes exactly one `number_space_exhausted` entry. The entry has subject (`entity_definition`, D), `version_before` and `version_after` NULL, and a `/number_space` diff with the four keys and no record data.
- (b) Gaps mode: five refused creates write five entries, and `counter` rises by one each time.
- (c) Gapless mode: five refused creates write five entries, **each with the same `counter`** (max + 1). The counter row is unchanged after the refusals.
- (d) Tenant B's audit log has no entry from tenant A's refusals.
- (e) An insert with `action = 'number_space_exhausted'` fails before the migration and succeeds after it.
- (f) A failing audit append returns the refusal joined with the audit error, and no record exists.
- (g) [TA] Two creates of D in **one** caller transaction, both refused in gaps mode, write two entries with distinct `txid` and no `ErrDuplicateEntry`.
- (h) [Sec S4] A principal with `audit.read` and a field mask on D reads the R2b row. The diff has exactly the four keys and no record payload path. The refusal's log line has no `scope_key` attribute.
- (i) [TA, CTO ruling] No allocator-pool connection writes a refused create's audit row. Use a `PoolRunner` whose pool is closed after allocation. The row must still be written, and only after the create's transaction has ended.

## Consequences

**Follow-up task, TICK-674's audit-row task (waits on E9):**
- R2b at both refusal sites:
  - `Write` returns the refusal carrying the entry;
  - `internal/server/records.go` appends the entry after `InTenantTx` returns;
  - `Epoch` and `Counter` are added to `ErrNumberSpaceExhausted`.
- Migrations A and B.
- The metric.
- CI-A18 (a) to (i).
- **[PG 1]** `TestP510bSubjectTypesMirrorRegistry` (`internal/db/dbtest/audit_log_test.go:182`) compares `audit_log_action_check` with `audit.Actions()` instead of a hard-coded list.
- **[PG 2]** Migration A's Down raises an exception when `EXISTS (SELECT 1 FROM audit_log WHERE action = 'number_space_exhausted')`. Otherwise it re-narrows as the `130000` Down does.
- Optional, only if it is one line: move `rejectTooDeep` under the same rule (RS-E9-3).
- The Technical Lead confirms whether the 409 is non-retryable for any future non-HTTP create path.

**In the ADR PR:**
- `docs/adr/ADR-004-E9-number-space-exhausted-audit-row.md` and its `.html` explainer.
- This RFC, in `docs/adr/ADR-004/rfc/` with `rev-0/` and `rev-1/`.
- The spikes, in `docs/adr/ADR-004/spikes/postgres/rfc-019/` and `docs/adr/ADR-004/spikes/technical-architect/rfc-019-s1/`.
- The allocator-pool rule in `docs/budgets.md`.
- The widened text and revisit triggers of AR-TICK-653-01, in `docs/security/accepted-risks.md`.
- The PR description names the handbook page for ADR-004 errata, for the Technical Lead.

**TICK-674 (P6-09)** merges without the row, as agreed. Story S18 (TICK-669) is not accepted until the follow-up task is done.
