-- RFC-021 critique spike (Postgres reviewer). Synthetic, shaped like main's
-- records / audit_log / ADR-006 D3 rfv_num. Not a migration.
CREATE ROLE nava_worker LOGIN PASSWORD 'x';
CREATE ROLE nava_app LOGIN PASSWORD 'x';
CREATE TABLE records (
  tenant_id uuid NOT NULL, id uuid NOT NULL, entity_definition_id uuid NOT NULL,
  version bigint NOT NULL, schema_version int NOT NULL, number text NOT NULL,
  state text NOT NULL, requester_id uuid, assignee_id uuid, assignment_group_id uuid,
  created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL, data jsonb NOT NULL,
  causation_origin_id uuid, causation_depth smallint NOT NULL DEFAULT 0,
  CONSTRAINT records_pkey PRIMARY KEY (tenant_id, id));
-- main's list index set (20261109140000.., 20261110100000)
CREATE INDEX records_list_updated ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC);
CREATE INDEX records_list_created ON records (tenant_id, entity_definition_id, created_at DESC, id DESC);
CREATE INDEX records_list_group_state ON records (tenant_id, entity_definition_id, assignment_group_id DESC, state DESC, updated_at DESC, id DESC);
CREATE TABLE record_shares (tenant_id uuid NOT NULL, record_id uuid NOT NULL, principal_id uuid NOT NULL,
  PRIMARY KEY (tenant_id, record_id, principal_id),
  FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id));
CREATE TABLE audit_log (
  tenant_id uuid NOT NULL, id uuid NOT NULL, occurred_at timestamptz NOT NULL DEFAULT now(),
  txid xid8 NOT NULL DEFAULT pg_current_xact_id(), domain text NOT NULL, action text NOT NULL,
  subject_type text NOT NULL, subject_id uuid NOT NULL, entity_definition_id uuid,
  version_before bigint, version_after bigint, actor_principal_id uuid, actor_system text,
  source text NOT NULL, source_ref text, change_set_id uuid, causation_id uuid, causation_depth smallint,
  diff_sha256 bytea, diff_truncated bool NOT NULL DEFAULT false, diff json, diff_salt bytea,
  seq bigint, prev_hash bytea, entry_hash bytea, sealed_at timestamptz,
  PRIMARY KEY (tenant_id, occurred_at, id));
CREATE UNIQUE INDEX audit_log_commit_key ON audit_log (tenant_id, subject_type, subject_id, version_after);
CREATE INDEX audit_log_unsealed ON audit_log (tenant_id, occurred_at, id) WHERE sealed_at IS NULL;
-- ADR-006 D3 num side table + D4 trigger (one field)
CREATE TABLE rfv_num (tenant_id uuid NOT NULL, field_id uuid NOT NULL, record_id uuid NOT NULL,
  ord smallint NOT NULL, value float8 NOT NULL,
  PRIMARY KEY (tenant_id, record_id, field_id, ord),
  FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id));
CREATE INDEX rfv_num_value ON rfv_num (tenant_id, field_id, value, record_id);
CREATE STATISTICS rfv_num_mcv (mcv) ON tenant_id, field_id FROM rfv_num;
CREATE TABLE fields (tenant_id uuid, field_id uuid, entity_definition_id uuid, field_key text);
CREATE FUNCTION rfv_maint() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE f record;
BEGIN
  FOR f IN SELECT * FROM fields WHERE tenant_id = NEW.tenant_id AND entity_definition_id = NEW.entity_definition_id LOOP
    IF TG_OP = 'INSERT' OR (OLD.data -> f.field_key) IS DISTINCT FROM (NEW.data -> f.field_key) THEN
      DELETE FROM rfv_num WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = f.field_id;
      IF jsonb_typeof(NEW.data -> f.field_key) = 'number' THEN
        INSERT INTO rfv_num VALUES (NEW.tenant_id, f.field_id, NEW.id, 0, (NEW.data ->> f.field_key)::float8);
      END IF;
    END IF;
  END LOOP;
  RETURN NULL;
END $$;
CREATE TRIGGER rfv_maint AFTER INSERT OR UPDATE OF data ON records FOR EACH ROW EXECUTE FUNCTION rfv_maint();
CREATE TABLE schema_backfill_failures (tenant_id uuid, entity_definition_id uuid, target_version int, record_id uuid, code text,
  PRIMARY KEY (tenant_id, entity_definition_id, target_version, record_id));
DO $$ DECLARE t text; BEGIN
FOREACH t IN ARRAY ARRAY['records','audit_log','rfv_num','record_shares','schema_backfill_failures'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY; ALTER TABLE %I FORCE ROW LEVEL SECURITY', t, t);
  EXECUTE format($p$CREATE POLICY p ON %I TO nava_app, nava_worker USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)$p$, t);
  EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO nava_app, nava_worker', t);
END LOOP; END $$;
GRANT SELECT ON fields TO nava_app, nava_worker;
