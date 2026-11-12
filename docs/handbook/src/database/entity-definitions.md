---
title: "Entity definitions"
section: database
status: delivered
tables: [entity_definitions, entity_definition_versions]
adrs: [ADR-001, ADR-004, ADR-005]
sources: [docs/data/platform-tables.md, docs/data/entity-def-versions-jsonb-check.md]
---

An entity type (`entity_definitions`) is a per-tenant row, not a table, so
a tenant defines its own types with no DDL. Its JSON Schema lives in
`entity_definition_versions`, one immutable row per published version.
Publishing inserts a version and moves `entity_definitions.current_version`.
A version is never updated, so a record written under version 3 can always be
read against version 3.

`key` and `number_prefix` are set once. `numbering_mode` (`gaps` or
`gapless`, ADR-005) decides how the type's record numbers are allocated (see
[numbering](numbering.md)). Onboarding seeds a provisioning tenant's default
types under pinned policies. Both tables are audited.

## Diagram

<!-- gen:er area=entity-definitions -->

## Tables

<!-- gen:tables area=entity-definitions -->
