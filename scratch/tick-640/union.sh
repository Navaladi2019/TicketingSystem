#!/bin/sh
# TICK-640: the "mine" rule (requester OR assignee OR shared) decomposed into
# a UNION ALL of three index-ordered branches, each LIMITed over records_list_requester, records_list_assignee and the
# share join instead of walking records_list_updated with a filter. Each later
# branch excludes the earlier branches' rows, so no row repeats.
#   sh scratch/tick-640/union.sh <requester-uuid> [page20-cursor "ts,id"]
set -eu
PG=${PG:-docker exec -i tick640-pg psql -X -q -U pgadmin -d nava}
T=809d4580-aaed-4156-5abc-38d58f77f840 D=22a88e5c-3a76-b13d-e14d-5bdef46b3dc2 U=$1
ks=""
if [ -n "${2:-}" ]; then ks="AND (r.updated_at, r.id) < ('${2%,*}'::timestamptz, '${2#*,}'::uuid)"; fi
cols="r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth"
w="r.tenant_id = '$T' AND r.entity_definition_id = '$D' $ks"
# Each branch reads at most 51 rows in its index's order.
o="ORDER BY r.updated_at DESC, r.id DESC LIMIT 51"
share="EXISTS (SELECT 1 FROM record_shares s WHERE s.tenant_id = r.tenant_id AND s.record_id = r.id AND s.principal_kind = 'user' AND s.principal_id = '$U'::uuid)"
printf "BEGIN;\nSET LOCAL ROLE nava_app;\nSELECT set_config('app.tenant_id', '%s', true) \\g /dev/null\n%s\nCOMMIT;\n" "$T" "
EXPLAIN (ANALYZE, BUFFERS, SETTINGS)
SELECT x.*, d.key, coalesce(g.deleted_at IS NOT NULL, false)
FROM (
  (SELECT $cols FROM records r WHERE $w AND r.requester_id = '$U'::uuid $o)
  UNION ALL
  (SELECT $cols FROM records r WHERE $w AND r.assignee_id = '$U'::uuid AND r.requester_id IS DISTINCT FROM '$U'::uuid $o)
  UNION ALL
  (SELECT $cols FROM records r WHERE $w AND $share
     AND r.requester_id IS DISTINCT FROM '$U'::uuid AND r.assignee_id IS DISTINCT FROM '$U'::uuid $o)
  ORDER BY updated_at DESC, id DESC LIMIT 51
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.updated_at DESC, x.id DESC;" | $PG -At
