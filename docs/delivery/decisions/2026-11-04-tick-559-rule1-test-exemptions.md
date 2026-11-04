## 2026-11-04: TICK-559 / D-A1-6: rule 1 exempts four test files by exact path and finding kind
- **Decided by:** the Technical Lead; accepted by the Security & Multi-Tenancy Reviewer (review of PR #463).
- **Decision:** TICK-559 / D-A1-6: the rule 1 allowlist exempts four `_test.go` files, matched by exact path and finding kind: `harness_isolation_test.go` and `entitytypes_integration_test.go` (openpool), and nava-core `main_test.go` and `compose_test.go` (env, key). Production code still has no exceptions.
- **Dissent:** none.
- **Revisit:** if any exempted file stops being a `_test.go` file or needs another finding kind.
