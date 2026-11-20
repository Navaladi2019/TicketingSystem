# TICK-747 field-index measurements, raw (2026-11-20T15:14:43Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 3m28s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app plan_cache_mode: plan_cache_mode=force_custom_plan
- other containers at start: [nava-fiperf-pg-22606 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- transport floor (SELECT 1, 8 clients): p50 1.56 ms, p95 2.18 ms
- 8 fields flagged on all 400 ticket types; side rows copied in 8m6s; ANALYZE: rfv_text 327ms, rfv_num 190ms, rfv_ts 153ms, rfv_bool 20.628s
- side tables: rfv_bool 3200000 rows, 258 MB heap, 617 MB indexes; rfv_num 800000 rows, 71 MB heap, 152 MB indexes; rfv_text 1600000 rows, 129 MB heap, 309 MB indexes; rfv_ts 800000 rows, 71 MB heap, 152 MB indexes
- records: 1250 MB heap, 755 MB indexes
- nava_app random_page_cost = 1.1

## Lists: p95 ms, 8 clients, 15s per cell, ACL (agent, narrowing) / no ACL (admin)

Control before, no data filter: ACL p95 287.1 (p50 125.7, n 829) / no ACL p95 251.6 (p50 111.8, n 937)

| Operator | 40% | 2% | 0.25% | 0.01% |
| --- | --- | --- | --- | --- |
| text eq | 372.7 / 646.9 | 719.3 / 471.6 | 289.2 / 312.2 | 455.3 / 238.2 |
| text in (2 values) | 587.1 / 1168.7 | 722.9 / 789.3 | 431.5 / 561.1 | 241.0 / 335.7 |
| num lt | 620.8 / 478.3 | 354.3 / 467.4 | 370.0 / 309.5 | 161.8 / 289.3 |
| ts range | 490.7 / 250.1 | 394.2 / 454.4 | 297.3 / 312.0 | 182.8 / 354.8 |
| bool eq | 720.3 / 564.6 | 841.0 / 600.5 | 312.8 / 401.1 | 159.8 / 159.4 |

Control after: ACL p95 250.3 / no ACL p95 561.5

Transport floor p50 (SELECT 1, 8 clients) before each cell, ms: text eq 40% 2.2; text eq 2% 11.5; text eq 0.25% 1.8; text eq 0.01% 3.1; text in (2 values) 40% 5.7; text in (2 values) 2% 8.7; text in (2 values) 0.25% 5.8; text in (2 values) 0.01% 11.3; num lt 40% 2.3; num lt 2% 6.9; num lt 0.25% 4.8; num lt 0.01% 5.6; ts range 40% 6.2; ts range 2% 5.6; ts range 0.25% 4.3; ts range 0.01% 4.6; bool eq 40% 16.2; bool eq 2% 6.7; bool eq 0.25% 3.7; bool eq 0.01% 4.3

Targets (D13): p95 <= 150 ms in every cell, except 2% with ACL <= 350 ms.
**Lists: 40 miss(es):** text eq 40% ACL p95 372.7 > 150; text eq 40% no ACL p95 646.9 > 150; text eq 2% ACL p95 719.3 > 350; text eq 2% no ACL p95 471.6 > 150; text eq 0.25% ACL p95 289.2 > 150; text eq 0.25% no ACL p95 312.2 > 150; text eq 0.01% ACL p95 455.3 > 150; text eq 0.01% no ACL p95 238.2 > 150; text in (2 values) 40% ACL p95 587.1 > 150; text in (2 values) 40% no ACL p95 1168.7 > 150; text in (2 values) 2% ACL p95 722.9 > 350; text in (2 values) 2% no ACL p95 789.3 > 150; text in (2 values) 0.25% ACL p95 431.5 > 150; text in (2 values) 0.25% no ACL p95 561.1 > 150; text in (2 values) 0.01% ACL p95 241.0 > 150; text in (2 values) 0.01% no ACL p95 335.7 > 150; num lt 40% ACL p95 620.8 > 150; num lt 40% no ACL p95 478.3 > 150; num lt 2% ACL p95 354.3 > 350; num lt 2% no ACL p95 467.4 > 150; num lt 0.25% ACL p95 370.0 > 150; num lt 0.25% no ACL p95 309.5 > 150; num lt 0.01% ACL p95 161.8 > 150; num lt 0.01% no ACL p95 289.3 > 150; ts range 40% ACL p95 490.7 > 150; ts range 40% no ACL p95 250.1 > 150; ts range 2% ACL p95 394.2 > 350; ts range 2% no ACL p95 454.4 > 150; ts range 0.25% ACL p95 297.3 > 150; ts range 0.25% no ACL p95 312.0 > 150; ts range 0.01% ACL p95 182.8 > 150; ts range 0.01% no ACL p95 354.8 > 150; bool eq 40% ACL p95 720.3 > 150; bool eq 40% no ACL p95 564.6 > 150; bool eq 2% ACL p95 841.0 > 350; bool eq 2% no ACL p95 600.5 > 150; bool eq 0.25% ACL p95 312.8 > 150; bool eq 0.25% no ACL p95 401.1 > 150; bool eq 0.01% ACL p95 159.8 > 150; bool eq 0.01% no ACL p95 159.4 > 150

### Single-run EXPLAIN (ANALYZE) of the captured list statement

| Operator | Band | ACL | No ACL |
| --- | --- | --- | --- |
| text eq | 40% | planning 5.414 ms, execution 8.994 ms | planning 1.718 ms, execution 2.393 ms |
| text eq | 2% | planning 3.083 ms, execution 61.042 ms | planning 1.014 ms, execution 124.273 ms |
| text eq | 0.25% | planning 4.785 ms, execution 15.413 ms | planning 1.678 ms, execution 16.747 ms |
| text eq | 0.01% | planning 2.825 ms, execution 2.316 ms | planning 1.474 ms, execution 1.129 ms |
| text in (2 values) | 40% | planning 2.801 ms, execution 5.395 ms | planning 1.623 ms, execution 1.933 ms |
| text in (2 values) | 2% | planning 2.450 ms, execution 49.172 ms | planning 1.719 ms, execution 162.568 ms |
| text in (2 values) | 0.25% | planning 4.928 ms, execution 10.752 ms | planning 1.544 ms, execution 15.991 ms |
| text in (2 values) | 0.01% | planning 2.810 ms, execution 3.365 ms | planning 1.662 ms, execution 1.457 ms |
| num lt | 40% | planning 4.851 ms, execution 7.138 ms | planning 1.522 ms, execution 2.371 ms |
| num lt | 2% | planning 1.917 ms, execution 49.834 ms | planning 0.804 ms, execution 77.155 ms |
| num lt | 0.25% | planning 2.703 ms, execution 11.074 ms | planning 1.447 ms, execution 14.680 ms |
| num lt | 0.01% | planning 4.834 ms, execution 2.703 ms | planning 1.553 ms, execution 1.418 ms |
| ts range | 40% | planning 4.515 ms, execution 6.765 ms | planning 1.537 ms, execution 2.113 ms |
| ts range | 2% | planning 3.976 ms, execution 83.356 ms | planning 1.415 ms, execution 104.738 ms |
| ts range | 0.25% | planning 2.844 ms, execution 11.382 ms | planning 1.548 ms, execution 16.795 ms |
| ts range | 0.01% | planning 5.296 ms, execution 2.538 ms | planning 0.674 ms, execution 0.732 ms |
| bool eq | 40% | planning 16.058 ms, execution 7.116 ms | planning 1.961 ms, execution 2.333 ms |
| bool eq | 2% | planning 14.629 ms, execution 52.696 ms | planning 2.197 ms, execution 73.814 ms |
| bool eq | 0.25% | planning 11.491 ms, execution 11.415 ms | planning 1.248 ms, execution 7.556 ms |
| bool eq | 0.01% | planning 12.468 ms, execution 1.553 ms | planning 3.025 ms, execution 1.163 ms |
