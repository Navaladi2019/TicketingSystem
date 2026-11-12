# TICK-665: concurrent first upserts during the M2..M3 window

`race-stress.sh <container> 20000` against `postgres:16.10` (Docker Desktop on
WSL2, 16 vCPU, 19.4 GiB). 8 workers, each 20,000 upserts in a commit-per-row
DO loop; the key is the current 1 ms tick, so the workers race the FIRST
insert of each new row.

| Indexes | Statement | Rows | unique_violation |
|---|---|---|---|
| pkey (t, d) only (today) | R0 `ON CONFLICT (t, d)` | 84,501 | 0 of 160,000 |
| pkey + scoped_key (M2 applied) | R0 `ON CONFLICT (t, d)` | 63,595 | 15,533 on `rc_scoped_key` |
| pkey + scoped_key (M2 applied) | R1 `ON CONFLICT (t, d, scope_key)` | 87,373 | 19,829 on `rc_pkey` |
| 3-column pkey only (after M3) | R1 | 79,343 | 0 of 160,000 |

Cause: `INSERT ... ON CONFLICT` only absorbs conflicts on its arbiter
index(es). Two sessions that both pass the arbiter pre-check insert the heap
tuple; the loser then hits the other unique index with an ordinary unique
check and raises 23505. A row that already exists is unaffected (the pre-check
sees it and takes the UPDATE arm); only the first create of a (tenant, type)
races. Reproduced in CI-like conditions by
`TestAllocatorPoolDoesNotStarveThePool` (`direct, allocator pool`) at the time; since `48ae5047` that test runs
the first create alone, and the reproducer is `TestConcurrentFirstCreatesOfATypeAllSucceed`
(`internal/records/pipeline`, PR #581; skipped until TICK-666).
