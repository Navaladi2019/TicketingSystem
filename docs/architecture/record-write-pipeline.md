# Record write pipeline (P3-10, P3-11)

Scope: TICK-354, TICK-355. PLAN-003 D-3-2, D-3-6, D-3-7a, D-3-9, D-3-10,
D-3-13, D-3-14, D-3-22; ADR-001 I4, I8, I9. Code: `internal/records/pipeline` (the one entry point),
`internal/records/store` (its SQL), `outbox.EnqueueAtVersion`,
`authz.ACL` ([record-acl.md](record-acl.md)), `tenancy.TenantOf`. Tables:
[`platform-tables.md`](../data/platform-tables.md#records-pipeline-records-audit_log_v0-record_counters-p3-09).

`pipeline.Write(ctx, tx, cmd)` is the only way to create or update a record.
It runs in the caller's `tenancy.InTenantTx` transaction and never commits:
the caller commits on success and rolls back on any error, so a write leaves
the record, its audit row and whatever the `EventSink` wrote together, or
none of them.

```mermaid
flowchart TD
    A[check command shape] --> B[tenant = tenancy.TenantOf tx<br/>actor.TenantID must match]
    B --> C{idempotency key?}
    C -- same entity type, op, record --> R[return the earlier Result]
    C -- another write holds it --> X[ErrIdempotencyConflict]
    C -- none / not found --> D[load entity type + current version<br/>update: load record, CanRead<br/>unreadable = ErrNotFound, then check version]
    D --> E[update: JSON Merge Patch<br/>onto the stored payload]
    E --> G[ApplyDefaults from current version]
    G --> H[Validate against current version]
    H --> S[state model: create in Initial,<br/>update only along a listed transition]
    S --> I[split fixed columns out of data]
    I --> F[Authorizer.CanWrite before row, after row<br/>skipped for rule, workflow, function, system]
    F --> U[each unique property set or changed:<br/>advisory xact lock, then SELECT]
    U --> GR{assignment_group_id<br/>set or changed?}
    GR -- yes: not a live group of the tenant --> GX[ErrAssignmentGroupRemoved<br/>422, nothing written]
    GR -- no, or live --> N
    N[create: record_counters upsert RETURNING next - 1<br/>in its own committed transaction on the allocator pool]
    N --> J[INSERT, or UPDATE ... version = version + 1<br/>WHERE tenant_id, id, version = expected]
    J --> W[keyed write: one record_write_keys row<br/>D-5-3, P5-13a]
    W --> K[one audit_log entry<br/>(audit.Append, ADR-004)]
    K --> L[EventSink.RecordChanged in tx]
    L --> M[Result]
```

## Rules the code holds

- **Tenant from the transaction.** `Write` reads the tenant with
  `tenancy.TenantOf` and refuses a command whose `Actor.TenantID` differs
  (`authz.ErrForbidden`). Every statement also names that tenant
  explicitly, on top of RLS. Another tenant's record or entity type is
  `ErrNotFound`, the same as a missing one.
- **One version bump per write.** A create inserts version 1. An update's
  `UPDATE` is the only statement that bumps `version`; no row at the
  expected version is `ErrVersionConflict`, and so is a stale
  `ExpectedVersion` seen at load. A sink that enqueues outbox rows must use
  `outbox.EnqueueAtVersion(ctx, tx, entry, ev.Version)`, which locks the row
  without bumping it again. outbox has no bumping call (TICK-411).
- **Current version wins.** An update validates against the current
  definition after its defaults, and moves `schema_version` to it. The stored
  payload is rebuilt with the mapping of the version it was written under.
- **Fixed columns.** A property mapped to `state`, `requester_id`,
  `assignee_id` or `assignment_group_id` is stored only in its column.
  `store.Record.Payload` merges it back on read. The state column is never
  NULL: a create with no state gets the initial state, and an update that
  removes it is a `required` field error.
- **Audit.** One row per write: `action` is `create` or `update`, `before`
  (NULL on a create) and `after` are snapshots
  `{"number", "version", "schema_version", "data"}` with `data` the merged
  payload. Until P5-13b the pipeline still copies the idempotency key onto
  this row, but nothing reads it there.
- **Idempotency (system writers only).** A user write with a key is
  `ErrInvalidCommand`. The key lives in `record_write_keys` (PLAN-005 D-5-3,
  P5-13a), one row per keyed write inserted in the write's transaction, after
  the record and before the audit row. Each principal has its own key space
  (TICK-480, TICK-626): the row carries the writer's `actor_principal_id`
  (NULL for a system writer), and `record_write_keys_actor_key` is unique on
  `(tenant_id, idempotency_key, actor_principal_id) NULLS NOT DISTINCT`, so
  another principal's write under the same key is neither replayed nor
  refused, and a 409 never reveals that the key exists (AR-TICK-480-01;
  [record-write-keys-per-principal.md](../data/record-write-keys-per-principal.md)).
  The key is looked up there first, by the caller's principal; a hit for the
  same entity type (by `definition_id`), op and (on an update) record returns
  the earlier `Result` (record id, number, version, schema version) from that
  row and writes nothing. Replay never reads the audit log. A hit for any
  other write, or an entity type the tenant does not have, is
  `*ErrIdempotencyConflict` (P3-14 maps it to 409, as PLAN-001 T22 does for a
  different body). A concurrent duplicate by the same principal waits on the
  first write's uncommitted key and then hits `23505` on
  `record_write_keys_actor_key`, which aborts the transaction: `Write` returns
  `*ErrIdempotencyConflict{Concurrent: true}`, and the retry replays.
- **Numbering (D-3-9 as amended 2026-11-10; ADR-005 L2s for `gaps` types).**
  A create takes its number with one upsert, `INSERT INTO record_counters ...
  VALUES (..., 2) ON CONFLICT ... DO UPDATE SET next = record_counters.next +
  1 RETURNING next - 1, now()`, after the replay check, validation, the
  write check and the unique and group checks (ADR-001 I4, ADR-005 A6), so a
  refused or invalid create never moves the counter. It runs through
  `tenancy.AllocTx` on the dedicated allocator pool: one pipelined send of
  `BEGIN; set_config(app.tenant_id, tenant, true); SET LOCAL
  synchronous_commit = off; upsert; COMMIT`, with the tenant taken from the
  create's own transaction (A1, A2, A4). The allocation commits before the
  record is inserted, without waiting for its WAL flush: the record's own
  synchronous commit flushes it (AR-TICK-653-03). The counter row is locked
  only for that one round trip. The create checks the allocation (tenant,
  type, scope `''`, epoch 0) against its own transaction, and stores the
  allocation's `now()` as `created_at` (A7). An allocation that takes longer
  than 200 ms fails the create with `ErrRetryable` (A5). A create that fails
  or rolls back after its number leaves a gap; a retry takes a new number.
  Numbers stay unique and increasing per tenant and type. Nothing creates the
  counter row in advance, and there is no sequence (I8). A type whose
  `entity_definitions.numbering_mode` is `gapless` (TICK-659) takes its
  number with L1 instead: the same upsert, last in the create's own write
  batch (the TICK-648 layout), so a rollback gives the number back. The
  create reads the mode with its definition (`Compiled.NumberingMode`); a
  mode change applies to creates that start after it, and both paths upsert
  the same row, so no number is used twice (ADR-005 §2.1.1,
  `TestModeSwitchNeverReusesANumber`).
- **Allocator pool (ADR-005 A3, A5, A8).** The allocation needs a connection
  while the create's own transaction holds one, so nava-core gives it a pool
  of its own (`cmd/nava-core/allocator.go`, 4 connections, `pool_max_conns`
  in the DSN overrides it and must match `nava_alloc pool_size`) and, behind
  PgBouncer, a server pool of its own (the `nava_alloc` entry,
  `NAVA_ALLOCATOR_DATABASE_URL`, required with the record routes: there is
  no fallback to the API pool's DSN). The pool logs in as `nava_app` only:
  startup refuses another login, and each connection is checked for
  SUPERUSER and BYPASSRLS as it opens (`tenancy.AllocPoolConfig`). Metrics:
  `nava_numbering_alloc_acquire_seconds`, `nava_numbering_alloc_seconds`,
  `nava_numbering_alloc_timeouts`. On the API pool, or on the API pool's
  PgBouncer server pool, creates that hold every connection would starve
  their own allocations (`TestAllocatorPoolDoesNotStarveThePool`).

```mermaid
sequenceDiagram
    participant H as create (API pool)
    participant A as allocator pool
    participant P as PostgreSQL
    H->>P: BEGIN+tenant; prefetch; checks (CanWrite, unique, group)
    H->>A: AllocTx(createTx) - tenant = TenantOf(createTx)
    A->>P: one send: BEGIN; set_config; SET LOCAL synchronous_commit = off; upsert RETURNING next - 1, now(); COMMIT
    A-->>H: (tenant, type, scope '', epoch 0, n, instant)
    H->>P: batch: INSERT record (number n, created_at = instant), write key, audit
    H->>P: COMMIT, synchronous (a failure here leaves n as a gap)
```

  Format `<prefix><7 digits>` (`INC0000001`), growing past 7 digits; past
  that the text does not sort by number, so lists sort on `created_at`.
- **State model.** A create must be in the type's `Initial` state. An
  update that changes the state must follow a listed transition; keeping the
  state is no transition. Otherwise `*ErrIllegalTransition{From, To}` (`From`
  empty on a create).
- **Unique properties (D-3-22).** For each `"x-nava-unique": true` property
  set on a create, or changed on an update, in property order: take
  `pg_advisory_xact_lock(hashtextextended(tenant ‖ type ‖ field ‖ canonical
  value, 0))`, then a plain `SELECT` for another record of the type holding
  the value (jsonb equality on `data`, or the fixed column). A match is
  `*ErrUniqueViolation{Field}`. The lock is released at commit, so it is safe
  under PgBouncer transaction pooling, and the waiter's `SELECT` (READ
  COMMITTED) sees the holder's committed row. The canonical value makes
  values jsonb calls equal share a lock: numbers as exact rationals, object
  keys sorted, fixed uuid columns lowercase. A missing or null value is not
  checked. There is no JSONB unique index (D25); the check scans the type's
  records in the tenant.
- **Removed assignment group (ADR-004-E5 D6, D7, N4; TICK-537).** When a
  write sets `assignment_group_id` (a create with any non-null value) or
  changes it, `store.GroupLive` reads `groups` in the write's transaction,
  under RLS, never from a cache. A tombstoned group, another tenant's group
  (live or tombstoned) and an unknown id are all `ErrAssignmentGroupRemoved`,
  before the counter, the record, `record_write_keys`, the audit entry and the
  sink, so nothing is written. Because the check runs before the `INSERT` or
  `UPDATE`, neither ends at a `23503` on `records_assignment_group_fkey`. An
  unchanged value passes, even when its group is tombstoned, and clearing
  the field passes. Reads (`store.Get`) join `groups` and set
  `AssignmentGroupRemoved`; the API returns it as
  `assignment_group: {id, removed}`. Nothing clears or reassigns the field.

## Errors

| Error | Meaning |
| --- | --- |
| `ErrInvalidCommand` | Caller bug: bad op, source, key or causation, or no tenant on tx |
| `authz.ErrForbidden` | Actor of another tenant, or the Authorizer refused the write (before or after row) |
| `ErrNotFound` | No such entity type or record in the tenant, or (update) a record the actor cannot read |
| `*ErrValidation{Fields}` | Schema failures, JSON pointers and keywords only |
| `*ErrIllegalTransition{From, To}` | State change not in the state model, or a create not in `Initial` |
| `*ErrUniqueViolation{Field}` | Another record of the type holds the unique value |
| `ErrAssignmentGroupRemoved` | `assignment_group_id` set or changed to a group that is not live in the tenant (422 `assignment_group_removed`) |
| `ErrVersionConflict` | Not at `ExpectedVersion` |
| `*ErrIdempotencyConflict{Concurrent}` | Key held by another write (409), or a concurrent duplicate: roll back and retry |

## Stubs and later tasks

- `TxRunner` is taken and held for P3-12 (TICK-88), which writes a rejected
  cascade's audit row on it (D-3-7a). P3-12 also adds the depth and cycle
  checks.
- `NoopSink` is wired in S9. S11b's sink (P3-42) enqueues through
  `EnqueueAtVersion`.
- The S9 interim authorizer is deleted (P3-31, AR-S9-01 closed); `authz.ACL`
  decides every user and AI write ([record-acl.md](record-acl.md)).

## Tests

P3-11's concurrency tests (`numbering_integration_test.go`) reach the
platform database through the CI PgBouncer in transaction mode
(`scripts/ci/pgbouncer/pooled.ini`, pool of 5 server connections, so 50
clients share 5), as nava-core does on the compose stack.

`internal/records/pipeline/*_integration_test.go` (tag `integration`, real
PostgreSQL, `nava_app` under RLS) cover the P3-10 acceptance criteria. The
kill test re-runs the test binary as a child process (as
`internal/relay/dispatch_integration_test.go` does), parks it in the
test-only `beforeReturn` hook after every statement and before commit, kills
it with SIGKILL, and checks that no record or audit row is left.

## Rollout order: the audit_log_v0 contract step (P5-17, TICK-435)

Moving record history from `audit_log_v0` to `audit_log` is an expand and
contract change (Postgres review of PR #438). The decision and its trigger are
in `docs/delivery/decisions/2026-11-04-p5-17-drop-v0-with-the-reader.md`.

1. **Expand (P5-13b, already deployed).** The binary writes `audit_log`
   through `audit.Append`. `audit_log_v0` is still there, read-only for the
   runtime roles, and the old history endpoint reads it.
2. **Deploy the new binary, then run `nava-db migrate` at once.** The new
   binary reads a record's history from `audit_log` through
   `internal/audit/read` and never names `audit_log_v0`. The reader's
   `SELECT` on `audit_log` (`db/grants.yaml`) and the drop
   (`20261104100000_drop_audit_log_v0.sql`) are applied by the migrate. So
   **between the binary deploy and `nava-db migrate`, audit list and history
   reads fail**; record writes keep working. Run the two back to back.
3. **Never migrate first.** Revoking INSERT on the table (P5-13b) and dropping
   it are contract steps: an old binary that still reads `audit_log_v0` for
   history breaks once it is gone.

**Zero-downtime alternative.** For a stack that must not lose reads, split the
release: migrate with the reader grant (no drop), deploy the binary, then drop
`audit_log_v0` and remove its grant in a later release.

Rows of `audit_log_v0` are not copied to `audit_log` (S9 ran on development
stacks); a record written before P5-13b shows an empty history after the drop.
The drop is not reversible; restore from backup if the rows are needed.
