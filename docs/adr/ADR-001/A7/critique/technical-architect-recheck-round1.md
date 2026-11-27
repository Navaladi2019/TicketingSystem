<!-- TICK-1183 comment 01a21bb4-0e33, Technical Architect, 2026-11-27. -->

## Re-check — round 1 (RFC-024, Technical Architect)

**VERDICT: APPROVE.** F1, the only blocking item, is resolved. I checked the attached revision itself, not only its summary.

- **Rule 9** (RFC line 126) no longer claims that only `go.mod` and `go.sum` can change what the scan covers. It now says the module layout sets the coverage (`replace`, `exclude`, `ignore`, and any nested `go.mod`) and that rule 7 bounds that layout. Resolved.
- **Rule 7** (line 120) fails the build on a tracked `go.work`, on a tracked `vendor/`, and on any `go.mod` outside the allowed set. That set matches what I found on `main`: five modules plus the `testdata/` fixtures. The set lives in `tools/ci/checkpins`, and the revised rule 2 row makes a change there scan both modules. So adding a module is a reviewed edit, not a silent drop out of the scan. Resolved.
- **Tests and risk:** V-7 (a tracked `internal/foo/go.mod`, and a tracked `vendor/modules.txt`, each red first) covers the new check. Residual risk R-A7-1 records the three unscanned tooling modules instead of claiming them away.

The non-blocking items (F2 `GOENV: 'off'`, F6 precedence text, F7 exactly one version line per scan job) are folded in as I proposed. I have no new objections, per the re-check rule.

These go into the ADR's tests or risks as accepted:
- **S4**, one scratch workflow run, is the first delivery task. It covers U3 to U5 at the pinned versions (setup-go v6, go1.27.4, govulncheck v1.9.0) and records the build time per job.
- **U1 and U2** were settled on go1.22.2 with govulncheck v1.1.3. The S4 run should repeat the `go run` exit-status check at v1.9.0, so the record shows it at the pinned version.

The Solution Architect can write ADR-001-A7.
