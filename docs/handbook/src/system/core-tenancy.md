---
title: "Core: tenant transactions (InTenantTx, batching, allocator pool)"
section: system
status: delivered
normative: docs/architecture/tenant-transactions.md
sources: [docs/architecture/tenant-transactions.md, docs/architecture/tenant-tx-batching.md, docs/architecture/tenant-resolution.md, internal/tenancy/tenancy.go, internal/tenancy/firstbatch.go, internal/tenancy/alloc.go, internal/tenancy/locktimeout.go, internal/tenancy/tenancytest/guard.go, internal/platform/auth]
packages: [internal/tenancy, internal/tenancy/tenancytest, internal/tenancy/isolation, internal/platform/auth]
adrs: [ADR-001, ADR-005]
invariants: [I1, I2, I3]
tables: [tenants, record_counters, idempotency_keys]
tests: [TestBeginSetsTheTenantInOneRoundTrip, TestTenantIsTransactionLocal, TestTenantNotVisibleThroughPgBouncer, TestBatchCannotReachAnotherTenant, TestFirstBatchRunsUnderTheTenant, TestFirstBatchCannotReachAnotherTenant, TestFirstBatchFailureKeepsNothing, TestSavepointRollbackUndoesItsMemo, TestExecAtCommit, TestPanicRollsBackAndReraises, TestNoTenant, TestAllocTxIsOneSendAndLeavesNoSetting, TestAllocTxFailureLeaksNothing, TestAllocatorPoolRefusesOtherRoles, TestQueueUnderLockTimeout, TestTracedTxExportsNoValues, TestCreateRecordFailureInFirstOrLastTripLeavesNothing]
---

## Purpose

Every tenant-scoped database access in `nava-core` runs inside
`tenancy.InTenantTx`. It is the only Go code that sets `app.tenant_id`, the
setting every row-level security policy reads, and it sets it in the same
round trip as `BEGIN`. This page walks the path from a verified token to a
scoped transaction, and the helpers the record pipeline uses on top of it
(first-batch BEGIN, commit batch, memo, allocator transaction, lock timeout).
The rules are in the normative
[tenant-transactions.md](../../../architecture/tenant-transactions.md),
[tenant-tx-batching.md](../../../architecture/tenant-tx-batching.md) and
[tenant-resolution.md](../../../architecture/tenant-resolution.md).

## Entry point

- Resolution: `internal/platform/auth` middleware calls
  `identity.IdentityResolver.Resolve`; handlers read the tenant with
  `identity.FromContext`. Never from a path, query, body or header.
- `tenancy.InTenantTx(ctx, pool, tenantID, fn, opts...)` in
  `internal/tenancy/tenancy.go`.
- `tenancy.TenantOf(ctx, tx)`: the tenant a transaction runs under, used by
  the pipeline, the ACL store and the cascade claim instead of a parameter.

```mermaid
sequenceDiagram
    participant A as auth middleware
    participant H as handler
    participant T as tenancy.InTenantTx
    participant P as PostgreSQL (via PgBouncer)
    A->>A: Resolve token -> identity.Principal (tenant, principal, groups)
    A->>H: ctx with Principal
    H->>T: InTenantTx(principal.TenantID, fn, BeginInFirstBatch())
    alt tenantID is uuid.Nil
        T-->>H: ErrNoTenant, nothing sent
    else
        T->>H: fn(scopedTx), nothing sent yet (lazyTx)
        H->>P: first batch: BEGIN; set_config(app.tenant_id); caller's statements
        H->>P: further batches (TenantOf answers from memory)
        T->>P: commit batch: ExecAtCommit / QueueAtCommit statements; COMMIT
    end
```

## Steps

