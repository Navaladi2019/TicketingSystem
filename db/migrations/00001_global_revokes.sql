-- ADR-001 A.3 global settings. Runs as nava_migrator, the database owner.
-- No default GRANT anywhere: every grant comes from db/grants.yaml.

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database());
END
$$;
-- +goose StatementEnd

REVOKE ALL ON SCHEMA public FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE nava_migrator REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
