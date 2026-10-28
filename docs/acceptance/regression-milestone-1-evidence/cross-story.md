# Cross-story regression checks, main @ 4392c26, 2026-10-28, same stack nava-regr (after the kits)

Issuer config for every step: `KEYCLOAK_PUBLIC_URL=http://localhost:28081`, so `NAVA_OIDC_ISSUERS`, `NAVA_ONBOARD_ISSUER`, `NAVA_OPERATORS` and the binding issuer were all `http://localhost:28081/realms/nava`.

How 3a/3d ran: a throwaway Go test added with `go test -overlay` (nothing written into the repo; it uses the identitystack helpers `operatorToken`, `onboardPost`, `createIdPRealm`, `pointIdPAt`, `brokeredSignIn`, `groupsyncOnce`, `getMe`), run as `go test -overlay ... -tags 'identitystack acceptance' -run TestRegressionCrossStory ./internal/identitystack`. The outbox step used `cmd/demo-outbox` built with an overlay that reads the tenant from `DEMO_TENANT` (the same `outbox.Enqueue` in `InTenantTx` as `nava_app`), run exactly as `scripts/dev/demo-outbox.sh` runs it. The live `nava-groupsync` worker and `nava-relay` stayed up throughout.

## 3a Onboarded tenant -> sign-in -> group sync -> one tenant-scoped workflow: PASS
```
POST /api/v1/operator/tenants (nava-onboard, operator@nava.test) -> 201 {"state":"active","tenant_id":"e948c248-44b5-5217-96f9-b3843ccb38d6"}
replay same key/body -> 200 (same body)
brokered sign-in admin@regr-e948c248.test via idp-e948c248-... -> token eyJhbGciOi...
groupsync --once ; GET /api/v1/me -> 200 {"group_paths":[],"groups":[],"idp_alias":"idp-e948c248-...","tenant_id":"e948c248-..."}
Keycloak: create /t/e948c248-.../regr-team -> 201 ; add admin -> 204 ; groupsync --once
same token GET /api/v1/me -> 200 {"group_paths":[{"id":"023564ef-...","path":"/regr-team"}],...,"tenant_id":"e948c248-..."}
DB: principal admin@regr-e948c248.test tombstoned=false | group /t/e948c248-.../regr-team | binding http://localhost:28081/realms/nava idp=idp-e948c248-...
demo-outbox (DEMO_TENANT=e948c248-..., DEDUPE_KEY=xstory-e948c248) x2 -> identical ids both runs
outbox: 3 rows, tenant e948c248-..., attempts=1, dispatched=true, dead=false
Temporal: 1 execution t:e948c248-...:nava.smoke.relay:01a18399-886e-...; events 3/4 = signal 1, signal 2 in order
WorkflowId STARTS_WITH 't:e948c248-...:' -> 1 execution
```
## 3b No cross-tenant rows with group sync and relay running: PASS
`iso.sh` as nava_app / nava_groupsync / nava_worker through PgBouncer, for each of the 6 tenants and with no tenant; each cell is `visible/rows of another tenant`:
```
nava_app tenant=...a1:   principals=2/0 groups=1/0 group_members=1/0 org_units=3/0 records=5/0 workflow_ref=2/0 tenants=1/0 outbox=6/0
nava_app tenant=e948...: principals=1/0 groups=1/0 group_members=1/0 org_units=0/0 records=1/0 workflow_ref=1/0 tenants=1/0 tenant_email_domain=1/0 outbox=3/0
nava_groupsync tenant=e948...: principals=1/0 groups=1/0 group_members=1/0 pending_invite=1/0 (records/outbox/tenants: permission denied)
nava_worker tenant=...a1: records=5/0 workflow_ref=2/0 outbox=6/0 (identity tables: permission denied)
no tenant set: 0/0 everywhere for all three roles
```
The only non-zero "other tenant" cells in all 21 lines are `tenant_identity_binding=8/6..8/8`: that is the allow-listed `binding_lookup` policy (ADR-001 I1, S2 kit criterion 1), by design.
Superuser sweep: `group_members` whose group/principal tenants differ: 0. `workflow_ref` rows whose record is in another tenant: 0. `outbox` rows whose `record_id` is in another tenant: 1, the `regr-forge-signal` row from the S7 isolation probe (tenant B row aimed at A's record), which the relay marked dead.
Temporal: workflow ids per tenant prefix: a1=2, e948=1, others 0. Observation: `NavaTenantId='<tenant>'` returns 0 executions for every tenant; relay-started workflows carry no NavaTenantId search attribute (the guard only refuses a wrong one). Not a kit criterion.

## 3c S6 API ground rules on GET /api/v1/me (S5 endpoint): PASS for problem+json / request id / content type; rate limit and idempotency not present (known)
```
no token              -> 401 Content-Type: application/problem+json, WWW-Authenticate: Bearer, X-Request-Id: 0d2405f5...
                         {"type":"urn:nava:problem:invalid_token","title":"Invalid token","status":401,"code":"invalid_token","detail":...,"instance":"urn:nava:request:0d2405f5..."}
"not-a-jwt" / Basic / tampered signature -> same 401 problem+json shape, own X-Request-Id each
valid                 -> 200 Content-Type: application/json, X-Request-Id present
POST /api/v1/me       -> 405 problem+json, Allow: GET, HEAD
GET /api/v1/nope      -> 404 problem+json urn:nava:problem:not_found
nava-onboard 403      -> Content-Type: application/problem+json, X-Request-Id, urn:nava:problem:forbidden
```
Observed, not failures: `instance` always equals `urn:nava:request:<X-Request-Id>`; a client-sent `X-Request-Id` is not echoed (server always generates one); `Accept: application/xml` still gets JSON 200; no `RateLimit*`/`Retry-After` headers and 60 rapid calls all 200 (T22a/T22b not on main); `Idempotency-Key` on GET ignored (T24 not on main); 200 `/me` carries no `Cache-Control`. nava-core access log has `request_id`, `trace_id`, `route`, `status`, `duration` per request.

## 3d Tenant admin -> /api/v1/operator/* 403; operator -> tenant endpoint 401: PASS
```
admin@tenant-a.test (tenant-admin) -> nava-onboard GET|POST /api/v1/operator/tenants -> 403 urn:nava:problem:forbidden
onboarded admin@regr-e948c248.test -> nava-onboard POST /api/v1/operator/tenants -> 403
admin-a -> nava-core :28080 /api/v1/operator/tenants -> 501 not_implemented (tenant ingress never serves operator routes)
operator@nava.test (nava-operator-cli, aud nava-operator-api) -> nava-core GET /api/v1/me -> 401 invalid_token
```
