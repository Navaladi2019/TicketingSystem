---
type: adr-amendment
id: ADR-004-A1
title: "Amendment A1 to ADR-004: only nava_audit_reader can read audit_log.diff and diff_salt, through the masking reader's own pool"
status: ACCEPTED
accepted: 2026-11-04 (Solution Architect verdict, TICK-549 comment thread 01a1a557-490e)
date: 2026-11-04
issue: TICK-549
rfc: docs/adr/ADR-004/rfc/RFC-012-adr-004-a1-diff-read-grant.md (rev 1, archived with rev-0/ and rev-1/; thread under TICK-549 comment 01a1a4f3-eae2-72c8-a67c-e0972afc4880)
supersedes: none
amends: "[[ADR-004-audit-log]] R7, R12, R14, CI-A3; the `audit_log_select` and `entity_definition_versions_select` policies; `db/grants.yaml` (`nava_app` on `audit_log`)"
source: "TICK-541 (PR #443) Security review: the static guard `TestAuditDiffReadOnlyInReader` cannot see runtime-built SQL, so `nava_app`'s SELECT on `diff` and `diff_salt` is protected by convention only"
deciders: Solution Architect (author); Technical Architect (round 1 APPROVE with spike, comment 01a1a50c-0397); Security & Multi-Tenancy Reviewer (round 1 CHANGES B1, comment 01a1a504-ed7b; round 2 APPROVE with the audit_log_select condition, comment 01a1a54c-a718); Postgres & Data Modeling Specialist (round 1 CHANGES B2, comment 01a1a547-6361; round 2 APPROVE with the one-PR condition, comment 01a1a54d-76fd); CTO (routing and rulings on A2, records SELECT and AR-A1-1, comment 01a1a547-dad2)
---

# ADR-004 amendment A1: only the masking reader's role reads the raw diff

> **Status: ACCEPTED** on 2026-11-04 by the Solution Architect (TICK-549). All three reviewers approved RFC-012 rev 1: the Technical Architect in round 1, and Security and Postgres in round 2. Their round-2 conditions are written in as rules (**[Sec R2]**, **[PG R2]**). The Technical Architect's fidelity check of PR #452 was APPROVE (comment 01a1a556-c7a5). Its two wording fixes, to A1-9 and to `diff-denied`, are folded in per the CTO's ruling (comment 01a1a557-490e). No CTO tie-break was needed, and no human sign-off is required under the current policy. Changing a rule here needs a superseding ADR or a further amendment.
>
> **Invariants source.** The project description defines no invariants. As in ADR-004, this amendment is checked against [[ADR-001-platform-blueprint]] §1, I1–I9.

## Context

ADR-004 R7 masks secret pointers in `audit_log.diff` when the diff is read, and only `internal/audit/read` runs the masker. `nava_app` held column SELECT on `diff` and `diff_salt`, so any `nava_app` query outside that package returned the raw diff. That includes a query for a same-tenant user with no `mask` permission. RLS limits rows, not columns. The TICK-541 static guard reads Go source text, so it cannot see SQL built at runtime (`strings.Builder`, a column list in a package `var`, SQL outside Go). Nothing reads the diff outside the reader today. This closes a latent risk, not a live exploit.

## Decision

**A dedicated runtime login role, `nava_audit_reader`, is the only runtime role that can read `audit_log.diff` and `diff_salt`.** It has its own PgBouncer login and its own `pgxpool`, held by `internal/audit/read`. `nava_app` loses both column grants. The TICK-541 guard stays as a second layer.

