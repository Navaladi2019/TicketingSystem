# Audit log tables (P5-10b, TICK-450)

This document covers the [ADR-004](../adr/ADR-004-audit-log.md) data layer, with [E1](../adr/ADR-004-E1-cascade-entries-and-diff-truncated.md), [ADR-001-A6.1](../adr/ADR-001-A6-audit-roles-and-non-record-starts.md) and PLAN-005 D-5-3.
- Migration: `db/migrations/20261102233600_audit_log.sql`.
- Grants: `db/grants.yaml`.
- Tests: `internal/db/dbtest/audit_log_test.go`.
- Subject-type registry: `internal/audit/subjects.go`.

```mermaid
erDiagram
    tenants ||--o{ audit_log : "tenant_id"
    tenants ||--o| audit_chain_head : "tenant_id (PK)"
    tenants ||--o{ record_write_keys : "tenant_id"
    entity_definitions ||--o{ audit_log : "(tenant_id, entity_definition_id)"
    principals ||--o{ audit_log : "(tenant_id, actor_principal_id)"
    records ||--o{ record_write_keys : "(tenant_id, record_id)"
    entity_definitions ||--o{ record_write_keys : "(tenant_id, definition_id)"
    record_write_keys }o..|| audit_log : "audit_entry_id (soft, no FK)"
    audit_log ||--|| audit_log_default : "PARTITION DEFAULT"
    audit_log {
        uuid tenant_id PK
        timestamptz occurred_at PK "partition key; R10: = now()"
        uuid id PK "UUIDv7 / UUIDv5 backfill"
        xid8 txid "R10: = pg_current_xact_id()"
        text domain "CHECK"
        text action "CHECK, E1 cascade actions"
        text subject_type "CHECK = subjects.go"
        uuid subject_id
        bigint version_after "R2 key, NULLS NOT DISTINCT"
        boolean diff_truncated "E1, in the envelope"
        json diff "never jsonb; <= 65,536 bytes"
        bytea diff_salt "16 bytes; NULL after erase"
        bigint seq "seal: worker only"
        timestamptz erased_at "erase: nava_erasure only"
    }
    audit_chain_head {
        uuid tenant_id PK
        uuid id
        bigint version
        bigint seq
        bytea entry_hash "genesis 32 zero bytes"
        uuid next_seal_outbox_id
        int seal_interval_s "10..300"
        timestamptz export_lease_until
    }
    record_write_keys {
        uuid tenant_id PK
        text idempotency_key PK "record_write_keys_pkey"
        uuid definition_id
        text action "create | update"
        uuid record_id
        text number
        bigint version
        int schema_version
        uuid audit_entry_id
    }
```

## Who may do what

The DEFAULT partition is under ENABLE + FORCE RLS and has no policy and no grant, so every runtime role is refused a direct query on it.

| Role | `audit_log` | `audit_chain_head` | `record_write_keys` |
|---|---|---|---|
| `nava_app` | INSERT on the R10 columns; SELECT on the commit-check columns | SELECT; UPDATE `export_lease_until`, `next_seal_outbox_id`, `next_seal_at` | SELECT, INSERT |
| `nava_worker` | INSERT on the R10 columns; SELECT on every column except `diff` and `diff_salt`; UPDATE of the 4 seal columns | SELECT; UPDATE `version`, `seq`, `entry_hash`, `next_seal_outbox_id`, `next_seal_at`, `seal_interval_s` | SELECT, INSERT |
| `nava_erasure` | SELECT on the R23 columns only; INSERT on the R10 columns; UPDATE `diff`, `diff_salt`, `erased_at`, `erasure_ref` | — | — |
| everyone | no DELETE | no INSERT, no DELETE | no UPDATE, no DELETE |

