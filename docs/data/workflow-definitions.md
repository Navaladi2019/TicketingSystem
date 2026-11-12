# Workflow definitions, workflow_cause and the waiting state (P3-40)

TICK-709, S11b (TICK-337). The shapes are the S11b interface contract (D-3-29,
`docs/delivery/decisions/2026-11-11-d-3-29-s11b-interface-contract.md`).
Migrations: `db/migrations/20261112123300` to `20261112123400`.

## Model

```mermaid
erDiagram
    tenants ||--o{ workflow_definitions : owns
    entity_definitions ||--o{ workflow_definitions : "entity_definition_id"
    principals |o--o{ workflow_definitions : "created_by (NULL = seeded)"
    workflow_definitions ||--|{ workflow_definition_versions : "definition_id"
    entity_definitions ||--o{ workflow_definition_versions : "trigger_entity_definition_id"
    principals |o--o{ workflow_definition_versions : "published_by (NULL = seeded)"
    workflow_definitions ||--o{ cascade_start : "definition_id (P3-40 FK)"
    records ||--o{ workflow_ref : "record_id"
    tenants ||--o{ workflow_cause : owns

    workflow_definitions {
        uuid tenant_id PK
        uuid id PK
        text key "UNIQUE (tenant_id, key)"
        uuid entity_definition_id FK
        int current_version "nava_app UPDATE"
        uuid created_by FK
        timestamptz updated_at "nava_app UPDATE"
    }
    workflow_definition_versions {
        uuid tenant_id PK
        uuid definition_id PK
        int version PK
        uuid id "UNIQUE (tenant_id, id)"
        json body "exact bytes"
        bytea body_sha256 "CHECK = sha256(body)"
        jsonb step_ids "key -> uuid, signal:name -> uuid"
        text trigger_on "record.created | record.updated"
        uuid trigger_entity_definition_id FK
        uuid published_by FK
    }
    workflow_cause {
        uuid tenant_id PK
        uuid outbox_id PK
        uuid ref_outbox_id
        uuid causation_origin_id
        smallint causation_depth "CHECK >= 0"
    }
    workflow_ref {
        uuid tenant_id PK
        uuid outbox_id PK
        uuid record_id
        timestamptz closed_at "nava_worker UPDATE"
        uuid record_changed_signal_id "new; nava_worker UPDATE"
    }
```

Every key, index and foreign key leads with `tenant_id`. All three new tables
are `ENABLE` + `FORCE` RLS, with one canonical-predicate policy per command.

| Role | workflow_definitions | workflow_definition_versions | workflow_cause | workflow_ref (new) |
| --- | --- | --- | --- | --- |
| `nava_app` | SELECT, INSERT, UPDATE (`current_version`, `updated_at`) | SELECT, INSERT | SELECT, INSERT | none |
| `nava_worker` | SELECT | SELECT | SELECT, INSERT | UPDATE `record_changed_signal_id` (plus the existing `closed_at`) |
| others | none | none | none | unchanged |

No role has DELETE on any of the three tables, and no role has UPDATE on a version or a cause.

## Decisions beyond the contract

- **Nullable `created_by` / `published_by`.** This follows `entity_definition_versions.published_by` (P3-02). A NULL is a seeded definition (the demos, and future templates). A set value is held to its own tenant by the composite FK. P3-41's publish route always sets it.
- **Audit.** Both definition tables are `audited` under a new subject type, `workflow_definition` (`20261112123310`/`0123320`), and P3-41's publish writes the config entry. `audit_commit_check` is attached to both by P3-41b (`20261112180000`); a fixture that needs a raw row writes its own audit entry or turns the trigger off for its own seed transaction. `workflow_cause` is exempt: it copies the causation of an outbox row whose write is already audited.
- **No FK from `workflow_cause` to `outbox` or `workflow_ref`.** Both of those tables are purged, and a NO ACTION key would make the purge fail. `workflow_cause` has no purge yet. It should be purged with `workflow_ref` (30 days after close); that task is open.
- **`step_ids` is `jsonb`, `CHECK (jsonb_typeof = 'object')`.** It is read whole and never hashed. Validating its contents (UUIDv4 values) stays in `AssignStepIDs` (P3-41).

