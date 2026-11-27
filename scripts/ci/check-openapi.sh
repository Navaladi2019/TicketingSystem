#!/bin/sh
# Additive-only rule for /api/v1: fails when the bundled contract removes or
# narrows anything the base ref offers. Before the MS-6 split the base has the
# whole contract in api/openapi.yaml; after it, in api/openapi.bundled.yaml.
# Usage: check-openapi.sh <base-ref>
# On a pull_request run (GITHUB_EVENT_NAME) HEAD is the merge commit
# actions/checkout made of the PR into the base as it was then; its first
# parent is that base. <base-ref> (origin/main, fetched later in the run) may
# already hold another PR's additions, which would read as removals here
# (TICK-1179), so the first parent wins; a PR tip HEAD uses its merge-base
# with <base-ref>. push and schedule keep <base-ref>.
# OPENAPI_COMPAT overrides the compare command (test-check-openapi-base.sh).
set -eu
cd "$(dirname "$0")/../.."
base=${1:-origin/main}
if [ "${GITHUB_EVENT_NAME:-}" = pull_request ]; then
	if git rev-parse -q --verify 'HEAD^2' >/dev/null; then
		echo "check-openapi: pull_request, comparing against the merge commit's first parent ($(git rev-parse --short 'HEAD^1')), not $base"
		base='HEAD^1'
	else
		# HEAD is the PR tip itself (a checkout with ref: head.sha): compare
		# against the commit the PR is built on, never the moved <base-ref>.
		# No merge-base (a shallow clone) fails rather than guessing.
		mb=$(git merge-base HEAD "$base") || { echo "check-openapi: pull_request, HEAD is not a merge commit and has no merge-base with $base; fetch full history" >&2; exit 1; }
		echo "check-openapi: pull_request, HEAD is not a merge commit; comparing against its merge-base with $base ($(git rev-parse --short "$mb")), not $base"
		base=$mb
	fi
fi
compat=${OPENAPI_COMPAT:-go run ./cmd/openapi-compat}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
for f in api/openapi.bundled.yaml api/openapi.yaml; do
	if git cat-file -e "$base:$f" 2>/dev/null; then
		git show "$base:$f" > "$tmp/openapi.base.yaml"
		$compat "$tmp/openapi.base.yaml" api/openapi.bundled.yaml api/approved-breaking-changes.txt
		exit $?
	fi
done
echo "no api spec on $base: nothing to compare"
