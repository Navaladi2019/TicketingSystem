# CI pipeline (T05)

`make ci` runs every gate locally. The GitHub workflow `.github/workflows/ci.yml`
calls only `make` targets. Install the pinned tools once with `make tools`.

## Gates, in order

| Stage | Make target | What fails it | Containers |
| --- | --- | --- | --- |
| Lint | `ci-lint` | tool versions differ from `docs/pins.md`, `gofmt`, `go vet`, golangci-lint, `lint-fixture` (exactly 5 forbidigo findings), `check-taskfile`, `check-pins`, `check-openapi`, `govulncheck ./...` | none |
| Unit | `ci-unit` | `go test -race ./...`, temporal-server unit tests | none |
| Integration | `ci-integration` | any `//go:build integration` test (`internal/db/dbtest`, `internal/platform/health`) against a throwaway PostgreSQL (tag from `docs/pins.md`) | Docker |
| Stack | `ci-stack` | image build, every container healthy, `GET /healthz` smoke from inside the compose network, then the stack tests (`pgbouncer`, `temporal`, `dependencies`, `trace`, `redact`, `idempotent`, `dbroles`, `hba`, `internal`) | compose |

In GitHub, `lint-unit`, `temporal-server`, `integration` and `stack` run as four
parallel jobs, each a separate required check (decision log, 2026-10-19).
Inside each job, steps follow the order above and the first failure stops the job.
Lint and unit tests share one job with no containers.

Containers: `scripts/ci/test-integration.sh` owns the shared PostgreSQL that the
dbtest suites use and removes it on exit, Ctrl-C or TERM. Every container a Go
test starts itself is removed in `t.Cleanup`. Image tags come from
`docs/pins.md`: the script exports the `postgres` row as `NAVA_TEST_PG_IMAGE`.

## Risks and the layer that covers them

| Risk (by impact) | Layer |
| --- | --- |
| A failing integration test does not fail the PR | Canary `TestCICanary` (`NAVA_CI_CANARY=fail`). Under `CI`, the health integration test fails instead of skipping when Docker is missing |
| A forbidden or unpinned module or image gets in | `check-pins` (fixture tests in `tools/ci/checkpins`) |
| Taskfile and Makefile drift apart, so local runs differ from CI | `check-taskfile` (drift fixture in `tools/ci/checktaskfile`) |
| A security-path change goes without the D-P10 review | `review-paths` prints the summary lines on every PR (fixture: a PR touching `db/`) |
| A shallow clone passes `check-openapi` with nothing to compare against | `check-openapi` fails when `BASE` is missing |
| nava-core goes ready without PostgreSQL | compose healthcheck on `/readyz`, plus the `dependencies` case (200 with Keycloak stopped) |

## Deliberately not covered

- **govulncheck on `temporal-server`** (`make vulncheck-temporal-server`). That module has reachable advisories (Go 1.25.8 stdlib, grpc v1.79.3, x/net and others). TICK-111 fixes them and makes the check gating.
- **testcontainers-go.** Approved, not adopted (decision log, 2026-10-19). The integration tests start a pinned container through the docker CLI, the way T03 and T04 built them.

## Run it

```sh
make tools        # once
make ci           # all four stages
make ci-lint      # or one stage
NAVA_CI_CANARY=fail make test-integration ARGS="-run TestCICanary"   # canary, must fail
```
