-- RFC-009 follow-up spike (Postgres & Data Modeling Specialist): key shape and diff fallback index.
-- postgres:16.10, superuser, default config (shared_buffers 128MB). Schema = RFC-009 spike + R1 diff (json) + R28 indexes.
\pset pager off
\timing off
SELECT version();
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE records (tenant_id uuid, id uuid, version int, PRIMARY KEY (tenant_id, id));
CREATE TABLE audit_log (
  tenant_id uuid NOT NULL, id uuid NOT NULL DEFAULT gen_random_uuid(),
  occurred_at timestamptz NOT NULL DEFAULT now(), txid xid8 NOT NULL DEFAULT pg_current_xact_id(),
  domain text NOT NULL, action text NOT NULL, subject_type text NOT NULL CHECK (subject_type IN ('record','cascade_key')),
  subject_id uuid NOT NULL, version_before int, version_after int, causation_id uuid, causation_depth smallint, diff json,
  PRIMARY KEY (tenant_id, occurred_at, id),
  UNIQUE NULLS NOT DISTINCT (tenant_id, occurred_at, txid, subject_type, subject_id, version_after)
) PARTITION BY RANGE (occurred_at);
CREATE TABLE audit_log_default PARTITION OF audit_log DEFAULT;
CREATE INDEX audit_hist ON audit_log (tenant_id, subject_type, subject_id, occurred_at DESC);   -- R28
CREATE INDEX audit_time ON audit_log (tenant_id, occurred_at DESC, id DESC);                     -- R28
CREATE FUNCTION uuidv5(ns uuid, name bytea) RETURNS uuid LANGUAGE sql IMMUTABLE AS $$
  WITH h AS (SELECT substring(digest(uuid_send(ns) || name, 'sha1') FROM 1 FOR 16) AS b)
  SELECT encode(set_byte(set_byte(b, 6, (get_byte(b, 6) & 15) | 80), 8, (get_byte(b, 8) & 63) | 128), 'hex')::uuid FROM h $$;
CREATE FUNCTION ckey(t uuid, o uuid, r uuid, d uuid) RETURNS uuid LANGUAGE sql IMMUTABLE AS $$
  SELECT uuidv5('6ba7b812-9dad-11d1-80b4-00c04fd430c8', uuid_send(t) || uuid_send(o) || uuid_send(r) || uuid_send(d)) $$;
CREATE FUNCTION cdiff(o uuid, r uuid, d uuid) RETURNS json LANGUAGE sql IMMUTABLE AS $$
  SELECT json_build_array(json_build_object('op','add','path','/cascade','value',
         json_build_object('origin_id',o,'record_id',r,'definition_id',d,'reason','ErrCascadeCycle'))) $$;

\set T '\'11111111-1111-1111-1111-111111111111\''
\set R '\'22222222-2222-2222-2222-222222222222\''
\set O '\'44444444-4444-4444-4444-444444444444\''
\set D1 '\'55555555-5555-5555-5555-555555555555\''
INSERT INTO records VALUES (:T,:R,1);

\echo '== X0: uuid_send byte order is the textual (RFC 4122 network) order, same as Go uuid.UUID[:]'
SELECT encode(uuid_send('00112233-4455-6677-8899-aabbccddeeff'),'hex') AS uuid_send_hex;

\echo '== X1 (not in RFC spike): the same (O,R,D) suppressed twice on ONE write: same key, same version_after'
BEGIN; UPDATE records SET version=2 WHERE tenant_id=:T AND id=:R;
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after) VALUES (:T,'record','update','record',:R,2);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after,causation_id) VALUES (:T,'record','cascade_cycle','cascade_key',ckey(:T,:O,:R,:D1),2,:O);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after,causation_id) VALUES (:T,'record','cascade_cycle','cascade_key',ckey(:T,:O,:R,:D1),2,:O);
ROLLBACK;
SELECT 'X1: the user write is lost with it, R version=' || version AS x1 FROM records WHERE id=:R;

