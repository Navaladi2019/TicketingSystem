# Cross-story checks and Postman run, main @ 76350ef, 2026-10-28, stack nava-regr2 (after the kits)

Issuer config throughout: `KEYCLOAK_PUBLIC_URL=http://localhost:28081`, so `NAVA_OIDC_ISSUERS`, `NAVA_ONBOARD_ISSUER`, `NAVA_OPERATORS`, Keycloak `KC_HOSTNAME` and every binding issuer were `http://localhost:28081/realms/nava`.

4a ran as a throwaway Go test added with `go test -overlay` (no repo file written; uses the identitystack helpers `newOnboarding`, `operatorToken`, `onboardPost`, `createIdPRealm`, `pointIdPAt`, `brokeredSignIn`, `groupsyncOnce`, `getMe`, `adminRealm`). Outbox rows were written by `cmd/demo-outbox` built with an overlay that reads the tenant from `DEMO_TENANT` (same `outbox.Enqueue` in `InTenantTx` as nava_app), run in a container on the internal network as `scripts/dev/demo-outbox.sh` does. The live `nava-relay` ran throughout; `nava-groupsync` was stopped by the helper for the `--once` cycles and restarted after.

## 4a Onboarded tenant -> sign-in -> group sync -> one tenant-scoped workflow, same issuer: PASS
```
POST /api/v1/operator/tenants (nava-onboard, operator@nava.test) -> 201 {"state":"active","tenant_id":"7c28167c-68ce-57ce-b754-7fe41b5d3231"}
replay same key/body -> 200 identical=true
binding: http://localhost:28081/realms/nava org=22bd88d5-… idp=idp-7c28167c-…   (pinned issuer: http://localhost:28081/realms/nava)
brokered sign-in admin@onb-7c28167c.test via idp-7c28167c-… -> token eyJhbGciOi... iss=http://localhost:28081/realms/nava org={t-7c28167c-…}
groupsync --once ; /me -> 200 {"group_paths":[],"groups":[],"idp_alias":"idp-7c28167c-…","tenant_id":"7c28167c-…"}
Keycloak: create /t/7c28167c-…/regr2-team -> 201 ; add admin -> 204 ; groupsync --once
same token /me -> 200 {"group_paths":[{"id":"f6e1c258-…","path":"/regr2-team"}],...,"tenant_id":"7c28167c-…"}
DB: principal admin@onb-7c28167c.test enabled=true deleted=false ; group /t/7c28167c-…/regr2-team
demo-outbox (DEMO_TENANT=7c28167c-…, DEDUPE_KEY=xstory2-7c28167c) x2 -> identical ids
outbox: 3 rows, tenant 7c28167c-…, attempts=1, dispatched=true, dead=false
Temporal: WorkflowId='t:7c28167c-…:nava.smoke.relay:01a1848b-e2c8-…' -> 1 execution; STARTS_WITH 't:7c28167c-…:' -> 1
  events 3/4 SIGNALED demo.note outbox_id 01a1848b-e2ca-… then 01a1848b-e2cc-… (commit order)
onboarded admin -> nava-onboard POST /api/v1/operator/tenants -> 403
```

## 4b No cross-tenant rows with group sync and relay running: PASS
`iso.sh` as nava_app / nava_groupsync / nava_worker through PgBouncer, for each of the 6 tenants and with no tenant, 12 tables incl. `idempotency_keys`; each cell `visible/rows of another tenant`. Non-zero cells only:
```
nava_app a1:   principals=2/0 groups=1/0 group_members=1/0 org_units=3/0 records=5/0 workflow_ref=2/0 outbox=6/0 idempotency_keys=2/0 tenants=1/0
nava_app b1:   principals=1/0 groups=1/0 group_members=1/0 org_units=3/0 records=3/0 outbox=1/0 idempotency_keys=1/0 tenants=1/0
nava_app 7c28: principals=1/0 groups=1/0 records=1/0 workflow_ref=1/0 outbox=3/0 tenants=1/0 tenant_email_domain=1/0
nava_groupsync a1|b1: principals 2/0|1/0 groups 1/0|1/0 group_members 1/0|1/0 org_units 3/0|3/0 ; onboarded tenants: pending_invite=1/0
nava_worker a1: records=5/0 workflow_ref=2/0 outbox=6/0 ; 7c28: records=1/0 workflow_ref=1/0 outbox=3/0 (identity tables, idempotency_keys: denied)
no tenant set: 0/0 for every table and role, except tenant_identity_binding
```
The only non-zero "other tenant" cells: `tenant_identity_binding=6/5` (and 6/6 with no tenant), the allow-listed `binding_lookup` policy (ADR-001 I1), by design.
Superuser sweep: `group_members` whose group/principal tenants differ 0; groups outside `/t/<own tenant>/` 0; `workflow_ref` with another tenant's record 0; `idempotency_keys` whose principal is not in the row's tenant 0; `outbox` whose payload `record_id` is not its tenant's 1 = the S7 `regr2-forge-signal` probe row (tenant B aimed at A), marked dead by the relay.
Temporal ids per tenant prefix: a1 = 2 (S7 demo + crash drill, plus `accept` later), b1 = 0, 7c28 = 1, other onboarded tenants 0.

