## 2026-11-07: TICK-594: guarderrtype covers the audit.seal message; cause and details stay runtime-checked; a seal type held in a variable is deferred
- **Decided by:** the Technical Lead, agreed by the Security & Multi-Tenancy Reviewer (review of PR #493).
- **Decision:**
  - The `guarderrtype` analyzer reports `ErrTypeAuditSealUnsealable` and `ErrTypeAuditSealHeadNotFound` outside `internal/workerguard` and `internal/audit/seal`. It also reports any call inside `internal/audit/seal` that builds one of them with a message that is not a constant.
  - No lint rule is added for the cause or the details. At runtime, `recordable` (`internal/workerguard/results.go`) turns any error that has a cause into `ErrTypeOpaque`, and `checkedDetails` drops every detail `argOK` rejects. Both fail closed, so a lint rule would only find the mistake earlier.
  - AC deviation accepted: the forge case for a non-constant message inside seal lives in the analyzer testdata, not in `internal/platform/lintfixture`.
- **Dissent:** none.
- **Deferred (Low):** inside `internal/audit/seal`, a seal type held in a variable skips the constant-message check. Seal review covers it for now. **Revisit:** when `internal/audit/seal` gets a third failure producer. Then add a rule that tracks seal types held in variables, and narrow `checkSealCall` to the `temporal.NewApplicationError*` functions.
- **Accepted-risk row:** none needed. The fix ships inside TICK-420, so `AR-TICK-585-01` is not added.
