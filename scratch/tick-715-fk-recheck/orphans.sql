-- TICK-715: orphans of the two foreign keys whose VALIDATE ran vacuously
-- under FORCE ROW LEVEL SECURITY (20261104013500, 20261104043100).
-- Run as a role that bypasses RLS (the superuser), never as nava_migrator:
-- under FORCE RLS the owner sees no row and would report 0.
-- MATCH SIMPLE: a row with a NULL key column is not checked.
SELECT 'group_members_group_live_fkey' AS fkey, count(*) AS orphans
  FROM group_members gm
 WHERE NOT EXISTS (SELECT 1 FROM groups g
                    WHERE g.tenant_id = gm.tenant_id AND g.id = gm.group_id AND g.live = gm.live)
UNION ALL
SELECT 'records_assignment_group_fkey', count(*)
  FROM records r
 WHERE r.assignment_group_id IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM groups g
                    WHERE g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id);
