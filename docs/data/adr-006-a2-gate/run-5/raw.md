# TICK-1048 / TICK-1083 A2-8 (2), (2a), (3a), raw (2026-11-22T04:20:30Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 3m0s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app: plan_cache_mode=force_custom_plan
- CountBudget 200ms; other containers at start: []
- 1 fields flagged on the large tenant's ticket type only; side rows copied in 21s; ANALYZE: rfv_text 2ms, rfv_num 70ms, rfv_ts 1ms, rfv_bool 2ms
- side tables: rfv_bool 0 rows, 0 bytes heap, 16 kB indexes; rfv_num 400000 rows, 36 MB heap, 76 MB indexes; rfv_text 0 rows, 8192 bytes heap, 16 kB indexes; rfv_ts 0 rows, 0 bytes heap, 16 kB indexes
- records: 1042 MB heap, 755 MB indexes
- kp on the large type: 1: 3976, 2: 4081, 3: 191830, 4: 200113
- kp = 3 visible to agent-1 (desk rule groups): 11551

**(2) verdict: not run (NAVA_PERF_GATE_ATTEMPTS=0)**

## S-A2-cancel

- fixtures in 1m18s: pk values 3:155850 4:156021 10:26593 20:21487 30:19999 31:20000 77:50; visible a6 24055, a50 200379
- visibility map: rfv_bool 0% all-visible, rfv_num 27% all-visible, rfv_text 0% all-visible (the kp rows of rfv_num were vacuumed before the fixture; the pk rows were not)

- spike host gate, before: floor p50 0.60 ms (pass, gate 3), control p95 60.3 ms (pass, gate 80.3); control n = 2556, p50 48.0 ms; other containers: []

### Cell base: TICK-1048's cell: the slowest plain count under the server's settings

Unbudgeted count time of the handler's count statement, as nava_app: 3 plain runs (the cell picks on these) and 3 EXPLAIN (ANALYZE, BUFFERS) runs:

| Candidate | settings | count statement? | plan shape | plain runs, ms | plain median ms | EXPLAIN runs, ms | EXPLAIN median ms |
| --- | --- | --- | --- | --- | --- | --- | --- |
| a6, pk = 3 (broad 50% value, 6% ACL) | server settings | yes | any | 22.0, 22.8, 23.4 | 22.8 | 23.0, 29.9, 30.1 | 29.9 |
| a6, three broad fields | server settings | yes | any | 118.5, 120.4, 129.9 | 120.4 | 136.5, 138.7, 149.3 | 138.7 |
| a6, pk = 20 (partly correlated, 500 visible) | server settings | yes | any | 21.6, 22.3, 22.8 | 22.3 | 25.6, 25.7, 26.3 | 25.7 |
| a50, pk = 3 (broad 50% value, 50% ACL) | server settings | yes | any | 54.4, 55.5, 58.3 | 55.5 | 56.8, 56.8, 58.0 | 56.8 |
| a6, pk = 10 (anti-correlated, 0 visible) | server settings | no: the first page holds every match (A2-4 shortcut) | | | | | |
| a6, pk < 5 (every 3 and 4: 78% of the side rows) | server settings | yes | any | 13.5, 14.4, 15.0 | 14.4 | 15.4, 15.4, 17.4 | 15.4 |
| a50, pw = 1 (704k invisible side rows, 100 visible) | server settings | yes | any | 633.9, 648.7, 649.4 | 648.7 | 691.2, 709.4, 739.8 | 709.4 |
| a6, pw = 1 (704k invisible side rows, 100 visible) | server settings | yes | any | 69.3, 72.1, 73.2 | 72.1 | 72.3, 73.2, 73.5 | 73.2 |

Spike cell base: **a50, pw = 1 (704k invisible side rows, 100 visible)** (server settings), unbudgeted plain median 648.7 ms against a 200ms budget.
- transport floor before the spike: p50 1.62 ms; other containers: []
- before: temp_files 22, temp_bytes 239739952; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- after: temp_files 22, temp_bytes 239739952; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- after 60 s: temp_files 22, temp_bytes 239739952; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- temp files written during the spike (cumulative counter; it shows the spill, not a leak): 0 files, 0 bytes

| Run | n | status 200 + unknown + total null + full page | further statement after ROLLBACK TO | count phase p50 / p95 / p99 / max ms | p99 overshoot over 200ms | client latency p50 / p95 ms |
| --- | --- | --- | --- | --- | --- | --- |
| 1-client | 100 | 100 | 100 | 204.5 / 205.7 / 206.4 / 206.9 | **+6.4 ms** (limit +15) | 652.8 / 719.2 |
| 8-client | 200 | 200 | 200 | 208.6 / 211.3 / 213.6 / 214.4 | **+13.6 ms** (limit +15) | 1401.0 / 1577.4 |

