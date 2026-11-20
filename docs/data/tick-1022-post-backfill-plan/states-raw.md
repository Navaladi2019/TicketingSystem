# TICK-1022 post-backfill statistics, raw (2026-11-20T11:53:23Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 4m31s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, work_mem=4096kB
- other containers at start: [nava-it-pgb-noprep-162601072 nava-it-pgb-pooled-162601072 nava-it-pg-162601072 nava-t1022-pg-1499 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- large type pk: 1 3975, 2 3991, 3 192501, 4 199533

## Lists as a narrowing agent: single-client p50 / 8-client p95, ms

| State | rfv_num stats | pk=3 (48%) | pk=4 (50%) | pk=1 (1%) |
| --- | --- | --- | --- | --- |
| none | -1 reltuples, 0 pages, 0% all-visible, no value stats | 4536.0 / 15450.6 | 9376.4 / 15342.6 | 448.5 / 695.3 |
| mid | 200000 reltuples, 2273 pages, 0% all-visible, 4/3/1/2 @ 0.49646667/0.4822/0.011066667/0.010266666 | 259.3 / 425.1 | 180.1 / 1639.3 | 108.8 / 384.5 |
| late | 364000 reltuples, 4137 pages, 0% all-visible, 4/3/1/2 @ 0.5014667/0.47686666/0.011/0.010666667 | 201.9 / 901.7 | 184.8 / 569.1 | 179.1 / 762.1 |
| analyze | 400000 reltuples, 4546 pages, 0% all-visible, 4/3/2/1 @ 0.5011/0.47866666/0.010466667/0.009766666 | 254.9 / 526.9 | 315.3 / 786.5 | 202.8 / 468.0 |
| vacuum | 400000 reltuples, 4546 pages, 100% all-visible, 4/3/2/1 @ 0.5024667/0.47816667/0.010033334/0.009333333 | 63.6 / 215.1 | 67.1 / 188.8 | 128.7 / 322.4 |
