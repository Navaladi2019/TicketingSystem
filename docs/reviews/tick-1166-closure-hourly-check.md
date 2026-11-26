# Review: PR #923, TICK-1166 hourly org_unit_closure comparison per tenant

Reviewer: Quality, Testing & CI Advisor (second reviewer, after the Security & Multi-Tenancy Reviewer). Head reviewed: `agent/senior-software-engineer/0d770eb63a1d` at `681bb9b`. Specs read first: the TICK-1166 description (build items 1 to 4, the seven acceptance tests, accepted limits), AR-TICK-1118-01, and the existing `liveUnitsBestEffort`, `withHashPlan` and `observe` code in `internal/groupsync`.

**Verdict: CHANGES.** The comparison logic matches the spec, and all seven acceptance tests ran and passed in CI. Two things block the merge:

- lint is red;
- the new lag alert cannot fire at the default 5-minute cycle, so build item 4 is not met. Its promtool test does not catch this.

CI: run [45800566707](https://github.com/Navaladi2019/TicketingSystem/actions/runs/45800566707) (`ci:run`). lint failed. unit, integration, isolation, stack (smoke), coverage and commit-check-budget passed. All 7 `TestClosureCheck*` tests and `TestAlertRules` passed.

## Findings, most severe first

### 1. Bug: lint fails, navalint rejects `set_config` (`internal/groupsync/groupsync.go:332`)

```
internal/groupsync/groupsync.go:332:31: set_config outside internal/tenancy: scope the transaction with tenancy.InTenantTx (or tenancy.AllocTx for a number)
```

`checkClosureBestEffort` puts back the old timeout with `SELECT set_config('statement_timeout', $1, true)`. golangci-lint passes. The rule is in `scripts/ci/navalint.sh`, which `make lint` (`task lint`) runs after golangci-lint.

**Fix:** use `SET LOCAL statement_timeout = '<prev>'`, with `'` doubled in `prev`. This is the `SET LOCAL` pattern `withHashPlan` uses (`closure.go:205`). Do not add a navalint exception.

### 2. Bug and Missing test: `GroupsyncTenantLagHigh` never fires at the default 5-minute cycle (`deploy/compose/prometheus/rules/groupsync.yml:74-77`)

`groupsync_lag_seconds` gets one observation per tenant per cycle. `Run` ticks every `DefaultInterval` (5 min), and the scrape interval is 15 s. A `[5m]` range holds about 4 min 45 s of samples, so at regular intervals the window misses a step of `_count`. On those evaluations `increase()` is 0, the expression is false, and the `for: 15m` timer resets. The alert stays pending and never fires.

Reproduced with promtool (prom/prometheus v3.7.3), on the PR's rule file as it is:

- **Input:** one failing tenant, lag always above 600 s, one observation every 5 minutes, 15 s samples. `_count` is `0x19 1x19 2x19 …` and `le="600"` stays at 0.
- **Expected:** the alert is firing at 55 m.
- **Got:** `got:[]`.
- **After changing both ranges to `[10m]`:** the same input fires.

The PR's test adds one observation every minute, five times the real rate, so it passes on a rule that cannot fire in production. This is exactly the case build item 4 exists for: a tenant whose sync keeps failing.

**Fix:**
- Change both ranges to at least twice the cycle, e.g. `[10m]`.
- Change the fire case to the real rate: `interval: 15s` with a step every 20 samples on `_count`.
- Keep the silent case, at the same rate, with `le="600"` keeping up with `_count`.
- In the rule comment, say the range must stay at least 2× `Options.Interval`.

### 3. Breaks under a slow cycle: the comparison can use up the tenant deadline and roll back the revocations (`groupsync.go:328`)

The statement timeout is always `TenantTimeout/2` (60 s by default), however much of the tenant's `context.WithTimeout` is left. Suppose a run reaches the due comparison with less than 60 s left, for example after a slow Keycloak export. Then the context deadline ends the comparison, not `statement_timeout`. `sp.Rollback(ctx)` and the commit then fail on the expired context, and the whole run is lost, principal tombstones included. Without the comparison that run would have committed. Build item 3 says "a failed comparison never undoes a revocation".

This is unlikely at the measured cost (0.4 to 1.3 s against a 2-minute budget), so it does not block.

**Fix:** set the timeout to `min(TenantTimeout/2, time left on ctx minus a margin)`. Below a floor (say 5 s), skip the comparison and leave the tenant due.

**Test:** a swapped `compareClosure` that sleeps past a short `TenantTimeout`, on a run that tombstones a principal. The tombstone must commit.

### 4. Pre-existing, routed to the Technical Lead: the in-apply comparison has no savepoint

The engineer flagged this as an open item. A due tenant whose applied run touched no unit takes the comparison inside `apply` (`compared = st.ClosureCompared`, `groupsync.go:440`). That comparison is not in a savepoint, so a failure there still fails the whole sync, as before TICK-1166.

It is not a regression. But the spec's "a failed comparison never undoes a revocation" holds only for the new path. The Technical Lead, who owns the spec, should decide whether that path also moves into a savepoint (a follow-up task), or record the exception under the accepted limits.

### 5. Preference: the subtree gauge is stale after a repair on a run that touched units (`groupsync.go:436`)

`subtreeMax` comes from `apply`, before a due repair, so `groupsync_org_subtree_max` reports the drifted closure for one cycle. `unrooted` is already recomputed after the repair; recompute `subtreeMax` there too (`storedOrgSubtreeMax`).

## What is right

- The due rule is correct. A missing entry is due at once. The time is recorded with `Now()` after the commit, only when a comparison ran, and never on a failed, rolled-back or skipped cycle (lag is not observed for skipped cycles).
- The comparison is in the same transaction as the delta, after `rebuildClosure`, on both branches. A repair sets the same `driftRepaired` flag, so the counter and WARN fire only after the commit.
- The savepoint rolls back `SET LOCAL` with it, so the restore is needed only on the commit path, which is how it is written.
- The seven acceptance tests can go Red: with `due` forced false, 7 of 7 fail (pasted in the PR). In CI they cover separate timers, an unchanged cycle, drift outside the touched subtree, a failure with a tombstone, rollback, skip and restart.
- No tenant-isolation concern: every query is scoped by the tenant id inside the tenant's own transaction. The Security reviewer has the isolation review.

## To re-check after the fix

The lint job is green, and `TestAlertRules` passes with the real-rate fire case.
