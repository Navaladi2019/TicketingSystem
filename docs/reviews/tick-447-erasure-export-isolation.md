# Review: TICK-447 (P5-42), erasure and export isolation tests, PR #515

Reviewer: Quality, Testing & CI Advisor. Head `b3ed4512`, base `TICK-421/audit-export`.
Read against ADR-004 R11, R17, R19-R24, ADR-004-E1, PLAN-005 D-5-10 and the TICK-449 grooming additions.

**Verdict: APPROVE.** No Bug, Breaks-under or blocking Missing-test findings.
**Merge gate still open:** `internal/tenancy/isolation/registry.go` is a CTO sign-off path in `docs/delivery/review-paths.yaml`, and the CTO has not signed off yet.

## CI

Every job on `b3ed4512` is SUCCESS: `changes`, `lint`, `story-branches`, `unit`, `temporal-server`, `integration`, `commit-check-budget`, `stack` and `coverage`. `main-red` was skipped, as expected for a story-branch PR. The first run's red `unit` (the D-3-6 gate, `TestRecordWritersOnlyInPipeline`) is fixed by `b3ed4512`.

## Acceptance criteria vs tests

| Criterion | Test | Positive control | Red run in PR |
| --- | --- | --- | --- |
| CI-A7: erase, then pending, then pass (also under filters); R24 carries the erase entry | `TestCIA7EraseVerifiesPendingThenPass` | seq and entry_hash unchanged; erase entry shape checked | verifier pending branch removed |
| CI-A7: second erase rejected | `TestCIA7SecondEraseIsRejected` | first erase succeeds | R11 trigger off |
| CI-A7: `nava_erasure` cannot read `diff` | `TestCIA7ErasureCannotReadDiff` (8 probes) | M1 columns readable | `GRANT SELECT (diff, diff_salt)` |
| CI-A7: app and worker cannot erase | `TestCIA7AppAndWorkerCannotErase` | `nava_erasure` erases the same row | column grant, then grant plus trigger |
| CI-A7: nulled diff with no erase entry is `tampered` | `TestCIA7NulledDiffWithoutEraseEntryIsTampered` (3 forgeries incl. wrong-ref) | restore, then `pass`; honest erase `pass` | verifier step 7 skipped |
| Grooming: B erasing A's ids updates 0 rows and writes no erase entry | `TestCIA4EraseNeverCrossesTenants` | A erases its own | `audit_log` RLS off |
| CI-A4: exports and stubs never cross tenants | `TestCIA4ExportAndStubsNeverCrossTenants` (10 filters) | A's own export carries every needle | page predicate plus RLS |
| Grooming: no cross-tenant 409 `export_in_progress` | `TestCIA4ExportLeaseIsPerTenant` | A's own second export gets 409 | global lease plus head RLS |
| Grooming: reseed `chain_alive` reads only B's head | `TestCIA4ReseedReadsOnlyItsOwnHead` | A's own call, both ways | reseed gate reads any head |
| CI-A4: no tenant reaches zero rows | `TestCIA4NoTenantContextReachesNothing` | each probe in tenant A | RLS off |
| CI-A5: export bytes carry no canary; negative control; uncompressed and `< 1900` | `TestCIA5ExportBytesCarryNoCanary`, `auditControlsUncompressed` | planted canary found; padded control fails the bound (3,198 bytes) | `secrets.Check` off; padded control |
| Grooming: no `cmd/*` imports `erasetest` | `TestNoBinaryLinksTheEraseHelper` | the scan sees the helper in the server test binary | `cmd/nava-core` imports it |

Every criterion has a test, a positive control and a red run. The tests run through the real handler, sealer and verifier, and the helper never names the tenant, so the CI-A4 tests prove RLS and not the helper.

## D-3-6 gate allowance (`internal/records/arch_test.go`)

The allowance is the right shape. It covers one exact file, `UPDATE` only, and a SET list limited to `diff`, `diff_salt`, `erased_at` and `erasure_ref`, reusing the fail-closed parser that `sealColumnsOnly` already uses (TICK-581). `TestEraseColumnsOnly` pins the following:
- the row-constructor form `SET (a, b) = (...)` is refused;
- a stacked second statement is refused;
- a stray `seq` or `tenant_id` is refused;
- a sibling file in `erasetest/` gets no allowance;
- `seal.go` cannot use the erase allowance, and `erasetest.go` cannot use the seal allowance.

The allowance is a test-file change, but not a weakening: the trigger still enforces the R11 transition, and the gate now checks the SET list itself.

## Findings

1. **Missing test (low, non-blocking).** The `audit.MaxErasedIDs` cap of 1,500 is only unit-tested (`normalise`). No integration test runs an erase at the cap against PostgreSQL. A 1,500-id `audit/erase` diff through `audit.Append`, the sealer and the verifier is the largest entry this path writes, and a size limit in any of them would show up only at the boundary. Add one `Erase` of 1,500 ids followed by `sealAll` and a `pass` export when Epic 30's erasure job is built, or here if cheap.
2. **Spec drift (issue text, not code).** The issue description and R23 say "≤ 10,000 ids", but the code follows ADR-004-E1's cap of 1,500. The code is right. The Technical Lead should update the TICK-447 contract line so the next reader does not "fix" it.
3. **Preference.** The `1900` bound now exists twice: `auditMaxControlBytes` in `internal/tenancy/isolation` and `a5MaxControlBytes` in `internal/server`. One exported constant would stop them drifting apart. This is optional.

## Not verified by me

I did not reproduce the 13 red runs locally (CI rule). I accept them from the PR body's excerpts, which match each test's failure messages.

## Merge order

PR #516 (TICK-624) also targets `TICK-421/audit-export` and changes `internal/server/audit_export.go`, the handler these tests call. Whichever of #515 and #516 merges second needs a fresh `ci` run on the updated base before it merges.
