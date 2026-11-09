# Review: TICK-622 / PR #541, S10 ACL and mask attack rows (P3-37)

- PR: #541 (`TICK-622/acl-mask-attack-tests` → `TICK-336/group-acl-masking`), head `417f29f5` (reviewed).
- Author: Security & Multi-Tenancy Reviewer. Reviewer for security-authored tests: QA.
- Spec: TICK-622 attack list and acceptance criteria; the S10 interface contract on the issue; PLAN-003 D-3-19; ADR-001 CI-8.
- Merge gate: `internal/tenancy/isolation/registry.go` changed, so `docs/delivery/review-paths.yaml` also needs the CTO's sign-off.

## Verdict

**APPROVE.** I found no Bug, Breaks-under or Spec-drift findings. There is one low Missing-test item and two Preference items. None of them blocks the merge.

**Disposition (re-review of #541 at head `6c7d3992`).** The Technical Lead decided that findings 1 and 2 land on #541 before it merges, and both are fixed at `6c7d3992`. Finding 1: when the admin explains a2, row 5 now fails if the body contains `triage`. That check sits in an attack assertion, before the `masked_fields` control, and is shown red under M5b. Finding 2: `s10-acl-attacks.md` now says that M1 proves a tenant-scope break and that the `foreign` rule has no separate defence of its own. The `cichanged` timing note needed no change. `registry.go` is unchanged since `417f29f5`.

## What I checked

### 1. Each row asserts the attack, not a control

Every control in `acl_attack_paths.go` returns an error prefixed `control:`. The attack assertions have no prefix. I traced each row:

| Row | Attack assertion (no `control:` prefix) | Controls that keep the denial honest |
|---|---|---|
| CI-8 (d) | `:435`: the desk's `filter[cost]` is 400 `masked_field` | `:431`: the admin's same filter is 200 and returns the record |
| 1 | `:368` `s10Missing` (GET, history, PATCH with none/stale/current `If-Match`), `:372`/`:376` lists, `:380` nothing written | `:354` the stored policy really names `GroupB1`; `:364` B's caller reads B's own record |
| 2 | `:305` never 412/428, then `ci8Same` byte equality with the random UUID; `:407`/`:410` nothing written | `:388`/`:391`: a reader of the same record gets 412 and 428, so the 404 comes from the ACL, not from the precondition path |
| 3 | `:480`–`:490`: 400 `masked_field`, no `items`/`total`, value not echoed, identical bytes for match and no match | `:453`–`:461`: the admin gets true totals; the desk's open-field filter works |
| 4 | `:535`–`:560`: two entries, no `cost`/`assignee` in `before`/`after`/`changed`, cost-only entry has empty `changed` | `:527` the admin's history shows both fields |
| 5 | `:595` no `triage` for the admin; `:609` cross-tenant equals random; `:624`–`:630` 403 identical for real and random record | `:582`/`:592` 200 `allow`; `:614` B's admin explains its own record |
| 6 | `:649`/`:652` 403, unchanged; `:665`/`:668` 403, no row | `:641` the desk can write; `:672` it creates in its own group; `:681` the `handoff` rule lets the same move through |
| 7 | `:757` GET equals random; `:761`/`:764` lists; `:769` explainer `deny` | `:719`/`:726` genuine tenant A shares (user and group) do grant; `:751` the forged rows exist |
| 8 | `:829`–`:842`: same 422 for foreign and unknown id, pointer and rule named, id not echoed, no version/audit written | `:783` GET gives an ETag; `:847` tenant A's own ids publish 200 |

### 2. The mutation table fails on the row it names

I did not re-run the mutations: they need the isolation harness, which is a CI job, and I do not run that on this host. I checked each one by reading the code. For each mutation I worked through `s10Setup` and the row's controls to confirm the run gets past them, then confirmed that the quoted failure is one of the row's attack assertions.

- **M1 (row 1).** Setup and the `:364` control use tenant-own data only. The first failure is `s10Missing` at `:368`.
- **M2 (row 2).** The setup control `{s10Desk, fx.admin, false}` checks only `code == 200`, so a 403 still passes it. The failure is `ci8Same` at `:308`, as quoted.
- **M3 ((d), 3).** The admin controls are unaffected. The failures are at `:435` and `:480`.
- **M4 (row 4).** The admin control is unaffected. The failure is at `:543`.
- **M5 (row 5).** a2 can read `triage`, so the a2 control at `:585` still gives `[assignee cost]`. The a1 explanation then names `triage`, and the run fails at `:595`.
- **M6 (row 6).** The failure is at `:649`.
- **M7 (row 7).** The genuine-share controls are unaffected. The failure is at `:757`. **Both layers really are needed.** The forged rows carry `tenant_id = B`, and `record_shares` RLS under `InTenantTx(A)` hides them even when the predicate is dropped. So the row stays green with only one layer removed, as the PR says.
- **M8 (row 8).** The first case returns 200 and fails at `:829`.

### 3. CI-8 (d) and `isolation-report.md`

- `records_ci8_paths.go` drops the `Pending` (d) entry. (d) is now registered from `aclAttackPaths()` with `Case: "CI-8"` and the same `records CI-8: (d)` name prefix.
- The `registry.go` change is two lines. The CI-8 `Coverage` string moves from `partial → P3-37` to `full`, and `aclAttackPaths()` is appended. No other registered path is removed or reworded.
- The CI artifact `isolation-report.md` (run 42601537981, head `417f29f5`) lists the CI-8 coverage as **full**. It has a PASS row for (d) and for S10 attacks 1–8, and no PENDING row under CI-8.

### 4. CI

`gh pr checks 541` on head `417f29f5`: `changes`, `lint`, `story-branches`, `unit`, `integration` (8m17s, which includes the isolation suite), `coverage`, `temporal-server` and `commit-check-budget` all pass. `main-red` and `stack` are skipped by design for a PR with this diff.

## Findings

### Missing test (low, optional): row 5 does not check the case where the subject can see more than the admin

`s10Explainer` checks for `triage` only when the admin explains **a1** (`:595`). In that case neither the admin nor the subject can read the field. The riskier case is explaining **a2**, who can read `triage` when the admin cannot. If a future explainer field lists the subject's *readable* fields, it would put `triage` in front of the admin, and the a2 call (`:580`) does not check for it.

**Fix:** add `|| strings.Contains(r.body, "triage")` to the a2 check at `:585` and move it out of the `control:` error. Mutation to confirm: have `Explain` return the subject's readable fields.

### Preference: row 1's mutation shows a tenant-scope break, not a foreign-rule-specific one

Under M1, tenant B's caller would probably read tenant A's record through **B's own** policy, so the `foreign` rule plays no part in the failure. This is structural, not a test gap. A foreign id in tenant A's policy can only match inside tenant A's transaction, where tenant B's principals never appear. `s10-acl-attacks.md` already lists the defence as "the policy is per tenant". I would add one sentence there saying the `foreign` rule has no independent defence layer to mutate.

### Preference / CI watch item (not this PR): `tools/ci/cichanged` timing

The PR does not touch `tools/ci/cichanged`. In this CI run it took **14.1 s** under `-race` in `unit`, more than 40× under Go's 600 s default. The 364 s isolated and 600 s+ loaded times are from the author's host. There, the package's whole-module `go list` passes (`stages_test.go:16`, `:126`) start from a cold cache while other packages compete for the machine. It is not a flake risk in CI on this evidence. If it ever approaches the limit in `unit`, give the package an explicit `-timeout` in `make test`. Do not raise the global default.
