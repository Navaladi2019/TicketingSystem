#!/bin/sh
# Acceptance kits (TICK-1021): a fresh stack from this checkout's images.
# Builds every image under NAVA_IMAGE_TAG (default: this checkout's tag,
# scripts/acceptance/image-tag.sh), removes the project's containers and
# volumes, starts it with --no-build and waits until every service is
# healthy, then runs check-images.sh. Stops at the first failing step:
# `docker compose build` is never piped, so a failed or killed build fails
# here instead of leaving the previous images in place.
#
# Run from the repository root, with the tag exported so later docker compose
# commands in the same shell use it too:
#   export NAVA_IMAGE_TAG=$(sh scripts/acceptance/image-tag.sh)
#   sh scripts/acceptance/fresh-stack.sh
# A second stack on the same host also needs its own COMPOSE_PROJECT_NAME
# and host ports (compose.yaml, header).
set -eu
NAVA_IMAGE_TAG=${NAVA_IMAGE_TAG:-$(sh scripts/acceptance/image-tag.sh)}
export NAVA_IMAGE_TAG
case $NAVA_IMAGE_TAG in
    dev) echo "fresh-stack: NAVA_IMAGE_TAG=dev is shared by every checkout; unset it or use image-tag.sh" >&2; exit 2 ;;
esac
echo "fresh-stack: project ${COMPOSE_PROJECT_NAME:-nava}, images nava/*:$NAVA_IMAGE_TAG"
if ! docker compose build; then
    echo "fresh-stack: docker compose build failed; the stack was not touched" >&2
    exit 1
fi
docker compose down -v
docker compose up -d --wait --no-build
sh scripts/acceptance/check-images.sh
