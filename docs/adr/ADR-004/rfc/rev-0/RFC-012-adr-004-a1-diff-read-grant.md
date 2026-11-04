---
type: rfc
id: RFC-012
title: "ADR-004-A1: only the masking reader's role can read audit_log.diff and diff_salt"
status: DRAFT (frame, round 0)
date: 2026-11-04
issue: TICK-549
amends: ADR-004 (R7, R12); touches ADR-001 I3
author: Solution Architect
---

# RFC-012 · ADR-004-A1: Postgres enforces who reads the raw diff

> **Invariants source.** The project description defines no invariants (its description is empty). This RFC is checked against ADR-001 §1, I1–I9, the same source ADR-004 used. If the project is meant to carry its own list, the invariant checks below are incomplete until that list is published.

## 1. Recommendation

Use **option (e)**, a variant of (b) that the squad adds here: **a dedicated LOGIN role, `nava_audit_reader`, with its own PgBouncer login and its own `pgxpool` in nava-core.** The role is not a member of anything, and `nava_app` is not a member of it.

- `nava_audit_reader` gets SELECT on the reader's `audit_log` columns, `diff` and `diff_salt` included. It also gets SELECT `(definition_id, schema)` on `entity_definition_versions`, which the R7 masker needs. It gets nothing else: no INSERT, no UPDATE, nothing on `records`.
- `nava_app` **loses** SELECT on `audit_log.diff` and `audit_log.diff_salt`. It keeps SELECT on the other reader columns, so the audit handlers can still read envelopes.
- `internal/audit/read` is the only package that holds the reader pool. Its constructor `read.OpenPool(ctx, dsn)` returns an opaque type, and `cmd/nava` passes the DSN only to that constructor. Every diff read runs in its own `tenancy.InTenantTx` on that pool, so `set_config('app.tenant_id', …, true)` scopes it exactly as it does today.
- **The TICK-541 static guard stays, as a second layer.** It is not removed. Its main rule is retargeted to the new capability, and its SQL-text rule stays as an early warning (§6).

Why not (a) or (b) as written? **Both break ADR-001 I3.** (a) makes a `SECURITY DEFINER` function reachable from a runtime role, which I3 forbids outright ("No `SECURITY DEFINER` function or RLS-bypassing view is reachable from a runtime role"). (b) needs `nava_app` to be a member of `nava_audit_reader`, because `SET [LOCAL] ROLE` only works for members, and I3 says no runtime role is a member of any role. `nava-db roles` actively revokes such memberships. (e) gets what (b) was after (the diff is readable only through a role that the masker owns) without the membership, and without a role switch that could outlive its transaction.

## 2. Context

- ADR-004 R7 masks secret pointers in the diff at read time. The masker runs only in `internal/audit/read` (`mask.go`, `definition_masker.go`).
- On the TICK-419 story branch (`db/grants.yaml`, the `nava_app.tables.audit_log.SELECT` list), `nava_app` holds SELECT on every reader column, `diff` and `diff_salt` included. RLS limits rows, not columns. So any `nava_app` query outside the reader package gets the raw diff, including a query on behalf of a same-tenant user who holds no `mask` permission.
- The other runtime roles are already narrow (verified on `origin/TICK-541/audit-diff-reader-guard`):
  - `nava_worker`'s SELECT excludes `diff` and `diff_salt`, because the sealer hashes `diff_sha256`, not the diff.
  - `nava_erasure`'s SELECT excludes both (R23). Its erase UPDATE only *sets* them to NULL, and the R11 trigger reads `OLD` without needing a column grant.
  - **`nava_app` is the only runtime role with the gap.**
