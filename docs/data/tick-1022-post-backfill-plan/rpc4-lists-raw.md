# TICK-747 field-index measurements, raw (2026-11-20T14:43:35Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 6m5s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app plan_cache_mode: plan_cache_mode=force_custom_plan
- other containers at start: [nava-it-pgb-noprep-1420523722 nava-it-pgb-pooled-1420523722 nava-it-pg-1420523722 nava-fiperf-pg-13646 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- transport floor (SELECT 1, 8 clients): p50 7.78 ms, p95 17.17 ms
- 8 fields flagged on all 400 ticket types; side rows copied in 9m33s; ANALYZE: rfv_text 303ms, rfv_num 194ms, rfv_ts 139ms, rfv_bool 20.948s
- side tables: rfv_bool 3200000 rows, 258 MB heap, 619 MB indexes; rfv_num 800000 rows, 71 MB heap, 152 MB indexes; rfv_text 1600000 rows, 129 MB heap, 309 MB indexes; rfv_ts 800000 rows, 71 MB heap, 152 MB indexes
- records: 1250 MB heap, 755 MB indexes

## Lists: p95 ms, 8 clients, 15s per cell, ACL (agent, narrowing) / no ACL (admin)

Control before, no data filter: ACL p95 214.9 (p50 125.2, n 880) / no ACL p95 208.7 (p50 125.8, n 897)

| Operator | 40% | 2% | 0.25% | 0.01% |
| --- | --- | --- | --- | --- |
| text eq | 276.0 / 233.6 | 469.4 / 424.3 | 186.1 / 161.8 | 113.1 / 126.2 |
| text in (2 values) | 354.7 / 162.8 | 432.6 / 415.9 | 397.7 / 322.9 | 163.6 / 255.7 |
| num lt | 393.9 / 207.2 | 356.8 / 353.2 | 178.8 / 188.5 | 361.4 / 346.1 |
| ts range | 374.2 / 179.9 | 263.2 / 312.9 | 164.1 / 220.7 | 55.1 / 85.9 |
| bool eq | 291.9 / 240.5 | 382.2 / 388.0 | 184.3 / 158.1 | 96.4 / 71.9 |

Control after: ACL p95 88.7 / no ACL p95 74.2

Transport floor p50 (SELECT 1, 8 clients) before each cell, ms: text eq 40% 3.4; text eq 2% 4.0; text eq 0.25% 2.8; text eq 0.01% 2.2; text in (2 values) 40% 2.2; text in (2 values) 2% 2.1; text in (2 values) 0.25% 2.3; text in (2 values) 0.01% 4.8; num lt 40% 3.1; num lt 2% 1.7; num lt 0.25% 2.1; num lt 0.01% 2.7; ts range 40% 2.2; ts range 2% 3.5; ts range 0.25% 1.7; ts range 0.01% 1.7; bool eq 40% 1.3; bool eq 2% 2.2; bool eq 0.25% 2.1; bool eq 0.01% 1.2

Targets (D13): p95 <= 150 ms in every cell, except 2% with ACL <= 350 ms.
**Lists: 33 miss(es):** text eq 40% ACL p95 276.0 > 150; text eq 40% no ACL p95 233.6 > 150; text eq 2% ACL p95 469.4 > 350; text eq 2% no ACL p95 424.3 > 150; text eq 0.25% ACL p95 186.1 > 150; text eq 0.25% no ACL p95 161.8 > 150; text in (2 values) 40% ACL p95 354.7 > 150; text in (2 values) 40% no ACL p95 162.8 > 150; text in (2 values) 2% ACL p95 432.6 > 350; text in (2 values) 2% no ACL p95 415.9 > 150; text in (2 values) 0.25% ACL p95 397.7 > 150; text in (2 values) 0.25% no ACL p95 322.9 > 150; text in (2 values) 0.01% ACL p95 163.6 > 150; text in (2 values) 0.01% no ACL p95 255.7 > 150; num lt 40% ACL p95 393.9 > 150; num lt 40% no ACL p95 207.2 > 150; num lt 2% ACL p95 356.8 > 350; num lt 2% no ACL p95 353.2 > 150; num lt 0.25% ACL p95 178.8 > 150; num lt 0.25% no ACL p95 188.5 > 150; num lt 0.01% ACL p95 361.4 > 150; num lt 0.01% no ACL p95 346.1 > 150; ts range 40% ACL p95 374.2 > 150; ts range 40% no ACL p95 179.9 > 150; ts range 2% no ACL p95 312.9 > 150; ts range 0.25% ACL p95 164.1 > 150; ts range 0.25% no ACL p95 220.7 > 150; bool eq 40% ACL p95 291.9 > 150; bool eq 40% no ACL p95 240.5 > 150; bool eq 2% ACL p95 382.2 > 350; bool eq 2% no ACL p95 388.0 > 150; bool eq 0.25% ACL p95 184.3 > 150; bool eq 0.25% no ACL p95 158.1 > 150

### Single-run EXPLAIN (ANALYZE) of the captured list statement

| Operator | Band | ACL | No ACL |
| --- | --- | --- | --- |
| text eq | 40% | planning 3.346 ms, execution 5.714 ms | planning 1.184 ms, execution 2.085 ms |
| text eq | 2% | planning 6.033 ms, execution 84.767 ms | planning 0.926 ms, execution 75.126 ms |
| text eq | 0.25% | planning 3.907 ms, execution 11.109 ms | planning 1.480 ms, execution 18.436 ms |
| text eq | 0.01% | planning 3.355 ms, execution 2.114 ms | planning 1.476 ms, execution 0.683 ms |
| text in (2 values) | 40% | planning 3.071 ms, execution 5.472 ms | planning 0.872 ms, execution 1.415 ms |
| text in (2 values) | 2% | planning 3.554 ms, execution 68.491 ms | planning 1.644 ms, execution 66.482 ms |
| text in (2 values) | 0.25% | planning 4.954 ms, execution 10.346 ms | planning 0.856 ms, execution 11.636 ms |
| text in (2 values) | 0.01% | planning 3.242 ms, execution 2.189 ms | planning 1.109 ms, execution 0.989 ms |
| num lt | 40% | planning 2.680 ms, execution 4.383 ms | planning 1.620 ms, execution 1.095 ms |
| num lt | 2% | planning 3.253 ms, execution 75.258 ms | planning 0.959 ms, execution 87.557 ms |
| num lt | 0.25% | planning 2.418 ms, execution 13.108 ms | planning 0.995 ms, execution 14.520 ms |
| num lt | 0.01% | planning 1.841 ms, execution 1.098 ms | planning 0.817 ms, execution 0.907 ms |
| ts range | 40% | planning 3.389 ms, execution 7.520 ms | planning 1.805 ms, execution 2.424 ms |
| ts range | 2% | planning 2.829 ms, execution 57.542 ms | planning 0.937 ms, execution 81.883 ms |
| ts range | 0.25% | planning 3.391 ms, execution 11.100 ms | planning 1.497 ms, execution 19.085 ms |
| ts range | 0.01% | planning 5.381 ms, execution 1.789 ms | planning 1.531 ms, execution 1.247 ms |
| bool eq | 40% | planning 12.207 ms, execution 6.608 ms | planning 1.730 ms, execution 1.649 ms |
| bool eq | 2% | planning 9.501 ms, execution 75.460 ms | planning 1.776 ms, execution 91.366 ms |
| bool eq | 0.25% | planning 11.556 ms, execution 11.313 ms | planning 1.924 ms, execution 14.843 ms |
| bool eq | 0.01% | planning 10.392 ms, execution 1.799 ms | planning 1.811 ms, execution 0.693 ms |
