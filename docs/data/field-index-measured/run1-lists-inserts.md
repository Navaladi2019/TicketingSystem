# TICK-747 field-index measurements, raw (2026-11-19T09:02:33Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 4m35s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app plan_cache_mode: plan_cache_mode=force_custom_plan
- other containers at start: [tick940-pg nava-fiperf-pg-15597 nava-s18-nava-worker-1 nava-s18-nava-relay-1 nava-s18-nava-core-1 nava-s18-temporal-1 nava-s18-nava-groupsync-1 nava-s18-nava-onboard-1 nava-s18-temporal-server-1 nava-s18-nava-purge-1 nava-s18-keycloak-1 nava-s18-otel-collector-1 nava-s18-pgbouncer-1 nava-s18-postgres-1 nava-nava-core-1 nava-nava-worker-1 nava-temporal-ui-1 nava-nava-relay-1 nava-temporal-1 nava-nava-purge-1 nava-nava-groupsync-1 nava-nava-onboard-1 nava-temporal-server-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-prometheus-1 nava-openobserve-1 nava-otel-collector-1]
- transport floor (SELECT 1, 8 clients): p50 3.35 ms, p95 7.88 ms
- list phase: 8 fields flagged on all 400 ticket types; side rows copied in 9m6s; ANALYZE: rfv_text 464ms, rfv_num 353ms, rfv_ts 204ms, rfv_bool 16.252s
- side tables: rfv_bool 3200000 rows, 258 MB heap, 616 MB indexes; rfv_num 800000 rows, 71 MB heap, 152 MB indexes; rfv_text 1600000 rows, 129 MB heap, 307 MB indexes; rfv_ts 800000 rows, 71 MB heap, 152 MB indexes
- records: 1250 MB heap, 754 MB indexes

## Lists: p95 ms, 8 clients, 15s per cell, ACL (agent, narrowing) / no ACL (admin)

Control before, no data filter: ACL p95 219.1 (p50 114.0, n 933) / no ACL p95 208.0 (p50 127.2, n 900)

| Operator | 40% | 2% | 0.25% | 0.01% |
| --- | --- | --- | --- | --- |
| text eq | 655.8 / 793.5 | 338.7 / 454.3 | 314.3 / 327.3 | 132.3 / 185.3 |
| text in (2 values) | 571.3 / 226.9 | 327.5 / 453.2 | 293.5 / 154.0 | 114.1 / 97.1 |
| num lt | 408.6 / 138.8 | 300.2 / 401.6 | 205.8 / 188.9 | 106.1 / 130.6 |
| ts range | 857.3 / 140.5 | 1481.0 / 1079.9 | 1893.0 / 4196.6 | 1914.8 / 13418.1 |
| bool eq | 302.1 / 438.9 | 453.8 / 411.2 | 338.2 / 451.5 | 194.9 / 102.8 |

Control after: ACL p95 123.8 / no ACL p95 104.9

Transport floor p50 (SELECT 1, 8 clients) before each cell, ms: text eq 40% 5.2; text eq 2% 4.4; text eq 0.25% 2.8; text eq 0.01% 6.3; text in (2 values) 40% 3.3; text in (2 values) 2% 4.3; text in (2 values) 0.25% 3.9; text in (2 values) 0.01% 2.2; num lt 40% 3.3; num lt 2% 2.7; num lt 0.25% 2.3; num lt 0.01% 4.5; ts range 40% 4.4; ts range 2% 3.9; ts range 0.25% 2.7; ts range 0.01% 3.7; bool eq 40% 10.8; bool eq 2% 3.3; bool eq 0.25% 2.2; bool eq 0.01% 3.4

### ts range: the captured statements as direct SQL, as compiled (two EXISTS) and with both bounds in one EXISTS

| Cell | as compiled p95 / p50 ms | one EXISTS p95 / p50 ms | page statement EXPLAIN execution, as compiled / one EXISTS |
| --- | --- | --- | --- |
| ts range 40%, ACL | 602.0 / 426.4 | 387.1 / 256.8 | 14.311 ms (planning 10.728) / 8.041 ms (planning 4.923) |
| ts range 40%, no ACL | 100.6 / 72.4 | 188.9 / 78.0 | 3.004 ms (planning 3.180) / 2.290 ms (planning 1.207) |
| ts range 2%, ACL | 1674.5 / 1008.9 | 472.3 / 296.5 | 444.393 ms (planning 17.792) / 144.638 ms (planning 9.557) |
| ts range 2%, no ACL | 1387.6 / 860.1 | 462.7 / 306.4 | 54.798 ms (planning 3.134) / 139.970 ms (planning 2.682) |
| ts range 0.25%, ACL | 1744.5 / 1321.8 | 316.9 / 151.8 | 423.620 ms (planning 8.677) / 17.317 ms (planning 3.787) |
| ts range 0.25%, no ACL | 4416.3 / 3095.6 | 409.9 / 236.5 | 203.144 ms (planning 2.352) / 18.513 ms (planning 1.217) |
| ts range 0.01%, ACL | 1755.6 / 1238.4 | 134.1 / 70.5 | 721.377 ms (planning 11.300) / 2.703 ms (planning 5.748) |
| ts range 0.01%, no ACL | 11946.3 / 10473.7 | 162.0 / 47.7 | 3172.202 ms (planning 1.827) / 1.024 ms (planning 0.901) |

