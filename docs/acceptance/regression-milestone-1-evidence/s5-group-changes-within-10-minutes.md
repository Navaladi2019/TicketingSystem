# S5 regression evidence (TICK-47 kit), main @ 4392c26, 2026-10-28, stack nava-regr

Kit drift (not failures): the kit says the stack has no `nava-groupsync` service and no onboarding API; main has both. Steps 2-3 were done as main allows: bindings seeded with the pinned issuer `http://localhost:28081/realms/nava` and `idp_aliases {idp-tenant-a|b}` (S3 file), and the compose service `nava-groupsync` (interval 5m) as the live worker, with `docker compose run --rm --no-deps nava-groupsync --once` for on-demand cycles. Scripts: Python helpers (`urllib`) against `127.0.0.1:28081` / `:28080`, Keycloak admin API as `admin` (dev-only). Tokens redacted to 10 chars.

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | group removal reflected < 10 min | PASS, 75 s (live worker) | below |
| 2 | disabled user's issued token stops < 10 min | PASS, 285 s (live worker) | below |
| 3 | synced tables never show another tenant's users | PASS | below |
| 4 | kc.org tamper, ID token, foreign-IdP user rejected | PASS by hand; CI-5 suite COVERED-BY-CI | below |
| 5 | B's admin cannot get a token the API accepts for A | PASS by hand; CI-5.13 COVERED-BY-CI | below |
| 6 | tenant admin 403 on /api/v1/operator/*; operator token 401 on tenant endpoint | PASS (403 on nava-onboard; 501 on nava-core ingress) | below |
| steps 1-3 | /me shows service-desk; removal removes it | PASS | crit 1 |

`make identity-attacks` (kit step 7) NOT run: forbidden locally; COVERED-BY-CI (`ci5-report` artifact).

## Criterion 1 (story steps 1-3)
```
17:23:26Z alice token eyJhbGciOi... iss http://localhost:28081/realms/nava exp in 300 s
17:23:26Z step1 /me 200 {"group_paths":[{"id":"7ebfea97-...","path":"/service-desk"}],...,"tenant_id":"...a1"}
17:23:27Z DELETE /admin/realms/nava/users/<alice>/groups/<service-desk> -> 204
nava-groupsync-1 17:24:28Z "group sync applied","tenant_id":"...a1"      (worker cycle)
17:24:42Z +75s same token /me 200 {"group_paths":[],"groups":[],...,"tenant_id":"...a1"}  token unexpired: True
17:24:42Z bob /me 200 {"group_paths":[{"path":"/service-desk"}],...}   (B unaffected)
```
## Criterion 2
```
17:24:53Z fresh alice token, /me 200 ; PUT users/<alice> enabled=false -> 204
17:24:53Z new sign-in -> 400 {'error': 'invalid_grant', 'error_description': 'Account disabled'}
nava-groupsync-1 17:29:28Z "group sync applied","tenant_id":"...a1","principals_written":1
17:29:39Z +285s same token /me 401 {"code":"tenant_unresolved",...}  token unexpired: True
```
Alice then re-enabled and re-added to service-desk; `--once` -> `/me` 200 with `/service-desk`.

## Criterion 3: as nava_app / nava_groupsync through PgBouncer (`iso.sh`), "visible/other-tenant" counts
```
nava_app tenant=none: principals=0/0 groups=0/0 group_members=0/0 org_units=0/0
nava_app tenant=...a1: principals=2/0 groups=1/0 group_members=1/0 org_units=3/0
nava_app tenant=...b1: principals=1/0 groups=1/0 group_members=1/0 org_units=3/0
nava_groupsync tenant=...a1: principals=2/0 groups=1/0 ... ; tenant=...b1: principals=1/0 groups=1/0 ...
```
Both tenants have `/service-desk`, different ids (`7ebfea97…` vs `6860fab6…`). 0 other-tenant rows for every tenant (5 tenants incl. the S4 ones) and role; full table in `cross-story.md` (3b).

## Criterion 4 / 5 by hand
```
link alice -> idp-tenant-b (userId ext-qa-regr) -> 204 ; --once ; principals.federated_links=[{"identityProvider": "idp-tenant-b"}]
rows holding ext-qa-regr: 0
old token /me 401 tenant_unresolved ; new token /me 401 tenant_unresolved
unlink -> 204 ; --once ; /me 200 tenant A, /service-desk
bob token, organization rewritten to tenant-a, original signature -> 401 invalid_token
alg=none forged tenant-a token                                     -> 401 invalid_token
bob scope "openid organization:tenant-a" -> org claim None -> /me 401 tenant_unresolved
bob scope "openid organization:*"        -> {'tenant-b': ...} -> /me 200 tenant ...b1
alice ID token as bearer                 -> 401 invalid_token
```
## Criterion 6
```
admin@tenant-a.test (realm_access {'roles': ['tenant-admin']}) /me 200
admin-a GET  nava-onboard :8081/api/v1/operator/tenants -> HTTP/1.1 403 Forbidden
admin-a POST nava-onboard :8081/api/v1/operator/tenants -> HTTP/1.1 403 Forbidden
admin-a GET|POST nava-core :28080/api/v1/operator/tenants -> 501 urn:nava:problem:not_implemented (tenant ingress never serves operator routes)
operator@nava.test via nava-operator-cli (aud nava-operator-api, roles nava-operator) -> nava-core /api/v1/me 401 invalid_token
bob@tenant-b.test via nava-operator-cli password grant -> 401, Keycloak HTML page "Access denied"
```
