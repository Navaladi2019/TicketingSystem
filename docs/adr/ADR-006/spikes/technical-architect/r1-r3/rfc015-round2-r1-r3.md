# Technical Architect round 2: R1/R3 re-run against RFC-015 rev 1

Source: TICK-714 comment 01a1cb1a (2026-11-11), archived verbatim.

## Technical Architect, round 2: RFC-015 rev 1 (PR #582 at `d471c036`)

**Verdict: APPROVE.** F1 and F2 are closed. I re-ran R1 and R3 on PG 16.10, applying the rev 1 protocol exactly as D17, D10, D11 and §5 describe it. Both now give 0, and the controls without the barrier still fail, so the test can still catch the bug. Rev 1 broke none of the non-blocking items I touched.

This answers the barrier race only. The ADR is still gated on S1–S3 from the Postgres & Data Modeling Specialist, and on the human's review.

| Item | Rev 1 text | Verdict | Evidence (Known unless marked) |
|---|---|---|---|
| F1: insert before the flip, committed after the batch | D17 barrier as step 1 of `FieldIndexBackfill`; §5 steps 2 and 5 (UUIDv7 argument withdrawn) | **confirmed sound** | R1 control (no barrier): `records=3001 side_rows=3000 missing_or_stale=1`. R1 with D17: barrier waited on xid `[752]`, cleared after 5 polls, then `missing_or_stale=0`. |
| F2: write that saw `ready` and commits after the removal delete | D11: barrier → batched delete → `NOT EXISTS` loop → registry delete, plus the required D3 FK | **confirmed sound** | R3 control (no barrier): the registry delete fails with `violates foreign key constraint "record_field_values_tenant_id_field_id_fkey"`, leaving 1 side row and 1 registry row. The FK makes the orphan loud, as intended. R3 with D11: barrier waited on xid `[803]`, then `side rows left: 0, registry rows left: 0`. |
| D10(a): a backfill batch in flight vs. the flip to `removing` | Registry row `FOR SHARE`; the batch stops unless the state is `backfilling` | **confirmed sound** | R4: the publish `UPDATE … state='removing'` waited 1,541 ms on the batch's registry lock. The next batch logged `batch stopped: state=removing`. |
| D17: the poll design (snapshot once, then poll `pg_xact_status` once per attempt, workflow timer between polls) | D17 | **confirmed sound** | Ran as separate short transactions, the same as activity attempts, under `nava_worker` with only the D20 grants. No extra grant was needed. Inferred: if heartbeat details are lost and a retry takes a fresh snapshot, it is still safe. The fresh snapshot is a superset of every transaction that was in flight at the flip and is still running. |
| D20 grants vs. the D10 locks | `nava_worker` has `UPDATE (state, progress, updated_at)` on the registry; `records` UPDATE comes from `db/grants.yaml` | **confirmed sound** | `SELECT … FOR SHARE` needs UPDATE privilege and an UPDATE policy on the locked table. Both are present for `records` (`db/grants.yaml:170-174`, `00002_platform_tables.sql:90`) and for the registry under D20. The batches ran without a 42501 error. |
| F8 under rev 1: DELETE + INSERT, no UPDATE grant on the side tables | D5, D10, D20 | **confirmed sound** | The trigger and batch ran with `SELECT, INSERT, DELETE` only, and both function-level `SET` clauses were in place. |

**Setup:**
- `postgres:16.10` (the compose pin) in throwaway Docker, with 3,000 records in one tenant.
- Forced RLS with the repo's GUC policies, one policy per command.
- `records` grants as in `db/grants.yaml` (no DELETE). Side-table and registry grants as in D20, plus the required composite FK.
- The D5 trigger with rev 1's `SET search_path/TimeZone/DateStyle` and the per-field diff.
- Timing is fixed with `pg_sleep` in the live transaction only. The barrier itself never sleeps inside SQL.
- The container was removed afterwards.

