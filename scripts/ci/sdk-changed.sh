#!/bin/sh
# make ci-sdk-changed (TICK-827, RFC-017 CI-E10): whether the ci workflow's
# sdk job (regenerate the TypeScript client and fail on drift, lint, tests,
# npm pack) runs. Modelled on make ci-temporal-server-changed (TICK-695).
#
# Prints true or false; with GITHUB_OUTPUT set, also writes sdk=true|false
# there. Reasons go to stderr.
#
#   CI_FULL=1       a full run (ci:full, a manual run, the daily run): true,
#                   even with CI_LINT_ONLY=1.
#   CI_LINT_ONLY=1  a pull request without ci:run: false.
#   PUSH_BEFORE     a push to main: the diff is PUSH_BEFORE..HEAD. An unknown
#                   or all-zero commit runs the job.
#   CI_ALL=1        without PUSH_BEFORE (a manual run on main): true.
#   otherwise       the diff from the merge base with BASE (default
#                   origin/main). A pull request into a TICK-* story branch
#                   diffs against the story branch, so a task PR that touches
#                   neither the API nor the SDK skips the job; the full-suite
#                   job accepts that skip only when this script said false.
#
# Inputs (whole files): api/ (the bundle the client is generated from, and the
# sources it is bundled from), sdk/, make/sdk.mk, this script, and
# .github/workflows/ci.yml (unlike temporal-server: the job takes about a
# minute, so an edit to its own steps is exercised on the same pull request).
set -eu

say() {
    echo "ci-sdk-changed: $2" >&2
    echo "$1"
    [ -z "${GITHUB_OUTPUT:-}" ] || echo "sdk=$1" >>"$GITHUB_OUTPUT"
    exit 0
}

[ "${CI_FULL:-}" != 1 ] || say true "CI_FULL=1 (full run)"
[ "${CI_LINT_ONLY:-}" != 1 ] || say false "CI_LINT_ONLY=1 (no ci:run label)"

if [ -n "${PUSH_BEFORE:-}" ]; then
    case $PUSH_BEFORE in
        *[!0]*) ;;
        *) say true "push with no earlier commit ($PUSH_BEFORE)" ;;
    esac
    git cat-file -e "$PUSH_BEFORE^{commit}" 2>/dev/null || say true "push base $PUSH_BEFORE not in this checkout"
    from=$PUSH_BEFORE
elif [ "${CI_ALL:-}" = 1 ]; then
    say true "CI_ALL=1 without a push base (manual run on main)"
else
    base=${BASE:-origin/main}
    from=$(git merge-base "$base" HEAD) || { echo "ci-sdk-changed: no merge base with $base; fetch it first" >&2; exit 2; }
fi

files=$(git diff --name-only --no-renames "$from" HEAD -- \
    api sdk make/sdk.mk scripts/ci/sdk-changed.sh .github/workflows/ci.yml)
[ -z "$files" ] || say true "changed: $(echo "$files" | paste -s -d ' ' -)"
say false "no API or SDK input changed since $from"
