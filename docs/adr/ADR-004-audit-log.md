---
type: adr
id: ADR-004
title: One audit log for records and configuration, sealed by a per-tenant hash chain and exportable (CR-AUDIT-1)
status: PROPOSED
date: 2026-11-02
issue: TICK-409
rfc: docs/adr/ADR-004/rfc/RFC-008-audit-log.md (rev 1, archived with rev-0/ and rev-1/; thread under TICK-409 comment 01a19871-56f8-7769-b940-bda191f3d55c)
supersedes: none
companion: "[[ADR-001-A6-audit-roles-and-non-record-starts]] (the A.2/A.3 text this ADR needs)"
source: CR-AUDIT-1, a founder decision (vknavaladi, 2026-11-02) after buyer research. Acceptance criterion on TICK-7, 8, 9, 13, 14, 15, 19, 20, 23, 29, 31 and 33.
deciders: Solution Architect (author); Technical Architect (round 1 CHANGES; round 2 APPROVE with C1 and C2, comment 01a19883-7e0c); Security & Multi-Tenancy Reviewer (round 1 CHANGES; round 2 APPROVE with D13 conditions and four tightenings, comment 01a19882-ca7b); Postgres & Data Modeling Specialist (round 1 CHANGES; round 2 APPROVE with two seq rules, comment 01a19882-e822); CTO (routing, anchoring deferral)
---

# ADR-004: One audit log for records and configuration

> **Status: PROPOSED.** All three reviewers approved RFC-008 rev 1 in round 2. Their conditions are written into this ADR as rules (marked **[TA C1]**, **[Sec]**, **[PG]** and so on). The Solution Architect gives the ACCEPTED verdict after the CTO routes it.
>
> **Invariants source.** The project description defines no invariants, so this ADR is checked against [[ADR-001-platform-blueprint]] §1, I1–I9.

## Decision

1. **One log.** A single append-only table, `audit_log`, holds three domains: `record`, `config` and `audit` (exports, erasures, reseeds).
   - One writer, `audit.Append(ctx, pgx.Tx, Entry)`, writes it inside the caller's transaction.
   - One reader (`GET /api/v1/audit/entries`) and one export (`GET /api/v1/audit/entries:export`) serve it.
   - Record history and configuration history share one timeline.
2. **The database checks the same-transaction rule.**
   - Every table in `db/audited-tables.yaml` gets a deferred `SECURITY INVOKER` constraint trigger (`AFTER INSERT OR DELETE OR UPDATE OF <audited columns>`).
   - At COMMIT, the trigger requires at least one entry with the same `tenant_id`, `txid = pg_current_xact_id()`, `subject_type` and `subject_id`.
   - `UNIQUE (tenant_id, occurred_at, txid, subject_type, subject_id, version_after) NULLS NOT DISTINCT` allows at most one entry per (transaction, subject, version).
   - Every writer role's `audit_log` grants and policies are generated from that manifest.
3. **Each entry is tamper-evident through a per-tenant hash chain.**
   - The chain is `entry_hash = SHA-256(prev_hash ‖ envelope_sha256)`, ordered by `seq` only.
   - A self-rescheduling sealer assigns `seq` asynchronously, using ordinary outbox starts as ADR-001 D22 requires. Writers never wait on it.
4. **Every export verifies offline, under any filter.**
   - Excluded entries in the covered range appear as hash-only stub lines.
   - The range is anchored one entry before the window.
5. **Erasure is one of exactly two transitions allowed on a written row, and only `nava_erasure` may make it.**
   - The chain covers a salted diff hash, not the diff, so it still verifies after erasure.
   - An erasure that no chained erase entry names fails verification.
6. **S8's published versions are backfilled into the log** by two migrations. They are not exposed through a view.

## Rules

