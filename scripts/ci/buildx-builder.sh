#!/bin/sh
# make ci-builder (TICK-641): on the self-hosted runner, the persistent buildx
# builder whose BuildKit cache make image-ci (IMAGE_CACHE=local) reuses run
# after run. Creates it (docker-container driver) when missing, starts it, and
# selects it for the rest of the job through BUILDX_BUILDER in $GITHUB_ENV,
# not `docker buildx use`: that would change the default builder of the
# owner's own Docker work on the same host.
# Never removes the builder or its cache; make ci-builder-prune bounds its size.
# docker/setup-buildx-action is not used here: it makes a throwaway builder,
# and the cache would go with it at the end of every job.
#
# `buildx-builder.sh prune` (make ci-builder-prune): trims the builder's cache
# to at most KEEP_GB gigabytes (default 20), least recently used first. Never a
# full prune: the cache is the point of the builder. A missing builder is not
# an error (nothing to trim).
set -eu

name=${CI_BUILDER:-nava-ci}
if [ "${1:-}" = prune ]; then
    keep=${KEEP_GB:-20}
    case $keep in '' | *[!0-9]*) echo "ci-builder-prune: KEEP_GB=$keep: a whole number of GB" >&2; exit 2 ;; esac
    [ "$keep" -gt 0 ] || { echo "ci-builder-prune: KEEP_GB=0 would empty the cache" >&2; exit 2; }
    if ! docker buildx inspect "$name" >/dev/null 2>&1; then
        echo "ci-builder-prune: no builder $name, nothing to trim"
        exit 0
    fi
    # buildx 0.17 renamed --keep-storage to --max-used-space; older ones
    # know only the first, newer ones still accept it (deprecated).
    flag=--keep-storage
    docker buildx prune --help 2>/dev/null | grep -q -- --max-used-space && flag=--max-used-space
    docker buildx prune --builder "$name" "$flag" "${keep}gb" -f
    docker buildx du --builder "$name" 2>/dev/null | tail -n 1 || true
    exit 0
fi
if ! docker buildx inspect "$name" >/dev/null 2>&1; then
    echo "ci-builder: creating $name (docker-container driver)"
    # Two jobs can race to create it; the loser finds it made.
    docker buildx create --name "$name" --driver docker-container >/dev/null ||
        docker buildx inspect "$name" >/dev/null
fi
docker buildx inspect --bootstrap "$name"
docker buildx du --builder "$name" 2>/dev/null | tail -n 1 || true
out=${GITHUB_ENV:-/dev/stdout}
echo "BUILDX_BUILDER=$name" >>"$out"