| # | Rule |
|---|---|
| **A1-1** | **The role.** `nava_audit_reader` is in `db/grants.yaml`: `LOGIN`, `pgbouncer_login: true`, member of nothing, owns nothing, `NOSUPERUSER NOBYPASSRLS` (I3). Grants: SELECT on `audit_log (tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id, causation_depth, request_id, backfilled, content_sha256, diff_sha256, diff_truncated, diff, diff_salt, seq, sealed_at, erased_at)` and on `entity_definition_versions (definition_id, schema)`. Nothing else: no INSERT, UPDATE or DELETE, nothing on `records`, nothing on any partition directly. `txid` stays out of the grant and out of the reader's `columns` constant. |
| **A1-2** | **The revoke.** `nava_app.tables.audit_log.SELECT` loses `diff` and `diff_salt` and keeps every other reader column. `nava-db migrate` revokes them, because they are no longer listed. The generated R12 commit-check SELECT (`internal/db/audited/manifest.go`) never includes them. |
| **A1-3** | **Policies [Sec R2].** One migration, with the canonical predicate unchanged: <br>`ALTER POLICY audit_log_select ON audit_log TO nava_worker, nava_erasure, nava_audit_reader;` <br>`ALTER POLICY entity_definition_versions_select ON entity_definition_versions TO nava_app, nava_worker, nava_audit_reader;` <br>**`nava_app` is not in `audit_log_select`.** Its row access to `audit_log` comes only from the generated `audit_log_writer_select` (`internal/db/audited/policies.go`), so a hand-written copy cannot outlive the generator. CI asserts the exact role list of `audit_log_select` (A1-9). |
| **A1-4** | **The masker fails closed.** `DefinitionMasker.MaskedPaths` returns `ErrNoDefinitionVersions` when `EntityDefinitionID` is non-nil and no version row is visible. `mask()` propagates it. The request fails with a 500 and a log line that names the definition id, never a value, and returns no diff or salt. An entry with no definition still has no masked path. This rests on the assumption that every published definition has at least one version row (D-3-4); the `reader-masks` fixture asserts it. |
| **A1-5** | **The pool.** `read.OpenPool(ctx, dsn) (*Pool, error)` returns an opaque type that exposes no `*pgxpool.Pool`, with `MaxConns = 5`. `Reader.List`, `Reader.RecordEntries` and the export pages each run in their own `tenancy.InTenantTx` on that pool, with the read and the masker in the same transaction. The tenant setting stays transaction-local (`set_config(…, true)`). There is no role switch. |
| **A1-6** | **The DSN.** `NAVA_AUDIT_READER_DATABASE_URL` and its config key are read only by the `cmd/nava` config loader, which passes the DSN only to `read.OpenPool`. |
| **A1-7** | **Handlers.** `RequireAuditPermission`, cursor decoding, the export lease and the `audit/export` entry stay on `nava_app`, in their own transaction. Record history is two transactions. First, `nava_app` reads the record: same 404 for missing, unreadable and other-tenant records. Only then does the reader pool read its entries. READ COMMITTED gives each statement its own snapshot, so this adds no race (PG 16 §13.2.1). |
| **A1-8** | **Pooler cap.** `[users] nava_audit_reader = pool_size=5` in `deploy/compose/pgbouncer/pgbouncer.ini` and `scripts/ci/pgbouncer/isolation.ini`. No `CONNECTION LIMIT`. |
| **A1-9** | **CI-A3 (extended).** The catalogue gate checks two things. First, no runtime role except `nava_audit_reader` holds SELECT on `diff` or `diff_salt`: `has_column_privilege` over every runtime role (every manifest role except the database owner `nava_migrator`, which owns `audit_log` and so holds every privilege), on `audit_log` **and every `pg_inherits` child**. Second, no runtime role holds any privilege directly on a partition. **The role list of `audit_log_select` equals exactly `{nava_worker, nava_erasure, nava_audit_reader}`** (`pg_policy.polroles`) **[Sec R2]**. |
| **A1-10** | **The guard stays as layer 2.** <br>• **Rule 1 (new, primary)** checks Go identifiers and string constants. Only `internal/audit/read` may call `read.OpenPool`. Only the `cmd/nava` config loader may name `NAVA_AUDIT_READER_DATABASE_URL`, its config key or the DSN field. <br>• **Rule 2** is the existing TICK-541 SQL-text rule, kept as an early warning. After A1, a miss becomes a `42501` at Execute, not a leak. |
| **A1-11** | **Delivery [PG R2].** The manifest role, its password `DBROLES_PASSWORD_NAVA_AUDIT_READER` (in `compose.yaml`, the CI isolation stack and `.env.example`), the pooler lines, the policy migration, the revoke and the red tests make up §9 step 1. The reader pool, the fail-closed masker, the DSN loader and the handler changes make up step 2. **Steps 1 and 2 land in one PR. Neither merges alone**, because between them the audit endpoints return 500. db-roles fails closed for every role when a LOGIN role has no password (`roles.go`), so the password ships in the same change as the manifest. Guard rule 1 may follow in a later PR. Prefer landing inside story TICK-419 before it merges to `main`, so `main` never carries the `nava_app` grant. The Technical Lead decides. |

