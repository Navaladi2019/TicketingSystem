## nava_worker plans with force_custom_plan (TICK-990)

`db/grants.yaml` gives `nava_worker` `plan_cache_mode = force_custom_plan`, the setting `nava_app` has had since P3-01 (TICK-339). db-roles applies it role-wide and resets drift.

### Why

The field-index backfill's keyset step (ADR-006 D10.2 (b)) runs on a pgx cached statement:

```sql
SELECT id FROM records WHERE tenant_id = $1 AND entity_definition_id = $2 AND id > $3
ORDER BY id LIMIT $4 FOR SHARE
```

Under the default `auto`, PostgreSQL switches to the generic plan after five executions on one connection. That plan estimates 2 rows, so it costs less than the custom plan. It then reads every record of the type through another index and sorts them. P7-07 (TICK-747) measured this on a 400k-record type: generic **821 ms** (200,430 rows, 6.3 MB external sort), custom **17 ms**.

### Evidence after the change

Same harness as P7-07 (`TestFieldIndexPerf800k`, backfill phase, PR #760), at `d6797194`, 2026-11-19. 800k records, one 400k tenant. `postgres:16.10`, `shared_buffers=2GB`, `fsync=on`. Docker Desktop on WSL2, 16 vCPU, 24.5 GiB. The host was shared with about 30 other containers, so the timings are not idle-host figures.

| `FieldIndexBackfill`, alone, 60 batches of 1,000 | Batch p50 | p95 | Projected run |
| --- | --- | --- | --- |
| P7-07, as merged (generic keyset plan) | 1,042 ms | 1,852 ms | 1h03m |
| P7-07, fix candidate (`ALTER ROLE` by hand) | 430 ms | 965 ms | 1h00m |
| This change, from db-roles | **485 ms** | 870 ms | 54m |

One batch's statements at a mid-type cursor (EXPLAIN ANALYZE, rolled back):

| Step | Plan | Time |
| --- | --- | --- |
| (b) keyset | `records_pkey` range walk, 1,000 rows, 2,962 buffers | 18 ms |
| (c) DELETE side rows, field with 427k rows | `rfv_text_pkey`, `record_id = ANY`, 1,000 rows | 41 ms |
| (c) INSERT side rows | `records_pkey` + `record_field_coerce`, 2 FK triggers × 1,000 | 131 ms |

The keyset step is no longer the bottleneck. Most of the remaining batch time is the side-row write, the commit (about 6.8 MB of WAL per batch with `fsync=on`) and the activity round trip. **The target of a batch p50 under 400 ms is not met on this host.** TICK-992 re-scores D10 on an idle host after TICK-991 (pacer and per-batch count).

### Regression test

`TestTICK990KeysetStepKeepsCustomPlan` (`internal/fieldindex/backfill`, integration). It seeds 20k records of one type plus 200 single-record tenants, then runs the prepared keyset step 8 times on one `nava_worker` connection. It asserts that `EXPLAIN EXECUTE` is a `records_pkey` walk with no Sort. The negative control is the same run under `SET LOCAL plan_cache_mode = auto`. It must flip to the generic plan (`records_list_group_state` + Sort), or the test could not catch the regression. `TestP309PlanCacheModeFromDBRoles` pins the role setting.
