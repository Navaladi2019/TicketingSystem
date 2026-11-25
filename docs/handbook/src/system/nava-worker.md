---
title: "nava-worker"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-worker/main.go, internal/estimates/estimates.go, deploy/compose/prometheus/rules/authz.yml, docs/data/entity-record-estimates.md, cmd/nava-worker/interp_ports.go, db/grants.yaml, docs/budgets.md, deploy/compose/pgbouncer/pgbouncer.ini]
adrs: [ADR-001]
packages: [cmd/nava-worker, internal/workerguard, internal/estimates]
stories: [S31]
tables: [entity_record_estimates]
invariants: [I6]
tests: [TestEveryTenantGetsItsOwnTransaction, TestFailingTenantIsSkippedAndAgesTheGauge, TestRunDisabledReturnsAtOnce, TestIntervalFromEnv, TestWorkerPoolsFitPgBouncer, TestProductionBuildWiresTheDatabasePorts, TestCI6_WorkerCrossTenantSignalExternalDeniedWithAlert, TestComposeWorkerPublishesNothing, TestWorkerServiceReadyAndAdminListener]
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

## The estimates refresh job (S31)

Since S31 (TICK-1071) the worker also runs the `internal/estimates` ticker.
At start and then every `NAVA_ESTIMATES_REFRESH_INTERVAL` (default `1h`; `0`
turns it off) it lists the tenants from `tenant_identity_binding` (the
`tenant_id` column only) and, for each tenant, counts the live records of
each entity type and upserts them into `entity_record_estimates`; a type
with no records left is set to 0. Each
tenant's refresh runs in its own `tenancy.InTenantTx`, so it sees and writes
that tenant's rows only. A failing tenant is skipped, counted, and keeps its
old time. The `under` list leaf reads the estimate to pick its SQL shape
([Record estimates](../database/record-estimates.md)).

| Metric (no labels) | What it is |
|---|---|
| `authz_under_estimate_age_seconds` | Now minus the oldest per-tenant refresh time |
| `entity_record_estimates_refresh_failures_total` | Tenants whose refresh failed |

`EntityRecordEstimatesStale` fires when the age stays above 10,800 s
(3 hours) for 15 minutes. A stale estimate costs time only, never rows. With
the job turned off the alert fires by design.

## Boundary it enforces

**I6.** A workflow cannot reach another tenant's run: `TestCI6_WorkerCrossTenantSignalExternalDeniedWithAlert`.

[Back to the system map](index.md)
