-- S29 acceptance (TICK-754): what the filter must give, computed once after
-- seed.sql and kit.mjs held and before the Bruno folder, as `name=value` lines
-- run-kit.sh passes to Bruno as --env-var. Reads `data` directly, never the
-- side tables.
--
--   s29_p3_n / s29_p3_md5   tenant A incidents Alice may read whose priority,
--                           with the default 3 applied, is 3: no stored
--                           priority, or a stored 3 (criterion 1)
--   s29_missing_n           tenant A incidents with no stored priority (all
--                           of them, the poison record included)
--   s29_sample_missing      10 of the seeded records (ordinals 1 to 10)
--   s29_poison_id           the seeded record that breaks maxLength
-- The md5 is over the ids as text, sorted, joined by ',' (as lib/kit.js md5).
\set ON_ERROR_STOP on
CREATE TEMP TABLE s29_a AS
SELECT r.id, r.data, r.assignment_group_id, r.requester_id
  FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
 WHERE r.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND d.key = 'incident';
CREATE TEMP TABLE s29_alice AS
SELECT p.id AS alice, array_agg(gm.group_id) AS groups
  FROM principals p LEFT JOIN group_members gm ON gm.tenant_id = p.tenant_id AND gm.principal_id = p.id
 WHERE p.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND p.email = 'alice@tenant-a.test'
 GROUP BY p.id;

\pset format unaligned
\pset tuples_only on
SELECT 's29_p3_n=' || count(*) || E'\n' || 's29_p3_md5=' || md5(coalesce(string_agg(a.id::text, ',' ORDER BY a.id::text COLLATE "C"), ''))
  FROM s29_a a, s29_alice x
 WHERE (a.assignment_group_id = ANY (x.groups) OR a.requester_id = x.alice)
   AND (NOT a.data ? 'priority' OR a.data->>'priority' = '3');
SELECT 's29_missing_n=' || count(*) FROM s29_a WHERE NOT data ? 'priority';
SELECT 's29_sample_missing=' || string_agg(md5('s29a/' || n)::uuid::text, ',' ORDER BY n) FROM generate_series(1, 10) n;
SELECT 's29_poison_id=' || md5('s29a/2000')::uuid;
