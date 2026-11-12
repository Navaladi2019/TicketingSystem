# ADR-005-E1: one retry after M2 (Technical Architect spike, TICK-665)

Source: TICK-665 comment 01a1cb58-39fe (critique of PR #583).

- **Hypothesis:** with `pkey` and `scoped_key` both present, an R1 upsert that fails with `23505` succeeds on its single retry under READ COMMITTED, as a savepoint (`gapless`) or as a fresh transaction (`gaps`).
- **Pass:** 0 retry failures, and numbers = upserts. **Fail:** any second failure.
- **Setup:** `postgres:16.10`, Docker on WSL2, 16 vCPU. 8 workers x 5,000 R1 upserts; each key is the current 1 ms tick, so most upserts race a first insert.
- **Verdict:** PASS at READ COMMITTED. At REPEATABLE READ the race raises `40001`, so the retry does not fire: the isolation constraint in E1 is real.
- **Rerun:** `docker run -d --name e1spike -e POSTGRES_HOST_AUTH_METHOD=trust postgres:16.10`, then `sh retry-once.sh e1spike 5000`.

## Raw output

```
RC, savepoint (gapless): 8 x 5000; 22136 rows, 40000 numbers; first-attempt 23505: 6140; retry failures: none
RC, fresh tx (gaps):     8 x 5000; 21587 rows, 40000 numbers; first-attempt 23505: 6996; retry failures: none
RR, savepoint (gapless): workers aborted on uncaught 40001 "could not serialize access due to concurrent update" (7 of 8); 5344 numbers
RR, fresh tx (gaps):     same 40001 (6 of 8 workers); 3 first-attempt 23505s
```
