---
title: "Local stack"
section: operations
status: delivered
sources: [Taskfile.yml, compose.yaml, deploy/compose/README.md, deploy/compose/compose.internal.yaml, make/temporal-server.mk, .env.example, docs/acceptance/guide/tools/seed.sh, docs/acceptance/guide/troubleshooting.html]
tests: [TestComposePublishesNoAdminPort, TestComposePublishesOnlyAllowedUIs, TestConnectionBudgetFits]
---

The whole platform runs from `compose.yaml` on one machine with Docker. Every image is pinned (`docs/pins.md`). Passwords in `compose.yaml` are dev-only defaults so a fresh clone needs no setup; override them in `.env` (git-ignored, start from `.env.example`). Never reuse them anywhere else.

## Start

From the repository root:

```sh
cp .env.example .env              # once; optional, every value has a default
task stack:up                      # make dev-ca, then docker compose --profile operator up -d --build
docker compose ps                  # wait until every long-running service says healthy
sh docs/acceptance/guide/tools/seed.sh   # two demo tenants and one group sync (idempotent)
```

- `make dev-ca` writes the dev certificate authority and the Temporal mTLS certificates to `deploy/compose/certs/` (git-ignored). It keeps existing certificates; `FORCE=1 make dev-ca` reissues them.
- `task stack:up` starts the `operator` profile too: the Temporal UI, OpenObserve and Prometheus. Plain `docker compose up -d` leaves those three out.
- First boot takes a few minutes: Keycloak imports its realm, and the one-shot jobs (`db-roles`, `migrate`, `temporal-schema`) must finish before the services that depend on them start. `docker compose up --wait` blocks until everything is healthy.
- To keep every trace instead of the default 10 %, start with `OTEL_TAIL_SAMPLE_PERCENT=100 task stack:up`. Errors and requests slower than 500 ms are always kept.

## Stop and reset

| Goal | Command |
|---|---|
| Stop, keep data | `docker compose --profile operator stop` |
| Stop and delete every volume (tenants, users, workflows, telemetry) | `task stack:down` |
| Full reset for the acceptance guide | `sh docs/acceptance/guide/tools/tunnels.sh down`, `docker compose down -v`, `docker compose up --wait`, then `seed.sh` and `tunnels.sh up` again |
| Re-seed only | `sh docs/acceptance/guide/tools/seed.sh` |

## What runs

| Service | Role | Networks | Reaches the host |
|---|---|---|---|
| `postgres` | The one PostgreSQL 16 for the platform, Keycloak and Temporal | `internal`, `temporal-backplane` | No. The superuser logs in only over the Unix socket |
| `db-roles`, `migrate` | One-shot: roles and grants from `db/grants.yaml`, then goose migrations | none / `internal` | No |
| `pgbouncer` | Transaction-mode pooler in front of the `nava` database | `internal` | No |
| `keycloak` | Identity provider, realm `nava` | `default`, `internal` | `127.0.0.1:18081` (`KEYCLOAK_PORT`) |
| `temporal-schema`, `temporal-server`, `temporal` | Schema job, the mTLS Temporal server, and the gateway that forwards 7233 only | `temporal-backplane`; the gateway also `internal` and `operator` | No |
| `temporal-ui` | Temporal web UI, profile `operator` | `operator` | No: [tunnel](listeners.md) to `127.0.0.1:18233` |
| `otel-collector` | Receives OTLP traces and logs, tail-samples, forwards to OpenObserve | `internal` | No |
| `openobserve` | Trace and log search, profile `operator` | `internal`, `operator` | UI on `127.0.0.1:5080` (`OPENOBSERVE_PORT`) |
| `prometheus` | Scrapes every admin `/metrics`, evaluates alert rules, profile `operator` | `internal`, `operator` | UI on `127.0.0.1:9091` (`PROMETHEUS_PORT`) |
| `nava-core` | The public API | `default`, `internal` | `127.0.0.1:18080` (`NAVA_CORE_PORT`) |
| `nava-worker`, `nava-relay`, `nava-purge`, `nava-groupsync` | Workflow worker, outbox relay, retention and seal checks, Keycloak group sync | `internal` | No |
| `nava-onboard` | Operator onboarding API on `:8081` | `internal` | No: [tunnel or `exec`](listeners.md) |

Every published port binds `127.0.0.1` only. Each component has its own page in the [system section](../system/index.md).

## Variants

- **Ports taken.** Set `NAVA_CORE_PORT`, `KEYCLOAK_PORT` and `KEYCLOAK_PUBLIC_URL=http://localhost:<KEYCLOAK_PORT>` in `.env` together. A token's issuer must match `KEYCLOAK_PUBLIC_URL`, or every call answers 401.
- **No egress.** `docker compose -f compose.yaml -f deploy/compose/compose.internal.yaml up -d` makes every network internal. Published ports stop working; use `docker compose exec`.
- **A second stack on the same host** (`COMPOSE_PROJECT_NAME`, `NAVA_IMAGE_TAG` and the port variables) is what the isolated CI runs use; see `docs/quality/ci-pipeline.md`.
- **Group sync on demand:** `docker compose run --rm --no-deps nava-groupsync --once` runs one cycle instead of waiting for `NAVA_GROUPSYNC_INTERVAL` (default 5 minutes).

## Check that it works

- `task stack:test` runs every case under `deploy/compose/test/cases/` against the running stack. One case: `sh deploy/compose/test/stack-test.sh <name>`. Start the stack with `OTEL_TAIL_SAMPLE_PERCENT=100` for the `trace` case.
- `task demo:outbox` writes a record of the seeded `tenant-a` through the write pipeline and prints the workflow id it started. Open it in the Temporal UI (namespace `nava`) through `make operator-tunnel`.
- The acceptance guide walks each delivered story with Bruno: `docs/acceptance/guide/index.html`.

## Git Bash on Windows

MSYS rewrites a leading-slash container path in a `docker` argument. The repository's scripts and make targets set `MSYS_NO_PATHCONV=1` themselves. A `docker run`, `create` or `exec` you type with a container path must set it too.
