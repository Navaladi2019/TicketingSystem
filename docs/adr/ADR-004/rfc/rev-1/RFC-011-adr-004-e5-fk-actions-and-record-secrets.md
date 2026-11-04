---
type: rfc
id: RFC-011
title: "ADR-004-E5: no referential actions on audited tables (groups are tombstoned), and no secrets in record payloads"
status: REVISED (rev 1)
date: 2026-11-04
issue: TICK-529
thread: TICK-529 comment 01a1a373-04ed-79e0-9c10-5756243cfdc2 (rev 0: 01a1a377-1906; round 1 closed by the CTO: 01a1a37f-7cbc)
raised_from: TICK-452 (P5-13b), the engineer's RESULT BLOCKED (comment 01a1a371-b984)
target: ADR-004 erratum E5 (docs/adr/ADR-004-E5-*.md and ADR-004-E5-explained.html)
author: Solution Architect
---

# RFC-011: FK-action writes on audited tables, and the R6 `writeOnly` reading

> **Invariants source.** The project ("Ticket Management System") has no description, so it defines no invariants. The CTO ruled in round 1 that ADR-001 §1 (I1–I9) is the authoritative list and that "eight invariants" in the role text is stale. This RFC is checked against I1–I9 and against ADR-004 decision 2.
>
> **Numbering.** I assumed RFC-011 and ADR-004-E5: RFC-010 and E4 are the highest numbers in `docs/adr/`.

## Response — round 1 (changelog rev 0 → rev 1)

| Item | Source | Change |
|---|---|---|
| **PG-B1** (blocking) | Postgres, Q-C | New **D10**: migration constraint "expand, then contract". Step (i) and step (ii) are separate deploys. |
| **PG-B2** (blocking) | Postgres, Q-C; Security (minor point) | D4 now lists each `groups` reader explicitly. Sync's diff read sees all rows. Onboarding returns `ErrAdminGroupMissing`. |
| T8 / S1 | CTO ruling | New **D11**: a generated `live` column plus a composite FK, so the database refuses members in a tombstoned group. S1's RLS `WITH CHECK` is not adopted. S1's test is kept, with error `23503`. |
| S2 / T9 | CTO ruling | D5 now takes S2's form: a DELETE **policy** that reaches only pointers to tombstoned groups, a catalogue entry, and a split denial test. T9's dissent is recorded (§8). |
| T7, S3 | CTO fold-in | D1 no longer writes `synced_at`. D2 now sets the `groups` UPDATE grant to exactly `[path, deleted_at]` and drops `groups_delete` along with the DELETE grant. The `tenant_auditor_group` rule is added. §5 lists the tests to rewrite. |
| T2, T3/S4 | CTO fold-in | §6 now uses T2's wording. Rejecting (c) now also covers writes issued from inside triggers. |
| T4 | CTO fold-in | D3 (R29) now uses the Technical Architect's text and query (`conparentid = 0`). |
| T10, T11 | CTO fold-in | D9 now walks every subschema at the root instance location. D8 states that `writeOnly` is narrowed here. |
| S4, Security CI-E5-7 | CTO fold-in | CI-E5-6 gets the equal-422 test. CI-E5-7 adds checks on logs, traces and the idempotency `response_body`. |
| Spikes | CTO fold-in | New §9: the archive paths for both spike bundles. |

## 1. Summary