\echo '== X2 (not in RFC spike): two cascade_too_deep entries for R in ONE runner tx (version_after NULL)'
BEGIN;
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_before,version_after,causation_id,causation_depth) VALUES (:T,'record','cascade_too_deep','record',:R,1,NULL,:O,9);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_before,version_after,causation_id,causation_depth) VALUES (:T,'record','cascade_too_deep','record',:R,1,NULL,:O,9);
ROLLBACK;

\echo '== X3: json -> and ->> operators are IMMUTABLE (so an expression index on the json diff is allowed)'
SELECT DISTINCT p.proname, p.provolatile FROM pg_operator o JOIN pg_proc p ON p.oid=o.oprcode
 WHERE o.oprname IN ('->','->>') AND o.oprleft='json'::regtype ORDER BY 1;

\echo '== Load: 10 tenants, 1,600,000 update rows + 400,000 cascade_cycle rows (json diff), 400,000 records'
INSERT INTO audit_log(tenant_id,occurred_at,txid,domain,action,subject_type,subject_id,version_after,diff)
SELECT t, ts, (g)::text::xid8, 'record','update','record', r, g, '[{"op":"replace","path":"/data/status","value":"open"}]'::json
FROM (SELECT g, ('00000000-0000-4000-8000-0000000000' || lpad(to_hex(g % 10),2,'0'))::uuid t,
             ('00000000-0000-4000-9000-' || lpad(to_hex(g % 400000),12,'0'))::uuid r,
             timestamptz '2026-01-01' + g * interval '7 seconds' ts FROM generate_series(1,1600000) g) s;
INSERT INTO audit_log(tenant_id,occurred_at,txid,domain,action,subject_type,subject_id,version_after,causation_id,diff)
SELECT t, ts, (g)::text::xid8, 'record','cascade_cycle','cascade_key', ckey(t,o,r,d), g, o, cdiff(o,r,d)
FROM (SELECT g, ('00000000-0000-4000-8000-0000000000' || lpad(to_hex(g % 10),2,'0'))::uuid t,
             ('00000000-0000-4000-9000-' || lpad(to_hex(g % 400000),12,'0'))::uuid r,
             ('00000000-0000-4000-a000-' || lpad(to_hex(g % 50000),12,'0'))::uuid o,
             ('00000000-0000-4000-b000-' || lpad(to_hex(g % 200),12,'0'))::uuid d,
             timestamptz '2026-01-01' + g * interval '28 seconds' + interval '1 second' ts FROM generate_series(1,400000) g) s;
-- make record RR hot: 40 cycle rows over time (a busy "on update" record)
INSERT INTO audit_log(tenant_id,occurred_at,txid,domain,action,subject_type,subject_id,version_after,causation_id,diff)
SELECT t, timestamptz '2026-03-01' + g * interval '1 hour', (2000000+g)::text::xid8, 'record','cascade_cycle','cascade_key',
       ckey(t,o,r,d), 1000+g, o, cdiff(o,r,d)
FROM (SELECT g, '00000000-0000-4000-8000-000000000002'::uuid t, '00000000-0000-4000-9000-000000000002'::uuid r,
             '00000000-0000-4000-a000-000000000002'::uuid o, '00000000-0000-4000-b000-000000000002'::uuid d
      FROM generate_series(1,40) g) s;
VACUUM ANALYZE audit_log;
SELECT count(*) FILTER (WHERE action='update') updates, count(*) FILTER (WHERE action='cascade_cycle') cycles FROM audit_log;
\set TT '\'00000000-0000-4000-8000-000000000002\''
\set RR '\'00000000-0000-4000-9000-000000000002\''
\set OO '\'00000000-0000-4000-a000-000000000002\''

