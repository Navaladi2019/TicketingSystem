# Acceptance kit: S11a, the workflow interpreter runs a definition on Temporal (timer, branch, signal)

Story: TICK-334. Kit task: TICK-352 (P3-25). Plan: PLAN-003 (`docs/delivery/plan-records-path.md`). Design: ADR-001.
Verified on 2026-11-01 by the Quality, Testing & CI Advisor, on a fresh stack built from `TICK-334/workflow-interpreter` at `c486767`. That is the story tip with all of its tasks merged, including P3-27 (TICK-374): every run here is `nava.interpreter`, not the old smoke types. The demo targets in this change were added on top of it.

**Status: READY for the human.** All 4 criteria pass, and so does the cross-tenant check (the signal case). Run log: [`run.log`](s11a-workflow-interpreter/run.log).

## What you can now do

- Watch a workflow written in our own definition format run on Temporal: it waits on a timer, takes the branch its condition picks, waits for an `approve` signal, updates the record and completes.
- Publish a new version of a definition while runs are in flight. Those runs finish on the version they started with, and new runs use the new one.
- Kill the worker in the middle of a record update and restart it. The update is applied once, not twice.
- See CI refuse a code change that would break runs already in flight.

## Try it (about 15 minutes, most of it the first image build)

Needs Docker, Go 1.26 and `make`, from a clone of the story branch. The step-by-step version with expected output for each step is page **S11a** of the acceptance guide (`docs/acceptance/guide/s11a.html`).

1. Start a fresh stack with the Temporal UI and the test overlay, which defines the interpreter worker:
   ```sh
   git checkout TICK-334/workflow-interpreter
   cat >> .env <<'EOF'
   COMPOSE_PROFILES=operator
   COMPOSE_PATH_SEPARATOR=";"
   COMPOSE_FILE="compose.yaml;deploy/compose/compose.test.yaml"
   EOF
   make dev-ca
   docker compose down -v && docker compose build && docker compose up --wait
   ```
2. Put the interpreter worker in place, and open the UI tunnel in a second terminal:
   ```sh
   make demo-interpreter-up
   make operator-tunnel          # second terminal; then open http://127.0.0.1:18233, namespace "nava"
   ```
3. **Criterion 1:** `make demo-interpreter TENANT=a KEY=try PRIORITY=high`. In the UI the run is `nava.interpreter`, with id `t:00000000-0000-4000-8000-0000000000a1:nava.interpreter:<start row>`. After 5 s it waits for `approve`. Then run `make demo-interpreter-approve TENANT=a KEY=try` and it completes.
4. **Criterion 2:** run `make demo-interpreter-pinning TENANT=a KEY=pin PRIORITY=high`, which publishes fixture v2 while the v1 run waits on its timer. The v1 run still waits for `approve`. Then `make demo-interpreter TENANT=a KEY=new PRIORITY=high` starts a run on v2, and that run completes without waiting.
5. **Criterion 3:**
   ```sh
   make demo-interpreter-up PAUSE=1
   make demo-interpreter TENANT=a KEY=crash PRIORITY=high
   # wait about 10 s
   make demo-interpreter-crash TENANT=a KEY=crash
   # wait about 20 s
   make demo-interpreter-writes
   ```
   You should see two attempts, one effect.
