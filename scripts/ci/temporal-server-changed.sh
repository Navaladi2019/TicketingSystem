#!/bin/sh
# make ci-temporal-server-changed (TICK-641): whether the ci workflow's
# temporal-server job (the module's unit tests, golangci-lint and govulncheck)
# runs. The module has its own go.mod and nothing in the root module touches
# it, so a change outside its inputs (below) skips the job. Before, it ran on
# every Go change: about 95% of its runs tested an unchanged module.
#
# Prints true or false; with GITHUB_OUTPUT set, also writes
# temporal_server=true|false there. Reasons go to stderr.
#
#   CI_FULL=1       a pull request into a story branch: true (every job must
#                   run, TICK-638), even with CI_LINT_ONLY=1.
#   CI_LINT_ONLY=1  a pull request without ci:run: false.
#   PUSH_BEFORE     a push to main: the diff is PUSH_BEFORE..HEAD (the push's
#                   own commits). An unknown or all-zero commit runs the job.
#   CI_ALL=1        without PUSH_BEFORE (a manual run on main): true.
#   otherwise       the diff from the merge base with BASE (default
#                   origin/main), as make ci-stages.
#
# govulncheck is not in this job: the govulncheck workflow scans both modules
# on every pull request, push to main and day (TICK-641).
set -eu

say() {
    echo "ci-temporal-server-changed: $2" >&2
    echo "$1"
    [ -z "${GITHUB_OUTPUT:-}" ] || echo "temporal_server=$1" >>"$GITHUB_OUTPUT"
    exit 0
}

[ "${CI_FULL:-}" != 1 ] || say true "CI_FULL=1 (pull request into a story branch)"
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
    from=$(git merge-base "$base" HEAD) || { echo "ci-temporal-server-changed: no merge base with $base; fetch it first" >&2; exit 2; }
fi

# The module, how it is built, tested and linted, and the tool pins.
files=$(git diff --name-only --no-renames "$from" HEAD -- \
    temporal-server make/temporal-server.mk make/lint.mk Makefile \
    .golangci.yml docs/pins.md .github/workflows/ci.yml tools/ci/suiteran \
    scripts/ci/temporal-server-changed.sh)
[ -z "$files" ] || say true "changed: $(echo "$files" | paste -s -d ' ' -)"
say false "no temporal-server input changed since $from"
