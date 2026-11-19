# TICK-747 field-index measurements, raw (2026-11-19T11:11:41Z)

- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in 3m20s
- PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 14.2.0-19) 14.2.0, 64-bit
- settings: default_statistics_target=100, effective_cache_size=7864328kB, fsync=on, jit=off, max_parallel_workers_per_gather=2, random_page_cost=4, shared_buffers=2621448kB, synchronous_commit=on, work_mem=4096kB
- nava_app plan_cache_mode: plan_cache_mode=force_custom_plan
- other containers at start: [nava-fiperf-pg-33671 nava-s18-nava-worker-1 nava-s18-nava-relay-1 nava-s18-nava-core-1 nava-s18-temporal-1 nava-s18-nava-groupsync-1 nava-s18-nava-onboard-1 nava-s18-temporal-server-1 nava-s18-nava-purge-1 nava-s18-keycloak-1 nava-s18-otel-collector-1 nava-s18-pgbouncer-1 nava-s18-postgres-1 nava-nava-core-1 nava-nava-worker-1 nava-temporal-ui-1 nava-nava-relay-1 nava-temporal-1 nava-nava-purge-1 nava-nava-groupsync-1 nava-nava-onboard-1 nava-temporal-server-1 nava-pgbouncer-1 nava-keycloak-1 nava-postgres-1 nava-prometheus-1 nava-openobserve-1 nava-otel-collector-1]
- transport floor (SELECT 1, 8 clients): p50 2.25 ms, p95 3.54 ms
- 4 fields flagged on the large tenant's ticket type only; side rows copied in 1m50s; ANALYZE: rfv_text 143ms, rfv_num 72ms, rfv_ts 69ms, rfv_bool 4ms
- side tables: rfv_bool 0 rows, 0 bytes heap, 16 kB indexes; rfv_num 400000 rows, 36 MB heap, 76 MB indexes; rfv_text 800000 rows, 64 MB heap, 153 MB indexes; rfv_ts 400000 rows, 36 MB heap, 76 MB indexes
- records: 1250 MB heap, 755 MB indexes
- probe: 189 creates, 183 patches, errors map[]

## Backfill: priority (text, 5 values) flagged late by publish v2 on the 400k type; FieldIndexBackfill on a Temporal dev server, nava_worker

Live mix: 8 clients through the handlers, alternating POST (every band field set; tier, zone, score and opened flagged) and PATCH of score, as nava_app.

| Live mix | Window | ops/s | p50 ms | p95 ms | p99 ms | create p95 | patch p95 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| alone, before | 180 s | 83.5 | 95.3 | **180.1** | 234.5 | 83.4 | 202.0 |
| with the backfill as merged | 289 s | 91.1 | 68.8 | **138.3** | 197.0 | 82.7 | 159.4 |
| with the backfill, nava_worker force_custom_plan | 486 s | 53.9 | 96.7 | **302.4** | 560.9 | 165.7 | 418.4 |
| alone, after | 180 s | 54.6 | 78.1 | **276.4** | 513.8 | 130.8 | 408.9 |

Errors: before map[patch 412 version_conflict:80]; as merged map[patch 412 version_conflict:4000]; fix candidate map[create 503 retry_later:6 patch 412 version_conflict:4000]; after map[patch 412 version_conflict:4000]

**Live-write p95 delta, as merged: -23.2% against the control before** (-50.0% against the larger control), budget +25%: meets.
**Live-write p95 delta, fix candidate: +67.9% against the control before** (+9.4% against the larger control), budget +25%: **misses**.

