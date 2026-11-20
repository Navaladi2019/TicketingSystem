# TICK-1022 post-backfill statistics, raw (2026-11-20T21:02:31Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 4m3s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, work_mem=4096kB
- other containers at start: [nava-t1022-pg-5993 nava-s32-nava-core-1 nava-s32-nava-relay-1 nava-s32-nava-worker-1 nava-s32-nava-purge-1 nava-s32-nava-onboard-1 nava-s32-nava-groupsync-1 nava-s32-temporal-1 nava-s32-temporal-server-1 nava-s32-pgbouncer-1 nava-s32-keycloak-1 nava-s32-postgres-1 nava-s32-otel-collector-1 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- large type pk: 1 3975, 2 3991, 3 192501, 4 199533

## Lists as a narrowing agent: single-client p50 / 8-client p95, ms

| State | rfv_num stats | pk=3 (48%) | pk=4 (50%) | pk=1 (1%) |
| --- | --- | --- | --- | --- |
| analyze | 400000 reltuples, 4546 pages, 0% all-visible, 4/3/2/1 @ 0.4972/0.48276666/0.010333333/0.0097 | 81.5 / 262.2 | 70.0 / 178.4 | 121.9 / 317.7 |
| churn5 | 400000 reltuples, 4771 pages, 1% all-visible, 4/3/1/2 @ 0.49833333/0.4816/0.0101333335/0.009933333 | 149.7 / 249.7 | 59.9 / 193.0 | 104.4 / 303.3 |
| vacuum | 400000 reltuples, 4546 pages, 100% all-visible, 4/3/1/2 @ 0.4966/0.48486668/0.01/0.008533333 | 78.0 / 599.1 | 89.4 / 234.7 | 90.5 / 492.8 |