### Tests (red first)

Every denial asserts SQLSTATE `42501`, never the message text.

| Id | Case |
|---|---|
| CI-A3 | A1-9, including the `pg_inherits` walk and the exact `audit_log_select` role list. Red today. |
| `ADR-004-A1/diff-denied` (registry) | The cases run as `nava_app` through PgBouncer with tenant A set, and each must fail with `42501`: <br>• `SELECT diff` and `SELECT diff_salt`; <br>• `SELECT *`, `to_jsonb(a)` and `a::text`; <br>• the WHERE oracle `… WHERE diff::text LIKE '%x%'`, and `ORDER BY diff_salt`; <br>• `FROM audit_log_default` and `FROM ONLY audit_log`; <br>• `COPY (SELECT diff …)` and `COPY audit_log (diff)`; <br>• a CTE `SELECT *` and `LATERAL json_array_elements(a.diff)`; <br>• `UPDATE … RETURNING diff` and `INSERT … RETURNING diff_salt`. <br>The same probes run as `nava_worker`, `nava_relay`, `nava_purge`, `nava_onboard` and `nava_groupsync` through PgBouncer, and as `nava_erasure` direct to PostgreSQL, because `nava_erasure` has no `pgbouncer_login` (through PgBouncer it would fail at login, not with `42501`). As `nava_audit_reader`, INSERT, UPDATE and DELETE on `audit_log` and any SELECT on `records` must also fail. |
| `ADR-004-A1/reader-scoped` | As `nava_audit_reader`, tenant A reads its own diff. Tenant B set and filtering on A's id reads 0 rows. No tenant set reads 0 rows. `txid` gives `42501`. |
| `ADR-004-A1/reader-masks` | The fixture is a definition with an `x-nava-secret` property and a record change that sets it. `GET /api/v1/audit/entries`, as a principal with `audit.read` and no `mask` permission, must return `{"redacted":"mask"}`, `diff_redacted = true` and no salt. Red if A1-3's `entity_definition_versions` change is left out. |
| unit, `read` masker | 0 version rows for a non-nil definition gives `ErrNoDefinitionVersions`, and `mask()` returns no entry. A nil definition gives no paths. |

Existing tests that change. The list comes from the Postgres full-suite run with the revoke applied: 1103 passed, 6 integration failures and 1 unit failure, all of them listed here.

| Test | Package | Change |
|---|---|---|
| `TestListAuditEntriesPagesFiltersAndGate`, `TestListAuditEntriesTokenBindingAndTenant`, `TestListAuditEntriesMasksSecretProperties` | `internal/server` | move to the reader pool |
| `TestRecordHistoryMasksSecretProperties`, `TestPatchHistoryPagesNewestFirst` | `internal/server` | move; the record check stays on `nava_app` |
| `TestP510bAppReaderGrantAndTenantBoundary` | `internal/db/dbtest` | **inverted**: `nava_app` gets `42501`, and the tenant-boundary half moves to the reader |
| `TestLoadRealManifest` | `internal/db/audited` | **inverted**: only `nava_audit_reader` holds `diff` and `diff_salt` |

