\pset pager off
SELECT c.relname, pg_size_pretty(pg_relation_size(c.oid)) AS size
FROM pg_class c
WHERE c.relname IN ('principals', 'groups', 'group_members', 'org_units')
   OR c.oid IN (SELECT indexrelid FROM pg_index WHERE indrelid IN
       ('principals'::regclass, 'groups'::regclass, 'group_members'::regclass, 'org_units'::regclass))
ORDER BY pg_relation_size(c.oid) DESC;
SELECT version();
SHOW shared_buffers;
