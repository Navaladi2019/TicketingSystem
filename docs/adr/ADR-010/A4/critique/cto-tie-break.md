<!-- TICK-1143 comment 01a2154d-537e-7cc3-84a2-8061f0818e32, 2026-11-26T00:41:58Z. Archived verbatim; agent mentions removed. -->

**Decision (CTO tie-break, round 2 closed): adopt rev 1, the covering index without the order, with the five rulings below. No third round.** All three re-checks are in. Security approves. The Technical Architect (B2) and the Postgres & Data Modeling Specialist (guard, P3) left items that each name their own fix, so I rule on them under rung 4 rather than reopening the debate.

| Open item | Ruling |
| --- | --- |
| **B2**, cold page in `main` before A2-4 | Add the Technical Architect's A4-R3 sentence word for word; that closes B2. The merge order relative to S21 stays a Technical Lead decision at plan time (rung 3). If A4 lands first, the plan says so and cites this sentence. |
| **B1**, D4 fallback scope | Add the Technical Architect's D4 sentence: if D4 ever applies, the order goes only on counts with at most one R arm. |
| **P1**, the 1.3× guard | Take the specialist's fix: compare only the `records` scan node's run cost (total minus startup), not the whole plan. The ratio stays 1.3×; the specialist measured 5.2× on the CI-4 world. |
| **Pin 2** (Security and Postgres agree) | Replace "Rows Removed by Filter holds no other tenant's rows" with Security's wording: the count's `Index Cond` starts with `tenant_id`, and rows read ≤ A's rows of the type. For the mutant, rows read > A's rows. |
| **P3**, a negated R arm is not an IOS | **Option 3: record it as a residual with a trigger.** Do not order those arms now, and do not widen the index. CI-5 pins the un-negated R arm as an IOS, and the negated arm only as "no worse than today's plan class". |

**Why option 3.** A negated R arm plans the same way under A4 as it does today, because neither index holds `requester_id`. A4 makes no cell worse there. Option 1 brings back B2's 9 to 11 s cold walk on exactly those counts, which is the cost rev 1 exists to remove. Option 2 grows the index by one column per policy branch and still cannot cover a data filter (Security's default-value cell). That is more moving parts and an incomplete fix. A residual with a measured trigger is the simplest answer that is honest about the gap. Trigger: CI-5, TICK-1105 or a production slow-query report shows a negated R arm above the 150 ms budget, or reading more other-tenant rows than N̂_e. When it fires, the order on that arm (D4) is the fix already decided.

**AR-TICK-1143-01:** use Security's final text, plus one clause in the "it remains" list: "in an R arm that follows another branch of the policy, whose negation names a column the index does not hold". Security owns the row and confirms that clause on the ADR PR. It is a text check, not a debate round.

No reviewer dissents from adopting the index. The specialist left the P3 choice open and named all three options; it is recorded as decided by the CTO.

Solution Architect: Write ADR-010-A4, status PROPOSED, from rev 1 with the rulings above folded in, and archive RFC-023 (rev 0 and rev 1) and the three spike zips under `docs/adr/ADR-010/`. Ship `.md` and the self-contained `.html` explainer on one design PR, then report the PR link here.
