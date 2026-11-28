# TICK-747 field-index measurements, raw (2026-11-26T05:18:04Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 5m23s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app plan_cache_mode: plan_cache_mode=force_custom_plan
- other containers at start: [nava-fiperf-pg-16958]
- transport floor (SELECT 1, 8 clients): p50 8.23 ms, p95 21.57 ms
- 8 fields flagged on all 400 ticket types; side rows copied in 7m54s; ANALYZE: rfv_text 282ms, rfv_num 192ms, rfv_ts 192ms, rfv_bool 11.969s
- side tables: rfv_bool 3200000 rows, 258 MB heap, 618 MB indexes; rfv_num 800000 rows, 71 MB heap, 152 MB indexes; rfv_text 1600000 rows, 129 MB heap, 308 MB indexes; rfv_ts 800000 rows, 71 MB heap, 152 MB indexes
- records: 1250 MB heap, 755 MB indexes

- breadth agents (agent-1's principal), records of the large type each sees through any ACL branch:
  - agent-1: 3 groups, 24666 records (6.2%)
  - a25-1: 13 groups, 104490 records (26.1%)
  - a50-1: 25 groups, 200402 records (50.1%)

## TICK-1009: S1 (band edge) and S2 (ACL breadth), 8 clients, 2s per cell

- tier shares, all 800k records (TICK-992's v40 / v2 / v025 / v001 unchanged): v40 321384 (40.173%), others 181122 (22.640%), v20 159568 (19.946%), v10 79705 (9.963%), v5 40112 (5.014%), v2 15993 (1.999%), v025 2033 (0.254%), v001 83 (0.010%)
- large type matches: score<1000 40108, score<200 8098, score<2000 79979, score<500 20292, tier=v10 39642, tier=v2 7980, tier=v20 79871, tier=v5 20087
- transport floor before (SELECT 1, 8 clients): p50 3.79 ms, p95 7.62 ms: host gate (p50 <= 3 ms) **fails**; other containers: [nava-fiperf-pg-16958]

Control before: ACL p95 165.1 (p50 106.4, n 144) / no ACL p95 230.8 (p50 101.1, n 137)

### S1, band edge (ACL: the agents, 6% visible; no ACL: the admin)

Score line: 5% cells against 350 ms (a pass with a fan-out plan extends band (i) to 5%); 10% and 20% against 150 ms (over 150 ms is a new band: reported, D13 unchanged).

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S1 num lt 5%, ACL | **505.7** | 298.9 | 51 | **miss** | 2.8 | 0.0 / 63.2 / 19.7 | hash: records 24118, records_list_group 24118, rfv_num_value 20292, buffers 22642 | hash: records_list_group 20406, rfv_num_value 20292, buffers 13565 | 0 / 0 |
| S1 num lt 5%, no ACL | **642.1** | 595.4 | 32 | **miss** | 2.9 | 0.0 / 206.3 / 21.4 | nested loop: records_pkey 20292, rfv_num_value 20292, groups 53, buffers 81469 | hash: rfv_num_value 20292, records_list_updated 19835, buffers 13126 | 0 / 0 |
| S1 num lt 10%, ACL | **391.5** | 316.5 | 53 | **miss** | 4.5 | 0.0 / 71.7 / 29.1 | hash: rfv_num_value 40108, records 24118, records_list_group 24118, buffers 22857 | hash: rfv_num_value 40108, records_list_group 9991, buffers 7034 | 0 / 0 |
| S1 num lt 10%, no ACL | **222.4** | 137.6 | 110 | **miss** | 3.0 | 0.0 / 6.6 / 29.3 | nested loop: records_list_updated 446, groups_pkey 51, entity_definitions_number_prefix_key 1, buffers 2395 | hash: rfv_num_value 40108, records_list_updated 9799, buffers 6842 | 0 / 0 |
| S1 num lt 20%, ACL | **743.9** | 599.8 | 30 | **miss** | 3.1 | 0.0 / 127.1 / 58.2 | nested loop: records_list_group 24118, records 24117, entity_definitions_pkey 51, buffers 23121 | hash: rfv_num_value 79979, records_list_group 5128, buffers 4300 | 0 / 0 |
| S1 num lt 20%, no ACL | **509.9** | 293.3 | 52 | **miss** | 5.3 | 0.0 / 3.6 / 72.7 | nested loop: records_list_updated 236, groups_pkey 51, entity_definitions_number_prefix_key 1, buffers 1342 | hash: rfv_num_value 79979, records_list_updated 4983, buffers 4207 | 0 / 0 |
| S1 text eq 5%, ACL | **466.5** | 327.4 | 51 | **miss** | 6.6 | 0.0 / 76.8 / 21.5 | hash: records 24118, records_list_group 24118, entity_definitions_pkey 51, buffers 22653 | hash: rfv_text_value 20087, records_list_group 19401, buffers 12919 | 0 / 0 |
| S1 text eq 5%, no ACL | **714.7** | 616.4 | 32 | **miss** | 3.8 | 0.0 / 192.0 / 19.7 | nested loop: records_pkey 20087, groups 53, groups_live_key 53, buffers 80657 | hash: rfv_text_value 20087, records_list_updated 18680, buffers 12378 | 0 / 0 |
| S1 text eq 10%, ACL | **444.7** | 317.7 | 48 | **miss** | 4.1 | 0.0 / 78.9 / 33.8 | hash: records 24118, records_list_group 24118, entity_definitions_pkey 51, buffers 22849 | nested loop: records_list_group 24118, rfv_text_value 16798, buffers 15893 | 0 / 0 |
| S1 text eq 10%, no ACL | **257.9** | 166.2 | 90 | **miss** | 4.1 | 0.0 / 6.7 / 37.3 | nested loop: records_list_updated 503, groups_pkey 51, entity_definitions_number_prefix_key 1, buffers 2681 | hash: rfv_text_value 39642, records_list_updated 10318, buffers 7186 | 0 / 0 |
| S1 text eq 20%, ACL | **1327.7** | 547.9 | 30 | **miss** | 4.2 | 0.0 / 125.7 / 34.9 | nested loop: records_list_group 24118, records 24117, entity_definitions_pkey 51, buffers 22998 | nested loop: records_list_group 24118, rfv_text_value 16994, buffers 15897 | 0 / 0 |
| S1 text eq 20%, no ACL | **290.1** | 188.8 | 82 | **miss** | 5.1 | 0.0 / 3.4 / 38.9 | nested loop: records_list_updated 219, groups_pkey 51, entity_definitions_number_prefix_key 1, buffers 1258 | hash: rfv_text_value 79871, records_list_updated 5087, buffers 4293 | 0 / 0 |

### S2, ACL breadth, 2% (line 350 ms)

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S2 num lt 2%, agents seeing 6% (control) | **424.6** | 283.3 | 58 | **miss** | 4.3 | 0.0 / 75.8 / 26.9 | hash: records 24602, records_list_group 24118, rfv_num_value 16196, buffers 23039 | hash: records_list_group 24118, rfv_num_value 16196, records_list_requester 513, buffers 16744 | 0 / 0 |
| S2 num lt 2%, agents seeing 26% | **333.5** | 301.3 | 56 | pass (provisional) | 3.7 | 0.0 / 74.6 / 33.4 | hash: rfv_num_value 16196, records_list_requester 513, records 382, buffers 33634 | hash: records_list_group 48853, rfv_num_value 8098, buffers 31864 | 0 / 0 |
| S2 num lt 2%, agents seeing 50% | **291.7** | 229.3 | 72 | pass | 3.8 | 0.0 / 28.7 / 35.0 | nested loop: records_list_updated 2303, records_list_requester 59, entity_definitions_pkey 51, buffers 14761 | hash: records_list_group 48383, rfv_num_value 8098, buffers 31505 | 0 / 0 |
| S2 text eq 2%, agents seeing 6% (control) | **522.9** | 296.9 | 54 | **miss** | 4.5 | 0.0 / 45.3 / 12.9 | hash: records 24118, records_list_group 24118, records_list_requester 90, buffers 23019 | hash: records_list_group 24118, rfv_text_value 7980, records_list_requester 484, buffers 18596 | 0 / 0 |
| S2 text eq 2%, agents seeing 26% | **344.6** | 255.0 | 64 | pass (provisional) | 11.8 | 0.0 / 62.3 / 28.8 | nested loop: records_list_requester 68, records_list_assignee 52, entity_definitions_pkey 51, buffers 33034 | hash: records_list_group 48513, rfv_text_value 7980, buffers 31640 | 0 / 0 |
| S2 text eq 2%, agents seeing 50% | **354.0** | 211.0 | 72 | **miss** (provisional) | 3.8 | 0.0 / 33.5 / 26.7 | nested loop: records_list_updated 2480, records_list_requester 80, entity_definitions_pkey 51, buffers 15959 | hash: records_list_group 49906, rfv_text_value 7980, buffers 32503 | 0 / 0 |

Control after: ACL p95 156.1 / no ACL p95 148.8
- transport floor after (SELECT 1, 8 clients): p50 3.82 ms, p95 5.47 ms: host gate (p50 <= 3 ms) **fails**; other containers: [nava-fiperf-pg-16958]

**Scores:** S1 num lt 5%, ACL 505.7 vs 350: **miss**; S1 num lt 5%, no ACL 642.1 vs 350: **miss**; S1 num lt 10%, ACL 391.5 vs 150: **miss**; S1 num lt 10%, no ACL 222.4 vs 150: **miss**; S1 num lt 20%, ACL 743.9 vs 150: **miss**; S1 num lt 20%, no ACL 509.9 vs 150: **miss**; S1 text eq 5%, ACL 466.5 vs 350: **miss**; S1 text eq 5%, no ACL 714.7 vs 350: **miss**; S1 text eq 10%, ACL 444.7 vs 150: **miss**; S1 text eq 10%, no ACL 257.9 vs 150: **miss**; S1 text eq 20%, ACL 1327.7 vs 150: **miss**; S1 text eq 20%, no ACL 290.1 vs 150: **miss**; S2 num lt 2%, agents seeing 6% (control) 424.6 vs 350: **miss**; S2 num lt 2%, agents seeing 26% 333.5 vs 350: pass (provisional); S2 num lt 2%, agents seeing 50% 291.7 vs 350: pass; S2 text eq 2%, agents seeing 6% (control) 522.9 vs 350: **miss**; S2 text eq 2%, agents seeing 26% 344.6 vs 350: pass (provisional); S2 text eq 2%, agents seeing 50% 354.0 vs 350: **miss** (provisional).

**S2:** num lt: highest passing breadth none; text eq: highest passing breadth none.

## TICK-1039: arms D7 / (a) adaptive / (b) promoted column, 8 clients, 2s per cell, work_mem 4MB

- pk fixture (value: side rows, visible to a6 / a50): 3: 200000, 15819 / 133409; 4: 123921, 7869 / 62488; 10: 26593, 0 / 0; 20: 21487, 494 / 494; 30: 19999, 0 / 0; 40: 8000, 484 / 4011
- 40's updated_at: 2025-10-01 00:03:12.330002+00 .. 2025-10-22 10:04:20.605855+00; the type's: 2025-10-01 00:03:12.330002+00 .. 2026-10-01 00:00:00+00
- fixture, side rows, scratch column, VACUUM FULL records and prewarm in 2m57s; records: 1250 MB heap, 572 MB indexes
- transport floor before (SELECT 1, 8 clients): p50 4.23 ms, p95 9.44 ms: host gate (p50 <= 3 ms) **fails**; other containers: [nava-fiperf-pg-16958]
- control a6, no filter: p95 216.9 (p50 155.6, n 104)
- control a50, no filter: p95 266.0 (p50 146.4, n 107)
- control no ACL, no filter: p95 187.2 (p50 123.3, n 129)

### Arm D7 (line 150 ms)

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| D7 A1-6 (3) anti-correlated 2% (8,000 oldest), a6 | **313.7** | 260.6 | 65 | **miss** | 3.2 | 0.0 / 77.5 / 33.8 | hash: records_list_requester 513, records 484, records_list_assignee 61, buffers 33454 | hash: records_list_group 24118, rfv_num_value 16000, records_list_requester 513, buffers 17013 | 0 / 0 |
| D7 A1-6 (3) anti-correlated 2% (8,000 oldest), a50 | **439.3** | 296.3 | 54 | **miss** | 3.2 | 0.0 / 60.9 / 31.3 | hash: records_pkey 8000, records_list_requester 513, records 256, buffers 33374 | hash: records_list_group 56141, rfv_num_value 8000, buffers 37183 | 0 / 0 |
| D7 A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL | **468.2** | 411.2 | 40 | **miss** | 3.3 | 0.0 / 77.7 / 107.2 | nested loop: records_pkey 8000, groups 53, groups_live_key 53, buffers 32195 | hash: records_list_created 392850, rfv_num_value 24000, buffers 258451 | 0 / 0 |
| D7 none visible (26,593), a6 | **228.3** | 176.5 | 89 | **miss** | 3.8 | 0.0 / 88.9 / 0.0 | hash: records 24118, records_list_group 24118, records_list_requester 484, buffers 24503 | - | 0 / 0 |
| D7 none visible (26,593), a50 | **3442.6** | 3423.7 | 8 | **miss** | 3.4 | 0.0 / 1197.2 / 0.0 | nested loop: records_list_updated 200113, records_list_requester 256, records_list_assignee 32, buffers 1206005 | - | 0 / 0 |
| D7 partly correlated (494 of 21,487 visible to a6), a6 | **322.9** | 248.0 | 67 | **miss** | 3.2 | 0.0 / 57.3 / 31.5 | hash: records 24118, records_list_group 24118, records_list_requester 60, buffers 22326 | hash: rfv_num_value 42973, records_list_group 24118, records_list_requester 513, buffers 17464 | 0 / 0 |
| D7 partly correlated (494 of 21,487 visible to a6), a50 | **708.2** | 544.7 | 32 | **miss** | 4.2 | 0.0 / 139.6 / 108.6 | nested loop: records_list_requester 54, entity_definitions_pkey 51, groups_live_key 51, buffers 87227 | hash: records_list_group 200113, rfv_num_value 21487, records_list_requester 256, buffers 133870 | 0 / 0 |
| D7 19,999 invisible side rows, a6 | **196.0** | 166.4 | 98 | **miss** | 3.0 | 0.0 / 64.5 / 0.0 | hash: records 24602, records_list_group 24118, records_list_requester 513, buffers 22794 | - | 0 / 0 |
| D7 19,999 invisible side rows, a50 | **350.9** | 294.5 | 57 | **miss** | 2.9 | 0.0 / 104.1 / 0.0 | nested loop: records_list_requester 256, records_list_assignee 32, record_shares 3, buffers 82151 | - | 0 / 0 |
| D7 broad 50% (200,000), a6 | **522.8** | 437.4 | 41 | **miss** | 3.5 | 0.0 / 88.8 / 22.9 | nested loop: records_list_group 24118, records 24117, rfv_num_value 79, buffers 21948 | nested loop: records_list_group 24118, rfv_num_value 14080, buffers 16107 | 0 / 0 |
| D7 broad 50% (200,000), a50 | **388.6** | 293.6 | 56 | **miss** | 3.5 | 0.0 / 7.7 / 100.6 | nested loop: rfv_num_value 77, records_list_updated 74, entity_definitions_pkey 51, buffers 875 | hash+spill: rfv_num_value 200000, records_list_group 2906, buffers 5237 | 0 / 0 |

### Arm (a) (line 150 ms)

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| (a) A1-6 (3) anti-correlated 2% (8,000 oldest), a6 | **1796.7** | 1670.4 | 16 | **miss** | 2.7 | 3.5 / 581.8 / 82.4 | probe: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 49, buffers 494454 | probe: records_list_group 475, records_list_requester 9, record_shares 3, buffers 115150 | 0 / 0 |
| (a) A1-6 (3) anti-correlated 2% (8,000 oldest), a50 | **3943.5** | 3729.3 | 8 | **miss** | 2.6 | 1.6 / 1229.4 / 204.0 | probe: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 51, buffers 1182501 | probe: records_list_group 1001, rfv_num_value 0, buffers 261611 | 0 / 0 |
| (a) A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL | **6406.3** | 6291.4 | 8 | **miss** | 3.3 | 3.7 / 2972.3 / 1633.5 | probe: groups_pkey 51, records_list_updated 51, entity_definitions_number_prefix_key 1, buffers 1964076 | probe: records_list_created 1001, rfv_num_value 0, buffers 1820952 | 0 / 0 |
| (a) none visible (26,593), a6 | **1832.9** | 1668.8 | 16 | **miss** | 4.1 | 2.2 / 1322.7 / 0.0 | probe: record_shares 3, record_shares_principal 3, records_list_assignee 0, buffers 503017 | - | 0 / 0 |
| (a) none visible (26,593), a50 | **3620.0** | 3401.9 | 8 | **miss** | 4.1 | 3.9 / 1423.3 / 0.0 | probe: record_shares 3, record_shares_principal 3, records_list_assignee 0, buffers 1206005 | - | 0 / 0 |
| (a) partly correlated (494 of 21,487 visible to a6), a6 | **447.9** | 402.2 | 42 | **miss** | 2.8 | 3.4 / 94.5 / 93.5 | probe: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 47, buffers 56813 | probe: records_list_group 479, records_list_requester 14, record_shares 3, buffers 115151 | 0 / 0 |
| (a) partly correlated (494 of 21,487 visible to a6), a50 | **1191.0** | 644.0 | 32 | **miss** | 2.9 | 13.3 / 531.1 / 1374.7 | probe: entity_definitions_pkey 51, groups_live_key 51, records_list_updated 49, buffers 136356 | probe: records_list_group 487, records_list_requester 7, record_shares 3, buffers 933961 | 0 / 0 |
| (a) 19,999 invisible side rows, a6 | **1959.2** | 1442.3 | 16 | **miss** | 2.4 | 3.6 / 902.1 / 0.0 | probe: record_shares 3, record_shares_principal 3, records_list_assignee 0, buffers 503017 | - | 0 / 0 |
| (a) 19,999 invisible side rows, a50 | **3728.0** | 3530.5 | 8 | **miss** | 3.4 | 1.5 / 1391.1 / 0.0 | probe: record_shares 3, record_shares_principal 3, records_list_assignee 0, buffers 1206005 | - | 0 / 0 |
| (a) broad 50% (200,000), a6 | **190.8** | 125.9 | 122 | **miss** | 3.4 | 2.3 / 6.2 / 11.3 | probe: rfv_num_value 79, entity_definitions_pkey 51, groups_live_key 51, buffers 1856 | probe: rfv_num_value 1528, records_list_group 1001, buffers 7110 | 0 / 0 |
| (a) broad 50% (200,000), a50 | **187.1** | 129.7 | 120 | **miss** | 2.7 | 4.0 / 5.6 / 15.0 | probe: rfv_num_value 77, entity_definitions_pkey 51, groups_live_key 51, buffers 875 | probe: rfv_num_value 1441, records_list_group 1001, buffers 6732 | 0 / 0 |

(a)'s picks over the arm's requests: map[walk + pkey probe:418] (pre-count cap 4500)

- (b) index: 52 MB

### Arm (b) (line 150 ms)

| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
