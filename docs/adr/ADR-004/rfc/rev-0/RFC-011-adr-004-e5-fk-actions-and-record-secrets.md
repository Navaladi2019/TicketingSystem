---
type: rfc
id: RFC-011
title: "ADR-004-E5: no referential actions on audited tables (groups are tombstoned), and no secrets in record payloads"
status: DRAFT (rev 0)
date: 2026-11-04
issue: TICK-529
raised_from: TICK-452 (P5-13b), engineer's RESULT BLOCKED, comment 01a1a371-b984
target: ADR-004 erratum E5 (docs/adr/ADR-004-E5-*.md + ADR-004-E5-explained.html)
author: Solution Architect
---

# RFC-011: FK-action writes on audited tables, and the R6 `writeOnly` reading

> **Invariants source.** The TICK-529 project ("Ticket Management System") has **no description**, so it defines no invariants. As ADR-004 did, this RFC is checked against [[ADR-001-platform-blueprint]] §1, I1–I9, plus ADR-004's own decision 2 (the same-transaction rule). If the project is meant to carry a separate list of eight invariants, that list is missing and the invariant check below is incomplete against it.
>
> **Numbering.** RFC-010 is the highest RFC in `docs/adr/`, and ADR-004-E4 the highest erratum, so this is **RFC-011** targeting **ADR-004-E5**. Assumed, not confirmed against every issue thread.

## 1. Summary

- **Q1 → option (e), soft-delete variant.** Group sync stops hard-deleting groups: it **tombstones** them (`groups.deleted_at`), exactly as it already tombstones principals (`internal/groupsync/apply.go:253`). `records_assignment_group_fkey` drops `ON DELETE SET NULL` and becomes `ON DELETE RESTRICT`. Nothing outside the pipeline writes `records` any more, so the commit-check trigger attaches to `records` unchanged and group sync never touches an audited table.
- **The general rule (new R29):** no foreign key on an audited table may carry a referential action (`CASCADE`, `SET NULL`, `SET DEFAULT`) on delete or update. CI-A3 checks it from `pg_constraint`. This closes the class, not just this instance.
- **The real finding:** the `SET NULL` was already a second writer of `records` that bypasses I4 (no version bump, no audit, no outbox event). The trigger did not create the problem; it exposed it. Options (c) and (d) keep that writer and therefore break I4 and ADR-004 decision 2. They are rejected.
- **Q2 → No. Secrets never live in record payloads.** R6 is read as written: a value at an `x-nava-secret` or `writeOnly` path is refused. Storing it and masking it out of the diff is rejected. To stop this surfacing as a runtime surprise, a record definition may not list a marked property in `required` (refused at publish).
- **What the reviewer must accept:** a record whose group was removed in Keycloak keeps pointing at the tombstoned group (shown as removed) instead of being blanked; and every reader of `groups` must filter `deleted_at IS NULL`.

## 2. Decisions

