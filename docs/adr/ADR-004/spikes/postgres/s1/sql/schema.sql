-- Spike S1 (TICK-423) schema: ADR-004 R1, R2, R9-R12, R14 and R28, records-shaped.
-- Run as the superuser `postgres`, which owns every table. The app roles are
-- not superusers, so FORCE RLS applies to them.
\set ON_ERROR_STOP on
CREATE EXTENSION IF NOT EXISTS pg_walinspect;

CREATE ROLE nava_app      LOGIN PASSWORD 'spike';
CREATE ROLE nava_worker   LOGIN PASSWORD 'spike';
CREATE ROLE nava_migrator LOGIN PASSWORD 'spike';
CREATE ROLE nava_erasure  LOGIN PASSWORD 'spike';
-- Spike-only stand-ins: a reader for item 5, and a role with a full INSERT grant
-- for item 4 (to prove the trigger, not the column grant, stops it).
CREATE ROLE spike_export  LOGIN PASSWORD 'spike';
CREATE ROLE spike_fullgrant LOGIN PASSWORD 'spike';

-- ---------------------------------------------------------------- records
-- PLAN-002 shape: promoted columns plus data jsonb.
CREATE TABLE records (
    tenant_id            uuid        NOT NULL,
    id                   uuid        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    number               bigint      NOT NULL,
    version              bigint      NOT NULL DEFAULT 0,
    schema_version       integer     NOT NULL DEFAULT 1,
    data                 jsonb       NOT NULL,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT records_pkey PRIMARY KEY (tenant_id, id)
);
CREATE INDEX records_def_updated ON records (tenant_id, entity_definition_id, updated_at DESC);
ALTER TABLE records ENABLE ROW LEVEL SECURITY;
ALTER TABLE records FORCE ROW LEVEL SECURITY;
CREATE POLICY records_select ON records FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY records_insert ON records FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY records_update ON records FOR UPDATE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT, INSERT, UPDATE ON records TO nava_app;

-- ---------------------------------------------------------------- audit_log (R1, R2, R28)
CREATE TABLE audit_log (
    tenant_id            uuid        NOT NULL,
    id                   uuid        NOT NULL,
    occurred_at          timestamptz NOT NULL DEFAULT now(),
    txid                 xid8        NOT NULL DEFAULT pg_current_xact_id(),
    domain               text        NOT NULL CHECK (domain IN ('record','config','audit')),
    action               text        NOT NULL CHECK (action IN ('create','update','publish','enable','disable','delete','promote','rollback','state_change','export','erase','reseed')),
    subject_type         text        NOT NULL CHECK (subject_type IN ('record','entity_definition','erasure_request','audit_export','audit_chain')),
    subject_id           uuid        NOT NULL,
    entity_definition_id uuid,
    version_before       bigint,
    version_after        bigint,
    actor_principal_id   uuid,
    actor_system         text,
    source               text        NOT NULL CHECK (source IN ('user','ai','change_set','onboarding','plugin','rule','workflow','function','system')),
    source_ref           text,
    change_set_id        uuid,
    causation_id         uuid,
    causation_depth      smallint    NOT NULL DEFAULT 0,
    request_id           text,
    backfilled           boolean     NOT NULL DEFAULT false,
    content_sha256       bytea,
    diff_sha256          bytea,
    diff                 json,
    diff_truncated       boolean     NOT NULL DEFAULT false,
    diff_salt            bytea,
    seq                  bigint,
    prev_hash            bytea,
    entry_hash           bytea,
    sealed_at            timestamptz,
    erased_at            timestamptz,
    erasure_ref          uuid,
    CONSTRAINT audit_log_actor_one CHECK ((actor_principal_id IS NULL) <> (actor_system IS NULL)),
    CONSTRAINT audit_log_pkey PRIMARY KEY (tenant_id, occurred_at, id)
) PARTITION BY RANGE (occurred_at);
CREATE TABLE audit_log_default PARTITION OF audit_log DEFAULT;

