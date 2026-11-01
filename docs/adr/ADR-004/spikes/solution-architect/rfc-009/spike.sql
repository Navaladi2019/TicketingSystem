-- RFC-009 spike: the ADR-004 R2 unique key and the commit check against the two cascade entries.
-- postgres:16.10, superuser (RLS is out of scope here: the question is the key and the trigger).
\pset pager off
SELECT version();
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE records (tenant_id uuid, id uuid, version int, PRIMARY KEY (tenant_id, id));
CREATE TABLE audit_log (
  tenant_id uuid NOT NULL, id uuid NOT NULL DEFAULT gen_random_uuid(),
  occurred_at timestamptz NOT NULL DEFAULT now(), txid xid8 NOT NULL DEFAULT pg_current_xact_id(),
  domain text NOT NULL,
  action text NOT NULL CHECK (action IN ('create','update','publish','enable','disable','delete','promote','rollback','state_change','export','erase','reseed','cascade_cycle','cascade_too_deep')),
  subject_type text NOT NULL CHECK (subject_type IN ('record','cascade_key')), subject_id uuid NOT NULL,
  version_before int, version_after int, causation_id uuid, causation_depth smallint, diff json,
  PRIMARY KEY (tenant_id, occurred_at, id),
  UNIQUE NULLS NOT DISTINCT (tenant_id, occurred_at, txid, subject_type, subject_id, version_after)
) PARTITION BY RANGE (occurred_at);
CREATE TABLE audit_log_default PARTITION OF audit_log DEFAULT;
CREATE FUNCTION commit_check() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE t uuid; s uuid; BEGIN
  IF TG_OP = 'DELETE' THEN t := OLD.tenant_id; s := OLD.id; ELSE t := NEW.tenant_id; s := NEW.id; END IF;
  IF NOT EXISTS (SELECT 1 FROM audit_log a WHERE a.tenant_id = t AND a.txid = pg_current_xact_id()
                 AND a.subject_type = 'record' AND a.subject_id = s) THEN
    RAISE EXCEPTION 'audit_missing for record %', s; END IF;
  RETURN NULL; END $$;
CREATE CONSTRAINT TRIGGER records_audit AFTER INSERT OR UPDATE OR DELETE ON records
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION commit_check();
-- RFC 4122 UUIDv5: SHA-1(namespace bytes || name bytes), version and variant bits set.
CREATE FUNCTION uuidv5(ns uuid, name bytea) RETURNS uuid LANGUAGE sql IMMUTABLE AS $$
  WITH h AS (SELECT substring(digest(uuid_send(ns) || name, 'sha1') FROM 1 FOR 16) AS b)
  SELECT encode(set_byte(set_byte(b, 6, (get_byte(b, 6) & 15) | 80), 8, (get_byte(b, 8) & 63) | 128), 'hex')::uuid FROM h $$;
CREATE FUNCTION ckey(t uuid, o uuid, r uuid, d uuid) RETURNS uuid LANGUAGE sql IMMUTABLE AS $$
  SELECT uuidv5('6ba7b812-9dad-11d1-80b4-00c04fd430c8', uuid_send(t) || uuid_send(o) || uuid_send(r) || uuid_send(d)) $$;

\set T '\'11111111-1111-1111-1111-111111111111\''
\set R '\'22222222-2222-2222-2222-222222222222\''
\set R2 '\'33333333-3333-3333-3333-333333333333\''
\set O '\'44444444-4444-4444-4444-444444444444\''
\set D1 '\'55555555-5555-5555-5555-555555555555\''
\set D2 '\'66666666-6666-6666-6666-666666666666\''

BEGIN; INSERT INTO records VALUES (:T,:R,1),(:T,:R2,1);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after)
  VALUES (:T,'record','create','record',:R,1),(:T,'record','create','record',:R2,1); COMMIT;

\echo '== Case A (rejected): cycle entry subject = record R, version_after = the update entry''s version'
BEGIN; UPDATE records SET version=2 WHERE tenant_id=:T AND id=:R;
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after) VALUES (:T,'record','update','record',:R,2);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after) VALUES (:T,'record','cascade_cycle','record',:R,2);
ROLLBACK;

