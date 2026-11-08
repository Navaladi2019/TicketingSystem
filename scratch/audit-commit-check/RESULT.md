# P5-11 commit check: F1 plan-test additions (TICK-428)

**Environment:** `postgres:16.10` (PostgreSQL 16.10, Debian 16.10-1.pgdg13+1) in Docker Desktop on Windows 11 (MINGW64 host, 16 CPUs, 19.4 GiB for Docker). The container ran with `fsync=off synchronous_commit=off full_page_writes=off track_functions=pl`, like the integration clusters, plus `track_functions`. Fixtures come from `internal/db/dbtest/audit_commit_check_test.go` (`TestP511CommitCheck*`). The scripts here ran in the `nava` database that the young-partition test left behind.

## Results

| Check (binding source) | Fixture | Result |
|---|---|---|
| TICK-460 (a): DEFAULT partition, through the parent | 50 tenants × 200 rows, analysed | PASS: Index Scan on `audit_log_default_subject`; `Index Cond` holds `subject_id` and `occurred_at` |
| TA cost ratio, override on | same | PASS: 205.18 / 8.31 = **24.7x** (≥ 10x required) |
| TA negative control, override RESET | same | PASS: 8.31 / 8.31 = **1.0x** (< 2x required) |
| TICK-460 (b): never-analysed range partition holding now() | that partition: 1,000 rows, `reltuples = -1` | PASS: Index Scan on the partition's subject index, cost 8.30 |
| Finding 5 item 2 (a) | 10,000 tenants × 1 row, analysed | PASS: 10 of 10 samples |
| **Finding 5 item 2 (b)** | **young partition, 60 rows (2 pages), analysed** | **FAIL: 0 of 10 samples. Every sample is a Seq Scan** |
| Finding 5 item 3 (a) | 10,000 tenants × 1 row | PASS: 12.9 ms → 57.3 ms, ratio 4.45 |
| Finding 5 item 3 (b), a fresh session per import | young partition, 60 rows | PASS: 14.4 ms → 51.0 ms, ratio 3.54 |
| Item 3 (b), one session whose plan was cached while the partition had 2 pages | young partition, 60 rows | **O(n²): 172.8 ms → 2,923 ms, ratio 16.9** (`out/cached-plan-reuse.txt`) |
| Budget: self time per call, single subject | young partition, 5,000 calls as `nava_worker` | **42.2 µs** (≤ 100 µs; S1 baseline 34–76 µs). `out/budget.txt` |

## Why item 2 (b) fails (`out/young-partition-plans.txt`)

- The partition holds 60 rows in 2 pages. Its generic plan is `Seq Scan on audit_log_p511_young`, cost 3.35, with every probe column in `Filter`.
- With `enable_seqscan = off`, the best index path is a Bitmap Heap Scan on `(tenant_id, occurred_at)`, cost 6.24, and it too leaves `subject_id` out of `Index Cond`.
- With 60 rows of one tenant it is still a Seq Scan, cost 4.35. The result does not depend on how the rows are spread.
- After 4,000 more rows inside the same transaction (statistics still say 60 rows), a newly built plan is the subject Index Scan. The planner scales `reltuples` by the partition's current page count.

The failure is deterministic, not flaky: a partition that small makes a seq scan the cheapest plan.

## Why it matters even though item 3 passes

Item 3, as written, opens a new session for each import, so each commit check builds its plan at COMMIT, after the import has grown the partition. A pooled server session (PgBouncer transaction mode) keeps its cached generic plan until the relation is invalidated, which ANALYZE does. Such a session caches the Seq Scan plan early in the month. A later large import in that session then scans the whole growing partition once per subject: 4,000 subjects take 2.9 s at COMMIT.
