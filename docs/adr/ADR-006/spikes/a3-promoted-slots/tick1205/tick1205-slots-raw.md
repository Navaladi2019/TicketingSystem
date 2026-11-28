# TICK-747 field-index measurements, raw (2026-11-28T04:54:37Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 3m19s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app plan_cache_mode: plan_cache_mode=force_custom_plan
- other containers at start: [nava-fiperf-pg-61858]
- transport floor (SELECT 1, 8 clients): p50 1.68 ms, p95 2.76 ms
- 8 fields flagged on all 400 ticket types; side rows copied in 5m35s; ANALYZE: rfv_text 187ms, rfv_num 110ms, rfv_ts 96ms, rfv_bool 8.08s
- side tables: rfv_bool 3200000 rows, 258 MB heap, 619 MB indexes; rfv_num 800000 rows, 71 MB heap, 152 MB indexes; rfv_text 1600000 rows, 129 MB heap, 308 MB indexes; rfv_ts 800000 rows, 71 MB heap, 152 MB indexes
- records: 1250 MB heap, 765 MB indexes

- breadth agents (agent-1's principal), records of the large type each sees through any ACL branch:
  - agent-1: 3 groups, 24666 records (6.2%)
  - a25-1: 13 groups, 104490 records (26.1%)
  - a50-1: 25 groups, 200402 records (50.1%)

## TICK-1205: promoted filter slots (ADR-006-A3 draft), 8 clients, lists 20s per cell, writes 30s per cell, work_mem 4MB

- autovacuum: autovacuum=on, autovacuum_vacuum_scale_factor=0.2, checkpoint_timeout=300, full_page_writes=on, max_wal_size=8192, wal_compression=off
- pk fixture (value: records, visible to a6 / a50): 3: 200000, 15819 / 133409; 4: 123921, 7869 / 62488; 10: 26593, 0 / 0; 20: 21487, 494 / 494; 30: 19999, 0 / 0; 40: 8000, 484 / 4011
- fixture (pk into data, VACUUM FULL, targets) in 2m9s; records: 1250 MB heap, 579 MB indexes, 10 indexes
- transport floor before (SELECT 1, 8 clients): p50 1.66 ms, p95 2.62 ms: host gate (p50 <= 3 ms) passes; other containers: [nava-fiperf-pg-61858]

### Item 3: the A3-2 migration under the live mix (8 clients, POST + PATCH score as nava_app)

| Statement | Wall | Locks on records held (sampled every 2 ms) | DDL waiting on a lock | Writers waiting on a lock: max / time | Live ops overlapping | their p95 ms | their max ms |
| --- | --- | --- | --- | --- | --- | --- | --- |
| ALTER TABLE records ADD COLUMN x 8 (one statement) | 30ms | AccessExclusiveLock (seen over 6ms) | 17ms | 6 / 30ms | 13 | 102.9 | 102.9 |
| CREATE STATISTICS (mcv) fx_num_1 | 10ms | none sampled (held < 2 ms) | 0s | 0 / 0s | 9 | 85.3 | 85.3 |
| CREATE STATISTICS (mcv) fx_num_2 | 11ms | ShareUpdateExclusiveLock (seen over 0s) | 0s | 0 / 0s | 11 | 82.8 | 82.8 |
| CREATE STATISTICS (mcv) fx_text_1 | 11ms | ShareUpdateExclusiveLock (seen over 0s) | 0s | 0 / 0s | 9 | 89.4 | 89.4 |
| CREATE STATISTICS (mcv) fx_text_2 | 10ms | ShareUpdateExclusiveLock (seen over 6ms) | 0s | 0 / 0s | 8 | 96.4 | 96.4 |
| CREATE STATISTICS (mcv) fx_ts_1 | 15ms | ShareUpdateExclusiveLock (seen over 6ms) | 0s | 0 / 0s | 11 | 94.8 | 94.8 |
| CREATE STATISTICS (mcv) fx_ts_2 | 11ms | ShareUpdateExclusiveLock (seen over 0s) | 0s | 0 / 0s | 9 | 99.4 | 99.4 |
| CREATE STATISTICS (mcv) fx_bool_1 | 9ms | ShareUpdateExclusiveLock (seen over 0s) | 0s | 0 / 0s | 8 | 100.6 | 100.6 |
| CREATE STATISTICS (mcv) fx_bool_2 | 11ms | ShareUpdateExclusiveLock (seen over 6ms) | 0s | 0 / 0s | 8 | 106.7 | 106.7 |
| CREATE TRIGGER spike_slot_sync (BEFORE INSERT OR UPDATE OF data) | 29ms | ShareRowExclusiveLock (seen over 0s) | 22ms | 3 / 20ms | 11 | 107.4 | 107.4 |
| CREATE INDEX CONCURRENTLY spike_fx_num_1_grp (all NULL) | 2.09s | ShareUpdateExclusiveLock (seen over 2.075s) | 50ms | 0 / 0s | 232 | 108.7 | 121.7 |
| CREATE INDEX CONCURRENTLY spike_fx_num_2_grp (all NULL) | 1.723s | ShareUpdateExclusiveLock (seen over 1.711s) | 54ms | 0 / 0s | 204 | 105.9 | 119.0 |
| CREATE INDEX CONCURRENTLY spike_fx_text_1_grp (all NULL) | 1.817s | ShareUpdateExclusiveLock (seen over 1.804s) | 53ms | 0 / 0s | 209 | 104.0 | 119.3 |
| CREATE INDEX CONCURRENTLY spike_fx_text_2_grp (all NULL) | 1.651s | ShareUpdateExclusiveLock (seen over 1.636s) | 40ms | 0 / 0s | 204 | 97.1 | 113.4 |
| CREATE INDEX CONCURRENTLY spike_fx_ts_1_grp (all NULL) | 1.528s | ShareUpdateExclusiveLock (seen over 1.514s) | 52ms | 0 / 0s | 197 | 93.8 | 98.3 |
| CREATE INDEX CONCURRENTLY spike_fx_ts_2_grp (all NULL) | 1.64s | ShareUpdateExclusiveLock (seen over 1.629s) | 70ms | 0 / 0s | 201 | 102.7 | 123.6 |
| CREATE INDEX CONCURRENTLY spike_fx_bool_1_grp (all NULL) | 1.859s | ShareUpdateExclusiveLock (seen over 1.842s) | 54ms | 0 / 0s | 203 | 123.7 | 130.9 |
| CREATE INDEX CONCURRENTLY spike_fx_bool_2_grp (all NULL) | 1.885s | ShareUpdateExclusiveLock (seen over 1.868s) | 63ms | 0 / 0s | 206 | 113.6 | 123.1 |
| ALTER TABLE records ADD COLUMN (one per-field column, arm (b) literally) | 74ms | AccessExclusiveLock (seen over 0s) | 66ms | 6 / 55ms | 15 | 150.2 | 150.2 |

| Live mix window | Length | ops/s | p50 ms | **p95 ms** | p99 ms | create p95 | patch p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| before the migration | 60 s | 119.2 | 70.4 | **104.3** | 118.7 | 58.2 | 109.6 |
| during the migration (first ALTER to last CIC) | 20 s | 116.4 | 70.7 | **107.3** | 119.7 | 59.1 | 113.7 |
| after | 60 s | 123.7 | 67.2 | **99.6** | 157.2 | 57.9 | 130.7 |
- live errors: map[]
- records after the migration: 1276 MB heap, 598 MB indexes; slot indexes: spike_fx_bool_1_grp 8192 bytes, spike_fx_bool_2_grp 8192 bytes, spike_fx_num_1_grp 8192 bytes, spike_fx_num_2_grp 8192 bytes, spike_fx_text_1_grp 8192 bytes, spike_fx_text_2_grp 8192 bytes, spike_fx_ts_1_grp 8192 bytes, spike_fx_ts_2_grp 8192 bytes

### A3-5 backfill: 1,000-record batches, UPDATE records SET slot = record_field_coerce(data -> key) (no version or updated_at change)

| Field -> slot | Load | Duration | Batches | Batch p50 / p95 / max ms | WAL | WAL B/record | FPI | records HOT / updates | Heap before -> after | Slot index after | n_dead_tup after |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| score -> fx_num_1 (451222 filled) | live mix, 500 ms pause | 11m11s | 416 | 1133 / 1606 / 1896 | 3.92 GB | 9335 | 410216 | 2846 / 452428 | 1291 MB -> 1524 MB | 51 MB | 79859 |
| tier -> fx_text_1 (451222 filled) | alone | 3m12s | 452 | 404 / 626 / 1031 | 2.23 GB | 5317 | 180101 | 0 / 451222 | 1524 MB -> 1690 MB | 50 MB | 327736 |
| opened -> fx_ts_1 (451222 filled) | alone | 3m4s | 452 | 379 / 616 / 904 | 1.46 GB | 3467 | 45556 | 0 / 451222 | 1690 MB -> 1690 MB | 50 MB | 255106 |
| zone -> fx_text_2 (451222 filled) | alone | 2m50s | 452 | 377 / 623 / 1107 | 2.31 GB | 5487 | 208093 | 0 / 451222 | 1690 MB -> 1704 MB | 50 MB | 167382 |
| b2 -> fx_bool_1 (451222 filled) | alone | 3m17s | 452 | 386 / 717 / 1310 | 2.31 GB | 5504 | 203046 | 0 / 451222 | 1704 MB -> 1738 MB | 51 MB | 59114 |
| pk -> fx_num_2 (400000 filled) | alone | 2m41s | 452 | 338 / 520 / 1142 | 1.63 GB | 3867 | 44488 | 12256 / 451222 | 1738 MB -> 1738 MB | 43 MB | 94364 |

Live mix around the score backfill (D10: p95 within +25%):

| Live mix window | Length | ops/s | p50 ms | **p95 ms** | p99 ms | create p95 | patch p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| alone, before | 60 s | 124.9 | 69.0 | **101.7** | 141.4 | 58.0 | 118.8 |
| with the backfill | 671 s | 101.8 | 82.7 | **120.8** | 173.7 | 70.5 | 137.8 |
| alone, after | 60 s | 102.0 | 80.0 | **131.9** | 173.8 | 76.6 | 149.6 |
- D10 delta: +18.7% against before, -8.4% against after (budget +25%); live errors map[]
- autovacuum on records so far: vacuums 11, analyzes 18, last 2026-11-28 05:38:53.786115+00
- VACUUM (ANALYZE) records after the backfills: 1s; records 1738 MB heap, 1205 MB indexes

### Item 2: size per promoted field

| Slot (field) | Filled rows | Value bytes in heap (sum of pg_column_size) | Group index (as grown by the backfill) | Fresh build, group key | Fresh build, plain key |
| --- | --- | --- | --- | --- | --- |
| fx_num_1 (score) | 451222 | 3525 kB | 78 MB | 37 MB | 29 MB |
| fx_text_1 (tier) | 451222 | 1721 kB | 56 MB | 37 MB | 29 MB |
| fx_ts_1 (opened) | 451222 | 3525 kB | 56 MB | 37 MB | 29 MB |
| fx_text_2 (zone) | 451222 | 1891 kB | 56 MB | 37 MB | 29 MB |
| fx_bool_1 (b2) | 451222 | 441 kB | 55 MB | 37 MB | 29 MB |
| fx_num_2 (pk) | 400000 | 3125 kB | 43 MB | 33 MB | 26 MB |

### Item 1: write amplification, POST and PATCH through the handler, 8 clients, 30s per cell

D7 n: n of (score, tier, opened, zone, b2) flagged (side rows), no slot. Slot n: the same n flagged and mapped to slots (A3-5 keeps the side rows), each slot with its index. An update with n = 0 still names score, which is then neither flagged nor mapped. The PATCH targets of a slot arm carry exactly its n slot values.

| Arm | Write | n | ops/s | p50 ms | p95 ms | WAL B/op | WAL records/op | FPI/op | records HOT / updates | errors |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| D7 0 | insert | 6992 | 233.1 | 33.0 | **42.4** | 9231 | 31.4 | 0.14 | - | 0 |
| D7 0 | update promoted field (score) | 2830 | 94.3 | 80.9 | **122.2** | 32006 | 30.3 | 4.81 | 0 / 2830 (0%) | 0 |
| D7 0 | update unrelated field (b001) | 3092 | 103.1 | 77.4 | **93.2** | 19074 | 41.0 | 3.05 | 0 / 3092 (0%) | 0 |
| D7 1 | insert | 5326 | 177.5 | 39.3 | **78.9** | 13111 | 36.7 | 0.79 | - | 0 |
| D7 1 | update promoted field (score) | 2771 | 92.4 | 88.6 | **123.5** | 20234 | 36.0 | 1.92 | 0 / 2771 (0%) | 0 |
| D7 1 | update unrelated field (b001) | 2835 | 94.5 | 83.7 | **121.1** | 16922 | 30.9 | 1.51 | 0 / 2835 (0%) | 0 |
| D7 3 | insert | 4105 | 136.8 | 49.3 | **100.9** | 14081 | 45.2 | 0.78 | - | 0 |
| D7 3 | update promoted field (score) | 2579 | 86.0 | 89.6 | **134.4** | 13269 | 35.7 | 0.93 | 0 / 2579 (0%) | 0 |
| D7 3 | update unrelated field (b001) | 2687 | 89.6 | 86.7 | **120.7** | 35453 | 30.8 | 5.15 | 0 / 2687 (0%) | 0 |
| D7 5 | insert | 4585 | 152.8 | 48.5 | **80.6** | 17922 | 54.0 | 1.25 | - | 0 |
| D7 5 | update promoted field (score) | 2571 | 85.7 | 91.3 | **121.7** | 14590 | 35.0 | 1.19 | 0 / 2571 (0%) | 0 |
| D7 5 | update unrelated field (b001) | 2602 | 86.7 | 89.2 | **120.5** | 11313 | 30.7 | 0.74 | 0 / 2602 (0%) | 0 |
| slot 1, group index | insert | 4706 | 156.9 | 45.7 | **78.9** | 13158 | 47.2 | 1.78 | - | 0 |
| slot 1, group index | update promoted field (score) | 2401 | 80.0 | 97.0 | **132.8** | 29326 | 37.4 | 4.13 | 0 / 2401 (0%) | 0 |
| slot 1, group index | update unrelated field (b001) | 2603 | 86.8 | 89.4 | **123.0** | 10197 | 31.8 | 0.57 | 0 / 2603 (0%) | 0 |
| slot 3, group index | insert | 4222 | 140.7 | 54.6 | **78.9** | 17631 | 48.5 | 1.34 | - | 0 |
| slot 3, group index | update promoted field (score) | 2646 | 88.2 | 86.3 | **136.2** | 41506 | 39.6 | 6.21 | 0 / 2646 (0%) | 0 |
| slot 3, group index | update unrelated field (b001) | 2793 | 93.1 | 85.8 | **107.6** | 10214 | 33.6 | 0.52 | 0 / 2793 (0%) | 0 |
| slot 5, group index | insert | 4536 | 151.2 | 52.0 | **69.5** | 16577 | 58.8 | 0.92 | - | 0 |
| slot 5, group index | update promoted field (score) | 2496 | 83.2 | 91.7 | **139.7** | 32749 | 41.6 | 4.66 | 0 / 2496 (0%) | 0 |
| slot 5, group index | update unrelated field (b001) | 2550 | 85.0 | 93.2 | **131.5** | 10023 | 35.6 | 0.39 | 0 / 2550 (0%) | 0 |

### S-b2 (i) and S-c2 without an ACL: the group index only (A3-2's set), line 150 ms

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S-b2 (i) broad 50% (pk 3), no ACL, group index only | **290.0** | 230.6 | 690 | **miss** | 0.7 | 0.0 / 98.4 / 1.7 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 86090 | join: spike_fx_num_2_grp 1001, buffers 1066 | 0 / 0 |
| S-c2 num lt (score < 200, 2%), no ACL, group index only | **63.5** | 50.7 | 3078 | pass | 1.7 | 0.0 / 3.7 / 2.7 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 1493 | join: spike_fx_num_1_grp 1001, buffers 835 | 0 / 0 |
| S-c2 num gte (score >= 9800, 2%), no ACL, group index only | **111.0** | 90.3 | 1748 | pass | 1.8 | 0.0 / 33.8 / 2.1 | nested loop: spike_fx_num_1_grp 7680, groups 53, groups_live_key 53, buffers 7913 | join: spike_fx_num_1_grp 1001, buffers 1103 | 0 / 0 |
| S-c2 num range (2000 <= score < 2200, 2%), no ACL, group index only | **161.4** | 136.3 | 1176 | **miss** (provisional) | 1.7 | 0.0 / 59.4 / 2.0 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 46671 | join: spike_fx_num_1_grp 1001, buffers 1100 | 0 / 0 |
| S-c2 text eq (tier v2, 2%), no ACL, group index only | **79.1** | 63.8 | 2449 | pass | 1.7 | 0.0 / 11.7 / 2.3 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 7843 | join: spike_fx_text_1_grp 1001, buffers 1089 | 0 / 0 |
| S-c2 text in 3 values (tier v2, v025, v001, 2.26%), no ACL, group index only | **82.4** | 66.4 | 2359 | pass | 2.3 | 0.0 / 12.9 / 2.4 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 7473 | join: spike_fx_text_1_grp 1001, buffers 1104 | 0 / 0 |
| S-c2 num in 3 values (pk 10, 20, 30, 17%), no ACL, group index only | **283.5** | 230.1 | 688 | **miss** | 1.8 | 0.0 / 105.5 / 1.7 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 86171 | join: spike_fx_num_2_grp 1001, buffers 1058 | 0 / 0 |

### Item 3: the plain index, CREATE INDEX CONCURRENTLY on a filled slot

| Statement | Wall | Locks on records held (sampled every 2 ms) | DDL waiting on a lock | Writers waiting on a lock: max / time | Live ops overlapping | their p95 ms | their max ms |
| --- | --- | --- | --- | --- | --- | --- | --- |
| CREATE INDEX CONCURRENTLY spike_fx_num_1_plain (400k filled), live mix | 4.151s | ShareUpdateExclusiveLock (seen over 4.14s) | 48ms | 1 / 7ms | 368 | 136.1 | 158.1 |
| CREATE INDEX CONCURRENTLY group key on fx_num_1 (400k filled, a second copy), live mix | 3.877s | ShareUpdateExclusiveLock (seen over 3.863s) | 56ms | 0 / 0s | 347 | 132.9 | 167.7 |
| CREATE INDEX CONCURRENTLY spike_fx_text_1_plain (400k filled), alone | 4.605s | ShareUpdateExclusiveLock (seen over 4.586s) | 0s | 0 / 0s | 0 | 0.0 | 0.0 |
| CREATE INDEX CONCURRENTLY spike_fx_ts_1_plain (400k filled), alone | 3.534s | ShareUpdateExclusiveLock (seen over 3.508s) | 0s | 0 / 0s | 0 | 0.0 | 0.0 |
| CREATE INDEX CONCURRENTLY spike_fx_text_2_plain (400k filled), alone | 2.874s | ShareUpdateExclusiveLock (seen over 2.867s) | 0s | 0 / 0s | 0 | 0.0 | 0.0 |
| CREATE INDEX CONCURRENTLY spike_fx_bool_1_plain (400k filled), alone | 2.482s | ShareUpdateExclusiveLock (seen over 2.467s) | 0s | 0 / 0s | 0 | 0.0 | 0.0 |
| CREATE INDEX CONCURRENTLY spike_fx_num_2_plain (400k filled), alone | 2.333s | ShareUpdateExclusiveLock (seen over 2.32s) | 0s | 0 / 0s | 0 | 0.0 | 0.0 |

| Live mix window | Length | ops/s | p50 ms | **p95 ms** | p99 ms | create p95 | patch p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| before the builds | 60 s | 108.8 | 77.7 | **121.8** | 158.4 | 72.2 | 140.5 |
| during the two builds | 8 s | 86.9 | 94.4 | **135.0** | 143.4 | 79.5 | 138.2 |
- live errors: map[]

### Item 1 with both indexes (group + plain) per promoted field

| Arm | Write | n | ops/s | p50 ms | p95 ms | WAL B/op | WAL records/op | FPI/op | records HOT / updates | errors |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| slot 1, group + plain | insert | 4580 | 152.7 | 48.7 | **77.4** | 13215 | 38.9 | 0.70 | - | 0 |
| slot 1, group + plain | update promoted field (score) | 2630 | 87.7 | 89.7 | **120.8** | 36684 | 36.6 | 5.31 | 0 / 2630 (0%) | 0 |
| slot 1, group + plain | update unrelated field (b001) | 2348 | 78.3 | 92.1 | **167.8** | 19187 | 32.5 | 2.13 | 0 / 2348 (0%) | 0 |
| slot 3, group + plain | insert | 4990 | 166.3 | 46.7 | **61.2** | 16416 | 51.4 | 0.92 | - | 0 |
| slot 3, group + plain | update promoted field (score) | 2870 | 95.7 | 83.8 | **111.0** | 30540 | 41.0 | 3.96 | 0 / 2870 (0%) | 0 |
| slot 3, group + plain | update unrelated field (b001) | 2426 | 80.9 | 90.0 | **160.8** | 9307 | 36.5 | 0.22 | 0 / 2426 (0%) | 0 |
| slot 5, group + plain | insert | 4506 | 150.2 | 52.0 | **67.9** | 34158 | 70.1 | 3.58 | - | 0 |
| slot 5, group + plain | update promoted field (score) | 2234 | 74.5 | 104.9 | **144.4** | 30555 | 44.1 | 3.93 | 0 / 2234 (0%) | 0 |
| slot 5, group + plain | update unrelated field (b001) | 2112 | 70.4 | 100.5 | **203.5** | 11471 | 40.6 | 0.43 | 0 / 2112 (0%) | 0 |

### Item 1 summary: p95 ms and WAL B/op against D7 0 (the same writes, nothing flagged or mapped)

| Write | D7 0 | D7 1 | D7 3 | D7 5 | slot 1, group | slot 3, group | slot 5, group | slot 1, both | slot 3, both | slot 5, both |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| insert | 42.4 / 9231 | 78.9 / 13111 | 100.9 / 14081 | 80.6 / 17922 | 78.9 / 13158 | 78.9 / 17631 | 69.5 / 16577 | 77.4 / 13215 | 61.2 / 16416 | 67.9 / 34158 |
| update promoted field (score) | 122.2 / 32006 | 123.5 / 20234 | 134.4 / 13269 | 121.7 / 14590 | 132.8 / 29326 | 136.2 / 41506 | 139.7 / 32749 | 120.8 / 36684 | 111.0 / 30540 | 144.4 / 30555 |
| update unrelated field (b001) | 93.2 / 19074 | 121.1 / 16922 | 120.7 / 35453 | 120.5 / 11313 | 123.0 / 10197 | 107.6 / 10214 | 131.5 / 10023 | 167.8 / 19187 | 160.8 / 9307 | 203.5 / 11471 |

### S-b2 (ii) and S-c2 without an ACL: group + plain indexes, line 150 ms

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S-b2 (ii) broad 50% (pk 3), no ACL, group + plain | **386.0** | 320.8 | 495 | **miss** | 1.1 | 0.0 / 156.0 / 2.3 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 111945 | join: spike_fx_num_2_plain 1001, buffers 1193 | 0 / 0 |
| S-c2 num lt (score < 200, 2%), no ACL, group + plain | **70.8** | 56.5 | 2777 | pass | 2.1 | 0.0 / 5.0 / 2.5 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 1097 | join: spike_fx_num_1_plain 1001, buffers 911 | 0 / 0 |
| S-c2 num gte (score >= 9800, 2%), no ACL, group + plain | **130.6** | 103.6 | 1522 | pass | 2.3 | 0.0 / 36.0 / 2.4 | nested loop: spike_fx_num_1_plain 7675, groups 53, groups_live_key 53, buffers 7801 | join: spike_fx_num_1_plain 1001, buffers 1195 | 0 / 0 |
| S-c2 num range (2000 <= score < 2200, 2%), no ACL, group + plain | **127.0** | 101.7 | 1541 | pass | 2.3 | 0.0 / 29.3 / 2.6 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 25314 | join: spike_fx_num_1_plain 1001, buffers 1197 | 0 / 0 |
| S-c2 text eq (tier v2, 2%), no ACL, group + plain | **78.5** | 58.0 | 2648 | pass | 1.8 | 0.0 / 2.4 / 3.1 | nested loop: groups_pkey 52, spike_fx_text_1_plain 52, entity_definitions_number_prefix_key 1, 52 inner loops, buffers 216 | join: spike_fx_text_1_plain 1001, buffers 1183 | 0 / 0 |
| S-c2 text in 3 values (tier v2, v025, v001, 2.26%), no ACL, group + plain | **98.5** | 75.3 | 2060 | pass | 2.6 | 0.0 / 13.0 / 2.5 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 7393 | join: spike_fx_text_1_plain 1001, buffers 1188 | 0 / 0 |
| S-c2 num in 3 values (pk 10, 20, 30, 17%), no ACL, group + plain | **371.4** | 321.0 | 503 | **miss** | 1.8 | 0.0 / 150.7 / 1.9 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 112026 | join: spike_fx_num_2_plain 1001, buffers 1185 | 0 / 0 |
- churn before the ACL cells (no VACUUM FULL since the fixture): n_dead_tup 60889, n_live_tup 905901, updates since the fixture 3194291 (HOT 15102), autovacuums 11, relallvisible 162701 of 222358 pages

### S-b1: per-group merge page (one ordered arm per visible group), group index only, line 150 ms

Pass: no arm's scan of the slot index reads more than 51 + epsilon rows (taken as 102), no full Sort over more than 1,000 rows, p95 <= 150 ms.

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S-b1 broad 50% (pk 3), a6, compiled (b)+group | **466.0** | 358.8 | 435 | **miss** | 2.4 | 0.0 / 124.0 / 7.7 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 113753 | nested loop: spike_fx_num_2_grp 1001, buffers 1221 | 0 / 0 |
| S-b1 broad 50% (pk 3), a6, per-group merge | **159.0** | 123.0 | 1278 | **miss** (provisional) | 1.6 | 0.0 / 29.9 / 10.2 | nested loop: records_list_group 53, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 8876 | nested loop: spike_fx_num_2_grp 1001, buffers 1217 | 0 / 0 |
| S-b1 broad 50% (pk 3), a25, compiled (b)+group | **434.0** | 358.2 | 445 | **miss** | 2.3 | 0.0 / 144.9 / 6.0 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 112964 | nested loop: spike_fx_num_2_grp 1001, buffers 1159 | 0 / 0 |
| S-b1 broad 50% (pk 3), a25, per-group merge | **314.7** | 259.5 | 613 | **miss** | 1.7 | 0.0 / 78.9 / 15.6 | nested loop: records_list_group 63, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 42200 | nested loop: spike_fx_num_2_grp 1001, buffers 1217 | 0 / 0 |
| S-b1 broad 50% (pk 3), a50, compiled (b)+group | **413.2** | 322.1 | 492 | **miss** | 1.7 | 0.0 / 129.4 / 6.1 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 112828 | nested loop: spike_fx_num_2_grp 1001, buffers 1159 | 0 / 0 |
| S-b1 broad 50% (pk 3), a50, per-group merge | **559.3** | 414.6 | 372 | **miss** | 1.2 | 0.0 / 142.2 / 18.6 | nested loop: records_list_group 75, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 81768 | nested loop: spike_fx_num_2_grp 1001, buffers 1217 | 0 / 0 |
| S-b1 anti-correlated 2% (pk 40), a6, compiled (b)+group | **107.1** | 84.5 | 1845 | pass | 1.8 | 0.0 / 10.9 / 8.9 | nested loop: spike_fx_num_2_grp 475, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1984 | nested loop: spike_fx_num_2_grp 475, records_list_requester 9, record_shares 3, 3 inner loops, buffers 1777 | 0 / 0 |
| S-b1 anti-correlated 2% (pk 40), a6, per-group merge | **128.8** | 99.7 | 1549 | pass | 2.1 | 0.0 / 12.3 / 11.1 | nested loop: spike_fx_num_2_grp 99, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1608 | nested loop: spike_fx_num_2_grp 475, records_list_requester 9, record_shares 3, 3 inner loops, buffers 1777 | 0 / 0 |
| S-b1 anti-correlated 2% (pk 40), a25, compiled (b)+group | **118.3** | 94.7 | 1657 | pass | 2.4 | 0.0 / 14.5 / 7.2 | nested loop: spike_fx_num_2_grp 2101, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 3675 | nested loop: spike_fx_num_2_grp 1001, buffers 1229 | 0 / 0 |
| S-b1 anti-correlated 2% (pk 40), a25, per-group merge | **178.8** | 142.1 | 1093 | **miss** | 1.8 | 0.0 / 22.7 / 21.6 | nested loop: spike_fx_num_2_grp 429, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1993 | nested loop: spike_fx_num_2_grp 1001, buffers 1249 | 0 / 0 |
| S-b1 anti-correlated 2% (pk 40), a50, compiled (b)+group | **1485.1** | 1244.3 | 129 | **miss** | 1.7 | 0.0 / 835.1 / 6.0 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 495084 | nested loop: spike_fx_num_2_grp 1001, buffers 1259 | 0 / 0 |
| S-b1 anti-correlated 2% (pk 40), a50, per-group merge | **267.0** | 209.5 | 743 | **miss** | 1.7 | 0.0 / 56.7 / 42.8 | nested loop: spike_fx_num_2_grp 825, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 2453 | nested loop: spike_fx_num_2_grp 1001, buffers 1234 | 0 / 0 |

| Fixture | Agent | Page shape | p95 ms | slot-index scans in the page | max rows one scan read | max full-Sort input | S-b1 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| broad 50% (pk 3) | a6 | compiled (b)+group | 466.0 | 0 | 0 | 3 | control |
| broad 50% (pk 3) | a6 | per-group merge | 159.0 | 0 | 0 | 3 | **fail** |
| broad 50% (pk 3) | a25 | compiled (b)+group | 434.0 | 0 | 0 | 2 | control |
| broad 50% (pk 3) | a25 | per-group merge | 314.7 | 0 | 0 | 2 | **fail** |
| broad 50% (pk 3) | a50 | compiled (b)+group | 413.2 | 0 | 0 | 1 | control |
| broad 50% (pk 3) | a50 | per-group merge | 559.3 | 0 | 0 | 1 | **fail** |
| anti-correlated 2% (pk 40) | a6 | compiled (b)+group | 107.1 | 1 | 475 | 475 | control |
| anti-correlated 2% (pk 40) | a6 | per-group merge | 128.8 | 3 | 33 | 0 | pass |
| anti-correlated 2% (pk 40) | a25 | compiled (b)+group | 118.3 | 1 | 2101 | 2101 | control |
| anti-correlated 2% (pk 40) | a25 | per-group merge | 178.8 | 13 | 33 | 0 | **fail** |
| anti-correlated 2% (pk 40) | a50 | compiled (b)+group | 1485.1 | 0 | 0 | 0 | control |
| anti-correlated 2% (pk 40) | a50 | per-group merge | 267.0 | 25 | 33 | 0 | **fail** |

Heap fetches under churn (count / page plan, one EXPLAIN ANALYZE each):

| Cell | count Heap Fetches | page Heap Fetches |
| --- | --- | --- |
| S-b1 broad 50% (pk 3), a6, compiled (b)+group | 477 | 0 |
| S-b1 broad 50% (pk 3), a6, per-group merge | 461 | 0 |
| S-b1 broad 50% (pk 3), a25, compiled (b)+group | 433 | 0 |
| S-b1 broad 50% (pk 3), a25, per-group merge | 461 | 0 |
| S-b1 broad 50% (pk 3), a50, compiled (b)+group | 433 | 0 |
| S-b1 broad 50% (pk 3), a50, per-group merge | 461 | 0 |
| S-b1 anti-correlated 2% (pk 40), a6, compiled (b)+group | 215 | 0 |
| S-b1 anti-correlated 2% (pk 40), a6, per-group merge | 215 | 0 |
| S-b1 anti-correlated 2% (pk 40), a25, compiled (b)+group | 458 | 0 |
| S-b1 anti-correlated 2% (pk 40), a25, per-group merge | 468 | 0 |
| S-b1 anti-correlated 2% (pk 40), a50, compiled (b)+group | 478 | 0 |
| S-b1 anti-correlated 2% (pk 40), a50, per-group merge | 457 | 0 |

### S-c2 (operators and types) and S-c3 (10% and 20%) with an ACL, compiled shape and per-group merge, line 150 ms

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S-c2 num lt (score < 200, 2%), a6, compiled | **150.3** | 106.5 | 1424 | **miss** (provisional) | 2.0 | 0.0 / 16.8 / 15.9 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 3348 | nested loop: spike_fx_num_1_grp 1001, buffers 2400 | 0 / 0 |
| S-c2 num lt (score < 200, 2%), a6, per-group merge | **145.7** | 106.6 | 1447 | pass (provisional) | 2.9 | 0.0 / 14.3 / 24.3 | nested loop: records_list_group 53, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1272 | nested loop: spike_fx_num_1_grp 1001, records_list_group_state 0, buffers 1556 | 0 / 0 |
| S-c2 num lt (score < 200, 2%), a50, compiled | **113.8** | 84.9 | 1807 | pass | 3.2 | 0.0 / 11.0 / 9.5 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 2379 | nested loop: spike_fx_num_1_grp 1001, buffers 1509 | 0 / 0 |
| S-c2 num lt (score < 200, 2%), a50, per-group merge | **312.5** | 203.3 | 737 | **miss** | 2.3 | 0.0 / 42.7 / 35.8 | nested loop: records_list_group 75, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 2033 | nested loop: spike_fx_num_1_grp 1001, records_list_group_state 0, buffers 1556 | 0 / 0 |
| S-c2 num gte (score >= 9800, 2%), a6, compiled | **132.2** | 108.2 | 1452 | pass | 2.3 | 0.0 / 14.0 / 15.2 | nested loop: spike_fx_num_1_grp 460, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 2217 | nested loop: spike_fx_num_1_grp 460, records_list_requester 11, record_shares 3, 3 inner loops, buffers 2450 | 0 / 0 |
| S-c2 num gte (score >= 9800, 2%), a6, per-group merge | **152.4** | 115.1 | 1341 | **miss** (provisional) | 2.4 | 0.0 / 16.4 / 15.4 | nested loop: spike_fx_num_1_grp 460, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 2048 | nested loop: spike_fx_num_1_grp 460, records_list_requester 11, record_shares 3, 3 inner loops, buffers 2282 | 0 / 0 |
| S-c2 num gte (score >= 9800, 2%), a50, compiled | **159.0** | 126.2 | 1245 | **miss** (provisional) | 2.3 | 0.0 / 34.5 / 11.5 | nested loop: spike_fx_num_1_grp 3855, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 9051 | nested loop: spike_fx_num_1_grp 1001, buffers 2478 | 0 / 0 |
| S-c2 num gte (score >= 9800, 2%), a50, per-group merge | **332.8** | 262.2 | 590 | **miss** | 2.3 | 0.0 / 66.5 / 30.9 | nested loop: spike_fx_num_1_grp 3855, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 9387 | nested loop: spike_fx_num_1_grp 1001, records_list_group_state 0, buffers 2288 | 0 / 0 |
| S-c2 num range (2000 <= score < 2200, 2%), a6, compiled | **148.3** | 111.1 | 1402 | pass (provisional) | 1.8 | 0.0 / 16.5 / 12.6 | nested loop: spike_fx_num_1_grp 459, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 2197 | nested loop: spike_fx_num_1_grp 459, records_list_requester 9, record_shares 3, 3 inner loops, buffers 2445 | 0 / 0 |
| S-c2 num range (2000 <= score < 2200, 2%), a6, per-group merge | **156.0** | 124.6 | 1259 | **miss** (provisional) | 2.4 | 0.0 / 15.9 / 14.4 | nested loop: spike_fx_num_1_grp 459, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 2025 | nested loop: spike_fx_num_1_grp 459, records_list_requester 9, record_shares 3, 3 inner loops, buffers 2273 | 0 / 0 |
| S-c2 num range (2000 <= score < 2200, 2%), a50, compiled | **182.4** | 136.7 | 1126 | **miss** | 2.2 | 0.0 / 35.1 / 11.2 | nested loop: spike_fx_num_1_grp 3949, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 9203 | nested loop: spike_fx_num_1_grp 1001, buffers 2369 | 0 / 0 |
| S-c2 num range (2000 <= score < 2200, 2%), a50, per-group merge | **361.6** | 302.7 | 524 | **miss** | 1.7 | 0.0 / 76.5 / 33.7 | nested loop: spike_fx_num_1_grp 3949, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 9431 | nested loop: spike_fx_num_1_grp 1001, records_list_group_state 0, buffers 2327 | 0 / 0 |
| S-c2 text eq (tier v2, 2%), a6, compiled | **136.5** | 110.2 | 1430 | pass (provisional) | 2.2 | 0.0 / 16.4 / 12.5 | nested loop: spike_fx_text_1_grp 485, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1600 | nested loop: spike_fx_text_1_grp 485, records_list_requester 6, record_shares 3, 3 inner loops, buffers 1812 | 0 / 0 |
| S-c2 text eq (tier v2, 2%), a6, per-group merge | **137.3** | 112.6 | 1391 | pass (provisional) | 2.9 | 0.0 / 13.5 / 11.9 | nested loop: spike_fx_text_1_grp 99, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1209 | nested loop: spike_fx_text_1_grp 485, records_list_requester 6, record_shares 3, 3 inner loops, buffers 1812 | 0 / 0 |
| S-c2 text eq (tier v2, 2%), a50, compiled | **150.8** | 123.4 | 1268 | **miss** (provisional) | 2.2 | 0.0 / 24.4 / 10.2 | nested loop: spike_fx_text_1_grp 3956, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 5298 | nested loop: spike_fx_text_1_grp 1001, buffers 1222 | 0 / 0 |
| S-c2 text eq (tier v2, 2%), a50, per-group merge | **261.2** | 220.4 | 710 | **miss** | 1.7 | 0.0 / 39.9 / 30.9 | nested loop: spike_fx_text_1_grp 825, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 2135 | nested loop: spike_fx_text_1_grp 1001, buffers 1216 | 0 / 0 |
| S-c2 text in 3 values (tier v2, v025, v001, 2.26%), a6, compiled | **136.8** | 101.9 | 1523 | pass (provisional) | 2.2 | 0.0 / 12.3 / 15.4 | nested loop: spike_fx_text_1_grp 531, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1670 | nested loop: spike_fx_text_1_grp 531, records_list_requester 6, record_shares 3, 3 inner loops, buffers 1896 | 0 / 0 |
| S-c2 text in 3 values (tier v2, v025, v001, 2.26%), a6, per-group merge | **144.1** | 113.6 | 1381 | pass (provisional) | 2.4 | 0.0 / 15.7 / 13.3 | nested loop: spike_fx_text_1_grp 531, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1670 | nested loop: spike_fx_text_1_grp 531, records_list_requester 6, record_shares 3, 3 inner loops, buffers 1895 | 0 / 0 |
| S-c2 text in 3 values (tier v2, v025, v001, 2.26%), a50, compiled | **200.0** | 151.6 | 1026 | **miss** | 1.9 | 0.0 / 44.7 / 10.8 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 23491 | nested loop: spike_fx_text_1_grp 1001, buffers 1335 | 0 / 0 |
| S-c2 text in 3 values (tier v2, v025, v001, 2.26%), a50, per-group merge | **441.4** | 373.2 | 424 | **miss** | 2.1 | 0.0 / 98.0 / 28.5 | nested loop: records_list_group 75, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 36318 | nested loop: spike_fx_text_1_grp 1001, buffers 1251 | 0 / 0 |
| S-c2 num in 3 values (pk 10, 20, 30, 17%), a6, compiled | **586.3** | 513.2 | 313 | **miss** | 1.7 | 0.0 / 218.8 / 5.5 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 47, 105 inner loops, buffers 156597 | nested loop: spike_fx_num_2_grp 479, records_list_requester 14, record_shares 3, 3 inner loops, buffers 1799 | 0 / 0 |
| S-c2 num in 3 values (pk 10, 20, 30, 17%), a6, per-group merge | **187.6** | 144.3 | 1084 | **miss** | 1.1 | 0.0 / 44.1 / 13.0 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_group 49, 105 inner loops, buffers 11554 | nested loop: spike_fx_num_2_grp 479, records_list_requester 14, record_shares 3, 3 inner loops, buffers 1799 | 0 / 0 |
| S-c2 num in 3 values (pk 10, 20, 30, 17%), a50, compiled | **611.8** | 520.3 | 307 | **miss** | 1.9 | 0.0 / 252.5 / 6.7 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 49, 105 inner loops, buffers 156746 | nested loop: spike_fx_num_2_grp 487, records_list_requester 7, record_shares 3, 3 inner loops, buffers 2050 | 0 / 0 |
| S-c2 num in 3 values (pk 10, 20, 30, 17%), a50, per-group merge | **1170.8** | 944.8 | 168 | **miss** | 1.6 | 0.0 / 436.4 / 21.6 | nested loop: records_list_group 57, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 230096 | nested loop: spike_fx_num_2_grp 487, records_list_requester 7, record_shares 3, 3 inner loops, buffers 2053 | 0 / 0 |
| S-c3 num lt 10% (score < 1000), a6, compiled | **140.6** | 111.6 | 1413 | pass (provisional) | 1.2 | 0.0 / 14.3 / 15.6 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 3203 | nested loop: spike_fx_num_1_grp 1001, buffers 2400 | 0 / 0 |
| S-c3 num lt 10% (score < 1000), a6, per-group merge | **138.1** | 110.9 | 1417 | pass (provisional) | 2.5 | 0.0 / 12.2 / 15.1 | nested loop: records_list_group 53, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1127 | nested loop: spike_fx_num_1_grp 1001, records_list_group_state 0, buffers 1556 | 0 / 0 |
| S-c3 num lt 10% (score < 1000), a50, compiled | **122.6** | 97.6 | 1609 | pass | 2.4 | 0.0 / 13.1 / 9.9 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 2247 | nested loop: spike_fx_num_1_grp 1001, buffers 1509 | 0 / 0 |
| S-c3 num lt 10% (score < 1000), a50, per-group merge | **275.3** | 219.2 | 708 | **miss** | 2.8 | 0.0 / 33.2 / 30.4 | nested loop: records_list_group 75, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1901 | nested loop: spike_fx_num_1_grp 1001, records_list_group_state 0, buffers 1556 | 0 / 0 |
| S-c3 num lt 20% (score < 2000), a6, compiled | **131.8** | 108.3 | 1456 | pass | 1.8 | 0.0 / 14.4 / 12.7 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 3044 | nested loop: records_list_group_state 1001, buffers 1787 | 0 / 0 |
| S-c3 num lt 20% (score < 2000), a6, per-group merge | **145.5** | 112.7 | 1381 | pass (provisional) | 2.4 | 0.0 / 17.3 / 28.2 | nested loop: records_list_group 53, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1113 | nested loop: spike_fx_num_1_grp 1001, records_list_group_state 0, buffers 1556 | 0 / 0 |
| S-c3 num lt 20% (score < 2000), a50, compiled | **122.9** | 91.3 | 1685 | pass | 2.3 | 0.0 / 12.7 / 11.3 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 1641 | nested loop: spike_fx_num_1_grp 1001, buffers 1509 | 0 / 0 |
| S-c3 num lt 20% (score < 2000), a50, per-group merge | **290.0** | 230.2 | 679 | **miss** | 2.6 | 0.0 / 31.4 / 29.7 | nested loop: records_list_group 75, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1519 | nested loop: spike_fx_num_1_grp 1001, records_list_group_state 0, buffers 1556 | 0 / 0 |
| S-c3 text eq 10% (tier v10), a6, compiled | **128.0** | 94.2 | 1636 | pass | 1.7 | 0.0 / 14.7 / 7.2 | nested loop: spike_fx_text_1_grp 2320, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 3065 | nested loop: spike_fx_text_1_grp 1001, buffers 1214 | 0 / 0 |
| S-c3 text eq 10% (tier v10), a6, per-group merge | **113.6** | 83.4 | 1835 | pass | 1.7 | 0.0 / 11.3 / 10.9 | nested loop: spike_fx_text_1_grp 99, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 813 | nested loop: spike_fx_text_1_grp 1001, buffers 1204 | 0 / 0 |
| S-c3 text eq 10% (tier v10), a50, compiled | **103.9** | 84.4 | 1860 | pass | 2.2 | 0.0 / 10.2 / 10.6 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 1799 | nested loop: spike_fx_text_1_grp 1001, buffers 1162 | 0 / 0 |
| S-c3 text eq 10% (tier v10), a50, per-group merge | **245.9** | 183.4 | 826 | **miss** | 2.3 | 0.0 / 29.4 / 22.3 | nested loop: spike_fx_text_1_grp 825, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1671 | nested loop: spike_fx_text_1_grp 1001, buffers 1183 | 0 / 0 |
| S-c3 text eq 20% (tier v20), a6, compiled | **159.7** | 117.2 | 1328 | **miss** (provisional) | 1.6 | 0.0 / 29.6 / 7.7 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 20810 | nested loop: spike_fx_text_1_grp 1001, buffers 1202 | 0 / 0 |
| S-c3 text eq 20% (tier v20), a6, per-group merge | **111.4** | 82.5 | 1853 | pass | 1.7 | 0.0 / 10.6 / 10.2 | nested loop: spike_fx_text_1_grp 99, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 782 | nested loop: spike_fx_text_1_grp 1001, buffers 1213 | 0 / 0 |
| S-c3 text eq 20% (tier v20), a50, compiled | **89.2** | 72.1 | 2159 | pass | 2.2 | 0.0 / 9.2 / 8.4 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 1415 | nested loop: spike_fx_text_1_grp 1001, buffers 1180 | 0 / 0 |
| S-c3 text eq 20% (tier v20), a50, per-group merge | **229.4** | 178.6 | 860 | **miss** | 1.9 | 0.0 / 47.4 / 33.2 | nested loop: spike_fx_text_1_grp 825, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1881 | nested loop: spike_fx_text_1_grp 1001, buffers 1213 | 0 / 0 |

### S-c1: a shared slot (R-d)

- ticket2: 100k records copied in 27s; a6 sees 6049 of them

| Stage | Predicate | Estimate | Actual | Off by | S-c1 |
| --- | --- | --- | --- | --- | --- |
| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S-c1 own slot large type, pk 3, a6 | **507.9** | 314.7 | 475 | **miss** | 2.3 | 0.0 / 0.0 / 0.0 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 112925 | nested loop: spike_fx_num_2_grp 1001, buffers 744 | 0 / 0 |
| S-c1 own slot large type, pk 40, a6 | **115.5** | 80.2 | 1894 | pass | 2.0 | 0.0 / 0.0 / 0.0 | nested loop: spike_fx_num_2_grp 475, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1984 | nested loop: spike_fx_num_2_grp 475, records_list_requester 9, record_shares 3, 3 inner loops, buffers 1562 | 0 / 0 |
| S-c1 own slot ticket2, pk 3, a6 | **108.9** | 72.8 | 2054 | pass | 2.3 | 0.0 / 0.0 / 0.0 | nested loop: spike_fx_own_grp 111, entity_definitions_pkey 102, groups_live_key 51, 105 inner loops, buffers 738 | nested loop: spike_fx_own_grp 111, record_shares 3, record_shares_principal 3, 3 inner loops, buffers 327 | 0 / 0 |
| S-c1 own slot ticket2, pk 40, a6 | **111.7** | 75.8 | 1995 | pass | 2.8 | 0.0 / 0.0 / 0.0 | nested loop: spike_fx_own_grp 138, entity_definitions_pkey 102, groups_live_key 51, 105 inner loops, buffers 762 | nested loop: spike_fx_own_grp 138, record_shares 3, record_shares_principal 3, 3 inner loops, buffers 344 | 0 / 0 |
| S-c1 own slot ticket2, pk 3, no ACL | **68.6** | 50.9 | 3003 | pass | 1.7 | 0.0 / 0.0 / 0.0 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 1816 | join: spike_fx_own_grp 1001, buffers 459 | 0 / 0 |

- shared: fx_num_2 now holds pk for the large type (skewed), ticket2 and all 399 small tenants (uniform), 892000 filled; filled in 2m12s
- item 2 at 800k: fx_num_2 fresh builds, all 892000 filled rows: group 74 MB, plain 58 MB; the large type's 400k only: group 32 MB, plain 25 MB; value bytes 6969 kB

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S-c1 shared slot large type, pk 3, a6 | **456.9** | 368.7 | 433 | **miss** | 2.1 | 0.0 / 0.0 / 0.0 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 112925 | nested loop: spike_fx_num_2_grp 1001, buffers 745 | 0 / 0 |
| S-c1 shared slot large type, pk 40, a6 | **109.8** | 78.7 | 1951 | pass | 2.3 | 0.0 / 0.0 / 0.0 | nested loop: spike_fx_num_2_grp 475, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1988 | nested loop: spike_fx_num_2_grp 475, records_list_requester 9, record_shares 3, 3 inner loops, buffers 1566 | 0 / 0 |
| S-c1 shared slot ticket2, pk 3, a6 | **78.5** | 63.1 | 2485 | pass | 2.3 | 0.0 / 0.0 / 0.0 | nested loop: spike_fx_num_2_grp 111, entity_definitions_pkey 102, groups_live_key 51, 105 inner loops, buffers 743 | nested loop: spike_fx_num_2_grp 111, record_shares 3, record_shares_principal 3, 3 inner loops, buffers 348 | 0 / 0 |
| S-c1 shared slot ticket2, pk 40, a6 | **79.1** | 62.3 | 2497 | pass | 1.8 | 0.0 / 0.0 / 0.0 | nested loop: spike_fx_num_2_grp 138, entity_definitions_pkey 102, groups_live_key 51, 105 inner loops, buffers 766 | nested loop: spike_fx_num_2_grp 138, record_shares 3, record_shares_principal 3, 3 inner loops, buffers 360 | 0 / 0 |
| S-c1 shared slot ticket2, pk 3, no ACL | **72.3** | 54.3 | 2849 | pass | 2.1 | 0.0 / 0.0 / 0.0 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 1840 | join: spike_fx_num_2_grp 1001, buffers 581 | 0 / 0 |
| S-c1 shared slot tenant s1, pk 3, admin | **36.7** | 28.6 | 5431 | pass | 2.8 | 0.0 / 0.0 / 0.0 | nested loop: entity_definitions_number_prefix_key 13, groups_live_key 13, records_list_updated 13, 26 inner loops, buffers 1123 | - | 0 / 0 |

| Stage | Predicate | Estimate | Actual | Off by | S-c1 |
| --- | --- | --- | --- | --- | --- |
| own slot | large type, pk 3 (50%) | 200745 | 200000 | 1.0x | pass |
| own slot | large type, pk 40 (2%) | 8019 | 8000 | 1.0x | pass |
| own slot | large type, pk 3, a6's groups | 7301 | 15482 | 2.1x | pass |
| own slot | ticket2, pk 3 (2%, uniform) | 2147 | 2032 | 1.1x | pass |
| own slot | ticket2, pk 40 (2%, uniform) | 2248 | 2021 | 1.1x | pass |
| own slot | ticket2, pk 3, a6's groups | 78 | 111 | 1.4x | pass |
| shared slot | large type, pk 3 (50%) | 198930 | 200000 | 1.0x | pass |
| shared slot | large type, pk 40 (2%) | 8091 | 8000 | 1.0x | pass |
| shared slot | large type, pk 3, a6's groups | 7202 | 15482 | 2.1x | pass |
| shared slot | ticket2, pk 3 (2%, uniform) | 2149 | 2032 | 1.1x | pass |
| shared slot | ticket2, pk 40 (2%, uniform) | 2283 | 2021 | 1.1x | pass |
| shared slot | ticket2, pk 3, a6's groups | 78 | 111 | 1.4x | pass |
| shared slot | tenant s1's ticket, pk 3 (2%, uniform) | 1 | 13 | 13.0x | **fail** |

| Cell | own slot p95 | shared slot p95 | own page plan | shared page plan | S-c1 |
| --- | --- | --- | --- | --- | --- |
| large type, pk 3, a6 | 507.9 | 456.9 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 112925 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 112925 | **fail** |
| large type, pk 40, a6 | 115.5 | 109.8 | nested loop: spike_fx_num_2_grp 475, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1984 | nested loop: spike_fx_num_2_grp 475, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 1988 | pass |
| ticket2, pk 3, a6 | 108.9 | 78.5 | nested loop: spike_fx_own_grp 111, entity_definitions_pkey 102, groups_live_key 51, 105 inner loops, buffers 738 | nested loop: spike_fx_num_2_grp 111, entity_definitions_pkey 102, groups_live_key 51, 105 inner loops, buffers 743 | pass |
| ticket2, pk 40, a6 | 111.7 | 79.1 | nested loop: spike_fx_own_grp 138, entity_definitions_pkey 102, groups_live_key 51, 105 inner loops, buffers 762 | nested loop: spike_fx_num_2_grp 138, entity_definitions_pkey 102, groups_live_key 51, 105 inner loops, buffers 766 | pass |
| ticket2, pk 3, no ACL | 68.6 | 72.3 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 1816 | nested loop: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 51 inner loops, buffers 1840 | pass |

### Item 4: how many filled, indexed slots records carries (extra num slots fx_c_1..64, each filled for the large type and indexed with the group key)

- 64 extra columns added and filled for 400k records in 2m32s; records 2981 MB heap, 2061 MB indexes

| Extra slots (indexes on records) | insert p50 / p95 ms | insert WAL B/op | update unrelated p50 / p95 ms | update WAL B/op | list a6 pk 3 per-group merge p95 ms / page planning ms | list a6 no filter p95 ms | records indexes size |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 0 (18) | 47.8 / 62.9 | 16557 | 85.7 / 176.6 | 13855 | 119.3 / 5.35 | 75.1 | 2068 MB |
| 8 (26) | 59.1 / 78.6 | 15909 | 109.6 / 242.3 | 10251 | 120.7 / 5.93 | 75.6 | 2418 MB |
| 16 (34) | 77.0 / 99.5 | 26030 | 111.4 / 140.2 | 66280 | 120.9 / 6.80 | 87.9 | 2768 MB |
| 32 (50) | 95.0 / 131.5 | 21821 | 165.0 / 254.3 | 11965 | 170.2 / 8.96 | 121.1 | 3460 MB |
| 64 (82) | 160.4 / 192.6 | 55786 | 231.9 / 310.9 | 260548 | 144.0 / 8.38 | 95.3 | 4809 MB |

### S-b3: the large type at 2000000 records (4 copies of its 400k, same values, ACL and groups), group index only, line 150 ms

- 2616075 records of the large type after the copies (9m42s); a6 sees 160318; pk 3 visible to a6: 79083; records 4911 MB heap, 4364 MB indexes

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S-b3 2M, broad 50% (pk 3), a6, compiled (b)+group | **6762.4** | 4766.2 | 32 | **miss** | 1.7 | 0.0 / 1535.5 / 3.1 | nested loop: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, 105 inner loops, buffers 456738+274754 read | nested loop: spike_fx_num_2_grp 1001, buffers 996+15 read | 0 / 0 |
- compiled (b)+group: 0 slot-index scans, max 0 rows per scan, max full-Sort input 3, count Heap Fetches 0
| S-b3 2M, broad 50% (pk 3), a6, per-group merge | **105.3** | 85.0 | 1821 | pass | 2.0 | 0.0 / 13.4 / 5.9 | nested loop: spike_fx_num_2_grp 128, entity_definitions_pkey 51, groups_live_key 51, 105 inner loops, buffers 3937 | nested loop: spike_fx_num_2_grp 1001, buffers 1021 | 0 / 0 |
- per-group merge: 4 slot-index scans, max 56 rows per scan, max full-Sort input 3, count Heap Fetches 1
- transport floor after (SELECT 1, 8 clients): p50 2.58 ms, p95 3.68 ms: host gate (p50 <= 3 ms) passes; other containers: [nava-fiperf-pg-61858]

**Scores (line 150 ms, provisional within 10%):** S-b2 (i) broad 50% (pk 3), no ACL, group index only 290.0: **miss**; S-c2 num lt (score < 200, 2%), no ACL, group index only 63.5: pass; S-c2 num gte (score >= 9800, 2%), no ACL, group index only 111.0: pass; S-c2 num range (2000 <= score < 2200, 2%), no ACL, group index only 161.4: **miss** (provisional); S-c2 text eq (tier v2, 2%), no ACL, group index only 79.1: pass; S-c2 text in 3 values (tier v2, v025, v001, 2.26%), no ACL, group index only 82.4: pass; S-c2 num in 3 values (pk 10, 20, 30, 17%), no ACL, group index only 283.5: **miss**; S-b2 (ii) broad 50% (pk 3), no ACL, group + plain 386.0: **miss**; S-c2 num lt (score < 200, 2%), no ACL, group + plain 70.8: pass; S-c2 num gte (score >= 9800, 2%), no ACL, group + plain 130.6: pass; S-c2 num range (2000 <= score < 2200, 2%), no ACL, group + plain 127.0: pass; S-c2 text eq (tier v2, 2%), no ACL, group + plain 78.5: pass; S-c2 text in 3 values (tier v2, v025, v001, 2.26%), no ACL, group + plain 98.5: pass; S-c2 num in 3 values (pk 10, 20, 30, 17%), no ACL, group + plain 371.4: **miss**; S-c2 num lt (score < 200, 2%), a6, compiled 150.3: **miss** (provisional); S-c2 num lt (score < 200, 2%), a6, per-group merge 145.7: pass (provisional); S-c2 num lt (score < 200, 2%), a50, compiled 113.8: pass; S-c2 num lt (score < 200, 2%), a50, per-group merge 312.5: **miss**; S-c2 num gte (score >= 9800, 2%), a6, compiled 132.2: pass; S-c2 num gte (score >= 9800, 2%), a6, per-group merge 152.4: **miss** (provisional); S-c2 num gte (score >= 9800, 2%), a50, compiled 159.0: **miss** (provisional); S-c2 num gte (score >= 9800, 2%), a50, per-group merge 332.8: **miss**; S-c2 num range (2000 <= score < 2200, 2%), a6, compiled 148.3: pass (provisional); S-c2 num range (2000 <= score < 2200, 2%), a6, per-group merge 156.0: **miss** (provisional); S-c2 num range (2000 <= score < 2200, 2%), a50, compiled 182.4: **miss**; S-c2 num range (2000 <= score < 2200, 2%), a50, per-group merge 361.6: **miss**; S-c2 text eq (tier v2, 2%), a6, compiled 136.5: pass (provisional); S-c2 text eq (tier v2, 2%), a6, per-group merge 137.3: pass (provisional); S-c2 text eq (tier v2, 2%), a50, compiled 150.8: **miss** (provisional); S-c2 text eq (tier v2, 2%), a50, per-group merge 261.2: **miss**; S-c2 text in 3 values (tier v2, v025, v001, 2.26%), a6, compiled 136.8: pass (provisional); S-c2 text in 3 values (tier v2, v025, v001, 2.26%), a6, per-group merge 144.1: pass (provisional); S-c2 text in 3 values (tier v2, v025, v001, 2.26%), a50, compiled 200.0: **miss**; S-c2 text in 3 values (tier v2, v025, v001, 2.26%), a50, per-group merge 441.4: **miss**; S-c2 num in 3 values (pk 10, 20, 30, 17%), a6, compiled 586.3: **miss**; S-c2 num in 3 values (pk 10, 20, 30, 17%), a6, per-group merge 187.6: **miss**; S-c2 num in 3 values (pk 10, 20, 30, 17%), a50, compiled 611.8: **miss**; S-c2 num in 3 values (pk 10, 20, 30, 17%), a50, per-group merge 1170.8: **miss**; S-c3 num lt 10% (score < 1000), a6, compiled 140.6: pass (provisional); S-c3 num lt 10% (score < 1000), a6, per-group merge 138.1: pass (provisional); S-c3 num lt 10% (score < 1000), a50, compiled 122.6: pass; S-c3 num lt 10% (score < 1000), a50, per-group merge 275.3: **miss**; S-c3 num lt 20% (score < 2000), a6, compiled 131.8: pass; S-c3 num lt 20% (score < 2000), a6, per-group merge 145.5: pass (provisional); S-c3 num lt 20% (score < 2000), a50, compiled 122.9: pass; S-c3 num lt 20% (score < 2000), a50, per-group merge 290.0: **miss**; S-c3 text eq 10% (tier v10), a6, compiled 128.0: pass; S-c3 text eq 10% (tier v10), a6, per-group merge 113.6: pass; S-c3 text eq 10% (tier v10), a50, compiled 103.9: pass; S-c3 text eq 10% (tier v10), a50, per-group merge 245.9: **miss**; S-c3 text eq 20% (tier v20), a6, compiled 159.7: **miss** (provisional); S-c3 text eq 20% (tier v20), a6, per-group merge 111.4: pass; S-c3 text eq 20% (tier v20), a50, compiled 89.2: pass; S-c3 text eq 20% (tier v20), a50, per-group merge 229.4: **miss**; S-c1 shared slot large type, pk 3, a6 456.9: **miss**; S-c1 shared slot large type, pk 40, a6 109.8: pass; S-c1 shared slot ticket2, pk 3, a6 78.5: pass; S-c1 shared slot ticket2, pk 40, a6 79.1: pass; S-c1 shared slot ticket2, pk 3, no ACL 72.3: pass; S-c1 shared slot tenant s1, pk 3, admin 36.7: pass; S-b3 2M, broad 50% (pk 3), a6, compiled (b)+group 6762.4: **miss**; S-b3 2M, broad 50% (pk 3), a6, per-group merge 105.3: pass.
