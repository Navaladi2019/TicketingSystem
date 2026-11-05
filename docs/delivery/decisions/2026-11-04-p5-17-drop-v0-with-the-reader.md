## 2026-11-04: P5-17 drops audit_log_v0 in the same PR as the reader; reads fail between deploy and migrate
- **Decided by:** the Technical Lead on TICK-435, after the Postgres review of PR #438. No dissent recorded.
- **Why:** the TICK-449 grooming rule puts the DROP in the task that rebuilds the per-record history on `audit_log`. The new binary reads only `audit_log`, so it needs no `audit_log_v0`. But the new reader's `SELECT` on `audit_log` is applied by `nava-db migrate`, not by the deploy. Between the binary deploy and `nava-db migrate`, `GET /api/v1/audit/entries` and the record history fail. Record writes are not affected.
- **Decision:** the drop stays in this PR. The runbook ("Rollout order" in `docs/architecture/record-write-pipeline.md`) and the migration header say that the deploy and `nava-db migrate` run back to back. Every stack today is development.
- **Deferral trigger:** if a stack that must have zero downtime exists before this ships, the `DROP` and the removal of the `audit_log_v0` grant move to a later release. The sequence is then: migrate with the reader grant, deploy the binary, then drop.
- **Dissent:** none.
