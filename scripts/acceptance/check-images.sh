#!/bin/sh
# Acceptance kits (TICK-1021): fails unless every container of this compose
# project that runs a nava/* image runs it under NAVA_IMAGE_TAG (default:
# scripts/acceptance/image-tag.sh, this checkout's tag). Exited one-shot
# containers count too: db-migrate is the one that ran the migrations.
# A kit's run.sh calls it first, so a stack started from another checkout's
# images, or from the shared `dev` tag, stops the walk before any request.
# Run from the repository root.
set -eu
want=${NAVA_IMAGE_TAG:-$(sh scripts/acceptance/image-tag.sh)}
ps=$(docker compose ps -a --format '{{.Service}} {{.Image}}')
n=$(printf '%s\n' "$ps" | awk '$2 ~ /^nava\// { n++ } END { print n + 0 }')
bad=$(printf '%s\n' "$ps" | awk -v t=":$want" '$2 ~ /^nava\// && substr($2, length($2) - length(t) + 1) != t')
if [ "$n" -eq 0 ] || [ -n "$bad" ]; then
    echo "check-images: want every nava/* container on tag $want; found $n nava/* container(s)" >&2
    [ -z "$bad" ] || printf '%s\n' "$bad" | sed 's/^/  wrong tag: /' >&2
    echo "check-images: rebuild with: export NAVA_IMAGE_TAG=$want && sh scripts/acceptance/fresh-stack.sh" >&2
    exit 1
fi
echo "check-images: $n nava/* container(s) on tag $want"
