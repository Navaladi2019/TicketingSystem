---
type: adr-erratum
id: ADR-004-E5
title: "Erratum E5 to ADR-004: no referential actions on audited tables (group sync tombstones groups), and no secrets in record payloads"
status: ACCEPTED
accepted: 2026-11-04 (Solution Architect verdict, TICK-529 comment thread 01a1a38e-7b5e)
date: 2026-11-04
issue: TICK-529
rfc: docs/adr/ADR-004/rfc/RFC-011-adr-004-e5-fk-actions-and-record-secrets.md (rev 1, archived with rev-0/ and rev-1/ and the review record; thread under TICK-529 comment 01a1a373-04ed-79e0-9c10-5756243cfdc2)
supersedes: none
amends: "[[ADR-004-audit-log]] decision 2 (new rule R29), R6 (reading for records), CI-A3; `db/migrations/20261101190000_records_pipeline.sql` `records_assignment_group_fkey`; the P3-04 rule on `tenant_admin_group` (narrowed, D5)"
source: "TICK-452 (P5-13b) RESULT BLOCKED: attaching the commit-check trigger to `records` makes group sync fail when it deletes a group a record names (`TestP309GroupDeleteClearsRecordGroup`)"
deciders: Solution Architect (author); Technical Architect (round 1 APPROVE, round 2 APPROVE with condition I1, comments 01a1a37f-0be0 and 01a1a387-f770); Security & Multi-Tenancy Reviewer (round 1 APPROVE, comment 01a1a379-c6df); Postgres & Data Modeling Specialist (round 1 CHANGES, resolved in rev 1, comment 01a1a37c-61e4); CTO (rulings on T8 vs S1 and on D5, comment 01a1a37f-7cbc; closure, comment 01a1a388-3ce4; N1–N4 ruling, comment 01a1a38e-7b5e); Security & Multi-Tenancy Reviewer (PR #431 APPROVE, comment 01a1a38d-fdb2)
---

# ADR-004 erratum E5: no FK actions on audited tables, and no secrets in records

> **Status: ACCEPTED** on 2026-11-04 by the Solution Architect (TICK-529). The Technical Architect approved in round 2, and its condition I1 is carried verbatim (D5, D10). The Postgres Specialist's blocking items PG-B1 and PG-B2 are resolved. Security approved in round 1 and again on PR #431 (comment 01a1a38d-fdb2). The contested points were settled by CTO tie-breaks, not by the author: D11 over S1's RLS check, and D5 kept in S2's form (comment 01a1a37f-7cbc). Security's non-blocking N1 is folded into D5; N3 and N4 are plan notes; N2 needs no change (CTO, comment 01a1a38e-7b5e). No human sign-off is required under the current policy.
>
> **Invariants source.** The project description defines no invariants. As the CTO ruled in round 1, this erratum is checked against [[ADR-001-platform-blueprint]] §1, I1–I9, and against ADR-004 decision 2.

## Context

`records.assignment_group_id` was declared `ON DELETE SET NULL` from `groups`. When group sync deletes a group that Keycloak no longer has, PostgreSQL itself rewrites every record that names it. The rewrite runs as the table owner, without the caller's grants or RLS (spike T1), so it skips the write pipeline: the version is not bumped, no audit entry is written and no outbox event is sent. That is a second writer of `records`, which breaks I4 and ADR-004 decision 2.

The commit-check trigger refused that write at COMMIT, so group sync failed. The tenant's sync, deprovisioning included, stopped while any record named the deleted group. CI-A3 did not see it, because it checks privileges, and an FK action needs none.

## Decision

### Q1: tombstone groups, and ban referential actions on audited tables

| # | Decision |
|---|---|
| **D1** | `groups` gains `deleted_at timestamptz NULL`. For a group that Keycloak no longer has, group sync works in its tenant transaction and in this order: (1) delete the group's `group_members` rows; (2) set `deleted_at = now()`; (3) delete its admin/auditor pointer if one exists (D5). If the same `keycloak_group_id` comes back, sync sets `deleted_at = NULL` on the **same platform id**, then re-inserts its memberships. Sync never issues `DELETE FROM groups`. |
| **D2** | `records_assignment_group_fkey` becomes `ON DELETE RESTRICT`. `nava_groupsync`'s grant on `groups` becomes `SELECT, INSERT, UPDATE [path, deleted_at]`: no DELETE, and never `id` or `keycloak_group_id` in UPDATE. The `groups_delete` policy is dropped together with the DELETE grant. |
| **D3** | **R29 (new ADR-004 rule).** For every table in `db/audited-tables.yaml`, every foreign key whose referencing table is that table has `confdeltype` and `confupdtype` in {`a` NO ACTION, `r` RESTRICT}. CI-A3 asserts this from `pg_constraint`; a partitioned table is checked on its parent. The commit-check trigger already refuses an FK-action write at COMMIT (TICK-452). R29 moves that failure from a production sync stall to CI. The query is below. |
| **D4** | **Readers of `groups`:** <br>• **These see all rows:** sync's diff read (`internal/groupsync/apply.go:285`), so that a returning group is revived rather than re-inserted (which would give `23505`); and **the D5 policy subquery** (`g.deleted_at IS NOT NULL`), which must not be "fixed" with the filter below. <br>• **These filter `deleted_at IS NULL`:** the `/me` group paths (`internal/identity/pgstore/pgstore.go:52`); onboarding `RecordAdminGroup` (`internal/onboard/store.go:196-214`), in both the `EXISTS` probe and the `INSERT … SELECT`, so that a tombstoned admins group gives `ErrAdminGroupMissing`; group pickers; ACL group resolution, when it exists; and D6. <br>• **Unchanged:** `principalSQL` (`pgstore.go:38`) reads `group_members` only, and D11 guarantees that a tombstoned group has none. |
| **D5** | When the recorded admins or auditors group is tombstoned, sync deletes its pointer row in the same transaction and still reports `AdminGroupDeleted`. `nava_groupsync` gets DELETE on `tenant_admin_group` and `tenant_auditor_group`, each with a policy `FOR DELETE TO nava_groupsync USING (<canonical tenant predicate> AND EXISTS (SELECT 1 FROM groups g WHERE g.tenant_id = <table>.tenant_id AND g.id = <table>.group_id AND g.deleted_at IS NOT NULL))`, where `<table>` is the pointer table itself (`tenant_admin_group` or `tenant_auditor_group`). The outer column is always qualified (Security N1): unqualified, `group_id` would resolve to the pointer table only while `groups` has no column of that name, and the predicate would otherwise silently compare a column with itself. Each policy has its entry in `internal/db/catalog/policy.go`. **I1 (verbatim):** *D10 step (i) also grants `nava_groupsync` `SELECT (tenant_id, group_id)` on `tenant_auditor_group`, and adds `nava_groupsync` to `tenant_auditor_group_select`'s `TO` list, wherever that table lands. Both the `DELETE … WHERE` and the "was the recorded group removed" probe need them. CI-E5-5b gains a check: tombstoning the recorded auditors group commits, and its pointer row is gone.* `tenant_admin_group` already has the matching pair (`tag_select`, `SELECT: []`). The pointer FKs stay `ON DELETE CASCADE`: these tables are not audited, so R29 does not apply. This narrows the P3-04 rule "no runtime role may UPDATE or DELETE `tenant_admin_group`" to a pointer whose group is tombstoned. |
| **D6** | The pipeline refuses a write that **changes** `assignment_group_id` to a tombstoned group, with `422 assignment_group_removed`. An **unchanged** value passes. The lookup runs in the pipeline transaction, under RLS, and never against a cached list. Another tenant's group, live or tombstoned, gives the same 422 as a random UUID. |
| **D7** | A record keeps `assignment_group_id` after its group is tombstoned. Read models return that group with `removed: true`. Nothing clears or reassigns these records automatically (see Q-A). |
| **D10** | **Migration constraint: expand, then contract.** <br>**Step (i), expand:** `ADD COLUMN groups.deleted_at`; the D11 columns, unique key and FK; `UPDATE (deleted_at)` for `nava_groupsync`; the D5 DELETE grants and policies; **and I1, the `SELECT (tenant_id, group_id)` grant on `tenant_auditor_group` plus `nava_groupsync` in `tenant_auditor_group_select`'s `TO` list.** Step (i) ships with the tombstoning sync (D1, D5) and the D4 filters. <br>**Step (ii), contract, only after step (i)'s binaries run everywhere:** <br>(ii-a) under `SET LOCAL lock_timeout`, one `ALTER TABLE records DROP CONSTRAINT records_assignment_group_fkey, ADD CONSTRAINT … ON DELETE RESTRICT NOT VALID`; revoke DELETE on `groups` from `nava_groupsync`; drop `groups_delete`. <br>(ii-b) a **separate** migration: `VALIDATE CONSTRAINT records_assignment_group_fkey`. <br>Then attach the commit-check trigger to `records` (`attached: true`) and add R29 to CI-A3. |
| **D11** | **The database refuses members in a tombstoned group.** `groups.live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED` and `UNIQUE (tenant_id, id, live)`. `group_members.live boolean NOT NULL DEFAULT true CHECK (live)`, with `FOREIGN KEY (tenant_id, group_id, live) REFERENCES groups (tenant_id, id, live)` (NO ACTION). The existing `ON DELETE CASCADE` FK stays. Tombstoning a group that still has members fails with `23503`, and so does inserting a member into a tombstoned group. This holds for every role, the owner included. |

R29 query (spiked by the Technical Architect, T4):

```sql
SELECT c.conrelid::regclass, c.conname, c.confdeltype, c.confupdtype
  FROM pg_constraint c
 WHERE c.contype = 'f' AND c.conparentid = 0
   AND c.conrelid = ANY ($1::regclass[])          -- audited tables from the manifest
   AND (c.confdeltype NOT IN ('a','r') OR c.confupdtype NOT IN ('a','r'));
-- expect 0 rows
```

### Q2: no secrets in record payloads

| # | Decision |
|---|---|
| **D8** | **A secret may not live in a record payload.** R6 applies to records as written: a value at an `x-nava-secret` or `writeOnly` path is refused with `422 audit_secret_in_diff`, naming the pointer and never the value. There is no "store and mask" mode. In JSON Schema 2020-12, `writeOnly` means "accepted, not returned"; on this platform, for a record, it is **narrowed** to "never accepted in a record". The entity-definition docs say so. |
| **D9** | S8 publish refuses a record entity definition (`definition_secret_required`) when any `required` or `dependentRequired` entry names a marked property. The check covers the root and every subschema that applies at the root instance location (`allOf`, `anyOf`, `oneOf`, `if`, `then`, `else`, `not`, `dependentSchemas`, recursively). The marker itself stays legal. D9 is a usability guard; R6's runtime refusal is the security control. |

## Plan notes (binding on the plan)

- **I3:** the revive (`deleted_at = NULL`) goes in batch 1, with the other group writes. The membership deletes come first in batch 2; the tombstone and the pointer delete follow them in batch 2.
- **I4:** sync rolls out as **stop-old, then start-new**. An old instance that meets a revived group inserts members without reviving it, and D11 fails its transaction (this fails closed).
- **I5:** in step (i), add `group_members.live`'s `CHECK` as `NOT VALID` and validate it later, together with the D11 FK. Keep `lock_timeout` on the migration that rewrites `groups`, because adding `live` rewrites the table under ACCESS EXCLUSIVE (about 0.4 s at 50k groups and 2.5M memberships).
- **Existing tests to rewrite** (they delete groups as `nava_groupsync`): `internal/db/dbtest/identity_tables_test.go:89,161`, `records_pipeline_isolation_test.go:30,33`, `records_pipeline_test.go:386`, `tenant_admin_group_test.go:267`, and `docs/data/identity-bench/03_sync.sql:22`. No isolation-registry entry or adversarial test is removed.
- **N3 (Security):** R-E5-4's recovery path is the operator runbook from TICK-372 (`admin_group_runbook_isolation_test.go`). P5-13c extends that runbook and its cross-tenant test to `tenant_auditor_group`.
- **N4 (Security):** CI-E5-6 runs the equal-422 check on **create and on update**. Both go through the D6 lookup; neither may end at a mapped FK `23503` while the other ends at D6.
- **`tenant_auditor_group`** is on the story branch `TICK-419/one-audit-log` (`20261102130100_tenant_auditor_group.sql`). D5 and I1 apply wherever P5-13c lands.

## Acceptance tests

| # | Proves |
|---|---|
| CI-E5-1 | Sync drops group G with the trigger **attached** to `records`. The sync commits, G is tombstoned and has no members, and no `records` row changes (`version`, `updated_at` and `xmin` are all identical). This replaces `TestP309GroupDeleteClearsRecordGroup`. |
| CI-E5-2 | After step (ii): `DELETE FROM groups` as `nava_groupsync` is denied, an owner delete of a referenced group fails with `23503`, and `nava_groupsync` cannot UPDATE `id` or `keycloak_group_id`. |
| CI-E5-3 | R29: a scratch audited table with an `ON DELETE SET NULL (col)` FK fails CI-A3, and `records` passes. |
| CI-E5-4 | G comes back with the same id: it is revived, its memberships are restored and records are unchanged. If G was the admins group, `RequireTenantAdmin` still denies. |
| CI-E5-5 | Inserting a member into tombstoned G gives `23503`, and so does tombstoning G while it has a member. A former member of G gets `ErrForbidden` from `RequireTenantAdminTx`. `RecordAdminGroup` against a tombstoned admins group gives `ErrAdminGroupMissing` and writes no pointer. |
| CI-E5-5b | The denial test at `tenant_admin_group_test.go:81` is split. Deleting a live pointer affects 0 rows; deleting a tombstoned group's pointer succeeds. The same pair runs on `tenant_auditor_group`. **I1:** tombstoning the recorded auditors group commits, and its pointer row is gone. The policy catalogue lists both new policies. |
| CI-E5-6 | Changing `assignment_group_id` to a tombstoned group gives `422 assignment_group_removed` and writes nothing. Leaving it unchanged succeeds and writes one audit entry. Another tenant's group and a random UUID give byte-identical 422 responses, on create and on update (N4). |
| CI-E5-7 | A value at a `writeOnly` property gives `422 audit_secret_in_diff`. The value does not appear in the response, `records`, `audit_log`, request logs, trace attributes or the idempotency `response_body`. D9 refuses a marked property required at the top level, under `allOf`, under `if`/`then`, and through `dependentRequired`. |
| CI-E5-8 | (ii-a) holds no lock beyond `lock_timeout`, and (ii-b) is its own migration. Afterwards, `confdeltype = 'r'` and `convalidated = true`. |

## Consequences

- TICK-452 ships with `records` at `attached: false`. P5-13c carries D10 and attaches the trigger. D9 is independent and can ship first.
- `nava_groupsync` loses DELETE on `groups`. It gains only DELETE limited by a policy, plus I1's two-column SELECT, on the two pointer tables. It gets nothing on `records`, `audit_log` or `outbox`.
- A group drop no longer scans the tenant's records: under `SET NULL`, one drop cost 502 ms in a tenant with 400k records (Postgres spike).

## Rejected options

| Option | Why it lost |
|---|---|
| (a) Sync clears the group through the pipeline, then deletes it | One record that fails its schema, has a stale version or hits the I9 cascade limit stalls the tenant's sync, deprovisioning included. It also gives `nava_groupsync` grants on `records`, `audit_log` and `outbox` (I3). |
| (b) `nava_groupsync` writes `records` and appends the entries | A second `records` write path outside the pipeline (I4), with no version bump and no outbox event (I6). |
| (c) Exempt FK-action writes (`WHEN (pg_trigger_depth() = 0)`) | **It breaks decision 2, and nothing detects it.** A record changes at the same version with no entry, the diffs no longer replay to the row, and the chain still verifies. **Any write issued from inside any trigger escapes as well** (spike T3/S4). |
| (d) Permanent S1 fallback for `records` | The architecture test does not see FK actions either: the same gap as (c), plus no database guard on `records`. |
| (e) `RESTRICT`, with sync reassigning first | This is (a) again. |
| (e) `RESTRICT`, with sync deferring the delete | It leaves a live-looking ghost group with no marker. |
| S1's RLS `WITH CHECK` on `group_members` instead of D11 | It covers inserts only and non-owner roles only. The CTO ruled for D11. |
| Store a `writeOnly` value and mask it out of the diff | `records.data` reaches scripts, outbox and Temporal payloads, search and backups. R7 masks only audit reads. |

## Recorded dissent

- **Technical Architect, T9:** drop D5 and keep the pointer, so that a revived admins group restores its admins, with fewer grants and fewer write paths. **Overruled** by Security, which owns this rung, and by the CTO. A revived group must not silently grant admin or audit access.

## Accepted risks

| # | Risk | Owner | Revisit trigger |
|---|---|---|---|
| R-E5-1 | Open records of a removed group drop out of every group queue until someone reassigns them. | Technical Lead | Product asks for auto-reassignment (Q-A, option e2). |
| R-E5-2 | A write that read G as live can commit just after G is tombstoned: `FOR KEY SHARE` does not conflict with `FOR NO KEY UPDATE`. The end state is the same as under D7. | Solution Architect | D7 changes. |
| R-E5-3 | Tombstoned `groups` rows are kept. At 10% tombstoned, readers change by under 0.05 ms. | Postgres Specialist | More than 10% of any tenant's groups are tombstoned. |
| R-E5-4 | When the admins group is revived, the tenant has no admin until the pointer is re-recorded. This fails closed. | Security reviewer | A tenant reports a lockout after a Keycloak restore. |
| R-E5-5 | While old and new sync instances overlap, an old instance fails on a revived group (I4). This fails closed. | Technical Lead | The rollout is not stop-old, then start-new. |

Indexed in `docs/security/accepted-risks.md` as AR-TICK-529-01 to AR-TICK-529-05 (R-E5-1 to R-E5-5).

**Open question Q-A (product, not blocking):** should records of a removed group be reassigned automatically? If yes, that is option (e2): an audited pipeline write by a `nava_worker` job, in a separate RFC.

## Evidence

- Technical Architect, round 1 (S0–S7, L1–L4) and round 2 (R1–R8, step (i) timings): `docs/adr/ADR-004/spikes/technical-architect/rfc-011/round-1/` and `…/round-2/`.
- Postgres Specialist (migration locks, `VALIDATE` timing, reader cost, the 502 ms `SET NULL` delete): `docs/adr/ADR-004/spikes/postgres/rfc-011/`.
- RFC-011, rev 0 and rev 1, with the review record: `docs/adr/ADR-004/rfc/`.
