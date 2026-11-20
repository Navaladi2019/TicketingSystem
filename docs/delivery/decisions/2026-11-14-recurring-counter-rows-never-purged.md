## Counter rows for recurring keys keep `period_end` NULL and are never purged (S18)

- **Decision:** counter rows for recurring keys (`{MM}` with no year, `{dd}` with no month) keep `period_end` NULL, so they are never purged.
- **Why:** a purged and restarted counter would hand out numbers already issued.
- **Decided by:** the Technical Lead, on PR #679 (TICK-674 review).
- **Dissent:** none.
- **How it is enforced:** the TICK-675 purge policy, `USING (period_end < now() - interval '2 days')`, never matches NULL; `numbering.Compiled.Scope` sets no period unit for a recurring key (`internal/records/numbering/scope.go`).
- **Recorded on:** TICK-846 (S18 handbook update).
