---
title: "Numbering"
section: database
status: delivered
tables: [record_counters, number_formats, number_format_versions, numbering_epochs, entity_definition_versions, tenants]
adrs: [ADR-005, ADR-005-E1, ADR-005-E2]
sources: [docs/adr/ADR-005-record-numbering.md, docs/adr/ADR-005-E1-first-create-race-in-expand-window.md, docs/data/record-counters-scope-key.md, db/migrations/20261112193000_record_counters_scope_columns.sql, db/migrations/20261112193100_record_counters_scope_key_len_validate.sql, db/migrations/20261112193200_record_counters_scoped_key.sql, db/migrations/20261112193300_record_counters_period_end.sql, db/migrations/20261113210000_record_counters_pkey_scope_key.sql, docs/data/number-formats.md, db/migrations/20261113210100_number_format_tables.sql, db/migrations/20261113210200_entity_definition_versions_number_format.sql, db/migrations/20261113210400_tenants_time_zone.sql, db/migrations/20261113210800_record_counters_purge.sql, db/migrations/20261113210900_tenants_time_zone_update_policy.sql, db/migrations/20261115100000_tenants_select_worker.sql, internal/purge/purge.go]
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
  that the purge job reads. Both are built `CONCURRENTLY`; the partial
  index holds only the period counters of S18's dated formats.
- The primary key is still `(tenant_id, entity_definition_id)`. Until M3
  (S18, release R2) both unique indexes exist, so the pre-S17 build still runs
  and a rollout with mixed binaries works for existing counters.
- A type's first create can lose a race for its counter row with `23505` on
  the index the upsert does not name (ADR-005-E1). R1 retries once. The
  pre-S17 build has no retry (accepted risk AR-S17-01).
- Rolling back the S17 schema before M3 is `goose down-to 20261112192959`,
  only once no R1 replica serves.

## Primary key on the scope (S18, release R2: M3)

- **M3** (`20261113210000`) drops the 2-column key and promotes
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

## Number formats, epochs and the pinned link (S18)

Design note: [`number-formats.md`](../../../data/number-formats.md). Three
tenant tables, each under `ENABLE` and `FORCE ROW LEVEL SECURITY`, with every
key and foreign key leading with `tenant_id` (ADR-005 §3.7 C8: a foreign-key
check runs as the owner and skips RLS).

- **`number_formats`**: one row per format, `key` unique per tenant (never
  globally, which would be a cross-tenant oracle) and `current_version`.
  `nava_app` creates formats and moves `current_version` and `updated_at`.
- **`number_format_versions`**: immutable. `document` is the format as the
  admin published it (template, overflow, optional time-zone override);
  `compiled` is `numbering.Compiled` as JSON and `compiled_sha256` its
  SHA-256, computed in Go. No role may update or delete a version (C9).
- **`numbering_epochs`**: the identity of a compiled template within one
  type. A publish whose `compiled_sha256` is already there reuses its epoch;
  otherwise it takes `max(epoch) + 1` after `SELECT ... FOR UPDATE` on the
  type. Epoch 0 (`{prefix}{seq:7}`, scope `''`) is implicit and has no row.
  `literal_head` (the text before the first variable token, `{prefix}`
  resolved) is what the disjointness check compares across the type's
  epochs, so two epochs can never render the same number. Immutable.
- **The pinned link** is on the immutable `entity_definition_versions`:
  `number_format_id`, `number_format_version` and `number_compiled`, the
  snapshot a create reads, so a create never reads the format tables. All
  three NULL is the default numbering. The composite foreign key leads with
  `tenant_id`, so tenant B's format id fails exactly as a made-up id does
  (23503); the API answers both with `422 number_format_not_found`.
- **`tenants.time_zone`**: `text NOT NULL DEFAULT 'UTC'`, an IANA name. The
  CHECK pins its shape; whether the zone exists is checked in Go against the
  binary's tzdata. `nava_app` updates it through its own policy, and the
  change is audited under subject type `tenant_setting`. `nava_worker` reads
  it too (`20261115100000`), because a dated create reads the zone inside the
  counter upsert.
- Audited under subject type `number_format` (formats, versions) and
  `entity_definition` (epochs).

## Scope keys and the purge of period counters (S18)

- A linked format's counter scope is `e<epoch>` for a date-free format, or
  `e<epoch>|d:<date part>` for a dated one (`e1|d:202611`). PostgreSQL
  computes the scope key, the date part and `period_end` from `now()` in the
  tenant's time zone, in the upsert itself, so the date in the number and
  the counter it came from always agree.
- `period_end` is set only for a row whose scope ends (a dated format with a
  year). Rows of recurring keys (`{MM}` with no year, `{dd}` with no month)
  and default-scope rows keep `period_end` NULL, so they are never purged:
  a purged and restarted counter would hand out numbers already issued.
- `nava_purge` deletes rows whose `period_end` is more than 2 days past,
  through the paired policies `record_counters_purge_select` and
  `record_counters_purge` on one predicate, `period_end < now() - interval
  '2 days'` (`20261113210800`). The 2-day grace exceeds the widest UTC-offset
  span (26 h), so a time-zone change cannot reopen a purged period. NULL never
  matches. Its grant is `SELECT` on the key columns and `period_end`, and
  `DELETE`; never `next`. The batch `DELETE` reads the partial index
  `record_counters_period_end`.

## Diagram

<!-- gen:er area=numbering -->

## Tables

<!-- gen:tables area=numbering -->