Targets (D13): p95 <= 150 ms in every cell, except 2% with ACL <= 350 ms.
**Lists: 29 miss(es):** text eq 40% ACL p95 655.8 > 150; text eq 40% no ACL p95 793.5 > 150; text eq 2% no ACL p95 454.3 > 150; text eq 0.25% ACL p95 314.3 > 150; text eq 0.25% no ACL p95 327.3 > 150; text eq 0.01% no ACL p95 185.3 > 150; text in (2 values) 40% ACL p95 571.3 > 150; text in (2 values) 40% no ACL p95 226.9 > 150; text in (2 values) 2% no ACL p95 453.2 > 150; text in (2 values) 0.25% ACL p95 293.5 > 150; text in (2 values) 0.25% no ACL p95 154.0 > 150; num lt 40% ACL p95 408.6 > 150; num lt 2% no ACL p95 401.6 > 150; num lt 0.25% ACL p95 205.8 > 150; num lt 0.25% no ACL p95 188.9 > 150; ts range 40% ACL p95 857.3 > 150; ts range 2% ACL p95 1481.0 > 350; ts range 2% no ACL p95 1079.9 > 150; ts range 0.25% ACL p95 1893.0 > 150; ts range 0.25% no ACL p95 4196.6 > 150; ts range 0.01% ACL p95 1914.8 > 150; ts range 0.01% no ACL p95 13418.1 > 150; bool eq 40% ACL p95 302.1 > 150; bool eq 40% no ACL p95 438.9 > 150; bool eq 2% ACL p95 453.8 > 350; bool eq 2% no ACL p95 411.2 > 150; bool eq 0.25% ACL p95 338.2 > 150; bool eq 0.25% no ACL p95 451.5 > 150; bool eq 0.01% ACL p95 194.9 > 150

### Single-run EXPLAIN (ANALYZE) of the captured list statement

| Operator | Band | ACL | No ACL |
| --- | --- | --- | --- |
| text eq | 40% | planning 4.677 ms, execution 6.754 ms | planning 2.199 ms, execution 2.417 ms |
| text eq | 2% | planning 4.744 ms, execution 58.542 ms | planning 1.421 ms, execution 89.201 ms |
| text eq | 0.25% | planning 5.821 ms, execution 15.339 ms | planning 1.260 ms, execution 17.595 ms |
| text eq | 0.01% | planning 4.500 ms, execution 3.417 ms | planning 1.781 ms, execution 0.755 ms |
| text in (2 values) | 40% | planning 4.123 ms, execution 5.709 ms | planning 1.094 ms, execution 1.425 ms |
| text in (2 values) | 2% | planning 4.213 ms, execution 77.194 ms | planning 1.654 ms, execution 83.305 ms |
| text in (2 values) | 0.25% | planning 4.228 ms, execution 11.825 ms | planning 1.759 ms, execution 15.528 ms |
| text in (2 values) | 0.01% | planning 7.224 ms, execution 3.559 ms | planning 1.624 ms, execution 1.277 ms |
| num lt | 40% | planning 3.527 ms, execution 6.829 ms | planning 0.879 ms, execution 1.351 ms |
| num lt | 2% | planning 5.864 ms, execution 65.970 ms | planning 1.187 ms, execution 109.493 ms |
| num lt | 0.25% | planning 3.533 ms, execution 12.858 ms | planning 1.758 ms, execution 40.732 ms |
| num lt | 0.01% | planning 8.171 ms, execution 7.286 ms | planning 1.508 ms, execution 1.237 ms |
| ts range | 40% | planning 7.028 ms, execution 7.870 ms | planning 1.906 ms, execution 1.834 ms |
| ts range | 2% | planning 8.733 ms, execution 240.839 ms | planning 3.261 ms, execution 48.769 ms |
| ts range | 0.25% | planning 5.800 ms, execution 348.912 ms | planning 2.976 ms, execution 327.428 ms |
| ts range | 0.01% | planning 6.092 ms, execution 671.150 ms | planning 2.837 ms, execution 2849.063 ms |
| bool eq | 40% | planning 21.019 ms, execution 7.503 ms | planning 2.185 ms, execution 1.889 ms |
| bool eq | 2% | planning 18.483 ms, execution 87.778 ms | planning 2.212 ms, execution 101.708 ms |
| bool eq | 0.25% | planning 7.143 ms, execution 9.143 ms | planning 1.602 ms, execution 10.471 ms |
| bool eq | 0.01% | planning 29.413 ms, execution 8.000 ms | planning 3.153 ms, execution 1.202 ms |

## E1-5 guard (one current_setting() per fired row)

1,000,000 iterations, median of 5: with the check 983ms, constant condition 202ms: **0.781 µs per fired row**.

## Inserts: POST /api/v1/records/ticket, 8 clients, 1m0s per round, 5 flagged fields (tier, zone, score, opened, b2) vs 0

| Round | Arm | tps | p50 ms | p95 ms | storage probe p50 ms | errors |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 0 flagged (baseline) | 87.9 | 63.9 | 214.0 | 18.12 | 8 |
| 1 | 5 flagged | 115.0 | 61.2 | 113.9 | 6.46 | 0 |
| 2 | 0 flagged (baseline) | 66.4 | 87.3 | 287.9 | 7.09 | 13 |
| 2 | 5 flagged | 104.8 | 66.3 | 132.2 | 14.24 | 0 |
| 3 | 0 flagged (baseline) | 130.4 | 52.7 | 109.7 | 6.28 | 0 |
| 3 | 5 flagged | 79.9 | 80.7 | 208.0 | 6.83 | 0 |

Mean tps: baseline 94.9, 5 flagged 99.9: **105% of baseline** (floor 40%): meets.
- side rows written by the creates (large type): 36264

## Backfill: priority (text, 5 values) flagged late by publish v2 on the 400k type; FieldIndexBackfill on a Temporal dev server, nava_worker

Live mix: 8 clients through the handlers, alternating POST (every band field set; tier, zone, score and opened flagged) and PATCH of score, as nava_app.

