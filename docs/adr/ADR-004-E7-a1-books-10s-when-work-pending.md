---
type: adr-erratum
id: ADR-004-E7
title: "Erratum E7 to ADR-004 R16: A1 applies the 10 s interval when the tenant has unsealed entries; A2 stores the interval for later runs"
status: ACCEPTED
accepted: 2026-11-07 (Solution Architect verdict, TICK-600; Technical Architect round-1 critique F1–F10; CTO ruling on the path, comment 01a1b66a-6273)
date: 2026-11-07
issue: TICK-600
rfc: docs/adr/ADR-004/rfc/RFC-013-adr-004-r16-seal-interval-lag.md (rev 1; rev 0 under rfc/rev-0/)
supersedes: "ADR-004 R16, the bullet 'Interval: 10 s after a run that sealed anything; otherwise doubled, up to 300 s.' (wording only; the rule's intent is kept)"
amends: "[[ADR-004-audit-log]] R16 only. R-A2 is unchanged."
source: "TICK-443 S15 acceptance kit (PR #496), S15.9 observation"
deciders: Solution Architect (author, verdict); Technical Architect (critique, proposed the A1 probe); CTO (ruled the path: code fix to conform plus clarifying erratum)
---

# ADR-004 erratum E7: who applies the seal interval

> **Status: ACCEPTED, 2026-11-07,** by the Solution Architect (TICK-600), after the Technical Architect's round-1 critique and the CTO's ruling. This erratum **clarifies** R16. It does not change it: R16's intent, that a run which finds work is followed 10 s later, stays. The sealer code on `TICK-420/audit-sealer` does not yet meet that intent. A code fix makes it conform (the acceptance criteria are in RFC-013 rev 1; the task comes from the plan). Until that fix merges, the behaviour described under "Context" applies.
>
> **Invariants source.** The project description is empty. As with E1–E6, this erratum is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. That check is incomplete against any separate list of eight.

## Context

R16 has A1 commit the next run *o′* before A2 seals. That keeps the lineage alive through an A2 crash, A2 retry exhaustion or a poison entry (CI-A16, TICK-587). In the code, A1 books *o′* at the `seal_interval_s` the **previous** run's A2 stored, so every interval change lands one run late. After idle (stored interval 300 s):

- entries written after run *n*'s last read wait up to 300 s for *n+1* (S15.9 observed 160 s), even though *n* sealed something;
- when a burst exceeds one run's cap (10,000 entries), the leftover waits about 2 × 300 s plus run durations. That breaks R-A2 ("up to 300 s after idle") and can fire `AuditSealLag` (> 600 s) on a healthy system.

## Decision

R16's interval bullet now reads:

> **Interval.** A1, on the non-stale path and under the head lock, checks whether the tenant has any unsealed entry (`audit_log … sealed_at IS NULL`, partial index `audit_log_unsealed`). If it has, A1 books *o′* 10 s ahead and stores `seal_interval_s = 10`. Otherwise it books *o′* at the stored `seal_interval_s`. A2's last batch (or its capped batch) stores the interval for later runs: 10 s if the run sealed anything, otherwise doubled, up to 300 s. On an A2 error the interval is left as it was.
>
> **Bound, absent an A2 failure:** an entry is sealed within the delay already booked when it was written (≤ 300 s) plus one run's duration. While work is pending, runs follow every 10 s. A poison entry blocks every later entry until it is fixed; that is the `audit_seal_failures_total` / `chain:reseed` path, not this bound.

Unchanged: A1 commits first; the stale path writes nothing; A2's batching and `NextInterval`; R-A2's row; the 600 s alert.

## Options considered

| Option | Verdict | Why |
|---|---|---|
| A. Reword R16 to the current code and amend R-A2 to about 600 s | Rejected | Weakens a documented bound and blurs the alert's meaning to match a sequencing accident (TA F1, F2) |
| **B. A1 probes for pending work and books 10 s; code fix to conform, clarified here** | **Chosen** | Restores R16's intent and R-A2's bound. No workflow change, no new grant (TA F7, F10) |
| Reorder A2 before A1 | Rejected | Ends the lineage on poison or A2 retry exhaustion (TA F6) |
| A2 moves *o′*'s `available_at` forward | Rejected | A new sealer write path into `outbox` and a new grant, for the same effect |

## Consequences

- **Temporal determinism:** none. The change sits inside the A1 activity; `workflow.go` and `Rescheduled` are unchanged, so the recorded histories replay as they are. Changing either needs `workflow.GetVersion` and re-recorded histories, and is out of scope.
- **Idempotency:** unchanged. A retry before commit re-reads; a retry after commit is the stale no-op, which writes nothing.
- **Isolation and roles (R14):** no new grant. The probe is `tenant_id`-led, inside `workerguard.InTenantTx`.
- **Load:** one index probe per run, about 3.3/s per 1,000 idle tenants (R-A5).
- **Poison:** a tenant with a poison entry gets a run every 10 s, as an active tenant with poison already does today. CI-A16 pins this.
- **Docs:** `docs/architecture/audit-sealer.md` ("The interval is stored on the head", "Bounded A2", §Capacity) is rewritten with the fix.
- **No new accepted risk.** Nothing is added to `docs/security/accepted-risks.md`.

## Evidence

- Source: `internal/audit/seal/seal.go` (`Reschedule` books at the stored interval; `sealBatch` stores `NextInterval` on the last batch), `workflow.go` (A1 then A2), `activities.go` (`DefaultMaxBatches = 10`).
- Observation: TICK-443 S15.9, the second write after idle sealed after 160 s (AC3 worst case 270.8 s).
- Spike S-1 was not run. It was needed only to defend Option A. F1 follows from the source (Technical Architect, Known).
- Red to green: the fix's CI-A16 "idle tenant with backlog > cap" test fails on the current code (*o′* booked at about 300 s) and passes after the fix (about 10 s).
