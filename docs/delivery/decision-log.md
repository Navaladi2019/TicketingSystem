# Decision log

Delivery decisions that change a task's contract, newest first.

## 2026-10-23: PLAN-001-A2 splits ADR-001-A2 delivery into three tasks; R-CORE gates the first nava-core Temporal call
- **Decided by:** the Technical Lead on TICK-130 (`docs/delivery/plan-adr-001-a2.md`). Approved by nava on TICK-130 (2026-10-23). ADR-001-A2 was approved by nava on TICK-129 (2026-10-23).
- **Decision:** the ADR's single `core-dev` acceptance becomes three PRs under S7 (TICK-49), each waiting for PR #38 to merge:
  - T27i (TICK-131): D9b+, D9d (both clauses), D9e, D9f and D9g. Un-skips F2-1 to F2-6.
  - T27j (TICK-132), in parallel: the D6-K row `frontend.enableTokenNamespaceEnforcement` must be `true`. Un-skips F2-3b.
  - T27k (TICK-133), after T27i: the CI-C request-field drift golden.
- **R-CORE gate:** R-CORE-1 and R-CORE-2 aren't a task, because no planned task makes `nava-core` call Temporal yet. The first task that does must carry both as acceptance tests, plus the single workflow-id constructor, with Security reviewing (RS6). *Trigger:* that task is planned (likely Epic 09, TICK-12). It may use T23's (TICK-58) sealed cursor for the opaque-token clause. Owner: Technical Lead.
- **No grooming meeting:** same reason as D-A1-6.
- **Dissent:** none.

## 2026-10-20: PLAN-001-A1 splits ADR-001-A1 delivery into D9 first, then the 1.29.7 upgrade
- **Decided by:** the Technical Lead on TICK-119 (`docs/delivery/plan-adr-001-a1.md`). Approved by nava on TICK-119 (2026-10-20).
- **Decision:** RFC-002 §9's single T27d PR becomes two PRs, merged in order:
  - T27e (TICK-120): the D9 workload allow-list on 1.28.4. It closes S1, which is live on every version.
  - T27f (TICK-121): server 1.29.7 / api 1.53.0, schema 1.18, the D6-K guard and the vulncheck gate.
  - CI-A to CI-I must all be green on T27f's PR. T27d (TICK-111) closes when T27f merges.
- **E4:** no 1.28.4 data needs keeping (M1 is dev-only and single-node). T27f still checks the upgrade in place, schema-before-binary, with an in-flight execution. *Trigger:* the first non-dev deployment needs a runbook, together with F1.
- **RS5:** accepted. Each new workflow type or task queue is a policy PR plus an image rebuild. *Trigger:* more than about one such change a week, or PLAN-002 Epic 09 needing a type per definition.
- **Deferred:** F1 (mixed-version node smoke). *Trigger:* before any multi-node deployment. Owner: Technical Architect. F2 is T27g (TICK-122, `security`), released after T27f.
- **No grooming meeting.** The task text is lifted directly from the ADR, which was reviewed twice. The PR reviews carry that weight instead.
- **Dissent:** none recorded. RFC-002 §9 proposed one PR; the split keeps every test it named.

## 2026-10-20: CI wall clock under 5 minutes; stack tests split over three jobs
- **Decided by:** the Technical Lead on TICK-59, on the human's escalation (CI took about 22 minutes).
- **Decision:** `ci.yml` runs seven parallel jobs: `lint`, `unit`, `temporal-server`, `integration`, `stack`, `stack-db` and `stack-internal`. This replaces `lint-unit` and the single `stack` job. Every gate and every stack test still runs, and none is skipped or weakened. `make ci` and `make ci-stack` are unchanged locally.
- **Caching:** each job keeps a Go build cache keyed per job. CI tools are cached by Makefile hash, and image layers live in the Actions cache (`make image-ci`, `IMAGE_CACHE=gha`).
- **temporal-server image:** it no longer runs `go test -short` during the build. The `temporal-server` job already runs the full suite, which is a superset. Its Temporal dependencies now compile in a layer keyed on `go.mod`/`go.sum`.
- **Accepted cost:** a run that changes `go.sum` or a base image is cold and takes about 8 to 10 minutes. Three stacks per run use more Actions minutes.
- **Dissent:** none.

