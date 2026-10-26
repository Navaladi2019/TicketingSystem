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

- Files are `NNNNN_name.sql`, goose format (`-- +goose Up`). Forward only.
- A migration that creates a table adds that table's grants to `grants.yaml`
  in the same change. Policies stay in SQL.
- A `-- +goose NO TRANSACTION` file (e.g. `CREATE INDEX CONCURRENTLY`) must be
  idempotent (`IF NOT EXISTS`): on failure it leaves partial state and no
  version row. `TestNoTransactionMigrationsAreIdempotent` re-runs every such
  file over a migrated database.

## Adding a role

Add it under `roles:` in `grants.yaml`. A `LOGIN` role needs
`DBROLES_PASSWORD_<ROLE>` on the db-roles service. Set `pgbouncer_login: true`
only for a runtime role that connects through the pooler; validation refuses it
for a database owner or a `REPLICATION` role.

## Tests

`task test:integration` (or `make test-integration` with `NAVA_TEST_PG_URL`
pointing at a throwaway cluster's superuser) runs `internal/db/dbtest`. The
compose-level checks are `stack-test.sh pgbouncer|idempotent|dbroles|hba`.
`go test ./db/` checks `pg_hba.conf` line by line.

## pg_hba.conf

First match wins, so every `reject` stays above the catch-all
(`TestPgHBARejectsPrecedeCatchAll`). `pgadmin` only over the local socket;
`nava_backup` only as a physical walsender from the backup host.
