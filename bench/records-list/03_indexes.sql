-- The proposed index set (for P3-09). Every key leads with tenant_id.
\timing on
SET maintenance_work_mem = '512MB';
SET ROLE bench_owner;
-- Default list order; the walk for (c), and for (d) when the planner walks.
CREATE INDEX records_tenant_updated ON records (tenant_id, updated_at DESC, id DESC);
-- (b) and the requester branch of (d).
CREATE INDEX records_tenant_requester ON records (tenant_id, requester_id, updated_at DESC, id DESC);
-- The assignee and group branches of (d), and (a). Open rows only (15%).
CREATE INDEX records_tenant_assignee_open ON records (tenant_id, assignee_id, updated_at DESC, id DESC)
    WHERE state <> 'closed';
CREATE INDEX records_tenant_group_open ON records (tenant_id, assignment_group_id, updated_at DESC, id DESC)
    WHERE state <> 'closed';
RESET ROLE;
ANALYZE records;