-- R28 index set.
CREATE INDEX audit_log_tenant_time    ON audit_log (tenant_id, occurred_at DESC, id DESC);
CREATE INDEX audit_log_subject        ON audit_log (tenant_id, subject_type, subject_id, occurred_at DESC);
CREATE INDEX audit_log_change_set     ON audit_log (tenant_id, change_set_id) WHERE change_set_id IS NOT NULL;
CREATE INDEX audit_log_seq            ON audit_log (tenant_id, seq) WHERE seq IS NOT NULL;
CREATE INDEX audit_log_unsealed       ON audit_log (tenant_id, occurred_at, id) WHERE sealed_at IS NULL;
CREATE UNIQUE INDEX audit_log_commit_key ON audit_log
    (tenant_id, occurred_at, txid, subject_type, subject_id, version_after) NULLS NOT DISTINCT;

-- Item 2 fix (S1 finding F1). Without it the commit check's cached plan probes
-- audit_log_pkey on (tenant_id, occurred_at) only: every entry of a transaction
-- shares that prefix, so a transaction of n subjects costs O(n^2) at COMMIT.
-- Telling the planner occurred_at values repeat makes the 5-column R2 key (or the
-- R28 subject index) strictly cheaper. Not inherited: set on every partition.
-- item2.sh toggles it (FIX=off / FIX=on).
ALTER TABLE audit_log ALTER COLUMN occurred_at SET (n_distinct = -0.0001);
ALTER TABLE audit_log_default ALTER COLUMN occurred_at SET (n_distinct = -0.0001);

ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
CREATE POLICY audit_log_select ON audit_log FOR SELECT TO nava_app, nava_worker, spike_export, spike_fullgrant, nava_migrator
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY audit_log_insert ON audit_log FOR INSERT TO nava_app, spike_fullgrant, nava_migrator
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY audit_log_update ON audit_log FOR UPDATE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- R10 / R12 column grants for a writer: INSERT excludes occurred_at, txid,
-- backfilled, the seal columns and the erasure columns.
GRANT INSERT (tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id,
              version_before, version_after, actor_principal_id, actor_system, source, source_ref,
              change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256,
              diff, diff_truncated, diff_salt) ON audit_log TO nava_app;
GRANT SELECT (tenant_id, occurred_at, txid, subject_type, subject_id, version_after) ON audit_log TO nava_app;
GRANT SELECT, UPDATE (seq, prev_hash, entry_hash, sealed_at) ON audit_log TO nava_worker;
GRANT SELECT, INSERT ON audit_log TO nava_migrator, spike_fullgrant;
GRANT SELECT ON audit_log TO spike_export;
GRANT INSERT ON audit_log_default TO spike_fullgrant; -- item 4: direct partition insert

-- ---------------------------------------------------------------- R10 assert (BEFORE INSERT)
CREATE FUNCTION audit_log_assert_insert() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF current_user <> 'nava_migrator' THEN
        IF NEW.occurred_at IS DISTINCT FROM now()
           OR NEW.txid IS DISTINCT FROM pg_current_xact_id()
           OR NEW.backfilled IS DISTINCT FROM false
           OR NEW.seq IS NOT NULL OR NEW.prev_hash IS NOT NULL OR NEW.entry_hash IS NOT NULL OR NEW.sealed_at IS NOT NULL
           OR NEW.erased_at IS NOT NULL OR NEW.erasure_ref IS NOT NULL THEN
            RAISE EXCEPTION 'audit_log: protected insert column set by %', current_user
                USING ERRCODE = '42501';
        END IF;
    END IF;
    RETURN NEW; -- unchanged: assert, never rewrite
END $$;
CREATE TRIGGER audit_log_assert_insert BEFORE INSERT ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_assert_insert();

-- ---------------------------------------------------------------- R11 (BEFORE UPDATE): seal or erase only
CREATE FUNCTION audit_log_guard_update() RETURNS trigger LANGUAGE plpgsql AS $$
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
       AND NEW.erased_at IS NULL AND NEW.erasure_ref IS NULL THEN
        RETURN NEW; -- seal
    END IF;
    IF current_user = 'nava_erasure' AND same_envelope
       AND OLD.erased_at IS NULL AND OLD.diff IS NOT NULL
       AND NEW.diff IS NULL AND NEW.diff_salt IS NULL AND NEW.erased_at IS NOT NULL AND NEW.erasure_ref IS NOT NULL
       AND (NEW.seq, NEW.prev_hash, NEW.entry_hash, NEW.sealed_at) IS NOT DISTINCT FROM (OLD.seq, OLD.prev_hash, OLD.entry_hash, OLD.sealed_at) THEN
        RETURN NEW; -- erase
    END IF;
    RAISE EXCEPTION 'audit_log: update outside the two R11 transitions by %', current_user USING ERRCODE = '42501';
