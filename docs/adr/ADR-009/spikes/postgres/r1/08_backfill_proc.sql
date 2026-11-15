-- Server-side lower bound of one OpMaterialize batch: the statements
-- pipeline.update sends per record (store.Get, PayloadJSON before, the
-- versioned UPDATE, PayloadJSON after, audit INSERT), D4 trigger live.
-- No network round trips: real hold time is longer (see README).
CREATE OR REPLACE PROCEDURE backfill(batch int, lockmode text, pause_ms int DEFAULT 0)
LANGUAGE plpgsql AS $$
DECLARE cur uuid := '00000000-0000-0000-0000-000000000000'; ids uuid[]; vs bigint[]; i int; rec records; b text; a text;
  t0 timestamptz; lsn0 pg_lsn; nb int := 0; nrec bigint := 0; maxb interval := interval '0'; d interval; tb timestamptz;
  tid uuid := '00000001-0000-7000-8000-000000000001'; did uuid := '00000002-0000-7000-8000-000000000001';
BEGIN
  lsn0 := pg_current_wal_insert_lsn(); t0 := clock_timestamp();
  LOOP
    PERFORM set_config('app.tenant_id', tid::text, true);
    tb := clock_timestamp();
    EXECUTE format('SELECT array_agg(id ORDER BY id), array_agg(version ORDER BY id) FROM (SELECT id, version FROM records
       WHERE tenant_id = $1 AND entity_definition_id = $2 AND schema_version < 3 AND id > $3 ORDER BY id LIMIT $4 %s SKIP LOCKED) s', lockmode)
      INTO ids, vs USING tid, did, cur, batch;
    EXIT WHEN ids IS NULL;
    FOR i IN 1 .. array_length(ids, 1) LOOP
      SELECT * INTO rec FROM records WHERE tenant_id = tid AND id = ids[i];
      CONTINUE WHEN rec.data ? 'priority';           -- D3 no-op skip
      SELECT data::text INTO b FROM records WHERE tenant_id = tid AND id = ids[i] AND version = vs[i];
      UPDATE records SET data = data || '{"priority":3}', schema_version = 3, version = version + 1, updated_at = now()
       WHERE tenant_id = tid AND id = ids[i] AND version = vs[i];
      CONTINUE WHEN NOT FOUND;                         -- conflict skip
      SELECT data::text INTO a FROM records WHERE tenant_id = tid AND id = ids[i];
      INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after,
        actor_system, source, source_ref, causation_depth, diff_sha256, diff, diff_salt)
      VALUES (tid, gen_random_uuid(), 'record', 'update', 'record', ids[i], did, vs[i], vs[i] + 1, 'schema-backfill', 'system',
        'schema_backfills:' || did || ':3', 1, sha256(convert_to(a, 'UTF8')), '[{"op":"add","path":"/priority","value":3}]', gen_random_bytes(16));
      nrec := nrec + 1;
    END LOOP;
    cur := ids[array_length(ids, 1)];
    COMMIT;
    d := clock_timestamp() - tb; IF d > maxb THEN maxb := d; END IF; nb := nb + 1;
    IF pause_ms > 0 THEN PERFORM pg_sleep(pause_ms / 1000.0); END IF;
  END LOOP;
  RAISE NOTICE 'batch=% lock=% pause=%ms batches=% rewritten=% total=% max_batch=% wal=% wal_per_rec=% B',
    batch, lockmode, pause_ms, nb, nrec, clock_timestamp() - t0, maxb,
    pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_insert_lsn(), lsn0)),
    round(pg_wal_lsn_diff(pg_current_wal_insert_lsn(), lsn0) / greatest(nrec, 1));
END $$;
