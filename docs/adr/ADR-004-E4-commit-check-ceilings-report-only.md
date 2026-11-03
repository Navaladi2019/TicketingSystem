---
type: adr-erratum
id: ADR-004-E4
title: "Erratum E4 to ADR-004-E3 (budget): in CI the absolute commit-check ceilings are report-only; the job fails only on a same-run ratio, on flatness, or when nothing was measured"
status: PROPOSED
date: 2026-11-03
issue: TICK-517
rfc: docs/adr/ADR-004/rfc/TICK-517-escalation-commit-check-host-variance.md (escalation and CTO decision, archived; no separate RFC, see "Process")
supersedes: "ADR-004-E3 'Budget': the sentence 'An over-ceiling result with an in-range ratio is a runner problem: re-run it, and do not fire S3', and the claim that the CI job uses 'the same runner class every time'"
amends: "[[ADR-004-E3-commit-check-planned-per-call]] Budget section; `docs/budgets.md` commit-check rows (verdict column); AR-TICK-502-01; `docs/cto/risks.md` R-005 trigger wording"
source: "TICK-509 / PR #420 review (docs/reviews/tick-509-commit-check-budget-ci.md, finding 1): on one SHA the import row read 129.5 µs and 190.9 µs on two GitHub-hosted runners, against a 175 µs ceiling, with the same-run ratios unchanged"
deciders: CTO (decision, Option B with four conditions, TICK-517 comment 01a1a258-11d7); Technical Lead (escalation and recommendation, TICK-517); Solution Architect (author of this text); Technical Architect (critique, pending)
---

# ADR-004 erratum E4: absolute commit-check ceilings are report-only in CI

> **Status: PROPOSED, 2026-11-03.** The CTO decided Option B on TICK-517, with four conditions. This erratum is the Solution Architect's wording of that decision. It waits for the Technical Architect's critique and then the Solution Architect's verdict. TICK-509 / PR #420 stays held until this is ACCEPTED.
>
> **Numbering.** The CTO's routing called this "the ADR-004-E3 erratum". `ADR-004-E3` is already taken by the accepted erratum it corrects, so this is the next ADR-004 erratum: **E4**. It amends E3's Budget section only.
>
> **Invariants source.** The project description is empty and defines no invariants. As in ADR-004 and E1–E3, this erratum is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. That check is incomplete against any separate list of eight.

## Decision

The `commit-check-budget` CI job keeps measuring both budget rows exactly as ADR-004-E3 defines them. Only the **verdict rule** changes. Method, fixture, partition count (38), ceilings, ratio limits and flatness limit all stay the same.

1. **Absolute ceilings are report-only in CI.** If a row goes over its ceiling (300 µs per call, 175 µs per probe), the job does not fail on that alone. It emits a GitHub warning annotation and stays green. The ceilings become *review thresholds*. Their values do not change.
2. **The job fails on any of these, whatever the absolute numbers are:**
   - **No rows.** Either greppable line (`COMMIT-CHECK-BUDGET row=single_subject` or `row=import`) is missing from the output, or a measured value in it is zero, negative, NaN or missing. This is the TICK-509 finding 2 fix, and it stays a hard fail.
   - **Ratio.** A same-run ratio is out of range: per call at N = 200, clause / reference > 5, or import clause / reference > 15. A NaN ratio counts as out of range.
   - **Flatness.** For the clause arm per call, max/min ≥ 1.5 across N = 200, 1,000 and 5,000.
3. **Every run logs its numbers and the host CPU.** Each `COMMIT-CHECK-BUDGET` line carries `cpu="<model name>"` from `/proc/cpuinfo`, and the "cluster healthy" line does too. The job writes both rows to `$GITHUB_STEP_SUMMARY`. `docs/budgets.md` keeps the measured baselines, now with the CPU model of each baseline run.
4. **The `re-run` verdict goes.** E3 said that an over-ceiling result with in-range ratios meant "re-run". Now that case is `warn`, and the job passes. Nobody re-runs a green job to clear a warning.

### Verdict table (per run)

| Condition (first match wins; all failing conditions are logged) | Verdict | Job | `main-red` |
|---|---|---|---|
| A required line is missing, or a measured value is ≤ 0, NaN or missing | `fail_no_rows` | **fail** | opens P0 |
| A same-run ratio is out of range (per call N = 200 > 5, import > 15, or NaN) | `fail_ratio` | **fail** | opens P0 |
| Clause per call max/min ≥ 1.5 | `fail_flat` | **fail** | opens P0 |
| A row is over its review threshold, the ratios are in range, and it is flat | `warn` | pass, with a `::warning::` annotation per row over its threshold | no |
| Otherwise | `pass` | pass | no |

