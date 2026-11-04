# Review: TICK-552 (P5-19a), PR #451 and PR #450

Reviewer: Quality, Testing & CI Advisor. Source: findings 0-4 of `docs/reviews/tick-436-audit-isolation-role-matrix.md` and the TICK-552 body. The Technical Lead's ruling of 2026-11-04 replaces the CTO's allowlist ruling: finding 0 is fixed through `read.Reader`, the TICK-541 guard stays unchanged, and TICK-555 is cancelled.

## PR #451: finding 0 (reader probes). VERDICT: APPROVE, merged

What was checked:

- **The probes run as each role, not a privileged pool.** `auditReaderProbe` takes `cl.Pool(role)`, which logs in as the manifest role (`harness_isolation_test.go`, `cluster.Pool`). `inTx` sets the tenant through `tenancy.InTenantTx`, or sets no tenant for `uuid.Nil`. `read.Reader.List` filters with `WHERE tenant_id = $1` on the tenant it is asked for. It does not call `set_config` or `SET ROLE`. Asking for tenant A from inside tenant B's transaction is therefore a real forged-tenant call, and only row-level security stops it.
- **Red run.** I checked the excerpt in the PR against the code. I did not reproduce it, because the isolation suite is a CI job. With a planted `USING (true)` SELECT policy, the existing policies and the planted one are permissive and are ORed, so the reader returns tenant A's row in both forged cases. The excerpt shows exactly those two failures. The tenant-B-asks-for-B case can't go red under that plant, because the reader's own `WHERE tenant_id` hides the row. It is a consistency case, not an RLS proof. That is acceptable, because the forged cases carry the proof.
- **Positive control.** nava_app in tenant A gets tenant A's entry back with `a-only` in its diff and a non-empty salt, so an empty result can't pass silently.
- **Guard.** `TestAuditDiffReadOnlyInReader` is unchanged and passes. `go test ./internal/audit/read/...` passes locally.
- **CI:** every check on #451 is green, `unit` and `stack` included. Squash-merged into `TICK-419/one-audit-log` as `6580cd2`.

Findings: none blocking.

## PR #450: findings 1-4. VERDICT: APPROVE (merge on green CI after the rebase onto `6580cd2`)

1. **Finding 1, `audit_log.diff`: meets the criterion.** The `audit_log` markers reach the table only inside `diff`. A compressed marker diff hides the marker, and the existing marker byte control then fails with "the scan proves nothing". Red run A, which pads the record note, shows this failure. This avoids a direct `diff` read that the guard would flag.
2. **Breaks under incompressible fixture growth (non-blocking; follow-up).** `auditControlsUncompressed` checks only `pg_column_compression(v.schema) IS NULL`. Suppose a control schema grows past about 2 kB with content pglz can't shrink by 25%. PostgreSQL then stores it out of line and uncompressed, and `pg_column_compression` returns NULL. TOAST chunks of about 2 kB, each with its own tuple header, can then split a canary, so the byte scan misses it. Fix: also assert `pg_column_size(v.schema) < 1900`. A red run that pads with incompressible text would show the check working. Today's fixtures are far below the threshold, so this is not wrong today.
3. **Finding 2: OK.** `composeDefaults` checks `j < 0` before slicing. `TestComposeDefaults` covers three unterminated cases plus the missing-default case, and it recovers from panics.
4. **Finding 3: OK.** The Coverage strings name P5-42 (S16). The CTO has signed off on `registry.go`, and TICK-447 owns CI-A5 export bytes.
5. **Finding 4: OK.** The labels `every login` and `nava_app, nava_worker` read correctly.
6. **Spec drift (issue text):** the column is `schema`, not `definition`. The code is right.

## Commands

- Local: `go test ./internal/tenancy/isolation/ -run TestComposeDefaults`, `go test ./internal/audit/read/...`.
- CI: the `ci` workflow on each PR (`gh pr checks <pr>`).
