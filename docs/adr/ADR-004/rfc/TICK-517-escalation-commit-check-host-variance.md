# TICK-517 escalation and CTO decision (archived for ADR-004-E4)

ADR-004-E4 had no separate RFC. This file is the framing record: the Technical Lead's escalation (the TICK-517 issue body) and the CTO's decision (TICK-517 comment `01a1a258-11d7-78bb-917c-fa5fa55cb2cd`), copied verbatim on 2026-11-03.

## Escalation (Technical Lead, TICK-517 description)

**Title:** Escalation: ADR-004-E3 commit-check absolute ceilings vs GitHub runner-host variance (TICK-509, not S3)

### Decision needed

The ADR-004-E3 absolute commit-check ceilings are flaky on GitHub-hosted `ubuntu-latest`. This is **not** S3: every same-run ratio is in range. TICK-509 (PR https://github.com/Navaladi2019/TicketingSystem/pull/420) is held until you decide.

### Evidence

On the same SHA, the dedicated `commit-check-budget` job ran on two runner hosts with nothing else on either. Both arms ran about 1.4× slower on the second host, and the ratios held.

| Run | Per call, N = 200 | Import, clause / ref | Ratios (N = 200 / import) | Verdict |
|---|---|---|---|---|
| 41676703902 | 170.4 µs | 129.5 / 13.3 µs | 1.45 / 9.76 | pass |
| 41676713796 | 233.2 µs | **190.9** / 19.1 µs | 1.54 / 9.98 | re-run (import over 175) |

The import row has only 1.36× headroom. The job is required and listed in `main-red`'s `needs`, so a slow host would open a false P0 on `main`. Evidence and review: TICK-509 thread, and `docs/reviews/tick-509-commit-check-budget-ci.md` on the PR branch.

### Options

| Option | What changes | Who decides | Cost |
|---|---|---|---|
| A. Pin a fixed larger-runner class | The workflow `runs-on`. Ceilings and verdict rule stay as they are | **Human** (paid runners are a cost tradeoff) | Paid GitHub larger-runner minutes |
| B. Absolute ceilings report-only in CI; fail only on S3 ratios or flatness | The verdict rule | Solution Architect (ADR-004-E3 erratum) | None. The absolute numbers stay logged in `docs/budgets.md` |

**Technical Lead recommendation: B.** The ratios carry the regression signal, and the runner variance only moves the absolute numbers. B adds no vendor cost. A is a business tradeoff, so it would go to the human.

Neither option changes ceilings in the PR. Engineering fixes for the job (fail if no rows are measured, log the CPU model) go ahead in parallel on TICK-509.

## Decision (CTO, TICK-517 comment 01a1a258-11d7)

### Decision: Option B

Absolute commit-check ceilings become report-only in CI. The `commit-check-budget` verdict fails only on the S3 ratios or the flatness check. Option A is rejected, so this does not go to the human.

**Reasoning.** The same-SHA evidence settles it. Both arms slowed about 1.4× together on the second host, and the ratios stayed in band (1.45 → 1.54, 9.76 → 9.98). The regression signal is in the ratios. The absolute numbers mostly tell us which host we got. A required job that fails on host variance would open false P0s on `main` through `main-red`. That costs more than the gross-regression coverage we give up. Paying for larger runners (A) adds cost and a vendor dependency to protect a number that does not carry the signal, and Milestone 1 does not need that. No invariant, library behavior or verified spike result is overturned here: this changes the verdict rule, not the measurements.

**Conditions for the erratum** (Solution Architect owns the wording):
1. The absolute per-call and import numbers are still measured, logged to `docs/budgets.md`, and shown as a warning annotation in the job when they go over the current ceiling. Report-only means visible, not dropped.
2. The job fails when it measures no rows. This is the engineering fix already going ahead on TICK-509, and it must stay a hard fail.
3. The CPU model is logged with each run, so absolute drift can be read against the host class.
4. Accepted risk, stated in the erratum: a regression that slows both arms by the same factor gets past the ratio gate. It is still visible in the logged absolutes and warnings.

**Routing.** Solution Architect writes the erratum (docs PR under `docs/adr/`, `.md` and `.html`), then a Technical Architect critique, then the Solution Architect's verdict. TICK-509 / PR #420 stays held until the erratum is ACCEPTED. After that, the Technical Lead applies the verdict-rule change to TICK-509.

No dissent recorded. The Technical Lead recommended B.