### Content and vocabulary
- **R1 · columns.** The envelope is: `tenant_id`, `id` (UUIDv7, or UUIDv5 for backfill), `occurred_at`, `domain`, `action`, `subject_type`, `subject_id`, `entity_definition_id`, `version_before`, `version_after`, `actor_principal_id` **or** `actor_system` (exactly one), `source`, `source_ref`, `change_set_id`, `causation_id`, `causation_depth`, `request_id`, `backfilled`, `content_sha256`, `diff_sha256`.
  - Content: `diff` (`json`, RFC 6902, ≤ 64 KB, else `diff_truncated`) and `diff_salt` (16 random bytes).
  - Seal: `seq`, `prev_hash`, `entry_hash`, `sealed_at`.
  - Erasure: `erased_at`, `erasure_ref`.
  - `txid` (`xid8`) is used for the commit check only; it is never exported or hashed.
- **R2 · closed vocabularies**, checked by CHECK constraints.
  - `action` ∈ {create, update, publish, enable, disable, delete, promote, rollback, state_change, export, erase, reseed}.
  - `source` ∈ {user, ai, change_set, onboarding, plugin, rule, workflow, function, system}.
  - `subject_type` comes from `internal/audit/subjects.go`, mirrored by a CHECK. A new type is a platform migration (I8).
- **R3 · configuration is versioned.** Every audited configuration type keeps immutable `*_versions` rows with a content hash. A config entry carries `version_after` and its `content_sha256`; the diff is for human reading.
- **R4 · who and through which channel.** For AI changes, `source = ai` and `source_ref` points to the AI decision record (I8). Every item a promotion writes carries `change_set_id`, and the change set gets its own `promote` or `rollback` entry.

### Secrets and masks
- **R5 · one walker.** `internal/audit/patchwalk` visits every op's `path`, `from` and `value` at any depth, `test` ops included.
- **R6 · secrets are rejected at write time,** in `audit_log` and in every `*_versions` writer. A value at an `x-nava-secret` or `writeOnly` path fails with `audit_secret_in_diff` (or `config_secret_in_content`); the error and the log line name the path, never the value. Also enforced:
  - closed adapter and plugin schemas, checked by a CI lint;
  - rejection of URLs that contain userinfo;
  - a secret-shape scanner on config-domain diffs only.
- **R7 · masks are applied at read time,** for the reader.
  - A masked pointer reached through `path`, `from` or a nested `value` becomes `{"redacted":"mask"}`. When it cannot be cut out cleanly, the whole op is redacted.
  - `diff_redacted` is computed per reader; it is not stored.
  - **`diff_salt` is returned only with an unredacted diff**, in both the read and the export.
- **R8 · read gates.** `audit.read` and `audit.export` are standalone permissions. Only the seeded tenant-admin and `auditor` grants carry them. They are not filtered by record ACL.
  - The Epic 05 per-record history for other users is gated on that user's ACL for that record. It returns `domain = record` only and is masked by R7.

### Isolation and immutability
- **R9 · RLS.** Tables have `ENABLE` and `FORCE` RLS. Every policy is canonical; the "unsealed only" rule lives in triggers, not in `USING`. UPDATE policies carry a canonical `WITH CHECK`. Every key leads with `tenant_id`. **No DELETE grant or policy for any role.**
- **R10 · INSERT.**
  - Column grants exclude `occurred_at`, `txid`, `backfilled`, the seal columns and the erasure columns.
  - A `BEFORE INSERT` trigger **asserts**, and never rewrites, that for every role except `nava_migrator`: `occurred_at = now()`, `txid = pg_current_xact_id()`, `backfilled = false`, and the seal and erasure columns are NULL. A rewrite would move the row across partitions, which PG 16 forbids (PG round-2 U8).
- **R11 · exactly two UPDATE transitions.** The `BEFORE UPDATE` trigger rejects everything else.
  - **Seal**, by `nava_worker`: `OLD.sealed_at IS NULL`; the four seal columns go from NULL to a value; nothing else changes.
  - **Erase**, by `nava_erasure`: `OLD.erased_at IS NULL` and `OLD.diff IS NOT NULL`; `diff` and `diff_salt` become NULL; `erased_at` and `erasure_ref` are set; nothing else changes.
