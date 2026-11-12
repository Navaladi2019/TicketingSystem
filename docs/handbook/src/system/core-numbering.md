---
title: "Core: record numbering (gaps and gapless)"
section: system
status: delivered
normative: docs/adr/ADR-005-record-numbering.md
sources: [docs/adr/ADR-005-record-numbering.md, docs/architecture/record-write-pipeline.md, docs/architecture/tenant-tx-batching.md, internal/records/pipeline/alloc.go, internal/records/pipeline/pipeline.go, internal/records/pipeline/batch.go, internal/records/store/numbering.go, internal/tenancy/alloc.go, internal/entitytypes/entitytypes.go, cmd/nava-core/allocator.go]
packages: [internal/records/pipeline, internal/records/store, internal/tenancy, cmd/nava-core]
adrs: [ADR-005, ADR-001]
invariants: [I4, I8]
operations: [createRecord]
tables: [record_counters, entity_definitions, records]
tests: [TestRefusedCreatesLeaveTheCounter, TestCreateReplayTakesNoNumber, TestCreateTakesItsNumberInATransactionOfItsOwn, TestRolledBackCreateLeavesAGap, TestCreateRollsBackAtEveryStage, TestAllocationIsTheCreatesAndDatesTheRecord, TestAllocationTransactionIsTenantScoped, TestCreateChecksTheAllocation, TestAllocationTimeoutIsRetryableAndMeasured, TestAllocatorPoolDoesNotStarveThePool, TestAllocatorPoolRefusesABypassRLSRole, TestRolledBackCreateNumberIsReused, TestModeSwitchNeverReusesANumber, TestNumberingModesDoNotInterfere, TestNumbersPerTenantAndType, TestConcurrentCreatesNumberWithoutGapsThroughPgBouncer, TestFormatNumber]
---

## Purpose

Every record gets a human number, `<prefix><7 digits>` (`INC0000001`), unique
and increasing per tenant and entity type. Numbers come from one
`record_counters` upsert; there is no sequence (ADR-001 I8). A type's
`numbering_mode` picks where the upsert runs:

- `gaps` (default, ADR-005 L2s): in a transaction of its own on the allocator
  pool, committed before the record is inserted. The counter row is locked for
  one round trip. A create that fails after its number leaves a **gap by
  design**.
- `gapless` (ADR-005 L1): last in the create's own write batch, so a rollback
  gives the number back, at the cost of holding the counter lock until the
  create commits.

The decision and its rules are in the normative ADR-005 and the numbering
section of
[record-write-pipeline.md](../../../architecture/record-write-pipeline.md).

## Entry point

- `write.create` in `internal/records/pipeline/pipeline.go` reads
  `Compiled.NumberingMode()` and calls `write.allocate` (`alloc.go`) for a
  `gaps` type.
- `PoolRunner.Allocate` -> `tenancy.AllocTx` -> `store.QueueAllocate`
  (`internal/records/store/numbering.go`).
- `gapless`: `store.QueueInsertNumbered` in `write.writeCreate` (`batch.go`).
- Mode change: `entitytypes.Store.SetNumberingMode`.
- Pool wiring: `cmd/nava-core/allocator.go` (`NAVA_ALLOCATOR_DATABASE_URL`).

```mermaid
sequenceDiagram
    participant C as write.create (API pool tx)
    participant A as PoolRunner.Allocate
    participant T as tenancy.AllocTx (allocator pool)
    participant P as PostgreSQL
    C->>C: masks, defaults, validate, state, CanWrite, unique, group
    alt numbering_mode = gaps
        C->>A: allocate(createTx, definition), bounded by AllocTimeout
        A->>T: tenant = TenantOf(createTx)
        T->>P: one send: BEGIN; set_config; SET LOCAL synchronous_commit = off;<br/>record_counters upsert RETURNING next - 1, now(); COMMIT
        T-->>C: Allocation (tenant, type, scope '', epoch 0, n, instant)
        C->>C: check allocation is this create's; created_at = instant
        C->>P: INSERT record with FormatNumber(prefix, n), audit, COMMIT
        Note over C,P: failure after the allocation: n is a gap
    else numbering_mode = gapless
        C->>P: write batch: record_counters upsert + INSERT record, audit, COMMIT
        Note over C,P: rollback gives n back
    end
```

## Steps

| # | Step | Where | Invariant | Proving test |
|---|---|---|---|---|
| 1 | Take the number only after the replay check, validation, the write check and the unique and group checks | `write.create` | I4: a refused, invalid or replayed create never moves the counter | `TestRefusedCreatesLeaveTheCounter`, `TestCreateReplayTakesNoNumber` |
| 2 | `gaps`: allocate in its own transaction on the allocator pool; tenant from the create's transaction, never a parameter | `write.allocate`, `PoolRunner.Allocate`, `tenancy.AllocTx`, `store.QueueAllocate` | ADR-005 A1, A2, A4 | `TestCreateTakesItsNumberInATransactionOfItsOwn`, `TestAllocationTransactionIsTenantScoped` |
| 3 | Check the allocation names this create's tenant, type, scope `''` and epoch 0; store its `now()` as `created_at` | `write.allocate` | ADR-005 A2, A7 | `TestCreateChecksTheAllocation`, `TestAllocationIsTheCreatesAndDatesTheRecord` |
| 4 | Give up past the allocation timeout with `ErrRetryable`; record acquire and total time | `PoolRunner.Allocate`, `AllocMetrics` | ADR-005 A5, A8 | `TestAllocationTimeoutIsRetryableAndMeasured` |
| 5 | The allocator pool is separate from the API pool and logs in as `nava_app` only | `tenancy.AllocPoolConfig`, `tenancy.CheckAllocatorRole` | ADR-005 A3; I3 | `TestAllocatorPoolDoesNotStarveThePool`, `TestAllocatorPoolRefusesABypassRLSRole` |
| 6 | A failure after the allocation commits leaves a gap; numbers are never reused | `gaps_integration_test.go` | Gaps by design | `TestRolledBackCreateLeavesAGap`, `TestCreateRollsBackAtEveryStage` |
| 7 | `gapless`: upsert last in the write batch; rollback returns the number | `store.QueueInsertNumbered` | ADR-005 L1 | `TestRolledBackCreateNumberIsReused`, `TestConcurrentCreatesNumberWithoutGapsThroughPgBouncer` |
| 8 | A mode change applies to creates that start after it; both modes upsert the same row | `entitytypes.Store.SetNumberingMode` | No number used twice | `TestModeSwitchNeverReusesANumber`, `TestNumberingModesDoNotInterfere` |
| 9 | Format `<prefix><7 digits>`, growing past 7; lists sort on `created_at` | `store.FormatNumber` | Unique per tenant and type | `TestFormatNumber`, `TestNumbersPerTenantAndType` |

## Errors

| Error | HTTP |
|---|---|
| `ErrRetryable` (allocation past its timeout) | `503 retry_later`, nothing committed but possibly the number (a gap) |
| allocation not this create's (tenant, type, scope, epoch) | `500 internal_error` (a bug, never expected) |
| `entitytypes.ErrInvalidNumberingMode` | refused before anything is written |

## Notes

- Metrics: `nava_numbering_alloc_acquire_seconds`,
  `nava_numbering_alloc_seconds`, `nava_numbering_alloc_timeouts`.
- Measured create latency under both modes: [benchmarks page](../benchmarks/index.md).

## Links

- Table `record_counters`: [database section](../database/index.md).
- The allocator transaction: [core-tenancy](core-tenancy.md).
- The surrounding write: [core-write-pipeline](core-write-pipeline.md).
