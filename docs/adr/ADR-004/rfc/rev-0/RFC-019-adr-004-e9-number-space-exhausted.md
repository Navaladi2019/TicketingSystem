---
type: rfc
id: RFC-019
title: ADR-004 erratum E9, the audit row for a create refused with number_space_exhausted
status: DRAFT (rev 0)
date: 2026-11-14
issue: TICK-864
source: TICK-674 (P6-09), Principal Software Engineer comment 01a1db26; ADR-005 §3.5
amends: "[[ADR-004-audit-log]] R2 (`action`), adds R2b; acceptance test CI-A18"
precedent: "[[ADR-004-E1-cascade-entries-and-diff-truncated]] R2a, `cascade_too_deep`"
---

# RFC-019: ADR-004 E9, the `number_space_exhausted` audit row

> **Numbering.** RFC-018 is the highest number in `docs/`; this is RFC-019. ADR-004 has E1 to E8, so this is E9.
>
> **Invariants source.** The project description does not list the eight invariants. As in ADR-004 and E1, this RFC is checked against [[ADR-001-platform-blueprint]] §1 I1 to I9. Checks against a separate list of eight are incomplete.

## Plain-language summary

ADR-005 §3.5 says that a create refused because the number's width is full (`overflow: refuse`) is **audited**. ADR-004 R2 is a closed list of audit actions, and no action on that list fits a refused create. E1 fixed the same kind of gap for `cascade_too_deep`. This RFC adds one action and one row shape, built the same way. Nothing else in R2 changes.

## Decision (proposed)

**R2, `action`, new text:** {create, update, publish, enable, disable, delete, promote, rollback, state_change, export, erase, reseed, cascade_cycle, cascade_too_deep, **number_space_exhausted**}.

No new subject type. No new envelope column. No new index.

**R2b · the refused-create entry (new rule).**

| Field | Value |
|---|---|
| When and where | Written through `audit.Append` on the pipeline's `TxRunner`, in a transaction of its own that commits before `Write` returns. It is independent of the create's transaction, which rolls back (gapless) or never inserts (gaps). **One entry per runner transaction.** This is the same mechanism as `rejectTooDeep` (`internal/records/pipeline/pipeline.go:549`). |
| `domain` | `record` |
| `subject_type` / `subject_id` | `entity_definition` / D, the type the create targeted |
| `entity_definition_id` | D |
| `version_before` / `version_after` | NULL / NULL |
| `causation_id` / `causation_depth` | the refused command's (a cascade create can be refused too) |
| actor, `source`, `source_ref`, `request_id` | the refused command's |
| `content_sha256` | NULL (record domain, R3 does not apply) |
| `diff` | one `add` of `/number_space`: `{epoch, scope_key, width, counter}` |

- `epoch` is the format's epoch (ADR-005 §3.3). `scope_key` is the counter key (`e<epoch>\|<parts>`, at most 128 characters by CHECK). `width` is the `{seq:N}` width. `counter` is the value that did not fit.
- **The entry never carries the attempted record data**, the candidate field values, or the idempotency key. It names the type and the scope only.
- **If the audit write fails**, the caller gets the refusal joined with the audit error, exactly as `rejectTooDeep` does. The refusal is never reported as a success, and the create never commits.
- **Why the unique key holds.** Within one runner transaction the key reduces to (`entity_definition`, D, NULL). The runner writes one entry per transaction. No config entry can share the key: every config entry for D has a non-NULL `version_after` (R3), and it is never written in a runner transaction.

**Migration.** `audit_log_action_check` is dropped and re-added `NOT VALID`, then `VALIDATE`d in a second migration, the same two-step pattern as `20261109130000/130100_audit_log_subject_acl_policy*`. `audit.ActionNumberSpaceExhausted` is added to `internal/audit/entry.go`; the CHECK mirrors the Go list. The Postgres & Data Modeling Specialist reviews the migration pair.

## Triad question 1: the subject, and `scope_key` in `diff`

**Recommendation: subject = `entity_definition` / D, and keep `scope_key` unhashed in `diff`.**

Why D:
- **The fix lives on D.** The only remedy is to publish a wider format on D, which gives a new epoch. The refusal rows then sit in D's history, `(tenant_id, subject_type, subject_id, occurred_at DESC)`, next to the publish that fixes them. `domain = record` keeps them apart from D's config entries.
- **No new vocabulary (I8).** `entity_definition` is already in `audit_log_subject_type_check`. A scope subject would need a new subject type, a migration of that CHECK, and a UUIDv5 over (T, D, epoch, `scope_key`) with a new pinned namespace. A scope is not a row: `record_counters` has a composite key and no id. The only thing that UUID would hide is `scope_key`, and it would still have to be in `diff` for anyone to act on the row.
- **No record exists.** `record` is impossible. The `cascade_key` fallback of E1 was needed because a too-deep write has no other id anyone checked. Here D has been resolved under RLS before allocation, so it is a checked id.

Why `scope_key` is safe in `diff`:
- **Its parts are platform-built codes, not field values.** ADR-005 §3.4 rule 3 allows only enum properties, mapped to `^[A-Z0-9]{1,10}$` by a declared mapping. The date parts come from SQL `now()` (A7). No free text, no secret, and no record data can reach `scope_key`.
- **At refusal time no reader of the type has the field masked.** §3.4 rule 4 is re-checked on every mask-set change, and a mask on a referenced field needs a new format in the same publish. So the codes in `scope_key` are visible to every reader who can read records of that type.
- **Only `audit.read` holders see it**, tenant-scoped (R8, I1).
- **A later mask** leaves these rows showing the code. That is the case AR-TICK-653-01 already accepts for issued numbers and "audit rows". The row's text gains "and `number_space_exhausted` entries" in the same change; no new AR row.

