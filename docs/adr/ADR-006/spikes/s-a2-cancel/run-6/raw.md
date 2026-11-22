# TICK-1048 / TICK-1083 A2-8 (2), (2a), (3a), raw (2026-11-22T10:10:04Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 2m4s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app: plan_cache_mode=force_custom_plan
- CountBudget 200ms; other containers at start: []
- 1 fields flagged on the large tenant's ticket type only; side rows copied in 17s; ANALYZE: rfv_text 2ms, rfv_num 72ms, rfv_ts 2ms, rfv_bool 1ms
- side tables: rfv_bool 0 rows, 0 bytes heap, 16 kB indexes; rfv_num 400000 rows, 36 MB heap, 76 MB indexes; rfv_text 0 rows, 8192 bytes heap, 16 kB indexes; rfv_ts 0 rows, 0 bytes heap, 16 kB indexes
- records: 1042 MB heap, 756 MB indexes
- kp on the large type: 1: 3976, 2: 4081, 3: 191830, 4: 200113
- kp = 3 visible to agent-1 (desk rule groups): 11551

**(2) verdict: not run (NAVA_PERF_GATE_ATTEMPTS=0)**

## S-A2-cancel

- fixtures in 1m23s: pk values 3:155850 4:156021 10:26593 20:21487 30:19999 31:20000 77:50; visible a6 24055, a50 200379
- visibility map: rfv_bool 0% all-visible, rfv_num 27% all-visible, rfv_text 0% all-visible (the kp rows of rfv_num were vacuumed before the fixture; the pk rows were not)

### Cell spill: a hash-join count with work_mem 64kB that spills to temp files before the budget cancels it

- cell spill host gate, before: floor p50 0.63 ms (pass, gate 3), control p95 72.1 ms (pass, gate 80.3); control n = 2402, p50 49.1 ms; other containers: []

Unbudgeted count time of the handler's count statement, as nava_app: 3 plain runs (the cell picks on these) and 3 EXPLAIN (ANALYZE, BUFFERS) runs:

| Candidate | settings | count statement? | plan shape | plain runs, ms | plain median ms | EXPLAIN runs, ms | EXPLAIN median ms |
| --- | --- | --- | --- | --- | --- | --- | --- |
| a50, pk = 3 | max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | no hash join in branch 1 | 184.8, 191.8, 212.1 | 191.8 | 204.6, 233.0, 276.3 | 233.0 |
| a50, pw = 1 | max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | no hash join in branch 1 | 599.9, 629.7, 801.3 | 629.7 | 649.6, 778.1, 835.4 | 778.1 |
| a50, pk = 3 | enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | no hash join in branch 1 | 139.5, 141.8, 145.5 | 141.8 | 162.6, 162.8, 166.3 | 162.8 |
| a50, pw = 1 | enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | no hash join in branch 1 | 574.5, 590.5, 601.2 | 590.5 | 678.1, 685.3, 946.4 | 685.3 |
| a50, pw = 1 | enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | qualifies | 504.0, 542.7, 572.8 | 542.7 | 558.5, 698.2, 801.3 | 698.2 |
| a50, pk < 5 | enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | qualifies | 294.8, 312.3, 318.3 | 312.3 | 350.4, 352.6, 452.9 | 352.6 |
| a50, pw = 1 and pk < 5 | enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | qualifies | 1278.3, 1318.1, 1484.7 | 1318.1 | 1748.6, 1883.0, 2149.4 | 1883.0 |
| a50, three broad fields | enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | qualifies | 458.8, 496.9, 614.6 | 496.9 | 664.3, 667.1, 765.9 | 667.1 |

Spike cell spill: **a50, pw = 1 and pk < 5** (enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB), unbudgeted plain median 1318.1 ms against a 200ms budget.
- transport floor before the spike: p50 1.68 ms; other containers: []
- before: temp_files 26807, temp_bytes 1910759396; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- after: temp_files 105344, temp_bytes 4499359280; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- after 60 s: temp_files 105344, temp_bytes 4499359280; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- temp files written during the spike (cumulative counter; it shows the spill, not a leak): 78537 files, 2588599884 bytes

| Run | n | status 200 + unknown + total null + full page | further statement after ROLLBACK TO | count phase p50 / p95 / p99 / max ms | p99 overshoot over 200ms | client latency p50 / p95 ms |
| --- | --- | --- | --- | --- | --- | --- |
| 1-client | 100 | 100 | 100 | 210.5 / 214.9 / 219.0 / 222.9 | **+19.0 ms** (limit +15) | 1001.6 / 1253.2 |
| 8-client | 200 | 200 | 200 | 220.3 / 226.2 / 227.8 / 229.1 | **+27.8 ms** (limit +15) | 2096.8 / 2356.3 |

Count statement alone (QueryRow to its error, client side) on the unknown answers, and the two round trips around it (SAVEPOINT + SET LOCAL Exec, ROLLBACK TO Exec):

| Run | unknown n | count statement p50 / p99 / max ms | statement overshoot p99 | two round trips p50 / p99 ms |
| --- | --- | --- | --- | --- |
| 1-client | 100 | 207.6 / 214.1 / 214.3 | +14.1 ms | 2.8 / 8.1 |
| 8-client | 200 | 214.6 / 222.6 / 224.0 | +22.6 ms | 5.4 / 9.0 |

