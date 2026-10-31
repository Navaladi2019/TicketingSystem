-- RFC-005 / ADR-001-A5 spike: start-time cascade key, INSERT ... ON CONFLICT DO NOTHING
-- under FORCE ROW LEVEL SECURITY, run as a non-owner role. PostgreSQL 16.10.
\set ON_ERROR_STOP off
CREATE ROLE app NOLOGIN;
CREATE TABLE cascade_start (
    tenant_id           uuid        NOT NULL,
    causation_origin_id uuid        NOT NULL,
    record_id           uuid        NOT NULL,
    definition_id       uuid        NOT NULL,
    outbox_id           uuid        NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT cascade_start_pkey PRIMARY KEY (tenant_id, causation_origin_id, record_id, definition_id)
);
ALTER TABLE cascade_start ENABLE ROW LEVEL SECURITY;
ALTER TABLE cascade_start FORCE ROW LEVEL SECURITY;
CREATE POLICY cs_select ON cascade_start FOR SELECT TO app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY cs_insert ON cascade_start FOR INSERT TO app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT, INSERT ON cascade_start TO app;

-- The sink's statement: returns 1 row = start enqueued, 0 rows = cycle (suppress + audit).
PREPARE claim(uuid, uuid, uuid, uuid) AS
  INSERT INTO cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id)
  VALUES ($1, $2, $3, $4, gen_random_uuid())
  ON CONFLICT ON CONSTRAINT cascade_start_pkey DO NOTHING
  RETURNING 'START' AS outcome;

\set A '''aaaaaaaa-0000-0000-0000-000000000001'''
\set B '''bbbbbbbb-0000-0000-0000-000000000002'''
\set O '''00000000-0000-0000-0000-0000000000a0'''
\set R '''00000000-0000-0000-0000-0000000000b0'''
\set D '''00000000-0000-0000-0000-0000000000d1'''
\set D2 '''00000000-0000-0000-0000-0000000000d2'''

SET ROLE app;
BEGIN; SELECT set_config('app.tenant_id', :A, true);
\echo '=== case a: user creates R (origin O, depth 0) -> sink claims (O,R,D): expect START'
EXECUTE claim(:A, :O, :R, :D);
COMMIT;

BEGIN; SELECT set_config('app.tenant_id', :A, true);
\echo '=== case a/b/c: D writes back R under O (depth 1) -> sink claims (O,R,D) again: expect 0 rows (suppress, write commits)'
EXECUTE claim(:A, :O, :R, :D);
\echo '=== same again (second update_record step, or retried activity): expect 0 rows'
EXECUTE claim(:A, :O, :R, :D);
COMMIT;

BEGIN; SELECT set_config('app.tenant_id', :A, true);
\echo '=== mutual D1/D2: D writes R -> D2 claim: expect START; D2 writes R -> D claim: expect 0 rows; D writes again -> D2 claim: 0 rows'
EXECUTE claim(:A, :O, :R, :D2);
EXECUTE claim(:A, :O, :R, :D);
EXECUTE claim(:A, :O, :R, :D2);
COMMIT;

BEGIN; SELECT set_config('app.tenant_id', :A, true);
\echo '=== new user write starts a new origin O2: D starts again: expect START'
EXECUTE claim(:A, '00000000-0000-0000-0000-0000000000a2', :R, :D);
COMMIT;

BEGIN; SELECT set_config('app.tenant_id', :B, true);
\echo '=== case f: tenant B, same origin/record/definition uuids: expect START (no cross-tenant collision)'
EXECUTE claim(:B, :O, :R, :D);
\echo '=== case f: tenant B sees only its own rows: expect 1'
SELECT count(*) AS rows_visible_to_b FROM cascade_start;
\echo '=== tenant B claims a key in tenant A: expect ERROR new row violates row-level security policy'
EXECUTE claim(:A, :O, :R, '00000000-0000-0000-0000-0000000000d9');
ROLLBACK;

BEGIN; SELECT set_config('app.tenant_id', :A, true);
\echo '=== NULL origin: expect ERROR not-null violation (a NULL origin cannot bypass the key)'
EXECUTE claim(:A, NULL, :R, :D);
ROLLBACK;

BEGIN; SELECT set_config('app.tenant_id', :A, true);
\echo '=== rollback of the triggering write releases the key: claim (O3,R,D) then ROLLBACK, then claim again'
EXECUTE claim(:A, '00000000-0000-0000-0000-0000000000a3', :R, :D);
ROLLBACK;
BEGIN; SELECT set_config('app.tenant_id', :A, true);
EXECUTE claim(:A, '00000000-0000-0000-0000-0000000000a3', :R, :D);
COMMIT;
RESET ROLE;
\echo '=== final rows (superuser view)'
SELECT tenant_id, causation_origin_id, record_id, definition_id FROM cascade_start ORDER BY 1,2,4;
