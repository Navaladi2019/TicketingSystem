# Pinned versions

Every container image, Go module and CI tool the platform depends on, with the
exact version, why it is pinned there, and the ruling or task that set it.
`make check-pins` (`tools/ci/checkpins`) enforces this file:

- every direct `require` in every `go.mod` has a row in **Go modules**, and the
  row's version names the version in that `go.mod` (or is a `≥` floor it meets);
- no `*.go` or `*.sh` file (outside `docs/` and the checker's own
  `tools/ci/checkpins/`) spells an image
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
| Go (platform module) | `go 1.26.0`, toolchain go1.26.10 | Floor Go ≥ 1.26 for platform code. go1.26.10 is the toolchain the images build with (`golang:1.26.10-alpine`). Raised from go1.26.9 for GO-2026-6809 (crypto/x509), fixed in go1.26.10. | D-P3, TICK-124 |
| Go (`temporal-server`) | `go 1.26.0`, toolchain go1.26.10 | Pinned separately with the custom server build (R7); the module is not platform code. Raised from go1.25.8, whose standard library had reachable govulncheck advisories (net/http, crypto/tls, crypto/x509, html/template, net/url, encoding/xml, encoding/asn1, net, net/textproto, mime/multipart). Same floor and toolchain as the platform module (go1.26.10 for GO-2026-6809). | D-P3, T27a, TICK-111, TICK-124 |
| `golangci-lint` | v2.14.0 | Lint stage. Uses its bundled depguard (`OpenPeeDeeP/depguard/v2 v2.2.1`), so depguard has no row. Module plugins (`golangci-lint custom`) are forbidden (review rule, TICK-94 item 15). | TICK-93 audit, T05 |
| `govulncheck` (`golang.org/x/vuln`) | v1.9.0 | `make vulncheck`: no reachable vulnerability in the platform module. | TICK-94 item 16, T05 |
| `actions/checkout`, `actions/setup-go`, `actions/cache`, `docker/setup-buildx-action`, `crazy-max/ghaction-github-runtime` | v4, v5, v4, v3, v3 | GitHub Actions only check out, install Go, restore caches (Go build cache per job, CI tools by version) and give `make image-ci` a buildx builder with the Actions cache (`type=gha`); every gate runs through `make`. | T05, T28 |

## Images

| Component | Version | Why | Source |
| --- | --- | --- | --- |
| `postgres` | 16.10 | An exact `postgres:16.x` patch tag, floor 15. Role semantics and `pg_auth_members` columns differ between 15 and 16, which CI-3 depends on. Also the integration-test database (`make test-integration`, `internal/platform/health` integration test). | D-P3, ADR-001 §3 |
| `edoburu/pgbouncer` | v1.24.1-p1 | PgBouncer ≥ 1.21 (transaction mode with `max_prepared_statements`). Also the two poolers of the PgBouncer load test (`internal/db/pooltest`): `scripts/ci/test-integration.sh` reads this row. A bump must keep that test green. | D-P3, T02, T06 |
| `quay.io/keycloak/keycloak` | 26.8.1 | Floor 26.1.3; known-broken 26.0.0–26.1.2 (the `kc.org` picker tamper). Supersedes Epic 01's "26.0" row. | D-P3, D26 |
| `temporalio/auto-setup` | 1.29.7 | Temporal server 1.29.7 with PostgreSQL persistence and SQL visibility, no Elasticsearch, the same version as the custom build (ADR-001 A1 Decision 1; the last `auto-setup` tag). T28 replaced it in compose with the T27a build (`nava/temporal-server`); no compose file uses it now, so this row only keeps the version in step if one ever does. | D-P3, T02, T28, ADR-001 A1, TICK-121 |
| `temporalio/ui` | 2.40.1 | Temporal Web UI for the `operator` profile only. It publishes no port; operators reach it through `make operator-tunnel`. Wields the `nava-operator` certificate (TODO-006). | T28 |
| `otel/opentelemetry-collector-contrib` | 0.139.0 | The Collector (tail sampling, redaction layer two). | T01, T02 |
| `golang` | 1.26.10-alpine @deploy/compose, 1.26.10-bookworm @temporal-server | Build stages: 1.26.10 for nava-core, nava-db and the custom Temporal server (raised from 1.25.8-bookworm in TICK-111, then from 1.26.9 for GO-2026-6809 in TICK-124). Matches each module's toolchain. | D-P3, T27a, TICK-111, TICK-124 |
| `alpine` | 3.22.1 | Runtime base of nava-core, nava-worker, nava-db and the Collector wrapper. | T02 |
| `debian` | bookworm-20261005-slim (digest `sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587`) | Runtime base of the custom Temporal server. A dated Debian tag, digest-pinned in `temporal-server/Dockerfile`; replaces the floating `bookworm-slim`. Bump tag and digest together. | T27a, TICK-111 |
| `nava/nava-core`, `nava/nava-db`, `nava/nava-worker`, `nava/otel-collector`, `nava/temporal-server` | dev | Built locally from this repo by `docker compose build`; never pulled. `nava/temporal-server` is the T27a custom server (`temporal-server/Dockerfile`), used by the `temporal-schema`, `temporal-server` and `temporal` services. | T02, T28 |

