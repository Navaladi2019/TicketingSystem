-- Removes the 00005 indexes so 03_plans.sql measures the schema before them.
DROP INDEX outbox_purge_dispatched, outbox_purge_dead, workflow_ref_purge_closed,
           idempotency_keys_purge_expires;
