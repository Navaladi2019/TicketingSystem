---
title: "2. Sign in"
section: tour
status: delivered
sources: [docs/acceptance/s3-tenant-signin.md, docs/acceptance/s28-first-routed-record.md, docs/quality/x1a-first-routed-record.md, docs/acceptance/guide/bruno/S3/S3-4.bru, docs/acceptance/guide/bruno/S3/S3-5.bru]
adrs: [ADR-001, ADR-008]
stories: [S28]
operations: [getMe]
tables: [tenant_identity_binding]
---

## In plain words

People sign in with their company login, the same way they sign in to other work tools. The platform then knows who they are and which customer they work for, taken from the sign-in itself. It ignores any tenant name a caller types into a request, so one customer cannot ask to see another customer's data.

## What you do

Get an access token from the identity system (Keycloak), then send it to the platform. `GET /api/v1/me` shows who the platform thinks you are.

## Try it

Bruno: `S3.3 Sign in as tenant A's admin`, then `S3.4 Tenant A admin: GET /me`. The token comes from the `token` helper in [the setup](index.md#before-you-start), which uses the demo password flow of the local realm:

```sh
curl -i $CORE/api/v1/me -H "Authorization: Bearer $ADMIN"
curl -i $CORE/api/v1/me                                    # no token
curl -i $CORE/api/v1/me -H "Authorization: Bearer $ADMIN" \
  -H "X-Tenant-Id: 00000000-0000-4000-8000-0000000000b1"   # ask for tenant B
```

## What you see

The token's content (trimmed; `organization` is how Keycloak names the customer):

```json
{"iss":"http://localhost:18081/realms/nava","aud":"nava-api","azp":"nava-web",
 "realm_access":{"roles":["tenant-admin"]},
 "organization":{"tenant-a":{"id":"00000000-0000-4000-8000-0000000000a1"}},
 "email":"admin@tenant-a.test"}
```

`/me` with that token:

```json
{"group_paths":[{"id":"beed2119-…","path":"/admins"}],"groups":["beed2119-…"],
 "idp_alias":"","tenant_id":"00000000-0000-4000-8000-0000000000a1","user_id":"e4a636ce-…"}
```

With no token: `401 invalid_token`. With the header naming tenant B: the answer is still tenant A's, and the header is ignored.

## A newly onboarded tenant's first admin

The demo users above come from `seed.sh`. The first admin of a tenant you onboarded in step 1 is different in two ways.

- **The local stack has a stand-in company login, and the admin signs in through it.** A real admin signs in through the company login named at onboarding. On the local stack that login is a dev-only Keycloak realm, `standin-company`, and the onboarding request names it as the IdP (step 1): authorization URL `http://localhost:18081/realms/standin-company/protocol/openid-connect/auth`, token and JWKS URLs on `http://localhost:8080/realms/standin-company/...`, client `nava-broker` with the dev secret `dev-only-broker-secret`. Ask it for the scopes `openid profile` only (`defaultScope`): it has no `email` scope and refuses one with `invalid_scope`, and it sends the email and `email_verified` anyway. Onboarding accepts those two addresses only because the local compose file lists them in `NAVA_ONBOARD_DEV_IDP_HOSTS`; the setting is empty everywhere else, and every other private address is still refused. The admin creates their own user: open the platform sign-in with `kc_idp_hint=idp-<tenant_id>`, choose **Register** on the stand-in's page, enter the invited email and a password of your choice, and you are sent back signed in. No master admin or other staff help is needed. Bruno `S28.3` does the same steps in a script. The first sign-in creates the platform user and puts it in the tenant's organization.
- **`/me` answers `401 tenant_unresolved` until group sync has run.** The admin's invite is turned into membership of `/admins` by the next group-sync cycle (step 4): up to 5 minutes by default, 15 s with `NAVA_GROUPSYNC_INTERVAL=15s`. Until then a valid token gets `401 tenant_unresolved`. Wait one cycle, sign in again and repeat the call. Do not run `seed.sh`: it seeds the demo tenants, not yours.

## What happens behind it

The platform checks the token's signature, issuer and audience, then looks up the token's organisation in its own table (`tenant_identity_binding`) to find the tenant. Groups are read from the platform's database, not from the token. A token that names no organisation, or two, is refused. Details: [token validation](../security/01-token.md) and [tenant resolution](../security/02-tenant-resolution.md).

## Not built yet

- There is no sign-in screen. The password flow used here is for the local stack only; a real tenant signs in through its own company login.
- The local stack has no stand-in company login for a newly onboarded tenant. Its first admin signs in only with platform-staff help (the S28 kit's stand-in). We revisit this when a test company login is added to the stack or a human timed run is planned.
- There is no way to revoke a token before it expires except disabling the user, which the platform notices within one group-sync cycle (step 4).
