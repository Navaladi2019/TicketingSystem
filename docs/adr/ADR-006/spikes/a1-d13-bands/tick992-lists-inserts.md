# TICK-747 field-index measurements, raw (2026-11-19T20:02:00Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 3m33s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app plan_cache_mode: plan_cache_mode=force_custom_plan
- other containers at start: [nava-fiperf-pg-6033 nava-nava-core-1 nava-nava-worker-1 nava-temporal-ui-1 nava-nava-relay-1 nava-temporal-1 nava-nava-purge-1 nava-nava-groupsync-1 nava-nava-onboard-1 nava-temporal-server-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-prometheus-1 nava-openobserve-1 nava-otel-collector-1]
- transport floor (SELECT 1, 8 clients): p50 1.48 ms, p95 3.78 ms
- 8 fields flagged on all 400 ticket types; side rows copied in 5m47s; ANALYZE: rfv_text 199ms, rfv_num 109ms, rfv_ts 130ms, rfv_bool 8.503s
- side tables: rfv_bool 3200000 rows, 258 MB heap, 618 MB indexes; rfv_num 800000 rows, 71 MB heap, 152 MB indexes; rfv_text 1600000 rows, 129 MB heap, 307 MB indexes; rfv_ts 800000 rows, 71 MB heap, 152 MB indexes
- records: 1250 MB heap, 755 MB indexes

## Lists: p95 ms, 8 clients, 20s per cell, ACL (agent, narrowing) / no ACL (admin)

Control before, no data filter: ACL p95 92.7 (p50 53.1, n 2750) / no ACL p95 99.4 (p50 54.1, n 2715)

| Operator | 40% | 2% | 0.25% | 0.01% |
| --- | --- | --- | --- | --- |
| text eq | 176.1 / 117.8 | 317.6 / 247.6 | 114.2 / 115.0 | 66.7 / 85.4 |
| text in (2 values) | 327.6 / 100.1 | 223.9 / 267.6 | 160.9 / 152.8 | 84.8 / 93.9 |
| num lt | 290.3 / 121.6 | 238.3 / 236.6 | 109.1 / 117.8 | 60.7 / 88.6 |
| ts range | 280.9 / 110.1 | 223.1 / 260.2 | 129.1 / 110.6 | 47.9 / 83.7 |
| bool eq | 181.7 / 118.0 | 233.7 / 238.2 | 143.5 / 124.7 | 82.3 / 72.3 |

Control after: ACL p95 89.2 / no ACL p95 85.9

Transport floor p50 (SELECT 1, 8 clients) before each cell, ms: text eq 40% 1.8; text eq 2% 2.4; text eq 0.25% 1.2; text eq 0.01% 1.2; text in (2 values) 40% 1.2; text in (2 values) 2% 1.4; text in (2 values) 0.25% 1.8; text in (2 values) 0.01% 1.8; num lt 40% 1.9; num lt 2% 2.3; num lt 0.25% 1.2; num lt 0.01% 1.9; ts range 40% 1.8; ts range 2% 1.7; ts range 0.25% 1.2; ts range 0.01% 2.7; bool eq 40% 1.2; bool eq 2% 1.5; bool eq 0.25% 1.1; bool eq 0.01% 1.7

Targets (D13): p95 <= 150 ms in every cell, except 2% with ACL <= 350 ms.
**Lists: 12 miss(es):** text eq 40% ACL p95 176.1 > 150; text eq 2% no ACL p95 247.6 > 150; text in (2 values) 40% ACL p95 327.6 > 150; text in (2 values) 2% no ACL p95 267.6 > 150; text in (2 values) 0.25% ACL p95 160.9 > 150; text in (2 values) 0.25% no ACL p95 152.8 > 150; num lt 40% ACL p95 290.3 > 150; num lt 2% no ACL p95 236.6 > 150; ts range 40% ACL p95 280.9 > 150; ts range 2% no ACL p95 260.2 > 150; bool eq 40% ACL p95 181.7 > 150; bool eq 2% no ACL p95 238.2 > 150

