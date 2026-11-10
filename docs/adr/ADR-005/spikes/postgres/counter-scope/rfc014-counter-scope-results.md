# RFC-014 (TICK-653): record_counters scope_key migration and purge, query-plan evidence

Host: Docker Desktop on WSL2 (kernel 5.15.167.4), 16 vCPU, 19.4 GiB, shared with other containers (not quiet).
PostgreSQL 16.10 (the repo's pinned image), shared_buffers=256MB. Plans run as nava_app / nava_purge under FORCE RLS.
Buffer counts are the host-independent figure. Timings are from a single run.

Run in order on an empty database: 01_schema -> 02_migrate -> 03_plans_small -> 04_load_scopes -> 05/06 plans -> 07_purge -> 08_purge_steady.

| Step | Rows | Result |
|---|---|---|
| M1 ADD COLUMN scope_key text NOT NULL DEFAULT '' + CHECK NOT VALID | 20,000 | 12.0 ms, filenode unchanged (no rewrite); VALIDATE 7.1 ms |
| M2 CREATE UNIQUE INDEX CONCURRENTLY (tenant_id, entity_definition_id, scope_key) | 20,000 | 50 ms |
| Interim: old 2-col ON CONFLICT and new 3-col ON CONFLICT | - | both succeed (4001, 4002) |
| Interim: non-empty scope_key insert | - | ERROR duplicate key on the old record_counters_pkey |
| M3 DROP pkey, ADD PRIMARY KEY USING INDEX | 20,000 | 1.6 ms |
| After M3: old 2-col ON CONFLICT | - | ERROR: no unique or exclusion constraint matching the ON CONFLICT specification |
| Upsert, existing scope row, big tenant | 1,432,000 | arbiter record_counters_pkey, 9 shared hits, exec 0.37 ms |
| Upsert, first create in a new scope (FK check) | 1,432,000 | 10 shared hits, FK trigger 0.24 ms |
| Upsert from another tenant's session | - | ERROR: new row violates row-level security policy |
| Purge, period parsed from scope_key text | 1,432,000 | seq scan, 45 ms first batch |
| Purge, period_end without index, empty last batch | 531,000 | seq scan, 210-232 ms, 16.4k buffers |
| Purge, period_end + partial index, batch of 1,000 (2,000 due) | 531,000 | 11.6 ms, 1,062 buffers |
| Purge, period_end + partial index, empty last batch | 531,000 | 0.63 ms, 32 buffers |
| Partial index size | 912,000 period rows | 6.5 MB |
| HOT updates with the partial index present | 20,000 upserts | 19,813 HOT (99.1 %) |
