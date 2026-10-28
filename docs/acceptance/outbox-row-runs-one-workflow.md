# Acceptance kit: S7, a committed outbox row runs exactly one tenant-scoped workflow

Story: TICK-49. Plan: PLAN-001 (`docs/delivery/plan-platform-foundation.md`, T27a–T33). Design: ADR-001 (Annex B, CI-6 and CI-7).
Verified on 2026-10-27 by the Quality, Testing & CI Advisor, on a fresh stack built from `TICK-49/outbox-row-runs-one-workflow` at `7c7fde1` (current `main` merged in). The only later commit, `c9c88e5` (PR #214), changes CI and docs only.

**Status: READY for the human.** All 5 criteria and the cross-tenant check pass.

## What you can now do

- When the platform saves a change that should start background work, that work starts exactly once, even if the hand-off is repeated or the process doing it crashes halfway.
- Follow-up messages for that work arrive in the order they were saved.
- Background work for one tenant cannot touch another tenant's work. An attempt is refused and raises an alert.
- An operator can watch all of this in the Temporal UI.

## Try it (about 10 minutes, most of it the first image build)

Needs Docker and Go 1.26, from a fresh clone of the story branch. Windows Git Bash works as is: the targets set `MSYS_NO_PATHCONV=1` themselves (TICK-277).

1. Start a fresh stack:
   ```sh
   docker compose down -v
   make dev-ca image stack-up
   docker compose up --wait
   ```
   Every service ends `Healthy`.
2. Run the demo twice with the same key:
   ```sh
   make demo-outbox DEDUPE_KEY=try-it
   make demo-outbox DEDUPE_KEY=try-it
   ```
   Both runs print the same `start row`, `signal row 1`, `signal row 2` and `workflow id` (`t:00000000-0000-4000-8000-0000000000a1:nava.smoke.relay:<start row>`). The second run adds nothing.
3. In a second terminal, run `make operator-tunnel` and leave it running. Open http://127.0.0.1:18233 and choose namespace `nava`.
   - The workflow list has **one** row for that workflow id ([screenshot](outbox-row-runs-one-workflow/01-workflow-list.png)).
   - Open it. History shows the start, then two `demo.note` signals. The first carries `signal row 1`'s id, the second `signal row 2`'s id ([screenshot](outbox-row-runs-one-workflow/02-crit1-history.png)). It stays `Running`: no worker runs this demo workflow type.
4. Read the CI-6 report: the `ci6-report` artifact of the latest `ci` run on the story branch (first produced by run [40577539025](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40577539025)). Its last line reads **All 43 CI-6 and CI-7 cases pass.**
5. Clean up: stop the tunnel (Ctrl+C), then `docker compose down -v`.

There is no sign-in. Tenants never see workflows: no API endpoint exposes them, and the Temporal UI is reachable only through the operator tunnel.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | Exactly one workflow per start row, even when delivered twice | ✅ | `make demo-outbox DEDUPE_KEY=s7-acceptance` twice: the same ids, 3 outbox rows (not 6), each dispatched once (`attempts=1`), one execution with one run in Temporal ([01](outbox-row-runs-one-workflow/01-workflow-list.png), [02](outbox-row-runs-one-workflow/02-crit1-history.png)). CI: `TestDemoOutbox`, `TestCI6_RelayRedeliveredStartAfterCompletionStartsNoSecondRun` |
| 2 | Signals arrive in commit order and each is applied once | ✅ | Live history: event 3 is signal row 1, event 4 is signal row 2, and no repeats ([02](outbox-row-runs-one-workflow/02-crit1-history.png)). Applied once on the worker side, where the demo has no worker: CI-6 cases 21 to 23 (`…RedeliveredSignalAppliedOnceAcrossContinueAsNew`, `…SignalQueuesBehindItsStart`, `…TwoRelaysNeverReorderOneTarget`) |
| 3 | A worker trying to signal a tenant-B workflow is denied and an alert fires | ✅ | CI-6 report case 8 `TestCI6_WorkerCrossTenantSignalExternalDeniedWithAlert`: PermissionDenied, an `alert=true` error log line, and the victim's history has no signal. Also cases 7, 9 and 19 (child, cancel, wrong-tenant signal) |
| 4 | Killing the relay between claim and start loses no event and starts exactly one workflow | ✅ | Live drill, below ([03](outbox-row-runs-one-workflow/03-crit4-crash-history.png)). CI: `TestT33CrashBetweenClaimAndStart` |
| 5 | The CI-6 Authorizer cases (T32a) pass before the relay exists | ✅ | Cases 1 to 17 passed in the T32a report of 2026-10-23 (`docs/security/ci6-report.md`), before the relay cases existed. They pass again in today's report, with the relay in place |

CI on the verified tip: run [40574634701](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40574634701) (`7c7fde1`) and run [40577539025](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40577539025) (PR #214): all 9 jobs green. Local `task test` and `task lint` pass.

### Criterion 4: the crash drill

1. The stack's relay was stopped. A relay built with `-tags relaytest` (`NAVA_RELAYTEST_PAUSE=1`, see `docs/architecture/outbox-relay.md`, "Crash seam") ran in its place, with the same certificate and database user.
2. `make demo-outbox DEDUPE_KEY=s7-crash` wrote a start and two signals. The test relay claimed the start and printed `relaytest: paused before StartWorkflow 01a17efe-8756-…`.
3. It was killed with `SIGKILL`. The start row was leased with `attempts=1` and not dispatched; the signals were untouched. Temporal had no workflow for it.
4. The stack's relay was started again. Once the lease expired, it dispatched all three rows, 22 seconds after the kill (start row `attempts=2`).
5. Temporal now shows one execution with one run, and the two signals in order ([03](outbox-row-runs-one-workflow/03-crit4-crash-history.png)). No row is dead and none is missing.

## Isolation check

Run live as `nava_app` through PgBouncer, with `app.tenant_id` set as `InTenantTx` sets it.

| Tried as tenant B | Result |
|---|---|
| Read tenant A's record and its `workflow_ref` | 0 rows each (tenant A sees 1 of each) |
| Read the outbox | `permission denied for table outbox`: the app can write the outbox but never read it |
| Write an outbox row stamped with tenant A's id | `new row violates row-level security policy for table "outbox"` |
| Signal tenant A's workflow: a tenant-B row with A's `target_key` and A's record id | The relay sent it to `t:<tenant-B>:nava.smoke.relay:<A's start row>`, which does not exist. Row marked dead, error log `outbox signal target workflow not found or completed`. Tenant A's workflow history is unchanged (still 4 events) |

At the Temporal layer, CI-6 cases 7 to 11, 15 and 19 cover a worker that holds tenant B's work and reaches for tenant A's workflow. Each one is denied with an alert.

## Not included

- No business workflow runs yet. The demo's `nava.smoke.relay` type has no worker in the stack, so its workflow stays `Running`. Feature workflows arrive with Epic 05.
- No tenant-facing view of workflows. Only operators see Temporal, through the tunnel.

## Known limits

- **Criterion 2, "applied once":** the demo shows ordering and exactly-once delivery into history. "Applied once" by a worker is proved only by the CI-6 cases above, because no worker runs the demo type.
- **Committed CI-6 report:** `docs/security/ci6-report.md` is the 2026-10-23 T32a run (17 cases). The current 43-case report is the `ci6-report` CI artifact, uploaded since PR #214.
- **Plain `docker compose up`:** this does not start the Temporal UI. `make stack-up` does, because the Makefile sets `COMPOSE_PROFILES=operator`.
- Accepted risks recorded for this story: AR-039 to AR-043 in `docs/security/accepted-risks.md`.