- **Q1: option (e), soft-delete variant.** Group sync tombstones groups (`groups.deleted_at`) and stops deleting them. `records_assignment_group_fkey` becomes `ON DELETE RESTRICT`. Group sync then never writes `records`, so the commit-check trigger attaches to `records` unchanged.
- **New rule R29.** No FK on an audited table may carry a referential action. CI-A3 checks this from `pg_constraint`.
- **The database itself guarantees that a tombstoned group grants nothing (D11).** A tombstoned group cannot have members, and that is enforced by an FK, not by discipline in `apply.go`.
- **The rollout is expand, then contract (D10).** The FK swap ships only after the tombstoning sync is deployed. In the other order, the old binary's `DELETE` hits `RESTRICT` and stalls sync again.
- **Q2: no. Secrets never live in record payloads.** R6 is read as written, and `writeOnly` is narrowed to "never accepted in a record". D9 refuses a record definition that requires a marked property anywhere at the root.
- **What the reviewer accepts:** a record of a removed group keeps pointing at that group and is shown as removed. The race R-E5-2 is accepted. T9's dissent on D5 is recorded.

## 2. Decisions

| # | Decision | Why / what it prevents |
|---|---|---|
| **D1** | `groups` gains `deleted_at timestamptz NULL` (nullable, no default, so adding it is a catalogue change only). For a group that Keycloak no longer has, group sync, in the sync's tenant transaction and in this order: (1) deletes the group's `group_members` rows; (2) sets `deleted_at = now()` (D11 refuses this while members remain); (3) deletes its admin/auditor pointer if one exists (D5; the policy admits the delete only once the group is tombstoned). If the same `keycloak_group_id` comes back, sync sets `deleted_at = NULL` on the **same platform id** and re-inserts its memberships. Sync never issues `DELETE FROM groups`. No `synced_at` (`groups` has no such column). | No write reaches `records` from group sync, so sync cannot stall on a record (the TICK-452 failure). This mirrors the principal tombstone at `internal/groupsync/apply.go:253`. Spiked by the Postgres Specialist: tombstoning 500 groups in a tenant with 400k records took 7.2 ms, fired no RI trigger, and left `xmin` unchanged on every record. |
| **D2** | `records_assignment_group_fkey` becomes `ON DELETE RESTRICT`. In `db/grants.yaml`, `nava_groupsync`'s grant on `groups` becomes `SELECT, INSERT, UPDATE [path, deleted_at]`: no DELETE, and never `id` or `keycloak_group_id` in UPDATE. The `groups_delete` policy is dropped in the same migration as the grant. | A hard delete of a referenced group fails with `23503` instead of rewriting records behind the pipeline (Technical Architect S6). Removing the grant and the policy together keeps the policy/grant catalogue check clean. |
| **D3** | **R29 (new ADR-004 rule), the Technical Architect's text:** *For every table in `db/audited-tables.yaml`, every foreign key whose referencing table is that table has `confdeltype` and `confupdtype` in {`a` NO ACTION, `r` RESTRICT}. CI-A3 asserts this from `pg_constraint`; a partitioned table is checked on its parent. The commit-check trigger already refuses an FK-action write at COMMIT (TICK-452). R29 moves that failure from a production sync stall to CI.* The query is in §2.1. | A referential action runs as the table owner, without the caller's grants or RLS (Technical Architect T1, spiked), so CI-A3's privilege check cannot see it. R29 closes the whole class of such writers. |
| **D4** | **Readers of `groups`, one by one** (exhaustive at `27860f4`; the plan task re-greps): <br>• **Sees all rows:** sync's diff read `internal/groupsync/apply.go:285`. It must see tombstones, or a returning group looks new, its INSERT hits `groups_keycloak_group_key` (`23505`), and sync fails. <br>• **Filters `deleted_at IS NULL`:** the resolver's `/me` group paths (`internal/identity/pgstore/pgstore.go:52`, the `group_members JOIN groups` path); onboarding `RecordAdminGroup` (`internal/onboard/store.go:196-214`), both the `EXISTS` probe and the `INSERT … SELECT` into `tenant_admin_group`, so that a tombstoned admins group gives **`ErrAdminGroupMissing`** (confirmed by Security); group pickers; ACL group resolution, when it exists (records have no group ACL today, per Security Q-B); and the D6 pipeline check. <br>• **Unchanged:** the resolver's `principalSQL` (`pgstore.go:38`) reads `group_members` only, and D11 guarantees that a tombstoned group has none. | Postgres B2. A blanket "every reader" rule breaks sync's resurrection (CI-E5-4). The onboarding filter stops a pointer from being written to a tombstoned admins group. The filter costs under 0.05 ms on every measured reader and needs no new index (Postgres spike). |
| **D5** | **The pointer is deleted, in S2's form.** When the recorded admins group is tombstoned, sync deletes its `tenant_admin_group` row in the same transaction and still reports `AdminGroupDeleted`. Instead of a plain grant, `nava_groupsync` gets `DELETE` plus a policy `tenant_admin_group_delete … FOR DELETE TO nava_groupsync USING (<canonical tenant predicate> AND EXISTS (SELECT 1 FROM groups g WHERE g.tenant_id = tenant_admin_group.tenant_id AND g.id = group_id AND g.deleted_at IS NOT NULL))`, with a matching entry in `internal/db/catalog/policy.go`. The same rule, policy and test apply to `tenant_auditor_group`. That table exists on the story branch `TICK-419/one-audit-log` (`20261102130100_tenant_auditor_group.sql`), not yet on `main`, so it applies wherever P5-13c lands. Both pointer FKs stay `ON DELETE CASCADE`: the tables are not audited, so R29 does not apply to them. | Security owns this rung (S2). Resurrection keeps the same platform id (D1), so a kept pointer would silently re-make admins whoever is in the revived group. Deleting it fails closed. Because the policy reaches only tombstoned groups, a buggy or compromised sync can delete **no live pointer**. This narrows the P3-04 rule "no runtime role may UPDATE or DELETE `tenant_admin_group`" to the single case of a tombstoned group; it does not reverse it. |
| **D6** | The pipeline's declarative-check step refuses a write that **changes** `assignment_group_id` to a tombstoned group, with `422 assignment_group_removed` naming the field. An **unchanged** value passes. The lookup runs inside the pipeline transaction, under the caller's RLS, never against a cached or unscoped list (S4). Another tenant's group, live or tombstoned, gives the same 422 as a random UUID (the existing missing-group error), never `assignment_group_removed`. | The FK accepts a new reference to a tombstoned group, because the row exists (Technical Architect S7). The pipeline is the only `records` writer, so D6 is the guard. S4: the error must not reveal whether another tenant's group exists. |
| **D7** | A record keeps `assignment_group_id` after its group is tombstoned. Read models return the group with `removed: true`. Records are not cleared automatically in this erratum. | This is the honest state of the record, and it needs no write. Auto-reassignment is (e2), a product call (§8 Q-A). |
| **D8 (Q2)** | **Secrets may not live in record payloads.** R6 stands for records: a value at an `x-nava-secret` or `writeOnly` path is refused with `422 audit_secret_in_diff`, naming the pointer and never the value. There is no "store and mask" mode. **`writeOnly` is narrowed:** JSON Schema 2020-12 means "accepted, not returned". On this platform a record means "**never accepted in a record**". The erratum and the entity-definition docs say so, because tenants who import schemas will read the standard meaning (Technical Architect T11). | `records.data` is plaintext to every reader: GraphQL and scripts (I8), outbox and Temporal payloads (I6), search, embeddings and backups. R7 masks only audit reads. R6 can see every marked path, because the marker is legal only on top-level properties (`internal/metadata/compile.go:330-341`). |
| **D9 (Q2)** | S8 publish refuses a **record** entity definition (`definition_secret_required`) when any `required` or `dependentRequired` entry names a marked property. This covers the root schema **and every subschema that applies at the root instance location**: `allOf`, `anyOf`, `oneOf`, `if`, `then`, `else`, `not` and `dependentSchemas`, recursively. A root `$ref` is already refused when a marker exists (`checkMarked`). The marker itself stays legal. | Technical Architect T10's failing case: `{"properties":{"pw":{"type":"string","writeOnly":true}},"allOf":[{"required":["pw"]}]}` passes a top-level-only check, and then every create of that type is refused. D9 is a usability guard; R6's runtime refusal is the security control (Security). A gap in D9 locks a tenant out of creating that type, but it leaks nothing. |
| **D10** (PG-B1) | **Migration constraint: expand, then contract.** <br>**Step (i), expand:** `ADD COLUMN groups.deleted_at`; add D11's columns, unique key and FK; add `UPDATE (deleted_at)` to `nava_groupsync`'s grant on `groups`; add D5's DELETE policy and grant on the pointer tables. Ship with the tombstoning sync (D1, D5) and the D4 filters. Sync still holds `DELETE` on `groups` but no longer issues it. <br>**Step (ii), contract, only after step (i)'s binaries are deployed everywhere:** <br>(ii-a) one migration: `SET LOCAL lock_timeout`, then a single `ALTER TABLE records DROP CONSTRAINT records_assignment_group_fkey, ADD CONSTRAINT … ON DELETE RESTRICT NOT VALID`, then revoke `DELETE` on `groups` from `nava_groupsync` and drop `groups_delete`. <br>(ii-b) a **separate** migration: `ALTER TABLE records VALIDATE CONSTRAINT records_assignment_group_fkey`. <br>Then attach the commit-check trigger to `records` (`attached: true`), and add R29 to CI-A3. | Postgres B1's failing scenario: if (ii) runs while the old sync binary is live, its `DELETE FROM groups … = ANY(gone)` (`apply.go:389`) hits `RESTRICT` on any referenced group, and the tenant's whole sync rolls back, deprovisioning included. The single-statement swap leaves no moment without an FK. It holds ACCESS EXCLUSIVE on `records` and `groups` for about 9 ms, and that blocks the resolver, so it fails fast and is re-run instead of queueing behind a long sync. `VALIDATE` takes 1.4 s on 760k rows under SHARE UPDATE EXCLUSIVE, which blocks neither reads nor writes. Run inside (ii-a), it would hold ACCESS EXCLUSIVE for that whole time. This follows the 00010–00012 principal precedent. |
| **D11** (T8) | **A tombstoned group has no members, enforced by the database.** `groups.live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED` with `UNIQUE (tenant_id, id, live)`. `group_members.live boolean NOT NULL DEFAULT true CHECK (live)` with `FOREIGN KEY (tenant_id, group_id, live) REFERENCES groups (tenant_id, id, live)` (NO ACTION). The existing `group_members_group_fkey … ON DELETE CASCADE` stays. Effects: tombstoning a group that still has members gives `23503` (L1); deleting the members and then tombstoning, in one transaction, succeeds (L2); inserting a member into a tombstoned group gives `23503` (L3); reviving the group and then inserting succeeds (L4). It holds for every role, the owner included. | The CTO's ruling over S1's RLS `WITH CHECK`, which covered inserts only and non-owner roles only. D4's guarantee is now a constraint, not a convention, and it covers both directions. Cost: one constant boolean on `group_members` and one unique index on `groups`. The Postgres Specialist confirms the cost at plan time; it does not block. `group_members` is not audited, so R29 does not apply. |