**For S3:** this script is the deterministic part of S3, so the Postgres & Data Modeling Specialist can reuse it as is. The load-run diff, the duration and the WAL bytes per batch are still theirs to produce.

<details><summary>Spike code: run.sh, schema.sql and raw output (rerun: <code>docker run -d --name pg postgres:16.10</code>, copy both files in, <code>psql -U postgres -f /schema.sql</code>, then <code>bash /run.sh R1|R1-control|R3|R3-control|R4</code>)</summary>

```sql
-- schema.sql (run as postgres)
CREATE ROLE nava_app LOGIN; CREATE ROLE nava_worker LOGIN;
CREATE TABLE records (tenant_id uuid NOT NULL, id uuid NOT NULL, entity_definition_id uuid NOT NULL, data jsonb NOT NULL, PRIMARY KEY (tenant_id,id));
CREATE TABLE record_index_fields (tenant_id uuid NOT NULL, id uuid NOT NULL, entity_definition_id uuid NOT NULL, field_key text NOT NULL,
  value_type text NOT NULL, state text NOT NULL, progress int DEFAULT 0, updated_at timestamptz DEFAULT now(), PRIMARY KEY (tenant_id,id));
CREATE TABLE record_field_values (tenant_id uuid NOT NULL, field_id uuid NOT NULL, record_id uuid NOT NULL, ord smallint NOT NULL DEFAULT 0,
  value_text text COLLATE "C", value_num double precision, value_ts timestamptz, value_bool boolean,
  PRIMARY KEY (tenant_id, record_id, field_id, ord),
  FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields(tenant_id, id),   -- D3 required FK (rev 1)
  FOREIGN KEY (tenant_id, record_id) REFERENCES records(tenant_id, id));
DO $$ DECLARE t text; c text; BEGIN
 FOREACH t IN ARRAY ARRAY['records','record_index_fields','record_field_values'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY; ALTER TABLE %I FORCE ROW LEVEL SECURITY;',t,t);
  FOREACH c IN ARRAY ARRAY['SELECT','INSERT','UPDATE','DELETE'] LOOP
   IF c = 'INSERT' THEN
    EXECUTE format($p$CREATE POLICY %s ON %I FOR %s TO nava_app, nava_worker WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)$p$, 'p_'||lower(c), t, c);
   ELSE
    EXECUTE format($p$CREATE POLICY %s ON %I FOR %s TO nava_app, nava_worker USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)$p$, 'p_'||lower(c), t, c);
   END IF; END LOOP; END LOOP; END $$;
GRANT SELECT, INSERT, UPDATE ON records TO nava_app, nava_worker;                 -- db/grants.yaml: no DELETE
GRANT SELECT, INSERT, DELETE ON record_field_values TO nava_app, nava_worker;     -- D20: no UPDATE
GRANT SELECT ON record_index_fields TO nava_app, nava_worker;
GRANT INSERT, UPDATE (state, progress, updated_at) ON record_index_fields TO nava_app;
GRANT UPDATE (state, progress, updated_at), DELETE ON record_index_fields TO nava_worker;
CREATE FUNCTION record_field_values_sync() RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER
 SET search_path = pg_catalog, public SET TimeZone = 'UTC' SET DateStyle = 'ISO, YMD' AS $$
DECLARE f record; BEGIN
  FOR f IN SELECT id, field_key FROM public.record_index_fields
           WHERE tenant_id = NEW.tenant_id AND entity_definition_id = NEW.entity_definition_id AND state IN ('backfilling','ready') LOOP
    IF TG_OP = 'UPDATE' AND (OLD.data -> f.field_key) IS NOT DISTINCT FROM (NEW.data -> f.field_key) THEN CONTINUE; END IF;
    DELETE FROM public.record_field_values WHERE tenant_id=NEW.tenant_id AND record_id=NEW.id AND field_id=f.id;
    IF NEW.data ? f.field_key THEN
      INSERT INTO public.record_field_values(tenant_id,field_id,record_id,value_text) VALUES (NEW.tenant_id,f.id,NEW.id,NEW.data->>f.field_key);
    END IF;
  END LOOP; RETURN NULL; END $$;
CREATE TRIGGER rfv_sync AFTER INSERT OR UPDATE OF data ON records FOR EACH ROW EXECUTE FUNCTION record_field_values_sync();
```

