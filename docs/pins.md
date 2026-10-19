# Pinned versions

Every container image, Go module and CI tool the platform depends on, with the
exact version, why it is pinned there, and the ruling or task that set it.
`make check-pins` (`tools/ci/checkpins`) enforces this file:

- every direct `require` in every `go.mod` has a row in **Go modules**, and the
  row's version names the version in that `go.mod` (or is a `≥` floor it meets);
- no `*.go` or `*.sh` file (outside `docs/` and `testdata/`) spells an image
  from **Images** with a tag, such as `postgres:16.10`: code and scripts read
  the tag from this file (`scripts/ci/test-integration.sh` exports it as
  `NAVA_TEST_PG_IMAGE`);
- every `image:` in the compose files and every `FROM` in a Dockerfile has a
  row in **Images** whose version names that exact tag;
- nothing listed under **Forbidden** appears in any `go.mod` or `go.sum`,
  indirect dependencies included.

The first column holds the module path or image name in backticks. When
modules or images differ, the version column scopes each version to a
directory with `@dir`: `v1.84.0 @., v1.79.3 @temporal-server`. A scoped
version matches only the `go.mod` in exactly that directory (`.` is the
repository root), or only Dockerfiles under that directory. Text in
parentheses is a note and never matches.
Image tags in code and tests come from this file, never from memory (TICK-94
item 16, condition 3).

## Toolchain and CI tools

| Component | Version | Why | Source |
| --- | --- | --- | --- |
| Go (platform module) | `go 1.26.0`, toolchain go1.26.9 | Floor Go ≥ 1.26 for platform code. go1.26.9 is the toolchain the images build with (`golang:1.26.9-alpine`). | D-P3 |
| Go (`temporal-server`) | `go 1.25.8` | Pinned separately with the custom server build (R7); the module is not platform code. govulncheck reports reachable standard-library vulnerabilities fixed in go1.25.9 / go1.25.10: TICK-111. | D-P3, T27a |
| `golangci-lint` | v2.14.0 | Lint stage. Uses its bundled depguard (`OpenPeeDeeP/depguard/v2 v2.2.1`), so depguard has no row. Module plugins (`golangci-lint custom`) are forbidden (review rule, TICK-94 item 15). | TICK-93 audit, T05 |
| `govulncheck` (`golang.org/x/vuln`) | v1.9.0 | `make vulncheck`: no reachable vulnerability in the platform module. | TICK-94 item 16, T05 |
| `actions/checkout`, `actions/setup-go` | v4, v5 | GitHub Actions only check out and install Go; every gate runs through `make`. | T05 |

## Images

| Component | Version | Why | Source |
| --- | --- | --- | --- |
| `postgres` | 16.10 | An exact `postgres:16.x` patch tag, floor 15. Role semantics and `pg_auth_members` columns differ between 15 and 16, which CI-3 depends on. Also the integration-test database (`make test-integration`, `internal/platform/health` integration test). | D-P3, ADR-001 §3 |
| `edoburu/pgbouncer` | v1.24.1-p1 | PgBouncer ≥ 1.21 (transaction mode with `max_prepared_statements`). | D-P3, T02 |
| `quay.io/keycloak/keycloak` | 26.8.1 | Floor 26.1.3; known-broken 26.0.0–26.1.2 (the `kc.org` picker tamper). Supersedes Epic 01's "26.0" row. | D-P3, D26 |
| `temporalio/auto-setup` | 1.28.4 | Temporal server 1.28.4 with PostgreSQL persistence and SQL visibility, no Elasticsearch. The compose stack and CI use `auto-setup` (stock server plus schema setup) rather than bare `temporalio/server`; T28 swaps in the T27a build. | D-P3, T02 |
| `otel/opentelemetry-collector-contrib` | 0.139.0 | The Collector (tail sampling, redaction layer two). | T01, T02 |
| `golang` | 1.26.9-alpine @deploy/compose, 1.25.8-bookworm @temporal-server | Build stages: 1.26.9 for nava-core and nava-db, 1.25.8 for the custom Temporal server. Matches each module's toolchain. | D-P3, T27a |
| `alpine` | 3.22.1 | Runtime base of nava-core, nava-db and the Collector wrapper. | T02 |
| `debian` | bookworm-slim | Runtime base of the custom Temporal server. **Not an exact tag**: a known gap, fixed by TICK-111; the row stays until the Dockerfile pins a dated tag. | T27a |
| `nava/nava-core`, `nava/nava-db`, `nava/otel-collector` | dev | Built locally from this repo by `docker compose build`; never pulled. | T02 |

## Go modules

