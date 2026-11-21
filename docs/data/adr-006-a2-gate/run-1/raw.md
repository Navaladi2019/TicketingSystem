# TICK-1048 A2-8 (2), (2a), (3a), raw (2026-11-21T12:46:17Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 5m55s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app: plan_cache_mode=force_custom_plan
- CountBudget 200ms; other containers at start: [nava-it-pgb-noprep-1150116051 nava-it-pgb-pooled-1150116051 nava-it-pg-1150116051 nava-a2gate-pg-10755 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- 1 fields flagged on the large tenant's ticket type only; side rows copied in 36s; ANALYZE: rfv_text 3ms, rfv_num 69ms, rfv_ts 4ms, rfv_bool 2ms
- side tables: rfv_bool 0 rows, 0 bytes heap, 16 kB indexes; rfv_num 400000 rows, 36 MB heap, 76 MB indexes; rfv_text 0 rows, 8192 bytes heap, 16 kB indexes; rfv_ts 0 rows, 0 bytes heap, 16 kB indexes
- records: 1042 MB heap, 756 MB indexes
- kp on the large type: 1: 3976, 2: 4081, 3: 191830, 4: 200113
- kp = 3 visible to agent-1 (desk rule groups): 11551

## Gate attempt 1

- before: floor p50 1.90 ms (pass, gate 3), control p95 172.9 ms (FAIL, gate 80.3); control n = 1183, p50 90.7 ms; other containers: [nava-a2gate-pg-10755 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- after: floor p50 14.13 ms (FAIL, gate 3), control p95 408.0 ms (FAIL, gate 80.3); control n = 503, p50 215.9 ms; other containers: [nava-a2gate-pg-10755 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- **non-count p95 (count skipped), agents: 198.6 ms** (n = 1014, p50 107.1, p99 265.2); agent-1 only: 210.6 ms (n = 1153)
- with the budgeted count, agents: p95 665.5 ms (n = 431, p50 225.9); reported only
- statements of one request (attempt 1: band (ii) kp=3, agents, count skipped): 4.80 ms BEGIN; SELECT set_config('app.tenant_id', 'd91dc445-cc03-4d75-9fa1-cd4 | 5.39 ms SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_ver | 2.16 ms SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 | 2.29 ms SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity | 3.59 ms SELECT id, field_key, value_type, state, progress FROM record_index_fi | 17.19 ms SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x. | 1.82 ms commit
- statements of one request (attempt 1: band (ii) kp=3, agents, with the budgeted count): 14.78 ms BEGIN; SELECT set_config('app.tenant_id', 'd91dc445-cc03-4d75-9fa1-cd4 | 15.08 ms SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_ver | 11.09 ms SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 | 12.79 ms SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity | 11.62 ms SELECT id, field_key, value_type, state, progress FROM record_index_fi | 21.23 ms SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x. | 14.34 ms SAVEPOINT list_count; SET LOCAL statement_timeout = '200' | 34.36 ms SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r. | 12.39 ms ROLLBACK TO SAVEPOINT list_count | 9.79 ms commit
- **attempt 1 inconclusive: a host gate failed; not fed into the formula**

## Gate attempt 2

- before: floor p50 5.39 ms (FAIL, gate 3), control p95 242.5 ms (FAIL, gate 80.3); control n = 833, p50 134.4 ms; other containers: [nava-a2gate-pg-10755 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- after: floor p50 4.47 ms (FAIL, gate 3), control p95 168.7 ms (FAIL, gate 80.3); control n = 1410, p50 71.5 ms; other containers: [nava-a2gate-pg-10755 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- **non-count p95 (count skipped), agents: 139.5 ms** (n = 1370, p50 80.3, p99 167.0); agent-1 only: 144.1 ms (n = 1333)
- with the budgeted count, agents: p95 325.3 ms (n = 579, p50 199.5); reported only
- statements of one request (attempt 2: band (ii) kp=3, agents, count skipped): 2.58 ms BEGIN; SELECT set_config('app.tenant_id', 'd91dc445-cc03-4d75-9fa1-cd4 | 6.12 ms SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_ver | 4.94 ms SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 | 9.05 ms SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity | 4.43 ms SELECT id, field_key, value_type, state, progress FROM record_index_fi | 36.74 ms SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x. | 3.34 ms commit
- statements of one request (attempt 2: band (ii) kp=3, agents, with the budgeted count): 20.28 ms BEGIN; SELECT set_config('app.tenant_id', 'd91dc445-cc03-4d75-9fa1-cd4 | 6.98 ms SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_ver | 6.22 ms SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 | 6.03 ms SELECT a.entity_definition_id, a.version, a.rules, a.masks FROM entity | 12.70 ms SELECT id, field_key, value_type, state, progress FROM record_index_fi | 29.99 ms SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x. | 3.65 ms SAVEPOINT list_count; SET LOCAL statement_timeout = '200' | 34.32 ms SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r. | 2.54 ms ROLLBACK TO SAVEPOINT list_count | 2.35 ms commit
- **attempt 2 inconclusive: a host gate failed; not fed into the formula**

**(2) verdict: inconclusive (attempt 2: host gate failed)**

## S-A2-cancel

- fixtures in 4m9s: pk values 3:155850 4:156021 10:26593 20:21487 30:19999 31:20000 77:50; visible a6 24055, a50 200379
- visibility map: rfv_bool 0% all-visible, rfv_num 50% all-visible, rfv_text 0% all-visible (the kp rows of rfv_num were vacuumed before the fixture; the pk rows were not)

Unbudgeted count time (EXPLAIN (ANALYZE, BUFFERS) of the handler's count statement, as nava_app, 3 runs):

| Candidate | count statement? | runs, ms | median ms |
| --- | --- | --- | --- |
| a6, pk = 3 (broad 50% value, 6% ACL) | yes | 521.8, 591.5, 608.9 | 591.5 |
| a6, three broad fields | yes | 439.1, 501.1, 510.4 | 501.1 |
| a6, pk = 20 (partly correlated, 500 visible) | yes | 59.0, 86.7, 112.1 | 86.7 |
| a50, pk = 3 (broad 50% value, 50% ACL) | yes | 30.9, 34.4, 42.8 | 34.4 |
| a6, pk = 10 (anti-correlated, 0 visible) | no: the first page holds every match (A2-4 shortcut) | | |

Spike cell: **a6, pk = 3 (broad 50% value, 6% ACL)**, unbudgeted median 591.5 ms against a 200ms budget.
- transport floor before the spike: p50 8.11 ms; other containers: [nava-a2gate-pg-10755 nava-nava-relay-1 nava-nava-worker-1 nava-nava-core-1 nava-temporal-1 nava-temporal-server-1 nava-nava-purge-1 nava-nava-onboard-1 nava-nava-groupsync-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-otel-collector-1]
- before: temp_files 17, temp_bytes 200074288; after: temp_files 17, temp_bytes 200074288; after 60 s: temp_files 17, temp_bytes 200074288

| Run | n | status 200 + unknown + total null + full page | further statement after ROLLBACK TO | count phase p50 / p95 / p99 / max ms | p99 overshoot over 200ms | client latency p50 / p95 ms |
| --- | --- | --- | --- | --- | --- | --- |
| 1-client | 100 | 67 | 100 | 234.0 / 291.9 / 299.3 / 465.1 | **+99.3 ms** (limit +15) | 289.5 / 470.9 |
| 8-client | 200 | 200 | 200 | 295.7 / 469.0 / 655.3 / 1192.6 | **+455.3 ms** (limit +15) | 502.7 / 835.2 |

Statements after ROLLBACK TO in one request: commit

**(3a) verdict: FAIL** on at least one rule above (temp readings above)