```bash
#!/bin/bash
# run.sh: RFC-015 rev 1 protocol. Each worker/app call is one short transaction = one Temporal activity attempt.
T=00000000-0000-0000-0000-00000000000a; E=00000000-0000-0000-0000-0000000000e1; F=00000000-0000-0000-0000-0000000000f1
G="SELECT set_config('app.tenant_id','$T',true)"
q() { psql -U "$1" -d postgres -v ON_ERROR_STOP=1 -qtA -c "BEGIN" -c "$G" -c "$2" -c "COMMIT" | grep -v -x -e "$T"; }
app()    { q nava_app "$1"; }
worker() { q nava_worker "$1"; }
reset()  { psql -U postgres -qtA -c "TRUNCATE records, record_index_fields, record_field_values" \
  -c "INSERT INTO records SELECT '$T', gen_random_uuid(), '$E', jsonb_build_object('site','s'||(g%10)) FROM generate_series(1,3000) g" >/dev/null; }
publish_flag()   { app "INSERT INTO record_index_fields(tenant_id,id,entity_definition_id,field_key,value_type,state) VALUES ('$T','$F','$E','site','text','backfilling')"; }
publish_remove() { app "UPDATE record_index_fields SET state='removing', updated_at=now() WHERE tenant_id='$T' AND id='$F'"; }
# D17: snapshot once, keep xip list (heartbeat details); later attempts poll pg_xact_status; timer between polls
barrier() {
  local xids open polls=0
  xids=$(worker "SELECT coalesce(string_agg(x::text, ','),'') FROM pg_snapshot_xip(pg_current_snapshot()) x")
  [ -z "$xids" ] && { echo "  barrier: no in-progress xids"; return; }
  while :; do polls=$((polls+1))
    open=$(worker "SELECT count(*) FROM unnest('{$xids}'::xid8[]) x WHERE pg_xact_status(x) = 'in progress'")
    [ "$open" = "0" ] && break; sleep 0.5
  done; echo "  barrier: waited on xids [$xids], clear after $polls poll(s)"; }
# D10 batch: (a) registry FOR SHARE check, (b) records keyset FOR SHARE, (c) per (record, field) DELETE + INSERT, (d) progress
batch() { worker "DO \$b\$ DECLARE st text; r record; BEGIN
  SELECT state INTO st FROM record_index_fields WHERE tenant_id='$T' AND id='$F' FOR SHARE;
  IF st IS DISTINCT FROM 'backfilling' THEN RAISE NOTICE 'batch stopped: state=%', st; RETURN; END IF;
  FOR r IN SELECT id, data FROM records WHERE tenant_id='$T' AND entity_definition_id='$E' AND id > '00000000-0000-0000-0000-000000000000' ORDER BY id LIMIT 5000 FOR SHARE LOOP
    DELETE FROM record_field_values WHERE tenant_id='$T' AND record_id=r.id AND field_id='$F';
    IF r.data ? 'site' THEN INSERT INTO record_field_values(tenant_id,field_id,record_id,value_text) VALUES ('$T','$F',r.id,r.data->>'site'); END IF;
  END LOOP;
  UPDATE record_index_fields SET progress=100, updated_at=now() WHERE tenant_id='$T' AND id='$F'; END \$b\$"; }
set_ready() { worker "UPDATE record_index_fields SET state='ready', updated_at=now() WHERE tenant_id='$T' AND id='$F'"; }
# D11: barrier, batched delete, NOT EXISTS loop, registry delete (FK guards)
remove_all() { local left; barrier
  while :; do worker "DELETE FROM record_field_values WHERE tenant_id='$T' AND field_id='$F'"
    left=$(worker "SELECT count(*) FROM record_field_values WHERE tenant_id='$T' AND field_id='$F'"); [ "$left" = "0" ] && break; done
  worker "DELETE FROM record_index_fields WHERE tenant_id='$T' AND id='$F'"; }
report() { worker "SELECT 'records='||count(*)||' side_rows='||(SELECT count(*) FROM record_field_values)||' missing_or_stale='||(SELECT count(*) FROM records r WHERE NOT EXISTS (SELECT 1 FROM record_field_values v WHERE v.record_id=r.id AND v.value_text=r.data->>'site')) FROM records"; }
leftovers() { echo "  side rows left: $(worker "SELECT count(*) FROM record_field_values")  registry rows left: $(worker "SELECT count(*) FROM record_index_fields")"; }
live_insert() { psql -U nava_app -d postgres -qtA -o /dev/null -c "BEGIN" -c "$G" -c "INSERT INTO records VALUES ('$T','00000000-0000-0000-0000-000000000001','$E','{\"site\":\"s7\"}')" -c "SELECT pg_sleep(3)" -c "COMMIT"; }
live_update() { psql -U nava_app -d postgres -qtA -o /dev/null -c "BEGIN" -c "$G" -c "UPDATE records SET data='{\"site\":\"CHANGED\"}' WHERE tenant_id='$T' AND id=(SELECT id FROM records ORDER BY id DESC LIMIT 1)" -c "SELECT pg_sleep(3)" -c "COMMIT"; }
case $1 in
R1-control) reset; live_insert & sleep 1; publish_flag; batch; set_ready; wait; report ;;
R1)         reset; live_insert & sleep 1; publish_flag; barrier; batch; set_ready; barrier; wait; report ;;
R3-control) reset; publish_flag; barrier; batch; set_ready; barrier
            live_update & sleep 1; publish_remove
            worker "DELETE FROM record_field_values WHERE tenant_id='$T' AND field_id='$F'"; wait
            worker "DELETE FROM record_index_fields WHERE tenant_id='$T' AND id='$F'" 2>&1 | head -1 | sed 's/^/  registry delete: /'; leftovers ;;
R3)         reset; publish_flag; barrier; batch; set_ready; barrier
            live_update & sleep 1; publish_remove; remove_all; wait; leftovers ;;
R4)         reset; publish_flag; barrier
            psql -U nava_worker -d postgres -qtA -o /dev/null -c "BEGIN" -c "$G" -c "SELECT state FROM record_index_fields WHERE id='$F' FOR SHARE" -c "SELECT pg_sleep(2)" -c "COMMIT" & sleep 0.5
            s=$(date +%s%N); publish_remove; echo "  flip to removing waited $(( ($(date +%s%N)-s)/1000000 )) ms on the in-flight batch's registry lock"; wait
            batch 2>&1 | grep -o "batch stopped.*" | sed 's/^/  next batch: /' ;;
esac
```

```text
== R1-control
records=3001 side_rows=3000 missing_or_stale=1
== R1
  barrier: waited on xids [752], clear after 5 poll(s)
  barrier: no in-progress xids
records=3001 side_rows=3001 missing_or_stale=0
== R3-control
  registry delete: ERROR:  update or delete on table "record_index_fields" violates foreign key constraint "record_field_values_tenant_id_field_id_fkey" on table "record_field_values"
  side rows left: 1  registry rows left: 1
== R3
  barrier: waited on xids [803], clear after 5 poll(s)
  side rows left: 0  registry rows left: 0
== R4
  flip to removing waited 1541 ms on the in-flight batch's registry lock
  next batch: batch stopped: state=removing
```
</details>
