# ACL and share tables (P3-30, TICK-615)

PLAN-003 S10, D-3-2; ADR-001 I5 (one authorization layer, inside the query).
Migrations: `db/migrations/20261108124249_acl_tables.sql`, and
`db/migrations/20261108184658_record_shares_typed_principals.sql` (P3-30a,
TICK-625: typed principal columns). Tests:
`internal/db/dbtest/acl_tables_test.go`,
`internal/db/dbtest/record_shares_typed_test.go` and the isolation registry
(`internal/tenancy/isolation/acl_paths.go`).

## Model

```mermaid
erDiagram
    entity_definitions ||--o{ acl_versions : "ACL per publish"
    records ||--o{ record_shares : "shared with"
    principals |o--o{ acl_versions : "published_by"
    principals |o--o{ record_shares : "granted_by"
    principals |o--o{ record_shares : "user_id"
    groups |o--o{ record_shares : "group_id"
    acl_versions {
        uuid tenant_id PK
        uuid entity_definition_id PK
        int version PK
        jsonb rules "CHECK acl_rules_valid (TICK-629)"
        jsonb masks "CHECK acl_masks_valid (TICK-629)"
        uuid published_by
        timestamptz published_at
    }
    record_shares {
        uuid tenant_id PK
        uuid record_id PK
        text principal_kind PK
        uuid principal_id PK "generated: COALESCE(user_id, group_id)"
        uuid user_id FK
        uuid group_id FK
        uuid granted_by
        timestamptz granted_at
    }
```

Every foreign key is composite, `(tenant_id, …)`, so it holds the tenant even
though PostgreSQL checks it as the owner, past RLS.

`rules` and `masks` have the shape `internal/authz` decodes, enforced by
`acl_versions_rules_check` and `acl_versions_masks_check` (TICK-629; the
accepted shape and its cost are in
[`acl-versions-shape-check.md`](acl-versions-shape-check.md)).

A share names its principal in `user_id` (FK to `principals`) or `group_id`
(FK to `groups`). `record_shares_principal_check` holds exactly one of them
set, the one `principal_kind` names. So a group id written as a user, a user
id written as a group, an id the tenant does not have, and another tenant's
user or group are all refused (23503 for the foreign keys, 23514 for the
check). `principal_id` is a stored generated column,
`COALESCE(user_id, group_id)`, that no writer can set (428C9). It keeps the
primary key and the list-join index on one principal column. P3-30 had a
single unchecked `principal_id`; P3-30a (TICK-625) replaced it while the table
was still empty, which closed AR-S10-01.

No index backs `record_shares_user_fkey` or `record_shares_group_fkey` from
the parent side. Only a delete of a `principals` or `groups` row would use
one, and neither is ever deleted: both are tombstoned (ADR-004-E5).

## Access and grants

| Role | acl_versions | record_shares |
|---|---|---|
| nava_app, nava_worker | SELECT, INSERT | SELECT, INSERT |
| every other role | nothing | nothing |

Both tables are immutable: publishing an ACL inserts version n + 1; an unshare
is not in S10. `ENABLE` + `FORCE` RLS, one policy per command
(`_select`, `_insert`) testing `app.tenant_id`.

## Access patterns and indexes

| Query | Index | Plan |
|---|---|---|
| Newest ACL of a type (P3-31) | `acl_versions_pkey` | backward PK scan, `LIMIT 1` |
| Shares of one record (record view) | `record_shares_pkey` prefix `(tenant_id, record_id)` | PK range scan |
| Records shared with the caller's principal ids (list, P3-34) | `record_shares_principal (tenant_id, principal_id)` | bitmap scan, then `records_pkey` probes |
| `record_shares` → `records` FK | `record_shares_pkey` prefix | — |

## Evidence: the list-join index

Setup: PostgreSQL 16.10 (`postgres:16.10`, the compose image) in Docker 29.9.2
on a 12th Gen Intel Core i7-1260P laptop (16 threads, Docker VM 19.4 GiB),
`max_parallel_workers_per_gather = 0`, superuser (no RLS; this compares index
shapes only). 1,000,000 records over 50 tenants, one holding 400,000;
2,000,000 shares, 2,000 principal ids per tenant. Query: the big tenant's
newest 50 records shared with 4 principal ids (one user, three groups), which
reach 1,573 records. The script ran from `scratch/acl-tables-bench/`, which
TICK-625 removed; the figures below are the record. P3-30a keeps the same
`(tenant_id, principal_id)` key, so they still apply.

| Index | Size | Execution (2 runs, warm) | Buffers | 20,000-share insert |
|---|---|---|---|---|
| `(tenant_id, principal_id)` | 18 MB | 12.0, 13.0 ms | 7,973 | 304 ms |
| `(tenant_id, principal_id, record_id)` | 129 MB | 12.4, 16.5 ms | 6,506 | 298 ms |

The covering index makes the share lookup index-only (30 buffers instead of
1,497), but 6,476 of the buffers in both plans are the `records_pkey` probes,
so the total time does not improve. B-tree deduplication collapses the
two-column index's repeated `(tenant, principal)` keys, which is why it is a
seventh the size. Kept: the two-column index, as PLAN-003 states it. Revisit
in P3-36 if the list query's shape changes.
