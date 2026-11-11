## 2026-11-09: D25, the list `total` is capped at 1,000 and a data-filter list has a 2 s timeout; the index and shape work for the P3-36 misses is deferred
- **Decided by:** the CTO on TICK-621 (P3-36 re-run, 2026-11-09), points 1 and 2. Dissent: none.
- **Numbers (P3-36 re-run):** `filter[priority]=1` p95 841 ms and `filter[site]=s7` p95 1,287 ms, against the 150 ms list target.
- **Decision:**
  - The list `total` counts at most 1,000 rows (`LIMIT 1001` in `records/store.List`, was 10,000), under the same `authz.CompileRead` predicate as the page. The response keeps `total_capped`.
  - A list with a filter on a `data` property runs both its statements under `statement_timeout` = 2 s (`store.DataFilterTimeout`, set with ``SET LOCAL`` and put back after the page). Past it the answer is `422 filter_too_expensive` (not a hang, not a 500). The earlier value is restored so the handler's later statements are not bound.
  - 422, not 503: the same filter will time out again, and the caller's fix is a narrower filter, not a retry.
  - No bound existed before: the pool and server configuration set no `statement_timeout` for the list route.
- **Deferred (D25):** an index or query-shape change for `data` filters. **Triggers:** TICK-646 starts, or any customer tenant passes 100k records.
- **Reopens:** the P3-36 re-run on TICK-621 after merge. If `state=new` or the admin `state in` list is still over 150 ms, the shape work reopens (CTO condition).