### 2.1 R29 query (Technical Architect, T4)

```sql
SELECT c.conrelid::regclass, c.conname, c.confdeltype, c.confupdtype
  FROM pg_constraint c
 WHERE c.contype = 'f' AND c.conparentid = 0
   AND c.conrelid = ANY ($1::regclass[])          -- audited tables from the manifest
   AND (c.confdeltype NOT IN ('a','r') OR c.confupdtype NOT IN ('a','r'));
-- expect 0 rows
```

Before the change, this returns `records_assignment_group_fkey | n | a` (the PG15+ column-list `SET NULL` is caught). After the change it returns 0 rows. If the manifest names a table that does not exist, the `regclass` cast raises an error, so the check fails loudly (Postgres).

## 3. Options considered (Q1)

| Option | What it does | Effect on the audit trail | Verdict |
|---|---|---|---|
| **(a)** Sync clears the group through the pipeline, then deletes it | Each record gets an audited, versioned write with an outbox event | Full coverage | **Rejected.** It couples identity sync to the record pipeline. One record that fails its schema, has a stale version or hits the I9 cascade limit stalls the tenant's sync, deprovisioning included. Measured: one `SET NULL` group delete costs 502 ms in a tenant with 400k records, so a sync that drops 100 groups spends about 50 s in its transaction. Through the pipeline, it would be one write per record. It would also give `nava_groupsync` grants on `records`, `audit_log` and `outbox` (I3). |
| **(b)** `nava_groupsync` becomes a `records` writer and appends the entries | The FK action still runs, without a version bump | Entries exist, but they are inconsistent: each repeats a `version_after` that already has an entry, and there is no outbox event | **Rejected.** It is a second `records` write path outside the pipeline (I4), with no optimistic concurrency and no event for routing or SLA rules (I6). |
| **(c)** The trigger exempts FK-action writes (`WHEN (pg_trigger_depth() = 0)`) | Sync works, and records change silently | **Breaks ADR-004 decision 2, and nothing detects it.** Spiked (T3, S2): record R is at v7 with group G; G is deleted; the row now holds NULL at v7, `updated_at` is unchanged, and there are 0 audit rows. Replaying R's diffs gives G, but the row says NULL, and the chain still verifies. **The gap is wider than one FK (T3, S4): any write issued from inside any trigger also escapes.** In the spike, `nava_app` inserted into an unaudited table whose trigger updated `records`, and the change committed with 0 audit rows. | **Rejected.** It also breaks I4: the content changes with no version bump, so a client holding ETag v7 that re-sends G gets an FK error instead of `412`. |
| **(d)** Permanent S1 fallback for `records`, guarded by the P3-13 architecture test | The trigger is never attached to `records` | **Worse than (c).** The architecture test checks Go call sites, so it does not see the FK action either. The result is the same silent gap, plus no database enforcement of any raw write to `records`, and CI-A2 permanently skipped for the most important audited table. | **Rejected.** The fallback exists for a performance miss, and S1 passed. |
| **(e) RESTRICT, and sync reassigns first** | A group cannot be deleted while referenced, so sync rewrites the records first | Covered only if the rewrite goes through the pipeline | **Rejected.** This is (a) again, with the same stall. |
| **(e) RESTRICT, and sync defers the delete** | Sync strips the members but keeps the group row until nothing references it | No record write | **Rejected.** A live-looking ghost group stays with no marker. A soft delete is the same thing with the state made explicit. |
| **(e) soft delete: chosen** | D1–D11 | **Full coverage, with the trigger unchanged.** No database-side writer of `records` remains, and CI-A3 with R29 proves it. | **Chosen.** |
| (e2) Soft delete, plus a later audited clear | (e), then a `nava_worker` job clears or reassigns the records of tombstoned groups through the pipeline (`source = system`) | Full coverage | **Deferred, not rejected.** It is a product call (§8 Q-A) and composes with (e) without changing it. |

