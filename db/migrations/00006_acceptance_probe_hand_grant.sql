-- THROWAWAY (TICK-44 acceptance): a grant added by hand, not in
-- db/grants.yaml. CI must fail on it (ADR-001 CI-4). Do not merge.

-- +goose Up
GRANT SELECT ON records TO nava_relay;

-- +goose Down
REVOKE SELECT ON records FROM nava_relay;
