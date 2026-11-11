#!/bin/sh
# make ci-temporal-server-changed (TICK-641, TICK-695): whether the ci
# workflow's temporal-server job (the module's unit tests and golangci-lint)
# runs. The module has its own go.mod and nothing in the root module touches
# it, so a change outside its inputs (below) skips the job. Before TICK-641 it
# ran on every Go change: about 95% of its runs tested an unchanged module.
#
# Prints true or false; with GITHUB_OUTPUT set, also writes
# temporal_server=true|false there. Reasons go to stderr.
#
#   CI_FULL=1       a full run (ci:full, a manual run, the daily run): true,
#                   even with CI_LINT_ONLY=1.
#   CI_LINT_ONLY=1  a pull request without ci:run: false.
#   PUSH_BEFORE     a push to main: the diff is PUSH_BEFORE..HEAD (the push's
#                   own commits). An unknown or all-zero commit runs the job.
#   CI_ALL=1        without PUSH_BEFORE (a manual run on main): true.
#   otherwise       the diff from the merge base with BASE (default
#                   origin/main), as make ci-stages. A pull request into a
#                   TICK-* story branch lands here too (TICK-695): BASE is the
#                   story branch, so a task PR that leaves the module alone
#                   skips the job. This is the one exception to TICK-638's
#                   "every job runs on a story PR"; the full-suite job accepts
#                   a skipped temporal-server only when this script said
#                   false. The story -> main PR still runs it whenever the
#                   story touched the module.
#
# Inputs (TICK-695):
#   whole file      temporal-server/, make/temporal-server.mk, .golangci.yml,
#                   tools/ci/suiteran, this script.
#   pin lines only  Makefile, make/lint.mk: a changed line that mentions
#                   GOLANGCI_LINT_VERSION (the pin, how make tools installs it
#                   and how check-golangci-lint checks it). docs/pins.md: the
#                   golangci-lint row or the Go (`temporal-server`) row. The
#                   job's Go version is temporal-server/go.mod, a whole-file
#                   input.
#   not inputs      .github/workflows/ci.yml and any other Makefile edit.
#                   Accepted trade-off: an edit to the job's own steps in
#                   ci.yml is not exercised until the module next changes,
#                   a full run (ci:full) or the next manual or daily run on
#                   main (CI_ALL).
#
# govulncheck is not in this job: the govulncheck workflow scans both modules
# every second day and on pull requests that change a go.mod or go.sum
# (TICK-641, TICK-694).
set -eu

say() {
    echo "ci-temporal-server-changed: $2" >&2
    echo "$1"
    [ -z "${GITHUB_OUTPUT:-}" ] || echo "temporal_server=$1" >>"$GITHUB_OUTPUT"
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
    from=$(git merge-base "$base" HEAD) || { echo "ci-temporal-server-changed: no merge base with $base; fetch it first" >&2; exit 2; }
fi

# The module, how it is built, tested and linted.
files=$(git diff --name-only --no-renames "$from" HEAD -- \
    temporal-server make/temporal-server.mk .golangci.yml tools/ci/suiteran \
    scripts/ci/temporal-server-changed.sh)
[ -z "$files" ] || say true "changed: $(echo "$files" | paste -s -d ' ' -)"

# The golangci-lint pin and how it is installed and checked.
files=$(git diff --name-only --no-renames -G 'GOLANGCI_LINT_VERSION' "$from" HEAD -- \
    Makefile make/lint.mk)
[ -z "$files" ] || say true "golangci-lint pin changed: $(echo "$files" | paste -s -d ' ' -)"

# The pins table rows of golangci-lint and the module's Go version.
files=$(git diff --name-only --no-renames -G '^\| (`golangci-lint`|Go \(`temporal-server`\)) \|' "$from" HEAD -- \
    docs/pins.md)
[ -z "$files" ] || say true "pinned version changed: $files"
say false "no temporal-server input changed since $from"
