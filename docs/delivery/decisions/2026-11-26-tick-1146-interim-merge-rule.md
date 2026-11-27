## 2026-11-26: TICK-1146, until ADR-001-E4 lands, a merge into `main` or a story branch is checked against each required job's last executed result, not the merge button

- **Decided by:** the CTO on TICK-1146 (2026-11-26 00:00 UTC), on the Quality, Testing & CI Advisor's spike (PR #898, closed, not merged). Recorded and enforced by the Technical Lead.
- **Why:** GitHub takes each required check from the newest run on the commit. A label run skips every job, and skipped counts as passed, so adding a label (`MergePr`, `question`, ...) or `ci:run` after a red full run turns the PR `CLEAN` (spike cases b, d, e, f). `gh pr checks` and `statusCheckRollup` showed the same false picture. Report: `docs/quality/label-run-hides-red-check.md` (PR #902).
- **Rule:** see [merge safety, section 3, "Interim rule: last executed result"](../merge-safety.md#3-review-standard-merging-to-main-and-to-story-branches). It binds every merger the Technical Lead names, on every PR into `main` or a `TICK-*` story branch.
- **Ends:** for `isolation`, when PR #902 merges with the two `ci.yml` edits (named job, `isolation` on every labelled run). For the other required jobs (`lint`, `unit`, `integration`, `stack`, `coverage`, ...), when ADR-001-E4 (PR #895, TICK-1140) lands a gate that reads the last executed result. **Owner of the removal:** the Technical Lead.
- **Not in this fix:** required checks on story branches (ruleset "Mina branch" has none) go to ADR-001-E4.
- **Dissent:** none.
