---
type: adr
id: ADR-008
title: Competitive Gate in every ADR — four named axes against Salesforce, ServiceNow and Freshworks, parity on the rest, no unaccepted repeat of a registered competitor pain point
status: ACCEPTED
date: 2026-11-15
accepted: 2026-11-15
issue: TICK-857
rfc: RFC-020 (rev 1, docs/adr/ADR-008/rfc/RFC-020-competitive-gate.md; Technical Architect critique round 1 and re-check inside it). Renumbered from RFC-019, which ADR-004-E9 took in parallel.
supersedes: none
---

# ADR-008: Competitive Gate in every ADR

**Status: ACCEPTED** on 2026-11-15 by the Solution Architect (merge policy set by vknavaladi). Technical Architect critique round 1 was `CHANGES` (B1, B2); the re-check of rev 1 was `APPROVE`. The CTO decided Q4, and its two guards are in D6. No tie-break was needed. Q1 and Q2 are open with the human and do not block this ADR. Verdict on TICK-857, PR #683.

> **Invariants source.** TICK-857 has no project, so there is no project description to read invariants from. This ADR is checked against [[ADR-001-platform-blueprint]] §1, I1–I9.

## In plain words

- **What changes:** every new design record must say, with a number and how it is measured, how it makes us better than Salesforce Service Cloud, ServiceNow and Freshworks on one of four chosen measures. It must also name which of their known problems it avoids, or risks repeating.
- **Why:** the founder wants every design to aim at being ten times better than those products. "Ten times better" cannot be checked as written, so this ADR turns it into a short section that a reviewer can pass or fail.
- **What you'll see:** a register of 17 sourced competitor problems (`docs/adr/competitive/pain-points.md`), an ADR template with the gate section (`docs/adr/_template-adr.md`), and a one-line result for each accepted ADR (below).
- **Codes used below:** `CP-SF/SN/FW-nn` = an entry in the register. `X1–X4` = the four measures ("axes"). `M-…` = how a measure is taken. `I1–I9` = the platform invariants in ADR-001.

## Decision

| # | Decision |
|---|---|
| **D1** | **Vehicle.** This ADR plus `docs/adr/competitive/pain-points.md` (the register) and `docs/adr/_template-adr.md` (the template). ADR-001 is not changed. |
| **D2** | **Register.** Each entry has an ID `CP-<SF\|SN\|FW>-nn`, the pain point, a hypothesis result (confirmed, killed or unconfirmed), at least one source link and an evidence grade. IDs are never reused. The Solution Architect owns it and reviews it every 6 months, or when a vendor changes pricing, packaging or limits. |
| **D3** | **Grades.** A = vendor documentation or a public incident report. B = aggregated reviews or community threads. C = blogs and single anecdotes. Any grade can support "avoids". A send-back for "repeats" needs an A or B source. |
| **D4** | **Four axes, five metrics.** Every claim needs a number and the method that produces it. A **multiplier** ("10x") may be cited only when both sides were measured on the same quantity. |
| **D5** | **Parity, not trade.** Isolation, security, identity, audit and compliance, and reliability (I1–I9) are parity or better. An ADR that trades any I1–I9 property for an axis gain is a design escalation to the CTO, not a gate answer. |
| **D6** | **Mandatory section.** Every ADR numbered above 008, and every amendment or erratum dated on or after this ADR's acceptance, has `## Competitive Gate` in the template's shape. The result is one of: `PASS`, `PASS with accepted risk <CP-id>: <reason, owner, revisit trigger>`, or `N/A: <one line>`. **In-flight ADRs (Q4, CTO decision 2026-11-15):** an ADR already in critique when this one is accepted does not fill in the gate. Two guards apply. (1) "In critique" means the Technical Architect's round-1 critique was posted before ADR-008's `accepted` date. An RFC or ADR with no posted critique yet must fill in the gate. (2) An exempt ADR is back-applied once it is accepted: whoever accepts it appends a one-line result to this ADR's back-apply table. An erratum is raised only when it clearly repeats a grade A or B pain point. |
| **D7** | **Send-back.** The Technical Architect checks the gate in the critique, and a failing gate is a `blocking` objection. An ADR fails the gate when it: (a) makes a claim with no number or no method; (b) repeats a grade A or B pain point with no accepted reason; or (c) uses `N/A` while changing a tenant-visible behaviour, an API, a limit, a configuration surface or an upgrade path. The Solution Architect does not set `ACCEPTED` on an ADR that fails the gate. |
| **D8** | **The gate judges, it does not add scope.** "Axis moved: none" is a valid `PASS`. |
| **D9** | **Measured vs intended.** A method not yet in the repo is cited as "planned", and the claim counts as a target, not evidence. |
| **D10** | **Back-apply without rewriting.** Accepted ADRs get one line each (below). An erratum is raised only for an unaccepted repeat of a grade A or B entry. Result: none raised. |
| **D11** | **Limit test** (for CP-SF-02 and CP-FW-06). A platform limit does not count as a repeat when all five hold: (1) an invariant requires it; (2) it is published; (3) it is the same at every tier; (4) it returns a typed error before any side effect; (5) it is sized from a measurement. If any one fails, it is a "repeats" finding. |
| **D12** | **Additive configuration surface.** The workflow step catalog, the script surface and the type-definition keywords are additive-only within a major version, like the REST contract (ADR-001 D6). Removing an item needs a new major version and a migration note. |

