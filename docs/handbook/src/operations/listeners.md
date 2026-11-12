---
title: "Unpublished listeners"
section: operations
status: delivered
sources: [compose.yaml, make/temporal-server.mk, docs/acceptance/guide/tools/tunnels.sh, docs/operator/onboarding.md, docs/architecture/nava-core-skeleton.md, docs/architecture/observability-notes.md]
tests: [TestComposePublishesNoAdminPort, TestComposePublishesOnlyAllowedUIs]
---

Only `nava-core` (`127.0.0.1:18080`), Keycloak (`127.0.0.1:18081`), OpenObserve (`127.0.0.1:5080`) and Prometheus (`127.0.0.1:9091`) publish a port. Everything below listens only inside the stack's networks, on purpose. Reach it with a local tunnel or from inside a container.

## What is unpublished

| Listener | Serves | Why it stays off the host |
|---|---|---|
| Admin `:9090` on `nava-core`, `nava-worker`, `nava-relay`, `nava-purge`, `nava-onboard`, `nava-groupsync` (`NAVA_ADMIN_ADDR`) | `/metrics`, `/healthz`, `/readyz` | `/metrics` has no authentication of its own. Anything that reaches `:9090` can read it. `:8080/metrics` on nava-core answers 404 |
| `nava-onboard:8081` | The operator onboarding API | The only holder of the onboarding secret and the `nava_onboard` database URL |
| `temporal-ui:8080` | The Temporal web UI, as `nava-operator` | It acts with the operator's certificate |
| `postgres:5432`, `pgbouncer:5432`, `temporal:7233` | Database, pooler, Temporal gateway | Platform traffic only |

## Tunnels

Each tunnel is a short-lived container of the `nava/temporal-server` image running its TCP `proxy`, bound to `127.0.0.1` on the host.

| Command | Forwards |
|---|---|
| `make operator-tunnel` | `127.0.0.1:18233` (`UI_PORT`) to `temporal-ui:8080`, until Ctrl+C. Needs the `operator` profile |
| `sh docs/acceptance/guide/tools/tunnels.sh up` | `127.0.0.1:18082` to `nava-onboard:8081`; `127.0.0.1:19090` to `nava-core:9090`; `19091` to `19095` to the admin listeners of `nava-onboard`, `nava-groupsync`, `nava-relay`, `nava-worker` and `nava-purge`, in that order; `127.0.0.1:18233` to the Temporal UI when it runs |
| `sh docs/acceptance/guide/tools/tunnels.sh status` / `down` | Lists or removes those forwards |

The tunnels stop when the stack is recreated: run `tunnels.sh up` again.

**Close them when you are done.** While a tunnel runs, any container on a non-internal network can reach its target through the host. For the Temporal UI that means acting as `nava-operator` (`TODO-006` in `make/temporal-server.mk`). Tunnels are for a local dev stack only.

## Without a tunnel

Run the request from inside the container. The services' images carry `wget`:

```sh
docker compose exec nava-core wget -qO- http://127.0.0.1:9090/readyz
docker compose exec nava-relay wget -qO- http://127.0.0.1:9090/metrics
```

The onboarding API is called the same way, with the operator token passed through the environment rather than the command line; the full request is in [docs/operator/onboarding.md](../../../operator/onboarding.md). Under the no-egress overlay (`deploy/compose/compose.internal.yaml`) this is the only way in.

## Health endpoints

| Endpoint | Meaning | Use it for |
|---|---|---|
| `/readyz` | Ready to serve. On nava-core only PostgreSQL gates it | Readiness checks, the compose health checks |
| `/healthz` | A report: `{status, checks}` per dependency (PostgreSQL, Keycloak, Temporal). 503 while PostgreSQL is down; a failing Keycloak or Temporal shows as `degraded` | Reading why a service is unhappy. **Not** a liveness probe: a restart policy on it turns a database outage into a restart loop |
| `/metrics` | Prometheus text format | Prometheus scrapes it over `internal` |