\echo '== Case B (rejected): subject = record R, version_after NULL, D1 and D2 both suppressed on one write'
BEGIN; UPDATE records SET version=2 WHERE tenant_id=:T AND id=:R;
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after) VALUES (:T,'record','update','record',:R,2);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after) VALUES (:T,'record','cascade_cycle','record',:R,NULL);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after) VALUES (:T,'record','cascade_cycle','record',:R,NULL);
ROLLBACK;

\echo '== Case C (rejected): subject = record R lets a cycle entry alone pass the commit check (update entry forgotten)'
BEGIN; UPDATE records SET version=2 WHERE tenant_id=:T AND id=:R;
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after) VALUES (:T,'record','cascade_cycle','record',:R,NULL);
COMMIT;
SELECT 'C committed with no update entry, R version=' || version AS case_c FROM records WHERE id=:R;
-- reset C
ALTER TABLE records DISABLE TRIGGER records_audit; UPDATE records SET version=1 WHERE id=:R; ALTER TABLE records ENABLE TRIGGER records_audit;
DELETE FROM audit_log WHERE action='cascade_cycle';

\echo '== Case D (chosen): cascade_key subjects. One tx: R and R2 both written to v2; D1 and D2 suppressed on R, D1 on R2'
BEGIN;
UPDATE records SET version=2 WHERE tenant_id=:T AND id IN (:R,:R2);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after)
  VALUES (:T,'record','update','record',:R,2),(:T,'record','update','record',:R2,2);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after,causation_id) VALUES
 (:T,'record','cascade_cycle','cascade_key',ckey(:T,:O,:R,:D1),2,:O),
 (:T,'record','cascade_cycle','cascade_key',ckey(:T,:O,:R,:D2),2,:O),
 (:T,'record','cascade_cycle','cascade_key',ckey(:T,:O,:R2,:D1),2,:O);
COMMIT;

\echo '== Case E (chosen): R written twice in one tx (v3, v4), D1 suppressed after each: same key, distinct version_after'
BEGIN;
UPDATE records SET version=3 WHERE tenant_id=:T AND id=:R;
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after) VALUES (:T,'record','update','record',:R,3);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after,causation_id) VALUES (:T,'record','cascade_cycle','cascade_key',ckey(:T,:O,:R,:D1),3,:O);
UPDATE records SET version=4 WHERE tenant_id=:T AND id=:R;
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after) VALUES (:T,'record','update','record',:R,4);
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after,causation_id) VALUES (:T,'record','cascade_cycle','cascade_key',ckey(:T,:O,:R,:D1),4,:O);
COMMIT;

\echo '== Case F (chosen): a cascade_key entry alone does NOT pass the commit check for R (expect audit_missing)'
BEGIN; UPDATE records SET version=5 WHERE tenant_id=:T AND id=:R;
INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_after) VALUES (:T,'record','cascade_cycle','cascade_key',ckey(:T,:O,:R,:D1),5);
COMMIT;

\echo '== Case G (chosen): cascade_too_deep on the runner after the caller rolled back; subject record R, version_after NULL'
BEGIN; UPDATE records SET version=5 WHERE tenant_id=:T AND id=:R; ROLLBACK;
BEGIN; INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_before,version_after,causation_id,causation_depth)
  VALUES (:T,'record','cascade_too_deep','record',:R,4,NULL,:O,9); COMMIT;
\echo '== Case G2: a re-run of the rejected activity audits again in its own runner tx'
BEGIN; INSERT INTO audit_log(tenant_id,domain,action,subject_type,subject_id,version_before,version_after,causation_id,causation_depth)
  VALUES (:T,'record','cascade_too_deep','record',:R,4,NULL,:O,9); COMMIT;

\echo '== Case H: the cascade key is tenant-scoped: same O, R, D in another tenant gives a different subject_id'
SELECT ckey(:T,:O,:R,:D1) <> ckey('77777777-7777-7777-7777-777777777777',:O,:R,:D1) AS keys_differ_across_tenants;

\echo '== Result'
SELECT action, subject_type, left(subject_id::text,8) AS subj, version_before AS vb, version_after AS va, causation_depth AS depth
  FROM audit_log ORDER BY occurred_at, action, subject_id;
SELECT left(id::text,8) AS record, version FROM records ORDER BY id;
