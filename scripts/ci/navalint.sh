#!/bin/sh
# Builds tools/navalint, the platform's go/analysis analyzers (setconfig, T07),
# and runs it as go vet's tool. navalint is its own module so
# golang.org/x/tools stays out of the production go.mod (TICK-94 item 15).
#
# Two passes, so build-tagged Go files are checked too: one untagged, one with
# the test tags below. lintfixture is never in them: those fixtures are meant
# to fail (scripts/ci/lint-fixture.sh sets NAVALINT_TAGS=lintfixture).
# Arguments are packages for go vet; the default is ./...
set -eu
cd "$(dirname "$0")/../.."
test_tags="integration,temporalstack"
extra="${NAVALINT_TAGS:-}"
# Relative, so Go on Windows gets a path it understands; go vet makes it
# absolute. GOEXE is .exe on Windows, where exec needs the suffix.
tool="bin/navalint$(go env GOEXE)"
go -C tools/navalint build -o "../../$tool" .
[ $# -gt 0 ] || set -- ./...
code=0
go vet -vettool="$tool" ${extra:+-tags "$extra"} "$@" || code=1
go vet -vettool="$tool" -tags "${extra:+$extra,}$test_tags" "$@" || code=1
exit $code