The harness's SET LOCAL (`SET LOCAL enable_mergejoin = 'off'; SET LOCAL enable_nestloop = 'off'; SET LOCAL max_parallel_workers_per_gather = '0'; SET LOCAL work_mem = '64kB'`), sent after SAVEPOINT and left out of the count phase above: 300 of 300 requests, p50 2.27 / p99 3.74 ms; failed 0 times.

Statements after ROLLBACK TO in one request: commit

- cell spill host gate, after: floor p50 1.65 ms (pass, gate 3), control p95 79.1 ms (pass, gate 80.3); control n = 2005, p50 56.8 ms; other containers: []

**(3a) cell spill verdict: FAIL**

### Cell parallel: a count whose plan has a Gather that launches workers

- cell parallel host gate, before: floor p50 2.21 ms (pass, gate 3), control p95 80.6 ms (FAIL, gate 80.3); control n = 1939, p50 59.4 ms; other containers: []

Unbudgeted count time of the handler's count statement, as nava_app: 3 plain runs (the cell picks on these) and 3 EXPLAIN (ANALYZE, BUFFERS) runs:

| Candidate | settings | count statement? | plan shape | plain runs, ms | plain median ms | EXPLAIN runs, ms | EXPLAIN median ms |
| --- | --- | --- | --- | --- | --- | --- | --- |
| a50, pk = 3 | server settings | yes | branch-1 Gather ends at 113 ms (EXPLAIN) | 90.2, 94.5, 96.3 | 94.5 | 101.9, 119.2, 129.2 | 119.2 |
| a6, three broad fields | server settings | yes | qualifies | 193.1, 208.6, 255.0 | 208.6 | 197.4, 215.4, 299.6 | 215.4 |
| a50, pw = 1 | parallel_setup_cost = 0, parallel_tuple_cost = 0 | yes | no Gather in branch 1 | 678.7, 719.9, 837.8 | 719.9 | 853.9, 872.4, 934.4 | 872.4 |
| a50, pw = 1 | max_parallel_workers_per_gather = 1, parallel_setup_cost = 0, parallel_tuple_cost = 0 | yes | no Gather in branch 1 | 660.8, 676.9, 703.5 | 676.9 | 691.5, 710.7, 826.3 | 710.7 |
| a50, pw = 1 | enable_nestloop = off | yes | qualifies | 575.2, 578.1, 583.0 | 578.1 | 696.6, 702.1, 738.1 | 702.1 |
| a50, pk < 5 | enable_nestloop = off | yes | qualifies | 219.4, 233.6, 263.2 | 233.6 | 239.0, 239.2, 283.9 | 239.2 |
| a50, pw = 1 and pk < 5 | enable_nestloop = off | yes | no Gather in branch 1 | 1513.1, 1773.2, 2087.5 | 1773.2 | 1806.1, 1933.9, 2209.6 | 1933.9 |

Spike cell parallel: **a50, pw = 1** (enable_nestloop = off), unbudgeted plain median 578.1 ms against a 200ms budget.
- transport floor before the spike: p50 1.68 ms; other containers: []
- before: temp_files 106297, temp_bytes 5951009936; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- after: temp_files 113649, temp_bytes 13897618576; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- after 60 s: temp_files 113649, temp_bytes 13897618576; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- temp files written during the spike (cumulative counter; it shows the spill, not a leak): 7352 files, 7946608640 bytes

| Run | n | status 200 + unknown + total null + full page | further statement after ROLLBACK TO | count phase p50 / p95 / p99 / max ms | p99 overshoot over 200ms | client latency p50 / p95 ms |
| --- | --- | --- | --- | --- | --- | --- |
| 1-client | 100 | 100 | 100 | 232.6 / 238.7 / 239.5 / 261.6 | **+39.5 ms** (limit +15) | 760.1 / 926.0 |
| 8-client | 200 | 200 | 200 | 234.6 / 252.5 / 271.2 / 274.2 | **+71.2 ms** (limit +15) | 1673.6 / 2024.8 |

Count statement alone (QueryRow to its error, client side) on the unknown answers, and the two round trips around it (SAVEPOINT + SET LOCAL Exec, ROLLBACK TO Exec):

| Run | unknown n | count statement p50 / p99 / max ms | statement overshoot p99 | two round trips p50 / p99 ms |
| --- | --- | --- | --- | --- |
| 1-client | 100 | 229.1 / 236.6 / 258.1 | +36.6 ms | 3.3 / 5.1 |
| 8-client | 200 | 226.9 / 260.7 / 267.5 | +60.7 ms | 6.9 / 24.7 |

The harness's SET LOCAL (`SET LOCAL enable_nestloop = 'off'`), sent after SAVEPOINT and left out of the count phase above: 300 of 300 requests, p50 2.36 / p99 9.41 ms; failed 0 times.

Statements after ROLLBACK TO in one request: commit

- cell parallel host gate, after: floor p50 1.26 ms (pass, gate 3), control p95 84.3 ms (FAIL, gate 80.3); control n = 2074, p50 53.8 ms; other containers: []

**(3a) cell parallel verdict: FAIL (INCONCLUSIVE: a host gate failed)**

**(3a) verdict: FAIL or not scored** (spill: FAIL; parallel: FAIL (INCONCLUSIVE: a host gate failed))
