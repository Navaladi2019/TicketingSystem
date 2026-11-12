---
title: "8. Audit export and erasure"
section: security
status: delivered
sources: [docs/security/audit-erasure-export-isolation.md, internal/server/audit_erasure_isolation_integration_test.go, internal/audit, docs/adr/ADR-004-audit-log.md, docs/adr/ADR-004-A1-diff-read-role.md, docs/security/accepted-risks.md]
evidence: [docs/security/audit-erasure-export-isolation.md]
adrs: [ADR-004, ADR-004-A1, ADR-004-E1]
invariants: [I1, I3]
stories: [S13, S16]
tables: [audit_log, audit_chain_head]
packages: [internal/audit, internal/server]
tests: [TestCIA4EraseNeverCrossesTenants, TestCIA4NoTenantContextReachesNothing, TestCIA4ExportAndStubsNeverCrossTenants, TestCIA4ExportLeaseIsPerTenant, TestCIA4ReseedReadsOnlyItsOwnHead, TestCIA5ExportBytesCarryNoCanary, TestCIA7AppAndWorkerCannotErase, TestCIA7SecondEraseIsRejected, TestCIA7ErasureCannotReadDiff, TestCIA7NulledDiffWithoutEraseEntryIsTampered, TestCIA7EraseVerifiesPendingThenPass, TestNoBinaryLinksTheEraseHelper, TestP510bErasureCannotReadDiff, TestReseedGateIsTenantScoped]
---

## Boundary

Tenant B, holding a valid token or a database login bound to tenant B, must
not export, erase, hide or learn the existence of tenant A's audit entries.
Only the `nava_erasure` role can erase, it cannot read what it erases, and
every erasure must be visible to the offline verifier.

## Enforcement point

Lowest layer first (ADR-004 R11, R17, R19 to R24, ADR-004-A1):

- **`audit_log` RLS** on every read and every erase transition.
- **Column grants.** Only `nava_erasure` may update the erase columns;
  `nava_app` and `nava_worker` get 42501. `nava_erasure`'s `SELECT` grant
  leaves out `diff` and `diff_salt`, and only `nava_audit_reader` may read them.
- **R11 `BEFORE UPDATE` trigger.** An entry can be erased once; a second
  erase is rejected.
- **Export, lease and reseed predicates.** The export page read, the export
  lease and the reseed gate each filter on the caller's tenant, under RLS.
  The lease is a per-tenant row, so one tenant's export never blocks
  another's.
- **Write-time and read-time secret checks.** Secret values are refused at
  write (R6) and masked at read (R7), so they never reach export bytes.
- **The erase helper is test-only.** No `cmd/*` binary links
  `internal/audit/erasetest`.
- **Offline verifier, step 7.** A diff nulled without a sealed erase entry
  verifies `tampered`; an unsealed erase verifies `pending` (see
  [audit chain](10-audit-chain.md)).

```mermaid
sequenceDiagram
    autonumber
    participant B as Tenant B (valid token / nava_erasure in B)
    participant API as nava-core (nava_app, reader pool)
    participant PG as PostgreSQL (RLS, grants, R11 trigger)
    B->>PG: erase A's entry ids with app.tenant_id = B
    PG-->>B: 0 rows (audit_log RLS), no erase entry written
    B->>PG: nava_app / nava_worker run the erase transition
    PG-->>B: 42501 (column grant)
    B->>API: GET /api/v1/audit/entries:export (filter on A's record id)
    API->>PG: page read WHERE tenant_id = B, under RLS
    API-->>B: B's entries only, same shape as for an unknown id
    B->>API: export while A holds its lease
    API-->>B: 200 (the lease is per tenant)
```

## Proving tests

| Guard | Test |
|---|---|
| `audit_log` RLS | `TestCIA4EraseNeverCrossesTenants`, `TestCIA4NoTenantContextReachesNothing` |
| Erase-column grants | `TestCIA7AppAndWorkerCannotErase` |
| R11 trigger | `TestCIA7SecondEraseIsRejected` |
| No `diff` read for `nava_erasure` | `TestCIA7ErasureCannotReadDiff`, `TestP510bErasureCannotReadDiff` |
| Verifier step 7 | `TestCIA7NulledDiffWithoutEraseEntryIsTampered`, `TestCIA7EraseVerifiesPendingThenPass` |
| Export page and stubs | `TestCIA4ExportAndStubsNeverCrossTenants` |
| Export lease | `TestCIA4ExportLeaseIsPerTenant` |
| Reseed gate | `TestCIA4ReseedReadsOnlyItsOwnHead`, `TestReseedGateIsTenantScoped` |
| No secret in export bytes | `TestCIA5ExportBytesCarryNoCanary` |
| Erase helper not shipped | `TestNoBinaryLinksTheEraseHelper` |

The CI-A4 and CI-A5 tests run through the real handler, sealer and offline
verifier in `make test-integration`.

## Fixture that turns it red

Each guard was shown red with only that guard removed; the table with each
mutation and its result is in
[audit-erasure-export-isolation.md](../../../security/audit-erasure-export-isolation.md#guards-and-their-red-runs).
For example, with `audit_log` RLS disabled tenant B erases tenant A's entry,
and with the R11 trigger disabled a second erase moves `erasure_ref`. Each
test pins its own layer: the masker failing closed, or the read-time mask,
is not counted as the guard.

## Accepted risks

<!-- gen:accepted-risks ids=AR-TICK-409-06,AR-TICK-549-01 -->

## Related

- [Audit chain](10-audit-chain.md), where an erasure must stay visible.
- [Audit tables](../database/audit.md).
