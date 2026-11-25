#!/bin/sh
# TICK-1128: pulls OpenObserve by the digest compose.yaml pins, from the first
# source that serves it: the repository compose.yaml names (ECR Public), then
# Docker Hub, where OpenObserve publishes the same index. A source that
# refuses, or whose image has another digest, counts as failed. If both fail
# the script prints both errors and exits 1.
#
# The image is used as <repo>@<digest> (OPENOBSERVE_IMAGE, written to
# GITHUB_ENV), so the stack start finds it locally and contacts no registry
# for it. Docker ignores the tag of a tag@digest ref, so the pulled binary's
# version must also be the tag's, or a tag bumped without its digest would run
# the old version unnoticed.
#
# Inputs: COMPOSE_YAML (default compose.yaml; the tests set it), GITHUB_ENV.
# docs/quality/ci-pipeline.md "OpenObserve image".
set -eu
cd "$(dirname "$0")/../.."

second=docker.io/openobserve/openobserve

log() { echo "pull-openobserve: $*" >&2; }

pin=$(sed -n 's/^ *image: *\${OPENOBSERVE_IMAGE:-\([^}]*\)}.*/\1/p' "${COMPOSE_YAML:-compose.yaml}")
if ! printf '%s\n' "$pin" | grep -Eqx '[^:@ ]+:[^:@ ]+@sha256:[0-9a-f]{64}'; then
    log "compose.yaml does not pin openobserve as \${OPENOBSERVE_IMAGE:-<repo>:<tag>@sha256:<digest>}"
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
    if ! out=$(docker pull -q "$1@$digest" 2>&1); then
        log "$1 failed: $out"
        return 1
    fi
    got=$(docker image inspect -f '{{join .RepoDigests "\n"}}' "$1@$digest" 2>&1) || true
    # Docker lists a Docker Hub repository without its host.
    if ! printf '%s\n' "$got" | grep -Fqx -e "$1@$digest" -e "${1#docker.io/}@$digest"; then
        log "$1 failed: its image is not $digest but: $got"
        return 1
    fi
}

if fetch "$first"; then
    ref=$first@$digest
elif fetch "$second"; then
    ref=$second@$digest
else
    log "both sources failed (errors above): $first and $second"
    exit 1
fi

# The bytes are pinned, so another source cannot fix a version mismatch.
got=$(docker run --rm --network none --entrypoint /openobserve "$ref" --version 2>&1) || true
if [ "$got" != "openobserve $version" ]; then
    log "$ref says '$got', but compose.yaml's tag says $version; change the tag and the digest together"
    exit 1
fi

[ -z "${GITHUB_ENV:-}" ] || echo "OPENOBSERVE_IMAGE=$ref" >>"$GITHUB_ENV"
log "using $ref ($version)"
