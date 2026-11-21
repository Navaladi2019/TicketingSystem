<!-- Archived verbatim from TICK-1040 (thread 01a1fcaa), 2026-11-21. -->

## Security review of PR #818: A2-6 confirmed; AR-TICK-1040-01 accepted once its text is corrected

I read the amendment, the `accepted-risks.md` diff, and D7 and D5a in ADR-010. I found no path from tenant A to tenant B's data.

**A2-6 is confirmed:**
- **The probe stays inside the tenant.** It runs as `nava_app`, with `records` and `org_unit_closure` both under FORCE RLS, plus an explicit `tenant_id = $t` on both tables. The closure FKs carry `tenant_id`.
- **The client can't steer it.** U comes from `org_unit_members` or from policy ids that D4 `foreign_id` has already checked. `$e` is an entity type the caller already holds a policy on, because the leaf only survives folding in that case.
- **The probe never changes rows.** C2, W0 and `Allows` share the `org_unit_id IS NOT NULL AND â€¦` guard. The probe picks only the SQL text. An error returns 500 with no fallback, so it fails closed. Not caching anything across requests in v1 is correct.
- **The metric is fine.** `authz_under_list_shape_total{shape}` is one bit, aggregated with no tenant, user or unit label.

**AR-TICK-1040-01 is accepted as ðŸŸ¢ Low, not âšª Info.** The register text needs correcting first, because the bit is broader than "the caller's own subtree":

- **Attack:** a user in tenant A lists entity `e` and times the response. The probe counts **every** record of `e` under U. It does not apply the branch's other conjuncts, and it runs for `not under` leaves too. So the user learns whether more than 1,000 records sit under U, **including records they cannot read**. For example:
  - a branch like `under X AND assignee = caller`;
  - a `not under Legal` policy, where every record counted is hidden from the caller;
  - `of: [ids]`, where U is units the caller is not a member of.
- **Status:** exploitable today. Within one tenant it gives one bit per (U, entity type), behind a few milliseconds of timing noise. It does not cross tenants.
- **Severity:** Low.
- **Fix:** accept it; no code change. Putting the branch conjuncts into the probe would complicate A2-5 and still leave a bit. Replace the register row's description with:
  > The capped probe picks the `under` list shape (C2 at â‰¤ 1,000 records, the walk above), so a list's response time can reveal whether more than 1,000 records of that entity type sit under the leaf's units U. U is the caller's units, or units a policy names. The count covers records the caller cannot read: other conditions in the branch, `not under` leaves. Rows returned never change, the count never leaves the server, and the bit stays inside the caller's tenant.
  
  Change the re-decide trigger to: "A requirement that record volume per org unit is confidential from users whose policy names that unit, **or** any client-supplied filter that would run the probe."
- **Test:** none yet. This PR is docs only. The guards land in TICK-1035 and TICK-1032 (below).

**Requirements on the implementing tasks.** None of these blocks A2. They stop the bit, or a tenant leak, from growing later.
1. **TICK-1036:** run the probe only for ACL-policy `under` leaves. A future request filter on `org_unit` must not reuse it without a new security review, because a caller-chosen U would make this an oracle on any unit in the tenant.
2. **TICK-1036:** the shape and the count stay out of responses (no `Server-Timing` or debug headers) and out of per-request logs or spans that carry a tenant or user id.
3. **TICK-1036:** `$t` in the probe and the page SQL comes from the same request-context value as `set_config('app.tenant_id', â€¦, true)`. If they disagree, RLS returns zero rows, which fails closed; the code must not paper over that.
4. **TICK-1032 and TICK-1035:** the cross-tenant case is a test, not just prose. Tenant B has closure rows and records under the same unit ids that tenant A uses as U. A's probe, C2, W0 and `Allows` must each return nothing of B's, and the probe count must not change between B empty and B with more than 1,000 records.
5. **Any future cache of the probe result:** the key must include tenant, entity definition and U, and the cache needs re-review. This is recorded as an at-scale risk, not a blocker.

Your added agreement case (a live unit below a tombstoned one, given as an `of` id) is correct and fails closed for `under`. For `not under` it opens those records, which is D7's stated semantics, so it is not a new finding. Please keep that `not under` variant in the TICK-1035 case list so the behaviour stays pinned.

**On your open question 3:** don't skip the probe. Skipping it would trade a Low intra-tenant bit for the +30.6 ms W0 cost at area size.

VERDICT: APPROVE (A2-6 and AR-TICK-1040-01 at Low, after the register text above is applied). At-scale risks recorded: items 1 and 5.