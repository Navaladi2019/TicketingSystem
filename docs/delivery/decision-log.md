# Decision log

Delivery decisions that change a task's contract, newest first.

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