## Evidence

Run with `sh scratch/p340-workflow-definitions/run.sh` (raw output is in `scratch/p340-workflow-definitions/out.txt`).

- **Hardware:** postgres:16.10 in Docker Desktop (16 vCPU, 24.5 GiB VM) on an Intel i7-1260P with 39.7 GB RAM; warm cache, no parallel workers.
- **Data:** 2,000 tenants and 20,000 definitions (10 per tenant), with 3 versions each (60,000 version rows). There are 2,000,000 `workflow_ref` rows (1,000 per tenant over 200 records). 5% of them are open and 50,000 wait on `record_changed`.

| Query (per record write, P3-42) | Without the index | With the index |
| --- | --- | --- |
| (1) Trigger lookup: current versions matching `(tenant, entity, op)` | 0.308 ms, 52 buffers. The plan probes every definition of the tenant through `workflow_definitions_key_key`, then probes the version PK for each (10 loops), so it grows with definitions per tenant. | 0.139 ms, 16 buffers. An index-only scan on `workflow_definition_versions_trigger` (0 heap fetches), then one PK probe per matching version. |
| (2) Open refs on a record that wait on `record_changed` | 0.110 ms, 8 buffers. `workflow_ref_record` plus a filter over every ref of the record (5 here). | 0.104 ms, 7 buffers. `workflow_ref_waiting`, no filter. |

Storage at this shape:

- `workflow_definition_versions_trigger` is 6.5 MiB (the table is 52 MiB and its PK 4.5 MiB). It costs one extra index entry per publish, and publishes are rare.
- `workflow_ref_waiting` is 856 KiB for 50,000 waiting rows. `workflow_ref` itself is 223 MiB, its PK 129 MiB, and `workflow_ref_record` 63 MiB.

**Write cost of `workflow_ref_waiting`.** `closed_at` and `record_changed_signal_id` are in its predicate. Any UPDATE that changes either one is therefore no longer HOT. That covers every `CloseWorkflowRef` and every set or clear of the waiting state, and each such update writes new entries in `workflow_ref_pkey` and `workflow_ref_record` as well as the partial index. At this shape, (2) gains 0.006 ms per write. The index pays off only for a record with many refs inside the 30-day purge window. The contract specifies it, so it ships. Dropping it is a single `DROP INDEX CONCURRENTLY` if P3-42's load shows the HOT loss matters more.

## FORCE RLS makes a bare `VALIDATE CONSTRAINT` of a foreign key vacuous

This was found while proving the `cascade_start` FK pair, on postgres:16.10:

- When the caller owns both tables, PostgreSQL validates a FK with one SQL anti-join (`RI_Initial_Check`).
- That query is subject to the **referencing** table's RLS, and FORCE applies it to the owner.
- `nava_migrator` has no policy on any tenant table, so the scan sees no rows, and the key is marked valid over orphans.

| FORCE RLS on | Orphan row | VALIDATE result |
| --- | --- | --- |
| Referencing table | yes | **passes, `convalidated = true`** |
| Referenced table only | none | **fails**: every row looks orphaned (a false positive) |
| Both tables, with a temporary `TO nava_migrator USING (true)` SELECT policy on each | yes / none | fails / passes (correct) |
| `CHECK` constraint on a FORCE RLS table | violating row | fails (correct: a heap scan) |

`20261112123340` therefore creates those two temporary policies, validates and drops them in one transaction.
`TestP340CascadeStartDefinitionKeyOverExistingClaims` pins this: an orphan claim must stop the migration.

Two earlier FK validations have the same shape and were probably validated vacuously:

- `group_members_group_live_fkey` (`20261104013500`)
- `records_assignment_group_fkey` (`20261104043100`)

They are reported to the Technical Lead (TICK-337) and are not changed here.
