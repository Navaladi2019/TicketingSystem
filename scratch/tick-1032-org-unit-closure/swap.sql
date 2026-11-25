-- TICK-1032: the records_org_unit swap (20261121060000) at size. Run by
-- swap.sh as the superuser on a throwaway postgres:16.10, autocommit, as the
-- NO TRANSACTION migration runs. records follows the TICK-947 bench shape
-- (scratch/tick-947-org-units/bench.sql): 1,000,000 rows over 400 tenants,
-- 400,000 in tenant 0, all with one of 200 units; 4 entity types, spread
-- independently of the unit (g / 7), so every unit holds all 4. A 300-byte data payload.
\set ON_ERROR_STOP on
\timing on
SET jit = off;
SELECT version();

CREATE TABLE records (
    tenant_id uuid NOT NULL, id uuid NOT NULL, entity_definition_id uuid NOT NULL,
    org_unit_id uuid, updated_at timestamptz NOT NULL, data jsonb NOT NULL,
    CONSTRAINT records_pkey PRIMARY KEY (tenant_id, id));
INSERT INTO records
SELECT CASE WHEN g <= 400000 THEN '00000000-0000-4000-8000-000000000000'::uuid
            ELSE ('00000000-0000-4000-8000-' || lpad(to_hex(1 + g % 399), 12, '0'))::uuid END,
       gen_random_uuid(),
       ('00000000-0000-4000-8000-0000000000e' || ((g / 7) % 4))::uuid,
       CASE WHEN g <= 400000 THEN ('00000000-0000-4000-8000-0000000' || lpad(to_hex(g % 200), 5, '0'))::uuid END,
       now() - g * interval '1 second',
       jsonb_build_object('short_description', repeat('x', 300))
FROM generate_series(1, 1000000) g;
CREATE INDEX records_org_unit ON records (tenant_id, org_unit_id) WHERE org_unit_id IS NOT NULL;
VACUUM ANALYZE records;
SELECT pg_size_pretty(pg_relation_size('records')) AS heap,
       pg_size_pretty(pg_relation_size('records_org_unit')) AS old_index;

\echo '== step 1: drop an INVALID leftover (none here)'
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('records_org_unit_new') AND NOT indisvalid) THEN
        DROP INDEX records_org_unit_new;
    END IF;
END
$$;
\echo '== step 2: CREATE INDEX CONCURRENTLY'
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_org_unit_new
    ON records (tenant_id, org_unit_id, entity_definition_id) WHERE org_unit_id IS NOT NULL;
\echo '== step 3: the swap (one transaction)'
DO $$
DECLARE
    target CONSTANT text := 'CREATE INDEX records_org_unit ON public.records USING btree (tenant_id, org_unit_id, entity_definition_id) WHERE (org_unit_id IS NOT NULL)';
BEGIN
    IF pg_get_indexdef(to_regclass('records_org_unit')) = target THEN
        DROP INDEX IF EXISTS records_org_unit_new;
        RETURN;
    END IF;
    PERFORM set_config('lock_timeout', '2s', true);
    DROP INDEX IF EXISTS records_org_unit;
    ALTER INDEX records_org_unit_new RENAME TO records_org_unit;
END
$$;
SELECT pg_get_indexdef('records_org_unit'::regclass), pg_size_pretty(pg_relation_size('records_org_unit')) AS new_index;

\echo '== the under leaf with an entity type, 20-unit set: new key'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT count(*) FROM records
WHERE tenant_id = '00000000-0000-4000-8000-000000000000'
  AND entity_definition_id = '00000000-0000-4000-8000-0000000000e1'
  AND org_unit_id IS NOT NULL
  AND org_unit_id = ANY (ARRAY(SELECT ('00000000-0000-4000-8000-0000000' || lpad(to_hex(u), 5, '0'))::uuid FROM generate_series(0, 19) u));
