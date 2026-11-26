#!/bin/sh
# vulncheck-detect (ADR-001-E4 contract rules 1 and 2, TICK-1148): which
# modules the govulncheck workflow scans.
#
# Inputs (env):
#   EVENT     github.event_name. Anything but pull_request (schedule,
#             workflow_dispatch) scans both modules and reads no git history.
#   BASE_SHA  the pull request's base commit.
#   HEAD_SHA  the pull request's head commit.
#
# Appends exactly two lines, root=true|false and temporal_server=true|false,
# to the file named by GITHUB_OUTPUT (stdout when unset), and only once both
# are known: a failure writes nothing. One reason per line goes to stderr.
#
# On a pull request the diff runs from git merge-base BASE_SHA HEAD_SHA to
# HEAD_SHA with --no-renames, so a rename lists both paths and renaming go.sum
# or go.work away still matches. An empty or non-hex BASE_SHA or HEAD_SHA (a
# branch name, an option), no merge base or any git error exits non-zero, and
# the gate then fails.
#
# Paths are read with core.quotePath=false, so a non-ASCII path such as
# tools/ci/cisteps/<e-acute>_test.go prints as is and matches its row. Git
# still C-quotes a path holding a double quote, a backslash or a control
# character; such a path cannot be matched, so it fails closed: both (Security
# review of PR #912, finding 1).
#
#   go.mod, go.sum                                     -> root
#   temporal-server/go.mod, temporal-server/go.sum     -> temporal_server
#   .github/workflows/govulncheck.yml, its copy
#   tools/ci/cisteps/govulncheck.copy.yml, this script,
#   scripts/ci/vulncheck-gate.sh, anything under tools/ci/cisteps/ or
#   tools/ci/vulngate/ (this gate's tests, PLAN-001-E4 D-E4-3), an added or
#   removed line naming govulncheck or GOVULNCHECK_VERSION in Makefile or
#   make/lint.mk                                       -> both
#   an added or removed FROM line naming golang (any case of FROM, flags such
#   as --platform= allowed; wider than the contract's FROM golang:) in a file
#   whose base name starts with Dockerfile (Dockerfile.dev counts); a path whose base name is go.work or
#   go.work.sum                                        -> both
#   anything else                                      -> neither
set -eu

me=vulncheck-detect
root=false
ts=false

say() { echo "$me: $*" >&2; }
die() { say "$*"; exit 2; }

emit() {
    if [ -n "${GITHUB_OUTPUT:-}" ]; then
        printf 'root=%s\ntemporal_server=%s\n' "$root" "$ts" >>"$GITHUB_OUTPUT"
    else
        printf 'root=%s\ntemporal_server=%s\n' "$root" "$ts"
    fi
}

if [ "${EVENT:-}" != pull_request ]; then
    say "event '${EVENT:-}' is not pull_request -> both"
    root=true
    ts=true
    emit
    exit 0
fi

[ -n "${BASE_SHA:-}" ] || die "BASE_SHA is empty on a pull_request"
[ -n "${HEAD_SHA:-}" ] || die "HEAD_SHA is empty on a pull_request"
# A commit id only: never a ref name, and never an option to git.
case $BASE_SHA in *[!0-9a-fA-F]*) die "BASE_SHA '$BASE_SHA' is not a hex commit id" ;; esac
case $HEAD_SHA in *[!0-9a-fA-F]*) die "HEAD_SHA '$HEAD_SHA' is not a hex commit id" ;; esac

from=$(git merge-base "$BASE_SHA" "$HEAD_SHA") ||
    die "no merge base of $BASE_SHA and $HEAD_SHA (fetch with fetch-depth: 0)"

files=$(git -c core.quotePath=false diff --name-only --no-renames "$from" "$HEAD_SHA") ||
    die "git diff $from $HEAD_SHA failed"
# Changed lines, not whole files: an added or removed line that matches.
pins=$(git -c core.quotePath=false diff --name-only --no-renames -G 'govulncheck|GOVULNCHECK_VERSION' "$from" "$HEAD_SHA" -- Makefile make/lint.mk) ||
    die "git diff -G govulncheck failed"
golang=$(git -c core.quotePath=false diff --name-only --no-renames -G '^[[:space:]]*[Ff][Rr][Oo][Mm][[:space:]].*golang' "$from" "$HEAD_SHA") ||
    die "git diff -G 'FROM ... golang' failed"

while IFS= read -r f; do
    case $f in
        '') ;;
        '"'*)
            say "$f: quoted path, cannot match a row (fails closed) -> both"
            root=true
            ts=true
            ;;
        go.mod | go.sum)
            say "$f changed -> root"
            root=true
            ;;
        temporal-server/go.mod | temporal-server/go.sum)
            say "$f changed -> temporal_server"
            ts=true
            ;;
        .github/workflows/govulncheck.yml | tools/ci/cisteps/govulncheck.copy.yml | \
            scripts/ci/vulncheck-detect.sh | scripts/ci/vulncheck-gate.sh | \
            tools/ci/cisteps/* | tools/ci/vulngate/*)
            say "$f changed (the gate) -> both"
            root=true
            ts=true
            ;;
    esac
    case ${f##*/} in
        go.work | go.work.sum)
            say "$f changed (a go.work) -> both"
            root=true
            ts=true
            ;;
    esac
done <<END
$files
END

while IFS= read -r f; do
    [ -z "$f" ] || {
        say "$f: a line naming govulncheck or GOVULNCHECK_VERSION changed -> both"
        root=true
        ts=true
    }
done <<END
$pins
END

while IFS= read -r f; do
    case ${f##*/} in
        Dockerfile*)
            say "$f: a FROM ... golang line changed -> both"
            root=true
            ts=true
            ;;
    esac
done <<END
$golang
END

[ "$root" = true ] || [ "$ts" = true ] || say "no vulncheck input changed since $from -> none"
emit
