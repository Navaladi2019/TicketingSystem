---
title: "nava-purge"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-purge/main.go, db/grants.yaml, internal/purge/purge.go, db/migrations/20261113210800_record_counters_purge.sql]
adrs: [ADR-001, ADR-004, ADR-005]
stories: [S18]
tables: [idempotency_keys, outbox, workflow_ref, record_counters]
packages: [cmd/nava-purge, internal/purge]
invariants: [I3]
tests: [TestT13PurgeRoleCannotWidenItsReach, TestT13PurgeDeletesExpiredRowsOfTwoTenants, TestP610PurgeDeletesExpiredPeriodCountersOnly, TestP610PurgeBatchesCounterRows, TestP610PurgeRoleReachOnCounters]
---

The retention purge: on a ticker, deletes expired `idempotency_keys`, `outbox` and `workflow_ref` rows and expired period counters in `record_counters` (S18), and counts dead audit seal lineages. It does not use Temporal and holds no certificate.

| | |
|---|---|
| Network | `internal` |
| Listeners | `:9090` admin only |
| Host port | none (tunnel 19095) |
| DB role | `nava_purge` through PgBouncer |
| Talks to | pgbouncer; otel-collector |
| Introduced by | PLAN-001 T13 (TICK-143) |

## Period counters (S18)

Since S18 a dated number format (`{yyyy}{MM}`) starts a new `record_counters` row each period, with `period_end` set. ADR-005 §4.3 makes their purge mandatory, so the table does not grow without bound.

- The batch `DELETE` takes up to one batch of rows whose `period_end` is more than 2 days past, through the partial index `record_counters_period_end`. The grace is fixed, not a window: it exceeds the widest UTC-offset span (26 h), so a tenant time-zone change cannot reopen a purged period.
- Rows with `period_end` NULL (the default scope, date-free formats, and recurring keys such as `{MM}` with no year) never match, so they are never purged: a restarted counter would hand out numbers already issued.
- The role sees exactly the rows it may delete: paired `SELECT` and `DELETE` policies on one predicate, pinned in the ADR-001 A.3 allow-list. Its grant is `SELECT` on the key columns and `period_end`, and `DELETE`; never `next`.

## Boundary it enforces

**I3.** `nava_purge` may delete only expired rows and read only the columns that decide expiry: `TestT13PurgeRoleCannotWidenItsReach`; on counters, `TestP610PurgeRoleReachOnCounters` and `TestP610PurgeDeletesExpiredPeriodCountersOnly`.

[Back to the system map](index.md)
