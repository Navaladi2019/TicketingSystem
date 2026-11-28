# Review: PR #972, TICK-1188 check-pins enforces one govulncheck version (V-6) and the module layout (V-7)

Reviewer: Quality, Testing & CI Advisor. Head reviewed: `agent/security-multi-tenancy-reviewe/7b2b74316f3c` at `dee6746a`. Specs read first: the TICK-1188 contract (A7.2 and rule 7 as amended), its test list, and the existing `checkSameGo` code in `tools/ci/checkpins/samego.go`.

**Verdict: APPROVE.** The checks match the contract. Every listed test exists, failed at the red commit and passes at head. CI is green, with lint and unit included. The findings below are all Preference: none blocks the merge.

## Closing the items the implementer could not run

| Item | Result |
|---|---|
| Red commit comes before the check | `3e38d92a` adds only `adversarial_vulncheck_test.go`. `dee6746a` adds `vulncheck.go` and wires it in. I re-ran the package at `3e38d92a`: all 15 V-6 red cases and all 6 V-7 red cases failed with `exit 0, want 1`. The green cases passed, including `repository`, `tools/ci/checkpins/testdata/x/go.mod` and the three tooling modules. Head adds two "file missing → exit 2" cases. |
| Package tests at head | `go test -count=1 ./tools/ci/checkpins/`: ok. `go run ./tools/ci/checkpins`: `check-pins: ok`. |
| Full lint | CI `lint` job passed in run 46032280088 and run 46034094365. `make check-pins` printed `check-pins: ok`, and `check-ci-steps` passed. |
| Full unit | The first run was lint-only because the PR had no `ci:run` label. I added `ci:run`. Run 46034094365 passed: lint, unit, integration, isolation, coverage and commit-check-budget. In `unit`, `tools/ci/checkpins` reported ok, and `adversarial_vulncheck_test.go` ran 2 of 2 tests. |
| govulncheck workflow | run 46032279964: `vulncheck-root`, `vulncheck-temporal-server` and `vulncheck` all passed. |

## Contract coverage

| Contract item | Code | Test |
|---|---|---|
| Version read as text from pins row, `Makefile`, workflow `run:` steps parsed as YAML | `checkVulnVersion` | only in workflow / Makefile / pins: red |
| Exactly one line per scan job, none elsewhere | `seen`/`isScan` loop | third job (two variants), two lines (two variants), none, job removed: red |
| Tracked `go.work` | existing `checkSameGo` | existing tests |
| `vendor/` in the root module or `temporal-server/` | `checkLayout` + `inScannedModule` | `vendor/`, `temporal-server/vendor/`, `internal/pkg/vendor/`: red. `tools/navalint/vendor/`: green |
| `go.mod` outside the set, except under `testdata/` | `checkLayout` | `internal/foo`, `temporal-server/internal/x`, `tools/ci/checkpins`: red. `testdata/x`: green |
| Messages name the file, the value and the test id | all messages | `mustCLI` checks for `ADR-001-A7 V-6`/`V-7` |
| Do-not list | diff touches only `tools/ci/checkpins/` | n/a |

## Findings, most severe first

### 1. Preference: the `testdata/` exemption also covers `vendor/` (`tools/ci/checkpins/vulncheck.go:150`)

The contract exempts `testdata/` only for `go.mod`. The code `continue`s for every file under a `testdata/` path, so `internal/foo/testdata/vendor/modules.txt` passes. There is a test that pins this (`allowed internal/foo/testdata/vendor/modules.txt`). The behaviour is safe, because the go tool ignores `testdata/` and that code is never built or scanned. It is still a small drift from the wording. I would leave the code as it is and let the contract owner widen the wording.

### 2. Preference: a trailing make comment is read as part of the version (`vulncheck.go:41`)

`GOVULNCHECK_VERSION := v1.9.0 # pinned` reads as `v1.9.0 # pinned` and fails as a mismatch. That fails closed, which is the safe way. The message shows the whole value, so the cause is obvious. No change needed.

### 3. Preference: workflow drift is tested on one scan job only

`only in the workflow` changes the first `govulncheck@` line, which is `vulncheck-root`. The `vulncheck-temporal-server` line goes through the same loop, so another case would add little. Optional.

## Not reviewed here

Isolation is not affected: the change is CI tooling, and no tenant data is involved. Handbook: "check-pins now enforces V-6/V-7" is recorded for TICK-1191.