Count statement alone (QueryRow to its error, client side) on the unknown answers, and the two round trips around it (SAVEPOINT + SET LOCAL Exec, ROLLBACK TO Exec):

| Run | unknown n | count statement p50 / p99 / max ms | statement overshoot p99 | two round trips p50 / p99 ms |
| --- | --- | --- | --- | --- |
| 1-client | 100 | 201.6 / 203.0 / 204.7 | +3.0 ms | 2.7 / 4.7 |
| 8-client | 200 | 203.1 / 206.4 / 207.7 | +6.4 ms | 5.3 / 8.5 |

Statements after ROLLBACK TO in one request: commit

**(3a) cell base verdict: PASS**

### Cell spill: a hash-join count with work_mem 64kB that spills to temp files before the budget cancels it

Unbudgeted count time of the handler's count statement, as nava_app: 3 plain runs (the cell picks on these) and 3 EXPLAIN (ANALYZE, BUFFERS) runs:

| Candidate | settings | count statement? | plan shape | plain runs, ms | plain median ms | EXPLAIN runs, ms | EXPLAIN median ms |
| --- | --- | --- | --- | --- | --- | --- | --- |
| a50, pk = 3 | max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | no hash join | 106.4, 117.2, 118.6 | 117.2 | 114.7, 129.5, 136.4 | 129.5 |
| a50, pw = 1 | max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | no hash join | 591.3, 645.2, 676.1 | 645.2 | 526.0, 617.6, 632.8 | 617.6 |
| a50, pk = 3 | enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | qualifies | 147.3, 148.0, 152.7 | 148.0 | 162.5, 164.8, 165.2 | 164.8 |
| a50, pw = 1 | enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | no temp I/O | 617.6, 636.6, 643.0 | 636.6 | 680.1, 680.1, 1324.7 | 680.1 |
| a50, pw = 1 | enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | qualifies | 473.8, 480.6, 548.7 | 480.6 | 568.1, 580.9, 720.5 | 580.9 |
| a50, pk < 5 | enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | qualifies | 273.7, 285.8, 290.4 | 285.8 | 351.4, 370.4, 373.6 | 370.4 |
| a50, pw = 1 and pk < 5 | enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | qualifies | 1295.7, 1355.9, 1412.9 | 1355.9 | 1383.7, 1557.7, 1562.3 | 1557.7 |
| a50, three broad fields | enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB | yes | qualifies | 416.5, 440.3, 460.3 | 440.3 | 484.2, 496.3, 527.4 | 496.3 |

Spike cell spill: **a50, pw = 1 and pk < 5** (enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB), unbudgeted plain median 1355.9 ms against a 200ms budget.
- transport floor before the spike: p50 1.15 ms; other containers: []
- before: temp_files 26559, temp_bytes 1943928804; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- after: temp_files 101766, temp_bytes 4020045360; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- after 60 s: temp_files 101766, temp_bytes 4020045360; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- temp files written during the spike (cumulative counter; it shows the spill, not a leak): 75207 files, 2076116556 bytes

| Run | n | status 200 + unknown + total null + full page | further statement after ROLLBACK TO | count phase p50 / p95 / p99 / max ms | p99 overshoot over 200ms | client latency p50 / p95 ms |
| --- | --- | --- | --- | --- | --- | --- |
| 1-client | 100 | 100 | 100 | 211.0 / 214.6 / 216.1 / 217.3 | **+16.1 ms** (limit +15) | 1059.6 / 1259.7 |
| 8-client | 200 | 200 | 200 | 222.4 / 235.1 / 245.9 / 251.9 | **+45.9 ms** (limit +15) | 2787.6 / 4056.0 |

Count statement alone (QueryRow to its error, client side) on the unknown answers, and the two round trips around it (SAVEPOINT + SET LOCAL Exec, ROLLBACK TO Exec):

| Run | unknown n | count statement p50 / p99 / max ms | statement overshoot p99 | two round trips p50 / p99 ms |
| --- | --- | --- | --- | --- |
| 1-client | 100 | 208.2 / 213.4 / 215.6 | +13.4 ms | 2.6 / 4.2 |
| 8-client | 200 | 215.0 / 232.1 / 242.8 | +32.1 ms | 6.9 / 23.0 |

The harness's SET LOCAL (`SET LOCAL enable_mergejoin = 'off'; SET LOCAL enable_nestloop = 'off'; SET LOCAL max_parallel_workers_per_gather = '0'; SET LOCAL work_mem = '64kB'`), sent after SAVEPOINT and left out of the count phase above: 300 of 300 requests, p50 2.56 / p99 9.36 ms; failed 0 times.