## 4c S6 ground rules on GET /api/v1/me: PASS (problem+json, request id, 429 per tenant); Idempotency-Key: not applicable to /me
- problem+json + `type urn:nava:problem:<code>` + `instance = urn:nava:request:<X-Request-Id>` on 401 (no token, `not-a-jwt`), 404, 405 (`Allow: GET, HEAD`), 429: see `s6-api-ground-rules.md`.
- 429: tenant A flooded -> `{200: 121, 429: 279}`, `Retry-After: 1`, problem+json `rate_limited`; tenant B at the same moment 60/60 `200` (43 of them inside A's 429 window); and the reverse. Per tenant, shared by its users (alice + admin-a share one bucket).
- Idempotency-Key on `/me`: GET passes through (`200`, `200`, no `Idempotent-Replayed`, 0 rows); POST `/me` 405. As S6 defines it (writes only), it has no tenant write route to apply to on main; scoping proven on a harness route, see `idempotency.md`.
- Observed, unchanged from last run: client `X-Request-Id` not echoed (server generates its own); `Accept: application/xml` still 200 JSON. New: `HEAD /api/v1/me` -> 405 though `Allow` lists HEAD (Bug, low).

## 4d Tenant admin -> /api/v1/operator/* 403; operator -> tenant endpoint 401: PASS
```
admin@tenant-a.test (tenant-admin) -> nava-onboard GET|POST /api/v1/operator/tenants -> 403 urn:nava:problem:forbidden
onboarded admin@onb-7c28167c.test  -> nava-onboard POST /api/v1/operator/tenants   -> 403
admin-a -> nava-core :28080 /api/v1/operator/tenants -> 501 not_implemented (tenant ingress never serves operator routes)
operator@nava.test (nava-operator-cli, aud nava-operator-api) -> nava-core GET /api/v1/me -> 401 invalid_token
```

## Postman collection (Newman 6.2.3 via `npx -y newman@6`)
`docs/acceptance/guide/postman/TicketingSystem.postman_collection.json` with `local.postman_environment.json`, overridden with `--env-var` for this stack (`core_url :28080`, `keycloak_url localhost:28081`, `onboard_url :28082`, `admin_url :29090`, `temporal_ui_url :28233`, `demo_workflow_id` of `make demo-outbox DEDUPE_KEY=accept`). Before the run: `tools/seed.sh`, `tools/tunnels.sh up` (ports above), `make demo-outbox DEDUPE_KEY=accept` x2, `nava-groupsync` restarted with `NAVA_GROUPSYNC_INTERVAL=15s` (as the guide's verified run). Duration 5 m 41 s.

| | executed | failed |
|---|---|---|
| requests | 96 | 0 |
| test scripts | 84 | 0 |
| prerequest scripts | 24 | 0 |
| assertions | 245 | 1 |

The one failure: **S7.1** "ids are tenant-scoped: t:<tenant-a>:…": `expected 't:7c28167c-…' to match /^t:00000000-0000-4000-8000-0000000000a1:nava\.smoke\.relay:…$/`. S7.1 lists every `WorkflowType="nava.smoke.relay"` execution and asserts each is tenant A's; this stack also held the 4a workflow of the onboarded tenant `7c28167c`. Kit fragility (the request assumes the demo is the only workflow of that type), not a product failure: the 7c28 workflow is correctly scoped to its own tenant. Fix for the guide: filter by `WorkflowId STARTS_WITH 't:<tenant-a>:'` or by `{{demo_workflow_id}}`. Every other request and assertion passed (S1-S6, S7.2). The collection has no 429 request (see S6 notes). Raw Newman JSON holds bearer tokens and is not published.
