-- P5-10b (TICK-450): ADR-004's audit_log, its chain head and the pipeline's
-- idempotency keys (ADR-004 R1, R2, R9, R10, R11, R14, R28; ADR-004-E1;
-- ADR-001-A6.1; PLAN-005 D-5-3; spike S1 finding F1). audit_log is empty
-- until P5-13b moves the pipeline onto audit.Append, so every statement here
-- is on new, empty tables: no rewrite, no backfill, no CONCURRENTLY needed.
--
-- Who may do what (db/grants.yaml; R10, R11, R14, R23):
--   nava_app      audit_log: INSERT on the R10 columns, SELECT on the commit-
--                 check columns (R12). audit_chain_head: SELECT, UPDATE of
--                 export_lease_until, next_seal_outbox_id, next_seal_at.
--                 record_write_keys: INSERT, SELECT.
--   nava_worker   audit_log: INSERT on the R10 columns; SELECT on every column
--                 but diff and diff_salt (the sealer hashes the envelope, which
--                 holds diff_sha256, never the diff); UPDATE of the four seal
--                 columns (the R11 seal transition). audit_chain_head: SELECT,
--                 UPDATE of the six sealer columns. record_write_keys: INSERT,
--                 SELECT.
--   nava_erasure  audit_log: SELECT on the R23 M1 columns only, never diff or
--                 diff_salt; INSERT on the R10 columns (its audit/erase
--                 entries); UPDATE of diff, diff_salt, erased_at, erasure_ref
--                 (the R11 erase transition).
--   nobody        DELETE on any of the three. No grant at all on the DEFAULT
--                 partition: it is reached only through audit_log.
-- Later tasks add the rest: P5-11 the generated writer grants and policies
-- for nava_onboard and nava_groupsync, P5-17 the reader's SELECT, P5-33
-- onboarding's pinned INSERT on audit_chain_head (A6.2), P5-15 and P5-32 the
-- migrator's temporary policies.
--
-- Column grants stop a runtime role from naming a protected column; the R10
-- trigger stops a value arriving any other way (a default changed later, a
-- role granted the column by mistake). The R11 trigger is the only path to
-- an UPDATE: grants say which columns a role may name, the trigger says which
-- transition it may make. Both triggers are plain SECURITY INVOKER plpgsql
-- owned by nava_migrator; nothing here is SECURITY DEFINER (CI-1, and the
-- TICK-460 assertion that nava_migrator owns no SECURITY DEFINER function,
-- since R10 exempts current_user = 'nava_migrator').
--
-- Every key and foreign key leads with tenant_id, because PostgreSQL runs
-- foreign-key checks as the table owner and so ignores RLS (TICK-360).
-- Design and evidence: docs/data/audit-log.md.

-- +goose Up

