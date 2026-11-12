---
title: "nava-purge"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-purge/main.go, db/grants.yaml]
adrs: [ADR-001, ADR-004]
packages: [cmd/nava-purge, internal/purge]
invariants: [I3]
tests: [TestT13PurgeRoleCannotWidenItsReach, TestT13PurgeDeletesExpiredRowsOfTwoTenants]
---

The retention purge: on a ticker, deletes expired `idempotency_keys`, `outbox` and `workflow_ref` rows, and counts dead audit seal lineages. It does not use Temporal and holds no certificate.

| | |
|---|---|
| Network | `internal` |
| Listeners | `:9090` admin only |
| Host port | none (tunnel 19095) |
| DB role | `nava_purge` through PgBouncer |
| Talks to | pgbouncer; otel-collector |
| Introduced by | PLAN-001 T13 (TICK-143) |

## Boundary it enforces

**I3.** `nava_purge` may delete only expired rows and read only the columns that decide expiry: `TestT13PurgeRoleCannotWidenItsReach`.

[Back to the system map](index.md)
