-- P10-10 (TICK-925, folded into TICK-916; ADR-009 D15): the rollout. Records
-- written before ADR-009 lack the defaults their type's current version
-- declares, and nothing would ever write them. This migration queues, once at
-- deploy, one schema backfill job per tenant and per entity type whose
-- current version has any top-level default: a schema_backfills row
-- (pending, target_version = the current version, default_keys = every
-- top-level default key of it, causation_audit_id NULL) and its outbox start
-- row. The job (P10-05) does the writing, through the I4 pipeline.
--
-- It writes no data and never reads records: it reads tenants,
-- entity_definitions and entity_definition_versions only, so it costs one
-- small INSERT pair per defaulted type, whatever the record count.
--
-- It skips any type whose current version has default + x-nava-unique on the
-- same property (D13 refuses that at the next publish): a backfill would only
-- produce unique conflicts. Each skip is a NOTICE with ids only, and a final
-- NOTICE counts both. The Technical Lead reads them from the deploy log
-- (PLAN-010 D-10-7).
--
-- The outbox row (contract C3, as outbox_payload_check allows): kind
-- schema.backfill, dedupe_key schema-backfill:<entity_definition_id>:<target_version>,
-- target_key schema.backfill:<own id>, target_seq = target_version, causation
-- its own id at depth 0, payload {"entity_definition_id"}. The payload holds
-- UUIDs only (outbox_payload_check, db/migrations/00009): target_version is
-- in target_seq and the dedupe key, and causation_audit_id, NULL here, is on
-- the job row, as EnqueueFieldIndexBackfill does for requested_version.
--
-- Tenant scoping: the one place allowed to loop over tenants (D15). Per
-- tenant it sets app.tenant_id transaction-locally and names the tenant in
-- every predicate; every inserted row's tenant_id is the entity definition's
-- (CI-3 (7)). The nava_migrator policies it needs (FORCE RLS applies to the
-- owner) are created at the start and dropped at the end of this transaction,
-- and the closing check fails the migration if one is left (R26.3).
--
-- Re-running it inserts nothing new: the job's primary key and the outbox
-- dedupe key both conflict, and the outbox row is written only when the job
-- row was.

-- +goose Up

SET LOCAL lock_timeout = '5s';

-- RFC 9562 UUIDv7 for the outbox ids, as audit_seed_uuidv7 (20261107053707)
-- builds them. Dropped below.
-- +goose StatementBegin
CREATE FUNCTION schema_backfill_rollout_uuidv7() RETURNS uuid
    LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog AS $$
DECLARE
    b bytea := uuid_send(gen_random_uuid());
BEGIN
    b := overlay(b PLACING substring(int8send(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3)
                 FROM 1 FOR 6);
    b := set_byte(b, 6, (get_byte(b, 6) & 15) | 112); -- version 7
    RETURN encode(b, 'hex')::uuid;
END
$$;
-- +goose StatementEnd

CREATE POLICY sbr_tenants ON tenants FOR SELECT TO nava_migrator
    USING (true);
CREATE POLICY sbr_definitions ON entity_definitions FOR SELECT TO nava_migrator
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY sbr_versions ON entity_definition_versions FOR SELECT TO nava_migrator
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
-- ON CONFLICT reads the conflicting row, so both inserted tables need SELECT
-- too, for the same tenant.
CREATE POLICY sbr_jobs_select ON schema_backfills FOR SELECT TO nava_migrator
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY sbr_outbox_select ON outbox FOR SELECT TO nava_migrator
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
           AND kind = 'schema.backfill');
CREATE POLICY sbr_jobs_insert ON schema_backfills FOR INSERT TO nava_migrator
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY sbr_outbox_insert ON outbox FOR INSERT TO nava_migrator
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND kind = 'schema.backfill');

-- +goose StatementBegin
DO $$
DECLARE
    tid uuid;
    t record;
    n bigint;
    queued bigint := 0;
    skipped bigint := 0;
    oid uuid;
BEGIN
    FOR tid IN SELECT id FROM tenants ORDER BY id LOOP
        PERFORM set_config('app.tenant_id', tid::text, true);
        FOR t IN
            SELECT d.id AS definition_id, d.current_version AS version,
                   array_agg(p.key ORDER BY p.key) AS default_keys,
                   bool_or(p.value -> 'x-nava-unique' = 'true'::jsonb) AS unique_default
            FROM entity_definitions d
            JOIN entity_definition_versions v
              ON v.tenant_id = d.tenant_id AND v.definition_id = d.id AND v.version = d.current_version
            CROSS JOIN LATERAL jsonb_each(
                CASE WHEN jsonb_typeof(v.schema::jsonb -> 'properties') = 'object'
                     THEN v.schema::jsonb -> 'properties' ELSE '{}'::jsonb END) p
            WHERE d.tenant_id = tid AND v.tenant_id = tid
              AND jsonb_typeof(p.value) = 'object' AND p.value ? 'default'
            GROUP BY d.id, d.current_version
            ORDER BY d.id
        LOOP
            IF t.unique_default THEN
                skipped := skipped + 1;
                RAISE NOTICE 'schema_backfill_skipped tenant=% type=%', tid, t.definition_id;
                CONTINUE;
            END IF;
            INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, state, default_keys)
            VALUES (tid, t.definition_id, t.version, 'pending', t.default_keys)
            ON CONFLICT ON CONSTRAINT schema_backfills_pkey DO NOTHING;
            GET DIAGNOSTICS n = ROW_COUNT;
            IF n = 0 THEN
                CONTINUE;
            END IF;
            oid := schema_backfill_rollout_uuidv7();
            INSERT INTO outbox (id, tenant_id, kind, kind_version, dedupe_key, target_key, target_seq,
                                causation_id, causation_depth, payload)
            VALUES (oid, tid, 'schema.backfill', 1,
                    'schema-backfill:' || t.definition_id::text || ':' || t.version::text,
                    'schema.backfill:' || oid::text, t.version, oid, 0,
                    jsonb_build_object('entity_definition_id', t.definition_id::text))
            ON CONFLICT ON CONSTRAINT outbox_dedupe_key DO NOTHING;
            queued := queued + 1;
        END LOOP;
    END LOOP;
    PERFORM set_config('app.tenant_id', '', true);
    RAISE NOTICE 'schema_backfill_rollout queued=% skipped=%', queued, skipped;
END
$$;
-- +goose StatementEnd

DROP POLICY sbr_outbox_insert ON outbox;
DROP POLICY sbr_outbox_select ON outbox;
DROP POLICY sbr_jobs_insert ON schema_backfills;
DROP POLICY sbr_jobs_select ON schema_backfills;
DROP POLICY sbr_versions ON entity_definition_versions;
DROP POLICY sbr_definitions ON entity_definitions;
DROP POLICY sbr_tenants ON tenants;
DROP FUNCTION schema_backfill_rollout_uuidv7();

-- R26.3: no nava_migrator policy outlives the migration.
-- +goose StatementBegin
DO $$
DECLARE
    left_over text;
BEGIN
    SELECT string_agg(format('%s.%I', polrelid::regclass, polname), ', ') INTO left_over
    FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles);
    IF left_over IS NOT NULL THEN
        RAISE EXCEPTION 'schema_backfill rollout: nava_migrator policies left over: %', left_over;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down

-- The queued jobs are data, not schema: a rollback of the tables
-- (20261127110300) drops them. Nothing to undo here.
SELECT 1;
