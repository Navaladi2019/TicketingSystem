# TICK-747 field-index measurements, raw (2026-11-20T21:14:10Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 3m44s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app plan_cache_mode: plan_cache_mode=force_custom_plan
- other containers at start: [nava-fiperf-pg-7408 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- transport floor (SELECT 1, 8 clients): p50 0.54 ms, p95 1.17 ms
- 8 fields flagged on all 400 ticket types; side rows copied in 7m31s; ANALYZE: rfv_text 216ms, rfv_num 104ms, rfv_ts 143ms, rfv_bool 14.66s
- side tables: rfv_bool 3200000 rows, 258 MB heap, 619 MB indexes; rfv_num 800000 rows, 71 MB heap, 152 MB indexes; rfv_text 1600000 rows, 129 MB heap, 307 MB indexes; rfv_ts 800000 rows, 71 MB heap, 151 MB indexes
- records: 1250 MB heap, 754 MB indexes

## Lists: p95 ms, 8 clients, 15s per cell, ACL (agent, narrowing) / no ACL (admin)

Control before, no data filter: ACL p95 135.7 (p50 74.6, n 1467) / no ACL p95 117.5 (p50 62.9, n 1733)

| Operator | 40% | 2% | 0.25% | 0.01% |
| --- | --- | --- | --- | --- |
| text eq | 145.1 / 134.8 | 223.1 / 247.1 | 138.0 / 134.3 | 78.7 / 114.1 |
| text in (2 values) | 149.7 / 130.3 | 222.9 / 269.6 | 151.3 / 187.2 | 101.0 / 138.3 |
| num lt | 232.7 / 247.1 | 739.1 / 371.2 | 184.3 / 173.3 | 74.8 / 149.6 |
| ts range | 168.2 / 256.1 | 288.6 / 332.8 | 193.0 / 159.4 | 80.7 / 130.5 |
| bool eq | 184.5 / 161.2 | 841.5 / 587.3 | 341.1 / 318.1 | 245.5 / 203.9 |

Control after: ACL p95 193.3 / no ACL p95 186.3

Transport floor p50 (SELECT 1, 8 clients) before each cell, ms: text eq 40% 1.5; text eq 2% 1.6; text eq 0.25% 1.7; text eq 0.01% 1.7; text in (2 values) 40% 1.5; text in (2 values) 2% 1.6; text in (2 values) 0.25% 1.7; text in (2 values) 0.01% 2.6; num lt 40% 3.0; num lt 2% 1.8; num lt 0.25% 2.4; num lt 0.01% 1.9; ts range 40% 1.6; ts range 2% 2.9; ts range 0.25% 1.9; ts range 0.01% 1.9; bool eq 40% 1.7; bool eq 2% 1.7; bool eq 0.25% 2.4; bool eq 0.01% 3.2

Targets (D13): p95 <= 150 ms in every cell, except 2% with ACL <= 350 ms.
**Lists: 23 miss(es):** text eq 2% no ACL p95 247.1 > 150; text in (2 values) 2% no ACL p95 269.6 > 150; text in (2 values) 0.25% ACL p95 151.3 > 150; text in (2 values) 0.25% no ACL p95 187.2 > 150; num lt 40% ACL p95 232.7 > 150; num lt 40% no ACL p95 247.1 > 150; num lt 2% ACL p95 739.1 > 350; num lt 2% no ACL p95 371.2 > 150; num lt 0.25% ACL p95 184.3 > 150; num lt 0.25% no ACL p95 173.3 > 150; ts range 40% ACL p95 168.2 > 150; ts range 40% no ACL p95 256.1 > 150; ts range 2% no ACL p95 332.8 > 150; ts range 0.25% ACL p95 193.0 > 150; ts range 0.25% no ACL p95 159.4 > 150; bool eq 40% ACL p95 184.5 > 150; bool eq 40% no ACL p95 161.2 > 150; bool eq 2% ACL p95 841.5 > 350; bool eq 2% no ACL p95 587.3 > 150; bool eq 0.25% ACL p95 341.1 > 150; bool eq 0.25% no ACL p95 318.1 > 150; bool eq 0.01% ACL p95 245.5 > 150; bool eq 0.01% no ACL p95 203.9 > 150

### Single-run EXPLAIN (ANALYZE) of the captured list statement

| Operator | Band | ACL | No ACL |
| --- | --- | --- | --- |
| text eq | 40% | planning 2.839 ms, execution 4.272 ms | planning 0.962 ms, execution 1.441 ms |
| text eq | 2% | planning 2.898 ms, execution 38.236 ms | planning 1.668 ms, execution 83.331 ms |
| text eq | 0.25% | planning 3.203 ms, execution 11.215 ms | planning 1.146 ms, execution 13.579 ms |
| text eq | 0.01% | planning 3.388 ms, execution 2.394 ms | planning 1.186 ms, execution 0.785 ms |
| text in (2 values) | 40% | planning 2.992 ms, execution 5.239 ms | planning 0.982 ms, execution 1.281 ms |
| text in (2 values) | 2% | planning 3.124 ms, execution 45.991 ms | planning 1.228 ms, execution 80.519 ms |
| text in (2 values) | 0.25% | planning 2.979 ms, execution 14.119 ms | planning 2.184 ms, execution 16.381 ms |
| text in (2 values) | 0.01% | planning 3.804 ms, execution 3.392 ms | planning 0.891 ms, execution 1.066 ms |
| num lt | 40% | planning 4.745 ms, execution 5.550 ms | planning 0.924 ms, execution 1.560 ms |
| num lt | 2% | planning 4.120 ms, execution 71.561 ms | planning 1.042 ms, execution 94.580 ms |
| num lt | 0.25% | planning 4.463 ms, execution 16.964 ms | planning 1.104 ms, execution 16.485 ms |
| num lt | 0.01% | planning 4.868 ms, execution 1.728 ms | planning 1.737 ms, execution 0.918 ms |
| ts range | 40% | planning 2.354 ms, execution 3.532 ms | planning 0.824 ms, execution 1.202 ms |
| ts range | 2% | planning 2.560 ms, execution 52.316 ms | planning 0.973 ms, execution 92.743 ms |
| ts range | 0.25% | planning 3.606 ms, execution 12.672 ms | planning 1.078 ms, execution 14.370 ms |
| ts range | 0.01% | planning 5.181 ms, execution 4.081 ms | planning 0.779 ms, execution 0.781 ms |
| bool eq | 40% | planning 14.315 ms, execution 6.243 ms | planning 2.916 ms, execution 2.386 ms |
| bool eq | 2% | planning 11.965 ms, execution 55.279 ms | planning 1.837 ms, execution 97.994 ms |
| bool eq | 0.25% | planning 21.748 ms, execution 22.117 ms | planning 2.514 ms, execution 17.484 ms |
| bool eq | 0.01% | planning 16.985 ms, execution 2.229 ms | planning 2.551 ms, execution 0.942 ms |
