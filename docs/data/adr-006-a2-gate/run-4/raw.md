# TICK-1048 A2-8 (2), (2a), (3a), raw (2026-11-21T22:43:38Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 1m43s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app: plan_cache_mode=force_custom_plan
- CountBudget 200ms; other containers at start: [nava-a2gate-pg-1019]
- 1 fields flagged on the large tenant's ticket type only; side rows copied in 16s; ANALYZE: rfv_text 1ms, rfv_num 49ms, rfv_ts 1ms, rfv_bool 1ms
- side tables: rfv_bool 0 rows, 0 bytes heap, 16 kB indexes; rfv_num 400000 rows, 36 MB heap, 76 MB indexes; rfv_text 0 rows, 8192 bytes heap, 16 kB indexes; rfv_ts 0 rows, 0 bytes heap, 16 kB indexes
- records: 1042 MB heap, 756 MB indexes
- kp on the large type: 1: 3976, 2: 4081, 3: 191830, 4: 200113
- kp = 3 visible to agent-1 (desk rule groups): 11551

**(2) verdict: not run (NAVA_PERF_GATE_ATTEMPTS=0)**

## S-A2-cancel

- fixtures in 1m0s: pk values 3:155850 4:156021 10:26593 20:21487 30:19999 31:20000 77:50; visible a6 24055, a50 200379
- visibility map: rfv_bool 0% all-visible, rfv_num 27% all-visible, rfv_text 0% all-visible (the kp rows of rfv_num were vacuumed before the fixture; the pk rows were not)

Unbudgeted count time of the handler's count statement, as nava_app: 3 plain runs (the spike picks on these) and 3 EXPLAIN (ANALYZE, BUFFERS) runs:

| Candidate | count statement? | plain runs, ms | plain median ms | EXPLAIN runs, ms | EXPLAIN median ms |
| --- | --- | --- | --- | --- | --- |
| a6, pk = 3 (broad 50% value, 6% ACL) | yes | 9.3, 9.9, 10.4 | 9.9 | 9.2, 10.3, 12.1 | 10.3 |
| a6, three broad fields | yes | 64.5, 66.0, 66.6 | 66.0 | 72.5, 75.5, 77.8 | 75.5 |
| a6, pk = 20 (partly correlated, 500 visible) | yes | 13.2, 13.2, 13.8 | 13.2 | 14.2, 15.5, 19.9 | 15.5 |
| a50, pk = 3 (broad 50% value, 50% ACL) | yes | 32.6, 34.4, 35.1 | 34.4 | 35.8, 35.9, 38.9 | 35.9 |
| a6, pk = 10 (anti-correlated, 0 visible) | no: the first page holds every match (A2-4 shortcut) | | | | |
| a6, pk < 5 (every 3 and 4: 78% of the side rows) | yes | 7.6, 8.1, 9.4 | 8.1 | 7.3, 7.5, 7.7 | 7.5 |
| a50, pw = 1 (704k invisible side rows, 100 visible) | yes | 442.8, 452.6, 462.1 | 452.6 | 459.1, 472.4, 474.3 | 472.4 |
| a6, pw = 1 (704k invisible side rows, 100 visible) | yes | 38.6, 41.3, 41.6 | 41.3 | 39.8, 44.1, 47.1 | 44.1 |

Spike cell: **a50, pw = 1 (704k invisible side rows, 100 visible)**, unbudgeted plain median 452.6 ms against a 200ms budget.
- transport floor before the spike: p50 1.08 ms; other containers: [nava-a2gate-pg-1019]
- before: temp_files 22, temp_bytes 239756336; after: temp_files 22, temp_bytes 239756336; after 60 s: temp_files 22, temp_bytes 239756336

| Run | n | status 200 + unknown + total null + full page | further statement after ROLLBACK TO | count phase p50 / p95 / p99 / max ms | p99 overshoot over 200ms | client latency p50 / p95 ms |
| --- | --- | --- | --- | --- | --- | --- |
| 1-client | 100 | 100 | 100 | 203.8 / 205.0 / 205.5 / 205.6 | **+5.5 ms** (limit +15) | 615.1 / 654.3 |
| 8-client | 200 | 200 | 200 | 207.7 / 209.2 / 211.0 / 212.6 | **+11.0 ms** (limit +15) | 1301.2 / 1392.9 |

Count statement alone (QueryRow to its error, client side) on the unknown answers, and the two round trips around it (SAVEPOINT + SET LOCAL Exec, ROLLBACK TO Exec):

| Run | unknown n | count statement p50 / p99 / max ms | statement overshoot p99 | two round trips p50 / p99 ms |
| --- | --- | --- | --- | --- |
| 1-client | 100 | 201.4 / 202.6 / 202.6 | +2.6 ms | 2.4 / 3.6 |
| 8-client | 200 | 202.7 / 205.0 / 207.0 | +5.0 ms | 4.9 / 7.7 |

Statements after ROLLBACK TO in one request: commit

**(3a) verdict: PASS** (temp readings above)
