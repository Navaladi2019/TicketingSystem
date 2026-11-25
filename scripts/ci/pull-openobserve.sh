#!/bin/sh
# TICK-1128: pulls OpenObserve by the digest compose.yaml pins, from the first
# source that serves it: the repository compose.yaml names (ECR Public), then
# Docker Hub, where OpenObserve publishes the same index. A source that
# refuses, gives no answer within the time limit, or whose image has another
# digest, counts as failed. If both fail the script prints both errors and
# exits 1.
#
# The repository that served goes to GITHUB_ENV as OPENOBSERVE_REPO;
# compose.yaml adds its own tag and digest (TICK-1131). The stack start then
# finds the image locally and contacts no registry for it. Docker ignores the
# tag of a tag@digest ref, so the pulled binary's version must also be the
# tag's, or a tag bumped without its digest would run the old version
# unnoticed.
#
# Inputs: COMPOSE_YAML (default compose.yaml) and PULL_TIMEOUT (seconds per
# pull), both for the tests; GITHUB_ENV.
# docs/quality/ci-pipeline.md "OpenObserve image".
set -eu
cd "$(dirname "$0")/../.."

second=docker.io/openobserve/openobserve
limit=${PULL_TIMEOUT:-180}

log() { echo "pull-openobserve: $*" >&2; }

pin=$(sed -n 's/^ *image: *\${OPENOBSERVE_REPO:-\([^}]*\)}\(:[^[:space:]]*\).*/\1\2/p' "${COMPOSE_YAML:-compose.yaml}")
if ! printf '%s\n' "$pin" | grep -Eqx '[^:@ ]+:[^:@ ]+@sha256:[0-9a-f]{64}'; then
    log "compose.yaml does not pin openobserve as \${OPENOBSERVE_REPO:-<repo>}:<tag>@sha256:<digest>"
    exit 1
fi
digest=${pin#*@}
tag=${pin%@*}
first=${tag%:*}
version=${tag##*:}
version=${version%-debug}

# fetch <repo>: pull it by the digest and check what was pulled; the error is
# printed and the status is 1 when this source failed.
fetch() {
    status=0
    out=$(timeout -k 10 "$limit" docker pull -q "$1@$digest" 2>&1) || status=$?
    case $status in
    0) ;;
    # 124: stopped at the limit; 137: it ignored that and was killed.
    124 | 137)
        log "$1 failed: no answer in $limit s"
        return 1
        ;;
    *)
        log "$1 failed: $out"
        return 1
        ;;
    esac
    got=$(docker image inspect -f '{{join .RepoDigests "\n"}}' "$1@$digest" 2>&1) || true
    # Docker lists a Docker Hub repository without its host.
    if ! printf '%s\n' "$got" | grep -Fqx -e "$1@$digest" -e "${1#docker.io/}@$digest"; then
        log "$1 failed: its image is not $digest but: $got"
        return 1
    fi
}

if fetch "$first"; then
    repo=$first
elif fetch "$second"; then
    repo=$second
else
    log "both sources failed (errors above): $first and $second"
    exit 1
fi

ref=$repo@$digest
# The bytes are pinned, so another source cannot fix a version mismatch.
got=$(docker run --rm --network none --entrypoint /openobserve "$ref" --version 2>&1) || true
if [ "$got" != "openobserve $version" ]; then
    log "$ref says '$got', but compose.yaml's tag says $version; change the tag and the digest together"
    exit 1
fi

[ -z "${GITHUB_ENV:-}" ] || echo "OPENOBSERVE_REPO=$repo" >>"$GITHUB_ENV"
log "using $ref ($version)"