The rule is the same for local runs (`task test-commit-check-budget`). The constants in `internal/db/dbtest/commit_check_verdict_test.go` keep their values. `budgetPerCallCeiling` and `budgetImportCeiling` stay as the warning thresholds.

### What a failure means

- `fail_no_rows`: the gate measured nothing. That is a test or CI defect. Fix it, and do not re-run it until it goes green.
- `fail_ratio` or `fail_flat`: stop and tell the Technical Lead (the TICK-460 rule). These checks compare two arms in the same run on the same host, so a slow host does not trip them.
- **S3 trigger (E3-R2, R-005).** It fires when a run at 38 partitions is `fail_ratio` **and** the clause arm in that same run is over its review threshold. That is E3's trigger, unchanged, read from one run. If a run is `fail_ratio` and the absolutes are under the thresholds, the job is still red, but S3 does not fire on its own. The Technical Lead and the Postgres specialist first check whether the reference arm moved (for example after a Postgres bump) or the clause arm got worse.

## Context

E3 put both rows in a dedicated CI job, gave each a same-run reference, and assumed "the same runner class every time". The label is fixed (`ubuntu-latest`, 4 vCPU). The host behind it is not. On one SHA (`25ca403`), with nothing else on either runner:

| Run | Clause per call N = 200 / 1k / 5k (max/min) | Ref N = 200 | Ratio N = 200 | Import clause / ref | Ratio import | E3 verdict |
|---|---|---|---|---|---|---|
| 41676703902 | 170.4 / 170.6 / 165.1 (1.03) | 117.5 | 1.45 | 129.5 / 13.3 | 9.76 | pass |
| 41676713796 | 233.2 / 234.6 / 228.8 (1.03) | 151.4 | 1.54 | **190.9** / 19.1 | 9.98 | **rerun** (red) |
| 41673342312 (`2073e3c`) | 170.5 / 163.6 / 164.8 (1.04) | 133.5 | 1.28 | 127.7 / 13.1 | 9.75 | pass |

Both arms ran about 1.4× slower on the second host, and the ratios moved by 2–6 %. So the absolute numbers mostly tell you which host you got, and the regression signal is in the ratios. The import row's headroom is 175 / 129.5 = 1.35×, which is less than the observed host spread of about 1.44×. On the slow host the per-call row has 300 / 234.6 = 1.28× headroom. The job is required and sits in `main-red`'s `needs`. So under E3, a slow host opens a P0 "main is red" issue when nothing is wrong.

Headroom of the checks that still fail, against what was measured in CI: import ratio 15 / 9.98 = 1.50×; per-call ratio 5 / 1.54 = 3.2×; flatness 1.5 / 1.04 = 1.44×. All three are within-run comparisons. They varied by ≤ 6 % across the two hosts.

## Options considered

| Option | What changes | Verdict |
|---|---|---|
| **B. Absolute ceilings report-only in CI; fail on ratio, flatness or no rows (chosen)** | The verdict rule only | **Chosen (CTO).** No cost, no vendor. Host variance cannot redden `main`, and a regression in the signal that matters still fails |
| A. Pin a fixed larger-runner class | The workflow `runs-on`; the rule stays | Rejected (CTO). It adds paid minutes and a vendor dependency to protect a number that does not carry the signal. A paid class still does not promise one CPU model |
| Raise the ceilings to cover the host spread | The ceilings | Rejected. E3 and D-E3-6 forbid moving a ceiling to make CI pass. A ceiling wide enough for the host spread (≥ 1.45× the fast host) no longer catches a real regression of the same size |
| Keep E3's rule and re-run on red | Nothing | Rejected. It flaps on about one run in three. A required gate that has to be re-run is ignored, and every red run opens a false P0 |
| Normalise the absolutes by a CPU benchmark run in the same job | A calibration step plus a derived number | Rejected for M1. The same-run reference arm already does this calibration, for the workload that matters. A synthetic benchmark adds its own noise |

## Accepted risks

Indexed in `docs/security/accepted-risks.md` as **AR-TICK-517-01** (E4-R1) and **AR-TICK-517-02** (E4-R2), in this change, marked Proposed until the verdict.

