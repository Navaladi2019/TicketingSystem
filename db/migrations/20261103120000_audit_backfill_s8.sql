-- P5-15 (TICK-433): migration A's backfill (ADR-004 R25 A half, R26). Every
-- entity_definition_versions row written by S8 before audit_log existed gets
-- exactly one backfilled = true entry in its own tenant's log. P5-32
-- (migration B) re-runs the same function after the deploy; the NOT EXISTS
-- dedupe makes the second run add only versions published in between
-- (CI-A9).
--
-- The entry (R25):
--   id              UUIDv5(audit_backfill_namespace, tenant_id ‖ version_id),
--                   the 32 raw bytes in that order. The namespace is fixed
--                   here, 3767d3ad-0a89-4b41-847c-9e4bf30c2055, and never
--                   changes: a re-run must mint the same id.
--   domain          config; subject entity_definition / definition_id;
--                   entity_definition_id = definition_id.
--   action          create for version 1, publish otherwise.
--   version_before  NULL for version 1, else version - 1; version_after
--                   = version.
--   actor, source   published_by and source user; else (only version 1 may
--                   have no publisher: onboarding seeded it before the
--                   tenant had a principal, TICK-383) actor_system onboarding
--                   and source onboarding.
--   content_sha256  schema_sha256; occurred_at = published_at;
--                   backfilled = true. No diff: R3 makes the content hash the
--                   config entry's evidence, and the versions row is the
--                   content.
--
-- Access path (R26). The function, run by nava_migrator inside this
-- migration's transaction (no NO TRANSACTION directive):
--   1. creates four temporary policies: the one cross-tenant policy,
--      FOR SELECT TO nava_migrator USING (true) on tenants, to list ids; and
--      canonical tenant policies on entity_definition_versions (SELECT) and
--      audit_log (SELECT for the dedupe, INSERT);
--   2. per tenant, set_config('app.tenant_id', id, true), then one
--      INSERT ... SELECT, which reads and writes that tenant's rows only;
--   3. drops all four policies before it returns.
-- An error anywhere rolls the whole transaction back, the policies with it,
-- so no migrator policy outlives the run either way (R26.3; the migration
-- asserts it at the end and TestP515NoMigratorPolicyLeft re-checks it).
--
-- UUIDv5 needs SHA-1. ADR-004-E1 rules out pgcrypto in migrations and
-- PostgreSQL 16 has no built-in SHA-1, so audit_backfill_sha1 is FIPS 180-4
-- SHA-1 in plpgsql. TestP515UUIDv5MatchesGo checks it against Go's
-- uuid.NewSHA1 and the FIPS vectors. All three functions are plain SECURITY
-- INVOKER, owned by nava_migrator, with no EXECUTE grant (00001 revokes the
-- PUBLIC default): nothing but the migrator can run them, and running them
-- gives no privilege the caller lacks.
--
-- Locks. CREATE POLICY takes ACCESS EXCLUSIVE on tenants,
-- entity_definition_versions and audit_log until COMMIT, so tenant
-- resolution and S8 reads wait for the whole run. Measured on PG 16.10
-- (scratch/audit-backfill-s8/RESULT.md): 5.6 s for 30,000 versions across
-- 1,000 tenants, 54.7 s for 300,000 across 10,000, linear at about
-- 0.18 ms per version; P5-32's re-run with every entry present takes 0.14 s
-- and 2.3 s. lock_timeout makes the migration fail and be re-run instead of
-- queueing behind a long transaction to take the locks.
-- Design and evidence: docs/data/audit-log.md.

-- +goose Up

SET LOCAL lock_timeout = '5s';

-- FIPS 180-4 SHA-1 over msg. 32-bit words are held in bigint and masked.
-- +goose StatementBegin
CREATE FUNCTION audit_backfill_sha1(msg bytea) RETURNS bytea
    LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SECURITY INVOKER SET search_path = pg_catalog AS $$
