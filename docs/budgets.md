# Performance budgets

Each row is a cost the platform keeps in bounds: what is measured, how, the budget, and the baseline it was set from. A change that breaks a budget needs a new measurement and the owner's sign-off.

| Budget | Measured as | Limit | Baseline | Source | Owner |
|---|---|---|---|---|---|
| Commit-check trigger (`audit_commit_check`, ADR-004 decision 2), single subject | `pg_stat_user_functions` self_time / calls, `track_functions = pl` | ≤ 100 µs per call | 34–76 µs (S1, PR #371); 42.2 µs over 5,000 calls as `nava_worker` on PostgreSQL 16.10 (P5-11, `scratch/audit-commit-check/RESULT.md`) | Technical Architect's S1 review (TICK-423), tracked instead of a p95 ratio | Postgres & Data Modeling Specialist |

## Migration lock holds

Locks a migration holds on tables that request paths need. `lock_timeout` bounds only the wait to take a lock, not how long the migration then holds it; these rows bound the hold.

| Budget | Measured as | Limit | Baseline | Source | Owner |
|---|---|---|---|---|---|
| Migration A/B tenant-lock hold (S8 backfill, `audit_backfill_s8()`): ACCESS EXCLUSIVE on `tenants`, `entity_definition_versions` and `audit_log` until COMMIT | Wall time of `SELECT audit_backfill_s8()` as `nava_migrator`, `scratch/audit-backfill-s8/run.sh` | ≤ 18 s at 100k S8 versions | 0.18 ms per backfilled version, linear. A re-run with every entry present takes 2.32 s at 300k. Measured 2026-11-03 on postgres:16.10 with 16 CPUs | PR #414 `scratch/audit-backfill-s8/RESULT.md` (P5-15, TICK-433); Technical Architect's review §3 | Postgres & Data Modeling Specialist |
