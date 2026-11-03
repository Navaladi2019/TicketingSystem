# Performance budgets

Each row is a cost the platform keeps in bounds: what is measured, how, the budget, and the baseline it was set from. A change that breaks a budget needs a new measurement and the owner's sign-off.

| Budget | Measured as | Limit | Baseline | Source | Owner |
|---|---|---|---|---|---|
| Commit-check trigger (`audit_commit_check`, ADR-004 decision 2), single subject | `pg_stat_user_functions` self_time / calls, `track_functions = pl` | ≤ 100 µs per call | 34–76 µs (S1, PR #371); 42.2 µs over 5,000 calls as `nava_worker` on PostgreSQL 16.10 (P5-11, `scratch/audit-commit-check/RESULT.md`) | Technical Architect's S1 review (TICK-423), tracked instead of a p95 ratio | Postgres & Data Modeling Specialist |
