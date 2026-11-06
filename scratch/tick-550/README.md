# TICK-550: failed Parse inside a transaction through PgBouncer

`run.sh [pgbouncer-tag ...]` starts `postgres` (the `docs/pins.md` tag) and
each `edoburu/pgbouncer` tag (default: the pinned one) in transaction mode with
`max_prepared_statements = 100` and `default_pool_size = 1`. It then runs
`repro.go` in three pgx exec modes, and once directly against PostgreSQL.
`repro.go` runs a Parse-failing statement in three transactions in a row on
one connection.

Host: Windows 11, Docker 29.9.2, pgx v5.11.0, 2026-11-05.

| PgBouncer | `CacheStatement` (pgx default) | `CacheDescribe` | `Exec` |
| --- | --- | --- | --- |
| 1.24.1 (pinned) | 42883, then **08P01** FATAL, then conn closed | 42883 x3 | 42883 x3 |
| 1.25.2 | same as 1.24.1 | 42883 x3 | 42883 x3 |
| 1.26.1 (latest) | same as 1.24.1 | 42883 x3 | 42883 x3 |
| none (PostgreSQL 16.10) | 42883 x3 | not run | not run |

PgBouncer log for the failing case: `pooler error: prepared statement name is
already in use`. pgx v5.11.0 and v5.12.0 send a Close after a failed Prepare
only when ParseComplete arrived (`conn.go`, `failedDescribeStatement`). So
after a failed Parse nothing clears PgBouncer's record of the name. The
regression test is `internal/db/pooltest/parse_error_test.go`.
