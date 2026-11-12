---
title: "Workflow state"
section: database
status: delivered
tables: [workflow_ref, cascade_start]
adrs: [ADR-001, ADR-001-A5]
sources: [docs/data/cascade-start.md, docs/data/purge-indexes.md]
---

Two small tables track workflow starts:

- **`workflow_ref`**: one row per start queued in the outbox, linking the
  outbox row to its record. The final activity sets `closed_at`, and
  `nava_purge` deletes rows closed more than 30 days ago.
- **`cascade_start`**: the start key of a cascade (ADR-001-A5). A write
  caused by a workflow carries the origin of the chain that caused it. A
  definition starts at most once per `(tenant, origin, record)`: the start
  claims its row with `INSERT ... ON CONFLICT DO NOTHING`, so a cycle cannot
  start the same workflow twice. It is insert-only.

Both are exempt from the audit log, because the write that queued the start
is audited.

## Diagram

<!-- gen:er area=workflow-state -->

## Tables

<!-- gen:tables area=workflow-state -->
