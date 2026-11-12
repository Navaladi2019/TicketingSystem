---
title: "Database"
section: database
status: delivered
sources: [docs/handbook/src/database/schema.json, docs/handbook/src/database/areas.yaml, db/grants.yaml, db/audited-tables.yaml, db/global-tables.yaml, deploy/compose/pgbouncer/pgbouncer.ini, docs/adr/ADR-001-platform-blueprint.md]
adrs: [ADR-001, ADR-004]
---

The platform keeps every tenant's data in one PostgreSQL database, `nava`, in
schema `public`. Every table is a tenant table: it has a tenant key, and it
runs under `ENABLE` and `FORCE ROW LEVEL SECURITY` (ADR-001 I1).
`db/global-tables.yaml`, the list of tables exempt from that rule, is empty.

The pages in this section are generated from four files. Nothing here is
copied by hand from a migration:

| What | From |
|---|---|
| Tables, columns, keys, indexes, partitions, RLS flags and policies | `docs/handbook/src/database/schema.json`, a snapshot of a freshly migrated database ([how it is made](schema.md)) |
| Which area a table belongs to | `docs/handbook/src/database/areas.yaml` |
| Roles and grants | `db/grants.yaml` |
| Audited and exempt tables | `db/audited-tables.yaml` |
| Migration history | `db/migrations` and `docs/handbook/src/database/migration-history.json` |

## Areas

Every table is in exactly one area. The build fails on a table that is in no
area, or in two, so a new table cannot land without a place here.

<!-- gen:areas -->

Across the whole schema:

- [Row-level security](rls.md): the tenant predicate, and the few policies that differ from it.
- [Roles and grants](roles-and-grants.md): one role per system actor, and what each may touch.
- [Migration history](migrations.md): every migration, the change that added it, and how it runs online.

## Pooling and the role model

Each system actor connects as its own login role (ADR-001 D28): the API as
`nava_app`, workflow workers as `nava_worker`, the outbox relay as
`nava_relay`, and so on. No role is a superuser, has `BYPASSRLS`, owns an
object or is a member of another role. `nava_migrator` owns every object,
runs only as the one-shot migrate job and is granted to no one.

Runtime roles reach PostgreSQL through PgBouncer in **transaction** pool mode
(`deploy/compose/pgbouncer/pgbouncer.ini`), with `max_prepared_statements > 0`
so prepared statements work across pooled server connections. A role is let
through only when `db/grants.yaml` sets `pgbouncer_login: true` for it.
`nava_migrator`, `nava_erasure` and `nava_backup` never go through PgBouncer.

Because a server connection is shared between clients, the tenant is set per
transaction, never per session: the database helper runs
`set_config('app.tenant_id', $1, true)` at the start of each transaction, and
nothing else may call `set_config` (ADR-001 A.1 step 5). Every tenant policy
reads that setting, so a transaction with no tenant set sees no rows. The
setting dies with the transaction, so it cannot leak to the next client of a
pooled connection.
