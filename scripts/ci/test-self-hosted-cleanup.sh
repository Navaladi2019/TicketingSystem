#!/bin/sh
# make test-self-hosted-cleanup (TICK-641): scripts/ci/self-hosted-cleanup.sh
# <id> removes one run's containers (and their anonymous volumes), named
# volumes, networks and image tags, and nothing else: not another run's (an id
# that only starts with this one's), not an unlabelled volume or container
# (the owner's own Docker work), not another tag of the same image.
# Throwaway objects on an empty imported image, names of this run only; needs
# Docker, no build, no stack, no registry.
set -eu
cd "$(dirname "$0")/../.."

id="t641-$$-$(date +%s)"
p="nava-$id"
other="nava-${id}0"
repo="nava/zz-$id"
fail=0
check() { if eval "$2"; then echo "ok:   $1"; else echo "FAIL: $1" >&2; fail=1; fi; }
ctr_gone() { [ -z "$(docker ps -a -q --filter "name=^$1\$")" ]; }
vol_gone() { ! docker volume inspect "$1" >/dev/null 2>&1; }
net_gone() { ! docker network inspect "$1" >/dev/null 2>&1; }
img_gone() { [ -z "$(docker images -q "$1")" ]; }

# Everything this test makes, removed on exit whatever happened.
ctrs="$p-own-1 $p-int-own-1 nava-it-pg-$id-own $other-own-1 nava-it-pg-$id-other zz-$id-owner"
vols="$p-data $other-data zz-$id-owner-data"
nets="$p-default $p-int-default nava-it-net-$id-own $other-default"
imgs="$repo:$id $repo:dev $repo:${id}0 $p-svc:latest"
anon=""
cleanup() {
    docker rm -fv $ctrs >/dev/null 2>&1 || true
    docker volume rm -f $vols $anon >/dev/null 2>&1 || true
    docker network rm $nets >/dev/null 2>&1 || true
    docker image rm $imgs >/dev/null 2>&1 || true
}
trap cleanup EXIT

tar cf - --files-from /dev/null | docker import - "$repo:$id" >/dev/null
for t in "$repo:dev" "$repo:${id}0" "$p-svc:latest"; do docker tag "$repo:$id" "$t"; done
compose() { # compose <project> <name>: a container labelled as compose labels one
    printf '%s' "--label com.docker.compose.project=$1 --label com.docker.compose.service=zz --name $2"
}

# This run: a compose container with a named volume and an anonymous one, the
# overlay project's container and network, an integration cluster and network.
docker volume create --label "com.docker.compose.project=$p" "$p-data" >/dev/null
docker network create --label "com.docker.compose.project=$p" "$p-default" >/dev/null
# shellcheck disable=SC2046
docker create $(compose "$p" "$p-own-1") -v "$p-data:/data" -v /anon "$repo:$id" true >/dev/null
anon=$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/anon"}}{{.Name}}{{end}}{{end}}' "$p-own-1")
docker network create --label "com.docker.compose.project=$p-int" "$p-int-default" >/dev/null
# shellcheck disable=SC2046
docker create $(compose "$p-int" "$p-int-own-1") "$repo:$id" true >/dev/null
docker create --name "nava-it-pg-$id-own" --label "nava.ci.owner=$p" "$repo:$id" true >/dev/null
docker network create --label "nava.ci.owner=$p" "nava-it-net-$id-own" >/dev/null

# Not this run: an id that starts with this one's, and the owner's own
# unlabelled container and volume.
docker volume create --label "com.docker.compose.project=$other" "$other-data" >/dev/null
docker network create --label "com.docker.compose.project=$other" "$other-default" >/dev/null
# shellcheck disable=SC2046
docker create $(compose "$other" "$other-own-1") -v "$other-data:/data" "$repo:${id}0" true >/dev/null
docker create --name "nava-it-pg-$id-other" --label "nava.ci.owner=$other" "$repo:${id}0" true >/dev/null
docker volume create "zz-$id-owner-data" >/dev/null
docker create --name "zz-$id-owner" -v "zz-$id-owner-data:/data" "$repo:dev" true >/dev/null

out=$(sh scripts/ci/self-hosted-cleanup.sh "$id" 2>&1) && rc=0 || rc=$?
echo "$out"
check "cleanup $id exits 0 (nothing left)" '[ "$rc" -eq 0 ]'
check "compose container removed" 'ctr_gone "$p-own-1"'
check "its named volume removed" 'vol_gone "$p-data"'
check "its anonymous volume removed" '[ -n "$anon" ] && vol_gone "$anon"'
check "compose network removed" 'net_gone "$p-default"'
check "overlay project container removed" 'ctr_gone "$p-int-own-1"'
check "overlay project network removed" 'net_gone "$p-int-default"'
check "integration container removed" 'ctr_gone "nava-it-pg-$id-own"'
check "integration network removed" 'net_gone "nava-it-net-$id-own"'
check "image tag $repo:$id removed" 'img_gone "$repo:$id"'
check "compose default image name $p-svc removed" 'img_gone "$p-svc:latest"'
check "other run's container kept" '! ctr_gone "$other-own-1"'
check "other run's volume kept" '! vol_gone "$other-data"'
check "other run's network kept" '! net_gone "$other-default"'
check "other run's integration container kept" '! ctr_gone "nava-it-pg-$id-other"'
check "other run's tag $repo:${id}0 kept" '! img_gone "$repo:${id}0"'
check "the owner's unlabelled container kept" '! ctr_gone "zz-$id-owner"'
check "the owner's unlabelled volume kept" '! vol_gone "zz-$id-owner-data"'
check "the dev tag of the same image kept" '! img_gone "$repo:dev"'

out=$(sh scripts/ci/self-hosted-cleanup.sh "$id" 2>&1) && rc=0 || rc=$?
check "a second cleanup finds nothing and exits 0" '[ "$rc" -eq 0 ] && echo "$out" | grep -q "nothing of $p"'
out=$(sh scripts/ci/self-hosted-cleanup.sh dev 2>&1) && rc=0 || rc=$?
check "ID dev (the owner's tag) is refused" '[ "$rc" -eq 2 ]'

[ "$fail" -eq 0 ] || { echo "test-self-hosted-cleanup: FAIL" >&2; exit 1; }
echo "test-self-hosted-cleanup: ok"
