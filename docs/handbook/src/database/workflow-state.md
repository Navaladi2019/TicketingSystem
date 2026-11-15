---
title: "Workflow state"
section: database
status: delivered
tables: [workflow_definitions, workflow_definition_versions, workflow_ref, workflow_cause, cascade_start]
adrs: [ADR-001, ADR-001-A5]
stories: [S11b]
sources: [docs/data/cascade-start.md, docs/data/purge-indexes.md, docs/data/workflow-definitions.md, db/audited-tables.yaml]
---

Workflow definitions, and the small tables that track workflow starts:

- **`workflow_definitions`** and **`workflow_definition_versions`** (S11b,
  P3-40): a tenant's published workflow definitions on an entity type. Each
  version is immutable and keeps the exact body bytes (`body_sha256` is
  checked), the step ids the server assigned (`step_ids`) and its trigger
  (`trigger_on` `record.created` or `record.updated`, and
  `trigger_entity_definition_id`). `workflow_definitions.current_version` is
  the version new runs start on. The records pipeline's event sink finds the
  definitions a write fires through the index
  `workflow_definition_versions_trigger`. Both tables are audited under the
  `workflow_definition` subject type.
- **`workflow_ref`**: one row per start queued in the outbox, linking the
  outbox row to its record. The final activity sets `closed_at`, and
  `nava_purge` deletes rows closed more than 30 days ago. While a run waits on
  its record's `record_changed` signal, `record_changed_signal_id` holds that
  signal's id (S11b). The event sink signals only those runs, found through
  the index `workflow_ref_waiting`.
- **`workflow_cause`** (S11b, D-3-29): the causation (origin and depth) of
  each delivered start or signal row, written in the same transaction as the
  outbox row. A workflow's record write takes its causation only from its
  run's newest row (index `workflow_cause_ref_newest`), never from the
  workflow's input.
- **`cascade_start`**: the start key of a cascade (ADR-001-A5). A write
  caused by a workflow carries the origin of the chain that caused it. A
  definition starts at most once per `(tenant, origin, record)`: the start
  claims its row with `INSERT ... ON CONFLICT DO NOTHING`, so a cycle cannot
  start the same workflow twice. It is insert-only. Since S11b,
  `definition_id` is a foreign key to `workflow_definitions`, added `NOT VALID`
  and validated in a separate migration.

`workflow_ref`, `workflow_cause` and `cascade_start` are exempt from the audit
log, because the write that queued the start is audited.

## Diagram

<!-- gen:er area=workflow-state -->

## Tables

<!-- gen:tables area=workflow-state -->
