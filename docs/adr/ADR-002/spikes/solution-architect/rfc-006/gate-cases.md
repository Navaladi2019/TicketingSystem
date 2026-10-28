# Spike: the ADR-002-E1 nolint allow-list gate (RFC-006, AC3)

Tree: `origin/TICK-47/group-changes-within-10-minutes` at `6ad12b5` (PR #221 head), 2026-10-28, Windows Git Bash (gawk). The Technical Architect confirmed in round 2 that mawk (Debian bookworm) gives the same `awk` output. Each case resets `.golangci.yml` and `internal/server/api.go`, copies the gate into `scripts/ci/`, and applies ADR-002-E1 steps 1 and 2 (all except R0). Only the gate runs here. The golangci-lint behaviour of each case is in `forbidigo-nolint.md` and in the round-1 and round-2 reviewer comments that ADR-002-E1 cites.

Revision 2 of this spike adds the F3 generated-file check (CTO, TICK-272 comment 01a180a4) and cases F3/F3b.

## Result: 17/17 as intended

```
ok    want FAIL got FAIL R0  S5 head as-is (exclusion, no directive): red
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        got:
        
        nolint allow-list (ADR-002-E1): forbidigo may be excluded only for _test\.go and ^internal/platform/telemetry/
ok    want pass got pass P   steps 1+2 applied: green
ok    want FAIL got FAIL 8   directive copied onto 2nd caller
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        got:
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        internal/server/api.go:	_, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
ok    want FAIL got FAIL A   directive moved above package + uncovered 2nd caller
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        got:
        internal/server/api.go://nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
ok    want FAIL got FAIL G   directive moved above func GetMe
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        got:
        internal/server/api.go://nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
ok    want FAIL got FAIL D   //nolint:FORBIDIGO on 2nd caller
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        got:
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        internal/server/api.go:	_, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:FORBIDIGO // r
ok    want FAIL got FAIL E   //nolint: forbidigo on 2nd caller
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        got:
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        internal/server/api.go:	_, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint: forbidigo // r
ok    want FAIL got FAIL B   //nolint:gosec,forbidigo on 2nd caller
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        got:
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        internal/server/api.go:	_, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:gosec,forbidigo // r
ok    want FAIL got FAIL U   directive in a new untracked file
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        got:
        internal/server/api.go:	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
        internal/server/extra.go:func x() {} //nolint:forbidigo // r
ok    want pass got pass T   directive in a _test.go (out of scope; nolintlint reports it unused)
ok    want FAIL got FAIL N1a api.go file exclusion put back
        forbidigo exclusion rule:       - path: ^internal/server/api\.go$
                linters: [forbidigo]
ok    want FAIL got FAIL N1b forbidigo exclusion, block-list form
        forbidigo exclusion rule:       - path: ^internal/server/
                linters:
                  - gosec
                  - forbidigo
ok    want FAIL got FAIL N1c forbidigo rule with no path
        forbidigo exclusion rule:       - text: GroupPaths
                linters: [forbidigo]
ok    want FAIL got FAIL N1d global linters.exclusions.paths entry
        global exclusion path:       - ^internal/server/
ok    want pass got pass N1e unrelated gosec-only exclusion
ok    want FAIL got FAIL F3  generated-file header on api.go + uncovered 2nd caller
        api/api.gen.go
        internal/apitest/api.gen.go
        internal/server/api.go
ok    want FAIL got FAIL F3b generated-file header on a new untracked file
        api/api.gen.go
        internal/apitest/api.gen.go
        internal/server/extra.go
```

## The gate: `scripts/ci/nolint-allowlist.sh`

```sh
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
```

## Harness

```sh
E="python -I ../spike-scripts/ed.py"
NL=' //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller'
reset(){ git checkout -q -- .golangci.yml internal/server/api.go; rm -f scripts/ci/nolint-allowlist.sh internal/server/extra.go internal/server/extra_test.go; }
base(){ reset; cp ../spike-scripts/nolint-allowlist.sh scripts/ci/; $E noexcl; $E callsite "$NL"; }
gate(){ want=$1; shift; if sh scripts/ci/nolint-allowlist.sh >out.txt 2>&1; then r=pass; else r=FAIL; fi; [ "$r" = "$want" ] && ok=ok || ok=MISMATCH; printf '%-5s want %-4s got %-4s %s\n' "$ok" "$want" "$r" "$*"; sed -n '2,6p' out.txt | grep -v '^want:' | sed 's/^/        /'; rm -f out.txt; }
reset; cp ../spike-scripts/nolint-allowlist.sh scripts/ci/; gate FAIL "R0  S5 head as-is (exclusion, no directive): red"
base; gate pass "P   steps 1+2 applied: green"
base; $E second "$NL"; gate FAIL "8   directive copied onto 2nd caller"
base; $E pkgline; $E second ""; gate FAIL "A   directive moved above package + uncovered 2nd caller"
base; $E funcabove; gate FAIL "G   directive moved above func GetMe"
base; $E second ' //nolint:FORBIDIGO // r'; gate FAIL "D   //nolint:FORBIDIGO on 2nd caller"
base; $E second ' //nolint: forbidigo // r'; gate FAIL "E   //nolint: forbidigo on 2nd caller"
base; $E second ' //nolint:gosec,forbidigo // r'; gate FAIL "B   //nolint:gosec,forbidigo on 2nd caller"
base; printf 'package server\n\nfunc x() {} //nolint:forbidigo // r\n' > internal/server/extra.go; gate FAIL "U   directive in a new untracked file"
base; printf 'package server\n\nfunc y() {} //nolint:forbidigo // r\n' > internal/server/extra_test.go; gate pass "T   directive in a _test.go (out of scope; nolintlint reports it unused)"
base; $E yaml apigo; gate FAIL "N1a api.go file exclusion put back"
base; $E yaml blocklist; gate FAIL "N1b forbidigo exclusion, block-list form"
base; $E yaml nopath; gate FAIL "N1c forbidigo rule with no path"
base; $E globalpaths; gate FAIL "N1d global linters.exclusions.paths entry"
base; $E yaml gosec; gate pass "N1e unrelated gosec-only exclusion"
base; $E gen; $E second ""; gate FAIL "F3  generated-file header on api.go + uncovered 2nd caller"
base; printf '// Code generated by hand. DO NOT EDIT.

package server
' > internal/server/extra.go; gate FAIL "F3b generated-file header on a new untracked file"
reset
```

```python
import sys
mode = sys.argv[1]
arg = sys.argv[2] if len(sys.argv) > 2 else ""


def rw(p, f):
    s = open(p, newline="").read()
    t = f(s)
    assert t != s, (mode, p)
    open(p, "w", newline="").write(t)


CALL = "\traw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID)"
EXCL = r"      - path: ^internal/server/api\.go$" + "\n        linters: [forbidigo]\n"
if mode == "noexcl":
    rw(".golangci.yml", lambda s: s.replace(EXCL, ""))
if mode == "nll":
    rw(".golangci.yml", lambda s: s.replace("    - noctx\n", "    - noctx\n    - nolintlint\n", 1).replace(
        "  settings:\n", "  settings:\n    nolintlint:\n      require-specific: true\n      require-explanation: true\n      allow-unused: false\n", 1))
if mode == "callsite":
    rw("internal/server/api.go", lambda s: s.replace(CALL + "\n", CALL + arg + "\n", 1))
if mode == "stale":
    rw("internal/server/api.go", lambda s: s.replace("\tgroups := p.Groups\n", "\tgroups := p.Groups //nolint:forbidigo // stale\n", 1))
if mode == "second":
    # A second exported handler-style method, so `unused` does not mask the result.
    rw("internal/server/api.go", lambda s: s + "\n// Leak is a second GroupPaths caller (spike only).\nfunc (h *handlers) Leak(ctx context.Context, p identity.Principal) error {\n\t_, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID)" + arg + "\n\treturn err\n}\n")
DIR = " //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller"
LINE = "//nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller\n"
if mode == "pkgline":
    rw("internal/server/api.go", lambda s: LINE + s.replace(CALL + DIR + "\n", CALL + "\n", 1))
if mode == "funcabove":
    rw("internal/server/api.go", lambda s: s.replace(CALL + DIR + "\n", CALL + "\n", 1).replace("\nfunc (h *handlers) GetMe(", "\n" + LINE + "func (h *handlers) GetMe(", 1))
YAML = {
    "apigo": r"      - path: ^internal/server/api\.go$" + "\n        linters: [forbidigo]\n",
    "blocklist": "      - path: ^internal/server/\n        linters:\n          - gosec\n          - forbidigo\n",
    "nopath": "      - text: GroupPaths\n        linters: [forbidigo]\n",
    "gosec": "      - path: ^cmd/nava-db/\n        linters: [gosec]\n",
}
ANCHOR = "      - std-error-handling\n    rules:\n"
if mode == "yaml":
    rw(".golangci.yml", lambda s: s.replace(ANCHOR, ANCHOR + YAML[arg], 1))
if mode == "globalpaths":
    rw(".golangci.yml", lambda s: s.replace(ANCHOR, "      - std-error-handling\n    paths:\n      - ^internal/server/\n    rules:\n", 1))
if mode == "gen":
    rw("internal/server/api.go", lambda s: "// Code generated by hand. DO NOT EDIT." + chr(10) + chr(10) + s)
```
