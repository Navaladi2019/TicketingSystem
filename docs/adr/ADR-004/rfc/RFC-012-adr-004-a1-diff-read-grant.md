---
type: rfc
id: RFC-012
title: "ADR-004-A1: only the masking reader's role can read audit_log.diff and diff_salt"
status: REVISED (rev 1, after critique round 1)
date: 2026-11-04
issue: TICK-549
amends: ADR-004 (R7, R12); touches ADR-001 I3
author: Solution Architect
---

# RFC-012 · ADR-004-A1: Postgres enforces who reads the raw diff

> **Archived final text (rev 1).** Approved by the Technical Architect (round 1) and by Security and Postgres (round 2). Two round-2 conditions are written into [[ADR-004-A1-diff-read-role]] and override this text: (1) the `audit_log_select` role list is `nava_worker, nava_erasure, nava_audit_reader`, with no `nava_app` (the §4.1 line here is stale; `nava_app` reads rows through the generated `audit_log_writer_select`), and CI asserts that exact list; (2) §9 steps 1 and 2 land in one PR. Earlier revision: `rev-0/`. Spike: `../spikes/technical-architect/rfc-012/round-1/`. TICK-550 is the out-of-scope PgBouncer side finding.

> **Invariants source.** The project description defines no invariants (its description is empty). This RFC is checked against ADR-001 §1, I1–I9, the same source ADR-004 used. If the project is meant to carry its own list, the invariant checks below are incomplete until that list is published.
>
> **Rev 1 changelog.** B1 (Security): the reader role is added to the `entity_definition_versions` SELECT policy, and the masker now fails closed (§4.1, §4.2, §5). B2 (Postgres): §5 lists the 7 affected tests, 5 moved and 2 inverted. Folded-in rulings: the A2 cap (§4.3), no `records` SELECT, AR-TICK-549-01 and the DSN-source condition on guard rule 1 (§6, §7), the corrected (c) wording (§3), the TA items (password in §9 step 1, partitions in CI-A3, SQLSTATE assertions), and the Security N1 probes. The full response is in §12.

## 1. Recommendation

Use **option (e)**, a variant of (b) that the squad adds here: **a dedicated LOGIN role, `nava_audit_reader`, with its own PgBouncer login and its own `pgxpool` in nava-core.** The role is not a member of anything, and `nava_app` is not a member of it.

- `nava_audit_reader` gets SELECT on the reader's `audit_log` columns, `diff` and `diff_salt` included. It also gets SELECT `(definition_id, schema)` on `entity_definition_versions`, which the R7 masker needs. **Both tables' canonical SELECT policies name the new role.** Without that, FORCE RLS hides every schema row from it, and the masker would mask nothing (B1). It gets nothing else: no INSERT, UPDATE or DELETE, and no SELECT on `records`.
- **The masker fails closed.** An entry with an entity definition but no version row visible is an error, never an empty mask.
- `nava_app` **loses** SELECT on `audit_log.diff` and `audit_log.diff_salt`. It keeps SELECT on the other reader columns, so the audit handlers can still read envelopes.
- `internal/audit/read` is the only package that holds the reader pool. Its constructor `read.OpenPool(ctx, dsn)` returns an opaque type capped at `MaxConns = 5`. The DSN is read only by the `cmd/nava` config loader, which passes it only to that constructor. Every diff read runs in its own `tenancy.InTenantTx` on that pool, so `set_config('app.tenant_id', …, true)` scopes it exactly as it does today.
- **The TICK-541 static guard stays, as a second layer.** It is not removed. Its main rule is retargeted to the new capability and to the DSN's source, and its SQL-text rule stays as an early warning (§6).

Why not (a) or (b) as written? **Both break ADR-001 I3.** (a) makes a `SECURITY DEFINER` function reachable from a runtime role, which I3 forbids outright. (b) needs `nava_app` to be a member of `nava_audit_reader`, because `SET [LOCAL] ROLE` only works for members (PG 16 docs, confirmed by the TA). I3 says no runtime role is a member of any role, and CI-3 checks `pg_auth_members` recursively. (e) gets what (b) was after without the membership, and without a role switch that could outlive its transaction.

## 2. Context

