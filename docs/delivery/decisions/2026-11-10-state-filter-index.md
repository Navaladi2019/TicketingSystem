## 2026-11-10: the `filter[state]=new` miss reopens the list-shape work as one index task
- **Decided by:** the CTO on TICK-336 (2026-11-10). Recorded by the Technical Lead. Dissent: none.
- **Numbers (P3-36 final re-run, PR #544):** agent `filter[state]=new` p95 192.6 ms against 150. In its plans, `state` is only a `Filter` on `records_list_group`, and about 20k rows are read and discarded per statement.
- **Decision:**
  - The group access path gets a composite B-tree index with `state` right after `assignment_group_id`, then the default sort keys. It is built `CONCURRENTLY`.
  - It is not an expression index and not a partial index on "open" states, because what counts as open depends on each tenant's workflow.
  - `authz.CompileRead` is unchanged (I5).
  - This is not D25, and the 150 ms target stays. Task: TICK-656 (P3-36d).
- **Numbering:** D-3-9 (gaps by default, per-type gapless, no reclaim) is recorded separately in [2026-11-10-d-3-9-numbers-may-have-gaps.md](2026-11-10-d-3-9-numbers-may-have-gaps.md).
- **Reopens:** the P3-36 re-run on TICK-621 after TICK-656 merges, on the same harness and host. If `state=new` is still over 150 ms, the numbers go back to the CTO.
