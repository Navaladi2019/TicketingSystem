-- The newest stored access document of every (tenant, entity type), format 1
-- or 2, one line each for tools/acceptance/acld6 (psql -At -F '|').
-- Read-only, as the superuser (sees every tenant).
BEGIN READ ONLY;
select distinct on (a.tenant_id, a.entity_definition_id)
       a.tenant_id, d.key, a.version, a.rules::text
  from acl_versions a join entity_definitions d on d.id = a.entity_definition_id
 order by a.tenant_id, a.entity_definition_id, a.version desc;
ROLLBACK;
