# TICK-1022 post-backfill statistics, raw (2026-11-20T12:39:32Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 4m54s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, work_mem=4096kB
- other containers at start: [nava-it-pgb-noprep-1001932580 nava-it-pgb-pooled-1001932580 nava-it-pg-1001932580 nava-t1022-pg-9775 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- large type pk: 1 3975, 2 3991, 3 192501, 4 199533

## Lists as a narrowing agent: single-client p50 / 8-client p95, ms

| State | rfv_num stats | pk=3 (48%) | pk=4 (50%) |
| --- | --- | --- | --- |
| vacuum | 400000 reltuples, 4546 pages, 100% all-visible, 4/3/1/2 @ 0.5038/0.47763333/0.0097/0.008866667 | 48.3 / - | 43.3 / - |
| vm90 | 400000 reltuples, 4546 pages, 90% all-visible, 4/3/2/1 @ 0.4953/0.48436666/0.0102/0.0101333335 | 41.7 / - | 41.5 / - |
| vm80 | 400000 reltuples, 4546 pages, 80% all-visible, 4/3/1/2 @ 0.5036333/0.47666666/0.009966667/0.009733333 | 102.9 / - | 91.7 / - |
| vm70 | 400000 reltuples, 4546 pages, 70% all-visible, 4/3/2/1 @ 0.4917/0.4882/0.0101666665/0.009933333 | 43.1 / - | 42.4 / - |
| vm50 | 400000 reltuples, 4546 pages, 50% all-visible, 4/3/2/1 @ 0.49693334/0.4832/0.010366667/0.0095 | 193.8 / - | 219.0 / - |
| vm25 | 400000 reltuples, 4546 pages, 25% all-visible, 4/3/1/2 @ 0.49663332/0.48283333/0.0104/0.0101333335 | 130.1 / - | 112.4 / - |
| churn5 | 400000 reltuples, 4771 pages, 1% all-visible, 4/3/1/2 @ 0.4949/0.48473334/0.010733333/0.009633333 | 116.4 / - | 131.8 / - |
| churn10 | 400000 reltuples, 4999 pages, 0% all-visible, 4/3/2/1 @ 0.5003333/0.47993332/0.010333333/0.0094 | 147.2 / - | 167.1 / - |
| churn20 | 400000 reltuples, 5451 pages, 0% all-visible, 4/3/1/2 @ 0.50016665/0.4788/0.0106/0.010433333 | 125.4 / - | 131.8 / - |
| rpc | 400000 reltuples, 4546 pages, 0% all-visible, 4/3/2/1 @ 0.49703333/0.48416665/0.0098/0.009 | 42.4 / - | 42.6 / - |