## 2026-10-19: T28 operator UI reached only through a tunnel; UI login deferred
- **Decided by:** the Technical Lead on TICK-59, on the Security Reviewer's Finding 1 (PR #21).
- **Decision:** `temporal-ui` publishes no port. Operators reach it with `make operator-tunnel`, which lives only while it runs. `stack-test.sh temporal_operator_reach` checks the default state, with no tunnel.
- **Accepted risk (Low):** while the tunnel is open, a compromised nava-core can reach the UI through `host.docker.internal` and act as `nava-operator`.
- **Deferred:** Temporal UI login through Keycloak OIDC (TODO-006). Trigger: the UI must be reachable without a tunnel, the stack runs in a shared or non-local environment, or Milestone 1 closes, whichever is first.
- **Also decided:** `temporal-server` pins `go.temporal.io/server/cmd/tools/sql` with a `tool` directive. Its indirect dependencies (`urfave/cli`, `go-md2man`, `blackfriday`) come only from the pinned server module and are accepted under the no-new-dependencies rule.
- **Dissent:** none.

## 2026-10-19: T05 keeps docker-CLI integration containers; CI gates run as parallel required checks
- **Decided by:** the Technical Lead on TICK-56 (testcontainers, 2026-10-18; gate order, 2026-10-19), after the PR #19 review.
- **Decision 1:** the integration tests keep starting containers through the docker CLI. `testcontainers-go` and `moby/go-archive` are approved, not adopted.
  - **Reason:** T03 and T04 already use this pattern, and converting would add a dependency that fixes nothing.
  - **Intent kept (TICK-94 item 16):** real containers, image tags from `docs/pins.md` (`check-pins` fails on an inline tag), and per-test cleanup. Go-started containers are removed in `t.Cleanup`. The shared container that `scripts/ci/test-integration.sh` starts is removed by its trap on exit, Ctrl-C or TERM.
  - **Trigger:** if testcontainers is adopted later, the `moby/go-archive` ≥ v0.3.0 pin comes with it (GO-2026-6253).
- **Decision 2:** "Gates, in order" means each gate is a separate required check that blocks merge. The four CI jobs (`lint-unit`, `temporal-server`, `integration`, `stack`) run in parallel, with no `needs: lint-unit`. That keeps the pipeline under the 10-minute target (6m15s). `make ci` still runs the stages in order locally.
- **Dissent:** none.

## 2026-10-17: TICK-94 rulings folded into PLAN-001; T01g owns span redaction
- **Decided by:** the Technical Architect on TICK-94 (verdicts for items 1–16, 2026-10-16). Folded in by the Technical Lead on TICK-104.
- **Decision:** each affected task (T03, T04, T05, T07, T15, T18, T20, T24, T30a, T30b, T31, T33) has an "Audit TICK-93 / TICK-94" section that quotes its conditions. T05 commits the pin rows and a forbidden-module list to `docs/pins.md`.
- **Rejected, now forbidden:** go-oidc → `jwx/v3`; gocloak → a hand-written client in `internal/identity/keycloak`; ogen → `oapi-codegen` + `kin-openapi`; golang-migrate and tern → goose; gofrs/uuid → google/uuid; golangci-lint module plugins → `go vet -vettool` (`tools/navalint`).
- **New task:** T01g (TICK-105, `core-dev`) turns the span export filter into an allow-list (item 3). T07 and T15 depend on it.
- **Technical Lead calls:** T04 renames T01d's `NAVA_METRICS_ADDR` to `NAVA_ADMIN_ADDR` and builds the shared outbound HTTP client. T15's token skew tightens from ±60 s to ±30 s (item 11). The CI-11 depguard exclusion is narrowed to `internal/identity/keycloak`, to match D-P6.
- **Dissent:** none.

## 2026-10-16: T01 telemetry on OpenTelemetry; the edge is a trace trust boundary
- **Decided by:** nava (human), on TICK-89. Shaped by the Technical Lead's review.
- **Decision:** nava-core uses the OTel API for traces and metrics. Prometheus stays only as the `/metrics` scrape format, through the OTel Prometheus exporter. The public listener uses `otelhttp.WithPublicEndpoint()`, so an inbound `traceparent` is linked and never continued. The edge propagator is TraceContext only, which drops inbound baggage. Sampling is `parentbased_traceidratio`. The request id stays server-generated.
- **Supersedes:** T01's "inbound `traceparent` is honoured" and the metric names `http_requests_total` and `http_request_duration_seconds`.
- **Dissent:** the Technical Lead preferred a separate T01c, merged after T01, to keep the PLAN-001 root unblocked. The human chose the same PR. T02 and T20 wait on it.
- **Deferred:**
  - OTLP log export (`otelslog`). Trigger: the T02 Collector exists.
  - Collector tail sampling. Trigger: T02.
  - Trace continuity across the outbox (the producer stores the trace context on the outbox row, and the relay starts the workflow as a linked span). Trigger: T29 and T30a start.
- **Erratum (2026-10-16):** otelhttp v0.72.0 removed `WithPublicEndpoint()`. The code uses `WithPublicEndpointFn(func(*http.Request) bool { return true })`, which does the same thing (otelhttp `handler.go`: new root span, and a remote inbound context is added as a link).