- ADR-004 R7 masks secret pointers in the diff at read time. The masker runs only in `internal/audit/read` (`mask.go`, `definition_masker.go`).
- On the TICK-419 story branch, `nava_app` holds SELECT on every reader column of `audit_log`, `diff` and `diff_salt` included (`db/grants.yaml`). RLS limits rows, not columns. So any `nava_app` query outside the reader package gets the raw diff, including a query on behalf of a same-tenant user with no `mask` permission.
- `nava_worker` and `nava_erasure` already exclude both columns from their grants, and the TA's spike shows both denied on every probe. **`nava_app` is the only runtime role with the gap.**
- The canonical policies today: `audit_log_select … TO nava_app, nava_worker, nava_erasure` (`db/migrations/20261102233600_audit_log.sql:209`) and `entity_definition_versions_select … TO nava_app, nava_worker` (`db/migrations/20261031150000_entity_definitions.sql:110`). Policies are scoped to named roles, and nothing adds a new role to them automatically.
- TICK-541 (PR #443) adds `TestAuditDiffReadOnlyInReader`, a source-text check. It cannot see a query built with `strings.Builder`, a column list held in a package `var`, or SQL that is not Go text.
- Nothing outside the reader reads the diff today. The Postgres specialist confirmed this by running the full integration suite with the revoke applied (§5). This is a latent risk, not a live exploit.

### Who touches the diff columns

| Caller | Role | Needs | After (e) |
|---|---|---|---|
| `audit.Append` (records pipeline, config writers) | `nava_app`, `nava_worker` | INSERT `diff`, `diff_salt` | unchanged. No `RETURNING` or `ON CONFLICT`; INSERT with the SELECT revoked commits (TA spike A4). |
| S8 backfill (migrations B1/B2) | `nava_migrator` (owner, not runtime) | INSERT | unchanged |
| Sealer A2 | `nava_worker` | `diff_sha256` only | unchanged |
| Erase step (R23) | `nava_erasure` | UPDATE to NULL | unchanged |
| `GET /api/v1/audit/entries`, record history, export (R19–R21) | today `nava_app`; after (e) `nava_audit_reader` | SELECT `diff`, `diff_salt`, masked | **moves to the reader pool** |

## 3. Options against the constraints

The constraint rows come from the issue body; the I-rows are ADR-001.

| | (a) definer function | (b) `SET LOCAL ROLE` | (c) mask in SQL | (d) static guard only | **(e) reader login + pool** |
|---|---|---|---|---|---|
| Tenant setting stays per transaction | yes | yes, but the role switch rides the pooled connection | yes | yes | **yes; there is no role switch at all** |
| No RLS bypass; definer owner is not the table owner and has no BYPASSRLS | possible, but needs a NOLOGIN owner role holding SELECT on `diff` | yes | a view either needs the grant or bypasses column grants (see verdict) | yes | **yes: not the owner, NOBYPASSRLS, FORCE RLS applies** |
| Writers keep INSERT | yes | yes | yes | yes | **yes (spiked)** |
| CI-A3 and the registry can prove `nava_app` is denied | yes | yes | yes | **no** | **yes (spiked: 14/14 probes `42501`)** |
| **I3** least privilege | **violates**: a definer function reachable from `nava_app` | **violates**: `nava_app` must be a member | **violates** via a definer function, or is banned by CI-1 as a non-invoker view | holds | **holds**: a manifest role, LOGIN, member of nothing, owns nothing |
| I5 masks on read | moved, not enforced: any `nava_app` code can call the function | any `nava_app` code can run `SET ROLE` | enforced in the DB | convention | **enforced in the DB for every `nava_app` query; the masker fails closed** |
| Stops other Go code? | no. A guarded function name or a "capability argument" is a string that any package can pass | no. `SET ROLE` is one statement any package can send | yes | partly | a different credential and pool, held by one package (AR-TICK-549-01) |
| Masker logic | Go | Go | **duplicated** in PL/pgSQL | Go | **Go, unchanged** |
| Operational cost | 1 owner role, 1 function, a `search_path` pin | 1 role, 1 membership | functions plus a second masker | none | 1 role, 1 password, 1 PgBouncer pool capped at 5, 1 `pgxpool` |

**Verdicts**
- **(a) rejected.** It breaks I3 as written. The stronger reason: EXECUTE is granted to `nava_app`, and every package runs as `nava_app`, so (a) is the TICK-541 problem moved into SQL.
- **(b) rejected as written.** It breaks I3, and any package can send `SET ROLE`. (e) is (b) without the membership.
- **(c) rejected.** *(Wording corrected in rev 1.)* A non-invoker view does not bypass RLS, because FORCE RLS binds its owner too, but it does bypass column grants, and **CI-1 already bans views without `security_invoker`**. A `security_invoker` view **needs the very grant we are removing**, so it protects nothing. A definer function breaks I3. Second reason: it forks R7 into PL/pgSQL, which will drift from `patchwalk` once S10 adds per-field masks.
- **(d) rejected as the end state.** The bypasses are demonstrated, and CI cannot prove a denial. It is the interim state until (e) ships, and its guard stays afterwards as layer 2.
- **(e) recommended.**

## 4. Design (e)

### 4.1 Manifest and policies

`db/grants.yaml`:

```yaml
  # ADR-004-A1: the R7 reader. The only runtime role that can read audit_log.diff
  # and diff_salt. LOGIN, a member of nothing, owns nothing, through PgBouncer.
  # Reads only; never records (RFC-012 rev 1).
  nava_audit_reader:
    attributes: [LOGIN]
    pgbouncer_login: true
    database: [CONNECT]
    schemas: { public: [USAGE] }
    tables:
      audit_log:
        SELECT: [tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id,
                 version_before, version_after, actor_principal_id, actor_system, source, source_ref,
                 change_set_id, causation_id, causation_depth, request_id, backfilled, content_sha256,
                 diff_sha256, diff_truncated, diff, diff_salt, seq, sealed_at, erased_at]
      entity_definition_versions: { SELECT: [definition_id, schema] }
```

- `nava_app.tables.audit_log.SELECT` loses `diff` and `diff_salt`. Everything else stays. The generated R12 commit-check SELECT (`internal/db/audited/manifest.go:63`) cannot bring them back (TA, confirmed).
- `txid` is left out of the reader's grant, and `read.go`'s `columns` constant does not select it. That must stay consistent (Postgres N2).
- `nava-db migrate` revokes any grant the manifest does not list, so the column revoke needs no hand-written SQL.

**The A1 policy migration (B1).** Add the role to the existing canonical SELECT policies. This keeps one policy per command per table, with the predicate unchanged:

```sql
-- +goose Up
ALTER POLICY audit_log_select ON audit_log
    TO nava_app, nava_worker, nava_erasure, nava_audit_reader;
ALTER POLICY entity_definition_versions_select ON entity_definition_versions
    TO nava_app, nava_worker, nava_audit_reader;
```

- Both policies keep the canonical `tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid` USING clause (I1). That clause references `tenant_id`, which needs no column grant (TA, confirmed). The Postgres spike shows `ALTER POLICY … TO` turns the reader's 0 rows into 1.
- The masker's query `WHERE definition_id = $1`, under the policy's `tenant_id = …`, uses `entity_definition_versions_pkey (tenant_id, definition_id, version)`. The history query uses `audit_log_subject` (Postgres, confirmed). The implementing story captures `EXPLAIN` for both with the reader policy in place.
- Partitions: the policy on the partitioned parent governs reads through the parent. No role is granted anything on a partition directly. CI-A3 now proves that (§5).

### 4.2 Code

- `internal/audit/read`:
  - `OpenPool(ctx, dsn) (*Pool, error)`. The type is opaque, with no exported field or method that returns the `*pgxpool.Pool`. It sets `MaxConns = 5`.
  - `List` and `RecordEntries` take the `*Pool` they were built with, not a caller `pgx.Tx`. Each opens `tenancy.InTenantTx(ctx, pool, tenantID, …)` and runs the read and the masker in that one transaction.
  - **The masker fails closed (B1).** `DefinitionMasker.MaskedPaths` returns `ErrNoDefinitionVersions` when `EntityDefinitionID` is set and the query returns 0 rows. A published definition always has at least one version, so 0 rows means a missing policy, a missing grant or a broken invariant. It never means "no secrets". `mask()` propagates the error, the request fails as a 500 with a log line naming the definition id (never a value), and no diff or salt is returned. An entry with no definition still has no masked path, as today.
- `cmd/nava` config loader: it is the only code that reads the reader DSN's env var (`NAVA_AUDIT_READER_DATABASE_URL`) and its config key. It passes the DSN only to `read.OpenPool`.
- `internal/server/audit.go`: authorisation (`RequireAuditPermission`), cursor decoding and the export lease stay on `nava_app` in their own transaction. Then the reader is called.
- `internal/server/records_patch.go` (record history): two transactions. First, `nava_app` reads the record and returns the same 404 for missing, unreadable or other-tenant records, exactly as now. Only then does the reader transaction read the entries. Under READ COMMITTED each statement already takes its own snapshot, so this adds no race (PG 16 docs §13.2.1, confirmed by the TA). **No `records` SELECT for the reader (settled):** it would put every record payload behind the credential that reads raw diffs.
- Export (R19–R21, not built yet) reads pages through the reader pool. Its lease and the `audit/export` entry stay on `nava_app`.

### 4.3 Pooling (A2, settled)

- **PgBouncer side:** add `nava_audit_reader = pool_size=5` under a `[users]` section in `deploy/compose/pgbouncer/pgbouncer.ini` and `scripts/ci/pgbouncer/isolation.ini`. Spiked on 1.24.1: extra clients queue and do not fail.
- **Client side:** `pgxpool` `MaxConns = 5` in `read.OpenPool`.
- **No `CONNECTION LIMIT`.** It is a role attribute that `grants.AllowedSettings` cannot express, and it caps by failing logins instead of queueing them.
- Transaction pooling is safe: the only session state is the transaction-local `app.tenant_id`. `get_auth` serves the new role with no PgBouncer change or restart (TA spike, A3 confirmed).

## 5. Tests (red, then green)

**New and extended cases.** Every denial asserts **SQLSTATE `42501`**, never the message text: Postgres says "permission denied for table", not "column" (TA).

| Id | Case | Today | After (e) |
|---|---|---|---|
| CI-A3 (extended) | The catalogue gate: no runtime role except `nava_audit_reader` holds SELECT on `diff` or `diff_salt`. It uses `has_column_privilege` over every manifest role, on `audit_log` **and every child in `pg_inherits`** (today `audit_log_default`; Epic 30 adds range partitions). No runtime role holds any privilege on a partition directly. | **red** (`nava_app`) | green |
| registry `ADR-004-A1/diff-denied` (editing `registry.go` needs CTO sign-off) | As `nava_app` through PgBouncer, tenant A set. Each must fail with `42501`: `SELECT diff`, `SELECT diff_salt`, `SELECT *`, `to_jsonb(a)`, `a::text`, the WHERE oracle `SELECT id FROM audit_log WHERE diff::text LIKE '%x%'`, `ORDER BY diff_salt`, **`FROM audit_log_default`**, `FROM ONLY audit_log`, `COPY (SELECT diff …) TO STDOUT`, `COPY audit_log (diff) TO STDOUT`, a CTE `SELECT *`, `LATERAL json_array_elements(a.diff)`, `UPDATE … RETURNING diff`, and `INSERT … RETURNING diff_salt`. The same probes run as `nava_worker`, `nava_relay`, `nava_purge`, `nava_onboard` and `nava_groupsync`, and as `nava_erasure` directly. The reader writing must also fail: INSERT, UPDATE and DELETE on `audit_log`, and any SELECT on `records`, as `nava_audit_reader` (Security N1). | **red** for `nava_app` | green |
| registry `ADR-004-A1/reader-scoped` | As `nava_audit_reader`: tenant A set reads its own diff; filtering on tenant A's id with tenant B set reads 0 rows; with no tenant set, 0 rows. SELECT on `txid` fails with `42501`. | n/a | green |
| registry `ADR-004-A1/reader-masks` (**B1**) | Seed a definition whose schema has an `x-nava-secret` property, and a record change that sets it. Read `GET /api/v1/audit/entries` through the reader pool as a principal with `audit.read` and no `mask` permission. Assert the op is `{"redacted":"mask"}`, `diff_redacted = true` and `diff_salt` is absent. | n/a (red if the §4.1 policy change is left out) | green |
| unit `read` masker (**B1**) | `DefinitionMasker` given 0 version rows for a non-nil `EntityDefinitionID` returns `ErrNoDefinitionVersions`. `mask()` then returns an error and no entry. A nil definition still returns no paths. | **red** (returns nil today) | green |
| CI-A14 | `diff_salt` only with an unredacted diff (R7). | green | green |

**Existing tests that change (B2).** The Postgres specialist ran the full integration suite with the revoke applied: 1103 passed, 4 skipped, 6 failed. Every failure is in the table below, plus one unit test. No `nava_app` query outside the reader failed.

| Test | Package | Change |
|---|---|---|
| `TestListAuditEntriesPagesFiltersAndGate` | `internal/server` | **moves** to the reader pool (the handler is built with a `read.Pool`). Assertions unchanged. |
| `TestListAuditEntriesTokenBindingAndTenant` | `internal/server` | **moves**, as above |
| `TestListAuditEntriesMasksSecretProperties` | `internal/server` | **moves**, as above |
| `TestRecordHistoryMasksSecretProperties` | `internal/server` | **moves**: the record check runs on `nava_app` and the entries on the reader pool. Same-404 cases unchanged. |
| `TestPatchHistoryPagesNewestFirst` | `internal/server` | **moves**, as above |
| `TestP510bAppReaderGrantAndTenantBoundary` | `internal/db/dbtest` (`audit_log_adversarial_test.go:151`) | **inverted**: `nava_app` `SELECT diff, diff_salt` must fail with `42501`; the tenant-boundary half moves to `nava_audit_reader` |
| `TestLoadRealManifest` | `internal/db/audited` (`manifest_test.go:50`) | **inverted**: `nava_app` must not hold `diff` or `diff_salt`; only `nava_audit_reader` holds them |

The masker's own unit tests (`internal/audit/read/mask_test.go`) stay green and gain the fail-closed case above.

## 6. The TICK-541 guard: kept, as a second layer

**Decision: keep it, do not remove it.** Postgres becomes the enforcement layer, and the guard becomes the early, local signal.

- **Rule 1 (new, primary):** it checks Go identifiers and string constants, not SQL text, so the `strings.Builder` and package-`var` bypasses do not apply. It has two parts:
  - only `internal/audit/read` may call `read.OpenPool`;
  - **only the `cmd/nava` config loader may read the reader DSN's source.** That means a string constant equal to the env var name `NAVA_AUDIT_READER_DATABASE_URL` or the config key, and any reference to the config field that holds the DSN (Security condition).
  It covers the hole Postgres cannot see: Go code in nava-core that opens the reader pool on purpose. A name built at runtime from fragments is still invisible to it; that is the deliberate act AR-TICK-549-01 accepts.
- **Rule 2 (existing SQL-text rule, kept):** it still reports diff reads outside the reader. After (e), a miss is no longer a leak: Postgres rejects the query with `42501` at Execute, every time, and the PgBouncer connection survives (TA, confirmed). It stays because it costs nothing and fails sooner, in `task test` rather than in integration.

## 7. What we accept

- **AR-TICK-549-01 · in-process misuse (Low; accepted by Security in round 1).** The reader DSN is on the nava-core host, because nava-core serves the read endpoints. Code inside nava-core that deliberately opens a second pool with that DSN reads raw diffs. Guard rule 1 and review cover it. The DB cannot, because process memory is one trust domain, and that code would already hold the `nava_app` credential. Cross-tenant reads stay closed either way (FORCE RLS, tenant per transaction from the validated principal). It is recorded in `docs/security/accepted-risks.md`. *Triggers to revisit:* a second service needs audit reads, or tenant code (I8) ever runs in-process.
- **AR-A1-2 · one more pool, capped per PgBouncer instance.** At most 5 extra server connections **per PgBouncer instance** (`pool_size=5`), and at most 5 client connections per nava-core process (`MaxConns`). With N PgBouncer instances the server-side ceiling is 5 × N. *Trigger:* we run more than one PgBouncer instance against the same PostgreSQL.

## 8. Invariants touched

| Invariant | Effect |
|---|---|
| I1 tenant wall | unchanged. The new role is under FORCE RLS and is added to the existing canonical policies, with their predicates unchanged. |
| I3 least privilege | **strengthened**. One new manifest role with no membership, no definer function, and nothing on partitions or `records`. `nava_app` loses two column grants. |
| I5 masks on read | **strengthened**. The raw diff is reachable only through the package that applies R7, and that package fails closed. |
| I4, I6, I7, I8, I9, I2 | not touched |

ADR-004 changes:
- R7 gains "the raw diff is readable only by `nava_audit_reader`, held by `internal/audit/read`; the masker fails closed".
- R12's `nava_app` reader list loses `diff` and `diff_salt`.
- R14's role list gains the new role, with no write on `audit_chain_head`.

## 9. Sequencing

TICK-419 (the story, base of PR #443) is not yet merged to `main`. **The cheapest path is to land (e) inside that story before it merges**, so `main` never carries the `nava_app` grant. The order:

1. **One change, in this order:**
   - the `nava_audit_reader` manifest role;
   - **its password, `DBROLES_PASSWORD_NAVA_AUDIT_READER`, in every environment in the same change:** `compose.yaml`, the CI isolation stack and `.env.example`. db-roles fails closed for *every* role when a LOGIN role has no password (`roles.go:142`), so a missing password stops the whole converge.
   - the `[users] pool_size=5` lines;
   - the A1 policy migration (§4.1);
   - the red tests: CI-A3 with partitions, `diff-denied`, `reader-scoped`, `reader-masks`, the masker unit case, and the two inverted tests.
2. `read.OpenPool` with `MaxConns = 5`, the fail-closed masker, the DSN in the `cmd/nava` loader, then the handler changes (§4.2). The five moved tests and the new cases turn green.
3. Guard rule 1, both parts, and the PR #443 follow-up.

The Technical Lead decides whether this goes into TICK-419 or a follow-up story. If it ships after TICK-419 merges, (d) is the interim state, and AR-A1-0 ("`nava_app` can read `diff` until A1 ships") is recorded with that story as its trigger.

## 10. Assumptions

| # | Assumption | Status |
|---|---|---|
| A1 | No `nava_app` query outside `internal/audit/read` reads `diff` or `diff_salt`. | **Confirmed** by the Postgres full integration run; the 7 affected tests are listed in §5. |
| A2 | Mechanism for the connection cap | **Settled**: PgBouncer `[users] pool_size=5` plus `MaxConns = 5` (§4.3). |
| A3 | `pgbouncer.get_auth` serves a new `pgbouncer_login: true` role with no changes. | **Confirmed** by the TA spike, including a login through a running PgBouncer. |
| A4 | INSERT without `RETURNING` needs no SELECT on the inserted columns. | **Confirmed** by the TA spike. |
| A5 (new) | Every published entity definition has at least one `entity_definition_versions` row, so the fail-closed rule never rejects a valid entry. | Inferred from D-3-4 (a version is insert-only, and publishing creates one). The implementing story asserts it with a catalogue query in the `reader-masks` fixture. |

## 11. Questions for critique round 1 (answered)

- Security, AR-A1-1: accepted as AR-TICK-549-01 (Low), with no separate process for Milestone 1, on the condition about the DSN source (§6).
- Postgres, connection cap: PgBouncer `pool_size` (§4.3).
- Postgres, `records` SELECT: no.

## 12. Response — round 1

| Item | From | Label | Resolution |
|---|---|---|---|
| **B1** the masker fails open under the reader role | Security (confirmed by the Postgres spike) | blocking | **Accepted.** §4.1 adds `nava_audit_reader` to `entity_definition_versions_select` (and `audit_log_select`) by `ALTER POLICY … TO`, with the predicate unchanged. §4.2 makes `MaskedPaths` return `ErrNoDefinitionVersions` on 0 rows for a non-nil definition, and `mask()` propagates it. §5 adds the registry case `ADR-004-A1/reader-masks` and the unit case. A5 records the assumption the fail-closed rule rests on. |
| **B2** §5 is missing 7 tests | Postgres | blocking | **Accepted.** §5 lists all 7 by name and package: 5 move to the reader pool, and `TestP510bAppReaderGrantAndTenantBoundary` and `TestLoadRealManifest` are inverted. The inverted pair lands red-first in §9 step 1. |
| N1 more `diff-denied` probes | Security | non-blocking | **Accepted** (§5): the WHERE oracle, `INSERT … RETURNING diff_salt` as `nava_app` and `nava_worker`, and the reader's INSERT, UPDATE and DELETE on `audit_log` and SELECT on `records`. |
| AR-A1-1 ruling, plus the DSN-source condition | Security | ruling | **Folded in**: AR-TICK-549-01 (Low) in §7; guard rule 1's second part in §6. |
| (c) wording | Security | correction | **Folded in** (§3). |
| N2 keep `txid` out of the reader's grant and `columns` | Postgres | non-blocking | **Accepted** (§4.1). |
| A2 cap mechanism | Postgres (CTO settled) | answer | **Folded in** (§4.3); AR-A1-2 is reworded as per instance (§7). |
| `records` SELECT | Postgres (CTO settled) | answer | **Folded in**: no (§4.2). |
| TA 1 deploy order: the password ships with the manifest | TA | non-blocking | **Accepted** (§9 step 1). |
| TA 2 CI-A3 walks `pg_inherits`, and an `audit_log_default` probe | TA | non-blocking | **Accepted** (§5). |
| TA 3 assert SQLSTATE `42501`, not the message | TA | non-blocking | **Accepted** (§5 header). |
| TA 4 `MaxConns` as the client-side cap | TA | non-blocking | **Accepted**, alongside the PgBouncer cap (§4.3). |
| TICK-550 (PgBouncer `08P01` after a Parse-time error) | TA side finding | out of scope | No change here. `42501` happens at Execute and is unaffected. |

Re-check scope (per the CTO): Security re-checks B1 only; Postgres re-checks B2 only.
