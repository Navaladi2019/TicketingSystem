# TICK-1048 / TICK-1083 A2-8 (2), (2a), (3a), raw (2026-11-22T17:38:49Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 2m16s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app: plan_cache_mode=force_custom_plan
- CountBudget 170ms; other containers at start: []
- parallel settings (server): max_parallel_workers = 8, max_parallel_workers_per_gather = 2, max_worker_processes = 8
- 1 fields flagged on the large tenant's ticket type only; side rows copied in 16s; ANALYZE: rfv_text 2ms, rfv_num 62ms, rfv_ts 3ms, rfv_bool 2ms
- side tables: rfv_bool 0 rows, 0 bytes heap, 16 kB indexes; rfv_num 400000 rows, 36 MB heap, 76 MB indexes; rfv_text 0 rows, 8192 bytes heap, 16 kB indexes; rfv_ts 0 rows, 0 bytes heap, 16 kB indexes
- records: 1042 MB heap, 755 MB indexes
- kp on the large type: 1: 3976, 2: 4081, 3: 191830, 4: 200113
- kp = 3 visible to agent-1 (desk rule groups): 11551

**(2) verdict: not run (NAVA_PERF_GATE_ATTEMPTS=0)**

## (3e) serial-count check (ADR-006-E2 E2-4)

- arm A: shipped: CountBudget 170ms, workers off in the savepoint Exec; arm B: WithCountArm(140 ms, true): server settings. Both on e.pool (perfTracer only): no gateCountTracer, no forced GUC.
- 8 clients, blocks of 15s, order A, B, B, A per cell; each cell is its own ABBA (band (ii) before the S-A2-cancel fixture, a50 after it). The (2a) host gate runs before and after each block; adjacent blocks share the reading between them. A block with a failed gate is rerun, up to 2 reruns.
- band (ii) is TestA2Gate's: `filter[kp]=3` (48% of the type) round robin over P3-36's 200 agents (about 6% of the type each). The kit's Alice sees about 5% through one group; this ACL is the harness's documented stand-in (docs/data/adr-006-a2-gate.md).

### (3e) cell band (ii) kp = 3, agents: ABBA

- band (ii) kp = 3, agents block 1 (arm A), try 1: before: floor p50 1.67 ms (pass, gate 3), control p95 131.8 ms (FAIL, gate 80.3); control n = 1482, p50 73.8 ms; other containers: []
- band (ii) kp = 3, agents block 1 (arm A), try 1: after: floor p50 1.97 ms (pass, gate 3), control p95 125.4 ms (FAIL, gate 80.3); control n = 1583, p50 69.6 ms; other containers: []
- band (ii) kp = 3, agents block 1 (arm A), try 1: n 993, unknown 0, peak parallel workers 0 (296 samples); gates FAIL
- band (ii) kp = 3, agents block 1 (arm A), try 2: before: floor p50 1.35 ms (pass, gate 3), control p95 117.4 ms (FAIL, gate 80.3); control n = 1663, p50 66.6 ms; other containers: []
- band (ii) kp = 3, agents block 1 (arm A), try 2: after: floor p50 2.26 ms (pass, gate 3), control p95 110.2 ms (FAIL, gate 80.3); control n = 1712, p50 65.1 ms; other containers: []
- band (ii) kp = 3, agents block 1 (arm A), try 2: n 1223, unknown 0, peak parallel workers 0 (301 samples); gates FAIL
- band (ii) kp = 3, agents block 1 (arm A), try 3: before: floor p50 1.64 ms (pass, gate 3), control p95 136.6 ms (FAIL, gate 80.3); control n = 1485, p50 77.4 ms; other containers: []
- band (ii) kp = 3, agents block 1 (arm A), try 3: after: floor p50 1.87 ms (pass, gate 3), control p95 139.4 ms (FAIL, gate 80.3); control n = 1427, p50 77.3 ms; other containers: []
- band (ii) kp = 3, agents block 1 (arm A), try 3: n 1243, unknown 0, peak parallel workers 0 (301 samples); gates FAIL
- **band (ii) kp = 3, agents: block 1 (arm A) failed its host gates on 3 tries; the cell is inconclusive for host reasons**

## S-A2-cancel

- fixtures in 1m42s: pk values 3:155850 4:156021 10:26593 20:21487 30:19999 31:20000 77:50; visible a6 24055, a50 200379
- visibility map: rfv_bool 0% all-visible, rfv_num 27% all-visible, rfv_text 0% all-visible (the kp rows of rfv_num were vacuumed before the fixture; the pk rows were not)

### (3e) cell a50, pk = 3: ABBA

