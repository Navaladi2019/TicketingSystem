## 2026-11-26: S21 ships without the 350 ms p95 for the ≥ 40% band under a row ACL; the target moves to S21c (`AR-S21-01`)

- **Decided by:** the Technical Lead, approved by the owner (vknavaladi) in chat on 2026-11-26. Story S21 (TICK-739).
- **Problem:** S21 was planned as 9 tasks and grew to 44. Criteria 1, 3, 4 and 5 and most of criterion 2 were met by about 2026-11-19. The one cell left, a filter that matches 40% or more of a large type under a narrowing row ACL, measured 578–983 ms p95 against 350 ms (TICK-1022). Since then it has driven ADR-006-A1, A2 and E2, three merge-main re-stamps and a scored run that waits for a quiet host (TICK-1098). S12, S19, S22, S23, S24 and S26 all wait for S21 to reach `main`.
- **Decision:** S21 criterion 2 keeps the right rows, every other p95 cell, and for this cell the correct rows, isolation and the A2 count budget (`total_state`). The kit records this cell's p95 but does not assert it. The 350 ms target moves to Story S21c (TICK-1159), with TICK-1097 (PR #861), TICK-1098, TICK-1099, TICK-1095 and TICK-1002, all in backlog. The ADR text is unchanged.
- **Not cut:** isolation, determinism, idempotency and correctness of the list.
- **Owner:** the Technical Lead.
- **Trigger to revisit:** a tenant reports slow filtered lists, `RecordsListTotalUnknown` fires in production, or the owner prioritises S21c.
- **Delivery rules adopted with it:** a timing target is a budget, not an acceptance gate; at most one design change per story, after which the Technical Lead cuts scope; story branches live no longer than about 5 days; a story whose acceptance depends on a timing number on large data starts with a feasibility spike; a story's long tail does not block the next stories.
- **Dissent:** none recorded. Earlier rounds held the cell as a gate under ADR-006-A1, A2 and E2.