\echo '== B1 before: cycle rows for record RR (by diff), R28 indexes only'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id, occurred_at FROM audit_log
 WHERE tenant_id=:TT AND action='cascade_cycle' AND (diff->0->'value'->>'record_id')::uuid = :RR ORDER BY occurred_at DESC LIMIT 50;

\echo '== B2: filter by origin through causation_id (RFC-009 point 5). R28 has no causation_id index'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM audit_log WHERE tenant_id=:TT AND causation_id=:OO;

\echo '== Fallback index (partial, cycle rows only). CONCURRENTLY on the partitioned parent:'
CREATE INDEX CONCURRENTLY audit_cycle_record_bad ON audit_log (tenant_id, ((diff->0->'value'->>'record_id')::uuid), occurred_at DESC) WHERE action='cascade_cycle';
\timing on
CREATE INDEX audit_cycle_record ON audit_log (tenant_id, ((diff->0->'value'->>'record_id')::uuid), occurred_at DESC) WHERE action='cascade_cycle';
\timing off
ANALYZE audit_log;
SELECT c.relname, pg_size_pretty(pg_relation_size(c.oid)) AS size FROM pg_class c JOIN pg_index i ON i.indexrelid=c.oid
 WHERE i.indrelid='audit_log_default'::regclass ORDER BY 1;
SELECT pg_size_pretty(pg_relation_size('audit_log_default')) AS heap;

\echo '== B1 after'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id, occurred_at FROM audit_log
 WHERE tenant_id=:TT AND action='cascade_cycle' AND (diff->0->'value'->>'record_id')::uuid = :RR ORDER BY occurred_at DESC LIMIT 50;

\echo '== X4: erasure nulls diff (R23): the row drops out of the fallback index; the recomputed key still finds it'
UPDATE audit_log SET diff=NULL WHERE tenant_id=:TT AND action='cascade_cycle' AND occurred_at = timestamptz '2026-03-01' + interval '1 hour';
SELECT count(*) AS findable_by_diff FROM audit_log WHERE tenant_id=:TT AND action='cascade_cycle' AND (diff->0->'value'->>'record_id')::uuid = :RR;
SELECT count(*) AS findable_by_recomputed_key FROM audit_log WHERE tenant_id=:TT AND subject_type='cascade_key'
  AND subject_id = ckey(:TT, :OO, :RR, '00000000-0000-4000-b000-000000000002');

\echo '== W: write cost of the fallback index, 100,000 cycle-row inserts (3 runs each, rolled back)'
CREATE FUNCTION load_cycles() RETURNS void LANGUAGE sql AS $f$
INSERT INTO audit_log(tenant_id,occurred_at,txid,domain,action,subject_type,subject_id,version_after,causation_id,diff)
SELECT t, ts, (3000000+g)::text::xid8, 'record','cascade_cycle','cascade_key', ckey(t,o,r,d), g, o, cdiff(o,r,d)
FROM (SELECT g, ('00000000-0000-4000-8000-0000000000' || lpad(to_hex(g % 10),2,'0'))::uuid t,
             ('00000000-0000-4000-9000-' || lpad(to_hex(g % 400000),12,'0'))::uuid r,
             ('00000000-0000-4000-a000-' || lpad(to_hex(g % 50000),12,'0'))::uuid o,
             ('00000000-0000-4000-b000-' || lpad(to_hex(g % 200),12,'0'))::uuid d,
             timestamptz '2026-06-01' + g * interval '3 seconds' ts FROM generate_series(1,100000) g) s $f$;
\timing on
\echo 'with index'
BEGIN; SELECT load_cycles(); ROLLBACK;
BEGIN; SELECT load_cycles(); ROLLBACK;
BEGIN; SELECT load_cycles(); ROLLBACK;
DROP INDEX audit_cycle_record;
\echo 'without index'
BEGIN; SELECT load_cycles(); ROLLBACK;
BEGIN; SELECT load_cycles(); ROLLBACK;
BEGIN; SELECT load_cycles(); ROLLBACK;
\timing off
