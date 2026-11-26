<!-- TICK-1140 comment 01a2149d-6f9b-7635-9740-376eb612f65c, 2026-11-25T21:29:51Z, archived verbatim -->

**Security review of ADR-001-E4 (PR #895 at 4b1e3b49, `main` 461484e7): no isolation gap is exploitable today, and no tenant token reaches any of this. E4-3 (a required check that always reports) is right. E4-1, E4-2 and the two risk rows need text changes.**

## Answers to the CTO

| Question | Answer |
|---|---|
| AR-TICK-1140-01 | **Medium, confirmed.** The row must change (S1, S3) |
| AR-TICK-1140-02 | **Medium, confirmed**, assuming the green one wins. The row must change (S4) |
| RS4, a pull request edits its own gate | **Medium. Needs its own row, AR-TICK-1140-03**; I agree with T1. Owner: Security & Multi-Tenancy Reviewer |
| Must the name collision be fixed ahead of the rest of FU1? | **Yes**, as its own pull request (FU1a), before the next merge that changes a `go.mod` or `go.sum`, and within a week. It need not stop other merges. `strict: true` means every open pull request must take the rename before it merges |
| May E4-1 treat a clearing update as an ordinary bug fix? | **Yes, with limits** (S2). As written, **no** |

**Why AR-02 is not High.** Four things must all happen: a library update breaks the Authorizer, the tests catch it, GitHub lets the green scan decide, and the reviewer misses the red job. The push to `main` then runs the real `temporal-server` job, because the module changed. For #890 that was run 45685089268: the job finished 4 minutes after the merge. A red one alerts on #259. The row credits only the daily run.

## Findings

| # | Finding | Severity | Status | Blocks E4? |
|---|---|---|---|---|
| S1 | The scan can judge a different Go than the one we ship | Medium | will bite us at scale; mechanism proven | **yes** (text) |
| S2 | E4-1 is too wide, and a fast CI run is not enough evidence | Medium | will bite us at scale | **yes** (text) |
| S3 | AR-01's "about 50 hours" has conditions it does not state | Low | will bite us at scale | **yes** (text) |
| S4 | E4-2 and AR-02 do not say that a skipped job is red | Medium | will bite us at scale | **yes** (text; same as T5) |
| S5 | E4-4 relies on a list that nothing keeps equal to GitHub | Low | will bite us at scale | no; a simpler rule exists |
| S6 | A later `skipped` run may hide a red `isolation` | Medium | **not verified** | no; outside E4 |

### S1. The scanned Go is not always the shipped Go
- **Attack:** an engineer clears a standard library finding (as in TICK-699) by raising `toolchain` in `temporal-server/go.mod` and leaves `FROM golang:` alone. The scan goes green, the image still ships the flaw, and an attacker who can reach that code uses it. The same gap exists for `nava-core`, which tenants reach directly. Lowering `FROM` alone has the same effect and triggers no scan.
- **Known:** the scan takes Go from `go-version-file: temporal-server/go.mod`. The official image template sets `GOTOOLCHAIN=local`; with that setting, Go 1.22.2 here built a module that says `toolchain go1.99.9` (exit 0). I set the Dockerfile and its `docs/pins.md` row to 1.27.0, and `checkpins` said ok. Today both are 1.27.4, so nothing is wrong now.
- **Fix:** E4 says a standard library finding is cleared only when `toolchain` and every `FROM golang:` of that module match. FU1 adds that check. T1's list of files that force a scan also needs the Dockerfiles.
- **Test:** `TestScannedToolchainIsTheShippedToolchain` (attached): green today, red on the lowered Dockerfile.

### S2. E4-1
- **I agree with T3** (`go.temporal.io/api` is a pin too). Three things are missing:
  1. **`ci:full`.** The label `ci:run` gives a fast run, which skips `isolation` and the Temporal isolation, custody, handshake and operator-reach stack cases (AR-TICK-652-01). Those test the Authorizer in a running server. #579, #751 and #890 all had `ci:full`, but nothing requires it.
  2. **Libraries on the trust path.** `authz` imports `protobuf` reflection, and the module imports `grpc` and its credentials package. grpc was an ordinary bump before (TICK-128), so triad review is not needed. Security review is.
  3. **Which module.** A1 Decision 3 never covered the root module. If the sentence covers both, `jwx` (token validation) and `pgx` (tenant transactions) belong on the list.
- **Text:**

> A library update that clears a finding is an ordinary bug fix (as in TICK-1138) when its `ci:full` run has `temporal-server`, `unit`, `integration`, `isolation` and `stack` with conclusion `success`. It goes to the triad if it moves `go.temporal.io/server` or `go.temporal.io/api`, changes an Authorizer golden file, or uses `replace`, `exclude` or a fork. It needs a Security review if it moves `grpc` or `protobuf` in `temporal-server/`, `jwx` or `pgx` in the root module, or changes a test assertion. A standard library finding is cleared only when `toolchain` and every `FROM golang:` of that module match.

### S3. AR-TICK-1140-01
- State the conditions of the 50 h: the gate is unedited (RS4) and the scanned Go is the shipped Go (S1).
- 50 h is the time to a red run, not to a reader. Until FU3 the alert is one more comment on #259. Name FU3 as part of this row's fix.
- No alarm watches "more than 72 h between scheduled runs". Say so.
- The row names only the Authorizer. The module a tenant can reach is the root one (`nava-core` parses tenant tokens and JSON). Name it.
- Add a trigger: **the first real tenant** → ask the owner again about a daily scan. TICK-694 stands until then.

### S4. AR-TICK-1140-02 and E4-2
- `skipped` or missing counts as red for `temporal-server` on a library pull request. #895 itself is `CLEAN` with it skipped.
- The collision part of the row ends at FU1a, the `root` part at FU2. Write both triggers.
- Bounded by: add "the push to `main` runs the `temporal-server` job".
- **Until FU1a:** a reviewer opens the `ci` run by its run page. The check name in the merge box is not evidence.

### S5. Unique names without a list
Rule: **no two workflows report the same check name**, over every job of every workflow. No list of required names, so nothing can drift. One extra rename is needed: `main-red` in `govulncheck.yml`. T2's `name: ${{ … }}` needs both its values read; the attached test does not do that yet.
- **Test:** `TestNoTwoWorkflowsReportTheSameCheck` (attached). **Red today** on `temporal-server` and `main-red`. Green after renaming the three jobs, `needs` and `required-govulncheck.yaml`; the existing `cisteps` tests `TestRepository` and `TestDroppingAnyRequiredStepFails` stay green. That rename plus this test is FU1a.

### S6. For the CTO to route (Quality, Testing & CI Advisor)
Adding any label except `ci:run` or `ci:full` after a red `isolation` starts a run that reports it `skipped` on the same commit (`ci.yml`, the `if:` of `changes`). If the newest result wins, the required isolation check stops blocking. This is closer to tenant isolation than the scan.
- **Not verified, and no case found.** I read the last commit of all 398 pull requests into `main`. None has a red required check beside another result of the same name. A re-run inside one workflow run is not visible there.
- **Fix:** required checks come from jobs that always report, as E4-3 does. **This is the question worth a spike; the `temporal-server` one is not** (I agree with T4).

## The Technical Architect's critique
T1, T2, T3 and T5: I agree, Medium each. T6: agree, Low. I did not repeat T1's edits or T6's `go.work` run.

## Not checked
- The same-name spike was not run. Nothing was pushed.
- I did not run the `temporal-server` tests or a scan.
- The attached patch adds two test files only: names into FU1a, toolchain into FU1.

`VERDICT: CHANGES` (S1 to S4, as text in E4 and the register; no exploitable isolation gap, so no escalation to the human)

RESULT: DONE