- TICK-541 (PR #443) adds `TestAuditDiffReadOnlyInReader`, a source-text check. The Security review showed it cannot see a query built with `strings.Builder`, a column list held in a package `var`, or SQL that is not Go text.
- Nothing outside the reader reads the diff today, so this is a latent risk, not a live exploit.

### Who touches the diff columns

| Caller | Role | Needs | After (e) |
|---|---|---|---|
| `audit.Append` (records pipeline, config writers) | `nava_app`, `nava_worker` | INSERT `diff`, `diff_salt` | unchanged. `append.go` has no `RETURNING` or `ON CONFLICT`, so INSERT needs no SELECT. |
| S8 backfill (migrations B1/B2) | `nava_migrator` (owner, not runtime) | INSERT | unchanged |
| Sealer A2 | `nava_worker` | `diff_sha256` only | unchanged (already excludes `diff`) |
| Erase step (R23) | `nava_erasure` | UPDATE to NULL | unchanged |
| `GET /api/v1/audit/entries`, record history, export (R19–R21) | today `nava_app`; after (e) `nava_audit_reader` | SELECT `diff`, `diff_salt`, masked | **moves to the reader pool** |

## 3. Options against the constraints

The constraint rows come from the issue body; the I-rows are ADR-001.

| | (a) definer function | (b) `SET LOCAL ROLE` | (c) mask in SQL | (d) static guard only | **(e) reader login + pool** |
|---|---|---|---|---|---|
| Tenant setting stays per transaction | yes | yes, but the role switch rides the pooled connection (`SET LOCAL` resets at COMMIT or ROLLBACK; holds only if every path ends the transaction) | yes | yes | **yes; there is no role switch at all** |
| No RLS bypass; definer owner is not the table owner and has no BYPASSRLS | possible, but needs a new NOLOGIN owner role holding SELECT on `diff` | yes | needs a definer function or a non-invoker view (see below) | yes | **yes: not the owner, NOBYPASSRLS, FORCE RLS applies** |
| Writers keep INSERT | yes | yes | yes | yes | **yes** |
| CI-A3 and registry can prove `nava_app` is denied | yes | yes | yes | **no** | **yes** |
| **I3** least privilege | **violates**: a definer function reachable from `nava_app` | **violates**: `nava_app` must be a member | **violates** unless the view is `security_invoker`, and then it protects nothing | holds | **holds**: a manifest role, LOGIN, member of nothing, owns nothing |
| I5 masks on read | moved, not enforced: any `nava_app` code can call the function | any `nava_app` code can run `SET ROLE` | enforced in the DB | convention | **enforced in the DB for every `nava_app` query** |
| Stops other Go code? | no. A guarded function name or a "capability argument" is a string that any package can pass, so it is the TICK-541 problem moved into SQL | no. `SET ROLE` is one statement any package can send | yes | partly | a different credential and pool, held by one package (residual in §7) |
| Masker logic | Go | Go | **duplicated** in PL/pgSQL: R5's walk over `path`, `from` and nested `value`, plus S10's per-field ACL inputs | Go | **Go, unchanged** |
| Operational cost | 1 owner role, 1 function, its `search_path` pin | 1 role, 1 membership | functions plus a second masker to keep in step | none | 1 role, 1 password, 1 PgBouncer pool, 1 `pgxpool` |

**Verdicts**
- **(a) rejected.** It breaks I3 as written. Even with I3 amended, it does not answer the issue's own open question: EXECUTE is granted to `nava_app`, and every package runs as `nava_app`.
- **(b) rejected as written** for the same reasons: it breaks I3, and any package can send `SET ROLE`. (e) is (b) without the membership.
- **(c) rejected.** It needs definer semantics (which breaks I3) or protects nothing. It also forks R7 into a second implementation, which will drift from `patchwalk` once S10 adds per-field masks.
- **(d) rejected as the end state.** The bypasses are already demonstrated, and CI cannot prove a denial. It stays as the interim state until (e) ships, and its guard stays afterwards as layer 2.
- **(e) recommended.**

## 4. Design (e)

### 4.1 Manifest (`db/grants.yaml`)

```yaml
  # ADR-004-A1: the R7 reader. The only runtime role that can read audit_log.diff
  # and diff_salt. LOGIN, a member of nothing, owns nothing, through PgBouncer.
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

- `nava_app.tables.audit_log.SELECT` loses `diff` and `diff_salt`. Everything else stays (the R12 generated commit-check columns are a subset).
- `txid` is left out of the reader role on purpose: it is never exported (R1).
- `nava-db migrate` already revokes any grant the manifest does not list, so the revoke needs no hand-written SQL. The audit_log SELECT policy for the new role is canonical (I1) and ships in a migration, because policies stay in SQL.
- `DBROLES_PASSWORD_NAVA_AUDIT_READER` goes on the db-roles service, and the reader's DSN goes in nava-core's config.

### 4.2 Code

- `internal/audit/read`:
  - `OpenPool(ctx, dsn) (*Pool, error)`. The type is opaque, with no exported field or method that returns the `*pgxpool.Pool`.
  - `List` and `RecordEntries` take the `*Pool` they were built with, not a caller `pgx.Tx`. Each opens `tenancy.InTenantTx(ctx, pool, tenantID, …)` and runs the read and the masker in that one transaction.
- `internal/server/audit.go`: authorisation (`RequireAuditPermission`), cursor decoding and the export lease stay on `nava_app` in their own transaction. Then the reader is called.
- `internal/server/records_patch.go` (record history): today one `nava_app` transaction reads the record and then its entries. After (e) these are **two transactions**. First, `nava_app` reads the record and returns the same 404 for missing, unreadable or other-tenant records, exactly as now. Only then does the reader transaction read the entries. Both run under READ COMMITTED, where each statement already takes its own snapshot, so splitting them adds no race the single transaction did not have. (This is how PostgreSQL READ COMMITTED behaves; I am confident of it.)
- Export (R19–R21, not built yet) reads pages through the reader pool. Its lease and the `audit/export` entry stay on `nava_app`.

### 4.3 Pooling

- PgBouncer pools per (user, database), so the new role gets its own pool, up to `default_pool_size = 20` server connections (`deploy/compose/pgbouncer/pgbouncer.ini`). Audit reads are low-volume. I recommend capping this pool at 5 on PostgreSQL with `ALTER ROLE nava_audit_reader CONNECTION LIMIT` or the pooler equivalent. Which mechanism the manifest can express is for the Postgres specialist (unverified, A2).
- Transaction pooling is safe here: there is no session state other than the transaction-local `app.tenant_id`.

## 5. Tests (red, then green)

| Id | Case | Today | After (e) |
|---|---|---|---|
| CI-A3 (extended) | The catalogue gate: no runtime role except `nava_audit_reader` holds SELECT on `audit_log.diff` or `diff_salt` (`has_column_privilege` over every manifest role). | **red** (`nava_app`) | green |
| registry `ADR-004-A1/diff-denied` (adversarial, `registry.go`; editing it needs CTO sign-off) | As `nava_app` through PgBouncer, with tenant A set: `SELECT diff FROM audit_log`, `SELECT diff_salt …`, `SELECT * …`, `SELECT to_jsonb(a) FROM audit_log a`, `COPY (SELECT diff FROM audit_log) TO STDOUT`. Each must fail with `42501`. Repeat as `nava_worker`, `nava_erasure`, `nava_relay`, `nava_purge`, `nava_onboard`, `nava_groupsync`. | **red** for `nava_app` | green |
| registry `ADR-004-A1/reader-scoped` | As `nava_audit_reader`, tenant A set, tenant B's diff reads 0 rows. With no tenant set, 0 rows or an error. (The generic isolation checks already cover this once the role is in the manifest; the named case documents intent.) | n/a | green |
| unit | `read` tests run against the reader pool; the history and entries handler tests keep the same-404 cases. | green | green |
| CI-A14 | `diff_salt` only with an unredacted diff (R7). | green | green |

## 6. The TICK-541 guard: kept, as a second layer

**Decision: keep it, do not remove it.** Postgres becomes the enforcement layer, and the guard becomes the early, local signal.

- **Rule 1 (new, primary):** only `internal/audit/read` may call `read.OpenPool` or name the reader DSN's config field, and only `cmd/nava` wiring may pass that field. The guard checks Go identifiers, not SQL text, so the `strings.Builder` and package-`var` bypasses do not apply. This rule covers the one hole Postgres cannot see: Go code in nava-core that opens the reader pool on purpose (§7).
- **Rule 2 (existing SQL-text rule, kept):** it still reports diff reads outside the reader. After (e), a miss here is no longer a leak: Postgres rejects the query with `42501` in the integration run. It stays because it costs nothing and fails sooner, in `task test` rather than in integration.

## 7. What we accept (AR row proposed for ADR-004-A1)

- **AR-A1-1 · in-process misuse.** The reader DSN is on the nava-core host, because nava-core serves the read endpoints. Code inside nava-core that deliberately opens a second pool with that DSN reads raw diffs. Guard rule 1 and review cover it. The DB cannot, because process memory is one trust domain. This is a deliberate act, not the accidental convention break the issue is about. *Trigger to revisit:* a second service needs audit reads, or tenant code (I8) ever runs in-process.
- **AR-A1-2 · one more pool.** At most 5 extra server connections if the cap in §4.3 lands, otherwise up to 20.

## 8. Invariants touched

| Invariant | Effect |
|---|---|
| I1 tenant wall | unchanged; the new role is under FORCE RLS with a canonical policy |
| I3 least privilege | **strengthened**. One new manifest role, no membership, no definer function. `nava_app` loses two column grants. |
| I5 masks on read | **strengthened**. The raw diff is reachable only through the package that applies R7. |
| I4, I6, I7, I8, I9, I2 | not touched |

ADR-004 changes: R7 gains "the raw diff is readable only by `nava_audit_reader`, held by `internal/audit/read`". R12's `nava_app` reader list loses `diff` and `diff_salt`. The new role is added to R14's role list as "no write on `audit_chain_head`".

## 9. Sequencing

TICK-419 (the story, base of PR #443) is not yet merged to `main`. **The cheapest path is to land (e) inside that story before it merges**, so `main` never carries the `nava_app` grant. The order:

1. Manifest role, policy migration and password wiring. Red tests land with it: the CI-A3 extension and the registry case.
2. `read.OpenPool`, then the reader and handler changes (§4.2). The tests turn green.
3. Guard rule 1 and the PR #443 follow-up.

The Technical Lead decides whether this goes into TICK-419 or a follow-up story. If it ships after TICK-419 merges, (d) is the interim state, and AR-A1-0 ("`nava_app` can read `diff` until A1 ships") is recorded with that story as its trigger.

## 10. Assumptions not verified

| # | Assumption | How to settle it |
|---|---|---|
| A1 | No `nava_app` query outside `internal/audit/read` reads `diff` or `diff_salt` today, so the revoke breaks nothing. | The TICK-541 guard says so for Go text. Settle it by running the full integration suite with the revoke applied (Postgres specialist). |
| A2 | The manifest can express a connection cap for one role (`ALTER ROLE … CONNECTION LIMIT` is not in `grants.AllowedSettings` today), or PgBouncer can cap one user's pool. | Postgres specialist: check against the docs for our PgBouncer version. |
| A3 | `pgbouncer.get_auth` serves a new `pgbouncer_login: true` role without code changes. | `deploy/compose/test/cases/pgbouncer.sh` with the new role. |
| A4 | `INSERT` without `RETURNING` needs no SELECT privilege on the inserted columns. This is standard PostgreSQL privilege behavior, and `append.go` was checked for `RETURNING` and `ON CONFLICT`. | Covered by the existing append integration tests after the revoke. |

## 11. Questions for critique round 1

- **Security:** is AR-A1-1 acceptable, or do you want the read endpoints in a separate process so the DSN leaves the nava-core host? That is a new service, which needs CTO escalation.
- **Postgres:** should `nava_audit_reader` get SELECT on `records` so record history stays one transaction? I propose not (smaller grant, same READ COMMITTED semantics).
- **Postgres:** the connection-cap mechanism (A2).