-- ---------------------------------------------------------------- audit_log
-- R1 plus diff_truncated (E1). Annex A types, with three deliberate changes:
--   source_ref is text, not uuid (P5-02 review, TICK-454): C2 types it
--     *string; P5-12b checks it is a UUID when source = ai.
--   version_before and version_after are bigint, not integer: records.version
--     is bigint and C1 carries *int64, so an integer column would refuse a
--     legal record version past 2^31 - 1.
--   every envelope text column refuses '' (TICK-454): the canonical form
--     rejects empty text, so a stored '' could never be sealed and would stall
--     the tenant's chain.
-- diff is json, never jsonb: jsonb rewrites the bytes, and diff_sha256 is
-- over the stored bytes (R13, E1).
CREATE TABLE audit_log (
    tenant_id            uuid        NOT NULL,
    id                   uuid        NOT NULL, -- UUIDv7; UUIDv5 for backfill
    occurred_at          timestamptz NOT NULL DEFAULT now(),
    txid                 xid8        NOT NULL DEFAULT pg_current_xact_id(),
    domain               text        NOT NULL,
    action               text        NOT NULL,
    subject_type         text        NOT NULL,
    subject_id           uuid        NOT NULL,
    entity_definition_id uuid,
    version_before       bigint,
    version_after        bigint,
    actor_principal_id   uuid,
    actor_system         text,
    source               text        NOT NULL,
    source_ref           text,
    change_set_id        uuid,
    causation_id         uuid,
    causation_depth      smallint    NOT NULL DEFAULT 0,
    request_id           text,
    backfilled           boolean     NOT NULL DEFAULT false,
    content_sha256       bytea,
    diff_sha256          bytea,
    diff_truncated       boolean     NOT NULL DEFAULT false,
    diff                 json,
    diff_salt            bytea,
    seq                  bigint,
    prev_hash            bytea,
    entry_hash           bytea,
    sealed_at            timestamptz,
    erased_at            timestamptz,
    erasure_ref          uuid,
    CONSTRAINT audit_log_pkey PRIMARY KEY (tenant_id, occurred_at, id),
    -- R2: at most one entry per (transaction, subject, version). occurred_at
    -- = now() and txid are asserted by R10, so within one transaction the key
    -- reduces to (subject, version_after) (E1). It includes the partition key,
    -- as a unique constraint on a partitioned table must.
    CONSTRAINT audit_log_commit_key
        UNIQUE NULLS NOT DISTINCT (tenant_id, occurred_at, txid, subject_type, subject_id, version_after),
    CONSTRAINT audit_log_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT audit_log_entity_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT audit_log_actor_principal_fkey FOREIGN KEY (tenant_id, actor_principal_id)
        REFERENCES principals (tenant_id, id),
    -- R2 closed vocabularies (with E1's two cascade actions). subject_type
    -- mirrors internal/audit/subjects.go; a new type is a platform migration
    -- (I8), and TestP510bSubjectTypesMirrorRegistry fails on any drift.
    CONSTRAINT audit_log_domain_check CHECK (domain IN ('record', 'config', 'audit')),
    CONSTRAINT audit_log_action_check CHECK (action IN (
        'create', 'update', 'publish', 'enable', 'disable', 'delete', 'promote', 'rollback',
        'state_change', 'export', 'erase', 'reseed', 'cascade_cycle', 'cascade_too_deep')),
    CONSTRAINT audit_log_source_check CHECK (source IN (
        'user', 'ai', 'change_set', 'onboarding', 'plugin', 'rule', 'workflow', 'function', 'system')),
    CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
        'record', 'cascade_key', 'entity_definition', 'audit_export', 'audit_chain', 'erasure_request')),
    -- domain, action, source and subject_type are also non-empty through
    -- their lists; the explicit test keeps the rule in one place.
    CONSTRAINT audit_log_text_nonempty_check CHECK (
        domain <> '' AND action <> '' AND subject_type <> '' AND actor_system <> ''
        AND source <> '' AND source_ref <> '' AND request_id <> ''),
    -- R1: a principal or a named system actor, exactly one.
    CONSTRAINT audit_log_actor_check CHECK ((actor_principal_id IS NULL) <> (actor_system IS NULL)),
    CONSTRAINT audit_log_causation_depth_check CHECK (causation_depth >= 0),
    -- Annex A: a config entry carries its version's content hash, except a delete.
    CONSTRAINT audit_log_config_content_check CHECK (
        domain <> 'config' OR action = 'delete' OR content_sha256 IS NOT NULL),
    -- SHA-256 digests are 32 bytes, the salt 16 (R1, R13).
    CONSTRAINT audit_log_digest_length_check CHECK (
        octet_length(content_sha256) = 32 AND octet_length(diff_sha256) = 32
        AND octet_length(prev_hash) = 32 AND octet_length(entry_hash) = 32),
    CONSTRAINT audit_log_diff_salt_length_check CHECK (octet_length(diff_salt) = 16),
    -- E1: the stored diff is at most 65,536 bytes, truncated or not.
    CONSTRAINT audit_log_diff_size_check CHECK (octet_length(diff::text) <= 65536),
    -- The diff and its salt come and go together: both written, or both
    -- nulled by erasure (R11). Erasure also leaves diff_sha256 in place.
    CONSTRAINT audit_log_diff_salt_pair_check CHECK ((diff IS NULL) = (diff_salt IS NULL)),
    -- Seal and erasure columns are set as a group or not at all (R11).
    CONSTRAINT audit_log_seal_all_or_none_check CHECK (
        (seq IS NULL) = (prev_hash IS NULL) AND (seq IS NULL) = (entry_hash IS NULL)
        AND (seq IS NULL) = (sealed_at IS NULL)),
    CONSTRAINT audit_log_seq_check CHECK (seq >= 1),
    CONSTRAINT audit_log_erasure_all_or_none_check CHECK ((erased_at IS NULL) = (erasure_ref IS NULL)),
    CONSTRAINT audit_log_erased_has_no_diff_check CHECK (erased_at IS NULL OR diff IS NULL)
) PARTITION BY RANGE (occurred_at);

