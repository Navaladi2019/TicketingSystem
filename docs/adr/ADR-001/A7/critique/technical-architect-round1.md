<!-- TICK-1183 comment 01a21b51-cf7d, Technical Architect, 2026-11-27. Includes spikes S1 to S3. -->

## Critique — round 1 (RFC-024, Technical Architect)

**VERDICT: CHANGES.** One blocking item, and it is only wording: the proposed rule 9 claims more than the mechanism does. The design itself holds. Design A (direct `go run`) is sound and closes `AR-TICK-1167-01`. Design B is not needed: U1 and U2 are settled below.

Checked against `main` at `af5b3bde` (`.github/workflows/govulncheck.yml`, `docs/adr/ADR-001/E4/vulncheck-contract.md`, `docs/pins.md`, `make/lint.mk`, `tools/ci/checkpins/samego.go`).

### Findings

| # | Mechanism | Verdict | Evidence | Known / Inferred |
|---|---|---|---|---|
| F1 | Rule 9 text: "no file outside the workflow can change what the scan runs, except the scanned module's own `go.mod` and `go.sum`" | **blocking** (text) | A new `internal/foo/go.mod` takes `internal/foo/...` out of the root module's `./...` (patterns never cross into a nested module), and rule 2's "anything else → none" means detect does not even start a scan. Three nested modules are already tracked and never scanned: `docs/go.mod`, `tools/navalint/go.mod`, `tools/workflowcheck/go.mod`. The `go.mod` itself can also shrink the scan: the `ignore` directive (Go 1.25+) drops directories from `./...`. | Known (`go help packages`; Go 1.25 release notes, `ignore` directive; `git ls-files`) |
| F2 | `GOFLAGS: ''`, `GONOSUMDB: ''`, `GOPRIVATE: ''`, `GOINSECURE: ''` "so the build cannot be pointed elsewhere by a runner or org setting" | **non-blocking** | An empty variable does **not** override the go env file. cmd/go's `cfg.Getenv` treats `""` as unset and falls back to `$GOENV`. Spike S1 below: `GOFLAGS=''` still gives `-tags=spike`, and `GOPRIVATE=''` still gives `evil.example`. With `GOENV=off` both are empty. A repo edit cannot write that file (it sits outside the checkout and only pinned actions run first), so this is not a hole for the adversary in scope. It is a false rationale with a one-line fix. | Known (S1, go1.22.2; this code path has not changed since go env -w landed in 1.13) |
| F3 | `go run …@v1.9.0` fails the job when govulncheck reports a finding (U1) | **confirmed sound** | Spike S2: a program that runs `os.Exit(3)` under `go run` prints `exit status 3`, and `go run` exits **1**. The gate needs non-zero; it gets 1, not 3. Do not write any check that expects exit code 3. | Known (S2; `go help run`: "the exit status of Run is not the exit status of the compiled binary") |
| F4 | `pkg@version` ignores the local `go.mod`, including `replace golang.org/x/vuln => ./stub` (U2) | **confirmed sound** | Spike S3: a module with that replace ran the real `govulncheck@v1.1.3` and exited 0. Note: this was v1.1.3 on go1.22.2, not v1.9.0 on go1.27.4. The behavior is in cmd/go, not in govulncheck. | Known (S3; `go help run`: "ignoring the go.mod file in the current directory or any parent") |
| F5 | setup-go v6 plus `GOTOOLCHAIN=local` scans the shipped Go (U4) | **confirmed sound** | `docs/pins.md:36` already records that v6 installs the `toolchain` line (TICK-1142). The risky path is a missing toolchain line plus `go 1.27` (no patch), where setup-go would float to the newest 1.27.x and could give a false green. That path is closed: `samego.go:78` fails `check-pins` on "no toolchain line". If the tool needs a newer Go, or `go.mod` raises the `go` line, the step fails, which is the safe direction. | Known (repo); live `go version` print still wanted (S4) |
| F6 | Step `env:` beats `$GITHUB_ENV` (U3) | **non-blocking**, keep env at step level | It does not matter today: only `checkout` and `setup-go` run before the scan step, and the copy pins both. My understanding is that `$GITHUB_ENV` **overrides** job-level and workflow-level `env:`, so the RFC's precedence order is wrong in that part. Step level is the one level that wins, and the proposed rule 9 already says "in the step's `env:`". Keep it that way. | Inferred; S4 settles it |
| F7 | D2, a `check-pins` text match on Makefile, workflow and pins.md | **confirmed sound** | Reading the Makefile as text is correct. A later `make/*.mk` override of `GOVULNCHECK_VERSION` now affects local runs only. Tighten "at least two" to "exactly one per scan job": an extra `@v…` line in a third job would otherwise pass. | Inferred |
| F8 | Rule 2: drop the `Makefile` / `make/lint.mk` trigger and add `tools/ci/checkpins/` | **confirmed sound** | Once the scan no longer reads either file, a Makefile-only version bump fails `check-pins` in `ci.yml` before any scan is needed. | Inferred from D1/D2 |

