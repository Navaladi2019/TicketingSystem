-- TICK-640 (P3-34b): JIT off for every session on the platform database.
--
-- Why: the platform database serves OLTP only. JIT compiles a statement once
-- its estimated cost passes jit_above_cost (100k), and a custom plan
-- (nava_app runs force_custom_plan, pgx QueryExecModeExec) is compiled
-- again on every execution. On P3-01's 800k shape a list count(*) costs
-- about 1.3M and JIT added 300-330 ms to each one; P3-36 measured 200-430 ms
-- on the list pages. No statement here runs long enough to pay that back.
-- Rationale and numbers: db/README.md (JIT), docs/data/records-list-indexes.md.
--
-- Why the database and not the roles: ALTER ROLE ... SET needs CREATEROLE,
-- which nava_migrator lacks, and db-roles resets any role setting
-- db/grants.yaml does not list (grants.AllowedSettings knows only
-- plan_cache_mode). The migrator owns the database, so ALTER DATABASE ...
-- SET is its own to run. It covers nava_app, nava_worker and every other
-- role on this database, none of which runs analytical queries. A session
-- may still SET jit = on. New sessions pick it up: pools pick it up as they
-- reconnect.
--
-- current_database(): the migrator runs on the platform database, whatever
-- its name.

-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    EXECUTE format('ALTER DATABASE %I SET jit = off', current_database());
END
$$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DO $$
BEGIN
    EXECUTE format('ALTER DATABASE %I RESET jit', current_database());
END
$$;
-- +goose StatementEnd
