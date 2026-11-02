-- Item 4: the R10 BEFORE INSERT assert trigger on PARTITION BY RANGE (occurred_at)
-- with one DEFAULT partition. Every case prints expected vs actual; the last
-- query prints PASS only if every case matched. Run as postgres on an empty
-- audit_log (before seed.sql). Leaves no rows and no extra partitions behind.
\set ON_ERROR_STOP on
\pset footer off
CREATE TEMP TABLE r4 (n serial, name text, expect text, actual text);
GRANT ALL ON r4 TO PUBLIC; GRANT ALL ON SEQUENCE r4_n_seq TO PUBLIC;

-- Runs p_sql as p_role inside a subtransaction; records ok or the error.
CREATE FUNCTION pg_temp.t(p_name text, p_role text, p_expect text, p_sql text) RETURNS void LANGUAGE plpgsql AS $$
DECLARE v text;
BEGIN
    BEGIN
        EXECUTE format('SET LOCAL ROLE %I', p_role);
        EXECUTE replace(replace(p_sql, '$C',
            'tenant_id, id, domain, action, subject_type, subject_id, actor_system, source'), '$V',
            '''11111111-1111-1111-1111-111111111111'', gen_random_uuid(), ''record'', ''update'', ''record'', gen_random_uuid(), ''system'', ''system''');
        EXECUTE 'RESET ROLE';
        v := 'ok';
    EXCEPTION WHEN OTHERS THEN
        v := 'error ' || SQLSTATE || ': ' || SQLERRM;
    END;
    INSERT INTO r4 (name, expect, actual) VALUES (p_name, p_expect, v);
END $$;

BEGIN;
SELECT set_config('app.tenant_id', '11111111-1111-1111-1111-111111111111', true);
\set cols 'tenant_id, id, domain, action, subject_type, subject_id, actor_system, source'
\set vals '''11111111-1111-1111-1111-111111111111'', gen_random_uuid(), ''record'', ''update'', ''record'', gen_random_uuid(), ''system'', ''system'''

SELECT pg_temp.t('01 nava_app insert, defaults fill occurred_at/txid', 'nava_app', 'ok',
  $q$INSERT INTO audit_log ($C) VALUES ($V)$q$);
SELECT pg_temp.t('02 nava_app second statement, same txn (now() is txn-stable)', 'nava_app', 'ok',
  $q$INSERT INTO audit_log ($C) SELECT $V FROM pg_sleep(0.02)$q$);
SELECT pg_temp.t('03 nava_app names occurred_at: column grant denies', 'nava_app', 'error 42501: permission denied',
  $q$INSERT INTO audit_log ($C, occurred_at) VALUES ($V, now())$q$);
SELECT pg_temp.t('04 full-grant role, occurred_at = now() explicitly', 'spike_fullgrant', 'ok',
  $q$INSERT INTO audit_log ($C, occurred_at) VALUES ($V, now())$q$);
SELECT pg_temp.t('05 full-grant role, occurred_at backdated 1 day', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, occurred_at) VALUES ($V, now() - interval '1 day')$q$);
SELECT pg_temp.t('06 full-grant role, occurred_at = clock_timestamp()', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, occurred_at) VALUES ($V, clock_timestamp())$q$);
SELECT pg_temp.t('07 full-grant role, foreign txid', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, txid) VALUES ($V, '42'::xid8)$q$);
SELECT pg_temp.t('08 full-grant role, backfilled = true', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, backfilled) VALUES ($V, true)$q$);
SELECT pg_temp.t('09 full-grant role, seq set', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, seq) VALUES ($V, 1)$q$);
SELECT pg_temp.t('10 full-grant role, prev_hash set', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, prev_hash) VALUES ($V, '\x00')$q$);
SELECT pg_temp.t('11 full-grant role, entry_hash set', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, entry_hash) VALUES ($V, '\x00')$q$);
SELECT pg_temp.t('12 full-grant role, sealed_at set', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, sealed_at) VALUES ($V, now())$q$);
SELECT pg_temp.t('13 full-grant role, erased_at set', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, erased_at) VALUES ($V, now())$q$);
SELECT pg_temp.t('14 full-grant role, erasure_ref set', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, erasure_ref) VALUES ($V, gen_random_uuid())$q$);
SELECT pg_temp.t('15 full-grant role, multi-row insert with one bad row: whole statement fails', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, occurred_at) VALUES ($V, now()), ($V, now() - interval '1 hour')$q$);
SELECT pg_temp.t('16 full-grant role, direct insert into the DEFAULT partition, backdated', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log_default ($C, occurred_at) VALUES ($V, now() - interval '1 day')$q$);
SELECT pg_temp.t('17 superuser (not nava_migrator), backdated: not exempt', 'postgres', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, occurred_at) VALUES ($V, now() - interval '1 day')$q$);
SELECT pg_temp.t('18 nava_migrator backfill, backdated + backfilled', 'nava_migrator', 'ok',
  $q$INSERT INTO audit_log ($C, occurred_at, txid, backfilled) VALUES ($V, '2026-01-15 10:00:00+00', '7'::xid8, true)$q$);
SELECT pg_temp.t('19 INSERT .. SELECT by full-grant role, backdated', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log ($C, occurred_at) SELECT $V, now() - interval '1 minute' FROM generate_series(1,3)$q$);

