# CI pipeline (T05)

`make ci` runs every gate locally. The GitHub workflow `.github/workflows/ci.yml`
calls only `make` targets. Install the pinned tools once with `make tools`.

## Gates, in order

| Stage | Make target | What fails it | Containers |
| --- | --- | --- | --- |
| Lint | `ci-lint` | tool versions differ from `docs/pins.md`, `gofmt`, `go vet`, golangci-lint, `lint-fixture` (exactly 5 forbidigo findings), `check-taskfile`, `check-pins`, `check-openapi`, `govulncheck ./...` in both modules at once (`vulncheck-ci`, below) | none |
| Unit | `ci-unit` | `go test -race ./...` under the host-wide race lock (below), temporal-server unit tests | none |
| Integration | `ci-integration` | any `//go:build integration` test (`internal/db/dbtest`, `internal/db/pooltest`, `internal/tenancy`, `internal/platform/health`) against throwaway PostgreSQL clusters and two PgBouncers (tags from `docs/pins.md`), or the dbtest shards running fewer tests than the package lists | Docker |
| Isolation (in `ci-integration`, after `test-integration`) | `isolation` | the tenant isolation suite (`internal/tenancy/isolation`, T11, ADR-001 CI-2): any role × tenant table, registered path or fixture failing; writes the report `isolation-report.md` at the repository root (not yet uploaded as an artifact: TODO-008) | Docker |
| Stack | `ci-stack` | image build, every container healthy, `GET /healthz` smoke from inside the compose network, then the stack tests (`pgbouncer`, `temporal`, `dependencies`, `trace`, `redact`, `idempotent`, `dbroles`, `hba`, `internal`) | compose |

In GitHub, the `temporal-server` job runs the module's tests and then
`make vulncheck-temporal-server`, which must report zero findings: there is no
exception and no allowlist (ADR-001 A1 Decision 3, TICK-121).
`lint`, `unit`, `temporal-server`, `integration`, `stack`,
`stack-db` and `stack-internal` run as seven parallel jobs (the ones a pull
request needs, see "In GitHub" below), each a separate
check (decision log, 2026-10-20). Inside each job, steps follow the order above
and the first failure stops the job. The stack tests are split over three jobs,
each with its own compose stack: `stack` (healthy, smoke, PgBouncer, every
Temporal case, SDK, dependencies, trace, redact), `stack-db` (idempotent,
dbroles, hba) and `stack-internal` (the internal overlay).

`make ci-stack` runs all of them locally (TICK-153): the main stack, then the
internal overlay as a project of its own (`<project>-int`, host ports 0 so
Docker picks free ones). `STACK_INTERNAL=beside` runs the two at once; on this
host that is slower (698 s against 669 s), since two Keycloaks then start
together. On the main stack, `healthy` gates
the rest; the read-only checks and the two Go suites (`STACK_TESTS_PARALLEL`:
smoke, PgBouncer, every `temporal_*` case, hba, SDK, identity) then run at
once, and `dependencies`, `trace`, `logs`, `redact`, `idempotent` and `dbroles`
run one at a time after them (`STACK_TESTS_SERIAL`): `dependencies` stops
Keycloak, `trace` counts every trace the Collector saw, `idempotent` restarts
the stack and `dbroles` changes roles. `ci-stack` alone adds the CI overlay
(`deploy/compose/compose.ci.yaml`), which runs the stack's PostgreSQL with
`fsync`, `synchronous_commit` and `full_page_writes` off; that run ends in
`down -v`. The test overlay (`compose.test.yaml`, which `make stack-up` uses on
the dev project) and the dev and production files never set them.

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

