# TICK-747 field-index measurements, raw (2026-11-20T19:13:14Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 5m36s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app plan_cache_mode: plan_cache_mode=force_custom_plan
- other containers at start: [nava-fiperf-pg-1335 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- transport floor (SELECT 1, 8 clients): p50 6.43 ms, p95 12.19 ms
- 8 fields flagged on all 400 ticket types; side rows copied in 8m13s; ANALYZE: rfv_text 262ms, rfv_num 175ms, rfv_ts 162ms, rfv_bool 20.02s
- side tables: rfv_bool 3200000 rows, 258 MB heap, 619 MB indexes; rfv_num 800000 rows, 71 MB heap, 153 MB indexes; rfv_text 1600000 rows, 129 MB heap, 308 MB indexes; rfv_ts 800000 rows, 71 MB heap, 152 MB indexes
- records: 1250 MB heap, 755 MB indexes

## Lists: p95 ms, 8 clients, 15s per cell, ACL (agent, narrowing) / no ACL (admin)

Control before, no data filter: ACL p95 195.0 (p50 109.2, n 1003) / no ACL p95 193.0 (p50 104.4, n 1037)

| Operator | 40% | 2% | 0.25% | 0.01% |
| --- | --- | --- | --- | --- |
| text eq | 229.9 / 185.1 | 542.5 / 332.4 | 464.0 / 265.6 | 658.2 / 203.0 |
| text in (2 values) | 211.1 / 189.9 | 701.0 / 531.9 | 575.8 / 296.6 | 534.1 / 154.8 |
| num lt | 186.0 / 156.0 | 24113.8 / 350.1 | 3330.3 / 220.1 | 538.5 / 235.3 |
| ts range | 227.8 / 195.3 | 29462.8 / 348.9 | 4189.7 / 335.3 | 581.8 / 199.0 |
| bool eq | 191.3 / 163.3 | 712.5 / 522.1 | 526.3 / 351.7 | 462.6 / 217.5 |

Control after: ACL p95 280.8 / no ACL p95 192.9

Transport floor p50 (SELECT 1, 8 clients) before each cell, ms: text eq 40% 1.6; text eq 2% 1.8; text eq 0.25% 1.9; text eq 0.01% 2.3; text in (2 values) 40% 3.2; text in (2 values) 2% 3.8; text in (2 values) 0.25% 4.0; text in (2 values) 0.01% 4.1; num lt 40% 2.3; num lt 2% 2.3; num lt 0.25% 2.3; num lt 0.01% 2.9; ts range 40% 3.4; ts range 2% 2.4; ts range 0.25% 2.4; ts range 0.01% 2.9; bool eq 40% 2.8; bool eq 2% 2.2; bool eq 0.25% 4.8; bool eq 0.01% 4.1

Targets (D13): p95 <= 150 ms in every cell, except 2% with ACL <= 350 ms.
**Lists: 40 miss(es):** text eq 40% ACL p95 229.9 > 150; text eq 40% no ACL p95 185.1 > 150; text eq 2% ACL p95 542.5 > 350; text eq 2% no ACL p95 332.4 > 150; text eq 0.25% ACL p95 464.0 > 150; text eq 0.25% no ACL p95 265.6 > 150; text eq 0.01% ACL p95 658.2 > 150; text eq 0.01% no ACL p95 203.0 > 150; text in (2 values) 40% ACL p95 211.1 > 150; text in (2 values) 40% no ACL p95 189.9 > 150; text in (2 values) 2% ACL p95 701.0 > 350; text in (2 values) 2% no ACL p95 531.9 > 150; text in (2 values) 0.25% ACL p95 575.8 > 150; text in (2 values) 0.25% no ACL p95 296.6 > 150; text in (2 values) 0.01% ACL p95 534.1 > 150; text in (2 values) 0.01% no ACL p95 154.8 > 150; num lt 40% ACL p95 186.0 > 150; num lt 40% no ACL p95 156.0 > 150; num lt 2% ACL p95 24113.8 > 350; num lt 2% no ACL p95 350.1 > 150; num lt 0.25% ACL p95 3330.3 > 150; num lt 0.25% no ACL p95 220.1 > 150; num lt 0.01% ACL p95 538.5 > 150; num lt 0.01% no ACL p95 235.3 > 150; ts range 40% ACL p95 227.8 > 150; ts range 40% no ACL p95 195.3 > 150; ts range 2% ACL p95 29462.8 > 350; ts range 2% no ACL p95 348.9 > 150; ts range 0.25% ACL p95 4189.7 > 150; ts range 0.25% no ACL p95 335.3 > 150; ts range 0.01% ACL p95 581.8 > 150; ts range 0.01% no ACL p95 199.0 > 150; bool eq 40% ACL p95 191.3 > 150; bool eq 40% no ACL p95 163.3 > 150; bool eq 2% ACL p95 712.5 > 350; bool eq 2% no ACL p95 522.1 > 150; bool eq 0.25% ACL p95 526.3 > 150; bool eq 0.25% no ACL p95 351.7 > 150; bool eq 0.01% ACL p95 462.6 > 150; bool eq 0.01% no ACL p95 217.5 > 150

### Single-run EXPLAIN (ANALYZE) of the captured list statement

| Operator | Band | ACL | No ACL |
| --- | --- | --- | --- |
| text eq | 40% | planning 3.063 ms, execution 4.722 ms | planning 1.114 ms, execution 2.325 ms |
| text eq | 2% | planning 5.070 ms, execution 62.342 ms | planning 0.741 ms, execution 73.281 ms |
| text eq | 0.25% | planning 3.806 ms, execution 11.638 ms | planning 1.089 ms, execution 13.040 ms |
| text eq | 0.01% | planning 3.076 ms, execution 1.813 ms | planning 1.125 ms, execution 1.299 ms |
| text in (2 values) | 40% | planning 3.745 ms, execution 5.396 ms | planning 1.221 ms, execution 1.507 ms |
| text in (2 values) | 2% | planning 3.818 ms, execution 53.888 ms | planning 1.154 ms, execution 66.780 ms |
| text in (2 values) | 0.25% | planning 2.618 ms, execution 9.101 ms | planning 1.615 ms, execution 19.128 ms |
| text in (2 values) | 0.01% | planning 4.113 ms, execution 2.920 ms | planning 1.255 ms, execution 1.568 ms |
| num lt | 40% | planning 3.945 ms, execution 6.786 ms | planning 0.903 ms, execution 1.540 ms |
| num lt | 2% | planning 4.554 ms, execution 44.762 ms | planning 1.111 ms, execution 83.553 ms |
| num lt | 0.25% | planning 2.324 ms, execution 9.401 ms | planning 1.017 ms, execution 13.104 ms |
| num lt | 0.01% | planning 3.110 ms, execution 2.029 ms | planning 0.900 ms, execution 1.143 ms |
| ts range | 40% | planning 4.983 ms, execution 6.682 ms | planning 0.951 ms, execution 1.419 ms |
| ts range | 2% | planning 3.049 ms, execution 56.217 ms | planning 1.335 ms, execution 78.287 ms |
| ts range | 0.25% | planning 2.322 ms, execution 14.567 ms | planning 1.518 ms, execution 12.646 ms |
| ts range | 0.01% | planning 2.515 ms, execution 1.157 ms | planning 1.010 ms, execution 1.039 ms |
| bool eq | 40% | planning 8.431 ms, execution 3.681 ms | planning 2.918 ms, execution 2.066 ms |
| bool eq | 2% | planning 12.157 ms, execution 52.558 ms | planning 3.884 ms, execution 177.703 ms |
| bool eq | 0.25% | planning 11.318 ms, execution 11.795 ms | planning 1.748 ms, execution 12.759 ms |
| bool eq | 0.01% | planning 8.219 ms, execution 1.265 ms | planning 1.419 ms, execution 0.735 ms |
