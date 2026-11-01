## 2026-11-01: agents accept and merge stories and ADRs; `MergePr` retired
- **Decided by:** vknavaladi (workspace merge and close policy), recorded by the Technical Lead on TICK-402.
- **Decision:**
  - **Story → `main` PRs:** the Technical Lead verifies the acceptance kit end-to-end on a fresh stack, posts `VERDICT: ACCEPTED` on the story, approves and merges the story PR (CI green, whole-story diff reviewed) and sets the story `done`. On failure: `VERDICT: REJECTED` and one fix task per failure.
  - **ADRs, amendments and errata:** the Solution Architect gives the `ACCEPTED` verdict after the Technical Architect critique and any CTO tie-break, sets the status in the `.md` and `.html`, and merges the design PR. The CTO then creates `Plan ADR-NNN`. The `.md` + self-contained `.html` rule is unchanged.
  - **Tasks and subtasks:** unchanged from TICK-243.
  - The `MergePr` label is no longer used. Re-sequencing questions go to the CTO, not the owner.
  - **Still escalated to the human:** a tenant-isolation gap, and a business tradeoff (cost, new vendor, timeline vs correctness). Repository settings changes remain the owner's call.
- **Supersedes:** the story and ADR bullets of the 2026-10-26 TICK-243 entry, and "the owner merges story → `main` PRs and ADRs" in the 2026-10-26 TICK-231 entry. Updated: `docs/delivery/branching-strategy.md`, `docs/architecture/skills/design-triad/SKILL.md`, `docs/architecture/skills/design-docs/SKILL.md`.
- **Dissent:** none.
