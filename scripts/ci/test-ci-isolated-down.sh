#!/bin/sh
# make test-ci-isolated-down (TICK-211): ci-isolated.sh --down <id> untags
# every nava/*:<id> tag, including one of an image no longer in its list, and
# nothing else. Throwaway tags on an empty imported image, a fresh id per run;
# needs Docker, no build, no stack, no registry.
set -eu
cd "$(dirname "$0")/../.."

id="t211-$$-$(date +%s)"
fail=0
check() { if eval "$2"; then echo "ok:   $1"; else echo "FAIL: $1" >&2; fail=1; fi; }
has() { [ -n "$(docker images -q "$1")" ]; }

# nava/zz:dev must not exist already: the test removes only what it made.
has nava/zz:dev && { echo "test-ci-isolated-down: nava/zz:dev exists, refusing to touch it" >&2; exit 1; }
made=""
cleanup() { for t in $made; do docker image rm "$t" >/dev/null 2>&1 || true; done; }
trap cleanup EXIT

tar cf - --files-from /dev/null | docker import - "nava/zz:$id" >/dev/null
made="nava/zz:$id nava/zz:${id}x nava/zz:dev"
docker tag "nava/zz:$id" "nava/zz:${id}x"
docker tag "nava/zz:$id" nava/zz:dev

out=$(sh scripts/ci/ci-isolated.sh --down "$id" 2>&1) && rc=0 || rc=$?
echo "$out"
check "--down $id exits 0 (teardown clean)" '[ "$rc" -eq 0 ]'
check "unlisted nava/zz:$id removed" '! has "nava/zz:$id"'
check "nava/zz:${id}x survives (other tag)" 'has "nava/zz:${id}x"'
check "nava/zz:dev survives (shared tag)" 'has nava/zz:dev'

rc=0; sh scripts/ci/ci-isolated.sh --down dev >/dev/null 2>&1 || rc=$?
check "--down dev refused (exit 2)" '[ "$rc" -eq 2 ]'
rc=0; sh scripts/ci/ci-isolated.sh --down DEV >/dev/null 2>&1 || rc=$?
check "--down DEV sanitised to dev and refused (exit 2)" '[ "$rc" -eq 2 ]'
check "nava/zz:dev still there after the refusals" 'has nava/zz:dev'

# Untag only: no prune and no forced removal anywhere in the script.
check "ci-isolated.sh has no prune, rmi or forced image rm" \
    '! grep -nE "prune|docker rmi|image rm +(-f|--force)" scripts/ci/ci-isolated.sh'

[ "$fail" -eq 0 ] && echo "test-ci-isolated-down: PASS" || { echo "test-ci-isolated-down: FAIL" >&2; exit 1; }
