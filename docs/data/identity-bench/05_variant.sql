-- Run as pgadmin after the main run: compare the migration's membership
-- index, (tenant_id, principal_id), with a covering three-column
-- (tenant_id, principal_id, group_id) that allows an index-only scan. Both
-- freshly built; the three-column one then replaces it for the rerun.
\pset pager off
\timing on
REINDEX INDEX group_members_principal;
CREATE INDEX group_members_principal_3col ON group_members (tenant_id, principal_id, group_id);
\timing off
SELECT c.relname, pg_size_pretty(pg_relation_size(c.oid)) AS size
FROM pg_class c WHERE c.relname IN ('group_members_principal', 'group_members_principal_3col', 'group_members_pkey');
DROP INDEX group_members_principal;
ANALYZE group_members;