| Backfill | Duration | Batches | Batch min / p50 / avg / p95 / max | Pauses total (each) | First barrier | WAL | Registry | Diff |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| as merged, alone | stopped after 40 batches at 6m11s; projected 1h3m0s for 409 batches | 40 | 289 / 1042 / 1187 / 1852 / 2262 ms | 5m28s (500ms ×4, 1s ×2, 2s ×1, 4s ×1, 8s ×1, 10s ×31) | 169ms | 247 MB (6.18 MB per batch) | backfilling 9 | not checked (stopped) |
| as merged, under the live mix | stopped after 40 batches at 4m49s; projected 51m0s for 425 batches | 40 | 1462 / 1912 / 1929 / 2390 / 2431 ms | 3m32s (500ms ×11, 1s ×7, 2s ×1, 4s ×1, 5s ×1, 8s ×1, 10s ×18) | 1.157s | 587 MB (14.67 MB per batch) | backfilling 12 | not checked (stopped) |
| force_custom_plan, alone | stopped after 60 batches at 8m28s; projected 1h0m0s for 425 batches | 60 | 211 / 430 / 623 / 965 / 8397 ms | 7m52s (500ms ×9, 1s ×3, 2s ×1, 4s ×1, 8s ×1, 10s ×45) | 92ms | 316 MB (5.27 MB per batch) | backfilling 14 | not checked (stopped) |
| force_custom_plan, under the live mix | stopped after 60 batches at 8m6s; projected 59m0s for 440 batches | 60 | 549 / 794 / 803 / 1051 / 1399 ms | 7m2s (500ms ×8, 1s ×1, 1.25s ×1, 2s ×3, 2.5s ×2, 4s ×3, 5s ×3, 8s ×1, 10s ×37) | 1.264s | 809 MB (13.48 MB per batch) | backfilling 15 | not checked (stopped) |

- as merged, alone: batch ms in order: 1701 333 384 1005 288 1184 1574 1022 1393 1373 1546 1187 896 890 802 868 1019 1339 987 928 993 1042 1548 1453 869 1365 1014 1239 1752 2261 1851 1447 1385 1144 1997 1800 964 858 921 852; pauses s in order: 0.5 0.5 0.5 1 0.5 1 2 4 8 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10
- as merged, under the live mix: batch ms in order: 2085 1675 1690 1990 1638 1690 1699 1902 1832 1462 1944 1810 1816 1749 1802 1832 1525 1911 1887 1853 2170 1991 2399 2049 1464 1951 2105 1904 1916 2081 2067 2029 2002 2059 1929 2302 2431 2390 2246 1869; pauses s in order: 0.5 0.5 0.5 0.5 0.5 0.5 0.5 0.5 0.5 0.5 1 1 1 1 1 1 0.5 1 2 4 8 10 10 10 5 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10
- force_custom_plan, alone: batch ms in order: 466 253 270 303 273 218 210 229 245 278 246 252 313 304 290 422 785 478 396 375 411 544 634 422 399 429 586 492 372 335 340 436 441 325 286 412 474 431 400 482 906 452 385 653 441 860 841 843 840 777 670 488 756 517 965 1282 1058 8396 539 424; pauses s in order: 0.5 0.5 0.5 0.5 0.5 0.5 0.5 0.5 0.5 1 1 1 2 4 8 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10
- force_custom_plan, under the live mix: batch ms in order: 741 704 664 872 649 751 769 548 751 821 761 566 666 713 639 939 844 731 819 880 657 765 636 843 794 737 878 865 843 798 830 803 815 848 809 941 835 766 1050 660 1011 944 962 899 882 1046 1163 1111 581 625 737 780 774 1399 881 584 586 633 587 970; pauses s in order: 0.5 0.5 0.5 0.5 0.5 0.5 0.5 0.5 1 2 4 2 2 4 4 8 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 10 5 5 10 10 10 10 10 5 2.5 2.5 1.25
- live-mix error patch 412 version_conflict, first body: {"type":"urn:nava:problem:version_conflict","title":"Version conflict","status":412,"code":"version_conflict","detail":"The record has changed since the ETag in If-Match. The current ETag is in the ETag header. Current ETag: W/\"9\".","instance":"urn:nava:request:2ffb1f80277ae552ab6f404e765814aa"}
- live-mix error create 503 retry_later, first body: {"type":"urn:nava:problem:retry_later","title":"Try again later","status":503,"code":"retry_later","detail":"The write could not finish now and changed nothing. Retry after the interval in Retry-After.","instance":"urn:nava:request:5c55951d66d8cbe91e93d74a2d423e71"}