DECLARE
    m32 constant bigint := 4294967295; -- 0xFFFFFFFF
    h0 bigint := 1732584193;  -- 0x67452301
    h1 bigint := 4023233417;  -- 0xEFCDAB89
    h2 bigint := 2562383102;  -- 0x98BADCFE
    h3 bigint := 271733878;   -- 0x10325476
    h4 bigint := 3285377520;  -- 0xC3D2E1F0
    p bytea;
    w bigint[];
    a bigint; b bigint; c bigint; d bigint; e bigint; t bigint;
    o integer;
BEGIN
    -- Pad: 0x80, zeros to 56 mod 64, then the bit length as 64-bit big-endian.
    p := msg || '\x80'::bytea;
    p := p || decode(repeat('00', (119 - octet_length(msg) % 64) % 64), 'hex');
    p := p || int8send(octet_length(msg)::bigint * 8);
    FOR blk IN 0 .. octet_length(p) / 64 - 1 LOOP
        w := array_fill(0::bigint, ARRAY[80]);
        FOR i IN 0 .. 15 LOOP
            o := blk * 64 + i * 4;
            w[i + 1] := (get_byte(p, o)::bigint << 24) | (get_byte(p, o + 1)::bigint << 16)
                      | (get_byte(p, o + 2)::bigint << 8) | get_byte(p, o + 3)::bigint;
        END LOOP;
        FOR i IN 16 .. 79 LOOP
            t := w[i - 2] # w[i - 7] # w[i - 13] # w[i - 15];
            w[i + 1] := ((t << 1) | (t >> 31)) & m32;
        END LOOP;
        a := h0; b := h1; c := h2; d := h3; e := h4;
        -- Four loops, one per round function, so no round tests i.
        FOR i IN 0 .. 19 LOOP
            t := ((((a << 5) | (a >> 27)) & m32) + ((b & c) | ((b # m32) & d)) + e + 1518500249 + w[i + 1]) & m32; -- K = 0x5A827999
            e := d; d := c; c := ((b << 30) | (b >> 2)) & m32; b := a; a := t;
        END LOOP;
        FOR i IN 20 .. 39 LOOP
            t := ((((a << 5) | (a >> 27)) & m32) + (b # c # d) + e + 1859775393 + w[i + 1]) & m32; -- K = 0x6ED9EBA1
            e := d; d := c; c := ((b << 30) | (b >> 2)) & m32; b := a; a := t;
        END LOOP;
        FOR i IN 40 .. 59 LOOP
            t := ((((a << 5) | (a >> 27)) & m32) + ((b & c) | (b & d) | (c & d)) + e + 2400959708 + w[i + 1]) & m32; -- K = 0x8F1BBCDC
            e := d; d := c; c := ((b << 30) | (b >> 2)) & m32; b := a; a := t;
        END LOOP;
        FOR i IN 60 .. 79 LOOP
            t := ((((a << 5) | (a >> 27)) & m32) + (b # c # d) + e + 3395469782 + w[i + 1]) & m32; -- K = 0xCA62C1D6
            e := d; d := c; c := ((b << 30) | (b >> 2)) & m32; b := a; a := t;
        END LOOP;
        h0 := (h0 + a) & m32;
        h1 := (h1 + b) & m32;
        h2 := (h2 + c) & m32;
        h3 := (h3 + d) & m32;
        h4 := (h4 + e) & m32;
    END LOOP;
    RETURN substring(int8send(h0) FROM 5) || substring(int8send(h1) FROM 5) || substring(int8send(h2) FROM 5)
        || substring(int8send(h3) FROM 5) || substring(int8send(h4) FROM 5);
END
$$;
-- +goose StatementEnd

-- RFC 9562 UUIDv5 of the 32 bytes tenant_id ‖ version_id under the backfill
-- namespace: Go's uuid.NewSHA1(ns, tenant[:] ‖ version[:]).
-- +goose StatementBegin
CREATE FUNCTION audit_backfill_id(tenant_id uuid, version_id uuid) RETURNS uuid
    LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE SECURITY INVOKER SET search_path = pg_catalog, public AS $$
DECLARE
    h bytea := substring(audit_backfill_sha1(
        uuid_send('3767d3ad-0a89-4b41-847c-9e4bf30c2055'::uuid) || uuid_send(tenant_id) || uuid_send(version_id))
        FROM 1 FOR 16);
BEGIN
    h := set_byte(h, 6, (get_byte(h, 6) & 15) | 80);  -- version 5
    h := set_byte(h, 8, (get_byte(h, 8) & 63) | 128); -- RFC 9562 variant
    RETURN encode(h, 'hex')::uuid;
END
$$;
-- +goose StatementEnd

-- The backfill. Returns the number of entries it added. Only nava_migrator
-- may run it: it creates policies, which needs the table owner, and R10
-- lets only the migrator set occurred_at and backfilled.
-- enable_seqscan = off, for this function only: audit_log's statistics say
-- empty while the run fills it, so the cached plan seq-scanned
-- audit_log_default for every tenant's dedupe probe, O(n^2): 291.7 s for
-- 300,000 versions, 54.7 s with the probe on audit_log_default_subject.
-- Every query here has an index for its predicate.
-- +goose StatementBegin
CREATE FUNCTION audit_backfill_s8() RETURNS bigint
    LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog, public
    SET enable_seqscan = off AS $$
DECLARE
    tid uuid;
    added bigint := 0;
    n bigint;
BEGIN
    IF current_user <> 'nava_migrator' THEN
        RAISE EXCEPTION 'audit_backfill_s8: only nava_migrator may run the backfill, not %', current_user
            USING ERRCODE = '42501';
    END IF;

    CREATE POLICY audit_backfill_tenants ON tenants FOR SELECT TO nava_migrator
        USING (true);
    CREATE POLICY audit_backfill_versions ON entity_definition_versions FOR SELECT TO nava_migrator
        USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
    CREATE POLICY audit_backfill_log_select ON audit_log FOR SELECT TO nava_migrator
        USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
    CREATE POLICY audit_backfill_log_insert ON audit_log FOR INSERT TO nava_migrator
        WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

    FOR tid IN SELECT id FROM tenants ORDER BY id LOOP
        PERFORM set_config('app.tenant_id', tid::text, true);
        -- The tenant_id = tid predicate is defence in depth: the canonical
        -- SELECT policy already limits v to this tenant, and the INSERT
        -- policy refuses any row whose tenant_id is not this tenant.
        INSERT INTO audit_log (
            tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id,
            version_before, version_after, actor_principal_id, actor_system, source, backfilled, content_sha256)
        SELECT v.tenant_id,
               audit_backfill_id(v.tenant_id, v.id),
               v.published_at,
               'config',
               CASE WHEN v.version = 1 THEN 'create' ELSE 'publish' END,
               'entity_definition',
               v.definition_id,
               v.definition_id,
               CASE WHEN v.version = 1 THEN NULL ELSE v.version - 1 END,
               v.version,
               v.published_by,
               CASE WHEN v.published_by IS NULL THEN 'onboarding' END,
               CASE WHEN v.published_by IS NULL THEN 'onboarding' ELSE 'user' END,
               true,
               v.schema_sha256
        FROM entity_definition_versions v
        WHERE v.tenant_id = tid
          AND NOT EXISTS (
              SELECT 1 FROM audit_log l
              WHERE l.tenant_id = v.tenant_id
                AND l.subject_type = 'entity_definition'
                AND l.subject_id = v.definition_id
                AND l.version_after = v.version);
        GET DIAGNOSTICS n = ROW_COUNT;
        added := added + n;
    END LOOP;
    PERFORM set_config('app.tenant_id', '', true);

    DROP POLICY audit_backfill_log_insert ON audit_log;
    DROP POLICY audit_backfill_log_select ON audit_log;
    DROP POLICY audit_backfill_versions ON entity_definition_versions;
    DROP POLICY audit_backfill_tenants ON tenants;
    RETURN added;
END
$$;
-- +goose StatementEnd

SELECT audit_backfill_s8();

-- R26.3: no nava_migrator policy outlives the migration.
-- +goose StatementBegin
DO $$
DECLARE
    left_over text;
BEGIN
    SELECT string_agg(format('%s.%I', polrelid::regclass, polname), ', ') INTO left_over
    FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles);
    IF left_over IS NOT NULL THEN
        RAISE EXCEPTION 'audit backfill: nava_migrator policies left over: %', left_over;
    END IF;
END
$$;
-- +goose StatementEnd