- **R12 · writer grants are generated [PG B1 / TA-B3].** `db/audited-tables.yaml` lists each audited table's writer roles and audited columns. Each writer gets:
  - INSERT on the R10 columns;
  - SELECT on `(tenant_id, occurred_at, txid, subject_type, subject_id, version_after)`;
  - canonical SELECT and INSERT policies, or for `nava_onboard`, the pinned provisioning-predicate policies.

### Chain and sealer
- **R13 · hash definition.**
  - `envelope_sha256 = SHA-256(JCS(envelope))`.
  - `entry_hash = SHA-256(prev_hash ‖ envelope_sha256)`; genesis `prev_hash` is 32 zero bytes.
  - `diff_sha256 = SHA-256(diff_salt ‖ diff bytes)`.
  - The JCS mapping is pinned in `docs/api/audit-export-v1.md`: timestamps as RFC 3339 UTC with µs, `bytea` as lowercase hex, UUIDs in lowercase, NULL emitted, not omitted.
  - Golden vectors are shared by the writer, the sealer and the verifier.
- **R14 · `audit_chain_head`** holds one row per tenant: `id`, `version`, `seq`, `entry_hash`, `next_seal_outbox_id`, `next_seal_at`, `seal_interval_s`, `export_lease_until`. Column grants **[Sec]**:
  - `nava_worker` updates `version`, `seq`, `entry_hash`, `next_seal_outbox_id`, `next_seal_at` and `seal_interval_s`.
  - `nava_app` updates only `export_lease_until`, `next_seal_outbox_id` and `next_seal_at`.
  - `nava_onboard` and `nava_migrator` insert.
  - No other role writes it.
- **R15 · `seq` rules [PG].**
  - (a) A2 takes `seq` only from the head row it read under `SELECT … FOR UPDATE`, in the same transaction that assigns it, and advances `audit_chain_head.seq` in that same transaction.
  - (b) Nothing else writes `seq`; the R11 seal transition is the only path.
  - `seq` has a plain, non-unique index (a unique key cannot span partitions, PG U10). The verifier reports duplicates and gaps. CI-A16 proves concurrent sealers serialise.
- **R16 · the self-rescheduling sealer.** Run *o* is workflow `t:<tenant>:audit.seal:<o>`, an ordinary D22 start.
  - **A1** runs in its own transaction and commits first. It locks the head `FOR UPDATE`. If `next_seal_outbox_id = o`, it bumps `version`, inserts start row *o′* (`target_key = audit.seal:<o′>`, `target_seq = version`, `available_at = now() + seal_interval_s`), and repoints `next_seal_outbox_id`. Otherwise it does nothing (a stale lineage).
  - **A2** seals in batches of 1,000, in `(occurred_at, id)` order, one transaction per batch, under the head lock (R15).
  - **Interval:** 10 s after a run that sealed anything; otherwise doubled, up to 300 s.
  - **Seeding:** onboarding inserts the head and the first start row (ADR-001-A6.2). Migration B does the same for existing tenants.
  - **Writers never touch the outbox or the head.**
- **R17 · `chain:reseed` [Sec].** `POST /api/v1/audit/chain:reseed`, as `nava_app`, under the head lock.
  - It returns **409 `chain_alive`** unless `next_seal_at < now() − 10 min`.
  - Otherwise it inserts a new start row and repoints `next_seal_outbox_id`, which retires any older lineage.
  - Every call that is not rejected writes an `audit/reseed` entry.
- **R18 · prerequisite [TA C2].** The Temporal workload allow-list (`platformWorkflowTypes`, `temporal-server/authz/policy.go`) must contain `audit.seal` before D10a ships. The change goes through a policy PR with Security review and an image rebuild, per [[ADR-001-A1-temporal-server-1-29-7-workload-allow-list]]. It is **not** an A6 item.

