---
type: adr-erratum
id: ADR-004-E6
title: "Erratum E6 to ADR-004-E4 (accepted risk E4-R1): the uniform-regression cue is a per-model 1.2x band around the recorded baseline, not the absolute review threshold"
status: PROPOSED
date: 2026-11-05
issue: TICK-526
rfc: docs/adr/ADR-004/spikes/technical-architect/e4-r1-cpu-variance.md (spike note and proposed trigger, archived; no separate RFC, see "Process")
supersedes: "ADR-004-E4 E4-R1: the sentences 'A `warn` on `main` from a CPU model that a baseline in `docs/budgets.md` already records is the cue to look', 'an over-threshold baseline goes to the Technical Lead as a possible regression', and the trigger 'two `warn` runs on `main` in a row on a CPU model already recorded in a baseline, or an over-threshold baseline'"
amends: "[[ADR-004-E4-commit-check-ceilings-report-only]] Accepted risks, E4-R1 only; `docs/security/accepted-risks.md` AR-TICK-517-01 trigger; `docs/budgets.md` commit-check import baseline for AMD EPYC 7763 (annotation)"
source: "TICK-526 spike (ADR-004-E4 follow-up 6), Technical Architect, 2026-11-05, comment 01a1aa17-02f1"
deciders: Solution Architect (author, verdict); Technical Architect (spike and proposed wording; critique pending); CTO (tie-break if contested)
---

# ADR-004 erratum E6: the E4-R1 cue compares a model with its own baseline

> **Status: PROPOSED, 2026-11-05** (TICK-526). Awaiting the Technical Architect's critique. Nothing in the CI gate changes: no verdict, threshold, ratio limit or code. Only the wording of one accepted-risk trigger changes, plus one baseline annotation.
>
> **Numbering.** The spike comment called this "an ADR-004-E4 erratum". It amends E4, but `E1`–`E5` are taken, so it is the next ADR-004 erratum: **E6**.
>
> **Invariants source.** The project description is empty and defines no invariants. As in ADR-004 and E1–E5, this erratum is checked against [[ADR-001-platform-blueprint]] §1 I1–I9. That check is incomplete against any separate list of eight.

## Context

ADR-004-E4 made the absolute commit-check ceilings (300 µs per call, 175 µs per import probe) report-only in CI. It accepted E4-R1: a regression that slows the clause arm and the same-run reference arm equally keeps the ratios in range and passes. The cue to look was "two `warn` runs in a row on `main` on a CPU model already recorded in a baseline". E4 follow-up 6 asked for a spike to check that one CPU model gives stable timings.

The spike (TICK-526) **passed**: on every model with at least 3 runs, max/min < 1.2 on every row (worst 1.127, otherwise ≤ 1.07; 16 runs, 5 models). It also showed that the cue is wrong for a reason the variance test does not cover. `warn` is an absolute threshold, so it does not say whether one model regressed:

| Model | Recorded baseline (N = 200 / import, µs) | Effect of the E4 cue |
|---|---|---|
| AMD EPYC 7763 | 235.7 / 194.9 | The import baseline is already over 175 µs. 7 of 7 runs were `warn`. The cue fires with no regression behind it, and 7763 drew 7 of 16 runners |
| AMD EPYC 9V45 | 126.1 / 80.3 | A uniform slowdown must reach about 2.2× (import, 175 / 80.3) or 2.4× (per call, 300 / 126.1) before any `warn`. The cue is blind below that |

So the cue was always on for one model and blind on another.

## Decision

1. **Cue.** A run's row is *off-baseline* when the run's CPU model has a recorded baseline in `docs/budgets.md` and the row is more than **1.2×** that model's recorded value. The rows are clause per call at N = 200, 1,000 and 5,000, and import per probe. `warn` (over the absolute review threshold) is no longer the cue. It stays exactly as E4 defines it: an annotation, and the job still passes.
2. **Trigger to revisit E4-R1 (AR-TICK-517-01).** Either of these:
   - two `main` runs in a row on a recorded model, each with any off-baseline row; or
   - a baseline re-take for a model whose new value is more than 1.2× that model's previous baseline on any row.

   "In a row" counts runs on recorded models only. A run on a model with no baseline neither counts nor breaks the streak.
3. **An over-threshold baseline alone is not a regression signal.** E4 sent "an over-threshold baseline" to the Technical Lead as a possible regression. That sentence is withdrawn. A baseline over the review threshold on a slow host is recorded as *slow host* when its same-run ratios are in line with the other models' (within the E4 ratio limits). It needs no escalation. The EPYC 7763 import baseline (194.9 µs, ratio 8.58, while the other models read 8.13–9.75) is annotated that way in `docs/budgets.md` in this change.
4. **Who looks, and how.** Unchanged from E4: the check is manual. The Postgres & Data Modeling Specialist owns it, and nothing notifies anyone automatically. A reader compares the logged `COMMIT-CHECK-BUDGET` line with the model's baseline row. Automating the comparison is the optional follow-up 3. It is not a condition of this erratum.