-- R28: one DEFAULT partition; Epic 30 adds, detaches and retains range
-- partitions, and must apply F1 and FORCE RLS to each one it creates.
CREATE TABLE audit_log_default PARTITION OF audit_log DEFAULT;

-- R28 index set, every one led by tenant_id. Created on the parent, so each
-- partition gets its own copy. No GIN on diff. seq is not unique: a unique
-- key cannot span partitions (PG U10); the head lock serialises sealers and
-- the verifier reports duplicates and gaps (R15).
CREATE INDEX audit_log_tenant_time ON audit_log (tenant_id, occurred_at DESC, id DESC);
CREATE INDEX audit_log_subject ON audit_log (tenant_id, subject_type, subject_id, occurred_at DESC);
CREATE INDEX audit_log_change_set ON audit_log (tenant_id, change_set_id) WHERE change_set_id IS NOT NULL;
CREATE INDEX audit_log_seq ON audit_log (tenant_id, seq) WHERE seq IS NOT NULL;
CREATE INDEX audit_log_unsealed ON audit_log (tenant_id, occurred_at, id) WHERE sealed_at IS NULL;

-- A partition's copy of a parent index gets a generated name, and a unique
-- violation names the partition's constraint, not the parent's. Rename each
-- copy to audit_log_default_<suffix of its parent>, so the R2 key's
-- violation is always audit_log_default_commit_key here, and a partition
-- created later can follow the same rule (<partition>_commit_key). P5-12b
-- matches the _commit_key suffix.
-- +goose StatementBegin
DO $$
DECLARE
    r record;
BEGIN
    FOR r IN
        SELECT c.relname AS child, p.relname AS parent
        FROM pg_inherits i
        JOIN pg_class c ON c.oid = i.inhrelid
        JOIN pg_class p ON p.oid = i.inhparent
        JOIN pg_index x ON x.indexrelid = c.oid
        WHERE x.indrelid = 'audit_log_default'::regclass
          AND c.relname <> 'audit_log_default' || substr(p.relname, length('audit_log') + 1)
    LOOP
        EXECUTE format('ALTER INDEX %I RENAME TO %I', r.child, 'audit_log_default' || substr(r.parent, length('audit_log') + 1));
    END LOOP;
END
$$;
-- +goose StatementEnd

-- S1 finding F1 (TICK-423), as re-measured for the Technical Architect's
-- Finding 5 (TICK-450, scratch/audit-log-f1-margin): without an override
-- the commit check's probe ties between the R2 key and every
-- (tenant_id, occurred_at)-prefixed index, and the prefix plan is O(n^2) in
-- a large transaction (24.1 s against 142 ms for 10k subjects). F1 pinned
-- n_distinct = -0.0001; at 10,000 tenants per partition that still ties
-- (prefix 8.45 = R2 8.45, 1.0x, at 100k and 1M rows), so Finding 5 failed.
-- n_distinct = 1, the strongest value, gives 46x at 1M rows (390.05 against
-- 8.45) and 5.0x at 100k (42.59 against 8.45), and leaves time-window
-- estimates unchanged (they read the histogram). It is the value adopted.
-- The option is per relation and partitions do not inherit it, so it is set
-- on the parent and on the DEFAULT partition; every partition created later
-- must set it too (TestP510bOccurredAtNDistinct checks every leaf of
-- pg_partition_tree). ANALYZE so the planner reads it.
ALTER TABLE audit_log ALTER COLUMN occurred_at SET (n_distinct = 1);
ALTER TABLE audit_log_default ALTER COLUMN occurred_at SET (n_distinct = 1);
ANALYZE audit_log;

-- R9: ENABLE + FORCE on the parent and on the partition. The partition has
-- no policy and no grant, so a direct query is denied to every runtime role
-- and reaches no row even for the owner.
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_log_default ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log_default FORCE ROW LEVEL SECURITY;

-- Canonical policies only; "unsealed only" and the transitions live in the
-- triggers, not in USING (R9, PG N3). No DELETE policy for anyone.
CREATE POLICY audit_log_select ON audit_log FOR SELECT TO nava_app, nava_worker, nava_erasure
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY audit_log_insert ON audit_log FOR INSERT TO nava_app, nava_worker, nava_erasure
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY audit_log_update ON audit_log FOR UPDATE TO nava_worker, nava_erasure
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- R10: assert, never rewrite. A BEFORE trigger that changed occurred_at
-- could move the row to another partition, which PG 16 refuses (0A000;
-- S1 item 4). Only nava_migrator, the one-shot backfill (R25), may set
-- occurred_at, txid or backfilled. Nobody inserts a sealed or erased row.
-- +goose StatementBegin
CREATE FUNCTION audit_log_assert_insert() RETURNS trigger
    LANGUAGE plpgsql SECURITY INVOKER SET search_path = pg_catalog, public AS $$