**govulncheck once a day per dependency set (TICK-153 D2).** `make
vulncheck-ci` (`scripts/ci/vulncheck.sh`) scans the root module and
`temporal-server/` at the same time. A module is skipped when a scan of its
current `go.mod` and `go.sum`, with the same Go and govulncheck versions,
already passed on this host today (a stamp under `$(go env GOCACHE)/nava-vulncheck`).
So a dependency change is always scanned, and the first CI run of each day
scans both modules for new advisories (ADR-001-A1's R7 trigger). The trade-off,
accepted in D2: a code-only change that starts calling an already-listed
vulnerable symbol is caught by the next day's first run, not the same run.
`VULNCHECK=all make vulncheck-ci` scans regardless; `make vulncheck` and
`make vulncheck-temporal-server` always scan, and the GitHub workflow still
calls those.

**One race-detector run per host (TICK-153 D3).** `make test` and `task test`
run `go test -race` through `tools/ci/racelock`, which holds a listener on
`127.0.0.1:47613` (`NAVA_RACE_LOCK_PORT`) while the command runs. A second run
on the host, from any checkout or agent, waits and says so every 30 s; the OS
releases the lock when the holder exits or is killed, so there is no stale
lock. `NAVA_RACE_LOCK=off` skips it.

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
make ci-isolated ID=pr-57   # all four stages at once, on a stack of its own (below)
make ci-changed ID=pr-57    # only the stages the change against origin/main needs
```

`make ci-changed` (`task ci:changed`, TICK-153 O7) asks `tools/ci/cichanged`
which stages the files changed since the merge base with `BASE` need,
uncommitted and untracked files included, and runs `ci-isolated` with them.
Every change runs `ci-lint`. Documentation (`.md` and `.html` outside a Go
package and outside `testdata`) runs nothing else; any other file under
`docs/` runs all four, since tests read some of them. A Go package's files add
`ci-unit`, plus `ci-integration` when an integration test depends on the
package and `ci-stack` when an image or a stack test does (`go list -deps`).
Compose and `deploy/` files add `ci-unit` (unit tests read the compose and
collector files) and `ci-stack`; the integration script adds `ci-integration`.
`go.mod`, `go.sum`, `docs/pins.md`, the Makefile, and any file no rule knows
run all four. A renamed file counts at both paths (`git diff --no-renames`).
`go run ./tools/ci/cichanged -v` prints the reason for each file. Whether a
`ci-changed` run can replace the full pipeline as a PR's final gate (D5) was
settled by the human on 2026-10-25 (decision log): it can.

### In GitHub

A pull request runs only the jobs its change needs. The `changes` job runs
`make ci-stages`, which prints the same stages `cichanged` picks and writes
`unit`, `integration` and `stack` as `true` or `false` for the job conditions:

| Stage needed | Jobs that run |
| --- | --- |
| always | `changes`, `lint` |
| `ci-unit` | `unit`, `temporal-server` |
| `ci-integration` | `integration` |
| `ci-stack` | `stack`, `stack-db`, `stack-internal` |

A change under `.github/workflows` runs every job, since only a full run proves
a workflow edit. A skipped job shows as skipped, not failed. Nothing runs on a
push to `main`. For a full run on any branch, start the `ci` workflow by hand
from the Actions tab (`workflow_dispatch`, or `gh workflow run ci --ref
<branch>`); it sets `CI_ALL=1`, so every job runs. Locally, `make ci-stages`
prints the stages for the current branch and `CI_ALL=1 make ci-stages` prints
all four.

## Parallel runs (TICK-147)

`make ci-isolated ID=<pr-or-sha>` (`task ci:isolated ID=...`) runs the same four
stages as `make ci`, all at once (TICK-153 O4; `PARALLEL=0` runs them in
order), on a stack nothing else shares, so several checkouts run
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
`STAGES="ci-unit ci-stack"` runs a subset. Each stage writes its own log
to `$TMPDIR/ci-isolated-<ID>/`, printed whole when the stage ends. A run killed outright (SIGKILL,
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
there. The race lock (D3) now runs those one at a time on the host, so
parallel runs queue in `ci-unit` instead of running out of memory.

**Stack slots (TICK-153 D4).** At most 7 compose stacks on the host at once;
`ci-stack` holds 1 (2 with `STACK_INTERNAL=beside`) through `tools/ci/racelock
-slots 7` on `127.0.0.1:47620-47626`, and waits, saying so every 30 s, when
fewer are free. N = 7 comes from the memory a stack holds while its tests run
(1.34 GB, `docker stats`, two stacks up) against the Docker budget (19.4 GiB),
less about 4 GB for the other agents' containers and the integration
clusters, with half again as headroom for Keycloak's start-up:
(19.4 - 4) / (1.34 x 1.5) = 7.6. `STACK_SLOTS` changes N;
`NAVA_STACK_SLOTS=off` skips the queue.

`make ci` and `make stack-up` with no variables behave as before (project
`nava`, tag `dev`, the old ports). The GitHub workflow is unchanged.