| # | Decision | Why / what it prevents |
|---|---|---|
| **D1** | `groups` gains `deleted_at timestamptz NULL`. Group sync sets it (and `synced_at`) for a group Keycloak no longer has, deletes that group's `group_members` rows in the same transaction, and clears it (`deleted_at = NULL`) if the same `keycloak_group_id` comes back. It never issues `DELETE FROM groups`. | No write reaches `records` from group sync; sync can never stall on a record (the TICK-452 failure, `TestP309GroupDeleteClearsRecordGroup`). Mirrors the principals tombstone already in `apply.go`. |
| **D2** | `records_assignment_group_fkey` becomes `ON DELETE RESTRICT` (no action). `nava_groupsync` loses `DELETE` on `groups` in `db/grants.yaml`. | A hard delete of a referenced group now fails loudly instead of rewriting records behind the pipeline. Revoking the grant makes D1 enforced, not conventional. |
| **D3** | **R29 (new ADR-004 rule):** for every table in `db/audited-tables.yaml`, every FK whose referencing table is that table has `confdeltype` and `confupdtype` in {`a` NO ACTION, `r` RESTRICT}. CI-A3 asserts it from `pg_constraint`. | Referential actions run as the table owner, without the caller's grants or RLS (unverified detail, see §6), so CI-A3's privilege check cannot see them. R29 makes "the pipeline is the only writer" checkable for this whole class. |
| **D4** | Group memberships are deleted at tombstone, so a tombstoned group grants nothing. Every runtime reader of `groups` (group pickers, ACL group resolution, `/me` group paths) filters `deleted_at IS NULL`. A new isolation-registry case: a tombstoned group grants no record read and no admin/auditor role. | Keeps I5: a removed Keycloak group must not keep granting access through a pointer or a stale membership. |
| **D5** | The tenant admin/auditor pointers keep today's observable behaviour: when the recorded group is tombstoned, sync deletes the `tenant_admin_group` / `tenant_auditor_group` row in the same transaction (both tables are exempt directory tables; `nava_groupsync` gets `DELETE` on them only) and still reports `AdminGroupDeleted`. Their FKs stay `ON DELETE CASCADE` (the tables are not audited, so R29 does not apply). | Without it, the `CASCADE` that used to clear the pointer never fires on a soft delete, and a tombstoned admins group would stay "recorded". |
| **D6** | The pipeline's declarative-check step refuses a write that **changes** `assignment_group_id` to a tombstoned group: `422 assignment_group_removed`, naming the field. An **unchanged** value passes. | New assignments to a dead group are blocked; records already pointing at one can still be edited (otherwise they would be frozen). |
| **D7** | The record keeps `assignment_group_id` after its group is tombstoned. Read models return the group with `removed: true`. No automatic clearing in this erratum. | This is the honest state ("assigned to a group that no longer exists") and needs no write. Auto-reassignment, if product wants it, is an audited pipeline write (see §3, e2), decided separately. |
| **D8 (Q2)** | **Secrets may not live in record payloads.** R6 stands as written for records: a value at an `x-nava-secret` / `writeOnly` path is refused with `audit_secret_in_diff` (422, path named, value never). No "store and mask" mode. | `records.data` is plaintext to every reader the platform has: GraphQL/scripts (I8), outbox rows and Temporal payloads (I6), search/embeddings, backups and replicas. R7 masks only audit reads; it cannot follow the value anywhere else. |
| **D9 (Q2)** | S8 publish refuses a **record** entity definition that lists a marked property in `required` (`definition_secret_required`). The marker itself stays legal (it still keeps `default`/`const`/`enum`/`examples` out of the definition, `internal/entitytypes/audit.go`). | Otherwise a tenant can publish a type whose records can never be created. The marker without `required` documents an input the client must send elsewhere (a future secret reference, out of scope). |

## 3. Options considered (Q1)

