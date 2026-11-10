#!/bin/sh
# make ci-self-hosted-cleanup ID=<id> (TICK-641): the ci workflow's
# self-hosted-cleanup job. Removes what one run left on the self-hosted
# runner's Docker, whether the run passed, failed or was cancelled, and
# nothing else. The run's names come from scripts/ci/self-hosted-env.sh:
#   compose projects  nava-<id> and nava-<id>-int (the internal overlay):
#                     containers (with their anonymous volumes), named
#                     volumes and networks, by the com.docker.compose.project
#                     label, so a service of an overlay that compose down
#                     would not know is caught too
#   integration       containers and networks labelled nava.ci.owner=nava-<id>
#                     (test-integration.sh, isolation.sh, commit-check-budget.sh,
#                     identity-attacks.sh, the health integration test)
#   images            every nava/*:<id> tag, and compose's default names
#                     nava-<id>-*; untagged only: shared layers stay
#   logs              ${TMPDIR:-/tmp}/nava-<id>-int.log (make stack-internal-beside)
# Everything is selected by an exact label value or an exact tag, never by
# prune or by a name prefix that could match the owner's own containers
# (project nava, images nava/*:dev) or another run's. The buildx builder
# nava-ci and its cache are not touched (make ci-builder-prune trims the cache).
# Then it fails if anything of the run is left.
set -eu

[ $# -eq 1 ] || { echo "usage: $0 <id>" >&2; exit 2; }
id=$1
case $id in
    "" | dev | *[!a-z0-9_-]* | [!a-z0-9]*) echo "ci-self-hosted-cleanup: ID '$id': lowercase letters, digits, - and _, not dev" >&2; exit 2 ;;
esac
project="nava-$id"
projects="$project $project-int"

# The run's resources, one per line: "<kind> <id or name>".
list() {
    for p in $projects; do
        for c in $(docker ps -a -q --filter "label=com.docker.compose.project=$p"); do echo "container $c"; done
    done
    for c in $(docker ps -a -q --filter "label=nava.ci.owner=$project"); do echo "container $c"; done
    for p in $projects; do
        for v in $(docker volume ls -q --filter "label=com.docker.compose.project=$p"); do echo "volume $v"; done
        for n in $(docker network ls -q --filter "label=com.docker.compose.project=$p"); do echo "network $n"; done
    done
    for n in $(docker network ls -q --filter "label=nava.ci.owner=$project"); do echo "network $n"; done
    # reference= matches the whole name; the id has no glob characters.
    for i in $(docker images --format '{{.Repository}}:{{.Tag}}' --filter "reference=nava/*:$id"); do echo "image $i"; done
    for p in $projects; do
        for i in $(docker images --format '{{.Repository}}:{{.Tag}}' --filter "reference=$p-*"); do
            # nava-<id>-int-* belongs to the -int project, not to a service
            # of nava-<id> named int-*; either way it is this run's.
            echo "image $i"
        done
    done
}

# Containers first (a volume, network or image in use cannot go), then
# volumes and networks, then image tags.
before=$(list | sort -u)
if [ -z "$before" ]; then
    echo "ci-self-hosted-cleanup: nothing of $project on this host"
else
    for kind in container volume network image; do
        for r in $(printf '%s\n' "$before" | sed -n "s/^$kind //p"); do
            case $kind in
                container) docker rm -fv "$r" >/dev/null 2>&1 || true ;;
                volume) docker volume rm -f "$r" >/dev/null 2>&1 || true ;;
                network) docker network rm "$r" >/dev/null 2>&1 || true ;;
                image) docker image rm "$r" >/dev/null 2>&1 || true ;;
            esac
            echo "ci-self-hosted-cleanup: removed $kind $r"
        done
    done
fi
rm -f "${TMPDIR:-/tmp}/$project-int.log"

left=$(list | sort -u)
if [ -n "$left" ]; then
    echo "ci-self-hosted-cleanup: left on the host:" >&2
    printf '%s\n' "$left" | sed 's/^/  /' >&2
    exit 1
fi
echo "ci-self-hosted-cleanup: clean, nothing of $project or $project-int left"
