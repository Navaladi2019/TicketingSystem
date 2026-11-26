# Security & Multi-Tenancy Reviewer on AR-TICK-1118-01 (TICK-1118 comment 01a210f9, 2026-11-25, at `5aa3adbf`)

**AR-TICK-1118-01 is 🟠 Medium. I approve the row with the text below; the current text leaves out writes.** No tenant can reach another tenant's rows through this. I read PR #870 at `5aa3adbf` and the code at `57573b39`, and ran nothing.

## Why Medium
| Level | Fits? |
|---|---|
| High | No. FORCE RLS pins each role's closure reads and writes to the transaction's tenant, and both foreign keys carry `tenant_id`. No user token can write a closure row. |
| Medium | Yes: "data exposure", and "the control is in doubt". The repair check skips every run that touched a unit and every unchanged cycle, and no signal shows that it repaired anything. |
| Low | No. The scale says "no tenant data at risk". |

The two Low rows it resembles do not match:
- **AR-TICK-901-01:** that lag ends at the next sync cycle. This one has no time limit.
- **AR-TICK-714-03** (a wrong side row): Low because the side row is never a payload source. The closure decides access.

## Corrections to the row (text only)
1. **Writes are missing (known).** `ACL.CanWrite` resolves `under` from the closure (`internal/authz/aclstore.go`), and a policy may carry `ops: ["read","write"]`. If the `write` policy is `under(EMEA)` and a wrong row (EMEA, HR) exists, an EMEA user can also edit HR's tickets.
2. **The count can also be too high (inferred from code).** Take two units a and b of the caller's U, neither below the other. A wrong row (a, x), with x strictly below b, makes `regionFrom` give x twice, and `countArm` does not remove repeats. x's tickets then count twice.
3. **The trigger needs `org_units` too.** A second writer of `org_units` that leaves the closure alone also causes drift. Today `nava_groupsync` is the only service role granted writes on it.

## The row
> **Sev:** 🟠 Medium
> **Risk:** ADR-010-E4 E4-7. Group sync replaces only the affected units' `org_unit_closure` rows, and compares the whole table with the live tree only on an applied run that touched no unit. A wrong row elsewhere lasts until such a run, with no time limit. An extra row grants through `under` and hides through `not under`; a missing row does the reverse. Either can let a user read **or change** tickets outside their rules, or refuse them tickets inside their rules, inside one tenant (FORCE RLS, FKs that carry `tenant_id`). An extra row can also make a list's count lower or higher than the tickets its pages show (inferred from code). Only a group-sync bug, the migrator, a superuser or a holder of group sync's database credentials can cause it; no user token can. No metric counts drift repairs (TICK-1124).
> **Trigger:** Any drift found in a live tenant, or a second writer of the closure or of `org_units`; then compare the sets on every cycle

Keep the Owner (me) and the Source. Set Status to "Accepted (ADR-010-E4, 2026-11-25; rated by the Security & Multi-Tenancy Reviewer)" when E4 is accepted.

**Where it changes:**
- **E4:** the risk table; "one new low-severity row" (now medium); the I5 sentence (count or write access, not only the count); the "Known gap" paragraph (TICK-1124).
- **Explainer:** "proposes to accept this as low"; "show a person tickets it should not" needs "or let them change"; the section 5 table.
- **Handbook:** "show or hide tickets" in `adr-010-e4.md`, and the generated pages.
- **Register:** `docs/security/accepted-risks.md:157`, then rebuild.

If that text goes in as written, I do not need a re-check.

## Finding
- **Attack:** none from a tenant token. It needs a faulty or privileged writer first.
- **Status:** `will bite us at scale`.
- **Severity:** Medium. It does not block E4 or S31.
- **Fix:** count and alert on repairs, then give a wrong row's life a time limit. I filed [TICK-1124](mention://issue/01a210f8-e610-7497-be73-7374bc8e61bd) (backlog, unassigned) for the Technical Lead to schedule.
- **Test:** none added (reading only). `internal/tenancy/isolation/closure_paths.go` already tests that both foreign keys refuse a cross-tenant row.

VERDICT: APPROVE (AR-TICK-1118-01 at Medium, with the row text above)

RESULT: DONE