### Export
- **R19 · one export per tenant.** It is held by the `export_lease_until` lease; a second export gets 409 `export_in_progress`. The `audit/export` entry commits before the first byte is sent.
- **R20 · anchored range [TA-B5, TA C1].**
  - `first_seq` is the first sealed `seq` whose `occurred_at` falls in the time window, **regardless of the filter**.
  - The stream starts at **`first_seq − 1`**, emitted as a stub, or as genesis when `first_seq = 1`, and runs to `head_seq`, read at the start.
  - Matching entries are emitted in full. Every other `seq` is a stub: `{seq, prev_hash, entry_hash, envelope_sha256}`.
  - **The verifier rejects a file whose first chain line is not that anchor.** It checks that the range is contiguous and that every link holds.
  - With `stubs=false` the trailer says `verifiable: false`.
- **R21 · paging.**
  - Pages of 5,000 rows, each in its own short transaction, with `statement_timeout = 30s` per page and a 15-minute cap. Both values are unverified; S1 measures them.
  - Hashed columns never change after sealing, so a fixed `head_seq` bound gives a consistent trailer without a long-held snapshot.
  - Unsealed matches come last, flagged `sealed: false, in_verified_range: false`.
  - The trailer is `{anchor_seq, last_seq, head_hash, count, stub_count, unsealed_count}`.
  - Incremental consumers use `after_seq`.
- **R22 · formats.** JSONL `nava.audit.export/v1` and CSV with the same lines. In CSV, every cell beginning with `=`, `+`, `-`, `@`, tab or CR gets a `'` prefix (OWASP). The verifier ships as `nava audit verify <file>`, offline.

### Erasure (the contract for Epic 30)
- **R23 · the erase step.** `nava_erasure` (ADR-001-A6.1) runs with `app.tenant_id` set. In one transaction it:
  - applies the R11 erase transition;
  - appends `audit/erase` entries with `subject_type = erasure_request`, `subject_id = erasure_ref`, and a diff of the erased entry ids only, at most 10,000 per entry.

  It reads only the columns it needs, by column grant **[Sec]**: `(tenant_id, occurred_at, id, subject_type, subject_id, erased_at, erasure_ref)`, never `diff` or `diff_salt`. Its credential lives with the job runner, never on an app or worker host.
- **R24 · verifier outcomes for erased entries.**
  - An erased entry passes only if a chain-valid `audit/erase` entry with `subject_id = erasure_ref` lists its id. Otherwise it is **`tampered`**.
  - If that erase entry exists but is not yet sealed, the result is **`pending`** **[Sec]**.
  - The export always includes the matching erase entries for erased entries in scope.

### S8 backfill
- **R25 · order [PG B2].**
  - **Migration A** creates the tables and backfills.
  - **The deploy** makes the S8 publish path call `audit.Append`.
  - **Migration B**, in one transaction, re-runs the backfill, attaches the commit trigger to `entity_definitions` and `entity_definition_versions`, and seeds the head and the first `audit.seal` row per tenant.
  - Dedupe uses `NOT EXISTS` on `(tenant_id, 'entity_definition', definition_id, version_after = version)`.
  - The id is UUIDv5(namespace, `tenant_id ‖ version_id`).
  - Each version becomes one entry: `create` for v1, otherwise `publish`. The actor is `published_by`, or the system actor `onboarding`. `content_sha256 = schema_sha256`, `occurred_at = published_at`, `backfilled = true`.
- **R26 · access path [Sec conditions 1–3].**
  1. All temporary policies are created and dropped in **one transaction**. The migration file must not carry a `NO TRANSACTION` directive.
  2. The only cross-tenant policy is `FOR SELECT TO nava_migrator USING (true)` on `tenants`, used to list ids. Every tenant table is read and written through temporary **canonical** migrator policies, after `set_config('app.tenant_id', …, true)` (`is_local = true`) for each tenant.
  3. The post-run catalogue gate fails on any leftover `nava_migrator` policy, asserted in CI-A3.
