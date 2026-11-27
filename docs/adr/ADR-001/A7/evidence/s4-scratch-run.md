# ADR-001-A7 S4: scratch workflow run

Evidence for ADR-001-A7 decision A7.5 (spike S4), task TICK-1185. **Result: PASS, all five assumptions hold.**

- **Run:** https://github.com/Navaladi2019/TicketingSystem/actions/runs/46008327636 (conclusion `success`, 2026-11-27)
- **Jobs:** `root` (job 140537066765), `temporal-server` (job 140537066286), both `ubuntu-latest`
- **Commit:** `e56c56b15033b98579c675f17ede67e20e5b3483` on `scratch/a7-s4`, one commit on top of `main` `a66dd2fd`
- **Branch:** `scratch/a7-s4` was deleted after the run. The scratch workflow never reached `main`.
- **Pins:** `actions/checkout@v4`, `actions/setup-go@v6` (`go-version-file`, `cache: false`), `golang.org/x/vuln/cmd/govulncheck@v1.9.0`. Both `go.mod` files say `go 1.27.0` / `toolchain go1.27.4`.

## What ran

Each job: checkout, setup-go, then

- **step A:** `echo "GOFLAGS=-tags=x" >> "$GITHUB_ENV"`
- **step B:** the A7.1 `env:` block verbatim (`GOENV: 'off'`, `GOWORK: 'off'`, `GOFLAGS: ''`, `GOTOOLCHAIN: local`, `GOPROXY: https://proxy.golang.org`, `GOSUMDB: sum.golang.org`, `GONOSUMDB: ''`, `GOPRIVATE: ''`, `GOINSECURE: ''`), running `echo "GOFLAGS=[$(go env GOFLAGS)]"`, `go env GOFLAGS`, `go version`, `time go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -version`, `time go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`. `temporal-server` runs it with `working-directory: temporal-server`.
- **step C** (`root` only, same `env:`): a throwaway module in `$RUNNER_TEMP/s4vuln` requiring `golang.org/x/text v0.3.5` whose `main` calls `language.Parse` (GO-2021-0113, reachable); `set +e`, `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`, `echo "exit=$?"`, `exit 0`.

## Results

| # | Criterion | root | temporal-server | Pass |
|---|---|---|---|---|
| 1 | `go env GOFLAGS` empty although step A wrote `GOFLAGS=-tags=x` | `GOFLAGS=[]`, then an empty line | `GOFLAGS=[]`, then an empty line | yes |
| 2 | `go version` = go1.27.4 | `go version go1.27.4 linux/amd64` | `go version go1.27.4 linux/amd64` | yes |
| 3 | `govulncheck@v1.9.0 -version` builds and reports v1.9.0 | `Go: go1.27.4` / `Scanner: govulncheck@v1.9.0` | `Go: go1.27.4` / `Scanner: govulncheck@v1.9.0` | yes |
| 4 | `go run` exits non-zero on a reachable finding (A7.1 expects 1, not 3) | govulncheck `exit status 3`, `go run` `exit=1` | n/a (root only) | yes |
| 5 | Wall time recorded | see below | see below | yes |

### Wall time (`time`, `real`)

| Job | `-version` build (cold, `cache: false`) | scan `./...` |
|---|---|---|
| root | 0m26.366s (user 0m58.411s) | 0m7.977s (user 0m15.496s) |
| temporal-server | 0m18.582s (user 0m37.978s) | 0m50.996s (user 1m15.590s) |

Whole jobs: `root` 13:34:51 to 13:35:37 (46 s), `temporal-server` 13:34:50 to 13:36:06 (76 s). Both real scans report `No vulnerabilities found.` / `Your code is affected by 0 vulnerabilities.`

## Log lines

`root`, step B:

```
2026-11-27T13:34:58.8857120Z   GOFLAGS:
2026-11-27T13:34:58.8958963Z GOFLAGS=[]
2026-11-27T13:34:58.8989300Z
2026-11-27T13:34:58.9023276Z go version go1.27.4 linux/amd64
2026-11-27T13:35:25.1856767Z Go: go1.27.4
2026-11-27T13:35:25.1857323Z Scanner: govulncheck@v1.9.0
2026-11-27T13:35:25.2690603Z real	0m26.366s
2026-11-27T13:35:33.1198017Z Your code is affected by 0 vulnerabilities.
2026-11-27T13:35:33.2466819Z real	0m7.977s
```

`temporal-server`, step B:

```
2026-11-27T13:34:55.0918641Z   GOFLAGS:
2026-11-27T13:34:55.0999090Z GOFLAGS=[]
2026-11-27T13:34:55.1026412Z
2026-11-27T13:34:55.1048784Z go version go1.27.4 linux/amd64
2026-11-27T13:35:13.6224497Z Go: go1.27.4
2026-11-27T13:35:13.6224965Z Scanner: govulncheck@v1.9.0
2026-11-27T13:35:13.6869944Z real	0m18.582s
2026-11-27T13:36:04.1247299Z Your code is affected by 0 vulnerabilities.
2026-11-27T13:36:04.6831191Z real	0m50.996s
```

`root`, step C:

```
2026-11-27T13:35:33.8110852Z require golang.org/x/text v0.3.5
2026-11-27T13:35:35.3280082Z Vulnerability #1: GO-2021-0113
2026-11-27T13:35:35.3308332Z ##[error]      #1: main.go:10:26: s4vuln.main calls language.Parse
2026-11-27T13:35:35.3319047Z Your code is affected by 1 vulnerability from 1 module.
2026-11-27T13:35:35.3590665Z exit status 3
2026-11-27T13:35:35.3627976Z exit=1
```

The `GOFLAGS:` lines (trailing space trimmed here) are the step's `env:` listing printed by the runner. The empty line after `GOFLAGS=[]` is the plain `go env GOFLAGS` output.

## Notes

- Criterion 4 confirms S2 at v1.9.0: govulncheck exits 3 and `go run` reports that as exit 1. No check may expect 3 (A7.1).
- The two scans differ a lot (8 s vs 51 s) because `temporal-server` pulls in the Temporal server dependency graph. The cold `-version` build (19 to 26 s) is paid once per job because `cache: false`. Not a gate. Budget input for later tasks only.
- The workflow push needed the API upload path (`gh api ... contents`). The token has no `workflow` scope, as the workspace rule expects.
