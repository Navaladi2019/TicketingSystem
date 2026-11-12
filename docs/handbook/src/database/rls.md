---
title: "Row-level security"
section: database
status: delivered
sources: [docs/handbook/src/database/schema.json, internal/db/catalog/policy.go, docs/adr/ADR-001-platform-blueprint.md, db/audited-tables.yaml]
adrs: [ADR-001, ADR-004]
packages: [internal/db/catalog]
---

Every table runs under `ENABLE` and `FORCE ROW LEVEL SECURITY`, so its
policies apply to the owner as well. Almost every policy is the same tenant
predicate, so it is written once below, and the per-table table only names the
policies that use it.

The catalogue gate (`internal/db/catalog`, run at the end of every migrator
run) enforces the rules this page shows. It fails when a policy is `FOR ALL`
or `TO PUBLIC`, when a policy name is used on two tables, or when a policy is
not exactly the tenant predicate and is not one of the pinned exceptions
below. An exception that differs from its pinned text by one character also
fails the gate.

The `audit_log` policies are built when the migrator runs, from
`db/audited-tables.yaml` (ADR-004 R12), so no migration file holds them. They
appear here because this page reads a migrated database's catalogue
(`schema.json`), not the migration SQL. A partition has no policies of its
own: a query through the parent uses the parent's, and the `DEFAULT`
partition, queried directly, admits no runtime role.

<!-- gen:rls -->