| # | Step | Where | Invariant | Proving test |
|---|---|---|---|---|
| 1 | Resolve the tenant from the validated token and the tenant binding; only an `active` tenant resolves | `internal/platform/auth`, `keycloak.Adapter.Resolve`, `pgstore.Binding` | I2: tenant only from a platform-verified source | `TestNoTenant` (empty tenant), resolution tests listed in tenant-resolution.md |
| 2 | Open the transaction: `BEGIN; SELECT set_config('app.tenant_id', <uuid>, true)` in one simple-query round trip, transaction-local | `tenancy.InTenantTx`, `tenancy.BeginTenant` | I1: RLS sees the tenant before any statement of `fn` | `TestBeginSetsTheTenantInOneRoundTrip`, `TestTenantIsTransactionLocal`, `TestTenantNotVisibleThroughPgBouncer` |
| 3 | With `BeginInFirstBatch()`, defer BEGIN into the first `pgx.Batch` (BEGIN and tenant first, then the caller's statements) | `lazyTx.SendBatch`, `lazyTx.begin` (`firstbatch.go`), `tenancy.FirstBatchTenant` | No statement runs before the tenant; a failure aborts everything | `TestFirstBatchRunsUnderTheTenant`, `TestFirstBatchCannotReachAnotherTenant`, `TestFirstBatchFailureKeepsNothing` |
| 4 | Per-transaction memo for prefetched reads (definition, policy, the row a create stored); savepoint rollback undoes it | `tenancy.Remember`, `Recall`, `Forget` (`scopedTx`, `memo`) | The memo dies with the transaction, which has one tenant | `TestMemoIsPerTransaction`, `TestSavepointRollbackUndoesItsMemo` |
| 5 | Queue statements for the round trip that commits | `tenancy.ExecAtCommit`, `tenancy.QueueAtCommit`, `scopedTx.commitBatch` | A failing queued statement skips COMMIT | `TestExecAtCommit`, `TestCreateRecordFailureInFirstOrLastTripLeavesNothing` |
| 6 | Bound one statement's lock wait (the idempotency claim) without a round trip | `tenancy.QueueUnderLockTimeout` (`locktimeout.go`) | Only that statement waits at most the bound (55P03) | `TestQueueUnderLockTimeout`, `TestQueueUnderLockTimeoutFloor` |
| 7 | Run a number allocation in its own transaction on the allocator pool, tenant from the create's transaction, one pipelined send, asynchronous commit | `tenancy.AllocTx`, `tenancy.AllocPoolConfig`, `tenancy.CheckAllocatorRole` (`alloc.go`) | I3: the pool logs in as `nava_app` only, never SUPERUSER or BYPASSRLS | `TestAllocTxIsOneSendAndLeavesNoSetting`, `TestAllocTxFailureLeaksNothing`, `TestAllocatorPoolRefusesOtherRoles` |
| 8 | End: COMMIT on success; ROLLBACK on error (error returned unwrapped, a rollback failure joined) or panic (re-raised) | `tenancy.InTenantTx` | The setting ends with the transaction | `TestErrorRollsBack`, `TestFailedRollbackIsJoined`, `TestPanicRollsBackAndReraises`, `TestCommitFailureIsReturned` |

Every request in the tenancy, pipeline and server tests runs through
`tenancytest.Guard`, a pgx tracer that records any statement sent on a
connection with no tenant. The cross-tenant path registry is
`internal/tenancy/isolation`.

## Record-create round trips (TICK-692)

A `gaps` record create is four round trips:

| Round trip | What |
|---|---|
| 1 | BEGIN + tenant, idempotency claim under `QueueUnderLockTimeout`, prefetch (definition, ACL policy) |
| 2 | checks: unique locks and lookups, assignment group, payload preview |
| 3 | number: `tenancy.AllocTx` on the allocator pool (its own COMMIT, asynchronous) |
| 4 | `INSERT records` (asserted), write key, audit entry, idempotency completion, COMMIT |

The statement-level table and the history of the budget are in
[tenant-tx-batching.md](../../../architecture/tenant-tx-batching.md);
measured latency against the budget is on the
[benchmarks page](../benchmarks/index.md).

## Errors

| Error | Meaning |
|---|---|
| `tenancy.ErrNoTenant` | `uuid.Nil` tenant, or `TenantOf` on a transaction with no tenant; no transaction is opened |
| `tenancy.ErrAllocatorRole` | the allocator pool would log in as another role or a privileged one; startup refuses |
| SQLSTATE `55P03` | a statement under `QueueUnderLockTimeout` waited past its bound; the HTTP layer answers `409 idempotency_in_flight` for the claim |

## Notes

- navalint `setconfig` reports `set_config` / `app.tenant_id` outside
  `internal/tenancy`; navalint `tenancyconn` reports `.Conn()` and
  `.LargeObjects()` on a `pgx.Tx` outside it. The database grants are the
  control; the lint is defence in depth.
- The tracer exports no SQL text and no parameter values
  (`TestTracedTxExportsNoValues`).

## Links

- Policies, roles and grants: [database section](../database/index.md).
- Who uses it next: [core-write-pipeline](core-write-pipeline.md),
  [core-numbering](core-numbering.md).
