---
type: adr-amendment
id: ADR-001-A7
title: "Amendment A7 to ADR-001-E4 contract rules 2, 7 and 9: the govulncheck workflow runs the scanner directly, with its environment set in the workflow, not through make"
status: ACCEPTED
accepted: 2026-11-27 (Solution Architect verdict, TICK-1183; Technical Architect re-check APPROVE, comment 01a21bb4-0e33; no CTO tie-break needed, nothing contested)
date: 2026-11-27
revision: 1
issue: TICK-1183
rfc: docs/adr/ADR-001/rfc/RFC-024-vulncheck-direct-scan.md (rev 1)
supersedes: "in docs/adr/ADR-001/E4/vulncheck-contract.md: rule 9's recipe check (\"It also checks that `make -n vulncheck` prints exactly ...\" to \"... both expected lines.\"), rule 7's \"the build fails when a `go.work` is tracked\", and rule 2's clause \"a line naming `govulncheck` or `GOVULNCHECK_VERSION` in `Makefile` or `make/lint.mk`\""
amends: "[[ADR-001-E4-a1-d3-vulnerability-scan-gate]] contract rules 2, 7, 9 and its tests table (V-3; new V-6, V-7)"
closes: "AR-TICK-1167-01 (on delivery of the plan's last task)"
deciders: "Solution Architect (author; verdict ACCEPTED); Technical Architect (round 1: CHANGES, F1 blocking; re-check APPROVE; spikes S1 to S3)"
---

# ADR-001 amendment A7: the vulnerability scan runs without make

> **Status: ACCEPTED, rev 1, 2026-11-27,** by the Solution Architect (TICK-1183), under the merge policy set by vknavaladi. Technical Architect: round 1 CHANGES (F1, the wording of rule 9), re-check APPROVE. No CTO tie-break: nothing contested. It changes no code; the work comes from `Plan ADR-001-A7`.
>
> **Invariants source.** TICK-1183 has no project set, so the project's eight invariants were not loaded; the invariant check is incomplete. This is CI only: no tenant data, identity, workflow or runtime code. ADR-001-A1 Decision 3 (zero reachable findings, no allow-list) is unchanged.
>
> **Numbering.** A6 is the highest ADR-001 amendment; RFC-023 the highest RFC. So A7 and RFC-024.

## In plain words

- **Goal:** the required check `vulncheck` must mean "both affected modules, at the Go version we ship, have no reachable known flaw". Nothing outside the workflow file should be able to change that silently.
- **Today:** the scan runs `make vulncheck`. Make reads about twenty files, and any of them can change what the scan does while the recipe we check still looks the same. Three review rounds each closed one hole and found the next. The last hole, an edit that only applies when a CI variable matches, cannot be seen by a check that runs outside the scan job (`AR-TICK-1167-01`).
- **What's new:** the scan jobs run one command directly: `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`, with every Go setting that matters written in the step. The workflow file is already compared byte for byte with a checked-in copy, so the command and its settings are now both pinned.
- **Who is affected:** nobody outside CI. Engineers keep `make vulncheck` for local runs; it is no longer part of the gate.
- **Risks:** someone who edits both the workflow and its copy can still pass a scan that checks nothing (unchanged, RS3; review stops it). Three tooling modules are never scanned (R-A7-1). Edits of a scanned `go.mod` (`replace`, `exclude`, `ignore`) can narrow the scan; they trigger a scan and are reviewed.
- **How we'll know it worked:** a scratch workflow run (S4) shows the pinned Go and an empty `GOFLAGS`; the new checks fail red first; `govulncheck.yml` contains no `make`.

## Decisions

**A7.1 The scan step.** Both scan jobs: `actions/checkout@v4`, `actions/setup-go@v6` with `go-version-file` of that module and `cache: false`, then one step:

```yaml
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

`vulncheck-temporal-server` adds `working-directory: temporal-server`. No step of `govulncheck.yml` runs `make`. Settings live in the step's `env:` because a value an earlier step writes to `$GITHUB_ENV` overrides job- and workflow-level `env:`. `GOENV: 'off'` because cmd/go treats an empty value as unset and falls back to the go env file (spike S1). setup-go v6 installs the `toolchain` line (TICK-1142) and `GOTOOLCHAIN: local` forbids switching, so the scanned standard library is the one rule 8 ties to the Dockerfiles; v5 with `local` would scan go1.27.0 and fail falsely. `go run pkg@version` ignores the local `go.mod` and its `replace` lines (S3) and exits 1 when govulncheck exits 3 (S2); no check may expect 3.

**A7.2 One version.** `make check-pins` fails unless the `govulncheck` row of `docs/pins.md`, `GOVULNCHECK_VERSION` in `Makefile` (read as text, never by running make), and the `govulncheck@<v>` in `govulncheck.yml` are the same string, with exactly one such line in each scan job and none in any other job.

**A7.3 Make.** `tools-govulncheck`, `vulncheck`, `vulncheck-temporal-server` and `vulncheck-ci` stay for local use and `ci-lint`, and leave the contract. `vulncheck` and `vulncheck-temporal-server` run the same command line as the workflow. Retired: `pinnedRecipes` and `checkMakeDatabase` in `tools/ci/cisteps/copy.go` with their tests (TICK-1149), and the additions of PR #950 (TICK-1167). New `check-ci-steps` assertion: after parsing the YAML, no step's `run:` contains `make` as a word (`(^|[\s;&|(])make(\s|$)`). `docs/pins.md` "every gate runs through `make`" gains the exception for this workflow.

**A7.4 Contract text** (`docs/adr/ADR-001/E4/vulncheck-contract.md`, edited in this PR):

- **Rule 2, third row:** `govulncheck.yml`, its checked-in copy, the detect or gate script, `tools/ci/cisteps/`, `tools/ci/checkpins/`. The `Makefile` / `make/lint.mk` clause is removed.
- **Rule 7:** `GOWORK=off` in the scan step's `env:`. The build fails when a `go.work` or a `vendor/` directory is tracked in either module, or a `go.mod` is tracked outside `go.mod`, `temporal-server/go.mod`, `docs/go.mod`, `tools/navalint/go.mod`, `tools/workflowcheck/go.mod` and `testdata/` paths. A nested `go.mod` would take its directory out of `./...`; vendored source is not checked against `go.sum`.
- **Rule 9:** the copy, and no make (full text in the contract).
- **Tests:** V-3 loses the make-recipe case and gains `run: make vulncheck` and a step without `GOTOOLCHAIN: local`. V-6: each version location changed alone, and a third job with its own `govulncheck@` line, are red in `check-pins`. V-7: a tracked `internal/foo/go.mod`, and a tracked `vendor/modules.txt`, are red.

**A7.5 Evidence still to record (S4, first delivery task, does not gate this ADR).** One scratch workflow run at the pinned versions: `go env GOFLAGS` empty after an earlier step wrote `GOFLAGS=-tags=x` to `$GITHUB_ENV`; `go version` = go1.27.4 in both modules; `govulncheck@v1.9.0 -version` builds; `go run` exit status non-zero on a finding at v1.9.0; build time per job.

**A7.6 Closure.** The plan's last task closes `AR-TICK-1167-01`: with no make in the scan job, a CI-only makefile conditional has nothing to act on.

## Rejected options

| Option | Why not |
|---|---|
| Keep make, keep hardening (status quo + PR #950) | Open-ended; the CI-only conditional cannot be detected from `ci.yml` at all |
| A dedicated pinned makefile (`make -R -f tools/ci/vulncheck.mk`) | Keeps make's environment inheritance (`MAKEFLAGS`, `MAKEFILES`, `SHELL`) and adds a second pinned file; only buys the convention |
| `golang/govulncheck-action` | New third-party action with its own `setup-go`; the pin moves into action inputs |
| `go install` then call by absolute path | Sound, but two steps; kept only as fallback, not needed after S2 and S3 |

## Residual risks accepted

| Id | Risk | Why accepted |
|---|---|---|
| RS3 (AR-TICK-1140-03) | A deliberate edit of both workflow and copy | Unchanged from E4; review |
| R-A7-1 (AR-TICK-1183-01) | `docs`, `tools/navalint`, `tools/workflowcheck` are never scanned | Tooling, in no image; a new module is a reviewed edit of rule 7's set |
| R-A7-2 (AR-TICK-1183-02) | `replace`, `exclude`, `ignore` in a scanned `go.mod` narrow the scan | That edit triggers the scan and is reviewed with the module |
| R-A7-3 | Local `make vulncheck` and CI drift | Same command line and one checked version; CI is the gate |

## Dissent

None recorded. The Technical Architect's F2, F6 and F7 (non-blocking) were adopted as proposed.
