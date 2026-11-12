-- P3-40 (TICK-709), D-3-29: the waiting state. record_changed_signal_id is
-- non-NULL exactly while the run waits on its version's
-- "signal:record_changed" step id; NextStep (P3-42) sets it when it returns a
-- wait_signal directive on record_changed and clears it otherwise. The sink
-- signals only refs where it is set (D-3-14).
--
-- ADD COLUMN with no default is a catalogue change under a short ACCESS
-- EXCLUSIVE lock: no rewrite, and every existing row reads NULL (not
-- waiting). lock_timeout makes the migration fail and be re-run instead of
-- queueing reads behind a long write. nava_worker gets UPDATE on this column
-- (db/grants.yaml); the existing workflow_ref_update policy holds the tenant.
-- Its index is built online by the next migration.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE workflow_ref ADD COLUMN record_changed_signal_id uuid;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE workflow_ref DROP COLUMN record_changed_signal_id;
