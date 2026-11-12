# Schema snapshot: `src/database/schema.json`

`docs/handbook/src/database/schema.json` describes the platform database
schema as a freshly migrated database has it. The handbook build
(`tools/handbook`) and the coverage test read this file. They never parse the
migration SQL and never connect to a database, so `task handbook`, `task lint`
and `task test` stay database-free (TICK-718 D-HB-9, TICK-733).

## Who writes it and when it is checked

- `TestSchemaSnapshot` (`internal/db/dbtest`, build tag `integration`)
  migrates a fresh cluster with the real migrator. The run includes the
  runtime-built `audit_log` policies from `db/audited-tables.yaml`. The test
  then reads the catalogue with `catalog.TakeSnapshot` and fails when the
  committed file differs. The failure message names any table that was added
  or removed, or else the first line that differs.
- To refresh the file, run `make schema-snapshot` or `task schema:snapshot`.
  Both start a throwaway cluster and need Docker. Against your own cluster, run
  `go test -tags integration ./internal/db/dbtest -run '^TestSchemaSnapshot$' -update`.
  Commit the new file in the same change as the migration that caused it.
- `TestCommittedSnapshotShape` (`internal/db/catalog`, plain `task test`)
  checks the committed file without a database. The file must be in
  `Marshal`'s exact form (never hand-edit it), with tables sorted, and it must
  contain no connection details.

## Format 1

The file is UTF-8 with LF line endings, a two-space indent and one trailing
newline. HTML characters are not escaped, so `<` stays `<`. Keys appear in the
order shown below. A key with an `omitempty` value is left out when it is
empty or false.

| Key | Meaning |
|---|---|
| `format` | `1`. A consumer rejects any other value. New keys are additive and do not change it. |
| `postgres` | Server major version, for example `"16"`. |
| `migration` | The latest applied goose version, for example `"20261110100000"`. |
| `tables` | Every ordinary, partitioned, partition and foreign table in the platform schemas (every schema except `pg_*` and `information_schema`), sorted by `(schema, name)`. `goose_db_version` is left out because it is the migrator's bookkeeping. |
| `views` | `{schema, name, definition}` for each view, sorted the same way. Empty today. |

**Table names.** A table is referred to by its bare `name` when it is in schema
`public`, and as `schema.name` otherwise. This is the same rule as in
`db/grants.yaml`, and it applies to front-matter `tables:`, `partition_of` and
`references.table`.

Each entry in `tables`:

| Key | Meaning |
|---|---|
| `schema`, `name` | Where the table lives. |
| `kind` | `table`, `partitioned`, `partition` or `foreign`. |
| `partition_key` | Partitioned tables only. `pg_get_partkeydef`, for example `RANGE (occurred_at)`. |
| `partition_of`, `partition_bound` | Partitions only. The parent table and `pg_get_expr(relpartbound)`, for example `DEFAULT`. |
| `rls` | `{enabled, forced}`: ENABLE and FORCE ROW LEVEL SECURITY. |
| `columns` | In `attnum` order. Each is `{name, type, not_null, default?, identity?, generated?}`. `type` is `format_type`. `identity` is `always` or `by_default`. `generated` is the stored expression. |
| `constraints` | Primary key, foreign key, unique, check and exclusion constraints, sorted by name. Each is `{name, type, columns?, references?: {table, columns}, definition, inherited?, not_valid?}`. `type` is one of `primary_key`, `foreign_key`, `unique`, `check` or `exclusion`. `definition` is `pg_get_constraintdef`. `inherited` marks a partition's copy of its parent's constraint. NOT NULL is recorded on the column, not as a constraint. |
| `indexes` | Sorted by name. Each is `{name, method, unique, primary, keys, include?, predicate?, definition, invalid?}`. `keys` and `include` are per-column `pg_get_indexdef`. `predicate` is the partial-index `WHERE`. `definition` is the full `CREATE INDEX`. A partition lists its own copies of its parent's indexes. |
| `policies` | `pg_policies` rows sorted by name. Each is `{name, command, roles, permissive, using?, with_check?}`, with `roles` sorted. The predicates are PostgreSQL's deparse, so the tenant predicate reads exactly as `catalog.TenantPred("tenant_id")`. |

Not in format 1: sequences, functions, triggers, grants and comments. Grants
come from `db/grants.yaml`, and audited or exempt status comes from
`db/audited-tables.yaml`. Adding any of these later is an additive change.

## What it never contains

The file holds no DSN, host, port, role password, `pg_hba.conf` address or
local path. `catalog.SnapshotLeaks` enforces this when the file is written,
using the connection's actual host, user and password as well, and again in
`task test`. The handbook sensitivity gate (TICK-735) scans the file too.
