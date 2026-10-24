#!/bin/sh
# govulncheck for the CI lint stage (TICK-153 D2): both modules (the root and
# temporal-server/, separate go.mod files), at the same time, each at most
# once per day per dependency set. A module is scanned when no scan of its
# current go.mod and go.sum has passed today on this host, so a change to its
# dependencies is always scanned, and an unchanged module is scanned by the
# first CI run of each day for new advisories (ADR-001-A1's R7 trigger).
# A passing scan leaves a stamp in the Go build cache directory, shared by
# every checkout on the host.
#
# VULNCHECK=all scans both modules regardless of stamps. `make vulncheck` and
# `make vulncheck-temporal-server` are the plain, always-scanning runs.
set -eu
cd "$(dirname "$0")/../.."

stamps=${NAVA_VULNCHECK_STAMPS:-$(go env GOCACHE)/nava-vulncheck}
mkdir -p "$stamps"
today=$(date -u +%Y-%m-%d)
# The Go version counts too: govulncheck reports standard library advisories
# for the toolchain it runs with.
version=$(govulncheck -version 2>/dev/null | grep -e '^Go:' -e '^Scanner:' | sed 's/^[^:]*: *//' |
    paste -s -d _ - | tr -d '\r\n' | tr -c 'a-zA-Z0-9.@_-' '_')
logs=$(mktemp -d)
trap 'rm -rf "$logs"' EXIT
trap 'exit 130' INT TERM

procs=""
for mod in . temporal-server; do
    key=$(cat "$mod/go.mod" "$mod/go.sum" | cksum | cut -d ' ' -f 1)
    label=$( [ "$mod" = . ] && echo root || echo "$mod")
    stamp="$stamps/$label-$key-$version-$today"
    if [ "${VULNCHECK:-}" != all ] && [ -f "$stamp" ]; then
        echo "vulncheck: $label skipped, its dependencies already passed today ($stamp)"
        continue
    fi
    ( cd "$mod" && govulncheck ./... ) >"$logs/$label" 2>&1 && touch "$stamp" &
    procs="$procs $!:$label"
done

status=0
for p in $procs; do
    label=${p#*:}
    if wait "${p%%:*}"; then result=passed; else result=FAILED; status=1; fi
    echo "=== vulncheck $label: $result"
    cat "$logs/$label"
done
exit "$status"