- a50, pk = 3 block 1 (arm A), try 1: before: floor p50 16.62 ms (FAIL, gate 3), control p95 744.3 ms (FAIL, gate 80.3); control n = 435, p50 174.3 ms; other containers: []
- a50, pk = 3 block 1 (arm A), try 1: after: floor p50 3.25 ms (FAIL, gate 3), control p95 167.6 ms (FAIL, gate 80.3); control n = 1233, p50 88.8 ms; other containers: []
- a50, pk = 3 block 1 (arm A), try 1: n 446, unknown 200, peak parallel workers 0 (301 samples); gates FAIL
- a50, pk = 3 block 1 (arm A), try 2: before: floor p50 2.36 ms (pass, gate 3), control p95 174.9 ms (FAIL, gate 80.3); control n = 1574, p50 61.2 ms; other containers: []
- a50, pk = 3 block 1 (arm A), try 2: after: floor p50 3.01 ms (FAIL, gate 3), control p95 97.1 ms (FAIL, gate 80.3); control n = 1885, p50 57.8 ms; other containers: []
- a50, pk = 3 block 1 (arm A), try 2: n 492, unknown 481, peak parallel workers 0 (304 samples); gates FAIL
- a50, pk = 3 block 1 (arm A), try 3: before: floor p50 2.71 ms (pass, gate 3), control p95 108.7 ms (FAIL, gate 80.3); control n = 1723, p50 58.7 ms; other containers: []
- a50, pk = 3 block 1 (arm A), try 3: after: floor p50 1.27 ms (pass, gate 3), control p95 100.8 ms (FAIL, gate 80.3); control n = 1890, p50 59.2 ms; other containers: []
- a50, pk = 3 block 1 (arm A), try 3: n 509, unknown 254, peak parallel workers 0 (302 samples); gates FAIL
- **a50, pk = 3: block 1 (arm A) failed its host gates on 3 tries; the cell is inconclusive for host reasons**
- plan check, arm A: savepoint Exec `SAVEPOINT list_count; SET LOCAL statement_timeout = '170'; SET LOCAL max_parallel_workers_per_gather = 0`; Gather false, workers launched false (plans.md)
- plan check, arm B: savepoint Exec `SAVEPOINT list_count; SET LOCAL statement_timeout = '140'`; Gather true, workers launched true (plans.md)

## (3e) result

| Cell | Arm | n (200) | non-200 | unknown | unknown share | ran a count | count phase p95 / p99 ms | count statement p99 ms | whole request p50 / p95 ms | peak parallel workers |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| band (ii) kp = 3, agents | A (shipped: CountBudget 170ms, workers off in the savepoint Exec) | 0 | 0 | 0 | 0.00% | 0 | 0.0 / 0.0 | 0.0 | 0.0 / 0.0 | 0 |
| band (ii) kp = 3, agents | B (WithCountArm(140 ms, true): server settings) | 0 | 0 | 0 | 0.00% | 0 | 0.0 / 0.0 | 0.0 | 0.0 / 0.0 | 0 |
| a50, pk = 3 | A (shipped: CountBudget 170ms, workers off in the savepoint Exec) | 0 | 0 | 0 | 0.00% | 0 | 0.0 / 0.0 | 0.0 | 0.0 / 0.0 | 0 |
| a50, pk = 3 | B (WithCountArm(140 ms, true): server settings) | 0 | 0 | 0 | 0.00% | 0 | 0.0 / 0.0 | 0.0 | 0.0 / 0.0 | 0 |

- **(3e) cell band (ii) kp = 3, agents: INCONCLUSIVE (host): block 1 (arm A) failed its host gates on 3 tries**
- **(3e) cell a50, pk = 3: INCONCLUSIVE (host): block 1 (arm A) failed its host gates on 3 tries**

**(3e) INCONCLUSIVE** (band (ii) kp = 3, agents: host; a50, pk = 3: host)

(3a) leak check after (3e) (ADR-006-E2 E2-3: the verdict reads pg_ls_tmpdir() only):

- before the a50 cell: pg_ls_tmpdir() 0 files (0 bytes); spill context only (proves a spill happened, never a leak): temp_files 16, temp_bytes 193340464; reported, not gated: base/pgsql_tmp entries 0, parallel workers running 0
- (3a) leak reading (3e), after the run: pg_ls_tmpdir() 0 files (0 bytes); spill context only (proves a spill happened, never a leak): temp_files 1681, temp_bytes 4187708028; reported, not gated: base/pgsql_tmp entries 0, parallel workers running 0
- (3a) leak reading (3e), 60 s after the run: pg_ls_tmpdir() 0 files (0 bytes); spill context only (proves a spill happened, never a leak): temp_files 1681, temp_bytes 4187708028; reported, not gated: base/pgsql_tmp entries 0, parallel workers running 0
- temp_files rose by 1665 (3994367564 bytes) during the a50 cell: a spill happened, so the leak reading covered a spilling cell.

**(3a) leak verdict after (3e): PASS (no pgsql_tmp file after the run or 60 s later)**

**(3e) overall: (3e) INCONCLUSIVE**

(3a) S-A2-cancel cells: none selected (NAVA_PERF_SPIKE_CELLS)