### Proposed text for the revision

**F1: rule 9.** Replace the sentence from "So the copy pins…" with:

> So the copy pins the command and its environment. What the scan covers is still set by the repository's module layout: the scanned module's `go.mod` and `go.sum` (including `replace`, `exclude` and `ignore`), and any nested `go.mod`, which takes its directory out of `./...`. Rule 7 bounds the layout.

**F1 + Q2: rule 7.** Fold both checks into the existing `go.work` check:

> **`GOWORK=off`** for both scans, set in the scan step's `env:`. The build fails when a `go.work` or a `vendor/` directory is tracked in either module, or when a `go.mod` is tracked outside the allowed set: `go.mod`, `temporal-server/go.mod`, `docs/go.mod`, `tools/navalint/go.mod`, `tools/workflowcheck/go.mod`, and `testdata/` fixtures.

Adding a module then becomes a reviewed edit of that set, not a silent drop out of the scan. If you rule this out of scope, record it as accepted risk rather than leaving the claim in rule 9. That choice is yours (scope).

**F2: D1 `env:`.** Add one line, `GOENV: 'off'`. The explicit empty values then mean what the RFC says they mean. Without it, delete the "runner or org setting" sentence.

### Your questions
1. **Other repository inputs to the step:** nested `go.mod` files and the `go.mod` `ignore` / `replace` directives (F1). Your rule-outs hold: no composite actions are used, `vars` only enter by expression, and `go.env` / `GOENV` sit outside the checkout (but see F2 on empty values).
2. **`vendor/` refusal:** worth it. With `vendor/` present and `go ≥ 1.14` the default is `-mod=vendor`, and vendored source is **not** checked against `go.sum` (`go mod verify` checks only the module cache). Advisories still match on `modules.txt` versions, but reachability runs on source that can be edited by hand. It costs one line in an existing check.
3. **No-make assertion:** keep it, as a word match (`(^|[\s;&|(])make(\s|$)`) on the `run:` of every step after parsing the YAML, not a plain substring test. It is the red-first test for delivery and states the rule. Redundancy with the copy is fine.

### Remaining spike (S4, one scratch workflow run, no change to the design)
Run on a scratch branch at the pinned versions: `setup-go@v6`, go1.27.4, govulncheck v1.9.0. Done before or as the first delivery task, it records the evidence; the RFC does not wait on it.
- step 1 writes `GOFLAGS=-tags=x` to `$GITHUB_ENV`, and step 2 (step `env:` `GOFLAGS: ''`, `GOENV: 'off'`) prints `go env GOFLAGS`. **Pass:** empty.
- the same step prints `go version`. **Pass:** `go1.27.4` in both modules (U4).
- `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -version`. **Pass:** builds and prints `Scanner: govulncheck@v1.9.0` (U5).
- Record the build time per job (the RFC's 20 to 40 s figure is unmeasured).

<details><summary>Spikes S1 to S3 (run 2026-11-27, go1.22.2 windows/amd64, govulncheck v1.1.3)</summary>

```sh
# S1: empty env var vs go env file
export GOENV="$PWD/envfile/env"; go env -w GOFLAGS=-tags=spike GOPRIVATE=evil.example
GOFLAGS= go env GOFLAGS            # -> -tags=spike
GOPRIVATE= go env GOPRIVATE        # -> evil.example
GOENV=off GOFLAGS= go env GOFLAGS  # -> (empty)
# S2: go run exit status
printf 'package main\nimport "os"\nfunc main(){os.Exit(3)}\n' > main.go   # in module x
go run . ; echo $?                 # -> "exit status 3", then 1
# S3: pkg@version ignores a local replace
# go.mod: module y / require golang.org/x/vuln v1.1.3 / replace golang.org/x/vuln => ./stub
GOTOOLCHAIN=local GOWORK=off GOFLAGS= go run golang.org/x/vuln/cmd/govulncheck@v1.1.3 -version
# -> Scanner: govulncheck@v1.1.3 ... No vulnerabilities found.  exit 0
```
</details>

For the re-check I will look only at F1. F2 and F7 are non-blocking and can go into the ADR either way.
