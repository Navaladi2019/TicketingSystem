---
type: rfc
id: RFC-019
title: ADR-004 erratum E9, the audit row for a create refused with number_space_exhausted
status: REVISED (rev 1)
date: 2026-11-14
issue: TICK-864
source: TICK-674 (P6-09), Principal Software Engineer comment 01a1db26; ADR-005 §3.5
amends: "[[ADR-004-audit-log]] R2 (`action`), adds R2b; acceptance test CI-A18"
precedent: "[[ADR-004-E1-cascade-entries-and-diff-truncated]] R2a, `cascade_too_deep`"
gate: "ADR acceptance requires spike S1 (Technical Architect) to pass"
---

# RFC-019: ADR-004 E9, the `number_space_exhausted` audit row

> **Status: rev 1.** This revision resolves every round-1 item. Round 1 results:
> - Security & Multi-Tenancy Reviewer: APPROVE (comment `01a1db2e`).
> - Postgres & Data Modeling Specialist: APPROVE (comment `01a1db33-060c`).
> - Technical Architect: CHANGES REQUESTED (comment `01a1db31-5d54`). Asks 1 to 4 are non-blocking; ask 5 gates acceptance on S1.
>
> **Acceptance gate.** The ADR is **not accepted until spike S1 passes** (see "Gate: spike S1"). If S1 fails, F1 (shared allocator-pool contention) becomes a blocking item and the audit row's runner pool is redesigned.
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

## Decision (proposed)

**R2, `action`, new text:** {create, update, publish, enable, disable, delete, promote, rollback, state_change, export, erase, reseed, cascade_cycle, cascade_too_deep, **number_space_exhausted**}.

No new subject type, envelope column or index.

**R2b · the refused-create entry (new rule).**

| Field | Value |
|---|---|
| Where it is written | Through `audit.Append` on the pipeline's `TxRunner`, in a transaction of its own that commits before `Write` returns. This is the `rejectTooDeep` mechanism (`internal/records/pipeline/pipeline.go:549`). |
| `domain` | `record` |
| `subject_type` / `subject_id` | `entity_definition` / D, the type the create targeted, already resolved under RLS in the prefetch |
| `entity_definition_id` | D |
| `version_before` / `version_after` | NULL / NULL |
| `causation_id` / `causation_depth` | the refused command's |
| actor, `source`, `source_ref`, `request_id` | the refused command's |
| `content_sha256` | NULL (record domain; R3 does not apply) |
| `diff` | one `add` of `/number_space`: `{epoch, scope_key, width, counter}` |

**R2b also requires the following [TA Q1, F2, F3]:**
- **One entry per refused create, in its own runner transaction.** Each refused create opens its own runner transaction (`TxRunner.InTenantTx`) holding exactly that one entry. The entry is never queued on:
  - the create's transaction;
  - the allocation's `AllocTx`;
  - a batch shared with another refusal.
