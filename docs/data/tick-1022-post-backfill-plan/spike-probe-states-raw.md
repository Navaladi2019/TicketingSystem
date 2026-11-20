# TICK-1022 post-backfill statistics, raw (2026-11-20T18:40:28Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 3m53s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, work_mem=4096kB
- other containers at start: [nava-t1022-pg-55612 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- large type pk: 1 3975, 2 3991, 3 192501, 4 199533

## Lists as a narrowing agent: single-client p50 / 8-client p95, ms

| State | rfv_num stats | pk=3 (48%) | pk=4 (50%) | pk=1 (1%) |
| --- | --- | --- | --- | --- |
| none | -1 reltuples, 0 pages, 0% all-visible, no value stats | 4379.9 / 12883.8 | 4847.2 / 14137.0 | 214.4 / 751.4 |
| analyze | 400000 reltuples, 4546 pages, 0% all-visible, 4/3/1/2 @ 0.49896666/0.4799/0.010966667/0.0101666665 | 62.2 / 139.8 | 59.0 / 152.2 | 175.4 / 604.9 |
| churn5 | 400000 reltuples, 4771 pages, 1% all-visible, 4/3/1/2 @ 0.49963334/0.47923332/0.0111/0.010033334 | 77.2 / 172.2 | 75.0 / 173.4 | 201.7 / 910.0 |
| churn20 | 400000 reltuples, 5451 pages, 0% all-visible, 4/3/2/1 @ 0.4984/0.4826/0.009666666/0.009333333 | 86.2 / 245.8 | 71.0 / 148.1 | 180.2 / 583.2 |
| vacuum | 400000 reltuples, 4546 pages, 100% all-visible, 4/3/2/1 @ 0.50096667/0.47936666/0.01/0.009666666 | 70.1 / 211.9 | 69.3 / 227.5 | 175.4 / 568.1 |
