-- S21 acceptance (TICK-749): what a JSON scan of the records gives, computed
-- once after seed.sql and before the Bruno folder, as `name=value` lines that
-- run-kit.sh passes to Bruno as --env-var. The Bruno requests compare the
-- filtered lists against these (criteria 1, 2 and 5). The scan reads `data`
-- directly (data->>'priority', data->>'due_at'), never the side tables.
--
-- "agent" is what Alice (agent@tenant-a) may read under tenant A's default
-- incident ACL: records assigned to one of her groups (service-desk), or that
-- she raised (none are). The md5 is over the ids as text, sorted, joined by
-- ',' (Bruno sorts the same strings with the same comparison).
\set ON_ERROR_STOP on
CREATE TEMP TABLE s21_a AS
SELECT r.id, r.data, r.assignment_group_id, r.requester_id
  FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
 WHERE r.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND d.key = 'incident';
CREATE TEMP TABLE s21_alice AS
SELECT p.id AS alice, array_agg(gm.group_id) AS groups
  FROM principals p LEFT JOIN group_members gm ON gm.tenant_id = p.tenant_id AND gm.principal_id = p.id
 WHERE p.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND p.email = 'alice@tenant-a.test'
 GROUP BY p.id;
CREATE TEMP VIEW s21_agent AS
SELECT a.* FROM s21_a a, s21_alice x WHERE a.assignment_group_id = ANY (x.groups) OR a.requester_id = x.alice;

\pset format unaligned
\pset tuples_only on
SELECT 's21_p1_admin_n=' || count(*) || E'\n' || 's21_p1_admin_md5=' || md5(coalesce(string_agg(id::text, ',' ORDER BY id::text COLLATE "C"), ''))
  FROM s21_a WHERE data->>'priority' = '1';
SELECT 's21_p1_agent_n=' || count(*) || E'\n' || 's21_p1_agent_md5=' || md5(coalesce(string_agg(id::text, ',' ORDER BY id::text COLLATE "C"), ''))
  FROM s21_agent WHERE data->>'priority' = '1';
SELECT 's21_ac2_admin_n=' || count(*) || E'\n' || 's21_ac2_admin_md5=' || md5(coalesce(string_agg(id::text, ',' ORDER BY id::text COLLATE "C"), ''))
  FROM s21_a WHERE (data->>'priority')::numeric >= 2 AND (data->>'due_at')::timestamptz < timestamptz '2026-01-01T12:00:00Z';
SELECT 's21_ac2_agent_n=' || count(*) || E'\n' || 's21_ac2_agent_md5=' || md5(coalesce(string_agg(id::text, ',' ORDER BY id::text COLLATE "C"), ''))
  FROM s21_agent WHERE (data->>'priority')::numeric >= 2 AND (data->>'due_at')::timestamptz < timestamptz '2026-01-01T12:00:00Z';
SELECT 's21_p3_admin_n=' || count(*) FROM s21_a WHERE data->>'priority' = '3';
SELECT 's21_p3_agent_n=' || count(*) FROM s21_agent WHERE data->>'priority' = '3';
SELECT 's21_a_n=' || count(*) FROM s21_a;
SELECT 's21_agent_n=' || count(*) FROM s21_agent;
SELECT 's21_hidden_admin_n=' || count(*) FROM s21_a WHERE data->>'due_at' = '2026-12-31T23:59:59Z';
SELECT 's21_hidden_agent_n=' || count(*) FROM s21_agent WHERE data->>'due_at' = '2026-12-31T23:59:59Z';
SELECT 's21_control_agent_n=' || count(*) FROM s21_agent WHERE data->>'due_at' = '2026-12-31T23:57:59Z';
SELECT 's21_nowhere_n=' || count(*) FROM records WHERE data->>'due_at' = '2026-12-31T23:58:59Z';
-- Tenant B's priority-1 incidents, and the scan of every tenant: the
-- difference is what a tenant-blind filter would wrongly return to tenant A.
SELECT 's21_b_p1_n=' || count(*)
  FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
 WHERE r.tenant_id = '00000000-0000-4000-8000-0000000000b1' AND d.key = 'incident' AND r.data->>'priority' = '1';
SELECT 's21_p1_all_tenants_n=' || count(*) FROM records WHERE data->>'priority' = '1';
-- The records criterion 4 edits: priority 4, due later than October, no group,
-- so neither criterion 1's nor criterion 2's expected rows move.
SELECT 's21_edit_ids=' || string_agg(id::text, ',' ORDER BY id::text COLLATE "C")
  FROM (SELECT id FROM s21_a WHERE data->>'priority' = '4' AND (data->>'due_at')::timestamptz >= timestamptz '2026-10-01T00:00:00Z'
          AND assignment_group_id IS NULL ORDER BY id::text COLLATE "C" LIMIT 6) e;
