# Spike: forbidigo line-scoped nolint + nolintlint (RFC-006)

Tree: `origin/TICK-47/group-changes-within-10-minutes` (PR #221 head), run on 2026-10-28, Windows. Lint target: `./internal/server/`. Each case resets `.golangci.yml` and `internal/server/api.go` first.

nolintlint settings under test: `require-specific: true`, `require-explanation: true`, `allow-unused: false`.

```
golangci-lint has version 2.14.0 built with go1.26.10 from (unknown, modified: ?, mod sum: "h1:ot8QffRa4LzAAEgtvNYrVs9esxQ7xoTwoAv6uhp4ngA=") on (unknown)
### 0 as-is (file exclusion)
    0 issues.
### 1 exclusion removed, no directive
    internal\server\api.go:94:14: use of `h.paths.GroupPaths` forbidden because "group paths are display-only (ADR-002); only GetMe may call GroupPaths" (forbidigo)
    1 issues:
### 2 exclusion removed + line nolint
    0 issues.
### 3 = 2 + nolintlint
    0 issues.
### 4 = 3 + stale nolint on a clean line
    internal\server\api.go:90:21: directive `//nolint:forbidigo // stale` is unused for linter "forbidigo" (nolintlint)
    1 issues:
### 5 bare //nolint + nolintlint
    internal\server\api.go:94:65: directive `//nolint` should mention specific linter such as `//nolint:my-linter` (nolintlint)
    1 issues:
### 6 nolint:forbidigo without explanation + nolintlint
    internal\server\api.go:94:65: directive `//nolint:forbidigo` should provide explanation such as `//nolint:forbidigo // this is why` (nolintlint)
    1 issues:
### 7 second caller in api.go, no directive
    internal\server\api.go:116:12: use of `h.paths.GroupPaths` forbidden because "group paths are display-only (ADR-002); only GetMe may call GroupPaths" (forbidigo)
    1 issues:
### 8 second caller with copied directive
    0 issues.
### 9 exclusion kept + nolint + nolintlint
    internal\server\api.go:94:65: directive `//nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller` is unused for linter "forbidigo" (nolintlint)
    1 issues:
### 10 nolint:all with explanation + nolintlint
    internal\server\api.go:94:65: directive `//nolint:all // ADR-002 reason` should mention specific linter such as `//nolint:my-linter` (nolintlint)
    1 issues:
### 11 whole repo: exclusion removed + line nolint + nolintlint
```

Case 11 (the whole repo with the proposed config) timed out at 500 s locally, so there is no result for it. That is AC5 in the RFC.

## Harness

```sh
E="python -I ../spike-scripts/ed.py"
reset(){ git checkout -q -- .golangci.yml internal/server/api.go; }
run(){ echo "### $1"; timeout 300 golangci-lint run ./internal/server/ 2>&1 | grep -E "\.go:[0-9]+|issues|Error|level" | sed 's/^/    /'; }
NL=' //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller'
case "$1" in
0) reset; run "0 as-is (file exclusion)";;
1) reset; $E noexcl; run "1 exclusion removed, no directive";;
2) reset; $E noexcl; $E callsite "$NL"; run "2 exclusion removed + line nolint";;
3) reset; $E noexcl; $E callsite "$NL"; $E nll; run "3 = 2 + nolintlint";;
4) reset; $E noexcl; $E callsite "$NL"; $E nll; $E stale; run "4 = 3 + stale nolint on a clean line";;
5) reset; $E noexcl; $E callsite ' //nolint'; $E nll; run "5 bare //nolint + nolintlint";;
6) reset; $E noexcl; $E callsite ' //nolint:forbidigo'; $E nll; run "6 nolint:forbidigo without explanation + nolintlint";;
7) reset; $E noexcl; $E callsite "$NL"; $E nll; $E second ""; run "7 second caller in api.go, no directive";;
8) reset; $E noexcl; $E callsite "$NL"; $E nll; $E second "$NL"; run "8 second caller with copied directive";;
10) reset; $E noexcl; $E callsite " //nolint:all // ADR-002 reason"; $E nll; run "10 nolint:all with explanation + nolintlint";;
9) reset; $E callsite "$NL"; $E nll; run "9 exclusion kept + nolint + nolintlint";;
esac
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
```
