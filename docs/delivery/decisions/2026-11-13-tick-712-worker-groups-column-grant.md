## 2026-11-13: TICK-712, the worker reads `groups` through a column grant (TICK-782), not a pipeline-side fork

- **Decided by:** the Technical Lead on TICK-712 (PR #622), after the Principal Software Engineer's review of `8a75b243`. Endorsed by the CTO with no override.
- **Problem:** every `update_record` step through the worker's records adapter failed with `permission denied for table groups` (`42501`). `nava_worker` had no grant or policy on `groups`. Two reads hit it: the `LEFT JOIN groups` in `records/store` (`recordSelect`) and the `GroupLive` check when a write sets `assignment_group_id`.
- **Decision:** grant `nava_worker` column-level `SELECT (tenant_id, id, deleted_at)` on `groups`, with a tenant-scoped `FOR SELECT` policy. RLS stays enabled and forced. The grant and its `db/grants.yaml` entry go in the same migration (TICK-782). The four Security conditions in TICK-782 are binding, and Security's verdict is required before it merges. `8a75b243` (`store.GetForWrite` and the catch-all `42501` → `ErrForbidden` mapping) is reverted.
- **Reason:**
  - With the pipeline-side fix, a workflow could never assign a group: `GroupLive` fails every time on the worker role, so a step type would ship permanently broken for one field.
  - The grant fixes both reads with one reviewed change, instead of forking the shared store read.
  - Dropping the catch-all `42501` mapping means a future grant regression shows up as an error, rather than as a business refusal that is never retried.
- **Reversal on record:** at 23:31 on 2026-11-12 the Technical Lead briefly accepted `8a75b243` and deferred group assignment. That deferral is withdrawn and is not the outcome. The three `VERDICT: CHANGES` rounds on TICK-712 came from this decision churn, not from defects in the implementation. After TICK-782 merges and CI on the merged head is green (stale-cause replay included, 24 `cascade_too_deep` entries), the Principal Software Engineer re-checks only the merge result.
- **Dissent:** the Senior Software Engineer preferred the pipeline-side fix, because it gives no role new access. Security accepted either route.
