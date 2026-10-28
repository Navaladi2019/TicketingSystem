# TICK-250 claim `picked` plan evidence

The dataset is `scratch/tick-248/seed.sql` (50,827 pending rows, 202 tenants;
S1 flood, S2 stall, S3 stranded worst case). The plans come from
`scratch/tick-248/explain.sh` with `BASE=HEAD` (old is the story branch at
9e7598b, new is this change). `generic.sh` runs the claim as a prepared
statement with custom and then generic plans.

```sh
PSQL="docker exec -i pg psql" BASE=<old rev> sh scratch/tick-248/explain.sh -U pgadmin -d tick248
PSQL="docker exec -i pg psql" sh scratch/tick-250/generic.sh -U pgadmin -d tick248
```

Environment: PostgreSQL 16.10 in Docker Desktop, 12th Gen Intel i7-1260P,
16 vCPU, Windows 11. About ten other agents' test clusters were running on
the host at the same time, so wall-clock times are 2–4x the TICK-248 figures
for the same SQL. Buffer counts don't depend on load and are the primary
evidence.

## Cause

The old `picked` ran once per tenant (202 loops) and, on each loop:

1. Scanned the whole materialized `window_rows` CTE (5,575 rows) and dropped
   5,547 of them with a filter, because a CTE scan has no index.
2. Merge-joined that to `outbox_pkey` over the tenant's whole pending set
   (48k buffer hits in total). The planner expected 100 window rows and got
   5,575.
3. Ran the FIFO anti-join for every window row (5,575 probes, 17k buffers),
   then sorted, then kept 2.

Its total estimated cost of about 155k is over `jit_above_cost` (100k), so
every claim also paid 150–540 ms of JIT compilation. Services run pgx in
`QueryExecModeExec`, so every claim is custom-planned.

## Change

- `scan` also keeps the end of each tenant's window: the last row's
  (`available_at`, `id`).
- `picked` walks `outbox_pending_available` for the tenant directly. It reads
  in `available_at` order, bounded by `(available_at, id) <= window end`
  (`available_at <= last_at` becomes an index condition), and stops at its
  2nd row. It no longer sorts, so it probes the heads of only the rows it
  reads. The order ignores `id`; ties come in index order.
- The FIFO test is `target_seq <= min(pending target_seq on the target)`, a
  scalar subplan on `outbox_pending_target` (Index Only Scan, LIMIT 1). For a
  pending row this is the same as the old `NOT EXISTS (lower pending seq)`.
  It is written as `<=` so the planner estimates that 1/3 of rows pass rather
  than 0.5%. With `=`, the custom plan's cost was 351k (JIT on), and
  `NOT EXISTS` was planned as a Materialize anti-join over the tenant's rows
  (temp spill, 184 ms).
- No index or DDL change. Semantics are unchanged: the same window and the
  same FIFO and lock rules. The relay integration suite passes (T30a, T33,
  TICK-248's stranded tests).

## Results (3 runs each, rolled back)

| | old | new |
| --- | --- | --- |
| `picked` CTE time (EXPLAIN ANALYZE) | 463 / 494 / 556 ms | 14 / 27 / 30 ms |
| `picked` shared buffer hits | 65,838–65,865 | 4,415 |
| `picked` head probes | 5,575 | 844 |
| `picked` estimated cost | ~100k | ~6.4k |
| Whole claim buffer hits | 77.9k–78.6k | 16.8k–17.4k |
| Whole claim estimated cost (custom plan) | ~156k (JIT on) | ~63.8k (no JIT) |
| Whole claim, TIMING OFF, 5 runs | 950–1,235 ms | 107–131 ms |
| `stranded` CTE | 6.6–7.8 ms, 809 hits | 3.9–11.0 ms, 809 hits |
| Tenant `…0a` X's head claimed again | yes (attempt 2) | yes (attempt 2) |

`stranded` reads the same 809 buffers before and after, for 101 probes in
S3. It did not grow, so the "skip the probe when the head is leased" change
is not needed.

What remains in the claim is mostly the `claimed` UPDATE: index maintenance
for 401 rows, about 14.6k of the 17k buffers.

The ADR-001 5.4 ms baseline is `BenchmarkClaim` wall clock on a quiet host.
On this host the old SQL took 258–479 ms/claim in `BenchmarkClaim` (200x,
with run-to-run noise of up to 1.8x), so its absolute figure cannot be
reproduced here. Compare against it with the nightly `bench-relay`
(TODO-011).
