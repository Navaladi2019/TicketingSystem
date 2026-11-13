-- P6-06 (TICK-671, ADR-005 §3.2, §3.3, §3.7 C8, C9): number formats and the
-- per-type epoch registry. Three tenant tables under ENABLE + FORCE ROW LEVEL
-- SECURITY, shaped as entity_definitions / entity_definition_versions: every
-- key and foreign key leads with tenant_id, because PostgreSQL runs
-- foreign-key checks as the table owner and so ignores RLS (C8).
--
-- Who may do what (db/grants.yaml):
--   nava_app      reads all three, creates formats, publishes versions and
--                 epochs (INSERT), and moves a format's current_version /
--                 updated_at. key is set once.
--   nava_worker   reads all three.
--   nobody        UPDATEs or DELETEs a format version or an epoch: no grant,
--                 no policy (C9). nava_purge and nava_onboard have nothing:
--                 no onboarding seeds formats in S18, so there is no pinned
--                 "tenant is provisioning" policy here.
--
-- Every policy is FOR one command and TO named roles and tests exactly the
-- canonical tenant predicate.
--
-- Empty new tables, so plain CREATE TABLE and its index builds hold no lock
-- worth planning for. Every lookup is a primary-key or unique-key probe that
-- leads with tenant_id; there is no secondary index (docs/data/number-formats.md).

-- +goose Up

-- One row per number format. key is unique per tenant, never globally: a
-- global key would be a cross-tenant uniqueness oracle (C9). current_version
-- is the version a "link to the current version" names; nava_app moves it
-- forward when it publishes.
CREATE TABLE number_formats (
    tenant_id       uuid        NOT NULL,
    id              uuid        NOT NULL,
    key             text        NOT NULL,
    current_version integer     NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT number_formats_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT number_formats_key_key UNIQUE (tenant_id, key),
    CONSTRAINT number_formats_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT number_formats_key_check CHECK (key ~ '^[a-z][a-z0-9_]{1,40}$'),
    CONSTRAINT number_formats_current_version_check CHECK (current_version >= 1)
);
ALTER TABLE number_formats ENABLE ROW LEVEL SECURITY;
ALTER TABLE number_formats FORCE ROW LEVEL SECURITY;

CREATE POLICY number_formats_select ON number_formats FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY number_formats_insert ON number_formats FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY number_formats_update ON number_formats FOR UPDATE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- One row per published version of a format. Immutable: no role has UPDATE
-- or DELETE (C9).
--
-- document is the format as the admin published it (template, overflow,
-- optional time-zone override); compiled is the JSON of numbering.Compiled
-- (P6-02/P6-03) and compiled_sha256 its SHA-256, computed by the compiler in
-- Go (jsonb reorders keys, so the database cannot recompute it; the CHECK
-- only pins the length). A definition version snapshots compiled into
-- entity_definition_versions.number_compiled, so a create never reads this
-- table. created_by carries tenant_id into principals, so a version cannot
-- name another tenant's principal.
CREATE TABLE number_format_versions (
    tenant_id       uuid        NOT NULL,
    format_id       uuid        NOT NULL,
    version         integer     NOT NULL,
    document        jsonb       NOT NULL,
    compiled        jsonb       NOT NULL,
    compiled_sha256 bytea       NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    created_by      uuid        NOT NULL,
    CONSTRAINT number_format_versions_pkey PRIMARY KEY (tenant_id, format_id, version),
    CONSTRAINT number_format_versions_format_fkey FOREIGN KEY (tenant_id, format_id)
        REFERENCES number_formats (tenant_id, id),
    CONSTRAINT number_format_versions_created_by_fkey FOREIGN KEY (tenant_id, created_by)
        REFERENCES principals (tenant_id, id),
    CONSTRAINT number_format_versions_version_check CHECK (version >= 1),
    CONSTRAINT number_format_versions_document_check CHECK (jsonb_typeof(document) = 'object'),
    CONSTRAINT number_format_versions_compiled_check CHECK (jsonb_typeof(compiled) = 'object'),
    CONSTRAINT number_format_versions_compiled_sha256_check CHECK (length(compiled_sha256) = 32)
);
ALTER TABLE number_format_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE number_format_versions FORCE ROW LEVEL SECURITY;

-- No UPDATE or DELETE policy: there is nothing to allow (C9).
CREATE POLICY number_format_versions_select ON number_format_versions FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY number_format_versions_insert ON number_format_versions FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- The epoch registry (ADR-005 §3.3, TA-4): the identity of a compiled template
-- within one entity type. A publish whose compiled_sha256 is already here
-- reuses that epoch; a new one takes max(epoch) + 1 inside the definition
-- publish transaction, after SELECT ... FOR UPDATE on the type (C10), so the
-- two UNIQUE keys never turn a race into a 500. Epoch 0 ({prefix}{seq:7},
-- scope_key = '') is implicit and has no row, so epoch >= 1.
--
-- literal_head is the literal text before the first variable token, with
-- {prefix} resolved: the type's number_prefix (^[A-Z]{2,6}$, the leading
-- {prefix} rule of §3.1) and then the literal characters of §3.1. The
-- disjointness check (§3.3) compares it across this type's epochs.
-- Immutable: no role has UPDATE or DELETE (C9).
CREATE TABLE numbering_epochs (
    tenant_id            uuid        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    epoch                integer     NOT NULL,
    compiled_sha256      bytea       NOT NULL,
    literal_head         text        NOT NULL,
    created_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT numbering_epochs_pkey PRIMARY KEY (tenant_id, entity_definition_id, epoch),
    CONSTRAINT numbering_epochs_compiled_sha256_key UNIQUE (tenant_id, entity_definition_id, compiled_sha256),
    CONSTRAINT numbering_epochs_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT numbering_epochs_epoch_check CHECK (epoch >= 1),
    CONSTRAINT numbering_epochs_compiled_sha256_check CHECK (length(compiled_sha256) = 32),
    CONSTRAINT numbering_epochs_literal_head_check
        CHECK (literal_head ~ '^[A-Z]{2,6}[A-Z0-9_./-]*$' AND length(literal_head) <= 64)
);
ALTER TABLE numbering_epochs ENABLE ROW LEVEL SECURITY;
ALTER TABLE numbering_epochs FORCE ROW LEVEL SECURITY;

-- No UPDATE or DELETE policy: there is nothing to allow (C9).
CREATE POLICY numbering_epochs_select ON numbering_epochs FOR SELECT TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY numbering_epochs_insert ON numbering_epochs FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose Down

DROP TABLE numbering_epochs;
DROP TABLE number_format_versions;
DROP TABLE number_formats;
