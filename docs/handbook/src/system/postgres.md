---
title: "postgres"
section: system
status: delivered
sources: [compose.yaml, db/grants.yaml, db/pg_hba.conf, docs/budgets.md]
adrs: [ADR-001]
invariants: [I1]
tests: [TestCI2EveryRoleEveryTenantTable, TestPgHBA, TestConnectionBudgetFits]
---

PostgreSQL 16, the one database server of the stack. It holds the platform database `nava` and the service databases `keycloak`, `temporal` and `temporal_visibility`.

| | |
|---|---|
| Image | `postgres:16.10` |
| Networks | `internal`, `temporal-backplane` |
| Listeners | `:5432` (TCP), Unix socket in the `pgsocket` volume |
| Host port | none |
| Roles that log in | through PgBouncer: `nava_app`, `nava_audit_reader`, `nava_worker`, `nava_relay`, `nava_purge`, `nava_onboard`, `nava_groupsync`. Direct: `nava_migrator` (migrate), `keycloak`, `temporal`, and the bootstrap superuser over the socket only (db-roles). `nava_erasure` and `nava_backup` exist, but no service in the stack runs as them |
| Introduced by | PLAN-001 T02 (TICK-51) |

## What talks to it

- **pgbouncer**: the server pools for every `nava-*` service.
- **migrate**, **keycloak**, **temporal-schema**, **temporal-server**: direct TCP connections.
- **db-roles**: the superuser, over the Unix socket. `db/pg_hba.conf` replaces the image's file and opens no other path to the superuser.

Local settings: `fsync`, `synchronous_commit` and `full_page_writes` are off (dev data is disposable; reset with `docker compose down -v`). `max_connections=250`; every pool must fit under it, and the sum is kept in `docs/budgets.md` under "Connection sizing".

## Boundary it enforces

**I1, the tenant wall in the database.** Every tenant table has row-level security forced on, keyed on `app.tenant_id`. `TestCI2EveryRoleEveryTenantTable` runs every runtime role against every tenant table and proves it sees only its own tenant's rows. `TestPgHBA` pins the socket-only superuser rule.

[Back to the system map](index.md)
