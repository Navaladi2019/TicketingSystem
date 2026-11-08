---
type: rfc
id: RFC-013
title: "ADR-004 R16: the seal interval takes effect one run late"
status: Revised (rev 1), closed by ADR-004-E7
date: 2026-11-07
issue: TICK-600
adr: ADR-004
outcome: "Code fix to conform to R16 (A1 books 10 s when work is pending) + clarifying erratum ADR-004-E7"
previous: docs/adr/ADR-004/rfc/rev-0/RFC-013-adr-004-r16-seal-interval-lag.md
---

> **Invariants.** The project description is empty. As in ADR-004 and its errata E1–E6, this RFC is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. That check is incomplete against any separate list of eight.

## Changelog (rev 0 → rev 1)

| Item | rev 0 | rev 1 |
|---|---|---|
| Decision | Option A: erratum, no code change | **Option B**: code fix so the code conforms to R16, plus clarifying erratum **E7** (TA F8; CTO ruling) |
| F1 (blocking) | Claimed "each entry sealed within ≤ 300 s plus one run" | Withdrawn. With the current code, a post-idle backlog bigger than one run's cap waits **two** booked delays (about 600 s plus run durations). Option B removes that case |
| F2 (blocking for A) | — | Not needed: spike S-1 applies only to Option A. Under B, the capped post-idle run books its successor at 10 s |
| F3 | — | `audit-sealer.md` "interval is stored on the head", "Bounded A2" and §Capacity rewritten as part of the fix task |
| F4 | "the second entry … can wait up to 300 s" | Every entry written after run *n*'s last read waits for run *n+1* |
| F5 | Bound had no qualifier | Bounds hold "absent an A2 failure" (a poison entry blocks every later entry) |
| F9 | — | CI-A16 poison fan-out test added to the fix's acceptance criteria |
| F10 | — | Only A1's non-stale path writes; the stale path still writes nothing |

## The drift (unchanged facts, corrected consequence)

R16: *"Interval: 10 s after a run that sealed anything; otherwise doubled, up to 300 s."*

Code (`internal/audit/seal/seal.go` on `TICK-420/audit-sealer`): A1 (`Reschedule`, lines 95–132) commits first and books *o′* at the `seal_interval_s` it reads under the head lock. A2's last batch (lines 267–271) only then stores `NextInterval(...)`. So every interval change takes effect one run later than R16 says.

Consequences after idle (stored interval 300 s), absent an A2 failure:

1. **Small burst.** Run *n* seals what it finds. Every entry written after run *n*'s last read waits for run *n+1*, which is already booked at +300 s. At AC2's 14 entries/s that is about 4,200 entries. S15.9 observed 160 s.
2. **Burst larger than one run's cap (F1).** Run *n* seals 10,000 entries (`DefaultMaxBatches` × 1,000) and stores 10 s, but *n+1* is already booked at +300 s. The leftover entries wait about 2 × 300 s plus run durations and relay latency. `AuditSealLag` (`> 600`, no `for:`) can then fire on a healthy system (F2). This breaks R-A2 ("up to 300 s after idle") and the TICK-440 "bounded A2" continuation, which assumes a capped run is followed 10 s later.

## Options

### Option A: amendment A2, widen the bound (rejected)

Keeping the code would mean amending R-A2 to "≤ two booked delays (≤ 600 s) plus run durations when a post-idle backlog exceeds one run's cap, absent an A2 failure". The `AuditSealLag` threshold or a `for:` clause would need re-deciding, backed by spike S-1. That trades a documented bound and a meaningful alert for no code. **Rejected:** it weakens a guarantee to match a sequencing accident.

### Option B: A1 books 10 s when the tenant has unsealed entries (chosen)

Keep A1-first: reordering would end the lineage on poison or A2 retry exhaustion (F6). Inside A1's transaction, on the **non-stale path only**, after the head lock and before `EnqueueSeal`:

