---
title: "Numbering"
section: database
status: delivered
tables: [record_counters]
adrs: [ADR-005]
sources: [docs/adr/ADR-005-record-numbering.md]
---

`record_counters` holds one row per `(tenant_id, entity_definition_id)`:
the next record number for that type. A record takes its number with one
upsert, `INSERT ... ON CONFLICT DO UPDATE SET next`, so `next` is the only
column a role may update.

The type's `numbering_mode` (ADR-005) decides when the number is taken. A
`gaps` type takes it in its own short transaction before the create, so a
failed create leaves a gap. A `gapless` type takes it inside the create
transaction, so a rollback gives the number back. The table is exempt from the
audit log, because the record insert that takes the number is audited.

## Diagram

<!-- gen:er area=numbering -->

## Tables

<!-- gen:tables area=numbering -->
