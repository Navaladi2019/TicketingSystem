**Setup.** Windows 11 with Docker 28.0.1. Keycloak images `quay.io/keycloak/keycloak:<v>` running `start-dev` (H2). Temporal `temporalio/auto-setup:1.28` (server 1.28.4) on `postgres:16` with SQL visibility, namespace `nava`. Go SDK `go.temporal.io/sdk v1.49.0`, built with Go 1.26.0 inside `golang:1.24-bookworm` with `GOTOOLCHAIN=auto`. Python 3.13 with requests 2.32.5. The dataset is three orgs and three users. All credentials were throwaway dev-mode values.

**Hypotheses and pass criteria.**
- A1 passes if the access token can carry the org UUID, with a known shape for several orgs.
- A2 passes if `organization:<alias>` yields exactly that org, and a non-member alias yields no org.
- A3 passes if the tenant header is observable in the worker on start, signal, signal-with-start, activity, local activity, child, continue-as-new and retry, and a per-signal tenant can be compared.

**Raw output, Keycloak 26.0.7** (the 26.8.1 output is identical apart from the `initech` row, which shows `<absent>`):
```
mapper type oidc-organization-membership-mapper [... ('addOrganizationAttributes','false'), ('addOrganizationId','false')]
direct  bob   scope='organization'        -> {"organization": "<absent>"}
direct  bob   scope='organization:acme'   -> {"organization": {"acme": {"id": "66e5d316-..."}}}
direct  bob   scope='organization:*'      -> {"organization": {"globex": {...}, "acme": {...}}}
direct  alice scope='organization:globex' -> {"scope": "profile email", "organization": "<absent>"}
browser bob   scope='organization' pick=initech -> {"notes": ["org selection page shown, choices=['globex','acme']"], "organization": {"initech": {"id": "95de8021-..."}}}
alias change attempt: 400 {"errorMessage":"Cannot change the alias"}
members of initech: []
```

**Raw output, Temporal** (the `raw` column is the header copied by the interceptor; `prop` is the header carried by the `ContextPropagator`):
```
== T1 t:A:nocopy
OBS in:ExecuteWorkflow(attempt=1) wf=t:A:nocopy  raw=A        prop=A
OBS in:activity                  wf=t:A:nocopy  raw=<absent> prop=A
OBS in:localActivity             wf=t:A:nocopy  raw=<absent> prop=A
OBS in:ExecuteWorkflow(attempt=1) wf=t:A:nocopy:child raw=<absent> prop=A
OBS in:HandleSignal              wf=t:A:nocopy  raw=A        prop=A
OBS in:ExecuteWorkflow(attempt=1) wf=t:A:nocopy  raw=<absent> prop=A   <- continue-as-new run
== T1b signal carrying tenant B to A's workflow
OBS in:HandleSignal              wf=t:A:wait    raw=B        prop=B
== T2 retry
OBS in:ExecuteWorkflow(attempt=2) wf=t:A:retry  raw=A        prop=A
T3 t:A:nohdr-err      status=Failed  pendingWFTaskAttempt=0
T3 t:A:nohdr-panic    status=Running pendingWFTaskAttempt=2
T3 t:B:forged         (guard passed; header B and id t:B:… chosen by the client)
T4 running, default opts: err=<nil> sameRun=true
T4 after completion, default reuse policy: err=<nil> newRun=true
T4 after completion, REJECT_DUPLICATE (ErrorWhenAlreadyStarted=false): err=<nil> returnedRun==r3:true
T4 after completion, REJECT_DUPLICATE + ErrorWhenAlreadyStarted: err=Workflow execution already finished ... reject duplicate workflow Id.
T5 NavaTenantId = 'B' -> upsert wf present: true
```
In the `copy` mode, every hop showed `raw=A prop=A`.

**Rerun.**
- Keycloak:
  ```
  docker run -d -p 18080:8080 -e KC_BOOTSTRAP_ADMIN_USERNAME=admin -e KC_BOOTSTRAP_ADMIN_PASSWORD=<throwaway> quay.io/keycloak/keycloak:<v> start-dev
  python kc_spike.py http://localhost:18080
  ```
  Set the same password in `ADMIN_PW` inside the script.
- Temporal: start `postgres:16` plus `temporalio/auto-setup:1.28` with `DB=postgres12` and `DEFAULT_NAMESPACE=nava`, then run `TEMPORAL_ADDR=<host>:7233 go run .` with Go ≥ 1.26 and SDK v1.49.0.
