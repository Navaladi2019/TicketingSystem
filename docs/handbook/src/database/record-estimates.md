---
title: "Record estimates (entity_record_estimates)"
section: database
status: delivered
tables: [entity_record_estimates]
adrs: [ADR-010-A2, ADR-001]
stories: [S31]
sources: [docs/data/entity-record-estimates.md, db/migrations/20261121163600_entity_record_estimates.sql, db/migrations/20261121163700_binding_lookup_worker.sql, db/grants.yaml, internal/estimates/estimates.go]
---

`entity_record_estimates` holds N̂_e: about how many live records of entity
type `e` a tenant has. The `under` list leaf uses it to pick its SQL shape:
K\* = clamp(0.8 × √(page × N̂_e), 1,000, 20,000) (ADR-010-A2 A2-9; see
[Core: record ACL](../system/core-acl.md#the-under-leaf-s31)). One row per
(tenant, entity type), keyed `(tenant_id, entity_definition_id)`, with
`live_records` and `refreshed_at`.

- **Who writes it.** Only the hourly refresh job in
  [nava-worker](../system/nava-worker.md) (TICK-1071), one tenant at a time,
  each inside that tenant's transaction (`tenancy.InTenantTx`). It runs
  two statements per tenant: one upsert of every type's count, then an update
  that sets types with no records left to 0. A row is overwritten, never
  deleted.
- **Who reads it.** `nava_app`, one primary-key lookup per list request that
  has a region term. The probe is what runs once per `under` leaf.
- **A missing or stale row costs time, never rows.** No row means K\* = 1,000
  and the probe runs. Every shape computes the same predicate (A2-6), so a
  wrong estimate can only make a list slower. `EntityRecordEstimatesStale`
  fires when the oldest refresh is over 3 hours old
  ([observability](../operations/observability.md)).
- **Why a table.** No per-tenant, per-type count existed: `record_counters`
  is the numbering sequence and is purged per period, and PostgreSQL's
  extended statistics would show one tenant other tenants' volumes.

## Access

ENABLE and FORCE RLS, with the canonical `app.tenant_id` policy, one command
per policy. An upsert of another tenant's key fails with 42501, because the
`INSERT`'s `WITH CHECK` runs before the conflict arbiter.

| Role | Grant |
|---|---|
| `nava_app` | `SELECT` |
| `nava_worker` | `SELECT`, `INSERT`, `UPDATE (live_records, refreshed_at)` |
| other roles | none |

The table is exempt from audit: it is derived data and holds no fact of its
own.

**Listing tenants without a cross-tenant read (D-11A2-2).** The job reads
`SELECT DISTINCT tenant_id FROM tenant_identity_binding` through
`binding_lookup`, the allow-listed policy tenant resolution and group sync
already use. Migration `20261121163700_binding_lookup_worker` adds
`nava_worker` to that policy, and `db/grants.yaml` grants the worker
`SELECT (tenant_id)` only, so the issuer, org id and aliases stay out of its
reach. Changing the policy's roles needs CTO sign-off (PLAN-001 D-P10).

## Cost

Calculated, not measured (the change landed during a bench hold): about 80
bytes of heap and 56 bytes of index per row, so 400 tenants with 50 types each
is about 2.7 MB. One row version per type per hour. No backfill: the job fills
the table on its first run.

## Diagram

<!-- gen:er area=record-estimates -->

## Tables

<!-- gen:tables area=record-estimates -->
