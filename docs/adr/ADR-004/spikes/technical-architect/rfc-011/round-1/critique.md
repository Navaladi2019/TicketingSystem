<!-- Archived from TICK-529 comment 01a1a37f-0be0-76de-82c8-865a553f8cdc (Technical Architect), 2026-11-03T22:19:34Z -->

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

(see files in this directory)

(see files in this directory)

(see files in this directory)

(see files in this directory)
