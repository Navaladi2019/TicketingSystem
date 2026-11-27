<!-- TICK-1140 comment 01a21477-02b3-73df-8239-7e846ff5d0b4, 2026-11-25T20:47:53Z, archived verbatim -->

**Critique round 1, ADR-001-E4 (PR #895 at 4b1e3b49): three blocking items, each fixed by text in E4. E4-3 (a required check that always reports) is the right choice and holds.**

| # | Mechanism | Verdict |
|---|---|---|
| T1 | RS4 and V-3: a pull request edits the gate | **blocking** |
| T2 | Contract 2, RS3: a `vulncheck` result belongs to the commit, not to the pull request | **blocking** |
| T3 | E4-1: "unless it moves `go.temporal.io/server`" | **blocking** |
| T4 | Same-name question | **verify-by-spike**; partly answered from GitHub's records. **I did not run the spike** |
| T5 | E4-2 "both green" | **non-blocking** |
| T6 | `go.work` is not in the four files | **non-blocking** |
| T7 | `docs/pins.md` in FU1 | **non-blocking** |
| T8 | The rest of the contract | **confirmed sound** |

## T1. Can a pull request make `vulncheck` green without scanning? Yes (blocking)

**Known**
- The `pull_request` event runs on `refs/pull/N/merge` ([events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows)), so the pull request's own workflow, scripts and make targets judge it.
- E4 lists `check-ci-steps` as RS4's control. I ran today's `cisteps` (main 461484e7, Go 1.27.4) on edited copies of `govulncheck.yml`. **Six of eight edits pass.**

| Edit | `cisteps` |
|---|---|
| step `if: false`; make line followed by `\|\| true` | caught (2) |
| step `if: github.actor == 'nobody'` | **ok** |
| job `if: github.event_name == 'never'` | **ok** |
| `exit 0` on the line before the make line | **ok** |
| `shell: 'true {0}'` | **ok** |
| make line inside a heredoc that is only printed | **ok** |
| another job with `name: vulncheck` (the tool reads job ids, not `name:`) | **ok** |

- `detect` as specified looks only at the four `go.mod`/`go.sum` files, so a pull request that breaks the gate and nothing else is green with no scan.
- GitHub does not stop it as configured: 0 required approvals, no code owners, one collaborator (admin), owner type `User`. `review-paths.yaml` lists `.github/`, `Makefile`, `make/`, `scripts/ci/` and `tools/ci/`, but by its own header it only prints.

**Inferred:** once such an edit merges, the scheduled scan runs it too, so the 50 h bound of AR-TICK-1140-01 does not apply.

**What I need in E4**
1. Contract 2: a change to `govulncheck.yml`, the detect or gate script, `tools/ci/cisteps/`, or a line naming `govulncheck` or `GOVULNCHECK_VERSION` in `Makefile` or `make/lint.mk` scans **both** modules. This exercises an accidental break. It does not stop a deliberate one.
2. V-3: compare the `jobs:` block of `govulncheck.yml` exactly with a checked-in copy, instead of searching for a line. Tests: the six edits above, red first. E4-4 must compare the reported name (`name:`, else the job id) across every `*.yml` and `*.yaml` workflow.
3. RS4: "A pull request that edits the gate **and** its checked-in copy can report green without a scan. No GitHub setting we have prevents it. A Security review of any change to these files is the only control, also after FU2." Give it an accepted-risks row.

## T2. A green result can come from another pull request (blocking)

**Known**
- Checks attach to the commit. #816 (into `main`) and #817 (into `TICK-1031-verify`) shared head `e715ce2f`. Run 44869248489, the only one of the two whose `full-suite` ran (so #817's), is listed on #816 with `isRequired: true` for `unit`, `stack` and `isolation`. #532/#533 show the same.
- A later pass displaces an earlier non-passing check of the same name and app (T4, shown for `cancelled`).

**Failing scenario (inferred from these two facts):** branch L adds a flawed library. The pull request into `main` gets a red `vulncheck`. The same head opened into a story branch that already contains the library, or holds a fix, then gets a green one on the same commit, and the pull request into `main` is unblocked. Retargeting (RS3) is the same defect, so "revisit later" is not enough.

**Fix:** report the required name only for pull requests into `main`:

```yaml
vulncheck:
  name: ${{ github.base_ref == 'main' && 'vulncheck' || 'vulncheck-other' }}
```

Story-branch, scheduled and manual runs then never write `vulncheck`. A pull request retargeted to `main` has no `vulncheck` until its next push, so it waits instead of passing: RS3 fails closed. Nothing more is scanned, so TICK-694 stands. Add to V-4: the same head opened into `main` and into a story branch; only the first reports `vulncheck`. The name check in T1 must treat both values as taken.

**What would change my mind:** V-4 showing that a red required check stays blocking after a later green one of the same name on that commit.

## T3. E4-1 frees a version that A1 pins (blocking)

**Known:** A1 Decision 1 pins **two** versions: server v1.29.7 and `go.temporal.io/api` v1.53.0. `TestModuleIsSeparateFromSDKModule` (`temporal-server/gomod_test.go`) asserts both and forbids `replace`. E4-1 exempts only the server.

**Failing scenario:** an advisory against api v1.53.0 makes the api update "an ordinary bug fix", so an engineer edits the pin test under E4-1. The Authorizer decodes requests with that module. A new request field naming a workflow type, queue or id is outside D9 until the request-field golden is reviewed (I2).

**Text:** "A library update that clears a finding is an ordinary bug fix (as in TICK-1138), with the `temporal-server` tests run and passed, unless it moves `go.temporal.io/server` or `go.temporal.io/api`, changes an Authorizer golden file, or uses `replace`, `exclude` or a fork. Those go to the triad."

With that text the rule is sound. grpc (TICK-128), compress (TICK-583), the Go toolchain (TICK-699, TICK-962) and otel (TICK-1138) were handled as ordinary work.

## T4. Which same-named check wins (verify-by-spike)

| Hypothesis | Result |
|---|---|
| Every same-named check must pass | **Refuted for `cancelled`.** #896 (head `733b4f7d`, 20:33Z) has `temporal-server`, `unit` and other required checks twice, same app 15368, different check suites. For `temporal-server`: `cancelled` 20:10:07, `success` 20:10:37. Both are `isRequired: true`, and `mergeStateStatus` is **CLEAN**. #868 had the same pattern and merged. |
| The newest wins, or any pass wins | **Open.** Among all 889 pull requests, none into `main` has a same-named `failure` beside a pass. |

- **Not shown:** `failure` instead of `cancelled`, and two different workflows. Both #896 runs are `ci`.
- **Why no spike:** my role does not allow me to push, even for a throwaway pull request. I used the CTO's fallback: **assume the green one wins.**
- **If the newest wins:** in E4's evidence [4] the `ci` job started 37 to 39 s after the scan and finished last on all three, so it would decide. A red scan then would **not** block, and re-running the scan after a red `ci` job could hide that job. So `isRequired: true` on the scan cannot be relied on under either hypothesis.
- **Do not use `statusCheckRollup.state` as evidence:** on #896 it is `FAILURE` while the merge state is `CLEAN`.
- **Recommendation:** skip the spike. The rename makes the question irrelevant for this collision. Split it out as FU1a (two job ids, `main-red` `needs`, `required-govulncheck.yaml`) and do it first; Security rates the urgency.

If someone with push rights still runs it: a draft into `main` where `ci.yml` `temporal-server` runs `exit 1`, the scan job runs `true`, and every other required job passes. **Pass** = never mergeable, both with the scan started first and with it re-run last. `mergeStateStatus` is readable on a draft (#863), but I have not seen it on one into `main`.

## T5. E4-2: "green" must exclude skipped (non-blocking)

**Known:** #895 is `CLEAN` with `unit`, `integration`, `stack`, `isolation`, `coverage` and `temporal-server` all **skipped**, because it has no `ci:run` label. So a library pull request without the label can merge with the Authorizer tests never run; no name collision is needed. Text: "both with conclusion `success`; `skipped` or missing counts as red". AR-TICK-1140-02 should say so.

**Inferred, outside E4 (for the CTO to route to the Quality, Testing & CI Advisor):** adding any other label to a pull request starts a newer run that skips every required job on the same commit. By T4 it may hide a red `unit` or `isolation`.

## T6. `go.work` (non-blocking; none is tracked today)

**Known:** on a copy of `temporal-server/` with Go 1.27.4, a `go.work` in the parent directory with one `replace` moved `otel/sdk` from v1.47.0 to v1.45.0. The hash of `go.mod` plus `go.sum` did not change (`33a4e85f037fb696`). **Inferred:** the image is built from `temporal-server/` alone (`compose.yaml:232`), so a root `go.work` can also make the scan see a fixed version while the image ships the flawed one. Fix: `GOWORK=off` for both scans, and fail when a `go.work` is tracked.

## T7. `docs/pins.md` (non-blocking)

FU1 names line 141. Lines 142–143 also say "a new finding fails that workflow and is a new decision, not a pin bump", which E4-1 reverses.

## T8. Confirmed sound

| Point | How |
|---|---|
| Triggers, required names, `strict`, `enforce_admins` | re-read from GitHub; equal to `evidence.txt` [1] and [3] |
| Contract 1: a filtered workflow stays pending. Contract 4: a skipped job reports success | [GitHub docs](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/collaborating-on-repositories-with-code-quality-features/troubleshooting-required-status-checks); the skip is also visible on #895 |
| Contract 4: `always()`, and anything unknown is a failure | by reading: every `detect` × result pair fails closed; V-1 covers it |
| Contract 5: outside `ci.yml` | `ci.yml` `on.pull_request.types` has `labeled` |
| E4-4 rename | `ci.yml` `temporal-server` still reports the required name, so no pull request gets stuck |
| E4-5, and scanning only the changed module | no objection |

<details><summary>Rerun</summary>

T1: `go build -o cisteps ./tools/ci/cisteps`, apply one edit per copy of `govulncheck.yml` (to the `temporal-server` job), then `./cisteps -workflow <copy> -required tools/ci/cisteps/required-govulncheck.yaml`.

T2 and T4: GraphQL `pullRequest(number:N){mergeStateStatus commits(last:1){nodes{commit{oid statusCheckRollup{state contexts(first:100){nodes{... on CheckRun{name conclusion startedAt isRequired(pullRequestNumber:N) checkSuite{workflowRun{event databaseId}}}}}}}}}}` for 896, 868, 816, 817, 532 and 533.

T6: copy `temporal-server/` to a scratch directory, put this `go.work` beside it, and run `GOTOOLCHAIN=go1.27.4 go list -m go.opentelemetry.io/otel/sdk` in the module before and after:

```
go 1.27.4
use ./temporal-server
replace go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.45.0
```
</details>

`VERDICT: CHANGES` (T1, T2, T3; T5–T7 can go in the same revision)

RESULT: DONE
