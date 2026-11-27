---
title: "Records"
section: database
status: delivered
tables: [records, record_write_keys]
adrs: [ADR-001, ADR-004, ADR-010]
stories: [S31]
sources: [docs/data/org-units.md, docs/data/org-unit-closure.md, docs/data/records-list-indexes.md, docs/data/records-list-bench.md, docs/data/record-write-keys-per-principal.md]
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
  are no expression indexes (PLAN-003 D25). Filterable fields ([ADR-006](../adr/adr-006.md)) are built on the S21 story branch, not yet on `main`: a flagged field's values are copied into typed side tables, described in [Field index](field-index.md) and [below](#filters-on-data-fields-s21-story-branch).

**`org_unit_id` (S31, ADR-010 D8).** The org unit a record belongs to, for
the `under` access leaf ("tickets in my region"). The write pipeline sets it
once, on create, from the requester's primary unit, and records it in the
create audit entry. It is never recomputed: moving a person to another unit
leaves their old tickets where they were. NULL means no unit; such a record
is in nobody's region. A request body cannot set it: the schema compiler
refuses a property named `org_unit_id`, and the pipeline refuses a body that
names it with `422 field_not_writable` (`refuseOrgUnit`). It is not in a record's API body
or in its history (TICK-1119).
`nava_app` and `nava_worker` have no `UPDATE` on the column (TICK-1120): an
injection or a code bug in either service cannot move a ticket, not even inside
its own tenant (`TestTICK947RecordOrgUnitIsTenantScoped`).
The foreign key `records_org_unit_fkey` carries `tenant_id`, so a record
cannot name another tenant's unit, and units are tombstoned, never deleted
([Org units](org-units.md)). A write that changes it cannot commit without
the record's audit entry. Its index, `records_org_unit`, is
`(tenant_id, org_unit_id, entity_definition_id) WHERE org_unit_id IS NOT NULL`
(TICK-1032): tenants with no regions add nothing to it.

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

## Filters on data fields (S21, story branch)

On the `TICK-739/filterable-fields` branch (not yet on `main`), a list filter
on a `data` field is resolved against the [field-index
registry](field-index.md) after the masked-field check: not flagged is `400
field_not_filterable` for every type (the seed fields `incident.priority`,
`request.priority` and `change.risk` are flagged; the JSONB scan is gone),
`backfilling` is `422
field_not_ready` with `index_progress`, and `ready` compiles to one correlated
`EXISTS` per field on the typed side table, inside the list statement that
carries the row ACL (ADR-006 D6, D7; TICK-746, TICK-989). All bounds on one
field share the one `EXISTS`, so on an array field a single element must
satisfy every bound. See the [S21 story page](../stories/s21.md).
