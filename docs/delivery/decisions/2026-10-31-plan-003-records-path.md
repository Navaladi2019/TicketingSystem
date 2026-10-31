## 2026-10-31: PLAN-003 drafted: records path, serial stories, I9 cycle rule sent for an erratum
- **Decided by:** the Technical Lead on TICK-332 (`docs/delivery/plan-records-path.md`), pending grooming and nava's approval.
- **Decision:**
  - Stories S8 (entity types) → S9 (records) → S10 (access rules) → S11 (workflows) run serially; S10 and S11 share `db/grants.yaml`, `api/errors.yaml` and the isolation registry (D-R1). nava may choose a registry split to run S10 ‖ S11.
  - Spike R00 measures the Epic 06 150 ms list target on day 1; a fail escalates D25 to the Architecture squad (D-R2).
  - Until S10, record access is tenant-admin only, failing closed (D-R6).
  - ADR-001 I9's cycle rule, as worded, rejects a workflow writing back to the record that started it; R27 asks the Solution Architect for an erratum and TICK-88 waits on it. Depth limit 8 confirmed (D-R12).
- **Cut:** ACL conditions on `data` fields, sort on `data` fields, Temporal Worker Versioning, forced breaking schema changes, record deletion. Each has a trigger in PLAN-003.
- **Dissent:** none yet (grooming pending).
