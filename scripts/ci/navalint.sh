#!/bin/sh
# Builds tools/navalint, the platform's go/analysis analyzers (setconfig, T07),
# and runs it as go vet's tool. navalint is its own module so
# golang.org/x/tools stays out of the production go.mod (TICK-94 item 15).
# Arguments go to go vet; the default is ./...
set -eu
cd "$(dirname "$0")/../.."
# Relative, so Go on Windows gets a path it understands; go vet makes it
# absolute. GOEXE is .exe on Windows, where exec needs the suffix.
tool="bin/navalint$(go env GOEXE)"
go -C tools/navalint build -o "../../$tool" .
[ $# -gt 0 ] || set -- ./...
exec go vet -vettool="$tool" "$@"