Rejected: hashing `scope_key` or dropping it to `epoch` only. An admin could then not tell which branch or month is full, and the 409 already names the scope to the caller.

## Triad question 2: flood cap or one row per refusal

**Recommendation: one row per refused create, no cap and no dedupe.** Add a metric and a residual.

- **No amplification.** A refused create writes one audit row. A successful create also writes one audit row. A client loop of refusals therefore costs no more audit volume than the same loop succeeding, which we already accept.
- **Bounded upstream.** HTTP creates pass the per-tenant token bucket (`internal/platform/ratelimit`) before the pipeline. A full scope is per (tenant, type, scope), so the flood stays inside one tenant (I1).
- **Each row is a real refused attempt.** This keeps E1's rule for `cascade_too_deep` (RS-E1-2): a dropped row would make the log lie about how many creates were refused. In gaps mode `counter` rises on every attempt, so the rows themselves show the retry count.
- **A dedupe needs state we do not want.** A "last audited" column on the `record_counters` row puts audit bookkeeping on the hottest row of the create path. A per-process memory throttle is per replica, lossy on restart, and breaks one-entry-per-attempt. Both are a new component for a noise problem we have not seen.
- **Visibility without the log.** `records_number_space_exhausted_total`, a counter with no tenant or definition label (no unbounded cardinality), plus the existing 90 % alert (ADR-005 §3.5). The tenant and type go in the log line only.
- **Non-HTTP writers.** A workflow or rule create that gets `number_space_exhausted` must not retry it. Retrying cannot succeed until a new format is published. The Technical Lead confirms that the activity error mapping already treats the 409 as non-retryable. If it does not, that is a one-line change in the follow-up task, not part of this erratum.

## Rejected options

| Option | Why rejected |
|---|---|
| Subject = a scope key, UUIDv5 over (T, D, epoch, `scope_key`), new subject type `number_scope` | New vocabulary and a new pinned namespace for a subject that is not a row. The `scope_key` would still be needed in `diff`. |
| Subject = `cascade_key` / K(T, O, nil, nil), as the E1 fallback | Ties the row to a cascade origin when most refused creates are not cascades, and leaves it out of D's history. |
| Write the row inside the create's transaction | The create rolls back or never inserts, so the row would be lost. |
| Reuse `create` with a `refused` flag in `diff` | `create` entries are what the commit check pairs with a record version. A `create` with no record breaks that meaning. |
| Dedupe on the counter row, or a per-process throttle | See question 2. |
| Hash or drop `scope_key` | See question 1. |

## Accepted residuals

| # | Residual | Owner | Revisit when |
|---|---|---|---|
| RS-E9-1 | A client or workflow that retries a refused create writes one row per attempt, until an admin publishes a wider format. | Technical Lead | `number_space_exhausted` rows exceed 10 % of a tenant's daily audit volume, or a tenant's seal lag (R16) shows it. |
| RS-E9-2 | A field later masked still shows its code in earlier refusal rows. Covered by AR-TICK-653-01, whose text is widened. | Security & Multi-Tenancy Reviewer | As AR-TICK-653-01. |

## Invariants

| Invariant | Effect |
|---|---|
| I1 tenant wall | Holds. `audit.Append` with the tenant from `tenancy.TenantOf(tx)`; D is read under RLS; the flood is bounded per tenant. |
| I4 audit in the same transaction | Holds. The create commits nothing, so there is nothing for the same-transaction rule to cover; same argument as E1 for `cascade_too_deep`. |
| I5 masks | Holds through ADR-005 §3.4 rules 3 and 4; later masks under AR-TICK-653-01. |
| I7 idempotency | Holds. Each refused attempt is one entry. A replayed idempotency key that hit a refusal stored no write key, so it is a new attempt. |
| I8 platform-owned vocabulary | Holds. One action, by platform migration. |
| I2, I3, I6, I9 | Not touched. |

## Acceptance test

**CI-A18.** (a) A `refuse` format whose counter is full: one create returns 409 `number_space_exhausted`, writes no record, and writes exactly one `number_space_exhausted` entry with subject (`entity_definition`, D), `version_before`/`version_after` NULL, and a `/number_space` diff with the four keys and no record data. (b) Five refused creates write five entries; in gaps mode `counter` rises by one each time. (c) Gapless mode: the same, and the counter row is unchanged after the refusals. (d) Tenant B's audit log has no entry from tenant A's refusals. (e) An insert with `action = 'number_space_exhausted'` succeeds after the migration and fails before it. (f) A failing audit write returns the refusal joined with the audit error, and no record exists.

## Delivery impact

- TICK-674 (P6-09) merges without the row, as agreed. The follow-up task waits on E9 and implements R2b, the migration pair, the metric and CI-A18.
- On acceptance: `docs/adr/ADR-004-E9-number-space-exhausted-audit-row.md`, its `.html` explainer, and this RFC archived under `docs/adr/ADR-004/rfc/`. AR-TICK-653-01 text widened in `docs/security/accepted-risks.md`.

## Open questions for reviewers

1. **Technical Architect:** is the unique-key argument complete? In particular, can any path write a runner transaction with two refusals for the same D (bulk or import)? If yes, R2b needs "one entry per refused command, one command per runner transaction", stated as a test.
2. **Postgres Specialist:** confirm the `NOT VALID` / `VALIDATE` pair for the action CHECK on the partitioned `audit_log`, as for the subject CHECK.
3. **Security & Multi-Tenancy Reviewer:** confirm that widening AR-TICK-653-01 is enough, rather than a new AR row.
