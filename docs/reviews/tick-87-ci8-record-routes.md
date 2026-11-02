# Review: TICK-87 / PR #383, CI-8 on the record routes

- PR: #383 (`TICK-87/ci8-record-routes` → `TICK-335/records-pipeline`), head `6072442` (reviewed `e680c49` plus the Postgres-review commit `6072442`)
- Author: Security & Multi-Tenancy Reviewer. Primary review: QA. Second review (D-P10): Postgres & Data Modeling Specialist.
- Spec: TICK-87 contract and acceptance criteria; PLAN-003 D-3-10, D-3-19; ADR-001 CI-8.

## Verdict

**APPROVED.** No Bug, Breaks-under or Spec-drift findings. There is one low Missing-test item, which is optional.

## What I checked

### 1. The `ci8Leaky` mutation control really fails (e) and (f)

I traced `ci8Leaky` (`records_ci8_paths.go:314`) through both cases:

- **(e)** For tenant A's id, the lookup past RLS (`e.super`) finds the row, `If-Match` is set, and the handler answers `412` with an ETag. `ci8ETagCases` returns its explicit "never 412" error at `:257` on the first tag (`W/"2"`). The random UUID isn't found, so it falls through to the real handler and gets 404. That keeps the control's "missing" leg honest.
- **(f)** For tenant A's id, `If-Match` is empty, so the handler answers `428`. `ci8Same` fails at `:154` because `other.code != 404`.
- `ci8Mutation` (`:302`) turns "leaky passes" into a red path. So if a later edit makes either check vacuous, the run goes red instead of passing silently. The red output in the PR description matches the traced errors.
- The positive controls at `:270` and `:292` (B's own record answers 412 and 428) show the route reaches the version and precondition checks. A 404 here means the tenant scope, not a dead route.

### 2. `X-Request-Id` is the only exception to the byte-identical comparison

`ci8Same` (`:150`) compares the full status (both must be 404) and the body as an exact string. It compares the set of header keys and every value of each key. The one key it drops is `requestid.Header`, which is `"X-Request-Id"`, already in canonical form, so the filter matches. Nothing is stripped from the body: no `instance`, as D-3-10 requires, and that is asserted. Since headers must be equal, the "no ETag" assertion on the missing response also covers the other one.

### 3. `registry.go`

The diff is one `append(paths, recordCI8Paths()...)` line plus the CI-8 `Coverage` string. No registered case is removed, skipped or reworded. The new (d) is registered as `Pending: "P3-37 ..."`, and the coverage text names P3-37 / S10 as the only outstanding item. That satisfies acceptance criterion 2 and stays within the Technical Lead's ruling.

### 4. Acceptance criteria

- AC1: Each case has one named, `CI-8`-tagged path: (a), (b), (c), (e), (f), plus pending (d). Red, then green, is shown in the PR, and the red half is also committed as the mutation control.
- AC2: The report text shows only the masked-filter case outstanding, marked S10 / P3-37.

## Findings

1. **Missing test (low, optional):** `ci8Mutation` (`records_ci8_paths.go:303`) accepts *any* error from the leaky run as proof, including a `control:` error or a DB error from `ci8Unchanged`. Today the leaky run fails for the right reason (traced above). But suppose a future edit to `ci8Leaky` also intercepted the random-UUID request. The "missing" control would then fail with `control: ... want 404`, and the mutation control would still count it as a success, even though nothing exercised the oracle check. Suggested hardening: have `ci8Mutation` reject errors that start with `control:`, or return a sentinel from the oracle and 404-mismatch branches and match it with `errors.Is`.

## Routed elsewhere

- `ci8Unchanged` and the fixture audit count read `records` and `audit_log` through `e.super`, past RLS. For a "nothing changed" assertion, that is the stronger read: it would see a row written under any tenant. Whether it is the right context is the Postgres reviewer's call under D-P10, so I'm not ruling on it here.

## Verification

- Local: `go vet -tags integration ./internal/tenancy/isolation/` is clean, and `go test -count=1 ./internal/tenancy/isolation/` is ok. I did not run the isolation stage locally (that's a CI job).
- CI: GitHub Actions on `6072442`, every job green (both `ci` runs succeeded) (`gh pr checks 383`: changes, story-branches, lint, unit, integration, temporal-server, stack-internal, stack, stack-db pass; main-red skipped as expected).

## Addendum: `6072442` (Postgres review follow-up)

Re-read it. The fixture now requires exactly 2 audit rows after create plus one patch, so "no new audit row" can no longer pass against a pipeline that stopped auditing. `ci8Unchanged` checks that exactly one row with A's id exists across all tenants, then reads it scoped to `tenant_id = TenantA`. Both changes make existing assertions stricter. Nothing was removed, and my verdict stands.
