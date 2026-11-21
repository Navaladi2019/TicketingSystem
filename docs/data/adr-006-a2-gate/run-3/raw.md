# TICK-1048 A2-8 (2), (2a), (3a), raw (2026-11-21T22:39:51Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 1m52s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app: plan_cache_mode=force_custom_plan
- CountBudget 200ms; other containers at start: [nava-a2gate-pg-2045]
- 1 fields flagged on the large tenant's ticket type only; side rows copied in 17s; ANALYZE: rfv_text 1ms, rfv_num 46ms, rfv_ts 2ms, rfv_bool 1ms
- side tables: rfv_bool 0 rows, 0 bytes heap, 16 kB indexes; rfv_num 400000 rows, 36 MB heap, 76 MB indexes; rfv_text 0 rows, 8192 bytes heap, 16 kB indexes; rfv_ts 0 rows, 0 bytes heap, 16 kB indexes
- records: 1042 MB heap, 755 MB indexes
- kp on the large type: 1: 3976, 2: 4081, 3: 191830, 4: 200113
- kp = 3 visible to agent-1 (desk rule groups): 11551

**(2) verdict: not run (NAVA_PERF_GATE_ATTEMPTS=0)**

## S-A2-cancel

- fixtures in 59s: pk values 3:155850 4:156021 10:26593 20:21487 30:19999 31:20000 77:50; visible a6 24055, a50 200379
- visibility map: rfv_bool 0% all-visible, rfv_num 41% all-visible, rfv_text 0% all-visible (the kp rows of rfv_num were vacuumed before the fixture; the pk rows were not)

Unbudgeted count time of the handler's count statement, as nava_app: 3 plain runs (the spike picks on these) and 3 EXPLAIN (ANALYZE, BUFFERS) runs:

| Candidate | count statement? | plain runs, ms | plain median ms | EXPLAIN runs, ms | EXPLAIN median ms |
| --- | --- | --- | --- | --- | --- |
| a6, pk = 3 (broad 50% value, 6% ACL) | yes | 54.4, 54.8, 58.9 | 54.8 | 62.8, 65.2, 67.1 | 65.2 |
| a6, three broad fields | yes | 63.5, 63.7, 64.9 | 63.7 | 72.5, 75.6, 79.1 | 75.6 |
| a6, pk = 20 (partly correlated, 500 visible) | yes | 13.0, 13.5, 13.7 | 13.5 | 14.6, 14.8, 16.4 | 14.8 |
| a50, pk = 3 (broad 50% value, 50% ACL) | yes | 66.5, 71.1, 76.5 | 71.1 | 77.7, 82.2, 88.1 | 82.2 |
| a6, pk = 10 (anti-correlated, 0 visible) | no: the first page holds every match (A2-4 shortcut) | | | | |
| a6, pk < 5 (every 3 and 4: 78% of the side rows) | yes | 4.2, 4.5, 4.6 | 4.5 | 3.6, 4.3, 4.7 | 4.3 |
| a50, pw = 1 (176k invisible side rows, 100 visible) | yes | 99.5, 115.8, 117.2 | 115.8 | 127.1, 127.6, 134.4 | 127.6 |
| a6, pw = 1 (176k invisible side rows, 100 visible) | yes | 45.9, 54.1, 59.5 | 54.1 | 57.9, 60.3, 63.3 | 60.3 |

**No candidate count exceeds CountBudget unbudgeted: S-A2-cancel cannot run.**