## 4. Invariants touched

| Invariant | Effect |
|---|---|
| ADR-004 decision 2 | **Restored.** The trigger attaches to `records` with its current `UPDATE OF` list. R29 closes the FK-action class. Writes issued from triggers stay covered because no `WHEN` exemption is added. |
| I1 tenant wall | Unchanged, and slightly improved: removing the `SET NULL` removes a writer that ran as the owner and bypassed RLS (T1). Every key keeps `tenant_id`. D6 runs under RLS (S4). |
| I3 least privilege | **Tightened.** `nava_groupsync` loses `DELETE` on `groups`, and its UPDATE grant is exactly `[path, deleted_at]`. It gains only a policy-bounded `DELETE` on the two pointer tables (D5). It gains nothing on `records`, `audit_log` or `outbox`. |
| I4 one write pipeline | **Restored.** The `SET NULL` was an out-of-pipeline writer, and (e) removes it. |
| I5 one authorization layer | A tombstoned group grants nothing on any current read path (Security Q-B), and D11 makes that a database constraint. |
| I6 outbox only | Unchanged under (e). |
| I8 configuration, not code | Unchanged. D9 is a publish-time check, not tenant DDL. |
| I2, I7, I9 | Not touched. |

## 5. Acceptance tests (for the erratum's plan)

