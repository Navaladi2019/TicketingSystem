-- S17 P6-01 (TICK-665), ADR-005 §4.2 step M2 and §4.3: the index nava_purge's
-- batch DELETE (WHERE period_end < now() - interval '2 days') reads, partial
-- on "period_end IS NOT NULL" so field-only and default-scope rows, which are
-- never purged, are never in it. Measured (ADR-005 spike, 912k rows): 11.6 ms
-- per 1,000-row batch and 0.63 ms for the empty last batch, against 210+ ms
-- without it; 6.5 MB; 99.1 % of counter upserts stay HOT, because the upsert's
-- UPDATE arm sets next only and never period_end.
--
-- It does not lead with tenant_id: purge runs across tenants under its own
-- policies, as the 00005 purge indexes do. Every row R1 writes has
-- period_end NULL, so the index stays empty until S18 publishes period
-- formats. The purge policies and grants come with the purge change, not here.
--
-- Built CONCURRENTLY and re-runnable, as 20261112193200: an INVALID index left
-- by a failed build is dropped first, a VALID one is kept.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('record_counters_period_end') AND NOT indisvalid) THEN
        DROP INDEX record_counters_period_end;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS record_counters_period_end
    ON record_counters (period_end) WHERE period_end IS NOT NULL;

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS record_counters_period_end;