- **R27 · locks [PG].** Migration B sets `lock_timeout`. It then takes all five ACCESS EXCLUSIVE locks up front, in this fixed order: `tenants`, `entity_definitions`, `entity_definition_versions`, `audit_log`, `audit_chain_head`, `outbox`. It retries on timeout. *(Six names: `entity_definitions` joins the five because the trigger is attached to it too.)*

### Storage
- **R28 · partitioned key shape.** `PARTITION BY RANGE (occurred_at)` with one DEFAULT partition; Epic 30 owns partition management.
  - PK: `(tenant_id, occurred_at, id)`.
  - Indexes: `(tenant_id, occurred_at DESC, id DESC)`; `(tenant_id, subject_type, subject_id, occurred_at DESC)`; `(tenant_id, change_set_id) WHERE change_set_id IS NOT NULL`; `(tenant_id, seq) WHERE seq IS NOT NULL`; `(tenant_id, occurred_at, id) WHERE sealed_at IS NULL`; and the R2 commit-check unique key.
  - No GIN on `diff`.

## Consequences

- Each configuration epic (06, 10, 11, 12, 16, 17, 20, 28) registers its subject types and manifest rows; CI-A2 and CI-A3 become part of its acceptance.
- Epic 05 builds the core. Entries carry `sealed: false` until the sealer ships, after the Epic 08 relay and the R18 allow-list entry. That is honest and needs no retrofit.
- Sealing roughly doubles heap WAL on `audit_log`, because each row is rewritten once and the update is not HOT. S1 measures it.
- The sealer needs a second outbox producer entry point, not tied to a record (ADR-001-A6.3). The Technical Lead plans it.

## Rejected options

| Option | Why it lost |
|---|---|
| Separate record and configuration logs, or a union view | Two timelines and two exports, and the chain cannot cover them (CR-AUDIT-1 item 4). |
| A view over `entity_definition_versions` instead of a backfill | A second reader that the chain and the export cannot cover. Deletes and disables have no version row to show. |
| Chaining inside the writer's transaction | It serialises every write in a tenant on one head row. |
| A per-minute shared seal key in the writer's transaction (rev 0) | Writers queued behind an uncommitted key: 3.9–4.0 s in two spikes, plus a deadlock case. |
| `audit-seal:<txid>` per transaction | One outbox row per write, and the outbox stays on the write path. |
| An eternal per-tenant workflow with a fixed id | It needs a D22 workflow-id scheme change. Self-rescheduling gives the same liveness with ordinary starts. |
| Database triggers that write the audit rows | A trigger cannot know the actor, source, AI decision or change set without session variables that could be forged. |
| `jsonb` diff | jsonb rewrites the bytes, so the stored diff would no longer match its hash (TICK-360). |
| `USING (true)` on `entity_definition_versions` for the backfill | A join bug could file one tenant's version in another tenant's log, and FORCE RLS would not stop it. |
| Narrowing verification to unfiltered exports | Buyers test filtered exports (CTO). |
| A single `REPEATABLE READ` export snapshot | It holds the cluster-wide vacuum horizon for every tenant. Sealed rows are immutable, so a fixed bound is enough. |
| External anchoring in M1 (WORM storage, RFC 3161) | A new dependency with recurring cost. The CTO deferred it (R-A1). |
| A dedicated sealer role | It adds nothing while the sealer shares the worker credential (ADR-001 R8). Security accepted `nava_worker`. |

## Accepted risks

