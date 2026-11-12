---
title: "Records"
section: database
status: delivered
tables: [records, record_write_keys]
adrs: [ADR-001, ADR-004]
sources: [docs/data/records-list-indexes.md, docs/data/records-list-bench.md, docs/data/record-write-keys-per-principal.md]
---

`records` holds every tenant's tickets and other entities in one table.
Attributes are split two ways:

- **Promoted columns** are the attributes every type has and that lists
  filter and sort on: `entity_definition_id`, `number`, `state`,
  `requester_id`, `assignee_id`, `assignment_group_id`, `created_at` and
  `updated_at`, plus `version` for `If-Match`. Each has a type, a foreign key
  where it names another row, and a B-tree index where a list needs one.
- **`data jsonb`** holds the rest: the fields the type's JSON Schema defines.
  The service validates it against the schema version in `schema_version`.
  The diagram shows it as one attribute. Nothing indexes `data`, and there
  are no expression indexes (PLAN-003 D25). Filterable fields, which would add typed side tables, are accepted in [ADR-006](../adr/adr-006.md) but not on `main`.

The list indexes all start with `(tenant_id, entity_definition_id)` and end
with the sort key and `id`, so every allowed sort is an ordered index scan
with no `Sort` node. The evidence is in `docs/data/records-list-indexes.md`,
linked from each index below.

`record_write_keys` makes a pipeline write idempotent: a retry finds the key
and its audit entry instead of writing twice. `records` is audited.
`record_write_keys` is exempt, because the write it deduplicates is
audited.

## Diagram

<!-- gen:er area=records -->

## Tables

<!-- gen:tables area=records -->
