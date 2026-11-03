---
type: rfc
id: RFC-011
title: "ADR-004-E5: no referential actions on audited tables (groups are tombstoned), and no secrets in record payloads"
status: APPROVED (rev 1; round 2 APPROVE with condition I1, folded into ADR-004-E5)
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


---

# Review record (archived from TICK-529)

Spike scripts and raw output are not repeated here; they are under `docs/adr/ADR-004/spikes/technical-architect/rfc-011/` and `docs/adr/ADR-004/spikes/postgres/rfc-011/`.

## Security & Multi-Tenancy Reviewer — comment 01a1a379-c6df-76ed-9d60-4cce03529296 (2026-11-03T22:13:49Z)

**Security review of RFC-011 rev 0 (Q-B), checked against the code at `27860f4`.**

## Q-B: does a tombstoned group grant anything? No, on every read path that exists today
I checked every path where a group affects authorization:

| Path | Group source | Tombstoned G grants? |
|---|---|---|
| Resolver `principalSQL` (`internal/identity/pgstore/pgstore.go:38`) → `Principal.Groups` | `group_members` only | No, because D1 deletes G's memberships |
| `authz.decide` (`internal/authz/admin.go:64`) | `Principal.Groups` compared with `tenant_admin_group` | No: no membership, and the pointer is gone (D5) |
| `/me` `groupPathsSQL` (`pgstore.go:52`) | `group_members JOIN groups` | No, there are no member rows |
| Records read/list (`internal/records/store`) | No group ACL today, only the tenant predicate | Nothing to grant. D4's "no record read through G" holds trivially for now |
| Token claims | The resolver never reads a groups claim | A stale access token cannot carry G |

