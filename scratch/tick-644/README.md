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
| req1 | 1 | 10.93 ms | 1.76 ms | 1030.18 ms | 9.45 ms |
| req1 | 20 | 10.12 ms | 1.46 ms | 925.62 ms | 11.94 ms |
| req8 | 1 | 59.24 ms | 1.08 ms | 1012.46 ms | 2.10 ms |
| req8 | 20 | 63.24 ms | 1.03 ms | 945.01 ms | 1.93 ms |

Timings vary with host load (an earlier run: page before 13-68 ms, count before 270-540 ms, after under 18 ms); the ratios hold. Plans: the page statement is a
Merge Append of index-ordered `records_list_requester` / `records_list_assignee` arms; the only `Sort` is the
share arm's, over the rows shared with the caller.
