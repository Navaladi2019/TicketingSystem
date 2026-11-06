# Review: TICK-576 (P5-30c), PR #478

Reviewer: Quality, Testing & CI Advisor. Change by: Security & Multi-Tenancy Reviewer. Head reviewed: `11c498c`. Base: `TICK-420/audit-sealer`.

Source: the TICK-576 body (ADR-004 R16/R18, ADR-001-A1 D9c with erratum E1, A1 RS5) and the three low findings carried over from the TICK-573 review.

## VERDICT: APPROVE, merged

CI on `11c498c`: every job green (`main-red` skipped). Squash-merged into `TICK-420/audit-sealer` as `790a6cb`.

## What was checked

- **Rule placement.** `checkChildSchedule` runs in the `START_CHILD_WORKFLOW_EXECUTION` case of `checkCommand`, after `checkTarget`, the header, `NavaTenantId` and `checkCommandWorkload`. So the E1 and D9c reasons still win. `TestChildStartScheduleAfterTenantAndWorkloadRules` covers this for 8 earlier rules, each with cron and with retry (16 cases).
- **Semantics match the relay rule.** It reuses the TICK-573 `checkStartSchedule` and maps the reasons. A non-empty cron, blank included, is denied. So is any non-nil `RetryPolicy`, empty included. Cron is reported before retry. One bad child denies the whole response (`commandIdx` 2 is asserted).
- **Product code is not broken.** No non-test code sets `ChildWorkflowOptions.RetryPolicy`. A nil SDK retry policy goes on the wire as a nil `RetryPolicy`. `internal/workerguard` already denies a child cron. So the worker's real children stay allowed, and `TestChildStartWithoutScheduleAllowed` covers both an inherited and an explicit queue.
- **Red check, reproduced locally.** I disabled the new `checkChildSchedule` branch in `commands.go`. Exactly the 6 deny tests failed (`TestChildStartCronScheduleDenied`, `...DetailTruncated`, `...RetryPolicyDenied`, `...CronReportedBeforeRetryPolicy`, `...ScheduleDeniesWholeResponse`, `...CronNotAGateDeny`). With the branch restored, `go test -count=1 ./authz/` in `temporal-server` passes.
- **Golden and docs.** The request-field golden classifies `cron_schedule`, `retry_policy` and `retry_policy.*` on the child command. `docs/temporal-server.md` (E1 table), `docs/pins.md` (`nava/temporal-server` last change, TICK-576) and `docs/architecture/worker-guards.md` are updated per A1 RS5.
- **Carried-over TICK-573 cases.** Cron is reported before retry on both relay start paths. A queue deny beats cron for a STICKY kind and for a non-platform queue. `TestRelayStartCronScheduleDenied` asserts the quoted log `detail`, and a new case checks the truncation. All three are present and pass.

## Findings

1. **Missing test (CI gap, not in this diff): CI does not run golangci-lint on the `temporal-server` module.** `make lint` runs `golangci-lint run ./...` at the repo root (`make/lint.mk:10`). The repo has no `go.work`, and `temporal-server` has its own `go.mod`, so `./...` stops at the module boundary. The `temporal-server` CI job (`.github/workflows/ci.yml:278`) runs only `make temporal-server-test` (`go vet` plus tests) and `make vulncheck-temporal-server`. `fmt-check` does cover its files (`git ls-files '*.go'`). Impact: lint findings in the server module, which holds the D9/E1 authorizer, are never caught. That includes errcheck, staticcheck, and any `//nolint` that slips past the allow-list gate. The PR author could not run it locally either (the run did not finish within 10 minutes). It does not block this PR, because `go vet` and `gofmt` are clean and CI runs the tests. Fix: add a `lint-temporal-server` make target (`cd temporal-server && $(GOLANGCI_LINT) run ./...`, using the root `.golangci.yml` through `--config ../.golangci.yml`) and call it from the `temporal-server` job, with a golangci-lint cache. Then add the step to `tools/ci/cisteps/required.yaml`. I own CI definitions, so this becomes a follow-up task for me.
2. **Preference.** In `TestRelayStartQueueDenyBeforeCron`, the type switch has no `default`. If a third start path is ever added to `startPaths`, its queue is not mutated and the case fails on `start_cron`. That fails loudly, so it is not a gap. A `t.Fatalf` default would make the cause clearer.

## Commands

- Local: `cd temporal-server && go test -count=1 ./authz/`
- CI: the `ci` workflow on PR #478 (`gh pr checks 478`).
