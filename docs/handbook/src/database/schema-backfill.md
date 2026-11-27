---
title: "Schema backfill: job, queue and failures"
section: database
status: in-progress
adrs: [ADR-009]
stories: [S29]
invariants: [I1, I3]
tables: [schema_backfills, schema_backfill_pending, schema_backfill_failures, entity_definitions]
packages: [internal/schemabackfill, internal/entitytypes]
sources:
  - docs/adr/ADR-009-schema-version-backfill.md
  - docs/data/schema-backfill.md
  - db/migrations/20261127100400_schema_backfill_tables.sql
  - db/migrations/20261127100500_schema_backfill_rollout.sql
  - db/grants.yaml
  - db/audited-tables.yaml
  - internal/entitytypes/backfill.go
tests: [TestP1001Shape, TestP1001TenantIsolation, TestP1001Checks, TestP1001OneActiveJobPerTenant, TestP1001Grants]
---

**Built on the story branch `TICK-740/search-finish` (S29), not yet on
`main`.** Design: [ADR-009](../adr/adr-009.md) D2 and D7. Normative note for
the job: [schema-backfill.md](../../../data/schema-backfill.md).

When a published version gives a field a default, existing records that lack
the field are rewritten in the background by the [`schema.backfill`
workflow](../workflows/schema-backfill.md). These three tables are that job's
state. They hold ids, counts and codes, never a record's values.

## Tables

| Table | One row per | Key | Notes |
|---|---|---|---|
| `schema_backfills` (the job) | type and target version | `(tenant_id, entity_definition_id, target_version)` | `state` (`pending`, `filling`, `running`, `done`, `failed_records`, `superseded`), `default_keys` (every top-level default of the target version), `causation_audit_id` (the publish's audit entry; NULL for a rollout job), counters `rewritten`, `skipped`, `failed`, and `created_at`, `started_at`, `finished_at` |
| `schema_backfill_pending` (the queue) | record still to rewrite | the job's key plus `record_id` | filled once, after the barrier, with every record of the type that lacks one of `default_keys` |
| `schema_backfill_failures` | record that could not be rewritten | same | `code` (`^[a-z_]{1,64}$`, for example `validation_failed` or `db_error`) and `failed_at`. **No value column**, by design |

- The queue and the failures have a foreign key to their job, `ON DELETE
  CASCADE`: a superseded job's rows go with it.
- All three have a composite foreign key `(tenant_id, entity_definition_id)` to
  `entity_definitions`, so a row cannot name another tenant's type.
- `record_id` has **no** foreign key to `records`: a queued id never blocks or
  slows a record delete. The rewrite treats a missing record as `not_found`
  and moves on.
- `(tenant_id, record_id)` indexes on the queue and the failures serve the
  ADR-009 D12 filter shape, which joins records to them while a job is open.
- **One active job per tenant (D7).** The partial unique index
  `schema_backfills_one_active_per_tenant ON (tenant_id) WHERE state IN
  ('filling', 'running')` is the admission rule. A second job of the same
  tenant that tries to start gets `23505` and waits. Another tenant is not
  affected (`TestP1001OneActiveJobPerTenant`).

## Who writes them

| Writer | What | How |
|---|---|---|
| Entity-type publish (`nava_app`) | one `pending` job row, in the publish transaction, with its outbox row | `entitytypes.queueBackfill`. `nava_app`'s only write on these tables: the pinned policy `schema_backfills_insert_app_pending` admits a row of its own tenant only in `pending`, with zero counters and no timestamps (PLAN-010 D-10-11) |
| Rollout migration (`nava_migrator`, once) | one `pending` job and one outbox row per tenant and type that already had a default before S29 | `20261127100500_schema_backfill_rollout.sql` (D15). It skips a type that combines a default with `x-nava-unique` and logs a `schema_backfill_skipped` NOTICE. Running it again inserts nothing, and it never reads or writes `records` |
| `schema.backfill` workflow (`nava_worker`) | every state change, the queue, the failures, the counters | `internal/schemabackfill`, one tenant transaction per step |

## RLS and grants

All three tables are `ENABLE` and `FORCE` row-level security, with the
canonical tenant predicate. No new role, no `BYPASSRLS`, and no new entry on
the RLS allow-list.

| Table | `nava_app` | `nava_worker` |
|---|---|---|
| `schema_backfills` | SELECT; INSERT of a fresh `pending` row only | SELECT, INSERT, UPDATE, DELETE |
| `schema_backfill_pending` | SELECT | SELECT, INSERT, UPDATE, DELETE |
| `schema_backfill_failures` | SELECT | SELECT, INSERT, UPDATE, DELETE |

With no `app.tenant_id` set, each table reads zero rows. `TestP1001Grants` and
`TestP1001TenantIsolation` pin this, and S29 attack row 1 attacks it as both
roles (see [S29](../stories/s29.md)).

The three tables are `exempt` in `db/audited-tables.yaml`: they are job state.
What the job changes is audited where it happens: one `update` entry per
rewritten record, plus a "started" and a "finished" entry on the entity type
(see [the workflow page](../workflows/schema-backfill.md#audit)).

## Generated: diagram and tables

<!-- gen:er area=schema-backfill -->

<!-- gen:tables area=schema-backfill -->

## Related

- [Schema backfill workflow](../workflows/schema-backfill.md).
- [Entity types](entity-definitions.md), whose new versions create jobs.
- [ADR-009](../adr/adr-009.md) and the [S29 story](../stories/s29.md).