### The axes

| Axis | Metric | Competitor baseline | Target | Method | 10x status |
|---|---|---|---|---|---|
| **X1a** First routed record, self-serve | Time from "tenant onboarded" to "first record routed by a workflow the admin configured from a published example type", with no platform staff | Unmeasured on the same task | ≤ 8 h (aim 4 h) | M-X1a: timed acceptance-kit run (planned, after S11b and TICK-786); CG-SP-1 for competitors | Unproven. No multiplier. |
| **X1b** Production go-live, reference service desk | Contract to go-live for a defined reference scope | Months of partner-led work: CP-SF-01 (2–6 months standard), CP-SN-02 (3–6 months to first go-live) | Set after the first pilot | M-X1b: timed pilot (planned) | Unmeasured on our side. No multiplier. |
| **X2** Change and upgrade cost | (a) Steps to add a field, form rule or workflow step. (b) Tenant configuration items that need review after an upgrade within a major version | Review needed for every modified record (CP-SN-03). Customer regression testing three times a year (CP-SF-03) | (a) One publish, zero code, zero tenant-authored DDL. The one exception is the D25 indexed attribute, which the operator runs. (b) Zero items. | M-X2: upgrade-compatibility suite (planned). It uses every step type and keyword, plus one deprecated item, and fails on any removal. | Zero against per-record review. Stated as zero, not as a multiplier, until M-X2 runs. |
| **X3** Speed | Our in-process p95 budgets: read ≤ 50 ms, create ≤ 50 ms, list ≤ 150 ms (ADR-006 D13 band ≤ 350 ms). Competitor comparison only client-observed on both sides | No public number (CP-SF-05, CP-FW-01 are qualitative) | Budgets met. Known misses: the D13 band at 221–327 ms, and `data` filters at 224 ms (TICK-646) | M-X3: `docs/quality/acl-perf-800k.md`, `docs/budgets.md`; CG-SP-1 | Unproven |
| **X4** Cost to serve | Infrastructure cost per agent seat per month, at 50 and 500 seats | List prices: CP-SF-04 ($175 / $350 per user per month); CP-SN-01 (~$90–150, estimates only); CP-FW-04 (~$55 / $89) | **Human decides (Q1)** | M-X4: cost model (planned; owner TL with the CTO) | Not claimed |

**Spike CG-SP-1**, run only if Q2 = yes. It times both sides client-observed, from the same host and region, with 30 or more samples per action and the raw timings attached. Pass = a ratio of 0.1 or less on at least one action per vendor. Fail = X3 is demoted to parity, and this ADR is amended to say so.

## Back-apply (accepted ADRs)

