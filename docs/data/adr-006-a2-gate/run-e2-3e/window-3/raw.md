# TICK-1048 / TICK-1083 A2-8 (2), (2a), (3a), raw (2026-11-23T02:47:31Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 2m39s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app: plan_cache_mode=force_custom_plan
- CountBudget 170ms; other containers at start: []
- parallel settings (server): max_parallel_workers = 8, max_parallel_workers_per_gather = 2, max_worker_processes = 8