Later tasks add the rest:
- P5-11 generates the `nava_onboard` and `nava_groupsync` writer grants and policies.
- P5-17 adds the reader's SELECT.
- P5-33 adds onboarding's pinned INSERT on the head (A6.2).
- P5-15 and P5-32 add the migrator's temporary policies.

The column grants decide which columns a role may name. Two triggers decide what a role may do with them:
- `audit_log_assert_insert` (R10) **asserts** `occurred_at = now()`, `txid = pg_current_xact_id()` and `backfilled = false` for every role except `nava_migrator`. For every role, the migrator included, it requires the seal and erasure columns to be NULL. It never rewrites a row: a rewrite could move the row to another partition, which PG 16 refuses.
- `audit_log_guard_update` (R11) allows exactly two transitions, for every role, the superuser included:
  - **seal**, by `nava_worker`;
  - **erase**, by `nava_erasure`. It is allowed once, and only on a row that has a diff.

Both trigger functions are SECURITY INVOKER and owned by `nava_migrator`. The test asserts that `nava_migrator` owns no SECURITY DEFINER function, and that `pg_parameter_acl` grants no role `session_replication_role`, which would switch the triggers off.

## Choices that are not in ADR-004's text

- **`version_before` and `version_after` are `bigint`**, not Annex A's `integer`. `records.version` is bigint, and C1 carries `*int64`.
- **Row checks** go beyond R2:
  - digests are 32 bytes and the salt is 16 bytes;
  - the stored diff is at most 65,536 bytes (E1);
  - the diff and its salt are both present or both absent;
  - the seal columns are all set or all NULL, and so are the erasure columns;
  - an erased row has no diff;
  - a config entry has `content_sha256` unless it is a delete (Annex A).
- **Partition index names.** Each index copy on a partition is renamed to `<partition>_<parent suffix>`. A duplicate entry therefore reports `audit_log_default_commit_key`, never a generated name. P5-12b should match the `_commit_key` suffix. Epic 30 should rename the same way.
- **`occurred_at` statistics override: `n_distinct = 1`, not F1's `-0.0001`.** Finding 5 found that `-0.0001` ties at 10,000 tenants (see below). The override is per relation, so Epic 30 must set it on every partition it creates. `TestP510bOccurredAtNDistinct` catches any leaf that lacks it.

## Evidence: the commit-check probe at 10,000 tenants

`scratch/audit-log-f1-margin/` holds the script, the outputs and `RESULT.md`. The environment: postgres:16.10 in Docker with 16 CPUs, on a Windows 11 host. The table has 10,000 tenants. The probe is in its plpgsql-locals form and runs as `nava_app`.

| rows per partition | override | R2 or subject path | best prefix path | margin |
|---|---|---|---|---|
| 100,000 | `-0.0001` | 8.45 | 8.45 | 1.0× |
| 1,000,000 | `-0.0001` | 8.45 | 8.45 | 1.0× |
| 100,000 | `1` | 8.45 | 42.59 | 5.0× |
| 1,000,000 | `1` | 8.45 | 390.05 | 46× |

The margin under `n_distinct = 1` grows with rows per tenant, and passes 10× at about 20 rows per tenant per partition. No `occurred_at` override reaches 10× in a sparser partition. The probe form or the plan test (P5-11, TICK-428) has to cover that case.

Time-window estimates are unchanged by the override: the 3-day, one-tenant read estimates 20 rows and returns 20 under both overrides.

## Cost

All three tables are new and empty, so the migration takes no lock that blocks anything existing.

The write path has not been measured in this task:
- `audit_log` has 7 indexes per partition (the PK, the R2 key and 5 from R28), and every append writes all 7.
- `record_write_keys` adds one PK insert per keyed write.

S1 measured the same index set on the spike schema (`docs/adr/ADR-004/spikes/postgres/s1/RESULT.md`):
- item 3: 1,344 B of WAL per insert, plus 1,323 B per seal, so 1.98× with sealing;
- item 1: commit-check trigger overhead within the 10% p95 bar.
