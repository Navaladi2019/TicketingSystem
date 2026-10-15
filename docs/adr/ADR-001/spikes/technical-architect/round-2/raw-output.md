**A7, §4.3 list as written:**
```
AUTHZ DENY cn=nava-relay api=GetSystemInfo ns=
T2 dial nava-relay    err=failed reaching server: Request unauthorized.
T2 dial nava-core     err=failed reaching server: Request unauthorized.
T2 dial nava-worker   err=failed reaching server: Request unauthorized.
```

**A7, with `GetSystemInfo` added:**
```
T1 dial error: failed reaching server: write tcp 127.0.0.1:44140->127.0.0.1:7233: write: broken pipe   (rogue CA)
T3 worker.Start with nava-worker cert: failed reaching server: Request unauthorized.
AUTHZ DENY cn=nava-worker api=DescribeNamespace ns=nava
AUTHZ DENY cn=nava-worker api=ShutdownWorker ns=nava
```

**A7, with `GetSystemInfo`, `DescribeNamespace` and `ShutdownWorker`, and no command inspection:**
```
T4 nava-relay  Start OK | Signal OK | Cancel PermissionDenied | Describe PermissionDenied | Terminate PermissionDenied
T4 nava-worker Start PermissionDenied | Signal PermissionDenied | Cancel PermissionDenied | Terminate PermissionDenied
T5 result via core: out="t:A:ok:child" err=OK
T6 parent result: out="t:B:evil-child" err=OK
T6 t:B:evil-child EXISTS status=Completed
T7 t:B:victim status=Completed (Completed means the cross-tenant signal landed)
```

**A7, with command inspection (`INSPECT=1`):**
```
T5 result via core: out="t:A:ok:child" err=OK
AUTHZ DENY cn=nava-worker api=RespondWorkflowTaskCompleted ns=nava cross-tenant StartChildWorkflowExecution t:A:evil->t:B:evil-child
AUTHZ DENY cn=nava-worker api=RespondWorkflowTaskCompleted ns=nava cross-tenant SignalExternalWorkflowExecution t:A:signaller->t:B:victim
T6 t:B:evil-child describe: error: workflow not found for ID: t:B:evil-child
T7 t:B:victim status=Running
```

**A6 (Keycloak 26.8.1):**
```
-- Q3 existing email (bob@globex.example): [User with email bob@globex.example already exists. Please login to account management to link the account.]
    state bob: links=[] orgs=['globex']
-- Q2 in-domain eve@acme.example: TOKEN organization=<absent> idp=idp-a      state eve: links=['idp-a:eve'] orgs=[]
-- Q1 out-of-domain xavier@other.example: TOKEN organization=<absent> idp=idp-a
== flow create-if-unique + idp-add-organization-member
-- eve2@acme.example: TOKEN organization={"acme":{"id":"929bce13-..."}} idp=idp-a   orgs=['acme']
-- xavier2@other.example: TOKEN organization=<absent>   orgs=[]
-- xavier2, scope organization:acme: TOKEN organization=<absent>
== built-in 'first broker login', trustEmail=ON
-- existing email: confirm-link page (fields: submitAction, submitAction); bob links=[]
```

**Environment:** Windows 11 with Docker 28.0.1. Keycloak `quay.io/keycloak/keycloak:26.8.1` (`start-dev`). The Temporal server was built from `go.temporal.io/server` v1.28.4 with Go 1.25.8 or later, and the client used `go.temporal.io/sdk` v1.49.0 with Go 1.26.0. Both ran on `debian:bookworm-slim`, with throwaway ECDSA P-256 certs generated at server start.

**Rerun A7:** build `server/` and `client/` as separate modules (`CGO_ENABLED=0 go build`). Start the server with a `/certs` volume (`INSPECT=0|1`, `EXTRA=GetSystemInfo,DescribeNamespace,ShutdownWorker`). Run the client in the server's network namespace with `TEMPORAL_ADDR=127.0.0.1:7233`, then send SIGTERM to the server to print the decisions.

**Rerun A6:** `python a6_spike.py http://localhost:18081` against a dev-mode Keycloak.
