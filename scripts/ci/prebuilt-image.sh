#!/bin/sh
# Published CI images (TICK-641 temporal-server, TICK-693 keycloak).
#
#   prebuilt-image.sh <service> tag      print the registry reference of this
#                                        checkout's key for <service>
#   prebuilt-image.sh <service> inputs   print the files or directories the key
#                                        hashes (TestPublishedImageKeyCovers-
#                                        Build checks them against the
#                                        Dockerfile's COPY/ADD sources)
#   prebuilt-image.sh <service> publish  build and push it unless it exists
#                                        (the prebuilt-images workflow, on main
#                                        only)
#   prebuilt-image.sh <service> pull     pull it, check its key label and tag it
#                                        nava/<service>:$NAVA_IMAGE_TAG. Exit 0:
#                                        the published image is in place. Exit
#                                        1: it is not (the change edits a key
#                                        input, main has not published it yet,
#                                        a dirty tree, a failed pull or a label
#                                        mismatch); make image-ci then builds it
#                                        from source as before. make
#                                        prebuilt-pull turns the hits into
#                                        PREBUILT_SERVICES; a miss never fails it.
#
# The tag is a content key of exactly what the image build reads, so the same
# inputs are the same image whichever commit holds them:
#   temporal-server  git's hash of the temporal-server/ tree, the whole build
#                    context (unchanged from TICK-641, so existing tags hit).
#   keycloak         sha256 over "<path> <blob id>" of the Dockerfile and
#                    quarkus.properties. The build context is the repo root,
#                    so its tree hash would change on every commit; the
#                    Dockerfile pins the base image, so a base bump changes the
#                    key. import/ and test-realms/ are mounted at run time,
#                    not copied, and are not in the key.
#
# Inputs: <SERVICE>_REGISTRY (e.g. KEYCLOAK_REGISTRY) overrides the registry,
# NAVA_IMAGE_TAG (pull, default dev), GH_TOKEN and GH_ACTOR (publish).
set -eu
cd "$(dirname "$0")/../.."

log() { echo "prebuilt-image: $*" >&2; }
usage() { echo "usage: $0 temporal-server|keycloak tag|inputs|publish|pull" >&2; exit 2; }
sha256() { if command -v sha256sum >/dev/null 2>&1; then sha256sum; else shasum -a 256; fi | cut -c1-64; }

svc=${1:-}
cmd=${2:-}
# Per service: inputs (the key's paths, one quoted line: the guard test reads
# it), registry, label (the key's image label), context and dockerfile (the
# publish build, as compose.yaml builds it).
case $svc in
temporal-server)
    inputs='temporal-server'
    registry=${TEMPORAL_SERVER_REGISTRY:-ghcr.io/navaladi2019/nava-temporal-server}
    label=nava.temporal-server.tree
    context=temporal-server
    dockerfile=temporal-server/Dockerfile
    key() { git rev-parse HEAD:temporal-server; }
    ;;
keycloak)
    inputs='deploy/compose/keycloak/Dockerfile deploy/compose/keycloak/quarkus.properties'
    registry=${KEYCLOAK_REGISTRY:-ghcr.io/navaladi2019/nava-keycloak}
    label=nava.keycloak.key
    context=.
    dockerfile=deploy/compose/keycloak/Dockerfile
    key() {
        lines=
        for f in $inputs; do
            blob=$(git rev-parse "HEAD:$f")
            lines="$lines$f $blob
"
        done
        printf '%s' "$lines" | sha256
    }
    ;;
*)
    usage
    ;;
esac

case $cmd in
inputs)
    echo "$inputs"
    exit 0
    ;;
tag | publish | pull) ;;
*) usage ;;
esac

k=$(key)
ref=$registry:$k
# shellcheck disable=SC2086 # inputs is a list of paths without spaces
dirty=$(git status --porcelain -- $inputs)

case $cmd in
tag)
    echo "$ref"
    ;;
publish)
    : "${GH_TOKEN:?publish needs GH_TOKEN (packages: write)}" "${GH_ACTOR:?publish needs GH_ACTOR}"
    [ -z "$dirty" ] || { log "$svc: $inputs has uncommitted changes; publish only committed inputs"; exit 2; }
    printf '%s' "$GH_TOKEN" | docker login "${registry%%/*}" -u "$GH_ACTOR" --password-stdin
    if docker manifest inspect "$ref" >/dev/null 2>&1; then
        log "$ref already published"
        exit 0
    fi
    docker build \
        --label org.opencontainers.image.source="https://github.com/${GITHUB_REPOSITORY:-Navaladi2019/TicketingSystem}" \
        --label org.opencontainers.image.revision="$(git rev-parse HEAD)" \
        --label "$label=$k" \
        -f "$dockerfile" -t "$ref" "$context"
    docker push "$ref"
    log "published $ref"
    ;;
pull)
    local_ref=nava/$svc:${NAVA_IMAGE_TAG:-dev}
    if [ -n "$dirty" ]; then
        log "$svc: $inputs has uncommitted changes; building from source"
        exit 1
    fi
    if ! docker pull -q "$ref" >/dev/null 2>&1; then
        log "$ref is not published (the change edits $inputs, or main has not published it yet); building $svc from source"
        exit 1
    fi
    got=$(docker image inspect -f "{{ index .Config.Labels \"$label\" }}" "$ref" 2>/dev/null || true)
    if [ "$got" != "$k" ]; then
        log "$ref carries $label '$got', want $k; building $svc from source"
        exit 1
    fi
    if ! docker tag "$ref" "$local_ref"; then
        log "could not tag $ref as $local_ref; building $svc from source"
        exit 1
    fi
    log "using published $ref as $local_ref"
    ;;
esac