| # | Proves |
|---|---|
| CI-E5-1 | `TestP309GroupDeleteClearsRecordGroup` is replaced. Sync drops G from Keycloak. Then `groups.deleted_at` is set, G's memberships are gone, records still reference G, and **no `records` row changed** (`version`, `updated_at` and `xmin` are identical). Sync commits **with the trigger attached** to `records`. |
| CI-E5-2 | `DELETE FROM groups` as `nava_groupsync` is denied (after step ii). An owner delete of a referenced group fails with `23503`. `nava_groupsync` cannot UPDATE `id` or `keycloak_group_id`. |
| CI-E5-3 | CI-A3 with R29: a scratch audited table with an `ON DELETE SET NULL (col)` FK fails the catalogue check (negative test), and `records` passes. |
| CI-E5-4 | G comes back in Keycloak with the same id: `deleted_at` is cleared, the memberships are restored, and records are unchanged. **If G was the admins group, `RequireTenantAdmin` still denies** (the pointer is gone, per D5). |
| CI-E5-5 | Isolation (S1, now on D11): `nava_groupsync` inserting a member into tombstoned G gives **`23503`**. Tombstoning G while it still has a member gives `23503`. A principal who was in G before the tombstone gets `ErrForbidden` from `RequireTenantAdminTx`. Onboarding `RecordAdminGroup` against a tombstoned admins group returns `ErrAdminGroupMissing` and writes no pointer. |
| CI-E5-5b | D5 (S2): the existing denial test at `internal/db/dbtest/tenant_admin_group_test.go:81` is split. `nava_groupsync` deleting a **live** pointer affects 0 rows; deleting the pointer of a **tombstoned** group succeeds. The same pair runs for `tenant_auditor_group`. The policy catalogue lists the new policy. |
| CI-E5-6 | Pipeline: changing `assignment_group_id` to tombstoned G gives `422 assignment_group_removed`, and nothing is written. An update that leaves a tombstoned G unchanged succeeds and writes one audit entry. **S4:** tenant B's group (live or tombstoned) and a random UUID give byte-identical 422 responses to tenant A, never `assignment_group_removed`. |
| CI-E5-7 | Q2: a record write with a value at a `writeOnly` property gives `422 audit_secret_in_diff`, naming the pointer. The value is absent from the response, from `records` and `audit_log` bytes, **from request logs and trace attributes** (the refusal happens before any body capture), and **from the stored idempotency `response_body`** of the 422. CI-A5 is extended to records. D9: publishing a record definition with a marked property required at the top level, under `allOf`, under `if`/`then`, or through `dependentRequired` gives `definition_secret_required`. |
| CI-E5-8 | D10: (ii-a) holds no lock beyond `lock_timeout`, and (ii-b) runs as its own migration. Afterwards, `confdeltype = 'r'` and `convalidated = true`. |

