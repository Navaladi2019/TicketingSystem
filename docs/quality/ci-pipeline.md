# CI pipeline (T05)

`make ci` runs every gate locally. The GitHub workflow `.github/workflows/ci.yml`
calls only `make` targets. Install the pinned tools once with `make tools`.

## Gates, in order

| Stage | Make target | What fails it | Containers |
| --- | --- | --- | --- |
| Lint | `ci-lint` | tool versions differ from `docs/pins.md`, `gofmt`, `go vet`, golangci-lint, `lint-fixture` (exactly 5 forbidigo findings), `check-taskfile`, `check-pins`, `check-openapi`, `govulncheck ./...` in both modules (`vulncheck`, `vulncheck-temporal-server`) | none |
| Unit | `ci-unit` | `go test -race ./...`, temporal-server unit tests | none |
| Integration | `ci-integration` | any `//go:build integration` test (`internal/db/dbtest`, `internal/db/pooltest`, `internal/platform/health`) against a throwaway PostgreSQL and two PgBouncers in front of it (tags from `docs/pins.md`) | Docker |
| Stack | `ci-stack` | image build, every container healthy, `GET /healthz` smoke from inside the compose network, then the stack tests (`pgbouncer`, `temporal`, `dependencies`, `trace`, `redact`, `idempotent`, `dbroles`, `hba`, `internal`) | compose |

In GitHub, the `temporal-server` job runs the module's tests and then
`make vulncheck-temporal-server`, which must report zero findings: there is no
exception and no allowlist (ADR-001 A1 Decision 3, TICK-121).
`lint`, `unit`, `temporal-server`, `integration`, `stack`,
`stack-db` and `stack-internal` run as seven parallel jobs, each a separate
check (decision log, 2026-10-20). Inside each job, steps follow the order above
and the first failure stops the job. The stack tests are split over three jobs,
each with its own compose stack: `stack` (healthy, smoke, PgBouncer, every
Temporal case, SDK, dependencies, trace, redact), `stack-db` (idempotent,
dbroles, hba) and `stack-internal` (the internal overlay). `make ci-stack` still
runs all of them on one stack locally.

**Time budget: under 5 minutes wall clock on a warm cache.** What keeps it there:
- Each job restores its own Go module and build cache (`actions/cache`, key per
  job and commit, falling back to the newest entry), so `go vet`, `go test` and
  the temporal-server tests compile only what changed. The lint job also caches
  the golangci-lint cache and the pinned tools (`~/go/bin`, keyed on the
  Makefile, which holds the versions).
- `make image-ci` builds the images with `docker buildx bake` and keeps the
  layers in the Actions cache (`IMAGE_CACHE=gha`), while the registry images are
  pulled alongside. The temporal-server image compiles the Temporal dependencies
  in a layer keyed on `go.mod`/`go.sum`, so a source change rebuilds one layer.
  The image no longer runs `go test`; the `temporal-server` job runs the full
  suite.
A change to `go.sum` or a Dockerfile base stage is a cold run (about 8 to 10
minutes); the next run is warm again.

Containers: `scripts/ci/test-integration.sh` owns the shared PostgreSQL that the
dbtest suites use, plus the two PgBouncers of the load test (T06,
`docs/quality/pgbouncer-load.md`) on a private network, and removes them on
exit, Ctrl-C or TERM. Every container a Go
test starts itself is removed in `t.Cleanup`. Image tags come from
`docs/pins.md`: the script exports the `postgres` row as `NAVA_TEST_PG_IMAGE`
and starts the `edoburu/pgbouncer` row.

## Risks and the layer that covers them

| Risk (by impact) | Layer |
| --- | --- |
| A failing integration test does not fail the PR | Canary `TestCICanary` (`NAVA_CI_CANARY=fail`). Under `CI`, the health integration test fails instead of skipping when Docker is missing |
| A forbidden or unpinned module or image gets in | `check-pins` (fixture tests in `tools/ci/checkpins`) |
| Taskfile and Makefile drift apart, so local runs differ from CI | `check-taskfile` (drift fixture in `tools/ci/checktaskfile`) |
| A security-path change goes without the D-P10 review | `review-paths` prints the summary lines on every PR (fixture: a PR touching `db/`) |
| A shallow clone passes `check-openapi` with nothing to compare against | `check-openapi` fails when `BASE` is missing |
| pgx's prepared-statement cache breaks through PgBouncer in transaction mode | `internal/db/pooltest` (T06): 50 clients, 10k statements, zero errors; a second PgBouncer with `max_prepared_statements = 0` must fail |
| nava-core goes ready without PostgreSQL | compose healthcheck on `/readyz`, plus the `dependencies` case (200 with Keycloak stopped) |

## Deliberately not covered

- **testcontainers-go.** Approved, not adopted (decision log, 2026-10-19). The integration tests start a pinned container through the docker CLI, the way T03 and T04 built them.

## Run it

```sh
make tools        # once
make ci           # all four stages
make ci-lint      # or one stage
NAVA_CI_CANARY=fail make test-integration ARGS="-run TestCICanary"   # canary, must fail
```
