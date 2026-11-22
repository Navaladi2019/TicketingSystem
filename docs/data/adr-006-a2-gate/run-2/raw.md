# TICK-1048 A2-8 (2), (2a), (3a), raw (2026-11-21T22:34:10Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 1m44s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app: plan_cache_mode=force_custom_plan
- CountBudget 200ms; other containers at start: [nava-a2gate-pg-1383]
- 1 fields flagged on the large tenant's ticket type only; side rows copied in 17s; ANALYZE: rfv_text 1ms, rfv_num 53ms, rfv_ts 1ms, rfv_bool 1ms
- side tables: rfv_bool 0 rows, 0 bytes heap, 16 kB indexes; rfv_num 400000 rows, 36 MB heap, 76 MB indexes; rfv_text 0 rows, 8192 bytes heap, 16 kB indexes; rfv_ts 0 rows, 0 bytes heap, 16 kB indexes
- records: 1042 MB heap, 755 MB indexes
- kp on the large type: 1: 3976, 2: 4081, 3: 191830, 4: 200113
- kp = 3 visible to agent-1 (desk rule groups): 11551

## Gate attempt 1

- before: floor p50 0.54 ms (pass, gate 3), control p95 47.4 ms (pass, gate 80.3); control n = 3282, p50 38.0 ms; other containers: [nava-a2gate-pg-1383]
- after: floor p50 1.11 ms (pass, gate 3), control p95 44.4 ms (pass, gate 80.3); control n = 3176, p50 36.7 ms; other containers: [nava-a2gate-pg-1383]
- **non-count p95 (count skipped), agents: 50.7 ms** (n = 2787, p50 42.1, p99 57.8); agent-1 only: 49.2 ms (n = 2875)
- with the budgeted count, agents: p95 75.9 ms (n = 1824, p50 64.8); reported only
- statements of one request (attempt 1: band (ii) kp=3, agents, count skipped): 1.70 ms BEGIN; SELECT set_config('app.tenant_id', 'faf09730-fbb6-4b07-b56a-65c | 1.71 ms SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_ver | 1.74 ms SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 | 1.77 ms SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity | 1.68 ms SELECT id, field_key, value_type, state, progress FROM record_index_fi | 11.35 ms SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x. | 1.93 ms commit
- statements of one request (attempt 1: band (ii) kp=3, agents, with the budgeted count): 2.32 ms BEGIN; SELECT set_config('app.tenant_id', 'faf09730-fbb6-4b07-b56a-65c | 1.74 ms SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_ver | 1.48 ms SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 | 1.74 ms SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity | 1.73 ms SELECT id, field_key, value_type, state, progress FROM record_index_fi | 10.20 ms SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x. | 1.76 ms SAVEPOINT list_count; SET LOCAL statement_timeout = '200' | 12.95 ms SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r. | 1.17 ms ROLLBACK TO SAVEPOINT list_count | 1.15 ms commit

**(2) verdict: PASS: 50.7 ms <= 94 ms (attempt 1)**

## S-A2-cancel

- fixtures in 52s: pk values 3:155850 4:156021 10:26593 20:21487 30:19999 31:20000 77:50; visible a6 24055, a50 200379
- visibility map: rfv_bool 0% all-visible, rfv_num 50% all-visible, rfv_text 0% all-visible (the kp rows of rfv_num were vacuumed before the fixture; the pk rows were not)

Unbudgeted count time of the handler's count statement, as nava_app: 3 plain runs (the spike picks on these) and 3 EXPLAIN (ANALYZE, BUFFERS) runs:

| Candidate | count statement? | plain runs, ms | plain median ms | EXPLAIN runs, ms | EXPLAIN median ms |
| --- | --- | --- | --- | --- | --- |
| a6, pk = 3 (broad 50% value, 6% ACL) | yes | 49.4, 56.0, 58.1 | 56.0 | 55.8, 60.2, 60.8 | 60.2 |
| a6, three broad fields | yes | 68.0, 70.2, 74.2 | 70.2 | 74.4, 81.0, 89.3 | 81.0 |
| a6, pk = 20 (partly correlated, 500 visible) | yes | 13.3, 14.1, 15.5 | 14.1 | 16.8, 17.2, 19.0 | 17.2 |
| a50, pk = 3 (broad 50% value, 50% ACL) | yes | 7.4, 7.7, 8.2 | 7.7 | 7.1, 7.3, 7.3 | 7.3 |
| a6, pk = 10 (anti-correlated, 0 visible) | no: the first page holds every match (A2-4 shortcut) | | | | |
| a6, pk < 5 (every 3 and 4: 78% of the side rows) | yes | 4.7, 5.2, 6.2 | 5.2 | 4.4, 5.0, 5.0 | 5.0 |

**No candidate count exceeds CountBudget unbudgeted: S-A2-cancel cannot run.**
