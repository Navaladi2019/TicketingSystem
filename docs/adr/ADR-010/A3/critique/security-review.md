# Security review of ADR-010-A3 rev 0 (TICK-946 comment 01a20329, Security & Multi-Tenancy Reviewer, 2026-11-22)

## Security review of ADR-010-A3 (PR #856), A3-7: APPROVE, with one sentence rewritten

I checked A3 as an attacker who holds a valid token for tenant A and wants tenant B's rows. I read the A3 md, the register diff, the SP-1f spike diff, the A2-6 requirements, and the RLS migrations on `TICK-946/in-my-region`. **No cross-tenant path exists, and none is added.**

**Why R stays inside the tenant**
- **The closure set.** In R's hashed set, `org_unit_closure` is filtered by an explicit `tenant_id = $t` and also by its policy, which is ENABLE + FORCE RLS (`20261121050000_org_unit_closure.sql`). The folded probe's capped count reads the same table under the same two predicates. B's closure rows under A's unit ids can't raise A's n, so B can't flip A's shape.
- **The records R reads.** R reads only the type's records within the outer `r.tenant_id` predicate and RLS on `records`. That is the same reliance C3, W0 and CC already have. A B record whose `org_unit_id` equals one of A's units is never a candidate.
- **The estimate.** N̂_e comes from `entity_record_estimates`, which is FORCE RLS, and its key includes the tenant. B's estimate can't move A's skip rule or A's cap.
- **NULLs and negation.** Removing `IS NOT NULL` is sound: `(x IN …) IS TRUE` is false for a NULL, and `NOT(...)` is true for it, so `not under` keeps D5a in every shape. Error handling still fails closed: a probe error, or a NULL P3 value under the cap, returns 500 and never falls back to another shape.
- **The new label.** `shape="records"` is a metric label with no tenant, user or unit label, so it doesn't conflict with requirement 2.

**Finding 1 (Low, within the tenant, not blocking): the AR-TICK-1040-01 sentence misses one bit.**
- A3-2 step 3 (skip the count when N̂_e > α·U_cert) makes the response measurably faster, by about 5 ms in SP-1f. So timing also shows whether the tenant has more than about 33,400 records of the type. That count includes records the caller can't read.
- It is the same class as the existing K\* sentence, and the bit stays in the tenant. But the register should name it.
- Please replace the proposed sentence in A3-7 and in `docs/security/accepted-risks.md` with:

> Under ADR-010-A3, time-to-response can also reveal two more bits about the same entity type, both inside the caller's tenant: whether N̂_e exceeds α·U_cert (about 33,400 records, where the closure count is skipped), and whether S has at least ⌈N̂_e/α⌉ units (R is chosen). N̂_e counts records the caller cannot read. Neither bit changes rows, and neither depends on another tenant's data.

- Status column: append `; A3 sentence confirmed by Security & Multi-Tenancy Reviewer, TICK-946, 2026-11-22`, and drop both "re-check pending" markers. This also covers the K\* sentence, which I confirm too. The severity stays 🟡 Low.

**Finding 2 (not blocking): make A3-7 item 3 an exact test.** The implementation task's test must extend `TestUnderListShapesStayInTheTenant` (`internal/records/pipeline/acl_under_shape_integration_test.go`) so that:
1. `FixedListShapes(ShapeRecords)` joins `hows()`, and A's rows and totals under R are the same before and after B is added.
2. A's folded-probe n is unchanged by B, with the cap set to A's |S| + 1. If B's closure rows under A's ids were counted, n would reach the cap and flip A to R. Assert that A's `ChooseListShapes` result is also unchanged.
3. B gets an `entity_record_estimates` row whose value would flip A's step 3 or step 4 if A read it. A's choice must stay the same.
4. B holds records whose `org_unit_id` is one of A's unit ids. Where the FKs allow it, those records also use A's `entity_definition_id`. A's R rows exclude all of them.
5. A text test, in the style of `TestShapeTextsCastEveryPlaceholder`, asserts that the R leaf and the folded probe's closure count each carry the `tenant_id = $t` predicate. RLS would hide a dropped predicate in the integration test, so only a text test catches it, and requirement 3 needs both layers.

I'll review that test on the implementation PR. A3-7 already applies requirements 1 and 5 to the folded probe and to R unchanged: no request filter can use them, and there is no cross-request cache. I agree with that.

**At-scale risks recorded (no new security risk):**
- A3-R4: losing the hashed SubPlan above about 20,000 units costs time only.
- A3-R5: a stale-low N̂_e costs time only.

VERDICT: APPROVE (A3-7, given finding 1's wording)