BEGIN
    IF current_user <> 'nava_migrator' AND (
           NEW.occurred_at IS DISTINCT FROM now()
        OR NEW.txid IS DISTINCT FROM pg_current_xact_id()
        OR NEW.backfilled IS DISTINCT FROM false) THEN
        RAISE EXCEPTION 'audit_log: % may not set occurred_at, txid or backfilled', current_user
            USING ERRCODE = '42501';
    END IF;
    IF NEW.seq IS NOT NULL OR NEW.prev_hash IS NOT NULL OR NEW.entry_hash IS NOT NULL OR NEW.sealed_at IS NOT NULL
       OR NEW.erased_at IS NOT NULL OR NEW.erasure_ref IS NOT NULL THEN
        RAISE EXCEPTION 'audit_log: % may not insert a sealed or erased entry', current_user
            USING ERRCODE = '42501';
    END IF;
    RETURN NEW;
END
$$;
-- +goose StatementEnd
CREATE TRIGGER audit_log_assert_insert BEFORE INSERT ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_assert_insert();

-- R11: exactly two transitions, for every role, the owner included.
--   Seal (nava_worker): an unsealed row gets all four seal columns; nothing
--   else changes.
--   Erase (nava_erasure): a row not yet erased, with a diff, loses diff and
--   diff_salt and gets erased_at and erasure_ref; nothing else changes, the
--   seal columns and diff_truncated included. Sealed or unsealed alike.
-- json has no equality operator, so diff is compared as its stored text.
-- +goose StatementBegin
CREATE FUNCTION audit_log_guard_update() RETURNS trigger
    LANGUAGE plpgsql SECURITY INVOKER SET search_path = pg_catalog, public AS $$
DECLARE
    same_envelope boolean :=
        (NEW.tenant_id, NEW.id, NEW.occurred_at, NEW.txid, NEW.domain, NEW.action, NEW.subject_type, NEW.subject_id,
         NEW.entity_definition_id, NEW.version_before, NEW.version_after, NEW.actor_principal_id, NEW.actor_system,
         NEW.source, NEW.source_ref, NEW.change_set_id, NEW.causation_id, NEW.causation_depth, NEW.request_id,
         NEW.backfilled, NEW.content_sha256, NEW.diff_sha256, NEW.diff_truncated)
        IS NOT DISTINCT FROM
        (OLD.tenant_id, OLD.id, OLD.occurred_at, OLD.txid, OLD.domain, OLD.action, OLD.subject_type, OLD.subject_id,
         OLD.entity_definition_id, OLD.version_before, OLD.version_after, OLD.actor_principal_id, OLD.actor_system,
         OLD.source, OLD.source_ref, OLD.change_set_id, OLD.causation_id, OLD.causation_depth, OLD.request_id,
         OLD.backfilled, OLD.content_sha256, OLD.diff_sha256, OLD.diff_truncated);
BEGIN
    IF current_user = 'nava_worker' AND same_envelope
       AND OLD.sealed_at IS NULL AND OLD.seq IS NULL AND OLD.prev_hash IS NULL AND OLD.entry_hash IS NULL
       AND NEW.sealed_at IS NOT NULL AND NEW.seq IS NOT NULL AND NEW.prev_hash IS NOT NULL AND NEW.entry_hash IS NOT NULL
       AND NEW.diff::text IS NOT DISTINCT FROM OLD.diff::text AND NEW.diff_salt IS NOT DISTINCT FROM OLD.diff_salt
       AND (NEW.erased_at, NEW.erasure_ref) IS NOT DISTINCT FROM (OLD.erased_at, OLD.erasure_ref) THEN
        RETURN NEW; -- seal
    END IF;
    IF current_user = 'nava_erasure' AND same_envelope
       AND OLD.erased_at IS NULL AND OLD.diff IS NOT NULL
       AND NEW.diff IS NULL AND NEW.diff_salt IS NULL AND NEW.erased_at IS NOT NULL AND NEW.erasure_ref IS NOT NULL
       AND (NEW.seq, NEW.prev_hash, NEW.entry_hash, NEW.sealed_at)
           IS NOT DISTINCT FROM (OLD.seq, OLD.prev_hash, OLD.entry_hash, OLD.sealed_at) THEN
        RETURN NEW; -- erase
    END IF;
    RAISE EXCEPTION 'audit_log: update by % is neither the seal nor the erase transition', current_user
        USING ERRCODE = '42501';
