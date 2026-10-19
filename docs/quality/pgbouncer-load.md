# PgBouncer load test (T06)

## Scope

ADR-001 runs pgx in its default exec mode (`QueryExecModeCacheStatement`,
named prepared statements cached per client connection) through PgBouncer in
transaction mode. That works only because PgBouncer ≥ 1.21 tracks prepared
statements (`max_prepared_statements > 0`). This test proves it under
interleaved load, and proves the test would catch the fault.

## Risks, ranked

| Risk | Covered by |
| --- | --- |
| A PgBouncer image or setting change (`max_prepared_statements = 0`, version < 1.21) breaks every query that pgx prepares: `42P05 already exists` / `26000 does not exist` | `TestPgBouncerPreparedStatements`: 50 clients x 200 statements, zero errors |
| The test passes vacuously because clients never share a server connection | Same test: `SHOW POOLS` shows 50 clients logged in on at most 5 server connections; `SHOW STATS` shows one server assignment per transaction (≥ 10,000); at most 5 backend PIDs, at least one serving more than one client |
| The check cannot see the fault | `TestPgBouncerWithoutPreparedStatementTracking`: the same load and the same check through a PgBouncer with `max_prepared_statements = 0` must report prepared-statement errors |

## Setup

`scripts/ci/test-integration.sh` (the CI `integration` job) starts, on one
private Docker network, the shared PostgreSQL and two PgBouncers from the
`edoburu/pgbouncer` row of `docs/pins.md`:

| Pooler | Config | `max_prepared_statements` | Env var |
| --- | --- | --- | --- |
| Under test | `scripts/ci/pgbouncer/pooled.ini` | 100 (as in compose) | `NAVA_TEST_PGBOUNCER_URL` |
| Negative case | `scripts/ci/pgbouncer/noprep.ini` | 0 | `NAVA_TEST_PGBOUNCER_NOPREP_URL` |

Both: `pool_mode = transaction`, `default_pool_size = 5`, no reserve pool.
The test creates its own role and database `nava_it_pool` (test-only
password, also in `scripts/ci/pgbouncer/userlist.txt`), so it does not
depend on `db/grants.yaml` or on the dbtest suites.

Five distinct statements, each returning `pg_backend_pid()`; each client
walks them from its own offset, so clients Parse and Bind different
statement names on the same server connection. pgx names statements by a
hash of the SQL text, so names collide across clients, as in production.

## Not tested, and why

- **The compose PgBouncer itself.** `stack-test-pgbouncer` already checks
  its mode, version and `max_prepared_statements`; this test uses the same
  image and settings except the pool size, which must be small to force sharing.
- **`QueryExecModeCacheDescribe` / simple protocol.** Not the mode the core uses.
- **Throughput.** This is a correctness test; it reports no latency budget.

## Run it

```sh
make test-integration ARGS="-run TestPgBouncer"
```

Expected: both tests pass. `TestPgBouncerWithoutPreparedStatementTracking`
passes by logging `negative case failed as expected: N prepared-statement
errors`.
