# Review: TICK-1090, tagged lint passes (PR #852)

Verdict: **APPROVED**. No Bug or Missing-test findings. CI run 44996696176 is green. `lint` passed in 5m06s, and each of the six new passes reported `0 issues`.

## Checked

- `Taskfile.yml` and `make/lint.mk` list the same eight passes in the same order. `checktaskfile` passes in CI.
- `%v` to `%w` in `dropLinkedUsers` (`internal/identitystack/main_test.go`): the transport error and the status check are now separate branches, so `%w` only ever wraps a non-nil error. The old code printed `<nil>` on a bad status, and that is fixed too.
- `internal/workflow/interp/pause.go` is the only non-test file. It captures `tmp.Name()` once, adds the waiver, and does not change behaviour. `nolintlint` is in the waiver because G703 fires only under `interptest`, so the directive is unused in the untagged pass.
- `nolint` reasons pass `scripts/ci/nolint-allowlist.sh`.
- Cost: each tagged pass takes about 20s on the hosted runner, so the job adds about 2 minutes and stays well inside its 10-minute timeout.

## Findings

1. **Breaks under a new build tag** (follow-up, not blocking): the lint tag list is hard-coded, but `go run ./tools/ci/buildtags -list` discovers its sets. `relaytest` sits on production code (`internal/relay/seam_relaytest.go`) and is not linted. Any new tag will silently go unlinted too. This gap comes from the issue's scope, not from the code. Fix: drive the lint passes from `buildtags -list`, or have `buildtags` fail when a discovered set has no lint pass.
2. **Spec drift (in the issue, not the code)**: `stacktag accept` appears only under `tools/ci/buildtags/testdata/`, and `./...` skips that directory. The pass lints the same files as the untagged pass and costs about 20s. The issue I wrote named that set by mistake. You can drop it in the follow-up.
3. **Preference**: `tokReq, _ := http.NewRequestWithContext(...)` in `importTestRealms` throws away the error, as the neighbouring requests already do. A malformed `baseURL()` would panic instead of returning an error. Test-only code.
