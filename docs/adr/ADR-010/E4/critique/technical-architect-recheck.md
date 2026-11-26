# Technical Architect re-check of ADR-010-E4 rev 1 (TICK-1118 comment 01a210f7, 2026-11-25, at `5aa3adbf`)

**B1 to B3 are resolved in rev 1.** I re-checked only those, at PR #870 head `5aa3adbf`.

| Item | Verdict | Where |
|---|---|---|
| B1 locks of `org_unit_closure_rebuild_all()` | **confirmed sound** | E4-2 (c) states the three ACCESS EXCLUSIVE locks, `40P01` and the maintenance window. The `23505` inference is gone, and the Evidence row matches my runs. The explainer and the handbook page agree. |
| B2 `not under` | **confirmed sound** | Both directions are in E4's risk row, the register and the explainer's Risks. |
| B3 interval | **confirmed sound** | No "10 minutes" is left in E4, its explainer or its handbook page. |

**Severity:** Security's rating stands over my "Low". I judged writers and tenant wall, not the register's scale.

**Wording, optional, no re-check needed:**
- Evidence: "A drift repair and the function wait for each other" reads like a deadlock. Suggest "whichever starts second waits".
- "which the Technical Architect's spike covers (below)": that row is above.

**CI:** `runner` and `story-branches` were pending on `5aa3adbf` when I looked. I have not read a result for this head.

VERDICT: APPROVE

RESULT: DONE
