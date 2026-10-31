# Review: TICK-378 parts 1 and 2 (PR #319)

Reviewer: Quality, Testing & CI Advisor. Head `08a8d71`, base `TICK-333/entity-types-versioned`.

**VERDICT: APPROVE.** No Bug, Breaks-under, Missing-test or Spec-drift findings against the code.

## What the control has to prove, and the evidence

| Claim | Evidence (GitHub Actions, not a local run) |
|---|---|
| `1ee33a7` alone fails on the old seed | `1ee33a7` was pushed on its own to a temporary branch. CI run 41251001556, `integration` job: `--- FAIL: TestRegisteredPaths/entity_types:_control:_tenant_B_reads_its_own_seeded_ticket...`, `tenant B reading its own /api/v1/entity-types/ticket = 500 {...internal_error...}, want 200`. The other six entity-type cases passed. The temporary branch has since been deleted. |
| `08a8d71` makes it pass | PR run 41250812008 and push run 41250780651: `integration` passes, the control's `--- PASS` is in the log, `ok .../internal/tenancy/isolation`. |
| No isolation case removed or skipped (D-P10) | The diff only adds lines to `entitytype_paths.go`: one new `db(...)` entry, no `t.Skip`, no removed entries. The PR log has 7 entity-type PASS lines (6 existing plus the control), and there is no `--- SKIP` or `--- FAIL` in the isolation package. The only SKIP in the job is `TestRelayProcess` in `internal/relay`, which already existed and is untouched here. |

The control checks the right things. It hits both routes. It asserts the full body: key, name, prefix, version, current, compat, the schema verbatim, sha256 recomputed from the seed text, `fixed_columns` as a non-nil map, and the state model. It also requires the two bodies to be identical. All JSON tags match the server's response structs (`internal/server/entitytypes.go:283`, `internal/metadata/metadata.go:56`, `:66`).

## Findings

1. **Preference.** The control reads only tenant B's `ticket`. The seed fix also changes tenant A's v1 row and the third definition's rows, but no positive read covers them. B's v2 exercises the same scan path, so this is optional.
2. **Preference.** `States.Transitions` is not asserted. It is decoded but never compared. This is harmless, because `transitions: []` cannot fail the scan.

## Note on the issue text (not on this PR)

The issue says `seed.sql` is the only writer of `'[]'`. `internal/db/dbtest/entity_definitions_test.go:30` (`insertVersion`) writes it too. The production claim still holds, since that is a test helper. If the part 3 `CHECK` is adopted, that migration PR has to change the helper, and it must merge after #319.