| Component | Version | Why | Source |
| --- | --- | --- | --- |
| `go.temporal.io/sdk` | v1.49.0 | Temporal client and workers. | D-P3 |
| `go.temporal.io/server` | v1.28.4 @temporal-server | Custom server module (`temporal-server` only), matches the server image. | D-P3, R7 |
| `go.temporal.io/api` | v1.63.5 @., v1.50.1 @temporal-server | Temporal API types. Platform: required by SDK v1.49.0. `temporal-server`: matches server v1.28.4. The two modules are built separately, so the versions may differ. | D-P3, R7 |
| `google.golang.org/grpc` | v1.84.0 @., v1.79.3 @temporal-server | gRPC for the Temporal SDK client. Platform: not chosen directly; goose v3.29.0 requires it, so MVS raises it. `temporal-server`: what server v1.28.4 builds with (govulncheck reports reachable advisories; TICK-111). | T04 (v1.83.2), raised by goose in T03; T27a |
| `google.golang.org/protobuf` | v1.36.11 @temporal-server | Protobuf runtime for the custom server. | T27a |
| `github.com/prometheus/client_golang` | v1.25.0 | Prometheus `/metrics` per the observability standard. | T01, TICK-89 |
| `go.opentelemetry.io/otel` | v1.47.0 | OpenTelemetry API. | T01, TICK-89 (human decision 2026-10-16) |
| `go.opentelemetry.io/otel/metric` | v1.47.0 | OpenTelemetry metric API. | T01, TICK-89 |
| `go.opentelemetry.io/otel/trace` | v1.47.0 | OpenTelemetry trace API. | T01, TICK-89 |
| `go.opentelemetry.io/otel/sdk` | v1.47.0 | Trace SDK. | T01, TICK-89 |
| `go.opentelemetry.io/otel/sdk/metric` | v1.47.0 | Metric SDK. | T01, TICK-89 |
| `go.opentelemetry.io/otel/exporters/prometheus` | v0.69.0 | Bridges OTel metrics to `/metrics`. | T01, TICK-89 |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` | v1.47.0 | OTLP trace exporter to the Collector. | T01, TICK-89 |
| `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` | v0.72.0 | HTTP server and outbound-client spans. | T01, TICK-89 |
| `go.opentelemetry.io/contrib/instrumentation/runtime` | v0.72.0 | Go runtime metrics. | T01, TICK-89 |
| `github.com/go-logr/logr` | v1.4.4 | Routes the OTel SDK's internal logger into slog (`internal/platform/telemetry`). | T01 |
| `github.com/felixge/httpsnoop` | v1.1.0 | Captures the status and size of a response without hiding its optional interfaces (`internal/platform/httpobs`). Already in the graph through otelhttp. | T01 |
| `github.com/pressly/goose/v3` | v3.29.0 | Migration runner. Session advisory lock (`lock.NewPostgresSessionLocker`) set explicitly. `golang-migrate` (dirty flag) and `jackc/tern` (`text/template` over SQL) are forbidden; no hand-rolled runner. Its go.mod raises, by MVS: `google.golang.org/grpc` v1.83.2 to v1.84.0 (own row), `golang.org/x/net` v0.59.0 to v0.61.0, `golang.org/x/sys` v0.48.0 to v0.49.0, `golang.org/x/sync` v0.23.0 to v0.24.0, `golang.org/x/text` v0.42.0 to v0.43.0, `golang.org/x/tools` v0.49.0 to v0.51.0, `github.com/jackc/puddle/v2` v2.2.2 to v2.2.3 and `google.golang.org/genproto/googleapis/{api,rpc}` to `v0.0.0-20261005182115-fad411399dd8` (rows below). | TICK-94 item 8, T03 |
| `go.yaml.in/yaml/v3` | v3.0.5 | The one strict YAML loader (`internal/platform/strictyaml`, `KnownFields`) for `db/grants.yaml` and `api/errors.yaml`. Not the archived `gopkg.in/yaml.v3`. | TICK-94 item 9, T03 |
| `github.com/jackc/pgx/v5` | v5.11.0 | PostgreSQL driver for the jobs and the core. The version goose v3.29.0 is built against. | T03, T04 |
| `github.com/getkin/kin-openapi` | v0.151.0 | OpenAPI 3.1 validation. Register formats via options; never echo `Reason`. | TICK-94 item 13, T20 |
| `github.com/oapi-codegen/oapi-codegen/v2` | v2.8.0 | 3.1 codegen with ServeMux patterns; a `tool` directive, so `// indirect` in `go.mod`. `ogen` rejected (no `r.Pattern`, global otel). | TICK-94 item 13, T20 |
| `github.com/oapi-codegen/runtime` | v1.7.0 | Runtime helpers the generated `api/api.gen.go` imports. Comes with oapi-codegen v2.8.0. | T20 |
| `github.com/oapi-codegen/nethttp-middleware` | v1.2.0 | Request validation. The default error handler echoes values: always set our own. | TICK-94 item 13, T20 |
| `golang.org/x/sync` | v0.24.0 | Transitive. Raised from v0.23.0 by goose in T03. | T03 (Security Reviewer, TICK-54) |
| `golang.org/x/text` | v0.43.0 | Transitive. Raised from v0.42.0 by goose in T03. | T03 (Security Reviewer, TICK-54) |
| `github.com/jackc/puddle/v2` | v2.2.3 | Transitive (pgxpool). Raised from v2.2.2 by goose in T03. | T03 (Security Reviewer, TICK-54) |
| `google.golang.org/genproto/googleapis/api`, `google.golang.org/genproto/googleapis/rpc` | v0.0.0-20261005182115-fad411399dd8 | Transitive (gRPC). Raised by goose in T03. | T03 (Security Reviewer, TICK-54) |
| `github.com/exaring/otelpgx` | v0.13.0 | pgx tracing; query parameters off by default. Must pass providers; ban `WithIncludeQueryParameters`. Not in the graph yet. | TICK-94 |
| `go.temporal.io/sdk/contrib/opentelemetry` | v0.8.1 | Temporal tracing and metrics. Imports SDK internals: rebuild on any SDK bump; `OnError` defaults to panic. Not in the graph yet. | TICK-94 |
| `github.com/google/uuid` | v1.6.0 | UUIDs, including v5. Indirect today. | TICK-94 item 10 |
| `github.com/lestrrat-go/jwx/v3` | v3.3.0 | JWT and JWKS. jwx v2 before 2.0.21 has CVE-2024-28122 and others; `go-oidc` rejected (kid-miss amplification). Not in the graph yet. | TICK-94 item 11 |
| `golang.org/x/crypto` | ≥ v0.56.0 | Transitive (jwx). v0.55.0 has GO-2026-6354 and GO-2026-6355. Not in the graph yet. | TICK-94 |
| `golang.org/x/time` | v0.16.0 | `rate` token bucket. Today the graph holds v0.14.0 indirectly (through the Temporal SDK); the first direct use takes v0.16.0. | TICK-94 |
| `golang.org/x/tools` | v0.52.0 (`navalint` tools module, when it lands) | `navalint` analyzers. Not a direct require of the production `go.mod`; v0.51.0 is there indirectly (goose, oapi-codegen). | TICK-94 |
| `github.com/testcontainers/testcontainers-go` | v0.44.0 (+ `modules/postgres`) | Approved, not adopted (T05 ruling, 2026-10-18). The integration tests use the docker CLI with the `postgres` image from this file (`make test-integration`). If testcontainers is adopted later, it is for Postgres-level tests only, and it brings the `moby/go-archive` ≥ v0.3.0 pin with it: with `moby/go-archive` < v0.3.0 it has GO-2026-6253. | TICK-94 item 16; decision log 2026-10-19 |
| `github.com/moby/go-archive` | ≥ v0.3.0 | Approved, not adopted (T05 ruling, 2026-10-18). Transitive through testcontainers, so it comes in only if testcontainers is adopted. v0.2.0 has GO-2026-6253. | TICK-94 item 16; decision log 2026-10-19 |
| Keycloak admin client | none | Hand-written in `internal/identity`. `gocloak` rejected (no organization API in v13; a second JWT path). | TICK-94 item 12 |

## Forbidden

`make check-pins` fails when any of these appears anywhere in a `go.mod` or
`go.sum` (indirect included). The first column is a module path prefix, so
`github.com/coreos/go-oidc` also matches `github.com/coreos/go-oidc/v3`.

| Module | Replacement | Ruling |
| --- | --- | --- |
| `github.com/golang-migrate/migrate` | `pressly/goose/v3` | TICK-94 item 8 |
| `github.com/jackc/tern` | `pressly/goose/v3` | TICK-94 item 8 |
| `github.com/coreos/go-oidc` | `lestrrat-go/jwx/v3` | TICK-94 item 11 |
| `github.com/Nerzal/gocloak` | hand-written client in `internal/identity` | TICK-94 item 12 |
| `github.com/ogen-go/ogen` | `oapi-codegen/v2` + `kin-openapi` | TICK-94 item 13 |
| `github.com/gofrs/uuid` | `google/uuid` | TICK-94 item 10 |

Golangci-lint module plugins (`golangci-lint custom`) are also forbidden
(TICK-94 item 15). They are a build mode, not a module, so this one is a
review rule.
