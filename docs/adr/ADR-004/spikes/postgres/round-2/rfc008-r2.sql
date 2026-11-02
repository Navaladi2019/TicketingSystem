CREATE ROLE w_app; CREATE ROLE nava_migrator;
CREATE TABLE audit_log (tenant_id uuid NOT NULL, occurred_at timestamptz NOT NULL DEFAULT now(), id uuid NOT NULL DEFAULT gen_random_uuid(),
  txid xid8 NOT NULL DEFAULT pg_current_xact_id(), subject_type text NOT NULL, subject_id uuid NOT NULL, version_after int,
  backfilled boolean NOT NULL DEFAULT false, seq bigint, sealed_at timestamptz,
  PRIMARY KEY (tenant_id, occurred_at, id)) PARTITION BY RANGE (occurred_at);
CREATE TABLE audit_log_default PARTITION OF audit_log DEFAULT;
\echo == U1 D4 unique key NULLS NOT DISTINCT on partitioned table
CREATE UNIQUE INDEX audit_d4 ON audit_log (tenant_id, occurred_at, txid, subject_type, subject_id, version_after) NULLS NOT DISTINCT;
CREATE INDEX audit_seq ON audit_log (tenant_id, seq) WHERE seq IS NOT NULL;
CREATE FUNCTION assert_ins() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
  IF current_user <> 'nava_migrator' AND (NEW.occurred_at <> now() OR NEW.txid <> pg_current_xact_id() OR NEW.backfilled OR NEW.seq IS NOT NULL OR NEW.sealed_at IS NOT NULL)
  THEN RAISE EXCEPTION 'audit_log: forbidden insert column set by %', current_user; END IF; RETURN NEW; END $$;
\echo == U2 BEFORE INSERT row trigger on partitioned parent
CREATE TRIGGER audit_assert BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION assert_ins();
SELECT tgrelid::regclass, tgname FROM pg_trigger WHERE tgname='audit_assert';
GRANT INSERT (tenant_id, id, subject_type, subject_id, version_after) ON audit_log TO w_app;
GRANT SELECT, INSERT ON audit_log TO nava_migrator;
\echo == U3 w_app normal insert via column grant, defaults fire, assert passes
SET ROLE w_app; INSERT INTO audit_log (tenant_id, subject_type, subject_id, version_after) VALUES (gen_random_uuid(),'t',gen_random_uuid(),1); RESET ROLE;
\echo == U4 w_app sets backfilled (grant denies)
SET ROLE w_app; INSERT INTO audit_log (tenant_id, subject_type, subject_id, backfilled) VALUES (gen_random_uuid(),'t',gen_random_uuid(),true); RESET ROLE;
\echo == U5 owner-ish role (postgres superuser, not migrator) inserts backdated row: assert raises
INSERT INTO audit_log (tenant_id, occurred_at, subject_type, subject_id) VALUES (gen_random_uuid(), now()-interval '1 day','t',gen_random_uuid());
\echo == U6 assert fires when inserting directly into the partition
INSERT INTO audit_log_default (tenant_id, occurred_at, subject_type, subject_id) VALUES (gen_random_uuid(), now()-interval '1 day','t',gen_random_uuid());
\echo == U7 migrator backfills backdated row
SET ROLE nava_migrator; INSERT INTO audit_log (tenant_id, occurred_at, subject_type, subject_id, backfilled) VALUES (gen_random_uuid(), now()-interval '1 day','t',gen_random_uuid(), true); RESET ROLE;
\echo == U8 BEFORE trigger that rewrites partition key (what rev 1 avoids), with a second partition
CREATE TABLE audit_log_old PARTITION OF audit_log FOR VALUES FROM ('2000-01-01') TO ('2001-01-01');
CREATE FUNCTION rewrite() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.occurred_at := now(); RETURN NEW; END $$;
CREATE TRIGGER z_rewrite BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION rewrite();
SET ROLE nava_migrator; INSERT INTO audit_log (tenant_id, occurred_at, subject_type, subject_id) VALUES (gen_random_uuid(), '2000-06-01','t',gen_random_uuid()); RESET ROLE;
\echo == U9 duplicate seq possible without unique
DROP TRIGGER z_rewrite ON audit_log; UPDATE audit_log SET seq = 1; SELECT tenant_id IS NOT NULL, count(*) FILTER (WHERE seq=1) AS rows_with_seq1 FROM audit_log GROUP BY 1;
\echo == U10 unique (tenant_id, seq) on partitioned table without partition key
CREATE UNIQUE INDEX ON audit_log (tenant_id, seq) WHERE seq IS NOT NULL;