- **A time bound.** The refusal's runner transaction is bounded by `DefaultAllocTimeout`. Past it, the caller gets the refusal joined with the timeout, as for any other audit failure (CI-A18 f).
- **Two refusal sites, one shape.**

  | Site (PR #679) | Mode | `counter` source | Caller transaction state |
  |---|---|---|---|
  | `pipeline.go` after `allocate`, at the `Fits` check | `gaps` | `alloc.Number` (already committed on `AllocTx`) | still usable |
  | `batch.go` `writeCreate`, `gapless && numbered.N > 0 && !format.Fits(...)` | `gapless` | `numbered.N` | aborted. It holds the `record_counters` row lock until the caller rolls back. The runner transaction never touches `record_counters`, so no deadlock is possible. |

  `ErrNumberSpaceExhausted` today carries `{EntityKey, ScopeKey, Width}`. The follow-up adds `Epoch` and `Counter`, or passes them alongside. `InsertAtCommit` (TICK-692) is reached only after `Fits`, so it is not a third site.
- **The diff keys.**
  - `epoch` is the format's epoch (ADR-005 §3.3).
  - `scope_key` is the counter key (`e<epoch>|<parts>`, at most 128 characters by CHECK).
  - `width` is the `{seq:N}` width.
  - `counter` is the value that did not fit.
- **What the entry leaves out.** It never carries the attempted record data, the candidate field values, or the idempotency key. It names the type and the scope only.
- **The log line** for a refusal carries the tenant and the type, never `scope_key` [Sec S4].
- **Audit write failures.** If the audit write fails or times out, the caller gets the refusal joined with the audit error, exactly as with `rejectTooDeep`. The refusal is never reported as a success, and the create never commits.
- **Why the unique key holds.** `audit_log_commit_key` is `UNIQUE NULLS NOT DISTINCT (tenant_id, occurred_at, txid, subject_type, subject_id, version_after)`.
  - Within one runner transaction it reduces to (`entity_definition`, D, NULL), and that transaction holds one entry.
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

**Recommendation (unchanged): one row per refused create, with no cap and no dedupe. Acceptance depends on S1.**

- **The cost of a refusal is not the same as a success on the shared allocator pool** [TA F1].
  - The pipeline's `TxRunner` **is** the allocator pool: `internal/server/api.go:115`, `allocatorMaxConns = 4`, shared by all tenants.
  - Allocator transactions per create:

    | Mode | Success | Refusal |
    |---|---|---|
    | `gaps` | 1 | 2 (the committed allocation, plus the audit row) |
    | `gapless` | 0 | 1 |

  - Audit-log volume is the same as a success: one row per attempt. Allocator-pool load is up to 2× in `gaps` mode and goes from 0 to 1 in `gapless` mode.
- **Upstream limits bound the tenant's volume, not the shared pool.** HTTP creates pass the per-tenant token bucket (50 rps, burst 100, per replica; `internal/platform/ratelimit/ratelimit.go:38-39`). The rows stay in one tenant (I1 holds for data). Pool contention can still reach other tenants, and S1 measures whether it does.
- **Each row is a real refused attempt.** This keeps E1's rule for `cascade_too_deep` (RS-E1-2).
  - In `gaps` mode `counter` rises on every attempt, so the rows show the retry count.
  - In `gapless` mode every row of a scope shows the same `counter` [TA F4].
- **A dedupe needs state we do not want.**
  - A "last audited" column on `record_counters` puts audit bookkeeping on the hottest row.
  - A per-process throttle is per replica, loses its state on restart, and breaks one-entry-per-attempt.
  - If S1 fails, the fix is pool isolation, not a dedupe [TA F7].
- **Visibility.**
  - A metric, `records_number_space_exhausted_total`, with no tenant or definition label.
  - The existing 90 % alert (ADR-005 §3.5).
  - The tenant and type go in the log line only.

## Gate: spike S1 (Technical Architect)

**Hypothesis.** Tenant A retries refused `gaps` creates at its full rate (50 rps sustained, burst 100, one replica), with the R2b row on the allocator pool. Meanwhile, tenant B's creates still meet ADR-005 A5.

**Pass:**
- tenant B's `nava_numbering_alloc_seconds` p95 ≤ 200 ms;
- `nava_numbering_alloc_timeouts` = 0;
- B's p95 within 1.5× its baseline, measured with A idle.

The run uses the compose stack with the pool at 4 connections, against PR #679 plus an R2b stub.

**Fail:** any B timeout, or a p95 above 1.5× baseline. F1 then becomes blocking. The row moves off the allocator pool, either to a small dedicated audit-runner pool or under a per-tenant cap on in-flight refusal audits. It does not go on the API pool, because that would bring back the TICK-657 starvation. That redesign comes back as a revision with the numbers.

The report and its scripts are archived under `docs/adr/ADR-004/spikes/technical-architect/rfc-019-s1/` with the ADR PR.

## Rejected options

| Option | Why rejected |
|---|---|
| Subject = a scope key, UUIDv5 over (T, D, epoch, `scope_key`), new subject type `number_scope` | New vocabulary and a new namespace for a subject that is not a row. The `scope_key` would still be needed in `diff`. |
| Subject = `cascade_key` / K(T, O, nil, nil) | It ties the row to a cascade origin when most refused creates are not cascades, and it leaves the row out of D's history. |
| Write the row inside the create's transaction | The create rolls back or never inserts, so the row would be lost. |
| Reuse `create` with a `refused` flag in `diff` | The commit check pairs `create` entries with a record version. A `create` entry with no record breaks that meaning. |
| Dedupe on the counter row, or a per-process throttle | See question 2. |
| Hash or drop `scope_key` | See question 1. |
| Write the row on the API pool | It brings back the TICK-657 starvation. |

## Accepted residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS-E9-1 | A client that retries a refused create writes one row per attempt, until an admin publishes a wider format. Each `gaps` refusal costs two allocator-pool transactions. | Technical Lead | Any of: these rows exceed 10 % of a tenant's daily audit volume; a tenant's seal lag (R16) shows them; allocator timeouts appear during a refusal burst. |
| RS-E9-2 | A field masked later still shows its code in earlier refusal rows. These rows are never field-masked. Covered by AR-TICK-653-01, whose text is widened. | Security & Multi-Tenancy Reviewer | As for AR-TICK-653-01, **or reference-code tokens ship**. |

## Invariants

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. The row is written with `audit.Append` in `InTenantTx(ctx, tenant)`, with the tenant from `tenancy.TenantOf(tx)`, and D is read under RLS [Sec S1]. Cross-tenant pool contention is a noisy-neighbour risk, not a data-isolation gap; S1 gates it. |
| I4 audit in the same transaction | Holds. The create commits nothing, so the same-transaction rule has nothing to cover; this is E1's argument for `cascade_too_deep`. |
| I5 masks | Holds through ADR-005 §3.4 rules 3 and 4. The row is not field-masked, and later masks fall under AR-TICK-653-01. |
| I7 idempotency | Holds. A refusal rolls back the claim on its idempotency key, so a retry is a new attempt with a new entry [TA F5]. |
| I8 platform-owned vocabulary | Holds. One action, added by platform migration. |
| I2, I3, I6, I9 | Not touched. |

## Acceptance test

**CI-A18:**
- (a) A `refuse` format whose counter is full. One create:
  - returns 409 `number_space_exhausted`;
  - writes no record;
  - writes exactly one `number_space_exhausted` entry, with subject (`entity_definition`, D), `version_before` and `version_after` NULL, and a `/number_space` diff with the four keys and no record data.
- (b) Gaps mode: five refused creates write five entries, and `counter` rises by one each time.
- (c) Gapless mode: five refused creates write five entries, **each with the same `counter`** (max + 1), and the counter row is unchanged after the refusals.
- (d) Tenant B's audit log has no entry from tenant A's refusals.
- (e) An insert with `action = 'number_space_exhausted'` fails before the migration and succeeds after it.
- (f) A failing or timed-out audit write returns the refusal joined with the audit error, and no record exists.
- (g) [TA] Two creates of D in **one** caller transaction, both refused in gaps mode, write two entries with distinct `txid`, and no `ErrDuplicateEntry`.
- (h) [Sec S4] A principal with `audit.read` and a field mask on D reads the R2b row. The diff has exactly the four keys and no record payload path, and the refusal's log line has no `scope_key` attribute.

## Consequences

**Follow-up task, TICK-674's audit-row task (waits on E9):**
- R2b at both refusal sites, with the `DefaultAllocTimeout` bound, and `Epoch` and `Counter` added to `ErrNumberSpaceExhausted`.
- Migrations A and B.
- The metric.
- CI-A18 (a) to (h).
- **[PG 1]** `TestP510bSubjectTypesMirrorRegistry` (`internal/db/dbtest/audit_log_test.go:182`) compares `audit_log_action_check` with `audit.Actions()` instead of a hard-coded list.
- **[PG 2]** Migration A's Down raises an exception when `EXISTS (SELECT 1 FROM audit_log WHERE action = 'number_space_exhausted')`. Otherwise it re-narrows as the `130000` Down does.
- The Technical Lead confirms whether the 409 is non-retryable for any future non-HTTP create path.

**On acceptance:**
- `docs/adr/ADR-004-E9-number-space-exhausted-audit-row.md` and its `.html` explainer.
- This RFC archived under `docs/adr/ADR-004/rfc/` with `rev-0/` and `rev-1/`.
- The spikes under `docs/adr/ADR-004/spikes/postgres/rfc-019/` and `docs/adr/ADR-004/spikes/technical-architect/rfc-019-s1/`.
- AR-TICK-653-01 text and revisit triggers widened in `docs/security/accepted-risks.md`.
- The handbook page for ADR-004 errata is named in the PR description for the Technical Lead.

**TICK-674 (P6-09)** merges without the row, as agreed. Story S18 (TICK-669) is not accepted until the follow-up task is done.