## Go modules

| Component | Version | Why | Source |
| --- | --- | --- | --- |
| `go.temporal.io/sdk` | v1.49.0 | Temporal client and workers. | D-P3 |
| `go.temporal.io/server` | v1.29.7 @temporal-server | Custom server module (`temporal-server` only), matches the server image. Moved from v1.28.4 so that `make vulncheck-temporal-server` reports zero findings (GO-2026-5766 is flagged on the 1.28.4 module version); not v1.29.5, which reopens GO-2026-5578. Main schema 1.18, visibility 1.9. | D-P3, R7, ADR-001 A1, TICK-121 |
| `go.temporal.io/api` | v1.63.5 @., v1.53.0 @temporal-server | Temporal API types. Platform: required by SDK v1.49.0. `temporal-server`: what server v1.29.7 requires (`go mod tidy` resolves v1.53.0, no higher). The two modules are built separately, so the versions may differ. | D-P3, R7, ADR-001 A1, TICK-121 |
| `google.golang.org/grpc` | v1.84.0 @., v1.83.2 @temporal-server | gRPC for the Temporal SDK client. Platform: not chosen directly; goose v3.29.0 requires it, so MVS raises it. `temporal-server`: raised from v1.79.3 (what server v1.28.4 builds with), which had reachable govulncheck advisories, then from v1.83.1 to v1.83.2 for GO-2026-6443 (imported, not called; hygiene). Server v1.29.7 and API v1.53.0 unchanged by the bump. | T04 (v1.83.2), raised by goose in T03; T27a, TICK-111, TICK-128 |
| `google.golang.org/protobuf` | v1.36.11 @temporal-server | Protobuf runtime for the custom server. | T27a |
| `github.com/prometheus/client_golang` | v1.25.0 | Prometheus `/metrics` per the observability standard. | T01, TICK-89 |
| `go.opentelemetry.io/otel` | v1.47.0 | OpenTelemetry API. | T01, TICK-89 (human decision 2026-10-16) |
| `go.opentelemetry.io/otel/metric` | v1.47.0 | OpenTelemetry metric API. | T01, TICK-89 |
| `go.opentelemetry.io/otel/trace` | v1.47.0 | OpenTelemetry trace API. | T01, TICK-89 |
| `go.opentelemetry.io/otel/sdk` | v1.47.0 | Trace SDK. | T01, TICK-89 |
| `go.opentelemetry.io/otel/sdk/metric` | v1.47.0 | Metric SDK. | T01, TICK-89 |
| `go.opentelemetry.io/otel/exporters/prometheus` | v0.69.0 | Bridges OTel metrics to `/metrics`. | T01, TICK-89 |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` | v1.47.0 | OTLP trace exporter to the Collector. | T01, TICK-89 |
| `go.opentelemetry.io/contrib/bridges/otelslog` | v0.21.0 | slog bridge: exports log records over OTLP with the request's trace id (OTLP log export, deferred in the 2026-10-16 decision until the Collector existed). | TICK-101 |
| `go.opentelemetry.io/otel/sdk/log` | v1.47.0 | Log SDK behind the bridge. | TICK-101 |
| `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp` | v0.23.0 | OTLP log exporter to the Collector. | TICK-101 |
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
| `github.com/exaring/otelpgx` | v0.13.0 | pgx tracing (`internal/platform/db.NewPool`); query parameters off by default. Must pass providers; `WithIncludeQueryParameters` banned by forbidigo. | TICK-94 item 4, T07 |
| `go.temporal.io/sdk/contrib/opentelemetry` | v0.8.1 | Temporal tracing and metrics. Imports SDK internals: rebuild on any SDK bump; `OnError` defaults to panic, so `telemetry.NewTemporal` sets it to log. Used by the worker (T31) and the relay. | TICK-94, T31 (TICK-65) |
| `github.com/google/uuid` | v1.6.0 | UUIDs, including v5 (`internal/tenancy`). `gofrs/uuid` is forbidden. | TICK-94 item 10, T07 |
| `github.com/lestrrat-go/jwx/v3` | v3.3.0 | JWT and JWKS, imported only by `internal/identity/keycloak` (depguard `identity-port`). jwx v2 before 2.0.21 has CVE-2024-28122 and others; `go-oidc` rejected (kid-miss amplification). Raises `github.com/goccy/go-json` v0.10.2 to v0.10.6 by MVS. | TICK-94 item 11, T15 |
| `github.com/lestrrat-go/httprc/v3` | v3.0.6 | jwx's JWKS cache client, the version jwx v3.3.0 requires. A direct require because `jwk.NewCache` takes an `*httprc.Client`, which the adapter builds with the shared outbound client and a URL allow-list (never `http.DefaultClient`). Imported only by `internal/identity/keycloak`. | TICK-94 item 11, T15 |
| `golang.org/x/crypto` | ≥ v0.56.0 | Transitive (jwx v3.3.0 asks for v0.55.0, which has GO-2026-6354 and GO-2026-6355). The graph already holds v0.58.0 by MVS, so `go.mod` records v0.58.0 indirect; do not pin it lower. | TICK-94 item 11, T15 |
| `golang.org/x/time` | v0.16.0 | `rate` token bucket. Today the graph holds v0.14.0 indirectly (through the Temporal SDK); the first direct use takes v0.16.0. | TICK-94 |
| `golang.org/x/tools` | v0.52.0 @tools/navalint | `navalint` analyzers (`tools/navalint`, its own module). Not a direct require of the production `go.mod`; v0.51.0 is there indirectly (goose, oapi-codegen); `tools/navalint` tests this. | TICK-94 item 15, T07 |
| `github.com/testcontainers/testcontainers-go` | v0.44.0 (+ `modules/postgres`) | Approved, not adopted (T05 ruling, 2026-10-18). The integration tests use the docker CLI with the `postgres` image from this file (`make test-integration`). If testcontainers is adopted later, it is for Postgres-level tests only, and it brings the `moby/go-archive` ≥ v0.3.0 pin with it: with `moby/go-archive` < v0.3.0 it has GO-2026-6253. | TICK-94 item 16; decision log 2026-10-19 |
| `github.com/moby/go-archive` | ≥ v0.3.0 | Approved, not adopted (T05 ruling, 2026-10-18). Transitive through testcontainers, so it comes in only if testcontainers is adopted. v0.2.0 has GO-2026-6253. | TICK-94 item 16; decision log 2026-10-19 |
| Keycloak admin client | none | Hand-written in `internal/identity`. `gocloak` rejected (no organization API in v13; a second JWT path). | TICK-94 item 12 |

## `temporal-server` transitive raises

Indirect requires of `temporal-server/go.mod` that TICK-111 raised to clear
reachable govulncheck advisories, or that MVS raised with them, measured
against what server v1.28.4 builds with. TICK-121's move to server v1.29.7
left every row below unchanged; it raised only what v1.29.7 itself requires
(`go.temporal.io/sdk` v1.35.0, `github.com/apache/thrift` v0.23.0,
`github.com/klauspost/compress` v1.18.0, `go.uber.org/dig` v1.19.0,
`golang.org/x/exp` v0.0.0-20250620022241-b7579e27df2b) and dropped
`go.temporal.io/version`. `make check-pins` checks only direct requires, so
this table is documentation.

| Module (`temporal-server`) | Version | Why | Source |
| --- | --- | --- | --- |
| `golang.org/x/net` | v0.60.0 (from v0.51.0) | Reachable advisories in v0.51.0. | TICK-111 |
| `golang.org/x/text` | v0.42.0 (from v0.34.0) | Reachable advisories in v0.34.0. | TICK-111 |
| `github.com/jackc/pgx/v5` | v5.9.2 (from v5.7.2) | Reachable advisories in v5.7.2 (persistence driver). | TICK-111 |
| `github.com/go-jose/go-jose/v4` | v4.1.4 (from v4.1.3) | Reachable advisory in v4.1.3. | TICK-111 |
| `go.opentelemetry.io/otel`, `/metric`, `/trace`, `/sdk`, `/sdk/metric` | v1.45.0 (from v1.40.0) | Reachable advisories in otel / otel/sdk v1.40.0; the API and SDK modules move together. | TICK-111 |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace`, `/otlptrace/otlptracegrpc` | v1.45.0 (from v1.34.0) | Reachable advisories in v1.34.0; same release as the SDK. | TICK-111 |
| `go.opentelemetry.io/proto/otlp` | v1.11.0 (from v1.7.1) | Required by the otlptrace v1.45.0 exporters (MVS). | TICK-111 |
| `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`, `/net/http/otelhttp` | v0.61.0 (from v0.59.0) | Raised by MVS with otel v1.45.0. | TICK-111 |
| `go.opentelemetry.io/contrib/detectors/gcp` | v1.44.0 (from v1.39.0) | Raised by MVS with otel v1.45.0. | TICK-111 |
| `golang.org/x/crypto` | v0.57.0 (from v0.48.0) | Raised by MVS with x/net v0.60.0. | TICK-111 |
| `golang.org/x/sys`, `golang.org/x/sync`, `golang.org/x/oauth2`, `golang.org/x/time` | v0.48.0, v0.23.0, v0.36.0, v0.14.0 (from v0.41.0, v0.19.0, v0.35.0, v0.10.0) | Raised by MVS with x/net, grpc and otel. | TICK-111 |
| `google.golang.org/genproto`, `/googleapis/api`, `/googleapis/rpc` | v0.0.0-20260128011058-8636f8732409, v0.0.0-20260803160001-6ac0973c030d, v0.0.0-20260803160001-6ac0973c030d | Raised by MVS with grpc v1.83.1. | TICK-111 |
| `google.golang.org/api`, `cloud.google.com/go` (+ `auth`, `auth/oauth2adapt`, `iam`, `monitoring`, `storage`), `github.com/googleapis/gax-go/v2`, `github.com/googleapis/enterprise-certificate-proxy` | v0.264.0; v0.121.6 (v0.18.2, v0.2.8, v1.5.3, v1.24.3, v1.56.0); v2.17.0; v0.3.11 | Raised by MVS with grpc and otel (server's GCS archival and Stackdriver deps; not used by our config). | TICK-111 |
| `github.com/GoogleCloudPlatform/opentelemetry-operations-go/*` | detectors/gcp v1.33.0, exporter/metric and internal/resourcemapping v0.53.0 | Raised by MVS with otel v1.45.0. | TICK-111 |
| `github.com/grpc-ecosystem/grpc-gateway/v2`, `github.com/envoyproxy/go-control-plane/envoy`, `github.com/envoyproxy/protoc-gen-validate`, `github.com/cncf/xds/go`, `github.com/spiffe/go-spiffe/v2`, `cel.dev/expr`, `github.com/go-logr/logr` | v2.29.0, v1.37.0, v1.3.3, v0.0.0-20260202195803-dba9d589def2, v2.7.0, v0.25.2, v1.4.4 | Raised by MVS with grpc v1.83.1. `github.com/cenkalti/backoff/v5` v5.0.3 is new (otlptrace v1.45.0). | TICK-111 |

`make vulncheck-temporal-server` reports zero findings on server v1.29.7 and
is a required step of the `temporal-server` CI job (ADR-001 A1 Decision 3).
There is no exception and no allowlist: a new finding fails CI and is a new
decision, not a pin bump.

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