### Changes to ADR-004 text

- **R7** gains: "The raw `diff` and `diff_salt` are readable only by `nava_audit_reader`, whose pool only `internal/audit/read` holds. The masker fails closed (A1-4)."
- **R12:** `nava_app`'s reader list no longer includes `diff` or `diff_salt`.
- **R14:** `nava_audit_reader` is added to the roles that never write `audit_chain_head`.
- **CI-A3:** gains A1-9.

## Options considered

| Option | Verdict | Why |
|---|---|---|
| (a) Revoke, plus a narrow `SECURITY DEFINER` function | rejected | It breaks I3: no definer function may be reachable from a runtime role. EXECUTE goes to `nava_app`, which every package runs as, so this just moves the TICK-541 problem into SQL. |
| (b) `nava_audit_reader` entered with `SET LOCAL ROLE` | rejected as written | `SET ROLE` needs membership (PG 16 docs). I3 forbids memberships, and CI-3 checks `pg_auth_members`. Any package can also send `SET ROLE`. The chosen design is (b) without the membership. |
| (c) Mask in the database (view or function) | rejected | CI-1 bans views without `security_invoker`. A `security_invoker` view needs the very grant being removed. A definer function breaks I3. It would also fork the R7 masker into PL/pgSQL. |
| (d) Keep only the static guard | rejected as the end state | CI cannot prove a denial, and the bypasses are demonstrated. It is the interim state until A1 ships. Its guard stays as layer 2. |
| **(e) Separate login role and pool** | **chosen** | Postgres enforces the column wall for every `nava_app` query. I3 holds, there is no role switch, and the masker stays in Go. |
| Grant `records` SELECT to the reader (to keep history in one transaction) | rejected | It gives no stronger consistency under READ COMMITTED, and it puts every record payload behind the credential that already reads raw diffs. |
| `ALTER ROLE … CONNECTION LIMIT` as the cap | rejected | `grants.AllowedSettings` cannot express it, and it caps by failing logins instead of queueing them (Postgres spike). |

## Accepted risks

| Id | Severity | Risk | Trigger to revisit |
|---|---|---|---|
| **AR-TICK-549-01** | Low | The reader DSN is on the nava-core host. Code inside nava-core that deliberately opens a second pool with it reads raw diffs. Guard rule 1 and review cover this; Postgres cannot, because the process is one trust domain and that code already holds the `nava_app` credential. Cross-tenant reads stay closed (FORCE RLS, tenant per transaction from the validated principal). Accepted by Security in round 1. | A second service needs audit reads, or tenant code (I8) runs in-process. |
| **AR-TICK-549-02** | Info | One more pool: at most 5 server connections **per PgBouncer instance** (5 × N for N instances), and 5 client connections per nava-core process. | More than one PgBouncer instance in front of the same PostgreSQL. |

## Evidence

- **Technical Architect spike, PASS** (`docs/adr/ADR-004/spikes/technical-architect/rfc-012/round-1/`): PG 16.10, PgBouncer 1.24.1, pgx 5.11.0.
  - `get_auth` serves the new role, and it logs in through a PgBouncer that was already running.
  - `nava_app` INSERT still commits without the SELECT.
  - 14 of 14 diff-read probes give `42501` for `nava_app` and `nava_worker`.
  - The reader is scoped to one tenant.
  - The denial happens at Execute, and the connection survives.
- **Postgres spikes** (`docs/adr/ADR-004/spikes/postgres/rfc-012/round-1.md`):
  - the full integration run with the revoke applied (the 7 tests above);
  - a policy scoped to other roles gives the reader 0 rows and no error (the B1 mechanism);
  - `pool_size=5` queues clients, while `CONNECTION LIMIT` fails their logins.
- **Out of scope:** TICK-550. Through PgBouncer, a Parse-time error is followed by `08P01` on the next transaction. It does not affect the `42501` denials here.
