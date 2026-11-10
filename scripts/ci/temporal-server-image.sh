#!/bin/sh
# The published nava/temporal-server image (TICK-641).
#
#   temporal-server-image.sh tag      print the registry reference of this
#                                     checkout's temporal-server/ tree
#   temporal-server-image.sh publish  build and push it unless it exists
#                                     (the temporal-server-image workflow, on
#                                     main only)
#   temporal-server-image.sh pull     pull it and tag it TEMPORAL_SERVER_IMAGE;
#                                     with GITHUB_ENV set, also writes
#                                     TEMPORAL_SERVER_PREBUILT=1 there, so
#                                     make image-ci leaves temporal-server out
#                                     of its bake. A missing tag (the change
#                                     edits temporal-server/), a dirty tree or
#                                     a failed pull is not an error: the image
#                                     is then built from source as before.
#
# The tag is git's hash of the temporal-server/ tree, the whole build context
# (Dockerfile, sources, go.mod and pinned base images among them): the same
# tree is the same image, whichever commit holds it.
#
# Inputs: TEMPORAL_SERVER_REGISTRY (default ghcr.io/navaladi2019/nava-temporal-server),
# TEMPORAL_SERVER_IMAGE (pull), GH_TOKEN and GH_ACTOR (publish).
set -eu
cd "$(dirname "$0")/../.."

registry=${TEMPORAL_SERVER_REGISTRY:-ghcr.io/navaladi2019/nava-temporal-server}
tree=$(git rev-parse HEAD:temporal-server)
ref=$registry:$tree
log() { echo "temporal-server-image: $*" >&2; }

case ${1:-} in
    tag)
        echo "$ref"
        ;;
    publish)
        : "${GH_TOKEN:?publish needs GH_TOKEN (packages: write)}" "${GH_ACTOR:?publish needs GH_ACTOR}"
        [ -z "$(git status --porcelain -- temporal-server)" ] || { log "temporal-server/ has uncommitted changes; publish only a committed tree"; exit 2; }
        printf '%s' "$GH_TOKEN" | docker login "${registry%%/*}" -u "$GH_ACTOR" --password-stdin
        if docker manifest inspect "$ref" >/dev/null 2>&1; then
            log "$ref already published"
            exit 0
        fi
        docker build \
            --label org.opencontainers.image.source="https://github.com/${GITHUB_REPOSITORY:-Navaladi2019/TicketingSystem}" \
            --label org.opencontainers.image.revision="$(git rev-parse HEAD)" \
            --label nava.temporal-server.tree="$tree" \
            -t "$ref" temporal-server
        docker push "$ref"
        log "published $ref"
        ;;
    pull)
        : "${TEMPORAL_SERVER_IMAGE:?pull needs TEMPORAL_SERVER_IMAGE}"
        if [ -n "$(git status --porcelain -- temporal-server)" ]; then
            log "temporal-server/ has uncommitted changes; building from source"
            exit 0
        fi
        if ! docker pull -q "$ref" >/dev/null 2>&1; then
            log "$ref is not published (the change edits temporal-server/, or main has not published it yet); building from source"
            exit 0
        fi
        got=$(docker image inspect -f '{{ index .Config.Labels "nava.temporal-server.tree" }}' "$ref")
        if [ "$got" != "$tree" ]; then
            log "$ref carries tree label '$got', want $tree; building from source"
            exit 0
        fi
        docker tag "$ref" "$TEMPORAL_SERVER_IMAGE"
        [ -z "${GITHUB_ENV:-}" ] || echo "TEMPORAL_SERVER_PREBUILT=1" >>"$GITHUB_ENV"
        log "using published $ref as $TEMPORAL_SERVER_IMAGE"
        ;;
    *)
        echo "usage: $0 tag|publish|pull" >&2
        exit 2
        ;;
esac
