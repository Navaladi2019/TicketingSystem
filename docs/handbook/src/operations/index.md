---
title: "Operations"
section: operations
status: delivered
sources: [Taskfile.yml, compose.yaml, deploy/compose/README.md, docs/operator/onboarding.md, cmd/nava/main.go]
---

How to run the platform on one machine and see what it is doing. Everything here is the local Docker Compose stack in `compose.yaml`. There is no production deployment on `main` yet.

| Page | What it covers |
|---|---|
| [Local stack](local-stack.md) | Start, stop, reset and seed the stack; what each service is and which ports reach the host |
| [Unpublished listeners](listeners.md) | Admin endpoints, the onboarding API and the Temporal UI publish no port. How to reach them through a tunnel or `docker compose exec` |
| [Traces, logs and metrics](observability.md) | OpenObserve and Prometheus: finding a slow or failed request, alert rules, health endpoints |
| [Troubleshooting](troubleshooting.md) | Where to look first when something breaks |

## Operator tools

- **Tenant onboarding** goes through `nava-onboard`, an internal API that is never published. The request, its validation and how to recover a half-provisioned tenant are in [docs/operator/onboarding.md](../../../operator/onboarding.md). The cursor keyring is in [docs/operator/cursor-keyring.md](../../../operator/cursor-keyring.md).
- **`nava audit verify <file>`** checks an audit export (`nava.audit.export/v1`, JSONL or CSV) with no database and no network. `task build` writes it to `bin/nava`. Its exit codes are a contract: 0 every line verified, 1 tampered (the first failing line is reported), 2 usage error, 3 an erasure entry is not sealed yet, 4 unverifiable, 5 nothing sealed in the window. The audit chain itself is on the [ADR-004 page](../adr/adr-004.md).

## Commands at a glance

| Command | What it does |
|---|---|
| `task stack:up` | `make dev-ca`, then `docker compose --profile operator up -d --build` |
| `task stack:down` | `docker compose --profile operator down -v`: stops the stack **and deletes its volumes** |
| `task stack:test` | Runs every stack case in `deploy/compose/test/cases` against a running stack |
| `task demo:outbox` | The S7 demo: one record written through the pipeline, one workflow started and signalled twice |
| `make operator-tunnel` | Forwards `127.0.0.1:18233` to the Temporal UI until interrupted |
| `sh docs/acceptance/guide/tools/tunnels.sh up` | Forwards for every unpublished listener (see [Unpublished listeners](listeners.md)) |
| `sh docs/acceptance/guide/tools/seed.sh` | Seeds the two demo tenants and runs one group sync. Idempotent |
