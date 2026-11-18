-- PLAN-011 D6 check (TICK-933 item 4), verbatim from the issue: worst-case
-- read branches per newest document, format 1. Read-only.
BEGIN READ ONLY;
with newest as (
  select distinct on (tenant_id, entity_definition_id) tenant_id, entity_definition_id, version, rules
  from acl_versions order by tenant_id, entity_definition_id, version desc
), per_rule as (
  select n.tenant_id, n.entity_definition_id,
    greatest(
      case when jsonb_array_length(coalesce(r->'who'->'groups','[]'))+jsonb_array_length(coalesce(r->'who'->'users','[]'))>0 then 1 else 0 end,
      coalesce((r->'who'->>'requester')::bool,false)::int+coalesce((r->'who'->>'assignee')::bool,false)::int+coalesce((r->'who'->>'shared')::bool,false)::int
    ) as branches
  from newest n, jsonb_array_elements(case when jsonb_typeof(n.rules)='array' then n.rules else '[]' end) r
  where r->'ops' ? 'read'
)
select count(distinct tenant_id) tenants, max(t) max_total_branches
from (select tenant_id, entity_definition_id, sum(branches) t from per_rule group by 1,2) x;
-- Context: how many newest documents, and how many are format 2 (none expected).
select count(*) newest_docs, count(*) filter (where jsonb_typeof(rules) = 'object') format2_docs,
       count(distinct tenant_id) tenants_with_acl
from (select distinct on (tenant_id, entity_definition_id) tenant_id, rules
      from acl_versions order by tenant_id, entity_definition_id, version desc) n;
ROLLBACK;
