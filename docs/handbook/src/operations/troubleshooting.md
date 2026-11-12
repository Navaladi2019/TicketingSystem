---
title: "Troubleshooting"
section: operations
status: delivered
sources: [docs/acceptance/guide/troubleshooting.html, deploy/compose/README.md, compose.yaml, docs/operator/onboarding.md, deploy/compose/prometheus/rules/audit.yml]
evidence: [docs/acceptance/guide/troubleshooting.html]
---

Where to look first, by symptom. The acceptance guide's [troubleshooting page](../../../acceptance/guide/troubleshooting.html) covers the Bruno walk-through in more detail.

## First three looks

1. `docker compose ps`: is every long-running service `healthy`? A one-shot job (`db-roles`, `migrate`, `temporal-schema`) that exited non-zero blocks everything after it.
2. `docker compose logs --since 5m <service>`: the reason is usually on one line. Every error response's `instance` is `urn:nava:request:<id>`; search the logs for that id.
3. `/healthz` of the service, through a [tunnel or `exec`](listeners.md): it names the failing dependency.

## The stack does not start

| Symptom | Cause | Fix |
|---|---|---|
| `ca.pem: no such file` | No dev certificates yet | `make dev-ca` (`task stack:up` runs it) |
| `all predefined address pools have been fully subnetted` | Old Docker networks | `docker network prune` (removes unused networks only) |
| A port is already in use | Another stack or program on 18080, 18081, 5080 or 9091 | Set `NAVA_CORE_PORT`, `KEYCLOAK_PORT` with `KEYCLOAK_PUBLIC_URL`, `OPENOBSERVE_PORT` or `PROMETHEUS_PORT` in `.env` |
| Startup takes minutes | Keycloak imports its realm on first boot; one-shot jobs run first | Normal on a busy machine. `docker compose up --wait` keeps waiting |

## Calls fail

| Symptom | Cause | Fix |
|---|---|---|
| 401 `invalid_token` on a fresh token | The token's issuer is not `KEYCLOAK_PUBLIC_URL/realms/nava`: Keycloak was reached through another host or port | Use the shipped URLs, or change `KEYCLOAK_PORT` and `KEYCLOAK_PUBLIC_URL` together and restart |
| 401 `invalid_token` after a while | Access tokens live 5 minutes | Sign in again |
| 401 `tenant_unresolved` | The tenants are not seeded, or no group sync has run since | `sh docs/acceptance/guide/tools/seed.sh` |
| 403 `forbidden` on `/api/v1/operator/tenants` | A tenant token was sent | Use the operator sign-in (client `nava-operator-cli`) |
| Connection refused on `18082`, `19090` to `19095` or `18233` | The tunnels are not running; they stop when the stack is recreated | `sh docs/acceptance/guide/tools/tunnels.sh up` |
| `18233` only | The Temporal UI is not started | Start with the `operator` profile (`task stack:up`), then the tunnels again |
| 502 from Keycloak | Keycloak is still importing its realm | Wait for `docker compose ps keycloak` to say healthy |
| Every token refused (`400 unauthorized_client`) after PostgreSQL restarted | Keycloak's database connections died with it | Keycloak validates its pool on every borrow, so this should recover on its own; nava-core's `/healthz` keycloak check shows `fail` until it does |

## Something is slow or stuck

| Symptom | Where to look |
|---|---|
| One request is slow | Its trace in OpenObserve, by `request_id` ([Traces, logs and metrics](observability.md)) |
| A workflow did not start after a write | `nava-relay` logs and its admin `/metrics`; the workflow in the Temporal UI (namespace `nava`) |
| Audit entries are not sealed | Prometheus Alerts: `AuditSealLag`, `AuditDeadLineage`, `AuditDeadLineageCheckDown` |
| Users or groups changed in Keycloak do not show | Group sync runs every 5 minutes; run one cycle with `docker compose run --rm --no-deps nava-groupsync --once` |
| A tenant is stuck in provisioning | [docs/operator/onboarding.md](../../../operator/onboarding.md), "Steps and recovery" |

## Start over

```sh
sh docs/acceptance/guide/tools/tunnels.sh down
docker compose down -v          # deletes all data: tenants, users, workflows, telemetry
docker compose up --wait
sh docs/acceptance/guide/tools/seed.sh
sh docs/acceptance/guide/tools/tunnels.sh up
```
