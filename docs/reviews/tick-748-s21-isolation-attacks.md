# Review: PR #735 (TICK-748, S21 adversarial isolation tests)

Reviewer: Quality, Testing & CI Advisor. Head reviewed: `8fcff542`, plus the
allow-list fix below. Scope: TICK-748 items 1 to 6, ADR-006 D3, D6, D7, D14,
R8, R9.

## Verdict

APPROVE for the test content. Merge is held only on CI. See "CI" below.

## Rows against the TICK-748 scope

| Scope item | Row | Covered | Notes |
|---|---|---|---|
| 1 Filter oracle (hits, total, error code, `index_progress`, timing class) | 1 | Yes | Ten hidden/absent pairs over eq, in and ranges. Tenant B's fields go through four registry states and tenant A's eight answers stay byte-identical. Controls: tenant A's own state moves tenant A's answer, and the progress shown is tenant A's (13), not tenant B's (77). Timing is argued from the D7 plan (`TestListFilterPushdownIsAnIndexCond`) with no wall-clock test, which is the right call. |
| 2 ACL oracle, masked fields in every state | 2 | Yes | Desk and plain user, every operator. `cost` is masked in ready, backfilling, removing and unflagged states, with an admin control for each state. No `index_progress` or `total` in the body. |
| 3 Publish: no registry or outbox row for B; crafted payload refused | 3 (+3b pending) | DB layer yes, workflow pending | 404 bytes match the unknown key's. A forged `X-Tenant-Id` writes tenant A rows only. All seven D10 statements run as `nava_worker` with tenant B's field id: 0 rows, or FK 23503. The workflow half (3b) is registered `Pending` on TICK-745 and the Technical Lead has carried it over as a review condition there. |
| 4 Direct SQL per runtime role, D14 column grants | 4 | Yes | Roles come from `db/grants.yaml`, not a hand-written list. Each grant is checked column by column against `pg_attribute`, and each granted statement is shown to work on tenant A's own row. |
| 5 `field_not_ready` and progress carry no counts | 5 | Yes | 8 queries x 2 callers, before and after 6 more records and tenant B's state change. The member allow-list and the "no number but 422 and 37" check make it strict. |
| 6 accepted-risks only if a new risk is found | n/a | Yes | No new risk. The payload `tenant_id` gap went to TICK-745, which is correct: no client path can write that payload today. |

## Broken variants M1 to M5

I did not re-run the variants. Running them means running
`scripts/ci/isolation.sh`, which is a CI job and is not run on agent hosts. For
each one, I traced the stated failure message back to the assertion that
produces it and checked that the variant must trip it:

| Variant | Row | Message traced to | Trips the targeted check? |
|---|---|---|---|
| M1 cross-tenant registry fallback + `USING (true)` | 1 | `s21FilterOracle` base-code loop, `filter[label]=b-only` want 400 | Yes. Tenant B's `label` flag would resolve, giving 200 instead of `field_not_filterable`. |
| M2a row ACL OR-ed true on side-table filters | 2 | `s21Setup` control `s21Desk filter[qty][gte]=0 want [desk]` | Yes, but in the shared setup (see P2). |
| M2b mask check after registry resolve | 2 | `check("ready")`, `filter[cost]=abc` gives `validation_failed` | Yes. |
| M3 `rfv_text_field_fk` dropped | 3 | `s21BackfillAsA` step "index a value under the field", want class 23 | Yes. The FK is the only guard on `(tenant A, B's field id)`, because RLS WITH CHECK sees only `tenant_id`. |
| M4 `GRANT UPDATE (value) ON rfv_text TO nava_app` | 4 | `s21RoleProbe` UPDATE loop, `refused()` says "succeeded" | Yes. With no UPDATE policy, the statement matches 0 rows without an error, and `refused` rejects a nil error. |
| M5 `index_progress` + record count | 5 | `s21NotReady` want 37; 37 + tenant A's 2 records = 39 | Yes. |

## Findings (most severe first)

**B1. Bug (fixed in this review): the PR turned the `unit` job red.**
`TestD7SideTablesAreReadOnlyInTheListEXISTS`
(`internal/fieldindex/d7_test.go`) fails on 13 `rfv_*` literals in the new
`fieldindex_attack_paths.go`. The new file was not on the D7 allow-list, which
already lists `fieldindex_paths.go` and `acl_attack_paths.go`. Because `unit`
failed, `coverage` and `full-suite` were also red. I added the file to
`allowedFiles`, which is test code. `go test ./internal/fieldindex/
./internal/tenancy/isolation/` passes locally. The PR body says the full
`task test` was not run; that run would have caught this.

**S1. Spec drift (PR body vs code): row 4's "every role that writes `records`
can read the registry" is not asserted.** `s21RoleProbe` only checks registry
SELECT for the D14 roles. Either drop the sentence from the PR body or add the
assertion. The catalogue side may already be covered by `TestP701Grants`.
This is not blocking.

**P1. Preference: M1 breaks two things at once.** It shows the field-state
oracle failing. It does not show the value oracle (the hidden/absent pairs)
failing on its own. The value side is also guarded by the D7 EXISTS on tenant
A's record ids and by the older CI-2 rows, so this is optional.

**P2. Preference: M2a fails in the setup shared by rows 1, 2, 3 and 5, not in
`s21SamePair`.** The property it hits is the same: the desk's filtered list
shows the hidden record. For a cleaner demonstration next time, give each row
its own variant-specific assertion.

## CI (run 44396084358, head `8fcff542`, `ci:full`)

| Job | Result | Cause |
|---|---|---|
| lint | pass | The `workerguard/guard.go` gofmt finding does not fire in CI. |
| integration (ISOLATION=1) | pass | All five S21 rows pass. 3b is reported pending (TICK-745). |
| unit | fail | B1. This PR caused it; fixed by the allow-list line. |
| coverage, full-suite | fail | Follow from `unit`. |
| stack | fail | `sdk_ts` fails 4 of 8 with `field_not_filterable`. **Not this PR; already broken on the story branch** (see below). |
| sdk, temporal-server, changes, story-branches, runner, commit-check-budget | pass | |

`sdk_ts` is a story-branch regression. TICK-746 (#709) made a list filter on
an unflagged data field return 400 `field_not_filterable`.
`sdk/ts/test/integration/records.test.ts` (lines 42 and 72) filters on
`description` and `short_description`, which are not flagged. The case runs
only on `ci:full`, and neither #709 nor #734 carried that label, so the story
branch never ran it. It needs its own fix task: the SDK test should flag the
fields it filters on, or filter on a fixed column. It should not be fixed in
#735.
