\set ON_ERROR_STOP 0
\timing off
-- Reduced audit_log: same partitioning and same action CHECK as 20261102233600.
CREATE TABLE audit_log (
  tenant_id uuid NOT NULL, id uuid NOT NULL DEFAULT gen_random_uuid(),
  occurred_at timestamptz NOT NULL, action text NOT NULL, pad text,
  CONSTRAINT audit_log_action_check CHECK (action IN (
    'create','update','publish','enable','disable','delete','promote','rollback',
    'state_change','export','erase','reseed','cascade_cycle','cascade_too_deep'))
) PARTITION BY RANGE (occurred_at);
CREATE TABLE audit_log_default PARTITION OF audit_log DEFAULT;
DO $$ BEGIN FOR m IN 0..11 LOOP
  EXECUTE format('CREATE TABLE audit_log_%s PARTITION OF audit_log FOR VALUES FROM (%L) TO (%L)',
    to_char(date '2026-01-01' + make_interval(months=>m),'YYYY_MM'),
    date '2026-01-01' + make_interval(months=>m), date '2026-01-01' + make_interval(months=>m+1));
END LOOP; END $$;
INSERT INTO audit_log (tenant_id, occurred_at, action, pad)
SELECT gen_random_uuid(), timestamptz '2026-01-01' + (g * interval '1 second' * 31.5),
       (ARRAY['create','update','publish','delete'])[1 + g % 4], repeat('x', 400)
FROM generate_series(1, 1000000) g;
VACUUM ANALYZE audit_log;
SELECT 'rows', count(*), pg_size_pretty(sum(pg_total_relation_size(c.oid))) FROM audit_log, (SELECT inhrelid oid FROM pg_inherits WHERE inhparent='audit_log'::regclass) c GROUP BY 1 LIMIT 1;

\echo '== T1 before: insert number_space_exhausted fails'
INSERT INTO audit_log VALUES (gen_random_uuid(), DEFAULT, '2026-06-01', 'number_space_exhausted');

\echo '== T2 migration A (one tx, lock_timeout 5s): drop + add NOT VALID; locks held'
\timing on
BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE audit_log DROP CONSTRAINT audit_log_action_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_action_check CHECK (action IN (
    'create','update','publish','enable','disable','delete','promote','rollback',
    'state_change','export','erase','reseed','cascade_cycle','cascade_too_deep','number_space_exhausted')) NOT VALID;
\timing off
SELECT relation::regclass, mode FROM pg_locks WHERE locktype='relation' AND pid=pg_backend_pid()
  AND relation::regclass::text LIKE 'audit_log%' ORDER BY 1::text, 2 LIMIT 6;
SELECT count(*) AS rels_access_exclusive FROM pg_locks WHERE locktype='relation' AND pid=pg_backend_pid() AND mode='AccessExclusiveLock' AND relation::regclass::text LIKE 'audit_log%';
COMMIT;

\echo '== T3 after A: catalogue state, parent + every partition NOT VALID'
SELECT conrelid::regclass, convalidated, conislocal, coninhcount FROM pg_constraint
 WHERE conname='audit_log_action_check' ORDER BY conrelid::regclass::text LIMIT 4;
SELECT convalidated, count(*) FROM pg_constraint WHERE conname='audit_log_action_check' GROUP BY 1;

\echo '== T4 after A: new action insert works, unknown action still refused (new rows checked)'
INSERT INTO audit_log VALUES (gen_random_uuid(), DEFAULT, '2026-06-01', 'number_space_exhausted');
INSERT INTO audit_log VALUES (gen_random_uuid(), DEFAULT, '2026-06-01', 'bogus');

\echo '== T5 partition created while NOT VALID'
CREATE TABLE audit_log_2027_01 PARTITION OF audit_log FOR VALUES FROM ('2027-01-01') TO ('2027-02-01');
SELECT conrelid::regclass, convalidated FROM pg_constraint WHERE conname='audit_log_action_check' AND conrelid='audit_log_2027_01'::regclass;

\echo '== T6 ATTACH of a table carrying the OLD narrower check under the same name'
CREATE TABLE audit_log_2027_02 (LIKE audit_log);
ALTER TABLE audit_log_2027_02 ADD CONSTRAINT audit_log_action_check CHECK (action IN ('create'));
ALTER TABLE audit_log ATTACH PARTITION audit_log_2027_02 FOR VALUES FROM ('2027-02-01') TO ('2027-03-01');
\echo '== T6b ATTACH of a table with no action check'
CREATE TABLE audit_log_2027_03 (LIKE audit_log);
ALTER TABLE audit_log ATTACH PARTITION audit_log_2027_03 FOR VALUES FROM ('2027-03-01') TO ('2027-04-01');
SELECT conrelid::regclass, convalidated FROM pg_constraint WHERE conname='audit_log_action_check' AND conrelid='audit_log_2027_03'::regclass;

\echo '== T7 migration B: VALIDATE (lock mode + time)'
BEGIN;
SET LOCAL lock_timeout = '5s';
\timing on
ALTER TABLE audit_log VALIDATE CONSTRAINT audit_log_action_check;
\timing off
SELECT mode, count(*) FROM pg_locks WHERE locktype='relation' AND pid=pg_backend_pid() AND relation::regclass::text LIKE 'audit_log%' GROUP BY 1;
COMMIT;
SELECT convalidated, count(*) FROM pg_constraint WHERE conname='audit_log_action_check' GROUP BY 1;

\echo '== T8 VALIDATE refuses a bad row hidden in one partition'
ALTER TABLE audit_log DROP CONSTRAINT audit_log_action_check;
INSERT INTO audit_log VALUES (gen_random_uuid(), DEFAULT, '2026-03-01', 'bogus');
ALTER TABLE audit_log ADD CONSTRAINT audit_log_action_check CHECK (action IN ('create','update','publish','delete','number_space_exhausted')) NOT VALID;
ALTER TABLE audit_log VALIDATE CONSTRAINT audit_log_action_check;
SELECT convalidated, count(*) FROM pg_constraint WHERE conname='audit_log_action_check' GROUP BY 1;

\echo '== T9 baseline: plain ADD (validating) time for comparison'
DELETE FROM audit_log WHERE action='bogus';
ALTER TABLE audit_log DROP CONSTRAINT audit_log_action_check;
\timing on
ALTER TABLE audit_log ADD CONSTRAINT audit_log_action_check CHECK (action IN ('create','update','publish','delete','number_space_exhausted'));
\timing off
