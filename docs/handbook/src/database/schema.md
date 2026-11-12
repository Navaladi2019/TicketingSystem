---
title: "Schema snapshot (schema.json)"
section: database
sources: [docs/handbook/build/schema.sql, docs/handbook/build/schema_json.py, db/migrations]
status: delivered
---

# Schema snapshot: `schema.json`

`docs/handbook/src/database/schema.json` describes the platform database
schema as a freshly migrated database has it. The handbook build
(`docs/handbook/build/build.py`) reads this file. It never parses the
migration SQL and never connects to a database (TICK-718 D-HB-9, TICK-733).

This page is the contract between the writer (TICK-733) and the handbook
reader (TICK-719), as the Technical Lead decided on TICK-733.

## Regenerating it

The snapshot is generated, never hand-edited. Regenerate it after a migration
lands on `main`, and commit it with the handbook update for that change.

1. Start the local stack (`docker compose up`), so that the `migrate` one-shot
   has migrated the platform database `nava`. The migrator also builds the
   runtime `audit_log` policies from `db/audited-tables.yaml`.
2. From the repository root, run the one command:

   ```sh
   docker compose exec -T postgres sh -c 'PGPASSWORD="$(cat /run/pg_superuser_password)" psql -h /var/run/postgresql -U pgadmin -d nava -XAtq -v ON_ERROR_STOP=1' < docs/handbook/build/schema.sql | python3 docs/handbook/build/schema_json.py docs/handbook/src/database/schema.json
   ```

   On Windows, use `python` instead of `python3` if that is your interpreter's
   name.

The two build files do the following:

- `docs/handbook/build/schema.sql` reads the catalogue and prints one JSON
  document. Every name is sorted with `COLLATE "C"` (byte order), so the
  output does not depend on the database's default collation.
- `docs/handbook/build/schema_json.py` uses only the Python standard library.
  It checks the document against the key table below: exact keys per object,
  `kind` values, the partition-key rules, and byte order. It refuses a table
  outside schema `public` or a foreign table, and it refuses connection
  details (see the last section). It writes the file only when every check
  passes.

## Version 1

The file is UTF-8 with LF line endings, a two-space indent and one trailing
newline. Non-ASCII characters are written as-is, not escaped.

Every key below is always present: a string that does not apply is `""`, and
a list that is empty is `[]`. A reader ignores keys it does not know, so a new
key is additive and does not change `version`. Renaming or removing a key
bumps `version`.

The snapshot covers schema `public` only. Tables are named by their bare
name. `goose_db_version`, the migrator's bookkeeping table, is left out.

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
triggers, grants, owners, comments and table areas. Grants come from
`db/grants.yaml`, audited or exempt status from `db/audited-tables.yaml`, and
the table-to-area mapping from `docs/handbook/src/database/areas.yaml` on the
handbook side.

## What it never contains

The file holds no DSN, host, port, role password, `pg_hba.conf` address or
local path. Version 1 has no key that could carry one. `schema_json.py`
refuses to write a file that matches any of its leak patterns, and HB-12's
manual sensitivity sweep covers the file. A future version
that adds a free-text key, such as comments or owners, needs a new
sensitivity review first.
