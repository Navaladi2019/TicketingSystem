# db/

The platform database's migrations, `pg_hba.conf` (client authentication,
mounted into the postgres container) and `grants.yaml`, the single grant
manifest (ADR-001 I3, A.3; PLAN-001 D-P2). Both are embedded in the `nava-db`
binary (`db/embed.go`). `global-tables.yaml` arrives with T10. Changes here
need Postgres & Data Modeling Specialist review and a Security review (D-P10).
Design notes: [`docs/data/roles-and-grants.md`](../docs/data/roles-and-grants.md).

## Jobs

| Job | Runs as | Does |
| --- | --- | --- |
| `nava-db roles` (compose `db-roles`) | bootstrap superuser, over the Unix socket only | Fails first, changing nothing, if a role outside `grants.yaml` other than itself is a superuser. Creates or alters every role in `grants.yaml` (always `NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS`, plus the listed `LOGIN` / `REPLICATION`), revokes every role membership the file does not list, creates the databases with their owners (revoking PUBLIC's default grants on a new `nava` at once), revokes every grant on the service databases except the owner's (PUBLIC included), and builds PgBouncer's `pgbouncer.get_auth` with its grants. |
| `nava-db migrate` (compose `migrate`) | `nava_migrator`, directly to PostgreSQL | Applies `migrations/` with goose under a session advisory lock, then applies `grants.yaml`: grants what is listed, revokes and logs everything else on objects the migrator owns, `PUBLIC` and default privileges included. On a fresh database (the run applied the first migration) any such revoke means a migration granted off the manifest, and the run fails naming the grant. |

Both are idempotent: a second run logs `"created":0,"changed":0,"revoked":0` and
`"migrations_applied":0,"grants_granted":0,"grants_revoked":0`. nava-core never
migrates.

## Adding a migration

- New files are `YYYYMMDDHHMMSS_name.sql` (`goose create name sql`), goose
  format (`-- +goose Up`). Forward only. See
  [`migrations/README.md`](migrations/README.md).
- A migration that creates a table adds that table's grants to `grants.yaml`
  in the same change. Policies stay in SQL.
- A `-- +goose NO TRANSACTION` file (e.g. `CREATE INDEX CONCURRENTLY`) must be
  idempotent (`IF NOT EXISTS`): on failure it leaves partial state and no
  version row. `TestNoTransactionMigrationsAreIdempotent` re-runs every such
  file over a migrated database.
- Migrations run under a session `lock_timeout` (5s,
  `migrate.Options.LockTimeout`) and are retried up to 10 times on SQLSTATE
  55P03 (TICK-579). `CREATE/DROP INDEX CONCURRENTLY` waits for older
  transactions under that timeout, so a NO TRANSACTION migration that needs a
  longer wait must `SET lock_timeout = '<n>s'` before the statement and
  `RESET lock_timeout` at the end of the file, and stay re-runnable
  (`DROP ... IF EXISTS` before `CREATE ... IF NOT EXISTS`).

## JIT (off on the platform database)

`20261109110700_platform_database_jit_off.sql` runs
`ALTER DATABASE <platform> SET jit = off` (TICK-640). The platform database
serves OLTP only.

- **Why:**
  - A custom plan (`nava_app` runs `force_custom_plan` with pgx
    `QueryExecModeExec`) is JIT-compiled again on every execution once its
    cost passes `jit_above_cost` (100k).
  - On P3-01's 800k shape that added 200–990 ms to each list statement
    (P3-36, TICK-640), against a statement budget of 50–150 ms. No platform
    query runs long enough to pay compilation back.
  - P3-01 had not assumed JIT was off: it ran with `jit = on`, and its walks
    stayed under the threshold. The relay (TICK-250) already shapes its claim
    to stay under it.
- **Why the database and not the roles:** `ALTER ROLE ... SET` needs
  `CREATEROLE`, which the migrator lacks. db-roles resets every role setting
  `grants.yaml` does not list, and `grants.AllowedSettings` admits only
  `plan_cache_mode`. The migrator owns the database, so the database-level
  setting is its own to set, and db-roles never touches it.
- **What it covers:**
  - Every role on the platform database, including `nava_app` and
    `nava_worker`, and new sessions only: pools pick it up as connections
    are recycled.
  - A session that needs JIT can `SET jit = on`.
  - Keycloak's and Temporal's databases keep the server default.
- **Tests:**
  - `TestP334bPlatformDatabaseJITOff` pins it.
  - `migratedCluster` re-applies database settings to its template copies,
    because `CREATE DATABASE ... TEMPLATE` does not copy them.
- **On a managed server:** the setting travels with the database, so no
  server parameter is needed.

## Adding a role

Add it under `roles:` in `grants.yaml`. A `LOGIN` role needs
`DBROLES_PASSWORD_<ROLE>` on the db-roles service. Set `pgbouncer_login: true`
only for a runtime role that connects through the pooler; validation refuses it
for a database owner or a `REPLICATION` role.

## Tests

`task test:integration` (or `make test-integration` with `NAVA_TEST_PG_URL`
pointing at a throwaway cluster's superuser) runs `internal/db/dbtest`. The
compose-level checks are `stack-test.sh pgbouncer|idempotent|dbroles|hba`.
`go test ./db/` checks `pg_hba.conf` line by line, and the migration names,
versions and goose parse.

## pg_hba.conf

First match wins, so every `reject` stays above the catch-all
(`TestPgHBARejectsPrecedeCatchAll`). `pgadmin` only over the local socket;
`nava_backup` only as a physical walsender from the backup host.
