# S5 regression re-run (TICK-47 kit), main @ 76350ef (S5 post-merge PRs #267 in), 2026-10-28, stack nava-regr2

Kit drift as last run (kit predates the `nava-groupsync` service and onboarding): bindings came from the S3 `accept.py` seed (pinned issuer `http://localhost:28081/realms/nava`); the compose `nava-groupsync` (interval 5m) was the live worker; `docker compose run --rm --no-deps nava-groupsync --once` for on-demand cycles. Python `urllib` against `127.0.0.1:28081`/`:28080`, Keycloak admin API as `admin` (dev-only). Tokens cut to 10 chars.

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | group removal reflected < 10 min | PASS, 241 s (live worker, same unexpired token) | below |
| 2 | disabled user's issued token stops < 10 min | PASS, 291 s (live worker, same unexpired token) | below |
| 3 | synced tables never show another tenant's users | PASS | below |
| 4 | kc.org tamper, ID token, foreign-IdP user rejected | PASS by hand; CI-5 suite COVERED-BY-CI | below |
| 5 | B's admin cannot get a token the API accepts for A | PASS by hand; CI-5.13 COVERED-BY-CI | below |
| 6 | tenant admin 403 on /api/v1/operator/*; operator token 401 on tenant endpoint | PASS | below |
| steps 1-3 | /me shows service-desk; removal removes it | PASS | crit 1 |

`make identity-attacks` (kit step 7) NOT run: forbidden locally; COVERED-BY-CI.

## Criterion 1
```
21:47:57Z alice token eyJhbGciOi... iss http://localhost:28081/realms/nava exp in 300 s
21:47:57Z step1 /me 200 {"group_paths":[{"id":"fbb81a6d-...","path":"/service-desk"}],...,"tenant_id":"...a1"}
21:47:57Z DELETE /admin/realms/nava/users/<alice>/groups/<A service-desk> -> 204
nava-groupsync-1 21:51:58Z "group sync applied","tenant_id":"...a1"
21:51:59Z +241s same token /me 200 {"group_paths":[],"groups":[],...,"tenant_id":"...a1"} token unexpired: True
21:51:59Z bob /me 200 {"group_paths":[{"id":"f66af762-...","path":"/service-desk"}],...,"tenant_id":"...b1"}
```
## Criterion 2
```
21:52:08Z fresh alice token, /me 200 ; PUT users/<alice> enabled=false -> 204
21:52:08Z new sign-in -> 400 {"error":"invalid_grant","error_description":"Account disabled"}
nava-groupsync-1 21:56:58Z "group sync applied","tenant_id":"...a1","principals_written":1
21:56:59Z +291s same token /me 401 {"code":"tenant_unresolved",...} token unexpired: True
21:56:59Z bob /me 200 tenant ...b1
```
Alice re-enabled and re-added; `--once` -> `/me` 200 with `/service-desk`.

## Criterion 3 (`iso.sh` as nava_app / nava_groupsync / nava_worker through PgBouncer, `visible/other-tenant`)
```
nava_app tenant=none:  principals=0/0 groups=0/0 group_members=0/0 org_units=0/0 tenant_identity_binding=5/5 (allow-listed binding_lookup)
nava_app tenant=a1:    principals=2/0 groups=1/0 group_members=1/0 org_units=3/0
nava_app tenant=b1:    principals=1/0 groups=1/0 group_members=1/0 org_units=3/0
nava_groupsync a1|b1:  principals=2/0|1/0 groups=1/0|1/0 group_members=1/0|1/0 org_units=3/0|3/0 (records/outbox/tenants: denied)
```
0 other-tenant rows for all 5 tenants and all three roles, except the allow-listed `tenant_identity_binding`. Same `/service-desk` name in A and B, different ids (`fbb81a6d…` vs `f66af762…`). Full sweep in `cross-story.md` 4b.

## Criterion 4 / 5 by hand
```
link alice -> idp-tenant-b (userId ext-qa-regr2) -> 204 ; --once -> tenants 5, failed 0
rows holding ext-qa-regr2: 0
old token /me 401 tenant_unresolved ; new token /me 401 tenant_unresolved
unlink -> 204 ; --once ; /me 200 tenant A ['/service-desk']
bob token, organization rewritten to tenant-a, original signature -> 401 invalid_token
alg=none forged tenant-a token                                     -> 401 invalid_token
bob scope "openid organization:tenant-a" -> org claim None         -> /me 401 tenant_unresolved
bob scope "openid organization:*" -> ['tenant-b']                  -> /me 200 tenant ...b1
alice ID token as bearer                                           -> 401 invalid_token
```
(The `federated_links` read for the linked state failed on a wrong column name in my script; the two 401s are the evidence.)
## Criterion 6
```
admin@tenant-a.test realm_access {'roles': ['tenant-admin']} /me 200
admin-a GET|POST nava-onboard /api/v1/operator/tenants -> 403 urn:nava:problem:forbidden
admin-a GET|POST nava-core   /api/v1/operator/tenants -> 501 urn:nava:problem:not_implemented (tenant ingress never serves operator routes)
operator@nava.test (aud nava-operator-api, roles nava-operator) -> nava-core /api/v1/me 401 invalid_token
bob via nava-operator-cli password grant -> 401
```
nava-onboard reached through the guide's `tools/tunnels.sh up` (`ACCEPT_ONBOARD_PORT=28082`).