| Option | What it does | Audit-trail effect | Verdict |
|---|---|---|---|
| **(a)** sync clears the group through the pipeline as a system write, then deletes | Audited, versioned, outbox event per record | Full coverage | **Rejected.** Couples identity sync to the record pipeline: one record failing its schema (e.g. a definition that now requires a group), a stale version or the cascade limit (I9) stalls the tenant's sync, deprovisioning included, which is the failure we are fixing. A group with 50k open records turns one sync transaction into 50k pipeline writes. `nava_groupsync` would need `records`, `audit_log` and `outbox` grants (I3: its role is DML on directory tables only, ADR-001 §A.3). |
| **(b)** `nava_groupsync` becomes a `records` writer and appends one entry per cleared record | Entries exist, but the FK action does not bump `version` | Coverage present but inconsistent: an entry whose `version_after` already has an entry; no outbox event | **Rejected.** A second write path for `records` outside the pipeline (I4), with no optimistic concurrency and no event for routing/SLA rules (I6). Widens `nava_groupsync` into the records context. |
| **(c)** trigger exempts FK-action updates (`WHEN (pg_trigger_depth() = 0)`) | Sync works; records change silently | **Breaks ADR-004 decision 2.** Scenario: R is v7 with group G (v7's diff sets `/assignment_group_id` to G). G is deleted: the row now holds NULL, still v7, no entry. Replaying R's diffs gives G; the row says NULL. The v8 entry's "before" silently disagrees with v7's "after". The chain still verifies, so **nothing detects the gap** ("history is complete" in CR-AUDIT-1 becomes false). Also exempts every future write made from inside any trigger. | **Rejected.** Also breaks I4: content changes with no version bump, so a client holding ETag v7 that re-sends G gets an FK error instead of `412`. Would need Security sign-off to accept a known blind spot. |
| **(d)** permanent S1 fallback for `records`; P3-13 arch test guards | Trigger never attached to `records` | **Worse than (c).** The arch test checks Go call sites, so it does not see the FK action either: the same silent gap as (c), *plus* no database enforcement for any raw write to `records` (CI-A2 permanently skipped for the most important audited table). | **Rejected.** S1 passed; the fallback exists for a performance miss, not for this. |
| **(e) RESTRICT + sync reassigns first** | Group cannot be deleted while referenced; sync rewrites records first | Covered only if the reassignment goes through the pipeline | **Rejected** in this form: it is (a) again, with the same stall. |
| **(e) RESTRICT + defer the delete** | Sync strips members but keeps the group row until unreferenced | No record write | **Rejected:** a live-looking ghost group stays in pickers and ACL configuration with no marker; a soft delete is the same with the state made explicit. |
| **(e) soft delete — chosen** | D1–D7 | **Full coverage, unchanged trigger.** No database-side writer of `records` remains; CI-A3 + R29 prove it. | **Chosen.** |
| (e2) soft delete + async audited clear | (e), then a `nava_worker` job clears or reassigns records of tombstoned groups through the pipeline (`source = system`) | Full coverage; each clear is a versioned, audited write with an outbox event | **Deferred, not rejected.** Product call (§7, Q-A). It composes with (e) without changing it. |

## 4. Invariants touched

| Invariant | Effect |
|---|---|
| ADR-004 decision 2 (same-transaction rule) | **Restored.** Trigger attaches to `records` with its current `UPDATE OF` list; R29 removes the class of writers it could not see. |
| I1 tenant wall | Unchanged. `groups` keeps its canonical policies; the FK keeps `tenant_id`. |
| I3 least privilege | **Tightened:** `nava_groupsync` loses `DELETE` on `groups`; gains `DELETE` only on the two exempt pointer tables (D5). It gains nothing on `records`, `audit_log` or `outbox`. |
| I4 one write pipeline | **Restored.** Today's `SET NULL` is an out-of-pipeline writer; (e) removes it. (a)/(b) would add one, (c)/(d) keep one. |
| I5 one authorization layer | Tombstoned groups grant nothing (D4); needs a new isolation case. |
| I6 outbox only | Unchanged under (e). (b)/(c)/(d) change routing-relevant data with no event. |
| I8 configuration, not code | Unchanged; D9 is a publish-time check, not tenant DDL. |
| I2, I7, I9 | Not touched. |

## 5. Acceptance tests (for the erratum's plan)

| # | Proves |
|---|---|
| CI-E5-1 | `TestP309GroupDeleteClearsRecordGroup` is replaced by: group sync drops G from Keycloak → `groups.deleted_at` set, G's memberships gone, records still reference G, **no `records` row changed (version, `updated_at` identical)**, sync commits, with the trigger **attached** to `records`. |
| CI-E5-2 | `DELETE FROM groups` as `nava_groupsync` is denied; as the owner, a delete of a referenced group fails (`RESTRICT`). |
| CI-E5-3 | CI-A3 / R29: a scratch audited table with an `ON DELETE SET NULL` FK fails the catalogue check (negative test); `records` passes. |
| CI-E5-4 | G returns in Keycloak with the same id → `deleted_at` cleared, memberships restored, records unchanged. |
| CI-E5-5 | Isolation: a member of tombstoned G reads no record through G; a tombstoned admins/auditors group grants no role and its pointer row is gone (`AdminGroupDeleted` still reported). |
| CI-E5-6 | Pipeline: changing `assignment_group_id` to tombstoned G → `422 assignment_group_removed`, nothing written; an update that leaves a tombstoned G unchanged succeeds and writes one audit entry. |
| CI-E5-7 | Q2: a record write with a value at a `writeOnly` property → 422 `audit_secret_in_diff`, pointer named, value absent from the response, logs, `records` and `audit_log` bytes (CI-A5 extended to records). Publishing a record definition with a marked property in `required` → `definition_secret_required`. |

## 6. Assumptions not verified

1. **Referential-action queries run as the referencing table's owner and skip RLS** (`ri_triggers.c` switches user and does not force RLS). This explains why the deferred trigger failed as `nava_groupsync` only at COMMIT. It does not change the recommendation; it is the reason a privilege-based CI-A3 cannot catch it. Verify by reading the PG 16 source or a 15-minute spike if a reviewer disputes it.
2. **Option (c)'s mechanism** (`WHEN (pg_trigger_depth() = 0)` on a deferred constraint trigger; the WHEN is evaluated at event time, inside the RI trigger, so depth ≥ 1). Not spiked, because (c) is rejected on invariant grounds, not on mechanism.
3. **The full list of `groups` readers** that need the `deleted_at IS NULL` filter. Found so far: `internal/groupsync/apply.go`, `internal/onboard/store.go`, `internal/identity/pgstore/pgstore.go`, isolation fixtures. The plan task must grep and list them.
4. **No existing record definition lists a marked property in `required`.** If one does (dev/CI data only, pre-release), D9 refuses its next publish, not its existing version.
5. Changing the FK on a large `records` table: use `ADD CONSTRAINT … NOT VALID` then `VALIDATE CONSTRAINT` to avoid a long lock. The Postgres Specialist confirms the migration shape.

## 7. Risks and accepted residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R-E5-1 | Open records stay assigned to a removed group, so they fall out of every group queue until someone reassigns them. The same records were invisible to the group under `SET NULL` too; the difference is that the pointer now says where they were. | Technical Lead (UI shows "removed group") | Product asks for auto-reassignment → (e2). |
| R-E5-2 | A pipeline write that read G live can commit just after sync tombstones G (no row lock: `nava_app` has no `UPDATE` on `groups`, so it cannot take `FOR SHARE`). End state = a record assigned before the tombstone, which D7 already accepts. | Solution Architect | Never, unless D7 changes. |
| R-E5-3 | `groups` keeps tombstoned rows forever. Low volume (directory groups). | Postgres Specialist | Tombstoned rows > 10% of `groups` for any tenant. Tenant offboarding (Epic 30) hard-deletes records before groups; `RESTRICT` enforces that order. |

## 8. Open questions

- **Q-A (product / human, not blocking):** should records assigned to a removed group be cleared or reassigned automatically? If yes, it is (e2): an audited pipeline write by a `nava_worker` job, a separate small RFC. If no answer, D7 stands.
- **Q-B (Security & Multi-Tenancy Reviewer):** confirm D4/D5 (tombstoned groups grant nothing; pointer tables cleared) and the grant changes in D2/D5. This touches the identity path, so a parallel review is warranted.
- **Q-C (Postgres & Data Modeling Specialist):** the migration shape (D1, D2, assumption 5) and the R29 catalogue query.
- **Q-D (Technical Lead):** sequencing. TICK-452 ships with `records` at `attached: false`; P5-13c then carries: migration (D1, D2), the groupsync change (D1, D5), reader filters (D4), the pipeline check (D6), CI-A3/R29 (D3), D9, then attaches the trigger. D9 is independent and can ship first.
