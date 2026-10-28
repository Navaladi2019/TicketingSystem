# S7 regression evidence (TICK-49 kit), main @ 4392c26, 2026-10-28, stack nava-regr

Deviation from "Try it" step 1: `make image stack-up` NOT run (`stack-up` is forbidden locally); the stack was the S1 `docker compose build` + `up --wait`. The Temporal UI (`--profile operator`) was started with `docker compose --profile operator up -d --wait temporal-ui`, and the tunnel with the exact `make operator-tunnel` `docker run` on port 28233 (detached, removed at the end). Temporal was read through the UI's HTTP API (`/api/v1/namespaces/nava/workflows?query=WorkflowId='...'` and `/history`), not screenshots.

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | one workflow per start row, even when delivered twice | PASS | below |
| 2 | signals in commit order, each applied once | PASS (order + once into history); worker-side "applied once" COVERED-BY-CI (CI-6 21-23) | below |
| 3 | worker signalling tenant-B workflow denied + alert | COVERED-BY-CI (CI-6 case 8) | not run locally |
| 4 | kill relay between claim and start: no loss, one workflow | PASS (live drill) | below |
| 5 | CI-6 Authorizer cases pass | COVERED-BY-CI (`ci6-report` artifact) | not run locally |
| iso | tenant B cannot reach A's work | PASS | below |

## Criteria 1 and 2
```
make demo-outbox DEDUPE_KEY=regr-s7   (twice; both runs print identical ids)
start row: 01a18391-f48a-768b-...  signal row 1: 01a18391-f496-...  signal row 2: 01a18391-f49e-...
workflow id: t:00000000-0000-4000-8000-0000000000a1:nava.smoke.relay:01a18391-f48a-768b-ae5b-f7c7d0999708
outbox (pgadmin): 3 rows (not 6): regr-s7:start|signal-1|signal-2, attempts=1, dispatched=t, dead=f
Temporal: executions for <wid>: 1, RUNNING
  event 1 WORKFLOW_EXECUTION_STARTED / 2 WORKFLOW_TASK_SCHEDULED
  event 3 SIGNALED demo.note {"outbox_id":"01a18391-f496-...",...}   (signal row 1)
  event 4 SIGNALED demo.note {"outbox_id":"01a18391-f49e-...",...}   (signal row 2)
```
## Criterion 4: crash drill
```
go build -tags relaytest ./cmd/nava-relay ; docker compose stop nava-relay
test relay in a container on nava-regr_internal, nava-relay cert, nava_relay user, NAVA_RELAYTEST_PAUSE=1, stdin open
make demo-outbox DEDUPE_KEY=regr-crash -> start 01a18394-da5d-...
test relay stderr: relaytest: paused before StartWorkflow 01a18394-da5d-7d0f-96ad-fb6377cf3f15
docker kill -s KILL (17:35:41Z) -> start attempts=1 leased, not dispatched; signals attempts=0; Temporal executions: 0
docker compose start nava-relay -> 17:36:11Z start a=2 disp=true; signal-1 a=1 disp=true; signal-2 a=1 disp=true; none dead
Temporal: 1 execution, events 3/4 = signal 1, signal 2 in order. nava-relay /readyz ok
```
## Isolation (as nava_app through PgBouncer, `SET LOCAL app.tenant_id`)
```
as A: record=1 workflow_ref=1 outbox=6 ; as B: record=0 workflow_ref=0 outbox=0 outbox_A_rows=0
as B: INSERT outbox stamped A -> ERROR: new row violates row-level security policy for table "outbox"
as B: B row with A's target_key nava.smoke.relay:<A start> -> relay: "outbox signal target workflow not found or completed; row marked dead",
      workflow_id t:<tenant B>:nava.smoke.relay:...; A's workflow history unchanged (4 events)
```
Kit wording drift: the kit says nava_app reading the outbox gets `permission denied`. On main `db/grants.yaml` grants nava_app `SELECT (tenant_id, id, dedupe_key)` on outbox (for `INSERT ... RETURNING` and the dedupe lookup), behind tenant policy `outbox_select`. So `SELECT count(*)`/`dedupe_key` works for the caller's own tenant (A sees 6, B sees 0); `SELECT *` would still be denied. Isolation holds; the kit sentence is imprecise.
