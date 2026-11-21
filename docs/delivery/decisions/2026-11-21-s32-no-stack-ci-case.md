## 2026-11-21: S32 has no stack CI case; the fresh-stack acceptance kit is the gate

- **Decided by:** the Technical Lead, with the Principal Software Engineer's agreement (TICK-958). Recorded on TICK-959 (S32 handbook). Story S32 (TICK-955), ADR-010 D9.
- **Decision:** S32 adds no `s32_*.sh` case to the stack CI job. The named-policy test is gated by the unit and integration tests of TICK-956 and TICK-957 (`TestNextStepPolicyTest`, `TestNextStepPolicyTestFailsClosed`, `TestNamedPolicyAllows`, the `TestPolicyTest_*` RLS cases in `internal/records/pipeline/policyref_integration_test.go`, `TestWorkflowDefinitionPolicyTest`) and, end to end, by the fresh-stack kit `docs/acceptance/S32/` (`run.sh`, Bruno folder `S32 workflow policy`).
- **Why:** criterion 2 depends on a 20 s timer and the narrowing landing inside it, and the kit needs a tenant with no other incident workflow, which the shared CI stack (S11b's workflows) does not give. A stack case would be slow and clock-flaky; the run-time path it would cover is already proven against real Postgres in CI.
- **Deferral and trigger:** revisit when a later story changes `internal/workflow/interp` or the S11b incident workflows; that story adds a stack case or re-runs the S32 kit as part of its acceptance.
- **Dissent:** none.
