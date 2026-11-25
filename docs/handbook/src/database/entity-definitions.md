---
title: "Entity definitions"
section: database
status: delivered
tables: [entity_definitions, entity_definition_versions]
adrs: [ADR-001, ADR-004, ADR-005, ADR-010]
stories: [S31]
sources: [docs/data/platform-tables.md, docs/data/entity-def-versions-jsonb-check.md, docs/data/entity-versions-org-unit-id-reserved.md, db/migrations/20261120235900_entity_versions_org_unit_id_reserved.sql]
tests: [TestTICK1115MigrationRefusesStoredOrgUnitID, TestTICK1115InsertRefusesOrgUnitID, TestTICK1125Repairs]
---

An entity type (`entity_definitions`) is a per-tenant row, not a table, so
a tenant defines its own types with no DDL. Its JSON Schema lives in
`entity_definition_versions`, one immutable row per published version.
Publishing inserts a version and moves `entity_definitions.current_version`.
The platform never updates a version, so a record written under version 3 can always be
read against version 3.

`key` and `number_prefix` are set once. `numbering_mode` (`gaps` or
`gapless`, ADR-005) decides how the type's record numbers are allocated (see
[numbering](numbering.md)). Onboarding seeds a provisioning tenant's default
types under pinned policies. Both tables are audited.

## The reserved name `org_unit_id` (S31)

S31 keeps a ticket's org unit in `records.org_unit_id`, so a type may not have
a top-level property of that name. Two layers refuse it:

| Layer | Refuses |
|---|---|
| The schema compiler | Publish and create: `422 invalid_definition` at `/definition/schema/properties/org_unit_id` |
| CHECK `entity_definition_versions_org_unit_id_reserved` (TICK-1115) | Any insert, from any role, with 23514. It also stops the S31 upgrade when a stored version has the property, and refuses a publish by an old replica during the rollout |

The CHECK finds an escaped spelling such as `\u006frg_unit_id`, and adds 0.16
to 0.23 ms to a publish of a 2.4 kB schema (PostgreSQL 16.10, loaded host).

The API cannot repair a stored version that has the property, so an operator
does: [Migration history](migrations.md#upgrading-to-s31). That repair is the one
case where a stored version's schema changes. Decision and figures:
[entity-versions-org-unit-id-reserved.md](../../../data/entity-versions-org-unit-id-reserved.md).

## Diagram

<!-- gen:er area=entity-definitions -->

## Tables

<!-- gen:tables area=entity-definitions -->
