---
title: "nava-db (db-roles, migrate: one-shot)"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-db/main.go, db/grants.yaml, deploy/compose/nava-db/Dockerfile]
adrs: [ADR-001, ADR-004]
packages: [cmd/nava-db]
invariants: [I3]
tests: [TestD28NoSuperuserOrBypassRLS, TestDBRolesRolesHaveNoDangerousAttributes, TestMigratorOwnsDatabaseAndIsGrantedToNoOne]
---

One binary, `cmd/nava-db`, run as two one-shot compose services from the `nava/nava-db` image. Both exit when done; nava-core never migrates.

| | db-roles | migrate |
|---|---|---|
| Command | `nava-db roles` | `nava-db migrate` |
| Network | none (`network_mode: none`) | `internal` |
| Connects | over the Unix socket in the `pgsocket` volume | `postgres:5432` directly (PgBouncer refuses `nava_migrator`) |
| DB role | the bootstrap superuser (the only service holding its password, as a file under `/run`) | `nava_migrator`, owner of database `nava` |
| Runs after | `postgres` is healthy | `db-roles` completed |
| Listeners | none | none |

`db-roles` converges every role in `db/grants.yaml` (no `SUPERUSER`, `CREATEDB`, `CREATEROLE` or `BYPASSRLS`, no memberships), the databases and their owners, and PgBouncer's lookup function. `migrate` applies the migrations under goose's session lock, then the generated audit-log policies and grants (`db/audited-tables.yaml`), then the grant manifest, then the catalogue gate.

Introduced by PLAN-001 T03.

## Boundary it enforces

**I3, least-privilege roles, from one manifest.** Every grant comes from `db/grants.yaml`. `TestD28NoSuperuserOrBypassRLS` and `TestDBRolesRolesHaveNoDangerousAttributes` prove no runtime role gains a dangerous attribute; `TestMigratorOwnsDatabaseAndIsGrantedToNoOne` proves the owner role is never shared.

[Back to the system map](index.md)
