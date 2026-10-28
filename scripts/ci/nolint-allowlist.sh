#!/bin/sh
# ADR-002-E1: the only forbidigo waiver outside tests is the GetMe GroupPaths
# call, and forbidigo is excluded only for tests and the telemetry package.
#  1. Directives: matches every spelling golangci-lint honours (any case,
#     spaces, comma lists) and pins the waiver to the call line, so moving it
#     above `package` or a func declaration (file or function scope) fails.
#  2. Config: a forbidigo exclusion rule for any other path, a forbidigo rule
#     with no path, or a global linters.exclusions.paths entry (which turns
#     off every linter, forbidigo included) fails, so the api.go file
#     exclusion cannot come back.
#  3. Generated files: golangci-lint skips every linter, nolintlint included,
#     on a file with a "Code generated ... DO NOT EDIT." header, so the set of
#     such files is pinned too.
# Adding a waiver, an exclusion or a generated file means editing this file:
# a reviewed change.
set -eu
cd "$(dirname "$0")/../.."
code=0

want='internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller'
got=$(git grep --untracked -i -E 'nolint[^/]*forbidigo' -- '*.go' ':!*_test.go' ':!docs/' | tr -d '\r' || true)
if [ "$got" != "$want" ]; then
	printf 'nolint allow-list (ADR-002-E1): forbidigo waivers changed\nwant:\n%s\ngot:\n%s\n' "$want" "$got" >&2
	code=1
fi

bad=$(tr -d '\r' < .golangci.yml | awk '
	function flush() {
		if (item ~ /forbidigo/) {
			p = item; sub(/.*path:[ ]*/, "", p); sub(/[ \n].*/, "", p)
			if (item !~ /path:/ || (p != "_test\\.go" && p != "^internal/platform/telemetry/")) print "forbidigo exclusion rule: " item
		}
		item = ""
	}
	/^[^ #]/ { flush(); lin = ($0 ~ /^linters:/); excl = 0; sect = ""; next }
	lin && /^  [^ #]/ { flush(); excl = ($0 ~ /^  exclusions:/); sect = ""; next }
	lin && excl && /^    [^ #]/ { flush(); sect = $1; next }
	lin && excl && sect == "rules:" && /^      - / { flush(); item = $0; next }
	lin && excl && sect == "rules:" && item != "" && /^        / { item = item "\n" $0; next }
	lin && excl && sect == "paths:" && /^      - / { print "global exclusion path: " $0 }
	END { flush() }
')
if [ -n "$bad" ]; then
	printf 'nolint allow-list (ADR-002-E1): forbidigo may be excluded only for _test\\.go and ^internal/platform/telemetry/\n%s\n' "$bad" >&2
	code=1
fi

gen=$(git grep --untracked -l -E '^// Code generated .* DO NOT EDIT\.$' -- '*.go' ':!docs/' | tr -d '\r' | sort || true)
if [ "$gen" != "$(printf 'api/api.gen.go\ninternal/apitest/api.gen.go')" ]; then
	printf 'nolint allow-list (ADR-002-E1): generated-file headers changed (golangci-lint skips these files)\n%s\n' "$gen" >&2
	code=1
fi
exit $code
