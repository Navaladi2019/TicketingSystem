## 2026-11-22: S31, `groupsync_org_subtree_max` has no tenant label; the tenant is named in the 8,000+ warning log line

- **Decided by:** the Technical Lead; the Security & Multi-Tenancy Reviewer agreed (TICK-948, PR #805). Recorded on TICK-954 (S31 handbook). Story S31 (TICK-946), ADR-010 R8 (retired by ADR-010-A2).
- **Decision:** the histogram `groupsync_org_subtree_max` is recorded on every applied tenant sync with no `{tenant}` label. A tenant at 8,000 or more (`groupsync.SubtreeWarnAt`) is named in a WARN log line instead. The same rule holds for `groupsync_org_units_live_max` (TICK-1072) and `groupsync_org_closure_unrooted_units` (TICK-1058).
- **Why:** groupsync rule T01 (no tenant id is ever a metric attribute) overrides the TICK-948 task text, which asked for a `{tenant}` label. A tenant label would make the metric's cardinality grow with the tenant count and put tenant ids in the metrics store.
- **Deferral and trigger:** filling `nava_primary_org_unit` from a tenant identity provider's mapper is deferred to the follow-up issue. Revisit when a tenant's own IdP must set primary units.
- **Dissent:** none.
- **Note, 2026-11-25 (TICK-1116, PR #873):** the 8,000+ WARN line and `groupsync.SubtreeWarnAt` are removed: the 10,000-unit cap they warned about no longer exists (ADR-010-A2). The histogram stays. The decision that none of the three metrics has a tenant label still holds; the unrooted WARN line and the above-certified log line still name the tenant.
