## 2026-11-09: D-3-27, the stack seed publishes a temporary allow-all ACL until P3-35
- **Decided by:** the Technical Lead on TICK-616 (P3-31), option (b) of the CI Advisor's report on PR #523. Dissent: none.
- **Why:** P3-31 makes record authorization fail closed, and the stack's demo tenants never went through onboarding, so they have no ACL. The S9 Bruno collection then answers 403 and 404, the `stack` job fails, and `stack-db` and `stack-internal` never run for any S10 PR. The CI gate stays strict (no merge with a red job), so leaving the stack red was not an option.
- **Decision:**
  - `docs/acceptance/guide/tools/seed.sh` (the CI and dev stack seed only) publishes ACL version 1 for each seeded type (`incident`, `request`, `change` in tenants A and B). The rule `seed-allow-all-d-3-27` lets every principal and every live group of the tenant read and write every record, in the `acltest` shape. Only a type with no ACL yet gets one.
  - No production path changes: onboarding and the runtime still fail closed.
- **Removal owner:** TICK-620 (P3-35). Its acceptance criteria now include removing the temporary seed ACL and passing the S9 and S10 Bruno flows on the real onboarding default ACL.
- **Revisit when:** P3-35 merges.