-- Never rewrites: the migrator row is stored exactly as given, in DEFAULT.
INSERT INTO r4 (name, expect, actual)
SELECT '20 migrator row stored as given (no rewrite), routed to DEFAULT', 'audit_log_default 2026-01-15 10:00:00+00 7 true',
       string_agg(tableoid::regclass || ' ' || occurred_at || ' ' || txid || ' ' || backfilled, ',')
  FROM audit_log WHERE backfilled;
INSERT INTO r4 (name, expect, actual)
SELECT '21 live rows: occurred_at = now(), txid = this txn, in DEFAULT', '3 rows, all match',
       count(*) || ' rows, ' || CASE WHEN bool_and(occurred_at = now() AND txid = pg_current_xact_id() AND tableoid = 'audit_log_default'::regclass) THEN 'all match' ELSE 'MISMATCH' END
  FROM audit_log WHERE NOT backfilled;
INSERT INTO r4 (name, expect, actual)
SELECT '22 trigger is cloned onto the DEFAULT partition', 'audit_log, audit_log_default',
       string_agg(tgrelid::regclass::text, ', ' ORDER BY tgrelid::regclass::text)
  FROM pg_trigger WHERE tgname = 'audit_log_assert_insert';
DELETE FROM audit_log; -- cleanup as owner (no role has DELETE); keep r4
COMMIT;

-- Epic 30 adds range partitions later. The trigger must reach them too, both
-- CREATE .. PARTITION OF and ATTACH PARTITION, and a rewrite trigger (what R10
-- forbids) must fail to move a row across partitions on PG 16.
BEGIN;
SELECT set_config('app.tenant_id', '11111111-1111-1111-1111-111111111111', true);
CREATE TABLE audit_log_2026_01 PARTITION OF audit_log FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE audit_log_2026_02 (LIKE audit_log INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
ALTER TABLE audit_log ATTACH PARTITION audit_log_2026_02 FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
GRANT INSERT ON audit_log_2026_01, audit_log_2026_02 TO spike_fullgrant;
INSERT INTO r4 (name, expect, actual)
SELECT '23 trigger cloned onto CREATE .. PARTITION OF and ATTACH PARTITION', 'audit_log, audit_log_2026_01, audit_log_2026_02, audit_log_default',
       string_agg(tgrelid::regclass::text, ', ' ORDER BY tgrelid::regclass::text)
  FROM pg_trigger WHERE tgname = 'audit_log_assert_insert';
SELECT pg_temp.t('24 migrator backfill routes to the range partition', 'nava_migrator', 'ok',
  $q$INSERT INTO audit_log ($C, occurred_at, backfilled) VALUES ($V, '2026-01-15', true)$q$);
SELECT pg_temp.t('25 full-grant role, direct insert into the created partition', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log_2026_01 ($C, occurred_at) VALUES ($V, '2026-01-20')$q$);
SELECT pg_temp.t('26 full-grant role, direct insert into the attached partition', 'spike_fullgrant', 'error 42501: audit_log: protected',
  $q$INSERT INTO audit_log_2026_02 ($C, occurred_at) VALUES ($V, '2026-02-20')$q$);
INSERT INTO r4 (name, expect, actual)
SELECT '27 backfilled row landed in audit_log_2026_01', 'audit_log_2026_01', string_agg(tableoid::regclass::text, ',') FROM audit_log WHERE backfilled;
-- Contrast: a rewriting BEFORE trigger (rejected design).
CREATE FUNCTION pg_temp.rewrite() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.occurred_at := now(); RETURN NEW; END $$;
CREATE TRIGGER zz_rewrite BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION pg_temp.rewrite();
SELECT pg_temp.t('28 contrast: a rewrite trigger moving a row out of its partition fails (PG 16)', 'nava_migrator', 'error 0A000',
  $q$INSERT INTO audit_log ($C, occurred_at, backfilled) VALUES ($V, '2026-01-16', true)$q$);
DROP TRIGGER zz_rewrite ON audit_log;
DELETE FROM audit_log;
DROP TABLE audit_log_2026_01, audit_log_2026_02;
COMMIT;

-- COPY: RLS roles cannot COPY FROM at all (PG refuses); the superuser can, and
-- the trigger still fires for it.
\set ON_ERROR_STOP off
SET ROLE spike_fullgrant;
\copy audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source, occurred_at) from '/spike/sql/item4_copy.tsv'
RESET ROLE;
INSERT INTO r4 (name, expect, actual) VALUES ('29 COPY FROM by an RLS role is refused outright', 'error 0A000: COPY FROM not supported with row-level security',
  'error ' || :'LAST_ERROR_SQLSTATE' || ': ' || :'LAST_ERROR_MESSAGE');
\set ON_ERROR_STOP off
\copy audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source, occurred_at) from '/spike/sql/item4_copy.tsv'
\set ON_ERROR_STOP on
INSERT INTO r4 (name, expect, actual) VALUES ('30 COPY by superuser, backdated: trigger raises (error printed above), 0 rows', '0 rows', (SELECT count(*) || ' rows' FROM audit_log));

\echo
\echo == item 4 cases
SELECT n, name, expect, actual,
       CASE WHEN actual = expect OR (expect LIKE 'error %' AND actual LIKE expect || '%') THEN 'match' ELSE 'MISMATCH' END AS verdict
  FROM r4 ORDER BY n;
SELECT CASE WHEN bool_and(actual = expect OR (expect LIKE 'error %' AND actual LIKE expect || '%')) THEN 'ITEM 4: PASS' ELSE 'ITEM 4: FAIL' END AS result,
       count(*) AS cases FROM r4;
SELECT count(*) AS rows_left_in_audit_log FROM audit_log;
