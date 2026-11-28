# TICK-747 field-index measurements, raw (2026-11-26T05:45:30Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 4m16s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app plan_cache_mode: plan_cache_mode=force_custom_plan
- other containers at start: [nava-fiperf-pg-20143]
- transport floor (SELECT 1, 8 clients): p50 3.59 ms, p95 6.69 ms
- 8 fields flagged on all 400 ticket types; side rows copied in 6m49s; ANALYZE: rfv_text 302ms, rfv_num 199ms, rfv_ts 136ms, rfv_bool 12.761s
- side tables: rfv_bool 3200000 rows, 258 MB heap, 617 MB indexes; rfv_num 800000 rows, 71 MB heap, 152 MB indexes; rfv_text 1600000 rows, 129 MB heap, 307 MB indexes; rfv_ts 800000 rows, 71 MB heap, 153 MB indexes
- records: 1250 MB heap, 755 MB indexes

- breadth agents (agent-1's principal), records of the large type each sees through any ACL branch:
  - agent-1: 3 groups, 24666 records (6.2%)
  - a25-1: 13 groups, 104490 records (26.1%)
  - a50-1: 25 groups, 200402 records (50.1%)

## TICK-1039: arms D7 / (a) adaptive / (b) promoted column, 8 clients, 2s per cell, work_mem 4MB

- pk fixture (value: side rows, visible to a6 / a50): 3: 200000, 15819 / 133409; 4: 123921, 7869 / 62488; 10: 26593, 0 / 0; 20: 21487, 494 / 494; 30: 19999, 0 / 0; 40: 8000, 484 / 4011
- 40's updated_at: 2025-10-01 00:03:12.330002+00 .. 2025-10-22 10:04:20.605855+00; the type's: 2025-10-01 00:03:12.330002+00 .. 2026-10-01 00:00:00+00
- fixture, side rows, scratch column, VACUUM FULL records and prewarm in 3m11s; records: 1250 MB heap, 572 MB indexes
- transport floor before (SELECT 1, 8 clients): p50 5.66 ms, p95 21.41 ms: host gate (p50 <= 3 ms) **fails**; other containers: [nava-fiperf-pg-20143]
- control a6, no filter: p95 178.2 (p50 95.1, n 154)
- control a50, no filter: p95 189.6 (p50 96.8, n 150)
- control no ACL, no filter: p95 93.7 (p50 67.9, n 231)

### Arm D7 (line 150 ms)

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| D7 A1-6 (3) anti-correlated 2% (8,000 oldest), a6 | **333.7** | 287.1 | 64 | **miss** | 3.8 | 0.0 / 109.3 / 45.3 | hash: records_list_requester 513, records 484, records_list_assignee 61, 8169 inner loops, buffers 33453 | hash: records_list_group 24118, rfv_num_value 16000, records_list_requester 513, 67 inner loops, buffers 17012 | 0 / 0 |
| D7 A1-6 (3) anti-correlated 2% (8,000 oldest), a50 | **1079.0** | 312.8 | 32 | **miss** | 3.5 | 0.0 / 190.1 / 146.3 | hash: records_pkey 8000, records_list_requester 513, records 256, 8137 inner loops, buffers 33373 | hash: records_list_group 56141, rfv_num_value 8000, buffers 37181 | 0 / 0 |
| D7 A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL | **769.1** | 480.8 | 32 | **miss** | 5.4 | 0.0 / 139.3 / 148.4 | nested loop: records_pkey 8000, groups 53, groups_keycloak_group_key 53, 8000 inner loops, buffers 32195 | hash: records_list_created 393300, rfv_num_value 24000, 6 inner loops, buffers 258755 | 0 / 0 |
| D7 none visible (26,593), a6 | **215.8** | 181.6 | 91 | **miss** | 4.1 | 0.0 / 107.4 / 0.0 | hash: records 24118, records_list_group 24118, records_list_requester 484, 551 inner loops, buffers 24509 | - | 0 / 0 |
| D7 none visible (26,593), a50 | **3516.4** | 3462.3 | 8 | **miss** | 3.6 | 0.0 / 2337.8 / 0.0 | nested loop: records_list_updated 200113, records_list_requester 256, records_list_assignee 32, 200404 inner loops, buffers 1206004 | - | 0 / 0 |
| D7 partly correlated (494 of 21,487 visible to a6), a6 | **435.5** | 319.4 | 55 | **miss** | 6.2 | 0.0 / 101.9 / 48.3 | hash: records 24602, records_list_group 24118, records_list_requester 513, 120 inner loops, buffers 22903 | hash: rfv_num_value 42973, records_list_group 24118, records_list_requester 513, 67 inner loops, buffers 17465 | 0 / 0 |
| D7 partly correlated (494 of 21,487 visible to a6), a50 | **611.5** | 497.3 | 35 | **miss** | 8.4 | 0.0 / 186.6 / 167.0 | nested loop: records_list_requester 513, records 256, entity_definitions_pkey 51, 21880 inner loops, buffers 88433 | hash: records_list_group 200113, rfv_num_value 21487, records_list_requester 513, 291 inner loops, buffers 133866 | 0 / 0 |
| D7 19,999 invisible side rows, a6 | **212.1** | 181.4 | 94 | **miss** | 6.8 | 0.0 / 94.8 / 0.0 | hash: records 24602, records_list_group 24118, records_list_requester 513, 67 inner loops, buffers 22793 | - | 0 / 0 |
| D7 19,999 invisible side rows, a50 | **354.0** | 305.7 | 56 | **miss** | 3.8 | 0.0 / 216.5 / 0.0 | nested loop: records_list_requester 513, records 256, records_list_assignee 32, 20290 inner loops, buffers 82149 | - | 0 / 0 |
| D7 broad 50% (200,000), a6 | **571.6** | 499.9 | 35 | **miss** | 4.3 | 0.0 / 148.1 / 44.0 | nested loop: records_list_group 24118, records 24117, rfv_num_value 79, 188 inner loops, buffers 21948 | nested loop: records_list_group 24118, rfv_num_value 14080, buffers 16110 | 0 / 0 |
| D7 broad 50% (200,000), a50 | **314.0** | 276.6 | 63 | **miss** | 3.6 | 0.0 / 8.7 / 239.6 | nested loop: rfv_num_value 77, records_list_updated 74, entity_definitions_pkey 51, 179 inner loops, buffers 874 | hash+spill: rfv_num_value 200000, records_list_group 2906, buffers 5293 | 0 / 0 |

### Arm (a) (line 150 ms)

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| (a) A1-6 (3) anti-correlated 2% (8,000 oldest), a6 | **1996.5** | 1855.9 | 16 | **miss** | 5.9 | 3.4 / 1199.1 / 197.0 | probe: entity_definitions_pkey 51, groups_pkey 51, records_list_updated 49, 24339 inner loops, buffers 494453 | probe: records_list_group 475, records_list_requester 9, record_shares 3, 24669 inner loops, buffers 115149 | 0 / 0 |
| (a) A1-6 (3) anti-correlated 2% (8,000 oldest), a50 | **4051.8** | 3560.5 | 8 | **miss** | 5.2 | 3.4 / 1837.5 / 318.3 | probe: entity_definitions_pkey 51, groups_pkey 51, records_list_updated 51, 196547 inner loops, buffers 1182499 | probe: records_list_group 1001, rfv_num_value 0, 56141 inner loops, buffers 261609 | 0 / 0 |
| (a) A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL | **5408.8** | 5075.9 | 8 | **miss** | 5.1 | 3.0 / 2421.9 / 3178.7 | probe: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, 392102 inner loops, buffers 1964076 | probe: records_list_created 1001, rfv_num_value 0, 391042 inner loops, buffers 1820951 | 0 / 0 |
| (a) none visible (26,593), a6 | **1906.0** | 1545.4 | 16 | **miss** | 5.5 | 4.2 / 1236.8 / 0.0 | probe: record_shares 3, record_shares_principal 3, records_list_assignee 0, 24669 inner loops, buffers 503016 | - | 0 / 0 |
| (a) none visible (26,593), a50 | **5257.6** | 3897.2 | 8 | **miss** | 4.0 | 3.7 / 2839.7 / 0.0 | probe: record_shares 3, record_shares_principal 3, records_list_assignee 0, 200404 inner loops, buffers 1206004 | - | 0 / 0 |
| (a) partly correlated (494 of 21,487 visible to a6), a6 | **653.2** | 506.7 | 32 | **miss** | 4.3 | 4.4 / 100.7 / 234.1 | probe: entity_definitions_pkey 51, groups_pkey 51, records_list_updated 47, 2823 inner loops, buffers 56812 | probe: records_list_group 479, records_list_requester 14, record_shares 3, 24669 inner loops, buffers 115150 | 0 / 0 |
| (a) partly correlated (494 of 21,487 visible to a6), a50 | **867.2** | 775.3 | 24 | **miss** | 4.7 | 3.5 / 300.5 / 1368.8 | probe: entity_definitions_pkey 51, groups_pkey 51, records_list_updated 49, 22670 inner loops, buffers 136354 | probe: records_list_group 487, records_list_requester 7, record_shares 3, 200404 inner loops, buffers 933957 | 0 / 0 |
| (a) 19,999 invisible side rows, a6 | **2313.2** | 2074.6 | 8 | **miss** | 3.9 | 4.8 / 1375.3 / 0.0 | probe: record_shares 3, record_shares_principal 3, records_list_assignee 0, 24669 inner loops, buffers 503016 | - | 0 / 0 |
| (a) 19,999 invisible side rows, a50 | **4140.4** | 3531.4 | 8 | **miss** | 5.3 | 2.5 / 1579.2 / 0.0 | probe: record_shares 3, record_shares_principal 3, records_list_assignee 0, 200404 inner loops, buffers 1206004 | - | 0 / 0 |
| (a) broad 50% (200,000), a6 | **198.0** | 119.3 | 131 | **miss** | 4.7 | 2.3 / 8.8 / 22.2 | probe: rfv_num_value 79, entity_definitions_pkey 51, groups_pkey 51, 185 inner loops, buffers 1856 | probe: rfv_num_value 1528, records_list_group 1001, 1528 inner loops, buffers 7111 | 0 / 0 |
| (a) broad 50% (200,000), a50 | **586.1** | 381.1 | 41 | **miss** | 3.5 | 4.0 / 8.2 / 34.4 | probe: rfv_num_value 77, entity_definitions_pkey 51, groups_pkey 51, 179 inner loops, buffers 874 | probe: rfv_num_value 1441, records_list_group 1001, 1441 inner loops, buffers 6731 | 0 / 0 |

(a)'s picks over the arm's requests: map[walk + pkey probe:322] (pre-count cap 4500)

- (b) index: 52 MB

### Arm (b) (line 150 ms)

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| (b) A1-6 (3) anti-correlated 2% (8,000 oldest), a6 | **319.2** | 156.1 | 94 | **miss** | 6.2 | 0.0 / 8.7 / 14.4 | nested loop: spike_records_pk 52, entity_definitions_pkey 51, groups_pkey 51, 105 inner loops, buffers 1902 | nested loop: spike_records_pk 8000, records_list_requester 513, records 484, 3 inner loops, buffers 2287 | 0 / 0 |
| (b) A1-6 (3) anti-correlated 2% (8,000 oldest), a50 | **279.1** | 133.4 | 103 | **miss** | 3.9 | 0.0 / 5.5 / 7.3 | nested loop: spike_records_pk 52, entity_definitions_pkey 51, groups_pkey 51, 105 inner loops, buffers 1067 | nested loop: spike_records_pk 8000, records 1001, buffers 471 | 0 / 0 |
| (b) A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL | **166.3** | 107.2 | 149 | **miss** | 4.2 | 0.0 / 2.3 / 1.6 | nested loop: groups 53, groups_keycloak_group_key 53, spike_records_pk 52, buffers 112 | join: spike_records_pk 1001, buffers 644 | 0 / 0 |
| (b) none visible (26,593), a6 | **232.8** | 133.7 | 112 | **miss** | 4.8 | 0.0 / 68.3 / 0.0 | nested loop: record_shares 3, record_shares_principal 3, records_list_assignee 0, 3 inner loops, buffers 27647 | - | 0 / 0 |
| (b) none visible (26,593), a50 | **192.6** | 154.1 | 104 | **miss** | 4.7 | 0.0 / 67.7 / 0.0 | nested loop: record_shares 3, record_shares_principal 3, records_list_assignee 0, 3 inner loops, buffers 27691 | - | 0 / 0 |
| (b) partly correlated (494 of 21,487 visible to a6), a6 | **264.6** | 217.6 | 78 | **miss** | 6.6 | 0.0 / 13.1 / 36.1 | nested loop: spike_records_pk 52, entity_definitions_pkey 51, groups_pkey 51, 105 inner loops, buffers 3074 | nested loop: spike_records_pk 21487, records_list_requester 513, records 493, 3 inner loops, buffers 13281 | 0 / 0 |
| (b) partly correlated (494 of 21,487 visible to a6), a50 | **355.1** | 265.8 | 60 | **miss** | 5.5 | 0.0 / 10.5 / 45.8 | nested loop: spike_records_pk 52, entity_definitions_pkey 51, groups_pkey 51, 105 inner loops, buffers 3197 | nested loop: spike_records_pk 21487, records_list_requester 513, records 494, 3 inner loops, buffers 13325 | 0 / 0 |
| (b) 19,999 invisible side rows, a6 | **188.3** | 130.1 | 119 | **miss** | 6.1 | 0.0 / 48.9 / 0.0 | nested loop: record_shares 3, record_shares_principal 3, records_list_assignee 0, 3 inner loops, buffers 20944 | - | 0 / 0 |
| (b) 19,999 invisible side rows, a50 | **231.4** | 141.8 | 110 | **miss** | 5.9 | 0.0 / 54.6 / 0.0 | nested loop: record_shares 3, record_shares_principal 3, records_list_assignee 0, 3 inner loops, buffers 20988 | - | 0 / 0 |
| (b) broad 50% (200,000), a6 | **235.2** | 165.0 | 94 | **miss** | 6.3 | 0.0 / 5.8 / 24.1 | nested loop: spike_records_pk 52, entity_definitions_pkey 51, groups_pkey 51, 105 inner loops, buffers 932 | nested loop: records_list_group 24118, records 1001, buffers 1799 | 0 / 0 |
| (b) broad 50% (200,000), a50 | **468.1** | 339.5 | 48 | **miss** | 5.7 | 0.0 / 4.9 / 97.5 | nested loop: spike_records_pk 52, entity_definitions_pkey 51, groups_pkey 51, 105 inner loops, buffers 467 | nested loop: spike_records_pk 200000, records 1001, buffers 2188 | 0 / 0 |

- (b)+group index: 66 MB

### Arm (b)+group (line 150 ms)

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| (b)+group A1-6 (3) anti-correlated 2% (8,000 oldest), a6 | **249.5** | 171.8 | 91 | **miss** | 6.3 | 0.0 / 10.8 / 4.4 | nested loop: records 475, spike_records_pk_group 475, entity_definitions_pkey 51, 105 inner loops, buffers 1359 | nested loop: records_list_requester 513, spike_records_pk_group 475, records 9, 3 inner loops, buffers 933 | 0 / 0 |
| (b)+group A1-6 (3) anti-correlated 2% (8,000 oldest), a50 | **566.7** | 256.5 | 63 | **miss** | 6.7 | 0.0 / 42.2 / 4.4 | nested loop: records 4011, spike_records_pk_group 4004, records_list_requester 513, 105 inner loops, buffers 2664 | nested loop: spike_records_pk_group 1001, buffers 661 | 0 / 0 |
| (b)+group A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL | **326.9** | 217.9 | 73 | **miss** | 6.2 | 0.0 / 44.8 / 1.6 | nested loop: records 8000, spike_records_pk_group 8000, groups 53, buffers 1747 | join: spike_records_pk_group 1001, buffers 669 | 0 / 0 |
| (b)+group none visible (26,593), a6 | **141.7** | 89.7 | 170 | pass (provisional) | 9.7 | 0.0 / 5.2 / 0.0 | nested loop: record_shares 3, record_shares_principal 3, records 0, 3 inner loops, buffers 632 | - | 0 / 0 |
| (b)+group none visible (26,593), a50 | **150.6** | 79.3 | 178 | **miss** (provisional) | 7.5 | 0.0 / 8.2 / 0.0 | nested loop: records_list_requester 513, record_shares 3, record_shares_principal 3, 3 inner loops, buffers 763 | - | 0 / 0 |
| (b)+group partly correlated (494 of 21,487 visible to a6), a6 | **398.3** | 176.8 | 85 | **miss** | 6.9 | 0.0 / 8.2 / 4.8 | nested loop: records 479, spike_records_pk_group 479, entity_definitions_pkey 51, 105 inner loops, buffers 901 | nested loop: records_list_requester 513, spike_records_pk_group 479, records 14, 3 inner loops, buffers 930 | 0 / 0 |
| (b)+group partly correlated (494 of 21,487 visible to a6), a50 | **241.8** | 174.9 | 91 | **miss** | 7.7 | 0.0 / 11.3 / 6.8 | nested loop: records_list_requester 513, records 494, spike_records_pk_group 487, 105 inner loops, buffers 1547 | nested loop: records_list_requester 513, spike_records_pk_group 487, records 7, 3 inner loops, buffers 1067 | 0 / 0 |
| (b)+group 19,999 invisible side rows, a6 | **177.8** | 103.7 | 148 | **miss** | 8.9 | 0.0 / 5.4 / 0.0 | nested loop: record_shares 3, record_shares_principal 3, records 0, 3 inner loops, buffers 632 | - | 0 / 0 |
| (b)+group 19,999 invisible side rows, a50 | **100.1** | 61.9 | 242 | pass | 6.0 | 0.0 / 5.6 / 0.0 | nested loop: records_list_requester 513, record_shares 3, record_shares_principal 3, 3 inner loops, buffers 763 | - | 0 / 0 |
| (b)+group broad 50% (200,000), a6 | **354.0** | 265.6 | 64 | **miss** | 4.1 | 0.0 / 93.1 / 4.4 | nested loop: records 15780, spike_records_pk_group 15482, records_list_requester 513, 105 inner loops, buffers 15125 | nested loop: spike_records_pk_group 1001, buffers 686 | 0 / 0 |
| (b)+group broad 50% (200,000), a50 | **168.7** | 93.3 | 159 | **miss** | 5.5 | 0.0 / 3.3 / 2.8 | nested loop: entity_definitions_pkey 51, groups_pkey 51, records_list_updated 51, 105 inner loops, buffers 562 | nested loop: spike_records_pk_group 1001, buffers 670 | 0 / 0 |
- transport floor after (SELECT 1, 8 clients): p50 4.28 ms, p95 5.78 ms: host gate (p50 <= 3 ms) **fails**; other containers: [nava-fiperf-pg-20143]

### Summary: p95 ms by arm (line 150 ms, provisional within 10%)

| Fixture | D7 | (a) | (b) | (b)+group |
| --- | --- | --- | --- | --- |
| A1-6 (3) anti-correlated 2% (8,000 oldest), a6 | 333.7 **miss** | 1996.5 **miss** | 319.2 **miss** | 249.5 **miss** |
| A1-6 (3) anti-correlated 2% (8,000 oldest), a50 | 1079.0 **miss** | 4051.8 **miss** | 279.1 **miss** | 566.7 **miss** |
| A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL | 769.1 **miss** | 5408.8 **miss** | 166.3 **miss** | 326.9 **miss** |
| none visible (26,593), a6 | 215.8 **miss** | 1906.0 **miss** | 232.8 **miss** | 141.7 pass (provisional) |
| none visible (26,593), a50 | 3516.4 **miss** | 5257.6 **miss** | 192.6 **miss** | 150.6 **miss** (provisional) |
| partly correlated (494 of 21,487 visible to a6), a6 | 435.5 **miss** | 653.2 **miss** | 264.6 **miss** | 398.3 **miss** |
| partly correlated (494 of 21,487 visible to a6), a50 | 611.5 **miss** | 867.2 **miss** | 355.1 **miss** | 241.8 **miss** |
| 19,999 invisible side rows, a6 | 212.1 **miss** | 2313.2 **miss** | 188.3 **miss** | 177.8 **miss** |
| 19,999 invisible side rows, a50 | 354.0 **miss** | 4140.4 **miss** | 231.4 **miss** | 100.1 pass |
| broad 50% (200,000), a6 | 571.6 **miss** | 198.0 **miss** | 235.2 **miss** | 354.0 **miss** |
| broad 50% (200,000), a50 | 314.0 **miss** | 586.1 **miss** | 468.1 **miss** | 168.7 **miss** |

- D7 misses 11 of 11: A1-6 (3) anti-correlated 2% (8,000 oldest), a6; A1-6 (3) anti-correlated 2% (8,000 oldest), a50; A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL; none visible (26,593), a6; none visible (26,593), a50; partly correlated (494 of 21,487 visible to a6), a6; partly correlated (494 of 21,487 visible to a6), a50; 19,999 invisible side rows, a6; 19,999 invisible side rows, a50; broad 50% (200,000), a6; broad 50% (200,000), a50
- (a) misses 11 of 11: A1-6 (3) anti-correlated 2% (8,000 oldest), a6; A1-6 (3) anti-correlated 2% (8,000 oldest), a50; A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL; none visible (26,593), a6; none visible (26,593), a50; partly correlated (494 of 21,487 visible to a6), a6; partly correlated (494 of 21,487 visible to a6), a50; 19,999 invisible side rows, a6; 19,999 invisible side rows, a50; broad 50% (200,000), a6; broad 50% (200,000), a50
- (b) misses 11 of 11: A1-6 (3) anti-correlated 2% (8,000 oldest), a6; A1-6 (3) anti-correlated 2% (8,000 oldest), a50; A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL; none visible (26,593), a6; none visible (26,593), a50; partly correlated (494 of 21,487 visible to a6), a6; partly correlated (494 of 21,487 visible to a6), a50; 19,999 invisible side rows, a6; 19,999 invisible side rows, a50; broad 50% (200,000), a6; broad 50% (200,000), a50
- (b)+group misses 9 of 11: A1-6 (3) anti-correlated 2% (8,000 oldest), a6; A1-6 (3) anti-correlated 2% (8,000 oldest), a50; A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL; none visible (26,593), a50; partly correlated (494 of 21,487 visible to a6), a6; partly correlated (494 of 21,487 visible to a6), a50; 19,999 invisible side rows, a6; broad 50% (200,000), a6; broad 50% (200,000), a50
