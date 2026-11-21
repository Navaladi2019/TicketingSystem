# TICK-1022 A2-11 spikes, raw (2026-11-21T04:10:37Z)

- seed: 800k records in 3m29s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, work_mem=4096kB
- nava_app: plan_cache_mode=force_custom_plan
- other containers at start: [nava-a2spike-pg-11050 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- fixtures in 4m27s: pk values 3:155850 4:156021 10:26593 20:21487 30:19999 31:20000 77:50; visible a6 24055, a50 200379
- pk=20 visible to a6: 494

## State none: rfv_bool 0% all-visible, rfv_num 0% all-visible, rfv_text 0% all-visible

| Spike | Cell | before: count ms (plan) | A2: pre-counts + guard + count ms (plan) | A2 - before | A2 page ms | 1-client p50 before / A2 | 8-client p95 A2 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| anti | 6%, value on 26k invisible records | 122.6 (hash) | 82.9 + 88.8 + 468.0 = 639.7 (probe, 1 pre) | +517.1 | 267.8 | 469.5 / 850.9 | 4852.7 |
| anti | 50%, value on 26k invisible records | 513.5 (hash) | 90.1 + 88.1 + 344.9 = 523.1 (hash, 1 pre) | +9.6 | 3433.8 | 4670.0 / 3715.7 | 15985.8 |
| anti | 6%, partly correlated (500 visible) | 92.8 (hash) | 109.3 + 106.3 + 434.6 = 650.2 (probe, 1 pre) | +557.4 | 212.3 | 912.4 / 1493.6 | 3964.2 |
| step | 6%, 19,999 invisible side rows | 65.7 (hash) | 72.9 + 0.0 + 87.1 = 160.0 (hash, 1 pre) | +94.3 | 335.1 | 696.4 / 898.4 | 1532.3 |
| step | 6%, 20,000 invisible side rows | 144.3 (hash) | 65.5 + 111.9 + 432.5 = 609.9 (probe, 1 pre) | +465.5 | 182.3 | 537.4 / 1672.9 | 3658.0 |
| ref | 6%, broad 50% value | 256.0 (hash) | 48.5 + 81.3 + 42.9 = 172.6 (probe, 1 pre) | -83.4 | 349.5 | 1186.9 / 969.8 | 1882.9 |
| ref | 50%, broad 50% value | 38.7 (join) | 55.2 + 95.4 + 40.2 = 190.8 (join, 1 pre) | +152.1 | 11.8 | 396.6 / 349.3 | 1253.2 |
| multi | 6%, three broad fields | 384.8 (hash) | 285.6 + 65.0 + 286.5 = 637.1 (probe, 3 pre) | +252.3 | 402.8 | 1874.9 / 1788.6 | 4703.3 |

S-A2-skew (none): pre-count of 77 as tenant A (50 side rows; tenant B 200k): Index Only Scan on rfv_num_value, 14.68 ms

## State vacuum: rfv_bool 100% all-visible, rfv_num 100% all-visible, rfv_text 100% all-visible

| Spike | Cell | before: count ms (plan) | A2: pre-counts + guard + count ms (plan) | A2 - before | A2 page ms | 1-client p50 before / A2 | 8-client p95 A2 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| anti | 6%, value on 26k invisible records | 135.3 (hash) | 37.1 + 72.4 + 367.0 = 476.5 (probe, 1 pre) | +341.1 | 163.7 | 400.7 / 742.9 | 3063.4 |
| anti | 50%, value on 26k invisible records | 317.6 (hash) | 12.8 + 35.8 + 156.3 = 204.9 (hash, 1 pre) | -112.8 | 2198.3 | 3048.3 / 3283.2 | 8732.9 |
| anti | 6%, partly correlated (500 visible) | 54.5 (hash) | 9.8 + 24.7 + 169.7 = 204.2 (probe, 1 pre) | +149.8 | 96.0 | 231.0 / 869.8 | 1995.8 |
| step | 6%, 19,999 invisible side rows | 28.9 (hash) | 9.8 + 0.0 + 19.7 = 29.5 (hash, 1 pre) | +0.6 | 58.8 | 171.0 / 297.8 | 408.4 |
| step | 6%, 20,000 invisible side rows | 25.3 (hash) | 15.4 + 38.0 + 201.4 = 254.7 (probe, 1 pre) | +229.5 | 69.4 | 153.7 / 446.4 | 761.8 |
| ref | 6%, broad 50% value | 33.8 (join) | 9.3 + 27.0 + 15.0 = 51.3 (probe, 1 pre) | +17.4 | 92.2 | 246.2 / 232.2 | 667.1 |
| ref | 50%, broad 50% value | 100.1 (hash) | 15.5 + 58.5 + 109.8 = 183.9 (hash, 1 pre) | +83.8 | 5.7 | 183.6 / 165.2 | 570.0 |
| multi | 6%, three broad fields | 137.2 (hash) | 43.2 + 32.5 + 179.3 = 255.0 (probe, 3 pre) | +117.8 | 165.9 | 579.0 / 328.9 | 1353.9 |

S-A2-skew (vacuum): pre-count of 77 as tenant A (50 side rows; tenant B 200k): Index Only Scan on rfv_num_value, 0.59 ms
