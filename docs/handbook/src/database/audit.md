---
title: "Audit log and seal chain"
section: database
status: delivered
tables: [audit_log, audit_log_default, audit_chain_head]
adrs: [ADR-004, ADR-004-A1, ADR-001-A6]
sources: [docs/data/audit-log.md, docs/adr/ADR-004-audit-log.md]
---

`audit_log` is one append-only log for every audited table (ADR-004). An
audited write must insert its entry in the same transaction, or a deferred
commit-check trigger fails the commit. A sealer (`nava_worker`) then chains
each tenant's entries with `seq`, `prev_hash` and `entry_hash`.
`audit_chain_head` holds each tenant's chain position and its export lease.

**Partitioning.** `audit_log` is `PARTITION BY RANGE (occurred_at)` with one
`DEFAULT` partition, `audit_log_default`. Partition management (new ranges,
detaching old ones) belongs to Epic 30. The key shape is fixed now, so that
later work attaches and detaches partitions instead of rewriting the table.
Because of the partitioning:

- the primary key and the commit-check key include `occurred_at`. `seq` has a
  plain, non-unique index, because a unique key cannot span partitions; the
  verifier reports duplicate or missing `seq` values.
- a `BEFORE INSERT` trigger rejects any `occurred_at` other than `now()`. It
  never rewrites the value: a rewrite could move the row to another partition,
  which PostgreSQL 16 refuses.
- the `DEFAULT` partition has no policy and no grant, so a runtime role
  reaches its rows only through the parent.
- each partition sets `n_distinct = 1` on `occurred_at`, so the planner keeps
  the tenant-led indexes. Epic 30 must set it on every partition it creates.

Only `nava_audit_reader` reads `diff` and `diff_salt` (ADR-004-A1). The
migrator generates the `audit_log` policies from `db/audited-tables.yaml`.

## Diagram

<!-- gen:er area=audit -->

## Tables

<!-- gen:tables area=audit -->
