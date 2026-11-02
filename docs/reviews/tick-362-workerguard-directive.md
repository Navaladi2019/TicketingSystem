## TICK-362 (P3-26): review of PR #307, `workerguard.Directive`, AR-042, AR-044

Reviewer: Quality, Testing & CI Advisor. Base: `TICK-334/workflow-interpreter`. Reviewed against the TICK-362 contract and its acceptance criteria.

**VERDICT: APPROVE.** Nothing blocks the merge. The one gap is a missing test, and this review adds it.

### Findings (most severe first)

1. **Missing test (fixed in this review).** `internal/workerguard/results.go:292`: in a real worker, the rule that refuses a local activity's `Directive` (`resultOK(result, !local)`) had no test.
   - `TestDirectiveInWorkflow/local activity Directive is refused` passes for a different reason. The SDK test environment gives the guard a local activity's result already encoded (`*commonpb.Payloads`), so the generic rule refuses it whatever `local` is set to.
   - Mutation check: I replaced `!local` with `true`, and every `internal/workerguard` test still passed.
   - Impact if it broke: low. `directiveOK` still allows only references in a local activity's marker. But the contract says this case is refused.
   - Fix: added `internal/workerguard/directive_local_test.go`. It calls `activityOutcome` with a Go-valued `Directive`: it is recorded when `local=false` and withheld with `nava.TenantGuard`/`record_data` when `local=true`. With the mutation in place, this test fails.
2. **Preference.** A typed nil `*Directive` is refused, because it encodes as `null` and `directiveOK` needs an object. Before this change, the generic rule allowed a `null` result. The interpreter should never return a nil pointer, so failing closed is the right default. A one-line comment on `isDirective` would make that a stated choice.
3. **Preference.** `guarderrtype` folds only constant strings, so `"nava." + suffix` built at run time gets past it. The package doc says so, and `idpclaims` has the same limit. No action needed.

### Acceptance criteria

| Criterion | Status | Evidence |
|---|---|---|
| Each valid kind passes | ✅ | `TestDirectiveEachKindPasses`, as a value and as a pointer; minimum and maximum waits |
| CI-7 negatives | ✅ | `TestDirectiveGoValueNegatives` and `TestDirectiveEncodingNegatives`: all listed cases, plus repeated keys, trailing data, non-canonical UUIDs, and a wait over int32 |
| Lookalike struct, map, custom `MarshalJSON` | ✅ | `TestDirectiveOtherTypesRefused`, which also covers an embedded `Directive` |
| Test-env workflow test | ✅ | `TestDirectiveInWorkflow`; the local-activity plumbing is now also covered by finding 1 |
| Existing guard tests green | ✅ | `go test ./internal/workerguard/` passes locally |
| `navalint` fixture and `task lint` | ⏳ | `lint-fixture.sh` expects 4 findings (2 uses × 2 passes). I did not run `tools/navalint` tests locally because the analysistest build stalled on this host; read the CI `lint` job |

### Checked and fine
- Bytes are checked, not Go fields. The token-by-token read refuses unknown and repeated keys and trailing data. `stringField` refuses `null`.
- The per-kind table matches the contract exactly. `directiveRules` is the only list of valid kinds.
- Only regular activity results get `directive=true`. Workflow, update, side-effect and argument paths still use the generic rule (`TestDirectiveOnlyAsActivityResult`).
- No new failure types. The AR-044 row is deleted. The AR-042 rule is in `review-standard-core.md`.

### How to run
`go test ./internal/workerguard/` locally. Lint and fixture results come from the GitHub `ci` workflow on PR #307.