| ADR | Gate result |
|---|---|
| ADR-001 (+A1, A2, A5, A6, E3) | **PASS with accepted risk CP-SF-02.** X2: under I8, every tenant change is configuration (avoids CP-SN-03 and CP-SF-03). I1, I3 and I5 answer CP-SF-06. The I9 depth limit meets D11 items 1–4; item 5 is open until Epic 05 sets the value. Owner SA. Revisit when the value is set or a pilot tenant hits the limit. |
| ADR-002 (+E1) | **N/A.** Read-only caller information; E1 is a CI lint. |
| ADR-003 | **PASS.** X2: in-flight workflows keep their version across upgrades (I6). Avoids CP-SF-03 and CP-SN-03. |
| ADR-004 (+A1, E1–E9) | **PASS.** Compliance parity or better: one hash-chained audit log that can be verified offline. Avoids CP-FW-04's top-tier-only audit log, subject to Q1. |
| ADR-005 (+E1, E2) | **PASS.** X2: number formats are declarative configuration. |
| ADR-006 | **PASS with accepted risk CP-FW-01.** The D13 band at ≤ 350 ms, and `422 filter_too_expensive` on `data` filters at volume, were accepted by the CTO (D13, TICK-646). Owner TL. Revisit on ADR-006 R-a (p95 over 350 ms, or a tenant over 400k records). |
| ADR-007 (+A1) | **PASS with accepted risk CP-FW-06.** X1a is cited by mechanism only. Avoids CP-FW-05: one record store and one identity. Per-tenant limits protect other tenants, and no numbers are published yet. Owner TL. Revisit at the first pilot, or if Q1 brings in tier-dependent limits. |

## Consequences

- Every ADR author fills in four lines. Internal ADRs answer `N/A` in one line.
- New tasks, for the Technical Lead's plan:
  - a docs check, CI-1 to CI-4 (RFC-020 §8);
  - the M-X2 upgrade-compatibility suite;
  - the M-X1a timed run in the acceptance kit of the story that ships S11b;
  - the M-X4 cost model.
- The `design-triad` critique checklist should reference this gate. The skill owner updates it; this ADR does not change any skill.

## Risks and residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R1 | The gate decays into boilerplate | TA | Two consecutive ADRs with identical gate text → SA audits the claims |
| R2 | The register goes stale | SA | 6-monthly review, or a vendor announcement |
| R3 | X3 never gets a competitor baseline | SA | Q2 = no, or CG-SP-1 fails → X3 becomes a parity axis |
| R4 | X1a and X1b stay unmeasured | TL | The story that ships S11b; the first pilot |
| R5 | Most sources are grade C | SA | Replace them with A or B sources at each review |
| R6 | A removal from the configuration surface within a major version | SA | M-X2 fails |

## Rejected options

| Option | Why |
|---|---|
| Amend ADR-001 | ADR-001 is the signed record of the invariants. A process gate changes more often, and putting it there would drag ADR-001 through every revision. |
| Checklist in the `design-triad` skill only | Skills live outside the repo, so the checklist could not be cited or reviewed in a PR. |
| One composite "10x score" | It hides which axis moved, and it invites gaming. |
| Admin burden as a fifth axis | No sourced baseline. It stays citable as CP-SN-04. |
| An advisory gate (warnings only) | It would decay into boilerplate. The `N/A` path keeps internal ADRs cheap. |
| A single X1 compared with months of implementation | It compares different quantities (TA B1). |

**Dissent:** none outstanding. TA B1 and B2 were accepted in rev 1 and resolved on re-check.

## Open questions

| # | Question | Who |
|---|---|---|
| Q1 | **Business tradeoff.** Is "1/10 of competitor list price" a goal? What X4 target do we set? Are capabilities ever gated by tier? | Human, via the CTO |
| Q2 | May we time free competitor trial instances (CG-SP-1)? Vendor terms may restrict benchmarking. | Human, via the CTO |

## Competitive Gate (ADR-008)

**Result:** N/A. This ADR changes the design-review process only: no tenant-visible behaviour, API, limit, configuration surface or upgrade path. D12 restates an additive-only rule for the configuration surface; it does not change it. Parity: I1–I9 unchanged.