**Existing tests to rewrite** (they delete groups as `nava_groupsync`, T7(4)): `internal/db/dbtest/identity_tables_test.go:89,161`, `internal/db/dbtest/records_pipeline_isolation_test.go:30,33`, `internal/db/dbtest/records_pipeline_test.go:386`, `internal/db/dbtest/tenant_admin_group_test.go:267`, and `docs/data/identity-bench/03_sync.sql:22`. In each, the plan task replaces the delete with a tombstone or with an owner-side delete. No isolation-registry entry or adversarial test is removed.

## 6. Verified behaviour (formerly "assumptions not verified")

1. **FK actions run as the referencing table's owner, without RLS (T1, spiked).** `nava_groupsync` holds no privilege on `records`, and `records` has FORCE RLS with no policy for its owner, yet the `SET NULL` still nulled the row (`current_user=own`, `depth=2`). The version and `updated_at` did not change, and nothing was audited. **Why the failure surfaced at COMMIT (T2 wording):** the deferred commit check is queued by the RI update and runs at COMMIT as the session role, so the check runs as `nava_groupsync`. That role holds no `audit_log` grant (it is not a writer in the manifest), so the check fails with 42501 at COMMIT. The COMMIT timing comes from `DEFERRABLE INITIALLY DEFERRED`.
2. **Option (c)'s mechanism (T3, spiked).** See §3: the FK write and any write issued from a trigger escape, while a raw write at depth 0 is still caught.
3. **The `groups` readers** are listed in D4, against `27860f4`. The plan task re-greps on its own base.
4. **No existing record definition requires a marked property.** Not verified. If one does (dev or CI data only, pre-release), D9 refuses its next publish, not its existing version.
5. **The migration shape and lock levels** are measured by Postgres (Q-C) and fixed in D10.

