# TICK-644 (P3-34c) bench

The requester (`mine`) list on P3-01's 800k shape: the single OR statement
(before) against the UNION ALL of index-ordered branches (after), both built by
the real `authz` and `pipeline` code (`internal/server/records_list_mine_bench_integration_test.go`). Medians of 7 warm runs of
`EXPLAIN (ANALYZE, BUFFERS)` as `nava_app` in the large tenant. Not in CI.

```
docker run -d --name tick644-pg --shm-size=3g -p 127.0.0.1:55644:5432 \
  -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=bench postgres:16.10 \
  -c shared_buffers=2GB -c effective_cache_size=6GB -c max_wal_size=8GB -c track_io_timing=on
go run scratch/tick-640/setup.go -url postgres://pgadmin:bench@127.0.0.1:55644/postgres -seed
NAVA_TICK644_URL=postgres://pgadmin:bench@127.0.0.1:55644/nava [NAVA_TICK644_PLANS=1] \n  go test -tags integration ./internal/server -run TestMineListBench -v -count=1
```

`plans.txt` holds the plans of one run (page and count, before and after).

| Requester | Page | Page stmt before | Page stmt after | Count stmt before | Count stmt after |
| --- | --- | --- | --- | --- | --- |

| req1 | 1 | 17.17 ms | 1.45 ms | 257.98 ms | 8.20 ms |
| req1 | 20 | 8.39 ms | 0.74 ms | 175.52 ms | 7.66 ms |
| req8 | 1 | 29.09 ms | 0.80 ms | 203.65 ms | 2.78 ms |
| req8 | 20 | 34.70 ms | 0.92 ms | 203.75 ms | 2.35 ms |

Timings vary with host load (earlier runs: page before 10-90 ms, count before
0.2-1.8 s, after under 20 ms); the ratios hold.

`plans.txt` is `EXPLAIN (ANALYZE, BUFFERS)` of the page and the count
statement, page 1 and page 20, before and after, for both requesters
(`NAVA_TICK644_PLANS=1`). After: a Merge Append of index-ordered scans on
`records_list_requester` and `records_list_assignee`. The share arm is a
Bitmap Heap Scan on `record_shares` (the `record_shares_principal` index, the
caller's own shares) -> HashAggregate -> Nested Loop to `records` -> `Sort` ->
`Limit`; its Sort input is the caller's share rows (0 to 1 here), not the
tenant. The seeded shape has no large group with many shares; a bench for
that is deferred (TICK-644 review, round 1).