6. **Criterion 4:** read the `unit` job of the latest `ci` run on the story branch (green) and of demo PR [#347](https://github.com/Navaladi2019/TicketingSystem/pull/347) (red).
7. Clean up: `make demo-interpreter-down`, stop the tunnel, then `docker compose down -v`.

There is no sign-in. Tenants never see workflows: no API route exposes them, and the Temporal UI is reachable only through the operator tunnel. The two tenants are the seeded `tenant-a` (`…a1`) and `tenant-b` (`…b1`). `TENANT=a|b` picks one.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | The fixture run waits on its timer, takes the branch its condition selects, waits for the `approve` signal, and completes. The UI shows workflow type `nava.interpreter` and id `t:<tenant>:...` | ✅ | Run `c1-high` (tenant A, priority high). The 5 s timer fired, `is_high` took `then`, and the run waited on `await_approval` with its 1-day timeout timer pending ([01](s11a-workflow-interpreter/01-crit1-waiting-for-approve.png)). After `approve`: `nava.interpreter.signal`, `ApplyStep` (mark_approved), `NextStep`, `nava.workflow_ref.Close`, then `Completed`. Workflow Type `nava.interpreter`, id `t:00000000-0000-4000-8000-0000000000a1:nava.interpreter:01a195e6-…` ([02](s11a-workflow-interpreter/02-crit1-completed.png)). The writer recorded one call, `{"approved":true}`, applied. The control run `c1-low` took `else` straight to the end ([05](s11a-workflow-interpreter/05-crit1-low-priority-else.png)). The same flow passed for tenant B (`b-high`, [07](s11a-workflow-interpreter/07-tenant-b-run-completed.png)). CI: `TestInterpreterStartedByTheRelayCompletes`, `TestFixtureV1Branches` |
| 2 | If fixture v2 is published while a v1 run waits on its timer, the v1 run finishes v1's steps | ✅ | Run `c2-pin`: `TIMER_STARTED` at 07:07:54.952 and `TIMER_FIRED` at 07:07:59.955. v2 was published at 07:07:56.373, inside the timer. The `NextStep` after the timer (07:07:59.979) read the definition after the publish and still took v1's `then` edge. The run waited for `approve`, then applied `mark_approved` ([03](s11a-workflow-interpreter/03-crit2-v1-run-kept-v1.png), [history](s11a-workflow-interpreter/history-01a195ef-abb8-74ef-a57f-3c86d63e472f.json)). A run started after the publish (`c2-pin-new`, input `…000000000002`) took v2's `else` and completed in 5.4 s ([04](s11a-workflow-interpreter/04-crit2-new-run-on-v2.png), [history](s11a-workflow-interpreter/history-01a195f0-0fda-774c-9bc6-ecebe0b8806c.json)). CI: `internal/workflow/interp/pinning_test.go` |
| 3 | If the worker is killed during an `update_record` step and restarted, the step's effect is applied once | ✅ | Run `c3-crash`. Attempt 1 of `ApplyStep` (mark_approved) committed its write and paused. The container `937d0ac22ae5` was SIGKILLed 7 s into the attempt (exit 137), then the same container was started again. Attempt 2 ran on it with last failure `activity StartToClose timeout`, and the run completed ([06](s11a-workflow-interpreter/06-crit3-crash-run-completed.png): "2 • nava.interpreter.ApplyStep", [history](s11a-workflow-interpreter/history-01a195f2-a9f5-7ecb-bdff-d4a0e925827c.json)). The writer's log, in a volume that outlives the container: attempt 1 `applied:true`, attempt 2 `applied:false`, one key, and one applied-effect file for it. CI: `TestInterpreterCrashAppliesOnce` (stack `interpreter` case) |
| 4 | The CI replay check is green on `main` and red on the demo PR that introduces a non-deterministic change | ✅ (story branch) | Green: run [41281378349](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41281378349) on the story tip `c486767`, all jobs. Red: demo PR [#347](https://github.com/Navaladi2019/TicketingSystem/pull/347), one commit on `c486767` that adds an activity before every timer. Its `unit` job fails `TestReplayRecordedHistories` on all four timer histories with `[TMPRL1100] … a matching Timer command was expected` (run [41281729917](https://github.com/Navaladi2019/TicketingSystem/actions/runs/41281729917)). "Green on `main`" can only be read once the story merges, because `main` has no interpreter yet. It is part of the story → `main` checklist |

Local `task test` and `task lint` pass on this change.

## Isolation check

Two things were tried as tenant B with tenant A's ids, on the same stack. Only the first one crosses the tenant boundary.

- **The cross-tenant proof: signal tenant A's waiting run.** `make demo-interpreter-approve TENANT=b RECORD_ID=<tenant A's c2-v1 record>`. Refused in the producer before any row is written: `outbox: record not found`. Row-level security hides tenant A's record and its open run from tenant B's transaction. Tenant A's run stayed waiting. CI covers the worker side of the same boundary: `TestInterpreterCrossTenantIsNotFound` (stack `worker` case). There, a tenant A start row naming tenant B's definition version or record fails as not found.
- **Not a cross-tenant check: a tenant B start that reuses tenant A's record UUID.** `make demo-interpreter TENANT=b KEY=iso-start RECORD_ID=<tenant A's c3-crash record>`. `records` is keyed by `(tenant_id, id)`, so this creates tenant B's own record row with the same UUID. It never touches tenant A's row. The relay starts `t:…b1:nava.interpreter:…`, and its first `NextStep` fails with `workflow rejected: definition version, record or step not found` (`nava.WorkflowRejected`, non-retryable), because tenant B's fixture store has no such record ([08](s11a-workflow-interpreter/08-isolation-b-start-on-a-record-rejected.png)). Nothing crossed tenants. It shows only that a reused UUID stays inside its own tenant.

The workflow list shows the runs of both tenants, each id carrying its own tenant ([09](s11a-workflow-interpreter/09-workflow-list-both-tenants.png)). The one `Running` row is `c2-v1`. It was the first criterion 2 attempt, discarded because its publish came after the timer (see below), and it was left waiting for `approve`.

## Not included

- **Database-backed definitions, records and writer.** The stack runs the interpreter with S11a's test fakes (`nava-worker-interptest`, `interptest` build tag): definitions and records are files, and the writer is a file log. The production `nava-worker` image links none of them (`TestProductionBuildLinksNoFakes`). The real ports come with the records path stories in PLAN-003.
- **A UI or API to publish a definition.** "Publish v2" installs `fixture-v2.json` into the fixture store under a new version id. Publishing through the product is a later story.
- **Any tenant-facing view of workflows.**

## Known limits

- **The demo swaps workers.** `make demo-interpreter-up` stops the always-on `nava-worker`, because it also polls `nava-tq` and does not register the interpreter's activities. `make demo-interpreter-down` restores it. While the demo runs, the S7 demo workflow type has no other worker. Nothing on the stack needs one.
- **The crash restart starts only the killed container.** `make demo-interpreter-crash` restarts it with `docker start <id>`, not `docker compose start`. Compose would also start the service's `depends_on` chain, including the stopped one-shot jobs (`db-roles`, `migrate`, `temporal-schema`), and it fails with `missing dependency db-roles` when something on the host has pruned them (TICK-407, first attempt of the TICK-404 run). The drill now passes after `docker container prune` mid-run: [`tick-407-run.log`](s11a-workflow-interpreter/tick-407-run.log). If the killed container itself is pruned before the restart, the script stops and says so; run `make demo-interpreter-up PAUSE=1` and start over.
- **The first criterion 2 attempt did not count.** Run `c2-v1` used `make demo-interpreter` followed by `make demo-interpreter-publish-v2`. The publish landed 9 s after the start row was dispatched, after the 5 s timer had fired, so it proved nothing. The run log keeps it, marked. `make demo-interpreter-pinning` was added to publish on dispatch, and the timestamps above come from that run.
- **Each `make demo-interpreter*` call takes about 10 s.** It builds the small `cmd/demo-interpreter` binary and runs it in a throwaway container on the stack's network, as `make demo-outbox` does.
- **This run used an isolated project.** The developer's own `nava` stack was running on the host, so this run used `COMPOSE_PROJECT_NAME=nava-s11a`, `NAVA_IMAGE_TAG=s11a` and shifted host ports (UI tunnel 28233), and left that stack untouched. The steps above use the defaults.
- **The story branch is behind `main` by 7 commits** (TICK-361/371/377/379/380/332: story-branches tooling, trace stack case, docs). None of them touches `internal/workflow/` or the worker. Merging `main` in is part of the story → `main` checklist.
