---
type: rfc
id: RFC-013
title: "ADR-004 R16: the seal interval takes effect one run late"
status: Draft
date: 2026-11-07
issue: TICK-600
adr: ADR-004
proposes: ADR-004-E7 (erratum)
---

> **Invariants:** the Ticket Management System project description is empty, so the eight architectural invariants could not be loaded. The invariant check below is **incomplete**. It is made against the ADR-004 rules this change touches (R14, R15, R16, R17) and against Temporal determinism and activity idempotency.

## Decision (proposed)

**Erratum ADR-004-E7: reword the R16 interval line to describe what the code does.** No code change. The 300 s worst-case wait that the S15 acceptance criteria and the `AuditSealLag` alert rely on still holds. What the current text promises and the code doesn't deliver is how fast sealing speeds up after an idle period. It's a lag, not a broken bound.

## The drift

R16 says: *"Interval: 10 s after a run that sealed anything; otherwise doubled, up to 300 s."*

Code (`internal/audit/seal/workflow.go`, `seal.go` on `TICK-420/audit-sealer`):

1. **A1** (`Reschedule`) commits first. It books run *o′* with `available_at = now() + seal_interval_s`, the value **the previous run's A2** stored.
2. **A2** (`Seal`) seals, then its last batch writes `seal_interval_s = NextInterval(current, sealedAny)`.

So a new interval is used **one run later** than R16 says. After an idle period (interval 300 s):

| t | event | interval used for the next booking |
|---|---|---|
| 0 | run *n*: A1 books *n+1* at +300 s; A2 seals write 1 and sets 10 s | 300 s (already booked) |
| 0+ε | write 2 arrives | waits until *n+1*, up to 300 s (S15.9 observed 160 s) |
| 300 | run *n+1*: A1 books *n+2* at +10 s; A2 seals write 2 | 10 s, so steady sealing starts here |

The same lag applies when backing off: the interval doubles one run later than written. That's harmless.

**Bound unchanged:** every entry is still sealed within (the delay already booked, which is ≤ 300 s) + one run's duration. The S15 AC3 worst case was 270.8 s, and `AuditSealLag` fires at 600 s.

## Why A1 commits first (not negotiable here)

R16 deliberately has A1 commit the next run **before** A2 does any work. A2 can then crash, exhaust its 10 attempts, or hit a poison entry (`Unsealable`), and the lineage stays alive without `chain:reseed`. CI-A16 tests exactly this ("kill between A1 and A2", "poison"). Any fix that runs A2 before A1 brings back the dead-lineage failure, so neither option below reorders the activities.

## Options

### Option A: Erratum ADR-004-E7 (recommended)

Replace the R16 interval line with:

> **Interval.** A2's last batch stores `seal_interval_s` on the head: 10 s if the run sealed anything, otherwise doubled, up to 300 s. A1 commits first and books the next run with the interval stored at that moment, which the **previous** run's A2 set. A change therefore takes effect one run later. After an idle period the first two runs that find work can each be up to 300 s apart; 10 s sealing starts from the second of them. Bound: an entry is sealed within the booked delay (≤ 300 s) plus one run's duration.

| Concern | Impact |
|---|---|
| Temporal determinism | None. No workflow or activity code changes; the recorded histories (`testdata/histories/*.json`) replay as before. |
| Idempotency | None. A1's stale-lineage no-op and the A1-retried-after-commit path are unchanged. |
| Tenant isolation / roles (R14) | None. No new statement or grant. |
| Tests / kit | None. The S15 kit already tests this behaviour (S15.9). `docs/architecture/audit-sealer.md` already says it ("o′ takes the interval the previous run's A2 left"). |
| Cost | One docs PR (`.md` + `.html` + this RFC archived under `docs/adr/ADR-004/`). |
| What we accept | After idle, a burst's second entry can wait up to 300 s instead of about 10 s, once per idle-to-busy transition per tenant. |

### Option B: A1 resets the interval itself when work is pending

The CTO's framing was "A2 resets the interval before A1 books the next run". Taken literally, that means running A2 first, which breaks the liveness property above. The workable form moves the reset **into A1**, keeping A1-first:

- In A1's transaction, under the head lock and before `EnqueueSeal`, it runs `SELECT EXISTS (SELECT 1 FROM audit_log WHERE tenant_id = $1 AND sealed_at IS NULL)`. This is an index-only probe on `audit_log_unsealed`.
- If anything is pending, it books *o′* at 10 s and writes `seal_interval_s = 10`. Otherwise it books at the stored interval, as today.
- A2's `NextInterval` stays as is (it still handles doubling and the capped-run 10 s).

| Concern | Impact |
|---|---|
| Temporal determinism | Safe **if** the change stays inside the A1 activity body and `Rescheduled` keeps its shape. Activity results are replayed from history, not re-run, so the workflow function and its command sequence don't change and the recorded histories still replay. If the result type or the workflow code changes, it needs `workflow.GetVersion` plus re-recorded histories (`make seal-record-histories`). Implementers must not take that path. |
| Idempotency | Holds. An A1 retry before commit rolls back and re-reads (same or newer decision). A retry after commit is the existing stale no-op. Two concurrent A1s still serialise on the head lock (CI-A16). |
| Tenant isolation / roles (R14) | `nava_worker` already reads `audit_log` (A2) and already writes `seal_interval_s`. No new grant. The probe must stay `tenant_id`-led inside `workerguard.InTenantTx` (RLS), and `audit_paths.go` role-matrix cases must still pass. |
| Residual | A write that lands between A1's commit and A2's read still waits one long interval. The window is milliseconds, not one run. |
| Cost | It needs an **amendment** (ADR-004-A2), not an erratum, because behaviour changes. On top of that: an A1 code change, updated interval-rule and CI-A16 "idle tenant" tests, a new S15.9 expectation in the kit, and the `audit-sealer.md` rewrite. One more query per run per tenant (cheap, but it runs on every idle tenant every ≤ 300 s). Estimate: one task, about 1 day plus review. |
| What we gain | After idle, the second entry is sealed in about 10 s instead of up to 300 s. |

## Recommendation and why

**Option A.** Sealing is an integrity process. It isn't read latency: unsealed entries are already returned by `internal/audit/read` (with `sealed_at` null). Only `AfterSeq` consumers, such as export cursors, wait for a seal, and they have no 10 s SLA. The guarantees that matter (≤ 300 s per entry, alert at 600 s, no dead lineage) hold under both options. Option B spends an amendment, a code change, test churn and a per-run query to improve a cadence nobody depends on. If the product later wants seals within seconds after idle, Option B is the path. That is a product request, not a correctness fix.

## Open questions for the Technical Architect

1. Do you agree the literal reorder (A2 before A1) is off the table on liveness grounds, so Option B can only be the A1-probe form?
2. Is anything (a verifier, an export SLA, an S16+ criterion) relying on the 10 s cadence right after idle? If so, Option B becomes the recommendation.
3. Numbering: I assumed **RFC-013** (highest found in the repo is RFC-012) and **ADR-004-E7** (highest erratum is E6).
