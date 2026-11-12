---
title: "nava-worker"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-worker/main.go, db/grants.yaml]
adrs: [ADR-001]
packages: [cmd/nava-worker, internal/workerguard]
invariants: [I6]
tests: [TestCI6_WorkerCrossTenantSignalExternalDeniedWithAlert, TestComposeWorkerPublishesNothing, TestWorkerServiceReadyAndAdminListener]
---

The Temporal worker: namespace `nava`, task queue `nava-tq`. Its client and worker come only from `internal/workerguard`, and activity code gets no Temporal client.

| | |
|---|---|
| Network | `internal` |
| Listeners | `:9090` admin only; no public listener |
| Host port | none (tunnel 19094) |
| DB role | `nava_worker` through PgBouncer |
| Talks to | temporal (poll `nava-tq`, with the `nava-worker` certificate); pgbouncer; otel-collector |
| Introduced by | PLAN-001 T31b (TICK-159) |

`/readyz` follows both PostgreSQL and Temporal.

## Boundary it enforces

**I6.** A workflow cannot reach another tenant's run: `TestCI6_WorkerCrossTenantSignalExternalDeniedWithAlert`.

[Back to the system map](index.md)
