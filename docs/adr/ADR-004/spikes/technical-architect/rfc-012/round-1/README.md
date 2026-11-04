# RFC-012 round 1 spike (Technical Architect)

Verdict PASS. Posted on TICK-549, comment 01a1a50c-0397-7253-bd29-91cd035e104c.

- `spike_grants.diff`: the §4.1 manifest change applied to `db/grants.yaml` at `origin/TICK-541/audit-diff-reader-guard@fcd17e8`.
- `20261104200000_spike549_reader_policy.sql`: the reader SELECT policies used by the spike (the ADR uses `ALTER POLICY … TO` on the existing policies instead).
- `spike549_test.go.txt`: the integration test (`-tags integration`, `internal/db/dbtest/`). Renamed `.txt` so it is not compiled from `docs/`.

Setup: postgres:16.10, edoburu/pgbouncer:v1.24.1-p1 (transaction mode), pgx v5.11.0.
Results: A3 confirmed (`get_auth` serves the new role, login through a running PgBouncer); A4 confirmed (INSERT without SELECT); 14/14 diff-read probes `42501` for `nava_app` and `nava_worker`; reader scoped to one tenant; denial at Execute, connection survives.
