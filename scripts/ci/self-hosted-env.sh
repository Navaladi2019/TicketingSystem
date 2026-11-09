#!/bin/sh
# make ci-self-hosted-env ID=<id> (TICK-641): on the self-hosted runner, the
# Docker names and host ports of one workflow run, so it never meets the
# owner's own stack (compose project nava, images nava/*:dev, ports 18080...)
# or another run on the same Docker host:
#   COMPOSE_PROJECT_NAME  nava-<id>  (containers, networks, volumes; the
#                         integration clusters' nava.ci.owner label)
#   NAVA_IMAGE_TAG        <id>       (every nava/* image the run builds)
#   KEYCLOAK_PUBLIC_URL   http://localhost:<its Keycloak port> (the issuer)
#   host ports            a free slot of 14 in 22000-28999 derived from <id>:
#                         nava-core, Keycloak, Temporal, Temporal UI,
#                         OpenObserve, Prometheus, and the acceptance tunnels
#                         (onboard, six admin listeners, Temporal UI) of
#                         docs/acceptance/guide/tools/tunnels.sh
#   DOCKER_CONFIG         ~/.docker-ci: no credential store (see below)
#   NAVA_IT_GO_TIMEOUT    20m: go test -timeout of the integration shards (the
#                         job shares the host with other jobs; 6m timed out)
# The workflow passes ID=ci-<run id>; the self-hosted-cleanup job removes
# exactly these names (scripts/ci/self-hosted-cleanup.sh). Lines go to
# $GITHUB_ENV, or to stdout outside Actions.
# ci-isolated.sh (laptops, 20000-21999) does the same; the ranges differ, so
# the two never pick the same slot.
set -eu

[ $# -eq 1 ] || { echo "usage: $0 <id>" >&2; exit 2; }
id=$1
case $id in
    "" | dev | *[!a-z0-9_-]* | [!a-z0-9]*) echo "ci-self-hosted-env: ID '$id': lowercase letters, digits, - and _, not dev" >&2; exit 2 ;;
esac

# 14 host ports from a slot chosen by the id's checksum; the next slot when
# a running container already publishes one of them.
taken() { [ -n "$(docker ps -q --filter "publish=$1")" ]; }
slot=$(( $(printf '%s' "$id" | cksum | cut -d ' ' -f 1) % 500 ))
tries=0
while :; do
    base=$(( 22000 + slot * 14 ))
    free=1
    for i in 0 1 2 3 4 5 6 7 8 9 10 11 12 13; do taken $((base + i)) && { free=0; break; }; done
    [ $free = 0 ] || break
    tries=$((tries + 1))
    [ $tries -lt 500 ] || { echo "ci-self-hosted-env: no free port slot in 22000-28999" >&2; exit 1; }
    slot=$(( (slot + 1) % 500 ))
done

# DOCKER_CONFIG without a credential store: the owner's ~/.docker/config.json
# names "credsStore": "desktop.exe", a Windows program reached over WSL
# interop, and under four parallel jobs the call timed out ("error getting
# credentials", WSL "UtilAcceptVsock: accept4 failed 110") and failed the
# image build. CI pulls only public images, anonymously. buildx and contexts
# link back to ~/.docker, so the nava-ci builder is the same one (TICK-641).
ci_cfg=$HOME/.docker-ci
mkdir -p "$ci_cfg"
[ -s "$ci_cfg/config.json" ] || echo '{}' >"$ci_cfg/config.json"
for x in buildx contexts; do
    [ -e "$HOME/.docker/$x" ] || continue
    [ -e "$ci_cfg/$x" ] || ln -s "$HOME/.docker/$x" "$ci_cfg/$x" 2>/dev/null || true
done

out=${GITHUB_ENV:-/dev/stdout}
{
    echo "COMPOSE_PROJECT_NAME=nava-$id"
    echo "NAVA_IMAGE_TAG=$id"
    echo "NAVA_CORE_PORT=$base"
    echo "KEYCLOAK_PORT=$((base + 1))"
    # Keycloak's pinned issuer (KC_HOSTNAME, NAVA_OIDC_ISSUERS) on the run's
    # own port. On the default localhost:18081 the identity tests reached the
    # owner's own Keycloak there, whose keys signed nothing of this stack
    # (CI-5: invalid_token: kid).
    echo "KEYCLOAK_PUBLIC_URL=http://localhost:$((base + 1))"
    echo "TEMPORAL_TEST_PORT=$((base + 2))"
    echo "UI_PORT=$((base + 3))"
    echo "TEMPORAL_UI_PORT=$((base + 3))"
    echo "OPENOBSERVE_PORT=$((base + 4))"
    echo "PROMETHEUS_PORT=$((base + 5))"
    echo "ACCEPT_ONBOARD_PORT=$((base + 6))"
    echo "ACCEPT_ADMIN_PORT=$((base + 7))"
    echo "ACCEPT_UI_PORT=$((base + 13))"
    echo "NAVA_IT_GO_TIMEOUT=20m"
    echo "DOCKER_CONFIG=$ci_cfg"
} >>"$out"
echo "ci-self-hosted-env: project nava-$id, image tag $id, ports $base-$((base + 13))"
