# Platform tables under forced RLS (T09)

Scope: TICK-64. ADR-001 I1, D28 and A.3, PLAN-001 T09. Migration
`db/migrations/00002_platform_tables.sql`, grants in `db/grants.yaml`.
PostgreSQL 16.10.

## Tables

```mermaid
erDiagram
    tenants ||--o{ tenant_identity_binding : "binds (issuer, org_id)"
    records ||--o{ workflow_ref : "(tenant_id, record_id)"
    outbox ||--o| workflow_ref : "outbox_id (start row)"

    tenants {
        uuid id PK
        text state "provisioning | active"
        bytea onboarding_request_hash "T21 idempotency"
        timestamptz created_at
    }
    tenant_identity_binding {
        uuid tenant_id PK,FK
        text issuer PK "UNIQUE (issuer, org_id)"
        text org_id PK
        text_array idp_aliases
    }
    records {
        uuid tenant_id PK
        uuid id PK
        bigint version "PLAN-002 adds the rest"
    }
    outbox {
        uuid tenant_id PK
        uuid id PK "UUIDv7"
        text dedupe_key "UNIQUE (tenant_id, dedupe_key)"
        text target_key
        bigint target_seq
        smallint causation_depth
        jsonb payload "references only"
        timestamptz available_at
        timestamptz dispatched_at
        timestamptz dead_at
    }
    workflow_ref {
        uuid tenant_id PK,FK
        uuid outbox_id PK
        uuid record_id FK
        text kind
        timestamptz created_at
        timestamptz closed_at
    }
    idempotency_keys {
        uuid tenant_id PK
        uuid principal_id PK
        text key PK
        bytea request_hash
        text state "in_flight | completed"
        smallint response_status
        bytea response_body "byte-identical replay"
        timestamptz expires_at
    }
```

Every table is `ENABLE` + `FORCE ROW LEVEL SECURITY`. Every primary key,
unique key and foreign key leads with `tenant_id`. The exception is
`UNIQUE (issuer, org_id)`, which is global on purpose: one organization, one
tenant.

## Policies

Tenant policies test `tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid`.
With no tenant set, or an empty one, they match no row.

| Table | Tenant policies | Allow-listed (do not test `app.tenant_id`) |
| --- | --- | --- |
| `tenants` | `tenants_tenant` (SELECT, app), `tenants_onboard_select`, `tenants_onboard_insert` (provisioning only), `tenants_activate` (UPDATE provisioning → active, own tenant) (onboard) | none |
| `tenant_identity_binding` | none | `binding_lookup` (SELECT TO nava_app, nava_groupsync), `binding_onboard` (INSERT TO nava_onboard, provisioning tenants visible to it) |
| `records` | `records_tenant` (app, worker) | none |
| `outbox` | `outbox_tenant` (app, worker) | `outbox_relay_select`, `outbox_relay_update` (pending rows only), `outbox_purge_select`, `outbox_purge` |
| `workflow_ref` | `workflow_ref_tenant` (app, worker) | `workflow_ref_purge_select`, `workflow_ref_purge` |
| `idempotency_keys` | `idempotency_tenant` (app) | `idempotency_purge_select`, `idempotency_purge` |

Ten allow-listed policies, each `FOR` one command and `TO` named roles.
`TestT09PolicyAllowList` fails if one is added or removed. `binding_onboard`
reads `tenants` through `tenants_onboard_select`, so inside `InTenantTx(X)` it
sees only X. That is why a binding for another provisioning tenant Y is
refused.

Column grants narrow what the allow-list opens. `nava_relay` cannot read
`dedupe_key`, `last_error`, `records` or `workflow_ref`, and writes only the
dispatch columns. `nava_purge` reads only key and expiry columns.

## Evidence

Run with `sh scratch/t09-platform-tables/run.sh` (output in `output.txt`).
PostgreSQL 16.10 on Docker Desktop, 16 CPUs, 20.8 GB, `shared_buffers=1GB`.
The data has 200 tenants, and tenant 1 holds 40% of every table. Row counts:
`records` 1.0 M, `outbox` 2.05 M (50k pending), `workflow_ref` 1.0 M,
`idempotency_keys` 1.0 M. Plans run as the runtime role with RLS on.

| Query (role) | Plan | Buffers | Time |
| --- | --- | --- | --- |
| Binding lookup (app, no tenant) | Seq Scan, 200 rows | 4 | 0.114 ms |
| Idempotency key (app, big tenant) | Index Scan `idempotency_keys_pkey` | 4 | 0.100 ms |
| Same, RLS as the only tenant filter | Index Scan `idempotency_keys_pkey`, RLS predicate in Index Cond | 4 | 0.035 ms |
| Dedupe conflict (app) | Index Scan `outbox_dedupe_key` | 4 | 0.086 ms |
| Record row lock (app) | Index Scan `records_pkey` | 4 | 0.175 ms |
| Open workflow_ref (worker, big / small tenant) | Index Scan `workflow_ref_record` | 4 / 4 | 0.068 / 0.069 ms |
| Relay claim, oldest 100 pending (relay) | Index Scan `outbox_pending_available` | 12 | 0.136 ms |
| Relay FIFO head of a target (relay) | Index Scan `outbox_pending_target` | 3 | 0.067 ms |

Every tenant-scoped read is an index scan whose Index Cond leads with
`tenant_id`, including the RLS-only form. The binding lookup seq-scans
200 rows (4 pages), which is cheaper than an index at that size.
`tenant_identity_binding_issuer_org_key` is there for when the table grows.

### Index cost

| Index | Size | Why it exists |
| --- | --- | --- |
| `outbox_pkey` | 131 MB | PK |
| `outbox_dedupe_key` | 152 MB | producer `ON CONFLICT DO NOTHING` (A.3) |
| `outbox_pending_available` | 2.4 MB | relay claim; partial, pending rows only |
| `outbox_pending_target` | 5.5 MB | per-target FIFO; partial |
| `workflow_ref_pkey` | 64 MB | PK |
| `workflow_ref_record` | 95 MB | open-workflow lookup and the FK's index |
| `records_pkey`, `idempotency_keys_pkey` | 64 MB, 76 MB | PK |

Write cost as `nava_app` in one tenant transaction, with all indexes and RLS
checks: 10,000 `outbox` rows took 166 ms (16.6 µs/row, 4 indexes), and
10,000 `idempotency_keys` rows took 64 ms (6.4 µs/row). The two partial
indexes are under 2% of the outbox heap (471 MB), because they hold only the
pending rows.

## Not done here (owners)

- The CI-1 catalogue check across all future tables (RLS, FORCE, allow-list):
  T10.
- The identity tables (`principals`, `groups`, …) and their `nava_groupsync`
  grants: T16.
- `records` columns beyond `version`: PLAN-002.