The resolver has no cache, and `records.assignment_group_id` is routing data, not an ACL input. The tenant wall does not move: the FK is composite on `(tenant_id, …)` and `groups` keeps its RLS. Removing the `SET NULL` also removes a writer that bypasses RLS (if the Technical Architect's spike confirms it runs as the owner), so R29 is a net gain for I1 as well.

## Findings (none exploitable today)
**S1: nothing below Go stops a tombstoned group from having members.**
- **Attack:** D4 rests on one assumption: that sync deletes memberships at tombstone time. Suppose a sync ordering bug or a partial batch leaves, or re-inserts, a `group_members` row for tombstoned G (the resurrection path in D1 is the likely place). Then G grants whatever it granted before, admin included if the pointer survived. Every authorization check reads only `group_members`.
- **Status:** will bite us at scale. **Severity:** Medium.
- **Fix:** change `group_members_insert` to `WITH CHECK (tenant pred AND EXISTS (SELECT 1 FROM groups g WHERE g.tenant_id = group_members.tenant_id AND g.id = group_id AND g.deleted_at IS NULL))`. Do the tombstone `UPDATE` and the membership `DELETE` in one statement or one CTE.
- **Test:** add to CI-E5-5. `nava_groupsync` inserts a member into tombstoned G → `42501`. A principal who was in G before the tombstone gets `ErrForbidden` from `RequireTenantAdminTx`.

**S2: D5 reverses the P3-04 rule "no runtime role may UPDATE or DELETE `tenant_admin_group`".**
- **Attack:** with a plain `DELETE` grant, a buggy or compromised `nava_groupsync` can delete a **live** admin pointer, and nothing re-creates it. That fails closed, so it is a denial of service for tenant admin, not an escalation. Also note that resurrection now keeps the **same platform id** (today a returning group gets `uuid.New()`). So deleting the pointer is the right call: it stops a resurrected G from silently becoming admins again.
- **Status:** will bite us at scale. **Severity:** Low.
- **Fix:** add a DELETE policy `TO nava_groupsync USING (tenant pred AND EXISTS (… groups g … AND g.deleted_at IS NOT NULL))`. Sync can then delete only the pointer of a tombstoned group. Add the matching entry in `internal/db/catalog/policy.go`.
- **Test:** split the existing denial test (`internal/db/dbtest/tenant_admin_group_test.go:81`). `nava_groupsync` deleting a live pointer affects 0 rows; deleting a tombstoned one succeeds. Add to CI-E5-4: after G comes back, `RequireTenantAdmin` still denies.

**S3: the RFC's grant and table details do not match the schema.**
- **Status:** will bite us at scale. **Severity:** Low.
- **Problems:**
  - `groups` has no `synced_at` column (D1).
  - The `groups` UPDATE grant must become exactly `[path, deleted_at]`. It must never include `id` or `keycloak_group_id`.
  - When the `DELETE` grant goes, drop the `groups_delete` policy with it.
  - `tenant_auditor_group` does not exist. D4 and D5 should say the rule applies to it when it lands.
- **Fix:** correct D1, D2 and D5 in rev 1.

**S4: D6 could leak whether another tenant's group exists.**
- **Attack:** tenant A sets `assignment_group_id` to tenant B's group id. If the response differs from the one for a random UUID, A learns that B's group exists.
- **Status:** not exploitable as designed, because the D6 lookup runs under A's RLS and never sees B's rows. **Severity:** Low.
- **Fix:** keep the D6 check inside the pipeline transaction. Never run it against a cached or unscoped group list.
- **Test:** add to CI-E5-6. B's group (live or tombstoned) and a random UUID give the same 422, never `assignment_group_removed`.

Minor point: `internal/onboard/store.go:202,213` reads `groups` by `keycloak_group_id` without filtering on `deleted_at`. It only runs for provisioning tenants against a group id Keycloak has just created, so it is low risk, but add the filter for consistency with D4.

## R6 / D8 / D9: agreed
- **D8 is correct.** Secrets never belong in `records.data`. "Store and mask" would spread the value to GraphQL and scripts, the outbox and Temporal, search, and backups, and R7 cannot follow it to any of them.
- **D9 is a usability guard, not a security control.** The runtime refusal in R6 remains the control. To be complete, D9 must check every place a property can be required (`allOf`, `anyOf`, `oneOf`, `if`/`then`/`else`, `dependentRequired`, `dependentSchemas`, and `$ref` to `$defs`). If it misses one, the type is still unusable, but nothing leaks.
- **CI-E5-7 must also cover:** the refusal happens before any request logging or trace attribute captures the body. The stored idempotency `response_body` for the 422 must not contain the value.

No adversarial test is pushed this round: every case above depends on the `deleted_at` column, which does not exist yet. S1, S2 and S4 should be folded into the plan as CI-E5 extensions, and I will review them as code in P5-13c.

VERDICT: APPROVE (no exploitable isolation gap; fold S1–S3 into rev 1, S1 and S2 at the policy layer, not in Go)

## Postgres & Data Modeling Specialist — comment 01a1a37c-61e4-7522-9639-7206da1d5207 (2026-11-03T22:16:40Z)

**Q-C review of RFC-011 rev 0 (Postgres & Data Modeling Specialist)**

**VERDICT: CHANGES.** There are two small blocking items, both about sequencing and wording. The migration shape, the R29 query and the cost of the `deleted_at` filter all hold up when measured.

**Spike setup:** PostgreSQL 16.10 in Docker on an i7-1260P (16 vCPU, 19 GB given to Docker). The schema copies `groups`, `group_members` and `records` with their real FKs and the partial index. Data: 51,900 groups (10 tenants with 5,000 each, 190 with 10), 760,000 records (one tenant has 400k), and a worst-case principal in 200 groups. The scripts and raw output are attached.

## Confirmed
- **PG16 cannot change the FK action in place.** `ALTER CONSTRAINT … ON DELETE RESTRICT` fails with a syntax error, so the constraint has to be dropped and added again.
- **Step 1 is one statement:** `DROP CONSTRAINT` plus `ADD CONSTRAINT … ON DELETE RESTRICT NOT VALID` inside one `ALTER TABLE`, so there is no moment with no FK. It holds **ACCESS EXCLUSIVE on both `records` and `groups`** for 9 ms. The lock on `groups` blocks the resolver's read on every authenticated request. So keep `SET LOCAL lock_timeout`, as 00010 does: the migration fails and is re-run rather than making reads queue behind a long sync transaction.
- **Step 2 is `VALIDATE` in its own migration.** It took 1,388 ms on 760k rows. It holds only SHARE UPDATE EXCLUSIVE on `records` and ROW SHARE on `groups`, which blocks neither reads nor writes. If it ran in the step-1 transaction, the ACCESS EXCLUSIVE locks on both tables would be held for those 1.4 s. Afterwards: `confdeltype = r`, `convalidated = t`.
- **A tombstone writes nothing to `records`.** Tombstoning 500 of the large tenant's groups took 7.2 ms. No RI trigger fired, because the update leaves the key unchanged, and `xmin` on all 400k of that tenant's records stayed the same. A hard delete of a referenced group, run as the owner, now fails with `23503`.
- **R29 query.** Before the change it returns `records_assignment_group_fkey | n | a`; after it, zero rows:
  ```sql
  SELECT c.conrelid::regclass, c.conname, c.confdeltype, c.confupdtype
  FROM pg_constraint c
  WHERE c.contype = 'f'
    AND c.conrelid = ANY ($1::regclass[])   -- names from db/audited-tables.yaml
    AND (c.confdeltype NOT IN ('a','r') OR c.confupdtype NOT IN ('a','r'));
  ```
  - Filtering on `conrelid` (the audited table is the referencing side) is the correct direction: that is the side an FK action writes to.
  - If the YAML names a table that does not exist, the `::regclass[]` cast raises an error, so the check fails loudly rather than passing.
  - No partition handling is needed (D23).
- **Cost of `deleted_at IS NULL` (10% of groups tombstoned).** Every reader in the table below stays on the index it used before. The filter runs on a row the query has already fetched, and the buffer counts did not change. **No new index.**
- **Storage and writes:** the new column is nullable with no default, so adding it is a catalogue change (4 ms). Each tombstone is one HOT-eligible update of a small row, so the write cost is negligible.

| Reader | Before | With filter |
|---|---|---|
| `pgstore.go` resolver, 200 groups | 0.391 ms | 0.340 ms |
| group picker, 5,000-group tenant, `LIMIT 50` | 3.753 ms | 3.796 ms |
| D6 PK probe | – | 0.023 ms |
| onboard probe by `keycloak_group_id` | – | 0.021 ms |

- **An extra argument for (e), from evidence.** Today one group `DELETE` under `SET NULL` costs **502 ms** in the 400k tenant. The RI query cannot use `records_tenant_group_open`, because that index is partial (`state <> 'closed'`), so it falls back to a scan of the tenant prefix. A sync run that drops 100 groups spends about 50 s inside its transaction. Tombstoning removes that cost.
  - Under `RESTRICT` the same check runs only on an owner hard-delete (Epic 30 offboarding), after the tenant's records are gone, so the scan is short.
  - Again, no new index.

## Blocking
1. **The FK swap must ship after the tombstoning code, as a separate contract step.** Q-D lists "migration (D1, D2)" before the groupsync change. Failing scenario: the migrate job runs D2, and the old sync binary is still deployed. Its `DELETE FROM groups … = ANY(gone)` (`internal/groupsync/apply.go:389`) hits `RESTRICT` on any referenced group, and the tenant's whole sync rolls back, deprovisioning included. That is the TICK-452 failure again. Required order:
   - (i) expand: `ADD COLUMN deleted_at`, and sync tombstones instead of deleting;
   - (ii) once that is deployed: the FK swap migration, the `VALIDATE` migration, and revoking `DELETE` on `groups` from `nava_groupsync`. This is the same split 00010, 00011 and 00012 used for principals.

   This is a constraint on how the migration is shaped, not the Technical Lead's sequencing call, so it belongs in the erratum.
2. **D4's wording "every reader of `groups` must filter `deleted_at IS NULL`" is wrong for two readers.**
   - **`apply.go:285`** (sync's own diff read) must see tombstones. If it filters them out, a group that returns from Keycloak looks new. Sync then INSERTs it, hits `groups_keycloak_group_key` (`23505`), and the sync fails, which breaks CI-E5-4.
   - **`internal/onboard/store.go:196-214`** must filter, but the RFC's list does not name it. If the admins group is tombstoned and `RecordAdminGroup` runs again, the INSERT's `ON CONFLICT DO NOTHING` leaves the tombstone in place. The unfiltered `EXISTS` then returns true, and a `tenant_admin_group` row is written that points at a tombstoned group. With the filter, the call returns `ErrAdminGroupMissing` instead. Security (Q-B) should confirm that this is the intended outcome.

   Fix: replace "every reader" in D4 with an explicit list. Sync reads all rows; `pgstore` resolver, onboard, pickers, ACL resolution and D6 filter.

## Non-blocking
- Either `RESTRICT` or `NO ACTION` passes R29. `RESTRICT` is fine, because nothing needs the check deferred.
- R-E5-3's revisit threshold (more than 10% of a tenant's groups tombstoned) has headroom: at 10%, the filter changed the reader timings by under 0.05 ms.
- The race R-E5-2 is real, and I confirmed the reason by lock-mode analysis rather than a spike: the FK check takes `FOR KEY SHARE`, and the tombstone takes `FOR NO KEY UPDATE`, and those two do not conflict. Accepting it under D7 is correct.

## Technical Architect — comment 01a1a37f-0be0-76de-82c8-865a553f8cdc (2026-11-03T22:19:34Z)

## Critique — round 1 (RFC-011 rev 0)

**VERDICT: APPROVE.** There are no blocking objections. I spiked both of the RFC's unverified claims (§6.1 and §6.2) on the pinned `postgres:16.10` image, and both hold. Option (e) with tombstones works mechanically as designed. Five non-blocking items below need folding into the erratum text or the P5-13c plan.

Invariant baseline: ADR-001 §1, I1–I9.

### Findings

| # | Label | Mechanism | Evidence | Known / Inferred |
|---|---|---|---|---|
| T1 | **confirmed** | §6.1: the FK action runs as the table owner, without RLS | Spike S0. `nava_groupsync` holds no privilege on `records` (`has_table_privilege` = `f`). `records` has FORCE RLS and **no policy for its owner**. The `SET NULL` still nulled the row: `PROBE BEFORE UPDATE on records: current_user=own session_user=nava_groupsync depth=2`. Version stayed 7, `updated_at` stayed the same, nothing reached the audit log. | Known |
| T2 | **confirmed, with a wording fix** | Why the failure surfaced at COMMIT | Spike S1. The RI UPDATE runs as the owner, but the deferred commit check is **queued** and runs at COMMIT as the session role: `CHECK fires: current_user=nava_groupsync` → `ERROR: permission denied for table audit_log` (42501). The whole transaction rolls back, so `g2` and the record both survive. The COMMIT timing comes from `DEFERRABLE INITIALLY DEFERRED`, not from `ri_triggers.c`. Fix §6.1: "the check runs as `nava_groupsync`, which holds no `audit_log` grant (it is not a writer in the manifest), so it fails with 42501 at COMMIT." | Known |
| T3 | **confirmed** | Option (c): `WHEN (pg_trigger_depth() = 0)` on the constraint trigger | Spike S2: the delete commits, the record goes G→NULL at v7, `updated_at` is unchanged, and the audit log holds **0 rows**. The RFC's "nothing detects it" holds. S3: a depth-0 raw `nava_app` UPDATE is still caught (`audit_missing`). **S4: any write issued from inside any trigger escapes too.** `nava_app` inserted into an unaudited table whose trigger updated `records`, and the change committed with 0 audit rows. The rejection stands, and the scope of the gap is larger than the RFC says. | Known |
| T4 | **confirmed** | R29 catalogue query | Before the change it returns `records_assignment_group_fkey`, with `confdeltype = n`, `confupdtype = a` and `confdelsetcols = {4}`, so the PG15+ column-list form is caught. After the change it returns 0 rows. The query is in the proposed text below. | Known |
| T5 | **confirmed** | D2 and D1 mechanics | Spike S5: the tombstone UPDATE by `nava_groupsync` commits and fires nothing on `records`. Its `DELETE` is denied (`permission denied for table groups`). S6: the owner's delete of a referenced group fails with 23503 under `RESTRICT`. `DROP` plus `ADD … NOT VALID` in one transaction, then `VALIDATE`, ran cleanly. Lock levels and timing are Q-C, for the Postgres Specialist. | Known |
| T6 | **confirmed** | D6 is the only guard against new references to a tombstoned group | Spike S7: the FK accepts `records` → tombstoned `g5`, because the row exists. Only the D6 pipeline step refuses it. That is acceptable: the pipeline is the only `records` writer. | Known |
| T7 | **non-blocking** | D1/D2 do not match the schema and grants | (1) `groups` has no `synced_at` column (`db/migrations/00003_identity_tables.sql:53-60`). Either drop it from D1 or add the column in the migration. (2) `nava_groupsync` may UPDATE only `groups (path)` (`db/grants.yaml:237`). D2 must also **grant `UPDATE (deleted_at)`**, or D1 fails with 42501. (3) The `groups_delete` policy must be dropped together with the grant, or the policy/grant catalogue check flags a leftover policy. (4) Tests that delete groups as `nava_groupsync` must be rewritten: `identity_tables_test.go:89,161`, `records_pipeline_isolation_test.go:30,33`, `records_pipeline_test.go:386`, `tenant_admin_group_test.go:267`, and `docs/data/identity-bench/03_sync.sql:22`. | Known (source) |
| T8 | **non-blocking** | D4: what actually keeps I5 | Group grants resolve **through `group_members`** (`internal/identity/pgstore/pgstore.go:39,53`). Once the memberships are deleted, a tombstoned group grants nothing, with or without a `deleted_at` filter on readers. The reader filters are needed only for pickers, for D6 and for the D7 read model. The remaining risk is that "a tombstoned group has no members" depends on discipline in `apply.go`. A sync bug that inserts a membership into a tombstoned group would grant through it. **Mechanical option, spiked (L1–L4):** add `groups.live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED` and `UNIQUE (tenant_id, id, live)`. Add `group_members.live boolean NOT NULL DEFAULT true CHECK (live)` with an FK on `(tenant_id, group_id, live)`. Result: tombstoning a group that still has a member → 23503. Deleting the members and then tombstoning in one transaction → succeeds. Inserting a member into a tombstoned group → 23503. Reviving and inserting → succeeds. I am asking the Postgres Specialist to cost this (an extra column and index on `group_members`) against an isolation-registry test that does the same job. Either one is enough; the D4 sentence alone is not. | Known (spike), cost Inferred |
| T9 | **non-blocking** (Security to rule) | D5 adds a grant it does not need | The `tenant_admin_group` header says *"no runtime role may UPDATE or DELETE it"* (`20261031160000_tenant_admin_group.sql`), and D5 gives `nava_groupsync` DELETE on it and on `tenant_auditor_group`. Once memberships are deleted, `RequireTenantAdmin` already denies everyone (fails closed). `AdminGroupDeleted` can be computed from the tombstoned set, using the existing query at `internal/groupsync/apply.go:384` with `gone`. **Behaviour difference:** if the pointer is kept and the same Keycloak group comes back (D1 revives the same platform id), the admins are restored. Under D5, the tenant stays admin-less. My recommendation: drop D5, which means fewer grants and fewer write paths. Security decides whether revival should restore admin. | Known (source), recommendation |
| T10 | **non-blocking** | D9 checks only the top-level `required` | Markers are legal only on top-level properties (`internal/metadata/compile.go:330`). However, the root may carry `allOf`/`anyOf`/`oneOf`/`if`/`then`/`else`/`not`/`dependentSchemas` (`compile.go:67-72`), as well as `dependentRequired`. Failing case: `{"properties":{"pw":{"type":"string","writeOnly":true}},"allOf":[{"required":["pw"]}]}` passes D9 as written, and every create of that type is then refused with 422. The same happens with `"dependentRequired":{"a":["pw"]}` or `"if":{},"then":{"required":["pw"]}`. Fix: D9 refuses a definition in which any `required` or `dependentRequired` entry, at any walked subschema whose instance location is the root, names a marked property. (A root `$ref` is already refused when a marker exists, `checkMarked`.) The cost is bounded: the tenant locks itself out of creating that type, and nothing leaks. | Inferred from source, not spiked |
| T11 | **confirmed**, with a documentation note | D8: no secrets in record payloads | R6 is the only enforcement. The validator treats `writeOnly` as an annotation, and the marker is top-level only (`compile.go:330-341`), so R6 can see every marked path. The erratum should state that JSON Schema 2020-12 `writeOnly` ("accepted, not returned") is **narrowed** here to "never accepted in a record". Otherwise, tenants who import schemas will read it the standard way. | Known (source) |

**What would change my verdict:** a read path that grants through `groups` without joining `group_members`. I found none: every authz and resolver read joins `group_members` or a pointer table. If Security finds one, T8 becomes blocking.

### Proposed R29 text (for the erratum and CI-A3)
> **R29 · no referential actions on audited tables.** For every table in `db/audited-tables.yaml`, every foreign key whose referencing table is that table has `confdeltype` and `confupdtype` in {`a` NO ACTION, `r` RESTRICT}. CI-A3 asserts this from `pg_constraint`; a partitioned table is checked on its parent. The commit-check trigger already refuses an FK-action write at COMMIT (TICK-452). R29 moves that failure from a production sync stall to CI.

```sql
SELECT c.conrelid::regclass, c.conname, c.confdeltype, c.confupdtype
  FROM pg_constraint c
 WHERE c.contype = 'f' AND c.conparentid = 0
   AND c.conrelid = ANY ($1::regclass[])          -- audited tables from the manifest
   AND (c.confdeltype NOT IN ('a','r') OR c.confupdtype NOT IN ('a','r'));
-- expect 0 rows
```

### Spike report (RFC-011 S-FK)
- **Verdict:** pass. Both of §6's claims hold, and (c) leaks wider than stated (T3, S4).
- **Hypotheses (stated before running):** H1. The RI `SET NULL` UPDATE runs as the `records` owner, with RLS not applied. Pass if the row is nulled although the owner has no policy under FORCE RLS. H2. The deferred check runs at COMMIT as the session role. Pass if `current_user = nava_groupsync` in the check. H3. Under (c), the FK write commits with no audit row while a depth-0 raw write is still caught. H4. The R29 query flags the column-list `SET NULL`.
- **Setup:** `postgres:16.10` (Debian 16.10-1.pgdg13+1, x86_64), Docker 29.9.2, Windows 11 host. Tables reduced from `00003_identity_tables.sql` and `20261101190000_records_pipeline.sql`. The commit-check function is reduced from `20261103100000_audit_commit_check.sql` to the same `PERFORM` decision. 6 groups and 6 records. Run on 2026-11-04.
- **Rerun:** `docker run -d --name rfc011 -e POSTGRES_HOST_AUTH_METHOD=trust postgres:16.10`, then `psql -U postgres -f setup.sql`, `-f run.sql` and `-f live.sql`. Suggested archive path: `docs/adr/ADR-004/spikes/technical-architect/rfc-011/`.

*(spike files archived separately)*

*(spike files archived separately)*

*(spike files archived separately)*

*(spike files archived separately)*

## CTO — comment 01a1a37f-7cbc-7e5b-ab58-fc4ba99adb21 (2026-11-03T22:20:03Z)

Round 1 is closed. The Technical Architect and Security voted APPROVE; Postgres voted CHANGES with 2 blocking items.

Solution Architect: make one revision (rev 1) that resolves every blocking item, and fold in the rulings below. Do not open a new debate.

**Blocking (Postgres)**
1. **Expand, then contract.** Step (i): `ADD COLUMN deleted_at`, and sync tombstones instead of deleting. Step (ii), after (i) is deployed: swap the FK, run `VALIDATE` as its own migration, revoke `DELETE`, and keep `SET LOCAL lock_timeout`. State this order in the erratum as a migration constraint.
2. **D4: list the readers instead of saying "every reader".** Sync's diff read (`apply.go:285`) sees all rows. The resolver, onboard (`store.go:196-214`), pickers, ACL resolution and D6 filter `deleted_at IS NULL`. The onboard outcome is `ErrAdminGroupMissing`; Security already asked for that filter, so take it as confirmed.

**CTO rulings on reviewers who disagreed**
- **Guard against members in a tombstoned group: adopt T8 (the generated `live` column plus the composite FK), not S1's RLS `WITH CHECK`.** Reason: the FK covers both directions (L1 and L3) and applies to every role, owner included. RLS covers inserts only, and only for non-owner roles. That leaves exactly the "discipline in `apply.go`" gap open. Cost: one constant-default boolean column on `group_members` and a unique index on a small table. Postgres to confirm that at plan time; it does not block. Keep S1's test (`42501` becomes `23503`, and admin is denied).
- **D5: keep it, in S2's form.** Security owns this rung and has ruled that deleting the pointer is correct, because it stops a revived group from silently restoring admins. Add a DELETE policy restricted to pointers of tombstoned groups, a matching catalogue entry, and split the existing denial test. The Technical Architect's preference in T9 for dropping D5 is recorded as dissent.

**Fold in (non-blocking)**
- **Schema and grants:**
  - Drop `synced_at`.
  - The `groups` UPDATE grant becomes exactly `[path, deleted_at]`.
  - Drop `groups_delete` together with the grant.
  - The `tenant_auditor_group` rule applies when that table lands.
  - List the T7(4) tests to rewrite.
- **§6.1 wording:**
  - Use T2's wording.
  - In the rejection of (c), add T3/S4: any write issued from a trigger also escapes the audit.
  - R29 uses the Technical Architect's text and query (`conparentid = 0`).
- **D9:** walk `allOf`/`anyOf`/`oneOf`/`if`/`then`/`else`/`dependentSchemas`/`dependentRequired` (T10). State that `writeOnly` is narrowed to "never accepted in a record" (T11).
- **Tests:**
  - S4 equal-422 test in CI-E5-6.
  - CI-E5-7 checks that no value reaches logs, traces or the idempotency `response_body`.
- Archive both spike bundles under `docs/adr/ADR-004/spikes/`.

The Technical Architect then re-checks the blocking items only.

## Technical Architect — comment 01a1a387-f770-7255-8b13-bc5c4bead09b (2026-11-03T22:29:19Z)

## Critique — round 2 (RFC-011 rev 1): blocking items and the step (i) interaction

**VERDICT: APPROVE, on one condition.** PG-B1 and PG-B2 are resolved. The step (i) interaction check found **one verified defect (I1)**. It has a one-line mechanical fix and is not a design dispute, so it does not need a third round. The ADR must carry the fix below. If it does not, sync stalls whenever the auditors group is tombstoned. I spiked everything below on `postgres:16.10`. The code is at the end.

### Blocking items
| Item | Verdict | Evidence |
|---|---|---|
| **PG-B1: D10, expand then contract** | **resolved** | Step (ii) comes after step (i)'s binaries, (ii-b) `VALIDATE` is a separate migration, and `lock_timeout` is kept. Spike R1 also confirms that **the old binary still works against the step (i) schema**. Its hard `DELETE FROM groups` commits with D11 in place: the cascade removes the members first, so the new NO ACTION FK sees none, and `SET NULL` still clears the record, as it does today. So step (i) can deploy before the new sync. |
| **PG-B2: D4's reader list** | **resolved** | Every non-test reader of `groups` at `27860f4` is on the list: `apply.go:285`, `pgstore.go:39,52`, `onboard/store.go:196-214`. R8: the resolver join with the filter stays on index scans (78 buffers, 15 rows). One addition, not blocking: put **D5's policy subquery** (`g.deleted_at IS NOT NULL`) in the "sees all rows" group, so nobody "fixes" it with the D4 filter. |

### Interaction: D11 + D5 + D1's order inside step (i)
| # | Label | Finding | Evidence |
|---|---|---|---|
| I1 | **blocking, fold into the ADR** | **D5 as written does not work on `tenant_auditor_group`.** `nava_groupsync` has no grant and no SELECT policy on it (the branch's `db/grants.yaml` lists only `tenant_admin_group: { SELECT: [] }` for sync, and `tenant_auditor_group_select` is `TO nava_app, nava_worker`). A `DELETE … WHERE` needs SELECT on the columns it reads and the row has to pass a SELECT policy. | R2b, rev 1 as written: `ERROR: permission denied for table tenant_auditor_group`. The tenant's whole sync transaction rolls back, which is the TICK-452 stall again, triggered by tombstoning the auditors group. R2c, with the grant but no policy: **`DELETE 0`, silently.** The pointer survives, still pointing at the tombstoned group, so a revived auditors group regains audit access. That is exactly the outcome S2 rules out. R3, with the grant and a SELECT policy: `DELETE 1`. |
| I2 | **confirmed** | D1's order meets D11 and D5 in one transaction | R2: deleting the members, then tombstoning, then deleting the admin pointer gives `DELETE 1`, `UPDATE 1`, `DELETE 1`, and the commit succeeds. R4: the D5 policy refuses a live pointer (`DELETE 0`). R5: tombstoning before deleting the members fails with `23503 … group_members_group_live_fkey`. |
| I3 | **confirmed, with a plan note** | Revival order | R6: inserting a member before `deleted_at = NULL` fails with `23503`; reviving first, then inserting, succeeds. `apply.go` already sends group writes (batch 1) before memberships (batch 2), so the revive belongs in batch 1. The tombstone and the pointer delete belong in batch 2, **after** the membership deletes. |
| I4 | **non-blocking** | Old binary during the rollout | R7: an old sync instance that meets a tombstoned group which came back in Keycloak does not revive it. It inserts members, and D11 fails the transaction with `23503`. That fails closed and lasts only while old and new sync instances overlap. Inferred fix: deploy sync as stop-old, then start-new. |
| I5 | **non-blocking** | Step (i)'s locks | Measured with 50k groups and 2.5M memberships, in one transaction: adding the `live` column rewrites `groups` (201 ms, new relfilenode), `UNIQUE (tenant_id, id, live)` takes 39 ms, and adding `group_members.live` with its `CHECK` takes 160 ms. All of that runs under **ACCESS EXCLUSIVE on both tables**, which blocks the resolver for about 0.4 s. The FK added `NOT VALID` takes 1 ms; `VALIDATE` takes 166 ms under a non-blocking lock. Recommendation for the Postgres Specialist at plan time: add the `CHECK` as `NOT VALID` and validate it together with the FK, and keep `lock_timeout` on the rewrite migration. |

### Required text for I1 (D5 and D10 step (i))
> D10 step (i) also grants `nava_groupsync` `SELECT (tenant_id, group_id)` on `tenant_auditor_group`, and adds `nava_groupsync` to `tenant_auditor_group_select`'s `TO` list, wherever that table lands. Both the `DELETE … WHERE` and the "was the recorded group removed" probe need them. CI-E5-5b gains a check: tombstoning the recorded auditors group commits, and its pointer row is gone.

The same pair already exists for `tenant_admin_group` (`tag_select`, `SELECT: []`), which is why the admin case passes.

### Spike setup and how to rerun
- **Image and host:** `postgres:16.10` (Debian 16.10-1.pgdg13+1), Docker 29.9.2, run on 2026-11-04.
- **Data:** 10 tenants with 5,000 groups each, 50 members per group (2.5M rows), and one named tenant for the scenarios.
- **Schema:** `groups`, `group_members`, both pointer tables and `records` with their branch FKs, RLS policies and grants.
- **Rerun:** `docker run -d --name r2 -e POSTGRES_HOST_AUTH_METHOD=trust postgres:16.10`, then `psql -U postgres -f r2.sql`.
- **Archive with the rest:** `docs/adr/ADR-004/spikes/technical-architect/rfc-011/`.

*(spike files archived separately)*

*(spike files archived separately)*

## CTO — comment 01a1a388-3ce4-758e-8cde-91436ab9613d (2026-11-03T22:29:37Z)

**RFC-011 is approved by all reviewers.** Round 1: Security APPROVE. Round 2: Technical Architect APPROVE. The Postgres blocking items are resolved, and the Technical Architect confirmed that in round 2. Condition I1 is a spiked mechanical fix, not a design change, so no third round.

Solution Architect: write the ADR-004 erratum E5 (status PROPOSED) and archive RFC-011. Put it in one docs PR under `docs/adr/`, with both the `.md` and the self-contained `.html`. The PR must carry:
- **I1, verbatim, in D5 and in step (i) of D10:**
  - grant `nava_groupsync` `SELECT (tenant_id, group_id)` on `tenant_auditor_group`;
  - add `nava_groupsync` to the `TO` list of `tenant_auditor_group_select`;
  - extend the CI-E5-5b check.
- **In D4's "sees all rows" list:** the D5 policy subquery.
- **Plan notes:**
  - I3: the revive goes in batch 1; the tombstone and the pointer delete go in batch 2, after the membership deletes.
  - I4: roll out sync as stop-old, then start-new.
  - I5: add the `CHECK` as `NOT VALID` and validate it later; keep `lock_timeout` on the rewrite migration.
- **Dissent:** T9, the Technical Architect's preference for dropping D5.
- **Spike bundles** from both rounds, under `docs/adr/ADR-004/spikes/`.

I1 widens a grant on an identity table. Request a review from the Security & Multi-Tenancy Reviewer on the docs PR before you give your verdict.
