#!/bin/sh
# TICK-640 (P3-34b): EXPLAIN (ANALYZE, BUFFERS, SETTINGS${TIMING:+, TIMING $TIMING}) of the list
# statements in the shape the P3-34a / P3-34b shared contract fixes, over the
# seeded 800k database (setup.go). Runs as nava_app (SET LOCAL ROLE, forced
# RLS, app.tenant_id set) in the large tenant.
#
#   sh scratch/tick-640/explain.sh <actor> <sort> <page> [count]
#
# actor: agent | req1 | req8 | admin   (P3-36's actors: agent-1, the heaviest
#        and the lightest of the 8 heavy requesters, the admin)
# sort:  [-]updated_at | [-]created_at | [-]state | [-]requester_id |
#        [-]assignee_id | [-]assignment_group_id   ("-" = descending)
# page:  1 | 20 (keyset page 20: the 951st row onwards, cursor from OFFSET 950)
# count: also EXPLAIN the count(*) statement.
# Env: PG (default "docker exec -i tick640-pg psql -X -q -U pgadmin -d nava"),
#      FILTER (an extra AND predicate over r, e.g. "r.state = 'new'"),
#      JIT (on|off, default: the server's).
set -eu
PG=${PG:-docker exec -i tick640-pg psql -X -q -U pgadmin -d nava}
actor=$1 sort=$2 page=$3 count=${4:-}
T=809d4580-aaed-4156-5abc-38d58f77f840   # md5('t0')
D=22a88e5c-3a76-b13d-e14d-5bdef46b3dc2   # md5('def/0')
U31=59fb22d0-8db4-ebf3-0411-bc5e59f13344 # agent-1 = user 31
AGENTS=9ffce347-7338-3b1b-28d1-0bcfcc2519cd
G31=969c31f2-d8fe-635a-5a2d-95d39e4e7489 G48=2f067c2d-b39a-c447-6bd8-6f26d53e8105 G15=e43d6642-9bbe-05be-b6a6-8d3a87bf4141

mine() { # mine <user> <groups>: the compiled "mine" rule (requester, assignee, shared)
    echo "((r.requester_id IS NOT NULL AND r.requester_id = '$1'::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = '$1'::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = '$1'::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY('$2'::uuid[])))))"
}
case $actor in
agent) acl="(((true) AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY('{$G31,$G48,$G15}'::uuid[]))) OR ($(mine $U31 "{$AGENTS,$G31,$G48,$G15}")))" ;;
req1) acl="($(mine fccbb3fd-048d-a9ac-ec41-5c1548c480a0 '{}'))" ;;
req8) acl="($(mine f222cde6-739e-cb4e-eaa6-14c1a2bed245 '{}'))" ;;
admin) acl="(($(mine $U31 '{}')) OR ((true)))" ;;
*) echo "unknown actor $actor" >&2; exit 2 ;;
esac
[ -z "${FILTER:-}" ] || acl="$acl AND ${FILTER}"
col=${sort#-}
if [ "$col" = "$sort" ]; then dir=ASC nulls=LAST cmp='>'; else dir=DESC nulls=FIRST cmp='<'; fi
base="FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = '$T' AND r.entity_definition_id = '$D' AND $acl"
# TIE=updated_at: a state or foreign-key sort orders <col>, updated_at, id
# (this task's amendment to the contract), served by the access-path index.
tie=""
case $col in *_at) ;; *) [ -z "${TIE:-}" ] || tie="r.$TIE $dir, " ;; esac
order="ORDER BY r.$col $dir NULLS $nulls, ${tie}r.id $dir"
cols="r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false), r.$col::text"

tx() { # tx <sql>: run as nava_app in the large tenant
    printf "BEGIN;\nSET LOCAL ROLE nava_app;\nSELECT set_config('app.tenant_id', '%s', true) \\\\g /dev/null\n%s%s\nCOMMIT;\n" \
        "$T" "${JIT:+SET LOCAL jit = $JIT;
}" "$1" | $PG -At
}

keyset=""
if [ "$page" = 20 ]; then
    tv=""; [ -z "$tie" ] || tv="|| ',' || quote_literal(r.$TIE::text)"
    cur=$(tx "SELECT quote_nullable(r.$col::text) $tv || ',' || quote_literal(r.id::text) $base $order OFFSET 950 LIMIT 1;")
    v=${cur%%,*} id=${cur##*,}
    typ=$(case $col in *_at) echo timestamptz ;; state) echo text ;; *) echo uuid ;; esac)
    if [ -n "$tie" ]; then
        t=${cur#*,}; t=${t%,*}
        if [ "$v" = NULL ]; then # the NULL segment's branch: col IS NULL, then (tie, id)
            keyset=" AND r.$col IS NULL AND (r.$TIE, r.id) $cmp ($t::timestamptz, $id::uuid)"
        else
            keyset=" AND (r.$col, r.$TIE, r.id) $cmp ($v::$typ, $t::timestamptz, $id::uuid)"
        fi
    elif [ "$v" = NULL ]; then
        keyset=" AND r.$col IS NULL AND r.id $cmp $id::uuid"
    else
        keyset=" AND (r.$col, r.id) $cmp ($v::$typ, $id::uuid)"
    fi
fi
echo "## $actor sort=$sort page=$page${FILTER:+ filter: $FILTER}${JIT:+ jit=$JIT}"
echo '```sql'
echo "SELECT ... $base$keyset $order LIMIT 51"
echo '```'
echo '```'
tx "EXPLAIN (ANALYZE, BUFFERS, SETTINGS${TIMING:+, TIMING $TIMING}) SELECT $cols $base$keyset $order LIMIT 51;"
echo '```'
if [ -n "$count" ]; then
    echo "count:"
    echo '```'
    tx "EXPLAIN (ANALYZE, BUFFERS, SETTINGS${TIMING:+, TIMING $TIMING}) SELECT count(*) FROM records r WHERE r.tenant_id = '$T' AND r.entity_definition_id = '$D' AND $acl;"
    echo '```'
fi
