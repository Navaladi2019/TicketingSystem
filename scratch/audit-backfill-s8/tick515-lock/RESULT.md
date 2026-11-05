# TICK-515 item 3: explicit LOCK TABLE at the top of `audit_backfill_s8()`

Question: does adding

```sql
LOCK TABLE tenants, entity_definition_versions, audit_log IN ACCESS EXCLUSIVE MODE;
```

after the `current_user` check slow down the P5-32 re-run? Measured with the
unchanged `../run.sh "10000 10 3"` (10,000 tenants, 300,000 S8 versions),
postgres:16.10 in Docker, Windows 11 dev host, container sees 16 CPUs,
2026-11-03. Runs alternated on the same host. Raw output in this directory.

| Variant | Run | Migration A (first run) | P5-32 re-run, all present |
| --- | --- | ---: | ---: |
| No LOCK (as merged in PR #414) | 1 | 43.70 s | 1.85 s |
| No LOCK | 2 | 51.65 s | 1.62 s |
| `LOCK TABLE ... audit_log` | 1 | 83.14 s | 1.05 s |
| `LOCK TABLE ... audit_log` | 2 | 78.70 s | 1.07 s |
| `LOCK TABLE ... ONLY audit_log` | 1 | 61.86 s | 12.75 s |

## Decision: not adopted

- The re-run with the plain LOCK was not slower (1.05 to 1.07 s against
  1.62 to 1.85 s), but migration A's first run, which is the tenant-lock
  hold, went from 43.7 to 51.7 s to 78.7 to 83.1 s, both runs. That breaks the
  ≤ 18 s per 100k versions budget row this PR adds.
- The `ONLY audit_log` variant (no recursion into partitions) re-ran in
  12.75 s once. This host's run-to-run spread is wide, so neither LOCK form
  can be shown not to slow the P5-32 re-run.
- The lock order is already fixed: the four CREATE POLICY statements run in
  the order tenants, entity_definition_versions, audit_log, and take ACCESS
  EXCLUSIVE in that order. The function header comment records it.

The cause of the first-run slowdown was not diagnosed; it is not needed for
the decision.
