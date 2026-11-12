---
title: "pgbouncer"
section: system
status: delivered
sources: [compose.yaml, deploy/compose/pgbouncer/pgbouncer.ini, db/grants.yaml]
adrs: [ADR-001]
invariants: [I3]
tests: [TestGetAuthReachableOnlyByPgBouncerAuth, TestTenantNotVisibleThroughPgBouncer, TestConcurrentCreatesNumberWithoutGapsThroughPgBouncer]
---

The connection pooler between every `nava-*` service and PostgreSQL, in transaction pooling mode.

| | |
|---|---|
| Image | `edoburu/pgbouncer:v1.24.1-p1` |
| Network | `internal` |
| Listener | `:5432` |
| Host port | none |
| Pools | `nava` and `nava_alloc` (the record-number allocator, `pool_size=4`), both on database `nava` |
| DB role | `pgbouncer_auth`, which may only call `pgbouncer.get_auth(text)` (the `auth_query`) |
| Introduced by | PLAN-001 T02 (TICK-51) |

## What talks to it

`nava-core`, `nava-onboard`, `nava-groupsync`, `nava-relay`, `nava-worker` and `nava-purge`. It talks to `postgres:5432`. It starts only after `migrate` completes.

## Boundary it enforces

**I3, least-privilege roles.** PgBouncer logs in only roles marked `pgbouncer_login: true` in `db/grants.yaml`: `get_auth` returns nothing for a superuser, a `BYPASSRLS` role, a replication role or `nava_migrator`. `TestGetAuthReachableOnlyByPgBouncerAuth` proves only `pgbouncer_auth` can call the lookup, and `TestTenantNotVisibleThroughPgBouncer` proves a pooled server connection never carries one transaction's tenant into the next.

[Back to the system map](index.md)