### Why 1.2× and two in a row

- Measured within-model noise is ≤ 1.13× (one N = 200 outlier, 258.0 µs on 7763), otherwise ≤ 1.07×. The recorded baselines sit inside the spike's per-model ranges or within 1% of them, across days and SHAs. Against the 7763 baseline the outlier is 1.095×, under the band.
- A 1.2× band catches a uniform regression of 1.2× or more on every recorded model. The E4 cue caught nothing below 2.2× on 9V45 and fired constantly on 7763.
- One noisy run cannot trigger it alone, because the trigger needs two in a row. The margin between the band and the worst noise is small (1.2 vs 1.13). That is accepted: the cost of a false trigger is one look by the owner, not a red build.

### Options considered

| Option | Verdict | Why |
|---|---|---|
| A. Keep the E4 cue (two `warn` in a row) | Rejected | Always on for 7763, so people learn to ignore it. Blind below about 2.2× on 9V45 |
| **B. Per-model 1.2× band around the recorded baseline, two in a row (TA proposal, with Decision 2's re-take clause tightened)** | **Chosen** | Matches the measured noise, needs no code, catches ≥ 1.2× on every recorded model |
| C. Warn rate over a window (from E4 follow-up 6) | Rejected | Still keyed to the absolute threshold, so it inherits A's two faults |
| D. Make the band a CI gate now (fail or annotate automatically) | Deferred | Needs machine-readable baselines in the script, and a gate on 1.2× vs 1.13× noise would go red on hosted runners. Kept as optional follow-up 3 (annotation only) |
| E. TA wording as written: "...or an over-threshold baseline re-take" | Amended | Every 7763 import re-take is over the threshold, so that clause would keep A's always-on fault. Decision 2 keys the re-take clause to the model's previous baseline instead |

## Accepted risks

- **E4-R1 (AR-TICK-517-01), restated.** A uniform regression below 1.2× on a recorded model, or any uniform regression that shows only on models with no baseline, passes the gate unseen. On a recorded model, one of 1.2× or more is seen only when the owner reads the logs, because nothing notifies anyone automatically. Severity stays **Low**. Owner: Postgres & Data Modeling Specialist. Trigger: Decision 2.
- **E6-R1: the evidence is one SHA within about 70 minutes.** Day-to-day drift (runner image updates, noisy neighbours) is only partly checked, against the 2026-11-03/04 baselines (inside or within 1% of the spike ranges). If drift is larger, the band yields false triggers. That costs a look, not a red build. Trigger to revisit: two false triggers (no regression found) within 30 days. Then re-run the spike across days and widen the band, or base it on the median of the model's last N runs. Owner: Postgres & Data Modeling Specialist. Low. Indexed under AR-TICK-517-01 rather than as a new register row, because it is a property of that risk's trigger.

## Invariant check (ADR-001 I1–I9)

| Invariant | Effect |
|---|---|
| I1–I9 | None. No SQL, grant, policy, role, pipeline or code change. Only an accepted-risk trigger is reworded and one baseline is annotated |

## Evidence

- `docs/adr/ADR-004/spikes/technical-architect/e4-r1-cpu-variance.md`: the TICK-526 spike note, with raw `COMMIT-CHECK-BUDGET` lines and job links for 16 runs of CI run [41964296547](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41964296547), max/min per model and row, the cross-day check and the rerun script.
- `docs/budgets.md`: the 2026-11-03/04 baselines per model.

## Process

There is no separate RFC. The spike note frames the problem and proposes the trigger, so it is archived as the RFC. The Solution Architect took the proposal with one change (option E) and wrote this text. Next: the Technical Architect critiques it in one round. If it is contested, the CTO breaks the tie. Then the Solution Architect gives the verdict.

## Follow-up

1. **Done in this change:** the AR-TICK-517-01 trigger is reworded in `docs/security/accepted-risks.md`, the EPYC 7763 import baseline is annotated as slow host in `docs/budgets.md`, and the spike note is archived (this also covers the TA's ask 1 on TICK-526).
2. **`docs/budgets.md` (Postgres specialist, optional):** add baselines for `INTEL(R) XEON(R) PLATINUM 8573C` (3 spike runs) and relabel "CPU not logged" as `AMD EPYC 9V74` once a third 9V74 run confirms it. More recorded models mean more runs count toward the trigger.
3. **Optional, not planned:** have `scripts/ci/commit-check-budget.sh` print each row's ratio to the model's baseline in `$GITHUB_STEP_SUMMARY`, from a machine-readable baseline file. Annotation only. Never a fail condition (option D).
4. **No plan issue is needed.** No code changes. The CTO can confirm that no `Plan ADR-004-E6` is required.
