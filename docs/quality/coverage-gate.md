# Coverage gate (TICK-518)

CI fails when the Go test coverage of the platform module is below **95%**
(vknavaladi, 2026-11-03). This page covers how the total is measured, what is
left out, how the gate is rolled out and how to run it.

## What is measured

| Profile | Written by | How |
| --- | --- | --- |
| `coverage.out` | `make test` / `task test` (unit, `-race`) | `-covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...` |
| `coverage-integration.out` | `make test-integration` / `task test:integration` (real PostgreSQL) | every `go test` group of `scripts/ci/test-integration.sh` with the same flags when `NAVA_IT_COVERAGE` is set, merged at the end |

`make check-coverage` (`tools/ci/coverage check`) merges both, drops the files
in `tools/ci/coverage-exclude.txt`, and compares covered statements over all
statements with `COVERAGE_MIN` (default 95). It writes `coverage-total.out`
(the merged profile) and `coverage.html` even when it fails, and below the
minimum it lists the 20 worst-covered packages.

- `-coverpkg=./...`: a block counts as covered whichever package's tests ran
  it, so a handler test that runs the store counts for the store.
- Merging: the same block from several test binaries counts once, covered if
  any run hit it.
- The figure is the one `go test -cover` prints ("% of statements"). It can
  be about 0.1 points above the `total:` line of `go tool cover -func`, which
  skips func literals declared at package level. The gate uses the statement
  count so that every statement in the profile counts.
- Not counted: the separate modules `tools/navalint`, `tools/workflowcheck`
  and `temporal-server`, the isolation suite (`make isolation`), and the stack
  tests (`make stack-tests`), which test running binaries.

## Exclusions

`tools/ci/coverage-exclude.txt`, one pattern and its reason per line. The tool
fails on a line with no reason and on a pattern that matches no file, so the
list cannot go stale. Allowed: generated code, `cmd/*/main.go` wiring and
test-only helper packages. Anything else needs the Technical Lead's approval
on the issue. Excluding real logic to reach the minimum is not acceptable.

A `dir/...` line excludes a whole package tree. `make check-coverage` passes
`-mains ./cmd/...`, and the check fails (even report-only) when any such
package is in the import graph of a `cmd/*` binary: an excluded helper that
production code starts to import is production code and must count
(Technical Lead's condition on TICK-518). Build-tagged test binaries (the
`interptest` worker) are not production binaries and are not checked.

The total is covered statements over all statements; the Technical Lead
accepted this over the `go tool cover -func` line as the definition (TICK-518,
2026-11-03).

## Rollout

The baseline was below 95%, so the `coverage` CI job runs
`make check-coverage COVERAGE_REPORT_ONLY=1` for now: it prints the total and
every package's gap, and fails only on bad input (a missing profile, a stale
exclusion). The follow-up PR changes the step to `make check-coverage`, and
the line in `tools/ci/cisteps/required.yaml` with it, once the total reaches
95%. That PR closes TICK-518.

The `coverage` job runs on every change that runs `unit`: `make ci-stages`
adds `integration` to any change that needs `unit`, and the job runs even
when `unit` or `integration` failed or was skipped, so a missing profile
fails it rather than skipping it. Only a change that runs no Go tests at all
(documentation, lint configuration) skips it.

## Risks and the test that covers each

| Risk | Covered by |
| --- | --- |
| The gate passes below the minimum (rounding, a wrong merge) | `tools/ci/coverage` unit tests: 95.00% passes, 94.99% fails, `-min` moves the edge, repeated blocks are not double-counted |
| The integration run's coverage is lost | `TestCheckMergesProfiles`: a block only the integration profile covers counts; the `coverage-integration` artifact is a required step |
| An exclusion hides real code silently | `TestCheckBadInput`: lines without a reason and stale patterns fail; `TestRepoExclusionsFile` parses the real file |
| An excluded helper becomes production code | `TestCheckExcludedPackageImportedByBinary` (stubbed graph) and `TestRepoExcludedPackagesNotInBinaries` (the real `./cmd/...` graph) |
| The CI step is removed or allowed to fail | `make check-ci-steps` (`required.yaml`: the job's run line and its three artifacts) |
| `task test` and `make test` drift | `make check-taskfile` |

## Commands

```sh
task test                          # writes coverage.out
task test:integration              # needs Docker; writes coverage-integration.out
task check-coverage                # both profiles, fails below 95%
task check-coverage COVERAGE_PROFILES=coverage.out   # unit half only, no Docker
make check-coverage COVERAGE_MIN=90 COVERAGE_REPORT_ONLY=1
```

In CI, read the `coverage` job of the `ci` workflow and download its
`coverage` artifact (`coverage.html` and the profiles).
