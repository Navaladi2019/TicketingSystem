-- ADR-007 D2 shape, scratch only. 1,000 tenants x 20 rows.
CREATE ROLE nava_app LOGIN PASSWORD 'scratch-app' NOSUPERUSER NOBYPASSRLS;
CREATE TABLE embedding_client (
  tenant_id uuid NOT NULL,
  client_id text NOT NULL,
  allowed_origins text[] NOT NULL DEFAULT '{}',
  state text NOT NULL CHECK (state IN ('enabled','disabled')),
  rate_limit_share int NOT NULL DEFAULT 50,
  created_by text NOT NULL DEFAULT 'seed',
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, client_id)
);
ALTER TABLE embedding_client ENABLE ROW LEVEL SECURITY;
ALTER TABLE embedding_client FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON embedding_client
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
GRANT SELECT ON embedding_client TO nava_app;
INSERT INTO embedding_client (tenant_id, client_id, allowed_origins, state)
SELECT ('00000000-0000-0000-0000-' || lpad(t::text, 12, '0'))::uuid,
       'host-app-' || c,
       ARRAY['https://host' || c || '.example', 'http://127.0.0.1:9001'],
       CASE WHEN c % 7 = 0 THEN 'disabled' ELSE 'enabled' END
FROM generate_series(1, 1000) t, generate_series(1, 20) c;
ANALYZE embedding_client;
