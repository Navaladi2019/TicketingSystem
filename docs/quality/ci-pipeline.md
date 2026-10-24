# CI pipeline (T05)

`make ci` runs every gate locally. The GitHub workflow `.github/workflows/ci.yml`
calls only `make` targets. Install the pinned tools once with `make tools`.

## Gates, in order

| Stage | Make target | What fails it | Containers |
| --- | --- | --- | --- |
| Lint | `ci-lint` | tool versions differ from `docs/pins.md`, `gofmt`, `go vet`, golangci-lint, `lint-fixture` (exactly 5 forbidigo findings), `check-taskfile`, `check-pins`, `check-openapi`, `govulncheck ./...` in both modules (`vulncheck`, `vulncheck-temporal-server`) | none |
| Unit | `ci-unit` | `go test -race ./...`, temporal-server unit tests | none |
| Integration | `ci-integration` | any `//go:build integration` test (`internal/db/dbtest`, `internal/db/pooltest`, `internal/tenancy`, `internal/platform/health`) against throwaway PostgreSQL clusters and two PgBouncers (tags from `docs/pins.md`), or the dbtest shards running fewer tests than the package lists | Docker |
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

Containers: `scripts/ci/test-integration.sh` owns, on a private network, one
shared PostgreSQL with the two PgBouncers of the load test (T06,
`docs/quality/pgbouncer-load.md`) in front of it, plus `DBTEST_SHARDS` (default
3) more clusters for dbtest, and removes them on exit, Ctrl-C or TERM. Every
container a Go test starts itself is removed in `t.Cleanup`. Image tags come
from `docs/pins.md`: the script exports the `postgres` row as
`NAVA_TEST_PG_IMAGE` and starts the `edoburu/pgbouncer` row. Every cluster runs
with `fsync`, `synchronous_commit` and `full_page_writes` off: they are thrown
away, and the dev and production compose files never set these (TICK-153 O1).

Three groups run at once (TICK-153 O3): dbtest's top-level tests, dealt
round-robin over the shard clusters, one `go test -run` each; the other
packages that use the shared cluster (`pooltest`, `tenancy`, `-p 1`, since they
create the same role); and the packages that use none (`health`, which starts
its own container). The script lists dbtest's tests first and fails unless the
shards ran every one of them. Extra arguments (`ARGS=-run ...`) run every
package one after another on the shared cluster, as before.

Inside dbtest, `migratedCluster` copies each manifest database from a template
taken after the first real db-roles and migrator run in the process
(`CREATE DATABASE ... TEMPLATE`, TICK-153 O2), and gives the copy its grants
again. It copies only while every role, membership and role setting is still
byte for byte what it was when the templates were taken, and falls back to the
full rebuild otherwise; `TestMigratedClusterMatchesFreshRun` proves a copy
matches a real run. Tests of db-roles or of a migrator run from scratch
(`roles_test.go`, `d28_test.go`, and the migrate tests that read the run's own
result) still call `freshCluster`.

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
make ci-isolated ID=pr-57   # all four stages on a stack of its own (below)
```

## Parallel runs (TICK-147)

`make ci-isolated ID=<pr-or-sha>` (`task ci:isolated ID=...`) runs the same four
stages as `make ci`, on a stack nothing else shares, so several checkouts run
CI at once on one Docker host. Run it from each PR's own checkout; one run per
checkout at a time (the dev certificates and `ci6-report.md` live in it).

| Shared on the host before | Per run now | Set by |
| --- | --- | --- |
| compose project `nava` (containers, networks, volumes) | `nava-<ID>` | `COMPOSE_PROJECT_NAME` |
| image tags `nava/*:dev` | `nava/*:<ID>` | `NAVA_IMAGE_TAG` (compose.yaml, `TEMPORAL_SERVER_IMAGE`, the stack tests) |
| host ports 18080, 18081, 17233, 18233 | four ports in 20000-21999 from a checksum of ID, skipping ports a running container publishes; printed at the start | `NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `TEMPORAL_TEST_PORT`, `UI_PORT` |

The ID is lowercased; characters other than `a-z 0-9 _ -` become `-`; `dev`
is refused. On exit, pass or fail, the run removes its project
(`down -v --remove-orphans`) and untags its four `nava/*:<ID>` images. It never
prunes, so shared layers and the build cache stay. It then fails if any
`nava-<ID>` container, volume, network or image tag is left.
`STAGES="ci-unit ci-stack"` runs a subset. A run killed outright (SIGKILL,
a closed session) skips that cleanup; `make ci-isolated-down ID=<ID>` does it.
The `integration` stage was already safe in parallel: it names its containers
by PID and takes random host ports.

**Caches, shared by every run.** Nothing is set per run and nothing is cleaned.
- Go: the per-user `GOCACHE`, `GOMODCACHE` and golangci-lint cache, safe for
  concurrent use. The `go1.26.10` toolchain downloads once (`GOTOOLCHAIN=auto`).
- Pinned tools: `make tools` runs only when `make check-tools` fails.
- Images: `IMAGE_CACHE=local` makes `ci-stack` build with `make image-ci`
  (`buildx bake`) on the local builder. Its BuildKit cache is shared by every
  tag and safe for concurrent builds, so per-run tags still share layers. There
  is no `type=local` cache directory, because two concurrent exports to one
  directory can corrupt it.
- Registry images: `image-ci` pulls with `--policy missing`, `compose up`
  pulls only missing images, and the integration script's `docker run` uses the
  local copy.

**How many fit.** Measured on this host (16 CPU, about 20 GB for Docker); see
TICK-147 for the runs. One stack holds about 1.2 GB at rest; Keycloak (about
0.8 GB) is the largest part. The limit is memory during `ci-unit`, not the
stacks: `go test -race ./...` across both modules peaks at several GB, and a
fully loaded host has hit `fatal error: runtime: cannot allocate memory`
there. **Run at most 3 at once**, or 2 while other agents build. Stacks alone
would allow about 8.

`make ci` and `make stack-up` with no variables behave as before (project
`nava`, tag `dev`, the old ports). The GitHub workflow is unchanged.
