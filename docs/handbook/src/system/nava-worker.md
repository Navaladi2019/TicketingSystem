---
title: "nava-worker"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-worker/main.go, cmd/nava-worker/interp_ports.go, db/grants.yaml, docs/budgets.md, deploy/compose/pgbouncer/pgbouncer.ini]
adrs: [ADR-001]
packages: [cmd/nava-worker, internal/workerguard]
invariants: [I6]
tests: [TestWorkerPoolsFitPgBouncer, TestProductionBuildWiresTheDatabasePorts, TestCI6_WorkerCrossTenantSignalExternalDeniedWithAlert, TestComposeWorkerPublishesNothing, TestWorkerServiceReadyAndAdminListener]
---

The Temporal worker: namespace `nava`, task queue `nava-tq`. Its client and worker come only from `internal/workerguard`, and activity code gets no Temporal client.

It runs two workflows: [`audit.seal`](../workflows/audit-seal.md), and since S11b the [workflow interpreter](../workflows/nava-interpreter.md) on its database ports. The interpreter's record writes go through the records pipeline as `source=workflow`. The pipeline has its own pool, because it opens a second connection while the write's transaction is still open. Both pool widths are fixed in code: main pool 10, pipeline pool 4, plus 1 for the readiness probe. That makes 15, which equals the `nava_worker` PgBouncer pool (`TestWorkerPoolsFitPgBouncer`, see [benchmarks](../benchmarks/index.md)).

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
