## 2026-11-27: TICK-1154, vulncheck required on `main`, ADR-001-E4 FU2

- **Decided by:** ADR-001-E4 FU2 (ACCEPTED); applied by the Technical Lead on TICK-1154 (T-E4-7) after the TICK-1153 preflight passed (PR #946).
- **Change:** branch protection of `main` gains the required check `vulncheck` (GitHub Actions, app id 15368), next to the nine earlier ones. `strict` and `enforce_admins` stay true; nothing else changed. Before/after API output: `docs/adr/ADR-001/E4/fu2-applied.txt`.
- **Effect:** GitHub blocks a pull request into `main` whose `vulncheck` is red or missing. The E4-3 interim rule for the scan is now enforced by GitHub.
- **Still in force:** the TICK-1146 rule to read every check run on the head commit ([merge safety, section 3](../merge-safety.md#3-review-standard-merging-to-main-and-to-story-branches)), until TICK-1146's fix is merged.
- **Undo:** the same `PATCH` with the before `checks` (in `fu2-applied.txt`). Use it if `vulncheck` stays Pending on a pull request that cannot get it; a pull request opened before TICK-1150 merged needs one push, which `strict` already forces.
- **Dissent:** none.
