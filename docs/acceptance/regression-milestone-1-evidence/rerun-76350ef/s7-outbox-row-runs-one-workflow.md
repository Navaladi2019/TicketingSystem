# S7 regression re-run (TICK-49 kit), main @ 76350ef, 2026-10-28, stack nava-regr2

`make image stack-up` NOT run (forbidden); the stack is the S1 `docker compose build` + `up --wait`. Temporal UI: `docker compose --profile operator up -d --wait temporal-ui`, reached through the guide's `tools/tunnels.sh up` on `127.0.0.1:28233` (same proxy as `make operator-tunnel`). Temporal read through the UI's HTTP API (`/api/v1/namespaces/nava/workflows?query=...`, `.../workflows/<id>/history`).

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | one workflow per start row, even when delivered twice | PASS | below |
| 2 | signals in commit order, each applied once | PASS (order + once into history); worker-side "applied once" COVERED-BY-CI (CI-6 21-23) | below |
| 3 | worker signalling tenant-B workflow denied + alert | COVERED-BY-CI (CI-6 case 8) | not run locally |
| 4 | kill relay between claim and start: no loss, one workflow | PASS (live drill) | below |
| 5 | CI-6 Authorizer cases pass | COVERED-BY-CI | not run locally |
| iso | tenant B cannot reach A's work | PASS | below |

## Criteria 1 and 2
```
make demo-outbox DEDUPE_KEY=regr2-s7   (twice; both outputs identical, diff empty)
start row 01a18487-00fb-...  signal 1 01a18487-00fe-...  signal 2 01a18487-0101-...
workflow id: t:00000000-0000-4000-8000-0000000000a1:nava.smoke.relay:01a18487-00fb-7914-8d21-42c419b278d7
outbox (pgadmin): 3 rows (not 6) regr2-s7:start|signal-1|signal-2, attempts=1, dispatched=t, dead=f
Temporal WorkflowId='<wid>' -> 1 execution, RUNNING
  event 1 WORKFLOW_EXECUTION_STARTED / 2 WORKFLOW_TASK_SCHEDULED
  event 3 SIGNALED demo.note {"outbox_id":"01a18487-00fe-...",...}  (signal row 1)
  event 4 SIGNALED demo.note {"outbox_id":"01a18487-0101-...",...}  (signal row 2)
```
## Criterion 4: crash drill
```
CGO_ENABLED=0 GOOS=linux go build -tags relaytest ./cmd/nava-relay ; docker compose stop nava-relay
test relay: container on nava-regr2_internal, nava-relay cert, nava_relay DB user, NAVA_RELAYTEST_PAUSE=1, stdin open
make demo-outbox DEDUPE_KEY=regr2-crash -> start 01a18488-169c-7124-9f41-a260048fe0dd
test relay: "relaytest: paused before StartWorkflow 01a18488-169c-..."
docker kill -s KILL at 22:01:18Z -> start attempts=1 leased(available_at>now) not dispatched; signals attempts=0; Temporal: 0 executions
docker compose start nava-relay 22:01:29Z -> 22:01:44Z start attempts=2 dispatched; signal-1/2 attempts=1 dispatched; none dead
Temporal: 1 execution; events 3/4 = signal 1, signal 2 in order. nava-relay /readyz ok
```
## Isolation (as nava_app through PgBouncer, `SET LOCAL app.tenant_id`)
```
as A: records=5 workflow_ref=2 outbox=6 ; as B: records=3 workflow_ref=0 outbox=0 outbox_A_rows=0
as B: SELECT * FROM outbox -> ERROR: permission denied for table outbox
as B: INSERT outbox stamped A -> ERROR: new row violates row-level security policy for table "outbox"
as B: B row (regr2-forge-signal) with A's target_key nava.smoke.relay:<A start> ->
   relay: "outbox signal target workflow not found or completed; row marked dead", workflow_id t:<tenant B>:nava.smoke.relay:01a18487-00fb-...
   A's workflow history unchanged (4 events)
```
Kit wording drift (unchanged since last run): nava_app has `SELECT (tenant_id, id, dedupe_key)` on outbox under the tenant policy, so `count(*)` works for its own tenant; `SELECT *` is denied as the kit says.