### Single-run EXPLAIN (ANALYZE) of the captured list statement

| Operator | Band | ACL | No ACL |
| --- | --- | --- | --- |
| text eq | 40% | planning 3.632 ms, execution 5.446 ms | planning 1.477 ms, execution 2.027 ms |
| text eq | 2% | planning 4.531 ms, execution 72.374 ms | planning 0.957 ms, execution 67.080 ms |
| text eq | 0.25% | planning 3.359 ms, execution 9.574 ms | planning 1.661 ms, execution 13.084 ms |
| text eq | 0.01% | planning 3.837 ms, execution 3.734 ms | planning 0.762 ms, execution 0.777 ms |
| text in (2 values) | 40% | planning 2.872 ms, execution 5.131 ms | planning 1.155 ms, execution 1.383 ms |
| text in (2 values) | 2% | planning 2.679 ms, execution 56.192 ms | planning 1.025 ms, execution 68.622 ms |
| text in (2 values) | 0.25% | planning 3.064 ms, execution 10.301 ms | planning 1.718 ms, execution 16.818 ms |
| text in (2 values) | 0.01% | planning 3.373 ms, execution 2.923 ms | planning 0.954 ms, execution 0.914 ms |
| num lt | 40% | planning 2.525 ms, execution 3.863 ms | planning 1.456 ms, execution 1.631 ms |
| num lt | 2% | planning 2.266 ms, execution 49.241 ms | planning 0.817 ms, execution 67.042 ms |
| num lt | 0.25% | planning 2.522 ms, execution 9.002 ms | planning 1.318 ms, execution 13.402 ms |
| num lt | 0.01% | planning 2.905 ms, execution 1.379 ms | planning 0.872 ms, execution 0.943 ms |
| ts range | 40% | planning 2.989 ms, execution 4.538 ms | planning 0.835 ms, execution 1.630 ms |
| ts range | 2% | planning 2.602 ms, execution 57.656 ms | planning 0.816 ms, execution 72.030 ms |
| ts range | 0.25% | planning 3.377 ms, execution 10.847 ms | planning 0.964 ms, execution 12.177 ms |
| ts range | 0.01% | planning 3.388 ms, execution 1.204 ms | planning 0.559 ms, execution 0.625 ms |
| bool eq | 40% | planning 8.729 ms, execution 5.307 ms | planning 2.202 ms, execution 1.811 ms |
| bool eq | 2% | planning 15.435 ms, execution 51.740 ms | planning 1.549 ms, execution 58.995 ms |
| bool eq | 0.25% | planning 10.460 ms, execution 9.657 ms | planning 1.987 ms, execution 12.630 ms |
| bool eq | 0.01% | planning 8.492 ms, execution 1.465 ms | planning 2.970 ms, execution 1.124 ms |

## E1-5 guard (one current_setting() per fired row)

1,000,000 iterations, median of 5: with the check 259ms, constant condition 65ms: **0.193 µs per fired row**.

## Inserts: POST /api/v1/records/ticket, 8 clients, 1m0s per round, 5 flagged fields (tier, zone, score, opened, b2) vs 0

| Round | Arm | tps | p50 ms | p95 ms | storage probe p50 ms | errors |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 0 flagged (baseline) | 218.5 | 33.7 | 53.9 | 25.76 | 0 |
| 1 | 5 flagged | 165.6 | 43.6 | 75.5 | 4.77 | 0 |
| 2 | 0 flagged (baseline) | 201.2 | 35.6 | 62.7 | 4.77 | 0 |
| 2 | 5 flagged | 145.1 | 50.8 | 81.8 | 6.67 | 0 |
| 3 | 0 flagged (baseline) | 153.2 | 37.1 | 135.0 | 4.94 | 0 |
| 3 | 5 flagged | 162.4 | 46.1 | 69.8 | 5.64 | 0 |

Mean tps: baseline 190.9, 5 flagged 157.7: **83% of baseline** (floor 40%): meets.
- side rows written by the creates (large type): 57072
