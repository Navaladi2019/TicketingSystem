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
        uuid tenant_id UK "record_write_keys_actor_key"
        text idempotency_key UK
        uuid actor_principal_id UK "NULLS NOT DISTINCT (TICK-626)"
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
| `nava_purge` | — | SELECT `tenant_id`, `next_seal_at` of dead heads only (`audit_chain_head_purge_select`, R17's predicate, every tenant; TICK-590) | — |
| everyone | no DELETE | no INSERT, no DELETE | no UPDATE, no DELETE |

Later tasks add the rest:
- P5-11 generates the `nava_onboard` and `nava_groupsync` writer grants and policies.
- P5-17 (TICK-435) added the reader's SELECT (nava_app, diff and diff_salt included).
- P5-33 (TICK-441) added onboarding's pinned INSERT on the head and on `outbox` (`kind = 'audit.seal'` only) (A6.2); it commits while the tenant is `provisioning`, and `Activate` is a later transaction.
- P5-34f (TICK-590) added `nava_purge`'s read of dead heads for the `audit_dead_lineages` gauge (see [Audit sealer](../architecture/audit-sealer.md#dead-lineages-r17)).
- P5-40 (TICK-444) added `GET /api/v1/audit/entries:export`. `nava_app` takes the lease with one `UPDATE audit_chain_head ... RETURNING seq, entry_hash` and writes the `audit/export` entry in the same transaction; the pages stream on the `nava_audit_reader` pool, which needs SELECT on `prev_hash`, `entry_hash` and `erasure_ref` (TICK-612).
- P5-15 and P5-32 add the migrator's temporary policies, created and dropped inside the backfill and the seeder (see [S8 backfill](#s8-backfill-migration-a-p5-15) and [Migration B](#migration-b-p5-32)).

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

## S8 backfill: migration A (P5-15)

Migration: `db/migrations/20261103120000_audit_backfill_s8.sql`. It creates `audit_backfill_s8()`, runs it once, then fails if any `nava_migrator` policy is left (R26.3). P5-32 (migration B) calls the same function again.

```mermaid
erDiagram
    tenants ||--o{ entity_definition_versions : "tenant_id"
    entity_definition_versions ||--|| audit_log : "one backfilled entry"
    entity_definition_versions {
        uuid tenant_id
        uuid id
        uuid definition_id
        int version
        bytea schema_sha256
        uuid published_by
        timestamptz published_at
    }
    audit_log {
        uuid tenant_id "= version tenant_id"
        uuid id "UUIDv5(ns, tenant_id || version id)"
        timestamptz occurred_at "= published_at"
        text action "create for v1, else publish"
        uuid subject_id "= definition_id"
        bigint version_after "= version"
        bytea content_sha256 "= schema_sha256"
        boolean backfilled "true"
    }
```

- **Entry.** Domain `config`, subject `entity_definition`. `version_before` is NULL for v1 and `version - 1` otherwise. The actor is `published_by` with source `user`. Only a v1 seeded by onboarding has no publisher; its actor is `actor_system = onboarding` with source `onboarding`. No diff: R3 makes the content hash the evidence.
- **Id.** UUIDv5 under the fixed namespace `3767d3ad-0a89-4b41-847c-9e4bf30c2055`, over the 32 raw bytes `tenant_id ‖ version_id`. ADR-004-E1 rules out `pgcrypto` and PG 16 has no built-in SHA-1, so `audit_backfill_sha1` implements FIPS 180-4 in plpgsql. `TestP515UUIDv5MatchesGo` checks it against `crypto/sha1` at every padding boundary and against Go's `uuid.NewSHA1`.
- **Dedupe.** `NOT EXISTS (tenant_id, 'entity_definition', definition_id, version_after = version)`, so a re-run adds only versions published since the last run (CI-A9).
- **Access path (R26).** Four temporary policies, created and dropped inside the function, in the migration's single transaction:
  - `FOR SELECT TO nava_migrator USING (true)` on `tenants`, the only cross-tenant policy, used to list ids;
  - canonical SELECT on `entity_definition_versions`;
  - canonical SELECT and INSERT on `audit_log`.

  For each tenant it calls `set_config('app.tenant_id', id, true)` and runs one `INSERT … SELECT`. All three functions are SECURITY INVOKER, owned by `nava_migrator` and have no EXECUTE grant.
- **Plan pin.** The function runs with `enable_seqscan = off`. While the run fills `audit_log`, its statistics still say it is empty, so the cached plan seq-scanned `audit_log_default` for each tenant's dedupe: O(n²), 291.7 s for 300,000 versions. With the pin, the probe is an index scan on `audit_log_default_subject` (0.5–0.8 ms per tenant).
- **Lock cost.** CREATE POLICY holds ACCESS EXCLUSIVE on `tenants`, `entity_definition_versions` and `audit_log` until COMMIT, so tenant resolution waits for the whole run. Measured on postgres:16.10 in Docker (16 CPUs, Windows 11 host), from `scratch/audit-backfill-s8/RESULT.md`:

| Tenants | Versions | Migration A | Re-run, all present |
|---:|---:|---:|---:|
| 1,000 | 30,000 | 5.62 s | 0.14 s |
| 10,000 | 300,000 | 54.69 s | 2.32 s |

Runbook: the migration sets `lock_timeout = '5s'`. That limits only how long it **waits** to take the locks; once it has them it holds them for the whole run, about 0.18 ms per version to backfill (budget: ≤ 18 s at 100k versions, `docs/budgets.md`). Size the window from the S8 version count, not from `lock_timeout`. The function takes the locks in the order tenants, entity_definition_versions, audit_log (the order of its CREATE POLICY statements); an explicit `LOCK TABLE` up front was measured and not adopted (`scratch/audit-backfill-s8/tick515-lock/RESULT.md`).

## Migration B (P5-32)

Migration: `db/migrations/20261107053707_audit_migration_b.sql`, one transaction (ADR-004 R25 B half, R26, R27). In order:

1. **Locks (R27).** `lock_timeout = 5s`, then `LOCK TABLE tenants, entity_definitions, entity_definition_versions, audit_log, audit_chain_head, outbox IN ACCESS EXCLUSIVE MODE`, in that order, before any other work. A timeout fails B with `55P03`; its transaction rolls back, releasing whatever part of the set it took, and the migrator retries it (TICK-579: 10 attempts, backoff doubling from 1 s to 30 s). `lock_timeout` bounds each lock wait, so one attempt waits at most 30 s: about 7.5 min worst case, about 3.4 min when one lock is contended. The migrator is the retry's only owner; B has no retry loop of its own (TICK-606). Every later statement needs at most a lock already held.
2. **Backfill re-run.** `SELECT audit_backfill_s8()`: the versions published between migration A and the deploy get their backfilled entry; versions the publish path audited keep theirs (CI-A9).
3. **Attach.** `audit_commit_check` on `entity_definitions` (`AFTER INSERT OR DELETE OR UPDATE OF current_version, updated_at`, subject `id`) and on `entity_definition_versions` (`AFTER INSERT OR DELETE`, insert-only, subject `definition_id`). `db/audited-tables.yaml` marks both `attached: true`.
4. **Seed.** `SELECT audit_seed_chain_heads()`: per tenant without a head, the head and the first `audit.seal` start row, the rows onboarding writes (P5-33).
5. **Gate.** Fails if any `nava_migrator` policy is left (R26.3).

```mermaid
erDiagram
    tenants ||--|| audit_chain_head : "one per tenant"
    audit_chain_head ||--|| outbox : "next_seal_outbox_id"
    audit_chain_head {
        uuid tenant_id PK
        uuid id "UUIDv7"
        bigint version "1 (default)"
        bigint seq "0 (default)"
        bytea entry_hash "32 zero bytes (default)"
        uuid next_seal_outbox_id "= the start row"
        timestamptz next_seal_at "now()"
        int seal_interval_s "10 (default)"
    }
    outbox {
        uuid tenant_id PK
        uuid id PK "UUIDv7"
        text kind "audit.seal"
        smallint kind_version "1"
        text dedupe_key "audit.seal:1"
        text target_key "audit.seal:<id>"
        bigint target_seq "1 = head version"
        uuid causation_id "= id"
        smallint causation_depth "0"
        jsonb payload "{}"
        text trace_context "NULL"
        timestamptz available_at "now()"
    }
```

- **Same rows as onboarding and C5.** The start row is the row `outbox.EnqueueSeal(ctx, tx, 1)` writes with no option, and the head is the row `onboard.PGStore.SeedAuditChain` writes. `TestP532SealRowMatchesEnqueueSeal` compares all three column by column, except the ids and the clock (TICK-449 grooming). The ids are UUIDv7: 48 bits of `clock_timestamp()` milliseconds, then `gen_random_uuid()`'s random bits (`audit_seed_uuidv7`; PG 16 has no `uuidv7()`).
- **Idempotent both ways.** The head's primary key is the idempotency record (`ON CONFLICT DO NOTHING`). A tenant onboarding already seeded keeps its rows; a tenant still provisioning gets them from B, and onboarding's later `SeedAuditChain` writes nothing. A seal row with no head is corruption, so the start-row insert has no `ON CONFLICT` and fails B on it.
- **Access path (R26).** Three temporary policies, created and dropped inside `audit_seed_chain_heads()`: `FOR SELECT TO nava_migrator USING (true)` on `tenants` (the only cross-tenant one, to list ids), and canonical INSERT on `audit_chain_head` and on `outbox`. Plain INSERTs with no RETURNING, so no SELECT policy is needed. Both new functions are SECURITY INVOKER, owned by `nava_migrator`, with no EXECUTE grant.
- **Onboarding's entity types.** From B on, `entitytypes.Store.Seed` (nava_onboard) writes the v1 config `create` entry in its own transaction: actor `onboarding`, source `onboarding`, `content_sha256 = schema_sha256`, no diff (nava_onboard cannot read the stored schema; R3 makes the hash the evidence). It is the entry the backfill gives such a version, without `backfilled`.

Measured on postgres:16.10 in Docker (16 CPUs, Windows 11 host), from `docs/data/audit-migration-b-bench/RESULT.md`. B holds all six locks for the whole time:

| Tenants | S8 versions | Migration B | Backfill re-run | Seeding |
|---:|---:|---:|---:|---:|
| 1,000 | 30,093 | 0.48 s | 175.5 ms | 202.0 ms |
| 10,000 | 300,972 | 4.29 s | 2,864.3 ms | 1,281.7 ms |

Runbook: size the window as about 0.13 ms per tenant plus 9.5 µs per S8 version, plus about 0.1 s. Per tenant, B writes one head row (one index) and one outbox row (four indexes), once: 1.9 MB and 4.4 MB at 10,000 tenants. These are onboarding's rows, so B adds no index and no lasting write path.

## Cost

All three tables are new and empty, so the migration takes no lock that blocks anything existing.

The write path has not been measured in this task:
- `audit_log` has 7 indexes per partition (the PK, the R2 key and 5 from R28), and every append writes all 7.
- `record_write_keys` adds one unique-index insert per keyed write (`record_write_keys_actor_key` since TICK-626).

S1 measured the same index set on the spike schema (`docs/adr/ADR-004/spikes/postgres/s1/RESULT.md`):
- item 3: 1,344 B of WAL per insert, plus 1,323 B per seal, so 1.98× with sealing;
- item 1: commit-check trigger overhead within the 10% p95 bar.
