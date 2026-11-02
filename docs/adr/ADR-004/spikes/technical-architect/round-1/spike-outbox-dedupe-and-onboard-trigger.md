# Technical Architect spike, RFC-008 round 1

Source: TICK-409 comment 01a1987b-98ea-7733-8508-f9af0680d0b5 (Technical Architect critique round 1, 2026-11-01), copied verbatim.

<details><summary>Spike: outbox dedupe wait and onboard trigger privileges (Postgres 16.10, Docker, 2026-11-01)</summary>

**Hypotheses.**
- H1: `ON CONFLICT DO NOTHING` on a key that another open transaction holds blocks until that transaction ends.
- H2: under D8's grants, an INVOKER deferred trigger fails for `nava_onboard`.

**Pass/fail.** H1 holds if the second insert blocks for longer than 3 s. H2 holds if T4 or T5 errors.

**Raw output.**
```
== T1: A holds dedupe key 5s; B ON CONFLICT DO NOTHING ==
2026-11-01 18:57:51.903503+00
2026-11-01 18:57:55.91874+00
== T1b: same with lock_timeout=1s ==
ERROR:  canceling statement due to lock timeout
CONTEXT:  while inserting index tuple (0,2) in relation "outbox"
== T2: app path, entry present ==        (commits)
== T3: app path, entry missing ==
ERROR:  no audit entry for 33333333-3333-3333-3333-333333333333
== T4: onboard path (INSERT-only on audit_log, as D8), entry present ==
ERROR:  permission denied for table audit_log
CONTEXT:  SQL expression "NOT EXISTS (SELECT 1 FROM audit_log a WHERE ...)"
== T5: onboard with SELECT grant but no SELECT policy ==
ERROR:  no audit entry for 55555555-5555-5555-5555-555555555555
== T6: onboard outbox insert (no grant, as in db/grants.yaml) ==
ERROR:  permission denied for table outbox
PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu
```

**Schema (`s.sql`).**
```sql
CREATE ROLE onb; CREATE ROLE app;
CREATE TABLE outbox(tenant_id uuid, dedupe_key text, CONSTRAINT u UNIQUE(tenant_id,dedupe_key));
GRANT SELECT,INSERT ON outbox TO app;
CREATE TABLE cfg(tenant_id uuid, id uuid);
CREATE TABLE audit_log(tenant_id uuid, subject_id uuid, txid xid8 DEFAULT pg_current_xact_id());
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY; ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
CREATE POLICY a_app_sel ON audit_log FOR SELECT TO app USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY a_app_ins ON audit_log FOR INSERT TO app WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY a_onb_ins ON audit_log FOR INSERT TO onb WITH CHECK (true);
GRANT SELECT,INSERT ON audit_log, cfg TO app; GRANT INSERT ON audit_log, cfg TO onb;
CREATE FUNCTION chk() RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER AS $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM audit_log a WHERE a.tenant_id=NEW.tenant_id AND a.subject_id=NEW.id AND a.txid=pg_current_xact_id()) THEN
    RAISE EXCEPTION 'no audit entry for %', NEW.id; END IF; RETURN NULL; END $$;
CREATE CONSTRAINT TRIGGER cfg_audit AFTER INSERT OR UPDATE ON cfg DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION chk();
```

**How to rerun.**
```bash
docker run -d --name ta-rfc008-spike -e POSTGRES_PASSWORD=spike postgres:16.10
docker exec -i ta-rfc008-spike psql -U postgres -q < s.sql
T=11111111-1111-1111-1111-111111111111
# T1: A holds the key for 5 s; B starts 1 s later
docker exec ta-rfc008-spike psql -U postgres -c "SET ROLE app; BEGIN; INSERT INTO outbox VALUES ('$T','k') ON CONFLICT DO NOTHING; SELECT pg_sleep(5); COMMIT;" &
sleep 1; docker exec ta-rfc008-spike psql -U postgres -c "SET ROLE app; SELECT clock_timestamp(); INSERT INTO outbox VALUES ('$T','k') ON CONFLICT DO NOTHING; SELECT clock_timestamp();"
# T4: onboard, entry present
docker exec ta-rfc008-spike psql -U postgres -c "SET ROLE onb; BEGIN; INSERT INTO cfg VALUES ('$T','44444444-4444-4444-4444-444444444444'); INSERT INTO audit_log(tenant_id,subject_id) VALUES ('$T','44444444-4444-4444-4444-444444444444'); COMMIT;"
docker rm -f ta-rfc008-spike
```
T2, T3, T5 and T6 are the same pattern with the variations named in the output.
</details>
