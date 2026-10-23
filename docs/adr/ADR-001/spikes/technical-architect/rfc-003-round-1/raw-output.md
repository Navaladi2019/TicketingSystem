# RFC-003 round-1 spike (Technical Architect), raw output

Source: TICK-129 comment `01a16780-b3fc-7da9-9aee-4ec79ab35742` (Technical Architect critique, round 1). The section below is the comment's spike report, copied unedited; the program is `main_test.go` in this folder.

## Spike report (verbatim)

**Verdict:** A1 and A2 are confirmed. F2-6 is new and reproduced at the Authorizer layer. **Hypothesis:** no `nava-worker` API other than the polls registers a deployment, and every allow-listed worker API that names a queue or a workflow id is checked. **Pass/fail:** each crafted request is DENY → pass, and ALLOW → fail. **Setup:** Go 1.26.10 windows/amd64, the `temporal-server` module at `84d9bf0` (api v1.53.0, server v1.29.7), in-process `authz.NewAuthorizer`. Nothing here needs a live server.

```
ShutdownWorker sticky=nava-tq                              -> ALLOW
ShutdownWorker sticky=temporal-sys-per-ns-tq               -> ALLOW
ShutdownWorker heartbeat tq=temporal-sys-x                 -> ALLOW
RecordWorkerHeartbeat tq=temporal-sys-x                    -> ALLOW
F2-5 GetWorkflowExecutionHistory temporal-sys-scheduler:x  -> ALLOW
RespondQueryTaskCompleted token tq=temporal-sys-per-ns-tq  -> ALLOW
F2-3 RespondActivityTaskCompleted token wid=temporal-sys-x -> ALLOW
F2-1 PollWorkflowTaskQueue VERSIONED                       -> ALLOW
control: ShutdownWorker other namespace                    -> DENY
unloaded partition key={ns nava-tq 0 1}  nava-tq root WORKFLOW key={ns nava-tq 0 1}  equal=true
SDK default caps -> deployment=<nil>
UseVersioning+series+build -> deployment=series_name:"s" build_id:"b"
DeploymentOptions UNVERSIONED -> deployment=<nil>
```

**How to rerun:** create a module next to the checkout with `replace github.com/Navaladi2019/TicketingSystem/temporal-server => ../TicketingSystem/temporal-server`, copy `temporal-server/go.sum`, then run `go mod tidy && go test -v ./...`.
