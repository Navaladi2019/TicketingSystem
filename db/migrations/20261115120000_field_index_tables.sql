-- P7-01 (TICK-741): the field-index registry and the four typed side tables
-- of ADR-006 (D2, D3, D14). A tenant field flagged x-nava-filterable gets one
-- record_index_fields row; P7-02's trigger and P7-05's backfill copy its
-- coerced values into rfv_<type>. records.data stays the only source of
-- truth (D1): the side tables are derived and rebuildable, and nothing writes
-- them until P7-02.
--
-- Leakproof by construction (CI-2): every value column has a type whose
-- comparison operators are leakproof, so PostgreSQL can push a filter below
-- the forced RLS policy as an Index Cond: text COLLATE "C", double precision
-- (never numeric, whose comparisons are not leakproof), timestamptz, boolean.
--
-- Keys: tenant_id leads every key, index and foreign key (foreign-key checks
-- run as the table owner and ignore RLS, D-3-2). A side table's primary key
-- (tenant_id, record_id, field_id, ord) backs its records foreign key and the
-- trigger's per-record DELETE. Its value index (tenant_id, field_id, value,
-- record_id) backs the filter's EXISTS probe and the registry foreign key, so
-- a registry DELETE looks for side rows with one index probe. Both foreign
-- keys are NO ACTION: a record is not deleted under its side rows (P7-02's
-- BEFORE DELETE purge clears them first), and a registry row is not deleted
-- while side rows remain (D11 step 5).
--
-- Policies: one per command and per role, never FOR ALL, each testing exactly
-- the canonical tenant predicate. Every role that writes records (nava_app,
-- nava_worker) reads the registry: P7-02's trigger is SECURITY INVOKER and
-- runs as the writer (D14).
--
-- Who may do what (db/grants.yaml, D14):
--   side tables  nava_app, nava_worker: SELECT, INSERT, DELETE. No UPDATE: the
--                trigger and the backfill rewrite a value as DELETE then
--                INSERT (D4, D10).
--   registry     both SELECT. nava_app INSERT (publish, P7-04) and UPDATE of
--                state, progress, requested_version, updated_at. nava_worker
--                UPDATE of state, progress, updated_at (the backfill and
--                removal workflows) and DELETE (removal, D11).
--   nobody       updates a registry row's tenant_id, id, entity_definition_id,
--                field_key or value_type.
--
-- Audit (db/audited-tables.yaml): all five tables are exempt. The side tables
-- are derived from records.data, whose writes are audited. The registry is
-- index configuration: the entity-definition publish that writes or flips a
-- row is audited in that same transaction (subject entity_definition), and
-- the worker's state and progress writes are workflow bookkeeping.
--
-- Statistics (D3, R12): an MCV list on (tenant_id, field_id) for every side
-- table, so the planner knows how many values one field of one tenant holds,
-- and on (tenant_id, field_id, value) for text and bool, the low-cardinality
-- types whose equality filter is the common case. Bool's target is 10,000:
-- at 1,000 a rare value of a large tenant drops out of the list.
--
-- New, empty tables: CREATE TABLE and CREATE INDEX lock no existing table
-- except for the foreign keys (SHARE ROW EXCLUSIVE on entity_definitions and
-- records, held for milliseconds). lock_timeout makes the migration fail and
-- be re-run instead of queueing record writes behind a long transaction.
-- Design and evidence: docs/data/field-index.md; ADR-006 Evidence (PR #587).

-- +goose Up

SET LOCAL lock_timeout = '5s';

-- One row per flagged field of an entity type (D2). id is the field_id the
-- side tables carry. value_type is fixed for the row's life: publish refuses
-- a type change of a flagged field (D8). requested_version is the entity
-- definition version whose publish last asked for this row's state.
CREATE TABLE record_index_fields (
    tenant_id            uuid        NOT NULL,
    id                   uuid        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    field_key            text        NOT NULL,
    value_type           text        NOT NULL,
    state                text        NOT NULL,
    progress             smallint    NOT NULL DEFAULT 0,
    requested_version    integer     NOT NULL,
    updated_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT record_index_fields_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT record_index_fields_key UNIQUE (tenant_id, entity_definition_id, field_key),
    CONSTRAINT record_index_fields_entity_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT record_index_fields_field_key_check CHECK (field_key <> ''),
    CONSTRAINT record_index_fields_value_type_check CHECK (value_type IN ('text', 'num', 'ts', 'bool')),
    CONSTRAINT record_index_fields_state_check CHECK (state IN ('backfilling', 'ready', 'removing')),
    CONSTRAINT record_index_fields_progress_check CHECK (progress BETWEEN 0 AND 100),
    CONSTRAINT record_index_fields_requested_version_check CHECK (requested_version >= 1)
);
ALTER TABLE record_index_fields ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_index_fields FORCE ROW LEVEL SECURITY;

-- nava_app has no DELETE policy and nava_worker no INSERT policy: there is
-- nothing to allow.
CREATE POLICY record_index_fields_select_app ON record_index_fields FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY record_index_fields_select_worker ON record_index_fields FOR SELECT TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY record_index_fields_insert_app ON record_index_fields FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY record_index_fields_update_app ON record_index_fields FOR UPDATE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY record_index_fields_update_worker ON record_index_fields FOR UPDATE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY record_index_fields_delete_worker ON record_index_fields FOR DELETE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- rfv_text: one row per value of a text field. ord is the position in an array
-- value, 0 for a scalar.
CREATE TABLE rfv_text (
    tenant_id uuid     NOT NULL,
    field_id  uuid     NOT NULL,
    record_id uuid     NOT NULL,
    ord       smallint NOT NULL,
    value     text COLLATE "C" NOT NULL,
    CONSTRAINT rfv_text_pkey PRIMARY KEY (tenant_id, record_id, field_id, ord),
    CONSTRAINT rfv_text_record_fk FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id),
    CONSTRAINT rfv_text_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id),
    CONSTRAINT rfv_text_ord_check CHECK (ord >= 0)
);
CREATE INDEX rfv_text_value ON rfv_text (tenant_id, field_id, value, record_id);
CREATE STATISTICS rfv_text_field_mcv (mcv) ON tenant_id, field_id FROM rfv_text;
CREATE STATISTICS rfv_text_value_mcv (mcv) ON tenant_id, field_id, value FROM rfv_text;
ALTER TABLE rfv_text ENABLE ROW LEVEL SECURITY;
ALTER TABLE rfv_text FORCE ROW LEVEL SECURITY;
-- No UPDATE policy: there is nothing to allow.
CREATE POLICY rfv_text_select_app ON rfv_text FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_text_select_worker ON rfv_text FOR SELECT TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_text_insert_app ON rfv_text FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_text_insert_worker ON rfv_text FOR INSERT TO nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_text_delete_app ON rfv_text FOR DELETE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_text_delete_worker ON rfv_text FOR DELETE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- rfv_num: one row per value of a num field. ord is the position in an array
-- value, 0 for a scalar.
CREATE TABLE rfv_num (
    tenant_id uuid     NOT NULL,
    field_id  uuid     NOT NULL,
    record_id uuid     NOT NULL,
    ord       smallint NOT NULL,
    value     double precision NOT NULL,
    CONSTRAINT rfv_num_pkey PRIMARY KEY (tenant_id, record_id, field_id, ord),
    CONSTRAINT rfv_num_record_fk FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id),
    CONSTRAINT rfv_num_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id),
    CONSTRAINT rfv_num_ord_check CHECK (ord >= 0)
);
CREATE INDEX rfv_num_value ON rfv_num (tenant_id, field_id, value, record_id);
CREATE STATISTICS rfv_num_field_mcv (mcv) ON tenant_id, field_id FROM rfv_num;
ALTER TABLE rfv_num ENABLE ROW LEVEL SECURITY;
ALTER TABLE rfv_num FORCE ROW LEVEL SECURITY;
-- No UPDATE policy: there is nothing to allow.
CREATE POLICY rfv_num_select_app ON rfv_num FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_num_select_worker ON rfv_num FOR SELECT TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_num_insert_app ON rfv_num FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_num_insert_worker ON rfv_num FOR INSERT TO nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_num_delete_app ON rfv_num FOR DELETE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_num_delete_worker ON rfv_num FOR DELETE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- rfv_ts: one row per value of a ts field. ord is the position in an array
-- value, 0 for a scalar.
CREATE TABLE rfv_ts (
    tenant_id uuid     NOT NULL,
    field_id  uuid     NOT NULL,
    record_id uuid     NOT NULL,
    ord       smallint NOT NULL,
    value     timestamptz NOT NULL,
    CONSTRAINT rfv_ts_pkey PRIMARY KEY (tenant_id, record_id, field_id, ord),
    CONSTRAINT rfv_ts_record_fk FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id),
    CONSTRAINT rfv_ts_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id),
    CONSTRAINT rfv_ts_ord_check CHECK (ord >= 0)
);
CREATE INDEX rfv_ts_value ON rfv_ts (tenant_id, field_id, value, record_id);
CREATE STATISTICS rfv_ts_field_mcv (mcv) ON tenant_id, field_id FROM rfv_ts;
ALTER TABLE rfv_ts ENABLE ROW LEVEL SECURITY;
ALTER TABLE rfv_ts FORCE ROW LEVEL SECURITY;
-- No UPDATE policy: there is nothing to allow.
CREATE POLICY rfv_ts_select_app ON rfv_ts FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_ts_select_worker ON rfv_ts FOR SELECT TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_ts_insert_app ON rfv_ts FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_ts_insert_worker ON rfv_ts FOR INSERT TO nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_ts_delete_app ON rfv_ts FOR DELETE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_ts_delete_worker ON rfv_ts FOR DELETE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- rfv_bool: one row per value of a bool field. ord is the position in an array
-- value, 0 for a scalar.
CREATE TABLE rfv_bool (
    tenant_id uuid     NOT NULL,
    field_id  uuid     NOT NULL,
    record_id uuid     NOT NULL,
    ord       smallint NOT NULL,
    value     boolean NOT NULL,
    CONSTRAINT rfv_bool_pkey PRIMARY KEY (tenant_id, record_id, field_id, ord),
    CONSTRAINT rfv_bool_record_fk FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id),
    CONSTRAINT rfv_bool_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id),
    CONSTRAINT rfv_bool_ord_check CHECK (ord >= 0)
);
CREATE INDEX rfv_bool_value ON rfv_bool (tenant_id, field_id, value, record_id);
CREATE STATISTICS rfv_bool_field_mcv (mcv) ON tenant_id, field_id FROM rfv_bool;
CREATE STATISTICS rfv_bool_value_mcv (mcv) ON tenant_id, field_id, value FROM rfv_bool;
ALTER STATISTICS rfv_bool_value_mcv SET STATISTICS 10000;
ALTER TABLE rfv_bool ENABLE ROW LEVEL SECURITY;
ALTER TABLE rfv_bool FORCE ROW LEVEL SECURITY;
-- No UPDATE policy: there is nothing to allow.
CREATE POLICY rfv_bool_select_app ON rfv_bool FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_bool_select_worker ON rfv_bool FOR SELECT TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_bool_insert_app ON rfv_bool FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_bool_insert_worker ON rfv_bool FOR INSERT TO nava_worker
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_bool_delete_app ON rfv_bool FOR DELETE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_bool_delete_worker ON rfv_bool FOR DELETE TO nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose Down

SET LOCAL lock_timeout = '5s';

DROP TABLE rfv_bool;
DROP TABLE rfv_ts;
DROP TABLE rfv_num;
DROP TABLE rfv_text;
DROP TABLE record_index_fields;
