-- Insert cost of the candidate indexes: 100,000 copies of tenant M's incident
-- rows into tenant M, timed, rolled back. Superuser with
-- session_replication_role = replica (no audit or FK triggers), so this times
-- heap + index maintenance only. Three runs per index set.
\timing on
SET session_replication_role = replica;
SET max_parallel_workers_per_gather = 0;
BEGIN;
INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state, requester_id,
                     assignee_id, assignment_group_id, created_by, created_at, updated_at, data)
SELECT tenant_id, gen_random_uuid(), 1, entity_definition_id, 1, 'X' || number, state, requester_id,
       assignee_id, assignment_group_id, created_by, created_at, updated_at, data
  FROM records WHERE tenant_id = '00000000-0000-4000-8000-00000000000b' AND number LIKE 'INC%';
ROLLBACK;
