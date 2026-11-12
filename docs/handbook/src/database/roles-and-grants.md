---
title: "Roles and grants"
section: database
status: delivered
sources: [db/grants.yaml, db/audited-tables.yaml, db/pg_hba.conf, docs/data/roles-and-grants.md, docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-004-A1-diff-read-role.md]
adrs: [ADR-001, ADR-004, ADR-004-A1]
---

`db/grants.yaml` is the only place a privilege is granted. Two jobs read it:
`db-roles`, as the bootstrap superuser, creates each role with exactly the
attributes listed (always `NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS`)
and revokes any membership; `migrate`, as `nava_migrator`, grants what each
role lists and revokes, and logs, anything else on an object it owns,
`PUBLIC` included. The database itself starts with `REVOKE ALL ... FROM
PUBLIC` (`00001_global_revokes.sql`).

A grant decides which columns a role may touch. Row-level security decides
which rows: every table is under `FORCE ROW LEVEL SECURITY`, so a grant here
reaches only the rows its [policies](rls.md) allow.

Column lists matter. `UPDATE` never includes a key or a column that is set
once at insert, so a record's number or a group's identity provider id cannot
be rewritten. Only `nava_audit_reader` may read `audit_log.diff` and
`diff_salt` (ADR-004-A1).

## Roles

<!-- gen:grants -->

## Audited and exempt tables

Every tenant table is either audited, with a commit-check trigger that makes a
write fail unless it wrote its audit entry in the same transaction, or exempt
with a reason (`db/audited-tables.yaml`, ADR-004 R12). The gate fails on a
table that is neither.

<!-- gen:audited -->
