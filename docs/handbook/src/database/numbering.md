---
title: "Numbering"
section: database
status: delivered
tables: [record_counters]
adrs: [ADR-005, ADR-005-E1]
sources: [docs/adr/ADR-005-record-numbering.md, docs/adr/ADR-005-E1-first-create-race-in-expand-window.md, docs/data/record-counters-scope-key.md, db/migrations/20261112193000_record_counters_scope_columns.sql, db/migrations/20261112193100_record_counters_scope_key_len_validate.sql, db/migrations/20261112193200_record_counters_scoped_key.sql, db/migrations/20261112193300_record_counters_period_end.sql, db/migrations/20261113034010_record_counters_pkey_scope_key.sql]
stories: [S17, S18]
---

`record_counters` holds one row per `(tenant_id, entity_definition_id,
scope_key)`: the next record number of that type in that scope, keyed by
`record_counters_pkey` on those three columns since M3 (S18). The default
scope `''` is the counter of the default format; S18's number formats add
scopes beside it. A record takes its number with one upsert, `INSERT ... ON
CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next`, so
`next` is the only column a role may update; the insert arm sets `scope_key`
and `period_end`.

The type's `numbering_mode` (ADR-005) decides when the number is taken. A
`gaps` type takes it in its own short transaction before the create, so a
failed create leaves a gap. A `gapless` type takes it inside the create
transaction, so a rollback gives the number back. The table is exempt from the
audit log, because the record insert that takes the number is audited.

## Scope key and period end (S17, release R1)

Four migrations, all online (runbook:
[`record-counters-scope-key.md`](../../../data/record-counters-scope-key.md)):

- **M1** adds `scope_key text NOT NULL DEFAULT ''` and `period_end timestamptz
  NULL`, plus `record_counters_scope_key_len` (at most 128 characters), added
  `NOT VALID` and validated by the next migration. Existing rows read `''`.
- **M2** builds `record_counters_scoped_key`, a unique index on `(tenant_id,
  entity_definition_id, scope_key)` that the upsert names as its arbiter, and
  `record_counters_period_end`, a partial index on `period_end IS NOT NULL`
  that the purge job will read. Both are built `CONCURRENTLY`; the partial
  index stays empty until S18 publishes period formats.
- The primary key is still `(tenant_id, entity_definition_id)`. Until M3
  (S18, release R2) both unique indexes exist, so the pre-S17 build still runs
  and a rollout with mixed binaries works for existing counters.
- A type's first create can lose a race for its counter row with `23505` on
  the index the upsert does not name (ADR-005-E1). R1 retries once. The
  pre-S17 build has no retry (accepted risk AR-S17-01).
- Rolling back the S17 schema before M3 is `goose down-to 20261112192959`,
  only once no R1 replica serves.

## Primary key on the scope (S18, release R2: M3)

- **M3** (`20261113034010`) drops the 2-column key and promotes
  `record_counters_scoped_key` with `ADD PRIMARY KEY USING INDEX`, so the key
  keeps the name `record_counters_pkey` and covers `scope_key`. No build, no
  scan. It waits at most 2 s for its lock and retries up to 5 times with
  backoff (about 18 s at most), then fails with the table unchanged.
- From here a type can have rows of several scopes. There is one unique
  index, so first creates of a counter just wait on it: the E1 race and R1's
  retry are gone.
- **M3 is the point of no return.** The pre-S17 build's `ON CONFLICT
  (tenant_id, entity_definition_id)` fails with `42P10`, so it can never be
  deployed again. Its Down restores the 2-column key only while every
  `scope_key` is `''`, and refuses otherwise.

## Diagram

<!-- gen:er area=numbering -->

## Tables

<!-- gen:tables area=numbering -->
