---
title: "Schema snapshot (schema.json)"
section: database
sources: [internal/db/catalog/snapshot.go, internal/db/catalog/snapshot_keys.go, internal/db/dbtest/schema_snapshot_test.go]
status: delivered
---

# Schema snapshot: `schema.json`

`docs/handbook/src/database/schema.json` describes the platform database
schema as a freshly migrated database has it. The handbook build
(`tools/handbook`) and the coverage test read this file. They never parse the
migration SQL and never connect to a database, so `task handbook`, `task lint`
and `task test` stay database-free (TICK-718 D-HB-9, TICK-733).

This page is the contract between the writer (TICK-733) and the handbook
reader (TICK-719), as the Technical Lead decided on TICK-733. The key table
below is machine-read: a test fails when the snapshot is missing a key that
the table lists, or carries a key that the table does not list.

## Who writes it and when it is checked

- `TestSchemaSnapshot` (`internal/db/dbtest`, build tag `integration`)
  migrates a fresh cluster with the real migrator. The run includes the
  runtime-built `audit_log` policies from `db/audited-tables.yaml`. The test
  then reads the catalogue with `catalog.TakeSnapshot` and checks the result
  against the key table on this page. It fails when the committed file
  differs, and the failure message names any table that was added or removed,
  or else the first line that differs.
- To refresh the file, run `make schema-snapshot` or `task schema:snapshot`.
  Both start a throwaway cluster and need Docker. Against your own cluster, run
  `go test -tags integration ./internal/db/dbtest -run '^TestSchemaSnapshot$' -update`.
  Commit the new file in the same change as the migration that caused it.
- `TestCommittedSnapshotShape` (`internal/db/catalog`, plain `task test`)
  checks the committed file against this page without a database. The file
  must be in `Marshal`'s exact form (never hand-edit it), with tables sorted,
  and it must contain no connection details.

## Version 1

The file is UTF-8 with LF line endings, a two-space indent and one trailing
newline. HTML characters are not escaped, so `<` stays `<`.

Every key below is always present: a string that does not apply is `""`, and
a list that is empty is `[]`. A reader ignores keys it does not know, so a new
key is additive and does not change `version`. Renaming or removing a key
bumps `version`.

The snapshot covers schema `public` only. Tables are named by their bare
name. `catalog.TakeSnapshot` fails on a table in any other schema, and on a
foreign table, rather than leave it out silently. `goose_db_version` (the
migrator's bookkeeping) is left out.

| Key | Value |
|---|---|
| `version` | `1`. A reader rejects any other value. |
| `tables` | Every table in `public`, sorted by `name`. |
| `tables[].name` | The table name. |
| `tables[].kind` | Exactly one of `table`, `partitioned` or `partition`. |
| `tables[].partition_of` | A partition's parent table, else `""`. |
| `tables[].partition_by` | A partitioned table's key (`pg_get_partkeydef`), for example `RANGE (occurred_at)`, else `""`. |
| `tables[].partition_bound` | A partition's bound (`pg_get_expr(relpartbound)`), for example `DEFAULT`, else `""`. |
| `tables[].rls` | Row level security flags. |
| `tables[].rls.enabled` | ENABLE ROW LEVEL SECURITY. |
| `tables[].rls.forced` | FORCE ROW LEVEL SECURITY. |
| `tables[].columns` | Columns in ordinal (`attnum`) order. |
| `tables[].columns[].name` | Column name. |
| `tables[].columns[].type` | `format_type`, for example `timestamp with time zone`. |
| `tables[].columns[].nullable` | `true` unless the column is NOT NULL. |
| `tables[].columns[].default` | The deparsed default expression, else `""`. |
| `tables[].primary_key` | Primary key columns in key order, `[]` when there is no primary key. |
| `tables[].foreign_keys` | Foreign keys sorted by `name`. A partition lists its own copies of its parent's. |
| `tables[].foreign_keys[].name` | Constraint name. |
| `tables[].foreign_keys[].columns` | Referencing columns in key order. |
| `tables[].foreign_keys[].ref_table` | Referenced table. |
| `tables[].foreign_keys[].ref_columns` | Referenced columns in key order. |
| `tables[].foreign_keys[].on_delete` | `NO ACTION`, `RESTRICT`, `CASCADE`, `SET NULL` or `SET DEFAULT`. |
| `tables[].indexes` | Indexes sorted by `name`, including the primary key's. A partition lists its own copies of its parent's. |
| `tables[].indexes[].name` | Index name. |
| `tables[].indexes[].method` | Access method, for example `btree` or `gin`. |
| `tables[].indexes[].unique` | `true` for a unique index. |
| `tables[].indexes[].columns` | Key columns in order. An expression index's entry is the deparsed expression. |
| `tables[].indexes[].include` | The INCLUDE columns, `[]` when there are none. |
| `tables[].indexes[].predicate` | A partial index's deparsed `WHERE`, else `""`. |
| `tables[].policies` | `pg_policies` rows sorted by `name`. |
| `tables[].policies[].name` | Policy name. |
| `tables[].policies[].command` | `SELECT`, `INSERT`, `UPDATE`, `DELETE` or `ALL`. |
| `tables[].policies[].roles` | Roles, sorted. |
| `tables[].policies[].permissive` | `true` for PERMISSIVE, `false` for RESTRICTIVE. A RESTRICTIVE policy changes what isolation means, so the sensitivity check must see it. |
| `tables[].policies[].using` | The deparsed `USING`, else `""`. |
| `tables[].policies[].check` | The deparsed `WITH CHECK`, else `""`. |

The predicates are PostgreSQL's own deparse, so the tenant predicate reads
exactly as `catalog.TenantPred("tenant_id")`.

Not in version 1: check and unique constraints, sequences, functions,
triggers, grants and table areas. Grants come from `db/grants.yaml`, audited
or exempt status from `db/audited-tables.yaml`, and the table-to-area mapping
from `docs/handbook/src/database/areas.yaml` on the handbook side.

## What it never contains

The file holds no DSN, host, port, role password, `pg_hba.conf` address or
local path. `catalog.SnapshotLeaks` enforces this when the file is written,
using the connection's actual host, user and password as well, and again in
`task test`. The handbook sensitivity gate (TICK-735) scans the file too.
