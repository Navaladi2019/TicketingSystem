---
type: rfc
id: RFC-024
title: "The govulncheck workflow runs the scanner directly, with its environment set in the workflow, not through make"
status: REVISED (round 1 response; Technical Architect critique round 1 VERDICT: CHANGES, F1 blocking)
revision: 1
date: 2026-11-27
issue: TICK-1183
author: Solution Architect
target: ADR-001-A7, amending ADR-001-E4 contract rules 2, 7 and 9 (number assumed: highest ADR-001 amendment in docs/adr is A6; highest RFC is RFC-023)
depends-on: TICK-1167 (PR #950, open)
closes-on-delivery: AR-TICK-1167-01
---

## In plain words

The required check `vulncheck` is only as trustworthy as the command it runs. Today that command is `make vulncheck`, and make is a programming language: any of the 20-odd `make/*.mk` files can change what the scan does (its environment, its `PATH`, code run while make reads the files, edits that only happen when `GITHUB_JOB` matches) while the recipe text we pin stays the same. Three review rounds (PR #927, PR #950) each closed one such hole and found the next. The last one, a conditional on a CI-only variable, cannot be closed by a check that runs outside CI (accepted risk `AR-TICK-1167-01`).

**Recommendation: take make out of the gate.** The two scan jobs run `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` directly, with `GOWORK`, `GOFLAGS`, `GOTOOLCHAIN` and the proxy settings fixed in the step's `env:`. The workflow file is already pinned byte for byte by its checked-in copy (contract rule 9), so the whole scan, command and environment, becomes one pinned file. The make targets stay for local use and leave the contract; the recipe and makefile-environment checks of TICK-1149 and TICK-1167 are deleted.

## Context

| What | Where | State |
|---|---|---|
| Scan jobs `vulncheck-root`, `vulncheck-temporal-server` | `.github/workflows/govulncheck.yml` | `run: make tools-govulncheck` then `run: make vulncheck[-temporal-server]` |
| Byte-for-byte copy of the workflow | `tools/ci/cisteps/govulncheck.copy.yml`, `make check-ci-steps` | merged (TICK-1149, PR #927) |
| `make -n` recipe pins, make database check | `tools/ci/cisteps/copy.go` `pinnedRecipes`, `checkMakeDatabase` | merged (TICK-1149) |
| Installer pin, export allow-list, parse-time-code allow-list | `copy.go` `allowedExport`, `environment_test.go` | PR #950, open (TICK-1167) |
| govulncheck version | `Makefile:10` `GOVULNCHECK_VERSION := v1.9.0`, `docs/pins.md:35`, PR #950's `copy.go` const | three copies, no check ties Makefile to pins.md |
| Go version of the scan | `actions/setup-go@v5` with `go-version-file`, then go's default `GOTOOLCHAIN=auto` | v5 installs the `go` line; the `toolchain` line is then fetched by go itself (`docs/pins.md:36`) |

The asset being protected: a green `vulncheck` on a pull request into `main` means "both affected modules, at the Go version we ship, have zero reachable advisories" (ADR-001-A1 Decision 3, ADR-001-E4). The adversary in scope is a pull request that edits files in the repository (an accident, or an edit by someone who does not know about the copy). A deliberate edit of both the workflow and its copy stays out of scope, caught by review (RS3, unchanged).

## Why make cannot be pinned

Every route below changes what `make vulncheck` executes while `make -n vulncheck` still prints `GOWORK=off govulncheck ./...`:

| Route | Example (one file, `make/anything.mk`) | Status today |
|---|---|---|
| recipe environment | `export GOFLAGS=-tags=nope`; `vulncheck: PATH := ./stub:$(PATH)` | rejected by PR #950 |
| code run at parse time | `$(shell ...)`, `$(file >...)`, `!=`, a rebuilt `-include` | rejected by PR #950 (static allow-list) |
| CI-only conditional | `ifeq ($(GITHUB_JOB),vulncheck-root) ... endif` | **not detectable**: `check-ci-steps` runs in `ci.yml`, not in the scan job (AR-TICK-1167-01) |
| next one | `.SHELLFLAGS`, `MAKEFLAGS += -e`, `.ONESHELL`, `override`, `define` + `$(eval)`, `MAKEFILES` env, GNU make version differences | unknown until a reviewer finds it |

The list is open-ended because the language is. A gate whose correctness depends on enumerating a language's features is the wrong shape. The copy check already pins a closed, small surface: one YAML file that GitHub interprets. Moving the scan into that surface closes the class instead of the instances.

## Candidate designs

| | A. Direct `go run` (recommended) | B. Direct `go install` + absolute path | C. Keep make, keep hardening | D. A dedicated pinned makefile | E. `golang/govulncheck-action` |
|---|---|---|---|---|---|
| Scan step | `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` | `go install ...@v1.9.0` then `"$(go env GOPATH)/bin/govulncheck" ./...` | `make vulncheck` | `make -R -f tools/ci/vulncheck.mk vulncheck` | `uses: golang/govulncheck-action@<sha>` |
| What a repo edit can change | `go.mod`/`go.sum` (the scan's subject), `go.work`/`vendor/` (refused, see rule 7) | same as A, plus `PATH` is not used | every make feature, open-ended | that one file (pinned by checksum) plus make's environment | the action's inputs, its own `setup-go` defaults |
| Closes AR-TICK-1167-01 | yes | yes | no | yes, if no `include` and pinned | yes |
| Checks needed in `cisteps` | none beyond the copy | none beyond the copy | today's plus each new hole | copy + second pinned file | copy + action SHA pin |
| Breaks "every gate runs through make" (`docs/pins.md:36`) | yes, one recorded exception | yes | no | no | yes, and adds a third-party action |
| Steps per job | 1 | 2 | 2 | 2 | 1 |

**A over B:** `go run pkg@version` builds the tool outside the current module (it ignores `./go.mod`, its `replace` and `exclude` lines) and runs it without touching `PATH`. One step, nothing installed that a later step could find. B is the fallback if the spike shows `go run` mishandles the exit status or the toolchain.

**Rejected C** because it cannot close the CI-only conditional at all, and its cost grows with each round. **Rejected D** because it keeps make's environment inheritance (`MAKEFLAGS`, `MAKEFILES`, `SHELL` from the job env) and adds a second pinned file for no gain over inline YAML; its only benefit is the convention. **Rejected E** because it is a new third-party action that runs its own `setup-go` (a second, differently configured Go install), and it moves the version pin into action inputs we would pin anyway.

## The decision in detail (what the amendment says)

### D1. The scan jobs

```yaml
  vulncheck-root:
    # needs, if, runs-on, timeout unchanged
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: false
      - name: govulncheck (root module)
        env:
          GOENV: 'off'
          GOWORK: 'off'
          GOFLAGS: ''
          GOTOOLCHAIN: local
          GOPROXY: https://proxy.golang.org
          GOSUMDB: sum.golang.org
          GONOSUMDB: ''
          GOPRIVATE: ''
          GOINSECURE: ''
        run: go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

`vulncheck-temporal-server` is the same with `go-version-file: temporal-server/go.mod` and `working-directory: temporal-server` on the step. No step in either job calls `make`, `sh` or a repository script.

Why each setting:

- **Step-level `env:`**, never job or workflow level: a value an earlier step writes to `$GITHUB_ENV` overrides job- and workflow-level `env:`, and only the step's own `env:` beats it (Technical Architect F6; S4 records it). The earlier steps are only `checkout` and `setup-go`, both pinned by the copy.
- **`GOENV: 'off'`**: cmd/go treats an empty variable as unset and falls back to the go env file (spike S1: `GOFLAGS=''` still yields a `go env -w` value). With `GOENV=off` the empty values below mean empty. The env file sits outside the checkout, so this is hygiene, not a fix for the in-scope adversary (F2).
- **`GOWORK: 'off'`**: rule 7, unchanged in meaning; it moves from the recipe to the step.
- **`GOFLAGS: ''`**: no `-tags`, `-mod`, `-modfile` from anywhere.
- **`GOTOOLCHAIN: local` with `setup-go@v6`**: v6 installs the `toolchain` version of `go.mod` (`docs/pins.md:36`, TICK-1142), and `local` forbids go from switching to any other. The scanned standard library is then exactly the version rule 8 ties to the Dockerfiles. With v5 and `local`, the scan would run on the `go` line (go1.27.0) and report standard-library advisories fixed in go1.27.4: a false red, so the pair (v6, `local`) is one decision. If `govulncheck@v1.9.0` needs a newer Go than `local`, the step fails: fail-safe.
- **Proxy and checksum settings**: the default values, written down so the build of the tool cannot be pointed elsewhere by a runner or org setting. The tool is fetched by module path and version and verified against `sum.golang.org`.
- **`go run` exit status**: govulncheck exits 3 on findings; `go run` then exits **1** (spike S2, `go help run`). The gate needs non-zero. No check may expect exit code 3.

### D2. One version, checked in one place

The version appears in the workflow twice (one per job) and in `Makefile` and `docs/pins.md`. `make check-pins` (`tools/ci/checkpins`, which already reads `docs/pins.md`) gains one check: the `govulncheck` row of `docs/pins.md`, `GOVULNCHECK_VERSION` in `Makefile`, and every `golang.org/x/vuln/cmd/govulncheck@<v>` in `.github/workflows/govulncheck.yml` are the same string, and each scan job has exactly one such line and no other job has one (F7: "at least two" would let a third job carry another version). It reads `Makefile` as text (a regexp on `^GOVULNCHECK_VERSION :?= `), never by running make. The const PR #950 adds to `copy.go` is not needed.

Why `check-pins` and not `check-ci-steps`: the copy check is about the gate's shape; version agreement is a pin concern and `check-pins` owns `docs/pins.md`. A bump edits four lines in three files (workflow, its copy, Makefile, pins.md) and `check-pins` fails on any one missed.

### D3. The make targets

- `tools-govulncheck`, `vulncheck`, `vulncheck-temporal-server` and `vulncheck-ci` **stay, for local use and `ci-lint`**. They are no longer part of the gate and leave the contract. `vulncheck` and `vulncheck-temporal-server` change to the same command line as the workflow (`GOWORK=off GOFLAGS= go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...`) so a local run and CI agree; `tools-govulncheck` stays for `make tools`.
- **Retired:** `pinnedRecipes` and `checkMakeDatabase` in `tools/ci/cisteps/copy.go` with their tests (TICK-1149), and everything PR #950 adds (installer pin, `allowedExport`, parse-time allow-list, `environment_test.go`). The copy comparison itself stays.
- **New `check-ci-steps` assertion (replaces them):** after parsing the YAML, no step's `run:` in `govulncheck.yml` contains `make` as a word (`(^|[\s;&|(])make(\s|$)`); not a substring test. Redundant with the copy today; it states the rule so that a future regenerated copy cannot quietly reintroduce make. Red first: the current workflow fails it.
- **PR #950:** freeze it at its current round. If its reviewer has approved, merge it (it narrows the window until the follow-up lands); otherwise close it unmerged with a pointer here. No further review rounds on makefile holes. The Technical Lead decides which, on delivery cost.

### D4. Contract amendments (`docs/adr/ADR-001/E4/vulncheck-contract.md`)

**Rule 2, third row** becomes: "`govulncheck.yml`, its checked-in copy, the detect or gate script, `tools/ci/cisteps/`, `tools/ci/checkpins/`". The clause "a line naming `govulncheck` or `GOVULNCHECK_VERSION` in `Makefile` or `make/lint.mk`" is removed: the scan no longer reads either file, and a version bump edits the workflow, which already scans both. (`scripts/ci/vulncheck-detect.sh` runs from `main`, so the change takes effect after it merges, as today.)

**Rule 7** becomes:

> 7. **`GOWORK=off`** for both scans, set in the scan step's `env:`. The build fails when a `go.work` or a `vendor/` directory is tracked in either module, or when a `go.mod` is tracked outside the allowed set: `go.mod`, `temporal-server/go.mod`, `docs/go.mod`, `tools/navalint/go.mod`, `tools/workflowcheck/go.mod`, and any path under a `testdata/` directory. Adding a module is a reviewed edit of that set in `tools/ci/checkpins`, which rule 2's third row already scans both modules for.

Reasons: a nested `go.mod` takes its directory out of `./...`, and rule 2 then starts no scan at all (F1). With `vendor/` and `go 1.14`+ the default is `-mod=vendor`, and vendored source is not checked against `go.sum`, so reachability would run on hand-editable code (F1, Q2). The set matches `main` (`git ls-tree origin/main`: five `go.mod` outside `testdata/`, no `vendor/`). The three tooling modules (`docs`, `tools/navalint`, `tools/workflowcheck`) ship nothing and stay unscanned; recorded as residual risk R-A7-1, not claimed away.

**Rule 9** becomes:

> 9. **The copy, and no make.** `make check-ci-steps` compares the whole of `govulncheck.yml` with a checked-in copy, byte for byte with comment lines removed. The two scan jobs call govulncheck directly: `go run golang.org/x/vuln/cmd/govulncheck@<version> ./...`, with `GOENV=off`, `GOWORK=off`, an empty `GOFLAGS`, `GOTOOLCHAIN=local` and the default proxy and checksum settings in the step's `env:`, after `actions/setup-go@v6` reading that module's `go.mod`. No step of the workflow runs `make`, and `check-ci-steps` fails if one does. So the copy pins the command and its environment. What the scan covers is still set by the repository's module layout: the scanned module's `go.mod` and `go.sum` (including `replace`, `exclude` and `ignore`), and any nested `go.mod`, which takes its directory out of `./...`. Rule 7 bounds the layout. `make check-pins` fails unless the version in the workflow equals `GOVULNCHECK_VERSION` in `Makefile` and the `govulncheck` row of `docs/pins.md`. The make targets `vulncheck` and `vulncheck-temporal-server` remain for local use and are not part of the gate. **This stops an accidental edit, or one that does not know about the copy. It does not stop a deliberate edit of both (RS3).**

**Tests table:** V-3 loses "`|| true` in either make recipe" and gains "a step `run: make vulncheck`" and "a step `env:` without `GOTOOLCHAIN: local`" (both caught by the copy; the first also by the no-make assertion). New V-6 for D2: each of the three version locations changed alone is red in `check-pins`, and so is a third job with its own `govulncheck@` line. New V-7 for rule 7: a tracked `internal/foo/go.mod`, and a tracked `vendor/modules.txt` in either module, are each red in `check-pins`.

**`docs/pins.md:36`** ("every gate runs through `make`") gains: "except the `govulncheck` workflow's scans, which run govulncheck directly (ADR-001-A7)".

### D5. What closes `AR-TICK-1167-01`

The CI-only conditional needs make to run in the scan job. After D1 it does not, so there is nothing for `ifeq ($(GITHUB_JOB),...)` to act on. The last task of the plan records the closure with the evidence: the merged workflow, and a red-first run of the no-make assertion.

## Invariants touched

The project description of TICK-1183 has no project set (`project_id: null`), so the eight invariants were not loaded from it. **Invariant checks are incomplete.** From the ADR-001 record: this change touches CI only, no tenant data, identity, Temporal workflow code or runtime service. The relevant standing rules are ADR-001-A1 Decision 3 (zero reachable findings, no allow-list), unchanged, and ADR-001-E4 rules 2, 7 and 9, amended above. Nothing weakens: the gate gains coverage (the environment) and loses only checks of something it no longer runs.

## Tradeoffs

| Cost | Size | Accepted because |
|---|---|---|
| Local `make vulncheck` and CI can drift | low; D3 makes them the same command and D2 the same version | the gate is CI; local is a convenience |
| An exception to "every gate runs through make" | one workflow | the convention exists for reproducibility; the step is one `go run` line anyone can paste |
| `go run` rebuilds govulncheck each run | ~20 to 40 s per job, same as `go install` today (no cache in this workflow) | I don't have a measurement; the spike records it |
| A version bump touches four lines | three files plus the copy | `check-pins` fails on any one missed |
| Deliberate edit of workflow and copy | unchanged (RS3) | review; out of scope as before |
| R-A7-1: the `docs`, `tools/navalint`, `tools/workflowcheck` modules are never scanned | low: tooling, in no image | they ship nothing; rule 7's allowed set makes a new module a reviewed edit |
| `ignore`, `replace`, `exclude` in a scanned `go.mod` can narrow the scan | an edit of the scan's subject, which detect scans | visible in review of that `go.mod`; out of scope like RS3 |

## Assumptions (U1, U2 settled by the Technical Architect's spikes S2, S3; U3 to U5 by S4, the first delivery task, which does not block the ADR)

| # | Assumption | How to check |
|---|---|---|
| U1 | `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` exits non-zero when govulncheck exits 3 | run on a throwaway module requiring a known-vulnerable version (V-4b's library) |
| U2 | `go run pkg@version` ignores the current directory's `go.mod`, including a `replace golang.org/x/vuln => ./stub` | add that replace in the throwaway module; the real tool must run |
| U3 | step `env:` overrides a value written to `$GITHUB_ENV` by an earlier step | one workflow run on a scratch branch: step 1 writes `GOFLAGS=-tags=x` to `$GITHUB_ENV`, step 2 with `GOFLAGS: ''` prints `go env GOFLAGS` |
| U4 | `setup-go@v6` with `go-version-file` installs the `toolchain` version, so `GOTOOLCHAIN=local` gives `go version` = go1.27.4 for both modules | same scratch run, print `go version` |
| U5 | with `GOTOOLCHAIN=local`, `govulncheck` v1.9.0 builds on go1.27.4 | same run |

U1: confirmed, `go run` exits 1 (S2). U2: confirmed (S3, on govulncheck v1.1.3 / go1.22.2; the behaviour belongs to cmd/go). Design B is not needed. S4, one scratch workflow at the pinned versions, records: step-2 `go env GOFLAGS` empty after step 1 wrote `GOFLAGS=-tags=x` to `$GITHUB_ENV` (U3); `go version` = go1.27.4 in both modules (U4); `govulncheck@v1.9.0 -version` builds (U5); the build time per job.

## Questions for the Technical Architect

1. Is there any repository-controlled input to the D1 step that I have missed, other than the module's `go.mod`, `go.sum`, `go.work` and `vendor/`? (Candidates I ruled out: `.github/` composite actions, none used; repo or org variables, which only enter through `${{ vars }}`; `go.env` and `GOENV`, outside the checkout.)
2. Is `vendor/` refusal (rule 7) worth a check, or noise?
3. Do you prefer the no-make assertion in `check-ci-steps`, or leave the copy as the only check?

## Response — round 1

| # | Item | Resolution |
|---|---|---|
| F1 | Rule 9 overclaims: a nested `go.mod` or an `ignore` directive shrinks `./...` | **Accepted (blocking).** The rule 9 sentence is replaced with the Technical Architect's text, word for word. Rule 7 now lists the allowed `go.mod` set, checked against `main` (`go.mod`, `temporal-server/go.mod`, `docs/go.mod`, `tools/navalint/go.mod`, `tools/workflowcheck/go.mod`, plus `testdata/`), and refuses `vendor/`. The unscanned tooling modules are recorded as R-A7-1. New test V-7. |
| F2 | An empty env value does not override the go env file | **Accepted.** `GOENV: 'off'` is added to both scan steps and to rule 9, and the D1 rationale is rewritten. |
| F3, F4 | `go run` exit status; a local `replace` is ignored | **Confirmed by spike.** D1 now says exit 1, and no check expects 3. Design B is dropped. |
| F5 | setup-go v6 + `GOTOOLCHAIN=local` | Confirmed. The live `go version` print goes into S4. |
| F6 | `$GITHUB_ENV` overrides job- and workflow-level `env:` | **Accepted.** The precedence sentence is corrected; settings stay in the step's `env:`. |
| F7 | "at least two" version lines | **Accepted.** Exactly one per scan job and none elsewhere; V-6 extended. |
| F8 | Rule 2 trigger change | Confirmed; unchanged. |
| Q3 | No-make assertion | Kept, as a word match on the parsed `run:` values. |

Nothing else changed. S4 is the first delivery task and does not gate the ADR.