## 7. Risks and accepted residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R-E5-1 | Open records stay assigned to a removed group and drop out of every group queue until someone reassigns them. They were equally invisible to the group under `SET NULL`; now the pointer shows where they were. | Technical Lead (the UI shows "removed group") | Product asks for auto-reassignment → (e2). |
| R-E5-2 | A pipeline write that read G as live can commit just after sync tombstones G. The FK check takes `FOR KEY SHARE` and the tombstone takes `FOR NO KEY UPDATE`, and those do not conflict (Postgres, lock-mode analysis). The end state is the same as a record assigned before the tombstone, which D7 accepts. | Solution Architect | D7 changes. |
| R-E5-3 | Tombstoned `groups` rows are kept forever. At 10% tombstoned, reader timings changed by under 0.05 ms (Postgres). Tenant offboarding (Epic 30) hard-deletes records before groups, and `RESTRICT` enforces that order. | Postgres Specialist | More than 10% of any tenant's groups are tombstoned. |
| R-E5-4 | When the admins group is revived, the tenant has no admin until onboarding or an operator re-records the pointer (D5). This fails closed. | Security reviewer | A tenant reports a lockout after a Keycloak restore. |

## 8. Open questions and recorded dissent

- **Q-A (product, human, not blocking):** should records of a removed group be reassigned automatically? If yes, that is (e2) in a separate small RFC. Until then, D7 stands.
- **Q-D (Technical Lead, at plan time):** sequencing within D10's two steps. D9 is independent and can ship first.
- **Postgres, at plan time:** confirm the cost of D11. Ruled non-blocking.
- **Dissent (Technical Architect, T9):** drop D5 and keep the pointer, so that a revived admins group restores its admins, with fewer grants and fewer write paths. **Overruled** by Security, which owns this rung, and by the CTO's ruling. The reason is that a revived group could otherwise silently grant admin. This is recorded for the ADR.

## 9. Evidence to archive (with the ADR PR)

- Technical Architect spike S-FK (S0–S7, L1–L4): `docs/adr/ADR-004/spikes/technical-architect/rfc-011/` (`setup.sql`, `run.sql`, `live.sql`, output). Source: TICK-529 comment 01a1a37f-0be0.
- Postgres Specialist Q-C spike: `docs/adr/ADR-004/spikes/postgres/rfc-011/` (`spike-sql.txt`, `spike-out.txt`). Source: TICK-529 comment 01a1a37c-61e4.
- This RFC: `docs/adr/ADR-004/rfc/RFC-011-…md` (rev 1), with `rev-0/` and `rev-1/`.