- **E4-R1: a uniform regression passes the gate.** A change that slows the clause arm and the reference arm by the same factor keeps the ratios in range, so the job stays green. Shared code is the likely cause, for example a slower audit insert path, a Postgres bump or a fixture change. It stays visible: the run logs the absolute numbers with the CPU model, and annotates any row over its threshold. Nothing notifies anyone automatically. A `warn` on `main` from a CPU model that a baseline in `docs/budgets.md` already records is the cue to look. The Postgres specialist re-takes the baselines on each Postgres bump and whenever the method changes, and an over-threshold baseline goes to the Technical Lead as a possible regression. Owner: Postgres & Data Modeling Specialist. Trigger to revisit: two `warn` runs on `main` in a row on a CPU model already recorded in a baseline, or an over-threshold baseline.
- **E4-R2: the ratio gate fails more often than before.** Under E3, an out-of-range ratio failed only when a row was also over its ceiling. Now it fails alone. If the reference arm gets faster (the shipped copy improves after a Postgres bump), the ratio rises even when the clause arm did not get worse. That produces a red run that is not a regression. It is still a real change in what the two arms measure, so it is worth a look. The import ratio has 1.50× headroom over the highest CI value, and across hosts it moved ≤ 6 %. Laptop measurements under E3 ranged from 9.5 to 13.1, but those were not CI and are provenance only.

AR-TICK-502-01's trigger wording ("either budget row fails in CI **and** the ratio is out of range") reads correctly under E4 if "fails" means "is over its review threshold". The follow-up below changes the text so that it says so.

## Invariant check (ADR-001 I1–I9)

| Invariant | Effect |
|---|---|
| I1 tenant wall, I3 least privilege | None. No SQL, grant, policy or role changes |
| I4 one write pipeline, audit in the same transaction | None. The function, the clause and the probe stay the same, and so does E3's guarantee. Only how the CI gate reads its measurements changes |
| I2, I5–I9 | Not touched |

No verified library behaviour or spike result is overturned. The measurements stand. This erratum changes which of them fail the job.

## Evidence

- `docs/adr/ADR-004/spikes/ci/tick-509-host-variance.md`: the raw `COMMIT-CHECK-BUDGET` lines from CI runs 41676703902, 41676713796 and 41673342312, with the derived spread and headroom.
- `docs/reviews/tick-509-commit-check-budget-ci.md` on PR #420 (branch `TICK-509/commit-check-budget-ci`): the Quality, Testing & CI Advisor's review, findings 1–3.
- `docs/adr/ADR-004/rfc/TICK-517-escalation-commit-check-host-variance.md`: the escalation, the options and the CTO's decision, archived.

## Process

There is no separate RFC. The escalation on TICK-517 framed the options, the Technical Lead recommended B, and the CTO decided it with four conditions, so the RFC step has already happened. This text maps those conditions to the Decision: condition 1 → Decision 1 and 3; condition 2 → Decision 2 "No rows"; condition 3 → Decision 3; condition 4 → E4-R1. The Technical Architect critiques the wording (one round), and then the Solution Architect gives the verdict.

## Follow-up (once this is ACCEPTED)

1. **TICK-509 / PR #420 (Technical Lead assigns):** add the five-verdict rule to `commit_check_verdict_test.go`, with unit cases for each row of the verdict table. These cases must be covered: over a threshold with in-range ratios gives `warn` (today this case is `rerun`); in-range absolutes with a ratio over its limit give `fail_ratio`; a zero reference gives `fail_no_rows`; ratio and flatness both at the limit exactly. In `scripts/ci/commit-check-budget.sh`, grep both lines and fail if either is missing (finding 2), log the CPU model (finding 3), emit `::warning::` for each `warn` row, and write the rows to `$GITHUB_STEP_SUMMARY`. The red → green evidence is the unit cases.
2. **`docs/budgets.md` (Postgres specialist, on the PR #420 branch):** rename the ceiling column for the two rows to "review threshold (report-only in CI, ADR-004-E4)". The rows state the fail conditions (no rows, ratio, flatness). Record the CPU model with each baseline.
3. **Risk registers (Solution Architect, at the verdict; CTO for R-005):** set AR-TICK-517-01/02 to Accepted. Reword the AR-TICK-502-01 and `docs/cto/risks.md` R-005 triggers to say "over its review threshold **and** the same-run ratio is out of range, in the same run".
4. **No plan issue is needed.** This is one change to the verdict rule inside TICK-509, which is already planned. The CTO can confirm that it folds into `Plan ADR-004-E3` (PLAN-005-E3) rather than creating `Plan ADR-004-E4`.
