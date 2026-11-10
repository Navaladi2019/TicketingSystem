#!/bin/sh
# TICK-656 (P3-36d): EXPLAIN (ANALYZE, BUFFERS, SETTINGS) of the list
# statements exactly as store.List emits them since TICK-644 (CountStatement
# and the branched PageStatement, text copied from P3-36's final re-run,
# docs/quality/acl-perf-800k/final-2026-11-09-full.md), with the arguments
# inlined (the handler runs plan_cache_mode = force_custom_plan, so the plan is
# the custom plan of these constants). Runs as nava_app in the large tenant of
# scratch/tick-640's seed.
#
#   sh scratch/tick-656/explain.sh <scenario> [page]
#
# scenario: agent-state-new | agent | agent-group-desc | agent-group-asc |
#           admin-state-in
# page:     1 (default) | 20 (keyset from the 951st row, as matrix.sh)
# Env: PG (default "docker exec -i tick656-pg psql -X -q -U pgadmin -d nava").
set -eu
PG=${PG:-docker exec -i tick656-pg psql -X -q -U pgadmin -d nava}
scenario=$1 page=${2:-1}
T=809d4580-aaed-4156-5abc-38d58f77f840   # md5('t0'), the 400k tenant
D=22a88e5c-3a76-b13d-e14d-5bdef46b3dc2   # md5('def/0')
U=59fb22d0-8db4-ebf3-0411-bc5e59f13344   # agent-1 = user 31
# agent-1's groups, as P3-36 binds them ($3 and $7): agents, g31, g48, g15
G="'{9ffce347-7338-3b1b-28d1-0bcfcc2519cd,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]"
ADMIN=$U

grp="(true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($G)))"
req="(r.requester_id IS NOT NULL AND r.requester_id = '$U'::uuid)"
asg="(r.assignee_id IS NOT NULL AND r.assignee_id = '$U'::uuid)"
shr="EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '$U'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($G))))"
# authz.ReadBranches for the agent: group, then requester / assignee / shared
# not already covered (mutually exclusive).
B1="(${grp})"
B2="((${req}) AND NOT ${grp})"
B3="((${asg}) AND NOT ${grp} AND NOT (${req}))"
B4="((${shr}) AND NOT ${grp} AND NOT (${req}) AND NOT (${asg}))"

tx() { # tx <sql>: run as nava_app in the large tenant
    printf "BEGIN;\nSET LOCAL ROLE nava_app;\nSET LOCAL plan_cache_mode = force_custom_plan;\nSELECT set_config('app.tenant_id', '%s', true) \\\\g /dev/null\n%s\nCOMMIT;\n" "$T" "$1" | $PG -At
}

where=true col=updated_at desc=1 branched=1
case $scenario in
agent-state-new) where="r.state = ANY('{new}'::text[])" ;;
agent) ;;
agent-group-desc) col=assignment_group_id ;;
agent-group-asc) col=assignment_group_id desc= ;;
admin-state-in) branched= where="((((r.requester_id IS NOT NULL AND r.requester_id = '$ADMIN'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '$ADMIN'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '$ADMIN'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('{}'::uuid[])))))) OR ((true))) AND r.state = ANY('{new,in_progress}'::text[])" ;;
*) echo "unknown scenario $scenario" >&2; exit 2 ;;
esac
if [ -n "$desc" ]; then dir=DESC nulls="NULLS FIRST" cmp='<'; else dir=ASC nulls="NULLS LAST" cmp='>'; fi
order="$dir $nulls"
tie=""
[ "$col" = updated_at ] || { order="$order, r.updated_at $dir"; tie=1; }
order="$order, r.id $dir"
pre="r.tenant_id = '$T' AND r.entity_definition_id = '$D' AND $where"
acl="($B1 OR $B2 OR $B3 OR $B4)"
[ -n "$branched" ] || acl=true

keyset=""
if [ "$page" = 20 ]; then
    tv=""; [ -z "$tie" ] || tv="|| ',' || quote_literal(r.updated_at::text)"
    cur=$(tx "SELECT quote_nullable(r.$col::text) $tv || ',' || quote_literal(r.id::text) FROM records r WHERE $pre AND $acl ORDER BY r.$col $order OFFSET 950 LIMIT 1;")
    v=${cur%%,*} id=${cur##*,}
    typ=$(case $col in *_at) echo timestamptz ;; *) echo uuid ;; esac)
    if [ -n "$tie" ]; then
        t=${cur#*,}; t=${t%,*}
        keyset=" AND ((r.$col, r.updated_at, r.id) $cmp ($v::$typ, $t::timestamptz, $id::uuid))"
    else
        keyset=" AND ((r.$col, r.id) $cmp ($v::$typ, $id::uuid))"
    fi
fi

if [ -n "$branched" ]; then
    count="SELECT count(*) FROM (SELECT 1 FROM ("
    page_sql="SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM ("
    sep=""
    for b in "$B1" "$B2" "$B3" "$B4"; do
        count="$count$sep(SELECT 1 FROM records r WHERE $pre AND $b LIMIT 1001)"
        page_sql="$page_sql${sep:+
UNION ALL
}(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.$col AS sort_key FROM records r
WHERE $pre AND $b$keyset ORDER BY r.$col $order LIMIT 51)"
        sep=" UNION ALL "
    done
    count="$count) u LIMIT 1001) c"
    outer=$(echo "$order" | sed 's/r\.updated_at/updated_at/; s/r\.id/id/')
    xouter=$(echo "$outer" | sed 's/updated_at/x.updated_at/; s/\bid\b/x.id/; s/, id /, x.id /')
    page_sql="$page_sql
ORDER BY sort_key $outer LIMIT 51
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key $xouter"
else
    count="SELECT count(*) FROM (SELECT 1 FROM records r WHERE $pre LIMIT 1001) c"
    page_sql="SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.$col::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE $pre$keyset ORDER BY r.$col $order LIMIT 51"
fi

echo "## $scenario page=$page"
if [ "$page" = 1 ]; then
    echo "count:"
    echo '```'
    tx "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) $count;"
    echo '```'
fi
echo "page:"
echo '```'
tx "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) $page_sql;"
echo '```'