END $$;
CREATE TRIGGER audit_log_guard_update BEFORE UPDATE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_guard_update();

-- ---------------------------------------------------------------- R14 chain head
CREATE TABLE audit_chain_head (
    tenant_id           uuid        NOT NULL PRIMARY KEY,
    id                  uuid        NOT NULL DEFAULT gen_random_uuid(),
    version             bigint      NOT NULL DEFAULT 1,
    seq                 bigint      NOT NULL DEFAULT 0,
    entry_hash          bytea       NOT NULL DEFAULT '\x0000000000000000000000000000000000000000000000000000000000000000',
    next_seal_outbox_id uuid,
    next_seal_at        timestamptz,
    seal_interval_s     integer     NOT NULL DEFAULT 10,
    export_lease_until  timestamptz
);
ALTER TABLE audit_chain_head ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_chain_head FORCE ROW LEVEL SECURITY;
CREATE POLICY head_select ON audit_chain_head FOR SELECT TO nava_worker, spike_export
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY head_update ON audit_chain_head FOR UPDATE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT ON audit_chain_head TO nava_worker, spike_export;
GRANT UPDATE (version, seq, entry_hash, next_seal_outbox_id, next_seal_at, seal_interval_s) ON audit_chain_head TO nava_worker;

-- ---------------------------------------------------------------- R2 commit check (decision 2)
-- P5-11's pinned probe (TICK-449): full R2-key prefix.
CREATE FUNCTION audit_commit_check() RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER AS $$
DECLARE
    v_tenant uuid;
    v_id     uuid;
BEGIN
    IF TG_OP = 'DELETE' THEN v_tenant := OLD.tenant_id; v_id := OLD.id;
    ELSE v_tenant := NEW.tenant_id; v_id := NEW.id; END IF;
    PERFORM 1 FROM audit_log
     WHERE tenant_id = v_tenant AND occurred_at = now() AND txid = pg_current_xact_id()
       AND subject_type = TG_ARGV[0] AND subject_id = v_id
     LIMIT 1;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'audit_missing: % % written without an audit entry in this transaction', TG_ARGV[0], v_id
            USING ERRCODE = '23514';
    END IF;
    RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER records_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF version, data ON records
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('record');

-- ---------------------------------------------------------------- spike sealer (R15/R16 A2 stand-in)
-- One batch, one transaction (call it in autocommit). Hashes are computed in SQL
-- here; the real sealer hashes in Go. WAL is the same either way.
CREATE PROCEDURE spike_seal_batch(p_tenant uuid, p_n integer DEFAULT 1000) LANGUAGE plpgsql AS $$
DECLARE
    h    record;
    e    record;
    s    bigint;
    prev bytea;
BEGIN
    PERFORM set_config('app.tenant_id', p_tenant::text, true);
    SELECT seq, entry_hash INTO h FROM audit_chain_head WHERE tenant_id = p_tenant FOR UPDATE;
    s := h.seq; prev := h.entry_hash;
    FOR e IN SELECT occurred_at, id, content_sha256 FROM audit_log
              WHERE tenant_id = p_tenant AND sealed_at IS NULL
              ORDER BY occurred_at, id LIMIT p_n LOOP
        s := s + 1;
        UPDATE audit_log SET seq = s, prev_hash = prev,
               entry_hash = sha256(prev || coalesce(e.content_sha256, '\x'::bytea)), sealed_at = now()
         WHERE tenant_id = p_tenant AND occurred_at = e.occurred_at AND id = e.id;
        prev := sha256(prev || coalesce(e.content_sha256, '\x'::bytea));
    END LOOP;
    UPDATE audit_chain_head SET seq = s, entry_hash = prev, version = version + 1 WHERE tenant_id = p_tenant;
END $$;
GRANT EXECUTE ON PROCEDURE spike_seal_batch(uuid, integer) TO nava_worker;
