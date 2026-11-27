## 2026-11-26: S22 and S29 ship together in two days on one branch; 14 tasks fold into 6, and no p95 gate, spike or bench is in scope

- **Decided by:** the owner (vknavaladi) set the deadline, end of day 2026-11-28. The Technical Lead consolidated the task graph the same day. Stories S22 (TICK-740) and S29 (TICK-915).
- **Problem:** S22 (7 tasks) and S29 (14 tasks) were planned to run one after the other, each on its own story branch. That would not fit two days.
- **Decision:**
  - **One branch.** Both stories build on `TICK-740/search-finish`, which is cut from `TICK-739/filterable-fields`. Every task PR targets it. One PR per task, not one per absorbed issue.
  - **Folded tasks.** TICK-916 also covers TICK-917, TICK-925 and TICK-751. TICK-920 also covers TICK-918 and TICK-921. TICK-919 also covers TICK-922, TICK-923 and TICK-924. TICK-753 also covers TICK-926. TICK-754 also covers TICK-928. The absorbed issues are cancelled as duplicates, and their descriptions and acceptance criteria still apply.
  - **Handbook.** One task, TICK-929, covers both stories. S22 had no handbook task.
  - **Review.** One reviewer and green CI, then merge into the shared branch. The Technical Lead answers design questions within the hour, with no debate round.
- **Cut, and recorded:** no spike, no performance or p95 gate, and no scored bench. TICK-927 (S29 CI-12: 100k records with four live writers, p95 within +25%, WAL, sealer lag and history size) stays in backlog. No ADR amendment in this window.
- **Not cut:** isolation, determinism and idempotency tests, including the integration tests against real PostgreSQL and Temporal that the absorbed issues list, and the Security & Multi-Tenancy Reviewer's adversarial suite (TICK-753).
- **What went wrong (found 2026-11-27 by the Technical Lead on TICK-929):** PR #932 delivered only TICK-919's own scope. It said TICK-922, TICK-923 and TICK-924 were "still open follow-ups", but TICK-919 was set `done` and the three stayed cancelled. S29's filter behaviour (ADR-009 D10 to D12) was therefore on no open task. It is now TICK-1192 (Senior Software Engineer, due 2026-11-28). S29 attack rows 2 and 3 wait on it.
- **Rule adopted with it:** when a task absorbs others, its PR description lists each absorbed issue as delivered or not. A task is not set `done` while an absorbed issue's scope is undelivered and has no open task.
- **Trigger to revisit TICK-927:** the owner prioritises S29's performance numbers, or a tenant reports slow writes during a backfill.
- **Dissent:** none recorded.
