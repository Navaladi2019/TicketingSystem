# Audit erasure and export isolation (P5-42, TICK-447)

The threat model for ADR-004's erase step (R11, R23, R24) and for the export
surface (export, stubs, lease, reseed: R17, R19-R21). The attacker holds a
valid token, or a database login, for **one tenant** and wants another
tenant's audit entries, or wants to erase or hide them.

The proofs are `TestCIA7*`, `TestCIA4*` and `TestCIA5*` in
`internal/server/audit_erasure_isolation_integration_test.go`
(`make test-integration`). They run through the real handler, sealer and
offline verifier, with the test-only erase helper `internal/audit/erasetest`
logged in as `nava_erasure` (PLAN-005 D-5-10). Each test has a positive
control, and each was shown red with its guard removed (table below).

## Attack paths

```mermaid
sequenceDiagram
    autonumber
    participant B as Tenant B (valid token / nava_erasure in B)
    participant API as nava-core (nava_app, reader pool)
    participant PG as PostgreSQL (RLS, grants, R11 trigger)
    participant V as nava audit verify (offline)

    B->>PG: erase A's entry ids with app.tenant_id = B
    PG-->>B: 0 rows (audit_log RLS); helper refuses, no audit/erase entry
    B->>PG: nava_app / nava_worker run the erase transition
    PG-->>B: 42501 permission denied (column grant), then the R11 trigger
    B->>API: GET /audit/entries:export (any filter, stubs=false, subject_id = A's record)
    API->>PG: page reads WHERE tenant_id = B, under RLS
    PG-->>API: B's rows only
    API-->>B: B's entries and stubs; no A id, hash or erasure_ref
    B->>API: export while A holds its lease
    API-->>B: 200 (the lease is a per-tenant row)
    B->>API: POST /audit/chain:reseed while A's lineage is dead
    API->>PG: reseed gate on B's head only
    API-->>B: 409 chain_alive (B is alive)
    Note over V: a forger with table access who nulls a diff without an erase entry gets "tampered"; an unsealed erase entry gives "pending"
```

## Guards and their red runs

| Guard (lowest layer first) | Test | Red with the guard removed |
| --- | --- | --- |
| `audit_log` RLS | `TestCIA4EraseNeverCrossesTenants`, `TestCIA4NoTenantContextReachesNothing` | `DISABLE ROW LEVEL SECURITY`: B erases A's entry; with no tenant the reader reaches 5 of A's rows |
| Erase-column grants (`nava_erasure` only) | `TestCIA7AppAndWorkerCannotErase` | granting `nava_app`/`nava_worker` UPDATE: the error is no longer the grant's; with the trigger also off, A's diff is erased |
| R11 `BEFORE UPDATE` trigger | `TestCIA7SecondEraseIsRejected` | trigger disabled: the second erase moves `erasure_ref` |
| R23 SELECT grant (no `diff`, `diff_salt`) | `TestCIA7ErasureCannotReadDiff` | `GRANT SELECT (diff, diff_salt)`: 5 of 8 probes read the diff |
| R24 verifier, step 7 | `TestCIA7NulledDiffWithoutEraseEntryIsTampered`, `TestCIA7EraseVerifiesPendingThenPass` | step 7 skipped: a forged erasure verifies `pass`; pending branch removed: an unsealed erase verifies `pass` |
| Export page predicate + RLS | `TestCIA4ExportAndStubsNeverCrossTenants` | predicate dropped and RLS off: B's export carries A's ids and hashes |
| Lease predicate + RLS | `TestCIA4ExportLeaseIsPerTenant` | a global lease check with RLS off: B gets 409 while A exports |
| Reseed gate predicate + RLS | `TestCIA4ReseedReadsOnlyItsOwnHead` | gate reads any tenant's head with RLS off: B (alive) reseeds |
| R6 write-time secret check | `TestCIA5ExportBytesCarryNoCanary` | `secrets.Check` off: the userinfo canaries reach the export bytes |
| TICK-552 plain-bytes control, size bound | `TestCIA5ExportBytesCarryNoCanary` | control padded to 3,198 bytes stored: fails on the 1,900-byte bound (the test also keeps a padded self-control) |
| No `cmd/*` links `erasetest` | `TestNoBinaryLinksTheEraseHelper` | `cmd/nava-core` imports it: fails |

Defence in depth seen in the red runs:

- With only `audit_log` RLS off, the masker fails closed: it cannot see
  tenant A's definition versions, so it aborts the export.
- With R6 off, the secret-property canaries are stored, but the R7 read-time
  mask still keeps them out of the export.

Neither is counted as the guard. Each test pins its own layer.

## Notes

- The ADR text says at most 10,000 ids per erase entry. ADR-004-E1 lowered
  this to 1,500 (`audit.MaxErasedIDs`), and the helper and the verifier both
  enforce that.
- An export filtered on another tenant's subject id echoes that id in the
  header and in the caller's own `audit/export` entry. The caller supplied
  the id, so this reveals nothing. The file has the same shape as one for an
  unknown id (no existence oracle).