| # | Risk | Mitigation | Owner | Revisit trigger |
|---|---|---|---|---|
| R-A1 | A consistent whole-chain rewrite by the owner is detected only against an export the tenant already holds. **The CTO deferred external anchoring.** | Export trailers. The migrator is a one-shot job. | Solution Architect, tracked by the CTO | A regulated buyer asks for external attestation. |
| R-A2 | Sealing lag: about 10 s when the tenant is active, up to 300 s after idle. | `audit_unsealed_oldest_seconds` alert at 600 s, `audit_seal_failures_total`, `chain:reseed`. | Technical Lead | The alert fires in production. |
| R-A4 | Commit-trigger, sealing-WAL and deferred-queue costs are unmeasured. | **S1, owned by the Postgres Specialist and gating Epic 05 planning:** ≤ 10% p95 overhead, run through PgBouncer 1.24.1, a 10k-subject transaction, peak memory, WAL per entry, and the export caps. If it fails, record tables fall back to the architecture test. | Postgres Specialist | S1 result. |
| R-A5 | Idle-tenant seal starts: about 3.3/s per 1,000 tenants (estimated). | Back-off. Measured in the Epic 08 flood test. | Technical Lead | Starts above 10% of Temporal capacity. |
| R-A6 | Migration B holds ACCESS EXCLUSIVE locks across its transaction. | R27 ordering and `lock_timeout`, a deploy window, and batching if there are over 100k versions. | Postgres Specialist | S8 volume at Epic 05 planning. |
| (R8) | A compromised worker credential can seal or insert for any tenant. | Already accepted in ADR-001. R10, R11 and R15 bound what it can change. | Security reviewer | ADR-001 R8 trigger. |

**Dissent: none recorded.** All three reviewers approved in round 2.

## Acceptance tests

| # | Proves |
|---|---|
| CI-A1 | A kill or rollback leaves no entry. A commit leaves ≥ 1 entry per (transaction, subject) and ≤ 1 per version. |
| CI-A2 | A raw write to an audited table fails at COMMIT, for every writer role in the manifest. |
| CI-A3 | Every audited table has the trigger with `UPDATE OF` its audited columns. Every tenant table is audited or exempt with a reason. Every writer role can run the check. No `nava_migrator` policy is left over (R26.3). |
| CI-A4 | Read, export, stubs and reseed never cross tenants. With no tenant context, zero rows. |
| CI-A5 | Secret canaries are rejected as a top-level value, a nested `add`, a parent `replace`, and a `copy` from a secret path, in `audit_log` and in `*_versions`. Nothing secret appears in table bytes or export bytes. The error names the path only. |
| CI-A6 | Verification passes, both unfiltered and with subject, time and change-set filters. A byte edit, a deleted row, a single-row rewrite, **and a deleted first entry in the range** (R20) each fail under any filter. A file without the anchor is rejected. |
| CI-A7 | `nava_erasure` erases and the result verifies. `nava_app` and `nava_worker` cannot erase. A nulled diff with no erase entry is `tampered`. An unsealed erase entry is `pending`. A second erase is rejected. `nava_erasure` cannot read `diff`. |
| CI-A8 | An export commits its own entry first. A second concurrent export gets 409. |
| CI-A9 | The backfill leaves exactly one entry per S8 version after A, after the deploy and after B, including a version published between A and B. |
| CI-A10 | For `nava_app`, `nava_worker`, `nava_onboard` and `nava_groupsync`: protected INSERT columns are denied, UPDATE outside the two transitions is denied, and DELETE is denied. `nava_app` cannot write `seq`, `entry_hash` or `version` on the head. |
| CI-A11 | One timeline shows a configuration change and the breach it caused. |
| CI-A12 | Masks hold on a root `add`, a nested `replace`, `copy` and `move`, and `test`, in the read and the export. |
| CI-A13 | A promotion and its rollback each write per-item entries with `change_set_id`. |
| CI-A14 | A redacted entry carries no `diff_salt`. An unredacted entry carries it and its diff verifies. |
| CI-A15 | CSV formula prefixing. |
| CI-A16 | A kill between A1 and A2 is recovered by the next run. A poison batch keeps the lineage alive. Reseed returns 409 while the chain is alive, and audits itself when it runs. An idle tenant is sealed within 300 s. Two concurrent sealers never duplicate a `seq` (R15). Onboarding cannot insert an outbox row of any kind other than `audit.seal`. |
