#!/bin/sh
# make test-ci-isolated-down (TICK-211): ci-isolated.sh --down <id> untags
# every nava/*:<id> tag, including one of an image no longer in its list and
# one a stopped container of the killed run still uses, and nothing else; and
# removes the integration containers and network labelled with its project
# (nava.ci.owner, TICK-182), never another project's.
# Throwaway tags on an empty imported image, in a repo and id of this run
# only, so runs in parallel checkouts do not meet; needs Docker, no build, no
# stack, no registry.
set -eu
cd "$(dirname "$0")/../.."

id="t211-$$-$(date +%s)"
repo="nava/zz-$id"
old="nava/zzold-$id"
ctr="nava-$id-zzold-1"
# Integration objects of a killed run: this project's, and another's.
itc="nava-it-pg-$id-own"
itn="nava-it-net-$id-own"
otherc="nava-it-pg-$id-other"
othern="nava-it-net-$id-other"
fail=0
check() { if eval "$2"; then echo "ok:   $1"; else echo "FAIL: $1" >&2; fail=1; fi; }
has() { [ -n "$(docker images -q "$1")" ]; }

made=""
cleanup() {
    docker rm -f "$ctr" "$itc" "$otherc" >/dev/null 2>&1 || true
    docker network rm "$itn" "$othern" >/dev/null 2>&1 || true
    for t in $made; do docker image rm "$t" >/dev/null 2>&1 || true; done
}
trap cleanup EXIT

made="$repo:$id $repo:${id}x $repo:dev $old:$id"
tar cf - --files-from /dev/null | docker import - "$repo:$id" >/dev/null
docker tag "$repo:$id" "$repo:${id}x"
docker tag "$repo:$id" "$repo:dev"
# An image of its own, so the unlisted tag is its only tag: Docker refuses to
# remove it while a container uses it. On it, a stopped container of the
# killed run, labelled as compose labels one (compose down skips a container
# without a config hash).
tar cf - --files-from /dev/null | docker import -c "LABEL tick=211" - "$old:$id" >/dev/null
docker create --name "$ctr" \
    --label com.docker.compose.config-hash=0 \
    --label "com.docker.compose.project=nava-$id" \
    --label com.docker.compose.service=zzold \
    --label com.docker.compose.container-number=1 \
    --label com.docker.compose.oneoff=False \
    "$old:$id" true >/dev/null
docker create --name "$itc" --label "nava.ci.owner=nava-$id" "$repo:${id}x" true >/dev/null
docker network create --label "nava.ci.owner=nava-$id" "$itn" >/dev/null
docker create --name "$otherc" --label "nava.ci.owner=nava-${id}x" "$repo:${id}x" true >/dev/null
docker network create --label "nava.ci.owner=nava-${id}x" "$othern" >/dev/null

out=$(sh scripts/ci/ci-isolated.sh --down "$id" 2>&1) && rc=0 || rc=$?
echo "$out"
check "one --down $id exits 0 (teardown clean)" '[ "$rc" -eq 0 ]'
check "stopped project container removed" '[ -z "$(docker ps -a -q --filter "name=^$ctr\$")" ]'
check "unlisted $old:$id under that container removed" '! has "$old:$id"'
check "unlisted $repo:$id removed" '! has "$repo:$id"'
check "$repo:${id}x survives (other tag)" 'has "$repo:${id}x"'
check "$repo:dev survives (shared tag name)" 'has "$repo:dev"'
check "integration container of nava-$id removed" '[ -z "$(docker ps -a -q --filter "name=^$itc\$")" ]'
check "integration network of nava-$id removed" '[ -z "$(docker network ls -q --filter "name=^$itn\$")" ]'
check "integration container of nava-${id}x survives" '[ -n "$(docker ps -a -q --filter "name=^$otherc\$")" ]'
check "integration network of nava-${id}x survives" '[ -n "$(docker network ls -q --filter "name=^$othern\$")" ]'

rc=0; sh scripts/ci/ci-isolated.sh --down dev >/dev/null 2>&1 || rc=$?
check "--down dev refused (exit 2)" '[ "$rc" -eq 2 ]'
rc=0; sh scripts/ci/ci-isolated.sh --down DEV >/dev/null 2>&1 || rc=$?
check "--down DEV sanitised to dev and refused (exit 2)" '[ "$rc" -eq 2 ]'

# Untag only: no prune and no forced removal anywhere in the script.
check "ci-isolated.sh has no prune, rmi or forced image rm" \
    '! grep -nE "prune|docker rmi|image rm +(-f|--force)" scripts/ci/ci-isolated.sh'

[ "$fail" -eq 0 ] && echo "test-ci-isolated-down: PASS" || { echo "test-ci-isolated-down: FAIL" >&2; exit 1; }
