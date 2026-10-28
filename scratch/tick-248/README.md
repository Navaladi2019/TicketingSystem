# TICK-248 claim-plan evidence

`seed.sql` builds the outbox table and its two pending indexes on an empty
database and writes 50,827 pending rows over 202 tenants:

- S1, BenchmarkClaim's shape: one tenant with 1,000 targets 45 deep, and 199
  tenants with 25 one-row targets each.
- S2, T33's stall: tenant `…0a`, target X (a head and 250 successors), then
  target Y. X's head has `attempts = 1` and a lease that ended, so it sorts
  behind its successors.
- S3, the `stranded` worst case: tenant `…0b`, 200 targets 3 deep, every head
  still leased, so the full window covers 100 targets and every probe finds a
  leased head.

`explain.sh` runs `EXPLAIN (ANALYZE, BUFFERS)` of the old and the new
`claimSQL` three times each, rolled back.

```sh
docker run -d --name pg -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=x postgres:16.10
docker exec pg psql -U pgadmin -c 'CREATE DATABASE tick248'
docker cp scratch/tick-248/seed.sql pg:/seed.sql
docker exec pg psql -q -U pgadmin -d tick248 -f /seed.sql
PSQL="docker exec -i pg psql" sh scratch/tick-248/explain.sh -U pgadmin -d tick248
```

Results (PostgreSQL 16.10 in Docker Desktop, 12th Gen Intel i7-1260P, 16
vCPU, 20 GB to the VM, Windows 11), from the PR description:

| | old | new |
| --- | --- | --- |
| Whole claim, batch A (3 runs) | 231 / 254 / 279 ms | 278 / 265 / 243 ms |
| Whole claim, batch B (3 runs, busier host) | 323 / 482 / 352 ms | 420 / 470 / 507 ms |
| Shared buffer hits | 75.4k–77.9k | 78.3k–79.3k (one outlier at 32.9k) |
| `picked` CTE | 144–241 ms | 126–305 ms |
| `stranded` CTE | — | 2.4–4.8 ms, 855 buffer hits, 101 probes |
| Tenant `…0a` X's head claimed again | no | yes (attempt 2) |

`picked` dominates both versions and does not change. The host is noisy, so
whole-claim times overlap; `stranded` is the only new node.
