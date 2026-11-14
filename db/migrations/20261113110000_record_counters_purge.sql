-- S18 P6-10 (TICK-675), ADR-005 §4.3 and §5: the mandatory purge of period
-- counter rows. Paired nava_purge SELECT and DELETE policies on one
-- predicate, the outbox pattern (00002): purge sees exactly the rows it may
-- delete, a period counter whose period_end is more than 2 days past.
--
-- What the predicate leaves out, by construction:
--   - open periods and periods inside the 2-day grace (period_end >= the
--     cutoff). The grace exceeds the widest UTC-offset span, 26 h, so a
--     time-zone change cannot reopen a purged period (ADR-005 §4.3);
--   - field-only and default-scope rows (period_end NULL): NULL < x is not
--     true, so no NULL row matches.
--
-- Allow-listed in internal/db/catalog.AllowList (ADR-001 A.3), pinned to this
-- exact deparsed predicate. The grant (db/grants.yaml) is SELECT on the key
-- columns and period_end, and DELETE; never next. nava_app keeps no DELETE.
-- The batch DELETE (internal/purge) reads record_counters_period_end
-- (20261112193300), partial on period_end IS NOT NULL.
--
-- Catalogue only: no rewrite, no index. CREATE POLICY takes ACCESS EXCLUSIVE
-- on record_counters, which every create touches, so lock_timeout fails the
-- migration after 2 s instead of queueing every create behind it (ADR-005
-- §4.2); the migrator retries a 55P03 (TICK-579), then re-run it when the
-- table is quieter.

-- +goose Up

SET LOCAL lock_timeout = '2s';

CREATE POLICY record_counters_purge_select ON record_counters FOR SELECT TO nava_purge
    USING (period_end < now() - interval '2 days');
CREATE POLICY record_counters_purge ON record_counters FOR DELETE TO nava_purge
    USING (period_end < now() - interval '2 days');

-- +goose Down

SET LOCAL lock_timeout = '2s';

DROP POLICY record_counters_purge ON record_counters;
DROP POLICY record_counters_purge_select ON record_counters;
