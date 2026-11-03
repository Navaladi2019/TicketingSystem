# Performance budgets

Each row is a cost the platform keeps in bounds: what is measured, how, the budget, and the baseline it was set from. A change that breaks a budget needs a new measurement and the owner's sign-off.

| Budget | Measured as | Limit | Baseline | Source | Owner |
|---|---|---|---|---|---|
| Commit-check trigger (`audit_commit_check`, ADR-004 decision 2, ADR-004-E3), single subject | `pg_stat_user_functions` self_time / calls, `track_functions = pl`. One connection (pooled-backend state), N single-subject transactions as `nava_worker`, at N = 200, 1,000 and 5,000. Young partition analysed at 60 rows, autovacuum off on it. 38 partitions. Same-run reference: a copy of the function without the `plan_cache_mode` clause, same fixture, same job. Measured by `TestCommitCheckBudget` in the CI job `commit-check-budget` (`postgres:16.10`, `ubuntu-latest`, 4 vCPU, nothing else on the runner) | ≤ 300 µs per call, and flat: max/min < 1.5 across the three N. Over the ceiling fires E3-R2 S3 only with a same-run ratio out of range (per call at N = 200 > 5, or import > 15); otherwise re-run | 170.5 / 163.6 / 164.8 µs at N = 200 / 1,000 / 5,000, max/min 1.04. Reference 133.5 / 82.1 / 212.0 µs, ratio at N = 200 1.28. 2026-11-03, CI [run 41673342312](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41673342312/job/126047559767). Provenance only: TA S2 laptop runs (TICK-502), 248–251 µs at 38 partitions | ADR-004-E3 budget, Technical Architect S2 and re-check (TICK-502), CTO ruling | Postgres & Data Modeling Specialist |
| Commit-check trigger, import | COMMIT time / 10,000 for one 10,000-subject import, fresh `nava_worker` session, tenant absent from the statistics, median of 3, 38 partitions. Same-run reference and CI job as the row above | ≤ 175 µs per probe. Same S3 rule as the row above | 127.7 µs per probe (runs 127.4 / 127.7 / 129.3). Reference 13.1 µs, ratio 9.75. 2026-11-03, CI [run 41673342312](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41673342312/job/126047559767). Provenance only: TA laptop, 120–142 µs at 2 partitions | ADR-004-E3 budget, Technical Architect S2 and re-check (TICK-502), CTO ruling | Postgres & Data Modeling Specialist |

## Migration lock holds

Locks a migration holds on tables that request paths need. `lock_timeout` bounds only the wait to take a lock, not how long the migration then holds it; these rows bound the hold.

| Budget | Measured as | Limit | Baseline | Source | Owner |
|---|---|---|---|---|---|
| Migration A/B tenant-lock hold (S8 backfill, `audit_backfill_s8()`): ACCESS EXCLUSIVE on `tenants`, `entity_definition_versions` and `audit_log` until COMMIT | Wall time of `SELECT audit_backfill_s8()` as `nava_migrator`, `scratch/audit-backfill-s8/run.sh` | ≤ 18 s at 100k S8 versions | 0.18 ms per backfilled version, linear. A re-run with every entry present takes 2.32 s at 300k. Measured 2026-11-03 on postgres:16.10 with 16 CPUs | PR #414 `scratch/audit-backfill-s8/RESULT.md` (P5-15, TICK-433); Technical Architect's review §3 | Postgres & Data Modeling Specialist |