```sql
SELECT EXISTS (SELECT 1 FROM audit_log WHERE tenant_id = $1 AND sealed_at IS NULL)
```

- **true**: book *o′* with `WithDelay(10 s)` and set `seal_interval_s = 10` in the same transaction, so `next_seal_at` and the stored interval agree.
- **false**: book at the stored `seal_interval_s`, as today.
- A2's `NextInterval` is unchanged and still drives doubling while idle.

| Concern | Impact |
|---|---|
| Temporal determinism | None (TA F7). The change is inside the A1 activity body. `Rescheduled{NextOutboxID}` and `workflow.go` are unchanged. The replayer feeds activity results from history, so the recorded histories under `testdata/histories/` replay with no `GetVersion`. Any change to the workflow function or the result type is out of scope and needs `GetVersion` plus re-recorded histories. |
| Idempotency | Holds. A retry before commit rolls back and re-reads. A retry after commit takes the stale path, which writes nothing. Concurrent A1s serialise on the head lock (CI-A16). |
| Tenant isolation / roles (R14) | No new grant: `nava_worker` already reads `audit_log` (A2) and writes `seal_interval_s`. The probe is `tenant_id`-led, inside `workerguard.InTenantTx` (RLS), and served by the partial index `audit_log_unsealed`. |
| Cost | One index probe per run. R-A5 idle load is about 3.3 runs/s per 1,000 tenants, so about 3.3 probes/s (TA F10). |
| Poison (F9) | A tenant with a poison entry always has unsealed entries, so A1 books every 10 s. Today an active tenant with poison does the same, because A2 leaves the interval at 10 on error. That gives about 36 overlapping runs over A2's roughly 6 min retry window, serialised on the head lock. It is not a new failure mode; the fix pins it with a test. |
| Residual | An entry written between A1's probe and A2's last read, when the probe found nothing, is sealed by this run's A2. An entry written after A2's last read waits for the booked delay (≤ 300 s); that run's A1 then sees it and books 10 s. |

**Resulting bound (absent an A2 failure):** an entry is sealed within the delay already booked when it was written (≤ 300 s), plus one run's duration. After that, while work is pending, runs follow every 10 s. A backlog above about 950 entries/s still grows (the §Capacity limit), and `AuditSealLag` is the signal for that. R-A2 stays as written.

## Acceptance criteria for the fix task (Technical Lead decomposes)

1. `Reschedule` probes for unsealed entries on the non-stale path only. If any exist, it books *o′* at 10 s and writes `seal_interval_s = 10`; otherwise it books at the stored interval. The stale path writes nothing (existing test still green).
2. `workflow.go` and `Rescheduled` are unchanged; the replay gate passes on the existing histories with no re-recording.
3. Integration tests (real PostgreSQL): A1 with pending work books at 10 s; A1 with none books at the stored interval; CI-A16 **idle tenant with backlog > cap**: head at 300 s, 20,000 entries, run *n* → *o′* `available_at − now() ≈ 10 s`; CI-A16 **poison fan-out** (F9): a post-idle tenant with a poison entry gets runs at 10 s, none of them uses a `seq`, and the lineage stays alive.
4. S15 kit: update S15.8/S15.9 so the second write after idle is sealed within about 10 s plus one run.
5. `docs/architecture/audit-sealer.md`: rewrite "The interval is stored on the head", "Bounded A2" and §Capacity (F3: in steady state the next run is due 10 s after A1 commits, so runs overlap under the head lock).

## What was rejected and why

| Option | Why not |
|---|---|
| Reorder A2 before A1 | Ends the lineage on poison or A2 retry exhaustion (F6, TICK-587) |
| A2 rewrites *o′*'s `available_at` after sealing | A new write path into `outbox` from the sealer, needing a new grant and a relay-claim race to reason about. The A1 probe gets the same effect with no new grant |
| Amendment widening R-A2 to about 600 s (Option A) | Weakens a documented bound and blurs the 600 s alert to match a sequencing accident |
