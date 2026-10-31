## 2026-10-31: PLAN-003 sequences the records path in two lanes; S10, S11b and S12 tasks are groomed later
- **Decided by:** the Technical Lead on TICK-332 (`docs/delivery/plan-records-path.md`), pending nava's `APPROVED`.
- **Decision:**
  - Lane A runs S8 → S9 → S10 → S11b → S12 in series. Every lane A story changes the OpenAPI root, the generated `api.gen.go`, `db/grants.yaml` and the isolation registry.
  - Lane B (S11a, the interpreter) runs in parallel with S8 because it shares none of those files.
  - The tasks for S10, S11b and S12 are created when the story before them merges.
  - Tenant admin = member of the tenant's `admins` platform group (D-3-3, open to the Technical Architect in grooming).
  - One workflow type, `nava.interpreter`, serves every definition, so the A1 RS5 trigger is not hit (D-3-11).
- **Deferral:** D25 indexed-attribute governance. *Trigger:* P3-01 measures p95 > 150 ms on fixed-column ACL predicates. Then the Technical Lead escalates to the CTO before S10 is groomed.
- **Dissent:** none yet (grooming pending).