END
$$;
-- +goose StatementEnd
CREATE TRIGGER audit_log_guard_update BEFORE UPDATE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_guard_update();

-- ---------------------------------------------------------------- audit_chain_head
-- R14: one row per tenant, the sealer's lock and the chain's tip. Inserted by
-- onboarding (P5-33, A6.2) and migration B (P5-32); neither has an INSERT
-- policy yet, so today nobody but those later tasks can create a head. id
-- is minted by the inserter (UUIDv7, no sequence: I8). Genesis entry_hash is
-- 32 zero bytes (R13).
CREATE TABLE audit_chain_head (
    tenant_id           uuid        NOT NULL,
    id                  uuid        NOT NULL,
    version             bigint      NOT NULL DEFAULT 1,
    seq                 bigint      NOT NULL DEFAULT 0,
    entry_hash          bytea       NOT NULL DEFAULT '\x0000000000000000000000000000000000000000000000000000000000000000',
    next_seal_outbox_id uuid,
    next_seal_at        timestamptz,
    seal_interval_s     integer     NOT NULL DEFAULT 10,
    export_lease_until  timestamptz,
    CONSTRAINT audit_chain_head_pkey PRIMARY KEY (tenant_id),
    CONSTRAINT audit_chain_head_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT audit_chain_head_version_check CHECK (version >= 1),
    CONSTRAINT audit_chain_head_seq_check CHECK (seq >= 0),
    CONSTRAINT audit_chain_head_entry_hash_check CHECK (octet_length(entry_hash) = 32),
    -- R16: 10 s after a run that sealed anything, doubling to 300 s when idle.
    CONSTRAINT audit_chain_head_seal_interval_check CHECK (seal_interval_s BETWEEN 10 AND 300)
);
ALTER TABLE audit_chain_head ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_chain_head FORCE ROW LEVEL SECURITY;

-- Column grants decide which head columns each role writes (R14 [Sec]).
CREATE POLICY audit_chain_head_select ON audit_chain_head FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY audit_chain_head_update ON audit_chain_head FOR UPDATE TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- ---------------------------------------------------------------- record_write_keys
-- D-5-3: the pipeline's idempotency key leaves the audit log. One row per
-- keyed write, inserted in the write's transaction; replay (P5-13a) reads it
-- and never the audit entry. The primary key is named because the store
-- matches the concurrent-duplicate 23505 on record_write_keys_pkey.
-- audit_entry_id is a soft pointer to the write's primary entry: audit_log's
-- key is (tenant_id, occurred_at, id), so a foreign key would need
-- occurred_at here for nothing replay uses. The record and definition keys
-- are composite foreign keys; records are never deleted in M1, so nothing
-- scans the referencing side and no extra index is needed (the replay
-- lookup is the primary key).
CREATE TABLE record_write_keys (
    tenant_id       uuid    NOT NULL,
    idempotency_key text    NOT NULL,
    definition_id   uuid    NOT NULL,
    action          text    NOT NULL,
    record_id       uuid    NOT NULL,
    number          text    NOT NULL,
    version         bigint  NOT NULL,
    schema_version  integer NOT NULL,
    audit_entry_id  uuid    NOT NULL,
    CONSTRAINT record_write_keys_pkey PRIMARY KEY (tenant_id, idempotency_key),
    CONSTRAINT record_write_keys_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT record_write_keys_definition_fkey FOREIGN KEY (tenant_id, definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT record_write_keys_record_fkey FOREIGN KEY (tenant_id, record_id)
        REFERENCES records (tenant_id, id),
    CONSTRAINT record_write_keys_idempotency_key_check CHECK (idempotency_key <> ''),
    CONSTRAINT record_write_keys_action_check CHECK (action IN ('create', 'update')),
    CONSTRAINT record_write_keys_version_check CHECK (version >= 0),
    CONSTRAINT record_write_keys_schema_version_check CHECK (schema_version >= 1)
);
ALTER TABLE record_write_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_write_keys FORCE ROW LEVEL SECURITY;

-- Insert-only: a key is claimed once and never changed or removed in M1.
CREATE POLICY record_write_keys_select ON record_write_keys FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY record_write_keys_insert ON record_write_keys FOR INSERT TO nava_app, nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
