# Review: TICK-330 Jaeger and Prometheus in the local stack (PR #290)

**Verdict: APPROVE.** No bugs found. CI is green, and the founder's flow works on a fresh local stack. One missing test is added in this PR. One isolation note goes to the Security & Multi-Tenancy Reviewer.

Read first: the TICK-330 description, `docs/architecture/observability-notes.md`, `internal/platform/telemetry/admin.go`, `cmd/nava-core/compose_test.go`.

## What was checked

- **Image contents.** `jaegertracing/jaeger:2.11.0` is Alpine-based with `/usr/bin/wget` and sets `JAEGER_LISTEN_HOST=0.0.0.0`, so the UI is reachable through the published port and not only on the container's loopback. `prom/prometheus:v3.7.3` ships busybox `wget`. Both healthchecks call a binary the image actually has.
- **Configs, using the pinned binaries.** `promtool check config` passes on `deploy/compose/prometheus/prometheus.yml`. `otelcol-contrib validate` (0.139.0) passes on `deploy/compose/otel-collector/config.yaml`.
- **Collector without Jaeger (the CI stack).** `otlp/jaeger` has a sending queue, so the fan-out to `debug` never waits on Jaeger. Retries stop after 5 s and the queue is capped at 100 batches. The `health_check` extension does not depend on exporters. CI `stack` (no `operator` profile) is green.
- **Scrape targets.** These are the six long-running binaries: core, worker, relay, groupsync, purge and onboard. Every one serves `/metrics` on `NAVA_ADMIN_ADDR` (default `:9090`) and is on `internal`. Prometheus 3 negotiates OpenMetrics by default, and `NewAdminHandler` sets `EnableOpenMetrics`, so exemplars reach storage.
- **Admin port.** Prometheus listens on 9091 in the container, so `9090` never appears in a port mapping. `TestComposePublishesNoAdminPort` is unchanged and passes.

## Acceptance run (fresh local stack, `OTEL_TAIL_SAMPLE_PERCENT=100 task stack:up`, 2026-10-29)

| Criterion | Result |
|---|---|
| Jaeger and Prometheus start and pass healthchecks | Pass. Every service is `healthy` |
| Trace in Jaeger under `nava-core` | Pass. `GET /api/v1/me` (401) shows as a trace with `http.route` and `nava.request_id`. A 404 to `/healthz` shows as operation `GET` |
| Every Prometheus target `UP` | Pass. All 6 targets `up` with no error. `query_exemplars` returns exemplars for `http_server_request_duration_seconds_bucket` |
| Only the two UIs published, on 127.0.0.1 | Pass. Published: `127.0.0.1:16686`, `127.0.0.1:9091`, plus the existing `127.0.0.1:18080` and `127.0.0.1:18081`. OTLP, admin and Collector ports are exposed but not published |
| Pins and pin checker | Pass (CI `lint`) |
| `task test`, `task lint` | Pass (CI `unit`, `lint`) |
| CI green | Pass. All jobs on run 40818995169 |

## Findings

### 1. Missing test: the published-port allowlist (added in this PR)
Before this PR, a test caught a published admin port, but not a published OTLP port, a published Collector health port, or a UI bound to `0.0.0.0`. `TestComposePublishesOnlyAllowedUIs` in `cmd/nava-core/compose_test.go` checks two things in `compose.yaml` and `compose.internal.yaml`. Every mapping must be `127.0.0.1:<host>:<container>`. Each service may publish only its allowlisted container port: keycloak 8080, nava-core 8080, jaeger 16686, prometheus 9091. Verified red by publishing `127.0.0.1:4318:4318` on jaeger, and by dropping the `127.0.0.1` prefix.

### 2. Isolation note, for the Security & Multi-Tenancy Reviewer (not a finding against this PR)
`jaeger` and `prometheus` join both `internal` and `operator`. In `compose.yaml`, `operator` is not an internal network: `deploy/compose/compose.internal.yaml` makes it internal, the base file does not. The `temporal` gateway already joins both networks in the same way. Prometheus can also read every admin `/metrics` (no auth by design) and serves it on `127.0.0.1:9091`, along with any tenant-bearing labels. Local dev only and loopback only, but the Security & Multi-Tenancy Reviewer owns the call on whether metric labels may carry tenant ids.

### 3. Preference: the PR's sample request returns 404
The PR and README suggest `curl http://127.0.0.1:18080/healthz`. `/healthz` lives on the admin listener, so this returns 404. It still yields a trace, but under the bare operation name `GET`. `curl http://127.0.0.1:18080/api/v1/me` (401, operation `GET /api/v1/me`) shows a named route. Optional doc tweak.

## How to run

- Unit (includes the compose tests): `task test`, or `go test ./cmd/nava-core/ -run TestCompose -count=1`
- CI: the GitHub `ci` workflow on the PR (`gh pr checks <pr>`)
- Try it: `make dev-ca`, then `OTEL_TAIL_SAMPLE_PERCENT=100 task stack:up`, then open `http://127.0.0.1:16686` and `http://127.0.0.1:9091/targets`
