-- Write cost per takeover, as pgadmin (the cost does not depend on the role):
-- 2,000 takeovers of distinct abandoned in_flight rows, one transaction each,
-- with the new conditional UPDATE and with T22's FOR UPDATE, measured in WAL
-- bytes and HOT updates. Each pass runs on its own set of rows.
\pset pager off
CREATE TEMP TABLE pick AS
SELECT tenant_id, principal_id, key, row_number() OVER (ORDER BY key) AS n
  FROM idempotency_keys WHERE state = 'in_flight' AND lease_until < now() LIMIT 4000;
SELECT count(*) AS picked FROM pick;
SELECT pg_stat_reset_single_table_counters('idempotency_keys'::regclass);

-- before: T22, SELECT ... FOR UPDATE (the lock is a tuple header write)
SELECT pg_current_wal_insert_lsn() AS l0 \gset
CREATE PROCEDURE pg_temp.lock_pass() LANGUAGE plpgsql AS $$
DECLARE r record;
BEGIN
  FOR r IN SELECT * FROM pick WHERE n <= 2000 LOOP
    PERFORM 1 FROM idempotency_keys
     WHERE tenant_id = r.tenant_id AND principal_id = r.principal_id AND key = r.key FOR UPDATE;
    COMMIT;
  END LOOP;
END $$;
\timing on
CALL pg_temp.lock_pass();
\timing off
SELECT pg_wal_lsn_diff(pg_current_wal_insert_lsn(), :'l0') / 2000 AS wal_bytes_per_takeover_for_update;

-- after: T22a, the conditional UPDATE of lease_until
SELECT pg_current_wal_insert_lsn() AS l1 \gset
CREATE PROCEDURE pg_temp.lease_pass() LANGUAGE plpgsql AS $$
DECLARE r record;
BEGIN
  FOR r IN SELECT * FROM pick WHERE n > 2000 LOOP
    UPDATE idempotency_keys SET lease_until = now() + interval '60 seconds'
     WHERE tenant_id = r.tenant_id AND principal_id = r.principal_id AND key = r.key
       AND state = 'in_flight' AND lease_until < now();
    COMMIT;
  END LOOP;
END $$;
\timing on
CALL pg_temp.lease_pass();
\timing off
SELECT pg_wal_lsn_diff(pg_current_wal_insert_lsn(), :'l1') / 2000 AS wal_bytes_per_takeover_lease;
SELECT pg_sleep(1);
SELECT n_tup_upd, n_tup_hot_upd FROM pg_stat_user_tables WHERE relname = 'idempotency_keys';
