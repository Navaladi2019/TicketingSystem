-- TICK-795 (for TICK-672, ADR-005 §3.6): the tenant admin route reads and
-- sets its own tenant's time_zone as nava_app. tenants is under FORCE RLS
-- and nava_app had only tenants_tenant (FOR SELECT), so an UPDATE matched no
-- row. This policy lets nava_app update its own tenant's row and no other;
-- the column grant in db/grants.yaml (UPDATE (time_zone) only) keeps every
-- other column, state included, out of reach.
--
-- USING and WITH CHECK are the canonical predicate on the tenants key (id),
-- so CI-3 holds and the lookup is the tenants_pkey probe. With no tenant, or
-- an empty one, NULLIF gives NULL and the UPDATE matches nothing.
--
-- Locks: CREATE POLICY takes ACCESS EXCLUSIVE on tenants for a catalogue
-- change only (no scan, no rewrite); lock_timeout bounds the wait behind
-- tenant reads.

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE POLICY tenants_time_zone_update ON tenants FOR UPDATE TO nava_app
    USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose Down

SET LOCAL lock_timeout = '5s';

DROP POLICY tenants_time_zone_update ON tenants;
