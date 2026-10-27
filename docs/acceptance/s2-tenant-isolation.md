# Acceptance kit: S2, a query without the right tenant returns nothing

Story: TICK-44. Plan: PLAN-001 (`docs/delivery/plan-platform-foundation.md`). Design: ADR-001, Annex A.3 (D28) and Annex B (CI-1 to CI-4, CI-9).
Plain-language explainer for readers new to the architecture: [`s2-tenant-isolation-explained.html`](s2-tenant-isolation-explained.html) (open it in a browser).
Verified on 2026-10-26 by the Quality, Testing & CI Advisor: first on `main` at `9f8c39a` (fresh stack, all criteria), then again on `main` at `d32a18c`, after the criterion 3 fix (`make isolation`, and the hand-grant probe in CI).

**Status: READY for the human.** All 4 criteria pass. The first run failed criterion 3; TICK-233 (PR #139) and TICK-235 (PR #145) fixed it.

## What you can now do

- The database itself keeps each tenant's rows away from every other tenant. A query that names no tenant, an empty tenant, or a tenant left over from an earlier transaction gets nothing back.
- Every part of the platform (API, worker, relay, purge, onboarding, group sync, backup) logs in as its own database user. Each can do only its own job.
- One command, `make isolation`, proves this and writes a report with one line per user and table.
- A pull request that adds a table without the tenant wall, or grants a database privilege outside the manifest, fails CI.

## Try it (about 10 minutes, most of it waiting)

Needs Docker and Go 1.26, from a fresh clone of `main`.

1. Run the isolation suite. It starts its own throwaway PostgreSQL and PgBouncer and seeds tenants A and B. It takes about 3.5 minutes.
   ```sh
   make isolation
   ```
   It should end with `ok ... internal/tenancy/isolation`. Open `isolation-report.md` at the repository root: the first line reads `Result: **PASS**`.
2. In the report, read the table **Every runtime role × every tenant table**. Every line is `PASS` (0 rows, or an error such as "permission denied" or "cannot connect"), except three `ALLOWLISTED` lines. Those are intentional exceptions, explained under criterion 1 below.
3. Optional: start the full stack and look at the database users yourself.
   ```sh
   make dev-ca
   export COMPOSE_PROJECT_NAME=nava-mine NAVA_CORE_PORT=0 KEYCLOAK_PORT=0 TEMPORAL_TEST_PORT=0
   docker compose build && docker compose up --wait
   docker compose exec -T postgres sh -c 'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava' <<'SQL'
   SELECT rolname, rolsuper, rolbypassrls FROM pg_roles WHERE rolname ~ '^nava_' ORDER BY 1;
   SQL
   ```
   Every `nava_*` user shows `f | f`. Clean up with `docker compose down -v`.
4. Optional: look at the checks on the two probe PRs (closed, never merged). Both have red `integration` and `stack*` jobs:
   - PR #132 adds a tenant table without `FORCE ROW LEVEL SECURITY`.
   - PR #143 adds `GRANT SELECT ON records TO nava_relay` in a migration.

No sign-in or UI is involved. This story is entirely in the database.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | As each runtime role, a query with no tenant, an empty tenant, or a tenant left over on a pooled connection returns 0 rows or an error | ✅, with 3 intentional exceptions (allow-listed policies, see below) | `make isolation` report on `d32a18c` and `9f8c39a`; fresh-stack probe |
| 2 | No runtime role is a superuser, has `BYPASSRLS`, owns an object or is a member of any role | ✅ | Fresh-stack catalogue queries; CI tests `TestD28*` and `TestDBRoles*`, and the CI-3 gate. Not a section of the `make isolation` report: see Known limits |
| 3 | Granted privileges equal `db/grants.yaml` exactly; adding a grant by hand fails CI | ✅ | Probe PR #143 is red, CI run [40317464391](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40317464391) |
| 4 | `nava_relay` cannot read tenant data tables; `nava_backup` cannot open a SQL session | ✅ | `make isolation` CI-9 section; fresh-stack probe |
| extra | A tenant table without `FORCE ROW LEVEL SECURITY` fails CI (story "How to try it" step 3) | ✅ | Throwaway PR #132 is red |

### Criterion 1 evidence

`make isolation` passed with `Result: **PASS**` on `9f8c39a` (3 min 26 s) and again on `d32a18c`, with the same counts. The report has 121 lines for runtime role × tenant table: 11 roles, including the owner `nava_migrator`, × 11 tenant tables. None is FAIL. Every line was run with no tenant, `''` and a prior `SET LOCAL` of tenant A, directly and through PgBouncer, and then with tenant A aimed at tenant B's rows.

| Status | Lines | What it means |
|---|---|---|
| PASS, 0 rows on every probe | 21 | Roles with a SELECT, UPDATE or DELETE grant on the table |
| PASS, no grant | 53 | The role has no SELECT, UPDATE or DELETE on the table |
| PASS, cannot connect (42501) | 33 | `keycloak`, `temporal` and `nava_backup` have no `CONNECT` on `nava` |
| PASS, owner subject to `FORCE` | 11 | `nava_migrator`: 0 rows everywhere |
| ALLOWLISTED | 3 | Intentional exceptions, below |

**The 3 intentional exceptions.** ADR-001 I1 and Annex A.3 allow these policies by name. The report marks their lines `ALLOWLISTED` instead of hiding them, so they show rows with no tenant set:

| Role | Table | Policy | Why it does not test the tenant |
|---|---|---|---|
| `nava_app` | `tenant_identity_binding` | `binding_lookup` | Maps an identity provider's issuer and organisation to a tenant at sign-in, before any tenant is known |
| `nava_groupsync` | `tenant_identity_binding` | `binding_lookup` | The same lookup, for group sync |
| `nava_relay` | `outbox` | `outbox_relay_select`, `outbox_relay_update` | The relay delivers every tenant's events. It cannot read `dedupe_key` or `last_error` on `outbox`, or any other table (criterion 4) |

Any other policy that skips the tenant test fails CI: the allow-list has exactly 10 entries (CI-1).

The four fixtures in the report show that each check can fail: FORCE removed, RLS disabled, a fail-open policy, and a session-wide SET through PgBouncer each turn PASS into FAIL.

**Fresh-stack cross-check.** On a fresh compose stack, tenants A and B were seeded through PgBouncer (A as `nava_onboard`, then one or more records per tenant as `nava_app`). Results:

| Role, path | No tenant | `''` | Prior tx set A | Tenant A, B's rows | Tenant A, own rows |
|---|---|---|---|---|---|
| `nava_app`, direct | 0 | 0 | 0 | 0 | 3 |
| `nava_app`, PgBouncer | 0 | 0 | 0 | 0 | 3 |
| `nava_worker`, direct | 0 | 0 | 0 | 0 | 3 |
| `nava_worker`, PgBouncer | 0 | 0 | 0 | 0 | 3 |

As tenant B, `nava_app` sees only B in `tenants`, and `UPDATE records ... WHERE tenant_id = A` updates 0 rows. The own-row count is 3 because the seed ran three times while the probe script was being fixed.

### Criterion 2 evidence

Fresh compose stack, `docker compose down -v`, then `up --wait`: healthy in 188 s. Queried as the bootstrap superuser over the socket:

| Role | super | bypassrls | createrole | createdb | replication | login |
|---|---|---|---|---|---|---|
| `nava_app`, `nava_worker`, `nava_relay`, `nava_purge`, `nava_onboard`, `nava_groupsync`, `nava_migrator`, `pgbouncer_auth`, `keycloak`, `temporal` | f | f | f | f | f | t |
| `nava_backup` | f | f | f | f | **t** (by design, physical backup only) | t |
| `pgadmin` (bootstrap, in no service's environment) | t | t | t | t | t | t |

- `pg_auth_members` for non-`pg_` roles: **0 rows**. No role is a member of any role.
- Owners in database `nava`: all 39 relations belong to `nava_migrator`. The one function, `pgbouncer.get_auth`, and the schema `pgbouncer` belong to `pgadmin`. `nava` is owned by `nava_migrator`. Per ADR-001 A.3, `keycloak` and `temporal` own only their own databases. **No runtime role owns anything.**
- Every public table except goose's bookkeeping table has `relrowsecurity` and `relforcerowsecurity` set to true.
- The same facts are gated on every PR by `TestD28NoSuperuserOrBypassRLS`, `TestD28NoRoleMembershipsRecursive`, `TestDBRolesRolesHaveNoDangerousAttributes` and the CI-3 catalogue gate. All passed in CI run [40306371162](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40306371162).

### Criterion 3 evidence

**Granted privileges equal the manifest: ✅.** The migrator's catalogue gate (CI-1, CI-3, CI-4) runs after every migrator run. On the fresh stack, `migrate done` reported `migrations_applied: 5, grants_granted: 119, grants_revoked: 0`, with no gate problems. `TestT10GateCatchesCatalogueDrift` covers 20 drift cases, for example `GRANT DELETE ON records TO nava_app` added directly on the database.

**Adding a grant by hand fails CI: ✅.** Probe PR #143, on `main` after the fix, adds this migration:

```sql
-- +goose Up
GRANT SELECT ON records TO nava_relay;
```

CI run [40317464391](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40317464391) failed `integration`, `stack`, `stack-db` and `stack-internal` with:

```
CI-4: a migration granted SELECT table public.records TO nava_relay, which db/grants.yaml does not list (revoked on a fresh database; grant it in db/grants.yaml, not in a migration)
```

History: the first run on `9f8c39a` failed this half. Probe PR #133 had the same migration and stayed green, because the migrator revoked the grant and only logged a warning. TICK-233 (PR #139) makes any revoke on a fresh database a CI-4 problem. TICK-235 (PR #145) adds adversarial cases: column, `PUBLIC`, `WITH GRANT OPTION`, schema, database, function and default-privilege grants. On an already-deployed database the migrator still revokes and logs, and the run passes (AR-031).

### Criterion 4 evidence

From the `make isolation` report:
- `nava_relay`: PASS with "no SELECT, UPDATE or DELETE grant" on all 10 tenant data tables other than `outbox`. The CI-9 path "nava_relay cannot read records" is PASS, and the fixture "nava_relay granted a column of records" turns it to FAIL.
- `nava_backup`: "cannot connect to the platform database (SQLSTATE 42501)" on all 11 tables. The CI-9 path "nava_backup is rejected for SQL sessions and logical walsenders" is PASS, and the fixture "nava_backup granted CONNECT" turns it to FAIL.

Fresh-stack probe from a container on the stack's `internal` network:

| Attempt | Result |
|---|---|
| `nava_relay`: `SELECT` on `records`, `tenants`, `principals` | `permission denied for table ...` |
| `nava_relay`: `SELECT dedupe_key FROM outbox` | `permission denied for table outbox` (column not granted) |
| `nava_backup`: SQL session to `nava` | `pg_hba.conf rejects connection for host ..., user "nava_backup", database "nava"` |
| `nava_backup`: SQL session to `postgres` | `pg_hba.conf rejects connection ... database "postgres"` |
| `nava_backup`: logical walsender (`replication=database`) | `pg_hba.conf rejects connection ...` |
| `nava_backup`: physical replication from another host | `no pg_hba.conf entry for replication connection ...` (allowed only from the backup host) |

### Extra: missing `FORCE ROW LEVEL SECURITY` fails CI ✅

Throwaway PR #132 adds `acceptance_probe (tenant_id uuid NOT NULL, ...)` with `ENABLE` but not `FORCE ROW LEVEL SECURITY`. In CI run [40306356003](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40306356003), `integration`, `stack`, `stack-db` and `stack-internal` failed with:

```
migrate: catalogue gate (ADR-001 CI-1, CI-3, CI-4): 1 problems:
CI-1: tenant table public.acceptance_probe does not FORCE ROW LEVEL SECURITY
```

In the stack jobs: `service "migrate" didn't complete successfully: exit 1`.

## Isolation check

Tenant A and tenant B were both seeded in the `make isolation` suite and on the fresh stack. Every runtime role tried to read, update and delete tenant B's rows while tenant A was set, and every attempt reached 0 rows or was denied. The forged-tenant HTTP paths (query string, header, payload, `/api/v1/me` body, baggage) all stayed in tenant A. Inserting a row for tenant B was denied with 42501. Details are in the report's "Registered paths" table.

## Not included

- Human support access across tenants: TICK-40.
- CI-8 (byte-identical 404, stale `If-Match` 412, masked-field filter 400) is partial and finishes in TICK-87. CI-10 finishes in TICK-88.
- Uploading `isolation-report.md` as a CI artifact: TODO-008. It needs a workflow change with `workflow` scope.

## Known limits

- **The `make isolation` report has no role-attribute section**, although the story's criterion 2 points to one. Criterion 2 is proven on every PR by these `integration` tests instead: `TestD28NoSuperuserOrBypassRLS`, `TestD28NoRoleMembershipsRecursive`, `TestDBRolesRolesHaveNoDangerousAttributes`, `TestDBRolesRevokesStrayMemberships`, and the CI-3 catalogue gate after every migrator run. The fresh-stack queries above show the same facts. If you want the section printed in the report, say so and the Technical Lead will file a follow-up.
- **Two accepted low risks from the criterion 3 fix**, with no task: AR-031 and AR-032 in `docs/security/accepted-risks.md`.
  - A first production deploy with a hand grant revokes it, fails, then passes on a re-run. This fails safe.
  - The grant check does not cover types or foreign-data wrappers, which cannot reach tenant rows.
- `nava_backup` has `REPLICATION`. That is how physical backups work (ADR-001 D28). `pg_hba` allows it only to the replication pseudo-database from the backup host.
- The fresh full stack took 188 s to become healthy on a host that also ran 27 other containers. That is above S1's 180 s gate, which applies to CI, not to this story.

## Throwaway PRs

#132, #133 and #143 were draft probe PRs. They are closed and were never merged. To re-run a probe, push its migration from this kit onto a branch of `main` and open a draft PR.
