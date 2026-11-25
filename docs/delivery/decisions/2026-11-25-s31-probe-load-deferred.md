## 2026-11-25: S31, the probe load of one list is deferred with a trigger (`AR-S31-05`)

- **Decided by:** the Technical Lead, on finding 3 (Low, not blocking) of the Security & Multi-Tenancy Reviewer's whole-story review of PR #867 (TICK-946 comment 01a210af, 2026-11-25). Story S31 (TICK-946).
- **Finding:** one list can run up to 16 probes (`authz.ReadBranchThreshold`), each reading up to 20,001 records (`KStarMax` + 1), on the database all tenants share.
- **Decision:** ship S31 with that bound; nothing is built now. The R1 bench passed at the certified tree (70.08 / 82.69 ms p95 against 150 ms, TICK-1037). It has no row for a plan with 16 region branches.
- **Owner:** the Technical Lead.
- **Trigger to revisit:** `authz_under_probe_seconds` p95 above 25 ms for 15 minutes, or a slow-list report from a tenant whose policies use more than 4 `under` branches. No alert watches the first one: read it in Prometheus with `histogram_quantile(0.95, sum by (le) (rate(authz_under_probe_seconds_bucket[15m])))`.
- **Options when it fires:** a lower cap on region leaves per plan, or one probe for all leaves. Both need the Solution Architect.
- **Dissent:** none.