Statements after ROLLBACK TO in one request: commit

**(3a) cell spill verdict: FAIL**

### Cell parallel: a count whose plan has a Gather that launches workers

Unbudgeted count time of the handler's count statement, as nava_app: 3 plain runs (the cell picks on these) and 3 EXPLAIN (ANALYZE, BUFFERS) runs:

| Candidate | settings | count statement? | plan shape | plain runs, ms | plain median ms | EXPLAIN runs, ms | EXPLAIN median ms |
| --- | --- | --- | --- | --- | --- | --- | --- |
| a50, pk = 3 | server settings | yes | qualifies | 193.6, 233.8, 294.7 | 233.8 | 130.7, 158.0, 159.3 | 158.0 |
| a6, three broad fields | server settings | yes | qualifies | 546.6, 1321.4, 1370.9 | 1321.4 | 697.4, 1507.0, 1661.0 | 1507.0 |
| a50, pw = 1 | parallel_setup_cost = 0, parallel_tuple_cost = 0 | yes | no Gather | 2142.8, 2778.6, 6125.2 | 2778.6 | 4457.8, 7415.5, 7953.7 | 7415.5 |
| a50, pw = 1 | max_parallel_workers_per_gather = 1, parallel_setup_cost = 0, parallel_tuple_cost = 0 | yes | no Gather | 1020.8, 1978.9, 3487.5 | 1978.9 | 2376.0, 2658.3, 2848.6 | 2658.3 |
| a50, pw = 1 | enable_nestloop = off | yes | qualifies | 753.0, 768.3, 825.3 | 768.3 | 963.4, 1027.7, 1493.6 | 1027.7 |
| a50, pk < 5 | enable_nestloop = off | yes | qualifies | 275.9, 290.2, 296.1 | 290.2 | 292.2, 312.4, 334.1 | 312.4 |
| a50, pw = 1 and pk < 5 | enable_nestloop = off | yes | qualifies | 2223.0, 2562.8, 4652.9 | 2562.8 | 3770.4, 3836.5, 4234.9 | 3836.5 |

Spike cell parallel: **a50, pw = 1 and pk < 5** (enable_nestloop = off), unbudgeted plain median 2562.8 ms against a 200ms budget.
- transport floor before the spike: p50 1.95 ms; other containers: []
- before: temp_files 102699, temp_bytes 5444957328; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- after: temp_files 102699, temp_bytes 5444957328; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- after 60 s: temp_files 102699, temp_bytes 5444957328; pg_ls_tmpdir() 0 files (0 bytes); base/pgsql_tmp entries 0; parallel workers running 0
- temp files written during the spike (cumulative counter; it shows the spill, not a leak): 0 files, 0 bytes

| Run | n | status 200 + unknown + total null + full page | further statement after ROLLBACK TO | count phase p50 / p95 / p99 / max ms | p99 overshoot over 200ms | client latency p50 / p95 ms |
| --- | --- | --- | --- | --- | --- | --- |
| 1-client | 100 | 100 | 100 | 207.8 / 214.7 / 226.8 / 234.7 | **+26.8 ms** (limit +15) | 1930.5 / 3531.3 |
| 8-client | 200 | 200 | 200 | 229.0 / 296.2 / 395.6 / 421.2 | **+195.6 ms** (limit +15) | 4556.2 / 9786.1 |

Count statement alone (QueryRow to its error, client side) on the unknown answers, and the two round trips around it (SAVEPOINT + SET LOCAL Exec, ROLLBACK TO Exec):

| Run | unknown n | count statement p50 / p99 / max ms | statement overshoot p99 | two round trips p50 / p99 ms |
| --- | --- | --- | --- | --- |
| 1-client | 100 | 202.7 / 219.9 / 220.4 | +19.9 ms | 4.6 / 16.6 |
| 8-client | 200 | 209.3 / 278.1 / 370.8 | +78.1 ms | 18.2 / 82.7 |

The harness's SET LOCAL (`SET LOCAL enable_nestloop = 'off'`), sent after SAVEPOINT and left out of the count phase above: 300 of 300 requests, p50 3.73 / p99 56.23 ms; failed 0 times.

Statements after ROLLBACK TO in one request: commit

**(3a) cell parallel verdict: FAIL**

- spike host gate, after: floor p50 7.86 ms (FAIL, gate 3), control p95 300.1 ms (FAIL, gate 80.3); control n = 674, p50 165.4 ms; other containers: []

**(3a) verdict: FAIL or not scored** (base: PASS; spill: FAIL; parallel: FAIL; host gate failed after the spike)
