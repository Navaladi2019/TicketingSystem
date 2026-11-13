-- P3-40 (TICK-709): validates cascade_start_definition_fkey in its own
-- transaction. It fails, and changes nothing, if a claim names a definition
-- its tenant does not hold (see 20261113123330).
--
-- FORCE ROW LEVEL SECURITY makes a bare VALIDATE vacuous. When the caller
-- owns both tables, PostgreSQL validates a foreign key with one SQL anti-join
-- (RI_Initial_Check), and that query is subject to the referencing table's
-- RLS: cascade_start is FORCEd and has no policy for nava_migrator, so the
-- scan sees no row and the key is marked valid over any orphan. Checked on
-- postgres:16.10: an orphan claim validated (docs/data/workflow-definitions.md;
-- pinned by TestP340CascadeStartDefinitionKeyOverExistingClaims). A CHECK is
-- validated by a heap scan and is not affected.
--
-- The same query reads the referenced table under its RLS too:
-- workflow_definitions is FORCEd as well, so without a policy there every
-- claim would look orphaned and the validation would always fail.
--
-- So this transaction gives nava_migrator a SELECT policy on both tables,
-- validates, and drops both before it commits: no other session ever sees
-- them, and the catalogue gate (catalog.Check) runs after the migrations and
-- sees none. CREATE POLICY takes ACCESS EXCLUSIVE on each table until the
-- commit, so claims, publishes and reads wait for the scan.
-- That is accepted here: no shipped build writes cascade_start (nava-core
-- wires pipeline.NoopSink until P3-42), so the table holds only demo rows and
-- the scan is a few milliseconds. lock_timeout makes the migration fail and be
-- re-run instead of queueing behind a long transaction.

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE POLICY cascade_start_migrator_validate ON cascade_start FOR SELECT TO nava_migrator USING (true);
CREATE POLICY workflow_definitions_migrator_validate ON workflow_definitions FOR SELECT TO nava_migrator USING (true);
ALTER TABLE cascade_start VALIDATE CONSTRAINT cascade_start_definition_fkey;
DROP POLICY workflow_definitions_migrator_validate ON workflow_definitions;
DROP POLICY cascade_start_migrator_validate ON cascade_start;

-- +goose Down

SELECT 1;
