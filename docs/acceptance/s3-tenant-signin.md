# Acceptance kit: S3, a tenant user signs in and is scoped to their tenant only

Story: TICK-45. Plan: PLAN-001 (`docs/delivery/plan-platform-foundation.md`), tasks T14 to T17b. Design: ADR-001, Annex A.1.
Verified on 2026-10-27 by the Quality, Testing & CI Advisor, on the story branch `TICK-45/tenant-user-signin` with current `main` merged in (`fd38e34`). The stack was fresh: `docker compose down -v`, then `docker compose up --wait` (healthy in 220 s). `/readyz` returned `200 ok`.

**Status: READY for the human.** All 3 criteria, story step 3 and the cross-tenant check pass. 16 of 16 checks pass, and nothing failed.

## What you can now do

- Sign in through Keycloak as a seeded user and call `GET /api/v1/me`. You get back your tenant, your user id and your groups.
- The tenant comes only from the signed token and the platform's own binding table. Putting another tenant's id in a header, the query string or the body changes nothing.
- A token that names no organization, or more than one, is refused. So are an ID token and a token from a Keycloak issuer the platform does not trust.

## Try it (about 10 minutes, most of it waiting)

Needs Docker, Go 1.26 (for `make dev-ca`) and Python 3, from the repository root on this branch.

1. Start a fresh stack:
   ```sh
   make dev-ca
   docker compose down -v
   docker compose up --build --wait
   docker compose exec nava-core wget -qO- http://127.0.0.1:9090/readyz   # prints: ok
   ```
2. Seed the two tenants and walk every criterion:
   ```sh
   python -I docs/acceptance/s3-tenant-signin/accept.py
   ```
   It ends with `result: PASS`. It prints decoded token claims and each API response, but never a token.
3. Optional, by hand: get Alice's token and call the API.
   ```sh
   TOKEN=$(curl -s "${KEYCLOAK_PUBLIC_URL:-http://localhost:18081}/realms/nava/protocol/openid-connect/token" \
     -d grant_type=password -d client_id=nava-web -d scope=openid \
     -d username=alice@tenant-a.test -d password=dev-only-alice | python -c 'import sys,json;print(json.load(sys.stdin)["access_token"])')
   curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:18080/api/v1/me
   curl -s -H "Authorization: Bearer $TOKEN" -H 'X-Tenant-Id: 00000000-0000-4000-8000-0000000000b1' \
     'http://127.0.0.1:18080/api/v1/me?tenant_id=00000000-0000-4000-8000-0000000000b1'
   ```
   Both calls show `"tenant_id":"00000000-0000-4000-8000-0000000000a1"` (tenant A). The platform trusts only the pinned issuer `${KEYCLOAK_PUBLIC_URL}/realms/nava` (default `http://localhost:18081/realms/nava`), which Keycloak puts in every token whatever the request Host.
4. Clean up: `docker compose down -v`.

Seeded users, all with dev-only passwords from `deploy/compose/keycloak/import/realm-nava.json`: `alice@tenant-a.test` and `admin@tenant-a.test` (tenant A), `bob@tenant-b.test` (tenant B). The script also creates `carol@two-orgs.test` (in both organizations) and `dave@no-org.test` (in none).

There is no UI in this story, so the evidence is recorded requests and responses, not screenshots.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | `GET /api/v1/me` as Alice shows tenant A and her groups | ✅ | Transcript, criterion 1 |
| 2 | A token naming two organizations, or none, gets 401 `tenant_unresolved` | ✅ | Transcript, criterion 2. Logged reasons `organization count` and `no organization` |
| 3 | An ID token used as an access token, and a token from an issuer not on the allow-list, are rejected | ✅ | Transcript, criterion 3. Logged reasons `audience` and `issuer`. Unit test `TestIDTokenRejected` and the `typ` cases in `internal/identity/keycloak/keycloak_test.go` |
| step 3 | Tenant B's id in a header, the query string or the body still returns tenant A | ✅ | Transcript, story step 3 |
| iso | A tenant B user gets tenant B and never sees tenant A data | ✅ | Transcript, cross-tenant check |

The full transcript of the run is [`s3-tenant-signin/transcript.txt`](s3-tenant-signin/transcript.txt). The script that produced it is [`s3-tenant-signin/accept.py`](s3-tenant-signin/accept.py).

### Seed

Tenants A (`...a1`) and B (`...b1`) are `active`. Each is bound to its Keycloak organization under the pinned issuer `${KEYCLOAK_PUBLIC_URL}/realms/nava`. The tenant id is the organization id, as the realm file fixes. Alice, admin-a and Bob are principals with their Keycloak user ids and groups: Alice is in `/t/<A>/ou/eng/platform`, Bob in `/t/<B>/ou/eng/platform`, and admin-a in none.

In production, onboarding (T15) and group sync (T25) write these rows. Neither runs as a compose service yet, so the script writes them as the bootstrap superuser. See Known limits.

### Criterion 1

Alice's access token has `iss ${KEYCLOAK_PUBLIC_URL}/realms/nava`, `aud nava-api`, `typ Bearer`, `azp nava-web` and `organization {"tenant-a": {"id": "...a1"}}`.

```
GET /api/v1/me   (Alice)
200 {"groups":["4516c25a-..."],"idp_alias":"","tenant_id":"00000000-0000-4000-8000-0000000000a1","user_id":"764fee65-..."}
```

`groups` is exactly Alice's one group in the seed. As `admin@tenant-a.test`, the response is tenant A with `"groups":[]`.

### Story step 3: forged tenant, Alice's token

| Request carries tenant B's id in | Response |
|---|---|
| Headers `X-Tenant-Id`, `X-Nava-Tenant`, `X-Tenant` | 200, tenant A, Alice's groups |
| Query `?tenant_id=B&tenant=B` | 200, tenant A, Alice's groups |
| JSON body `{"tenant_id": B}` on `GET /api/v1/me` | 200, tenant A, Alice's groups |
| `baggage: nava.tenant_id=B,tenant_id=B` | 200, tenant A, Alice's groups |
| `POST /api/v1/records` body `{"tenant_id": B}` | 404 `not_found` (no records route yet). Tenant B not reached |

No response contains tenant B's id.

### Criterion 2

| Token | `organization` claim | Response | Logged reason |
|---|---|---|---|
| carol, in both organizations, `scope=openid organization:*` | `tenant-a` and `tenant-b` | 401 `tenant_unresolved`, `WWW-Authenticate: Bearer` | `organization count` |
| carol, default scope | absent (Keycloak omits it for a member of several organizations) | 401 `tenant_unresolved` | `no organization` |
| dave, in no organization | absent | 401 `tenant_unresolved` | `no organization` |

The reason was matched to each response through its request id in the `nava-core` log. The log line carries only the fixed reason, never a token or a claim.

### Criterion 3

| Token | Claims | Response | Logged reason |
|---|---|---|---|
| Alice's **ID token** as the bearer | `typ ID`, `aud nava-web` | 401 `invalid_token` | `audience` |
| Alice's access token from the same realm, fetched under another `Host` | the pinned `iss` (TICK-160), a valid tenant A token | 200, tenant A | n/a |
| Keycloak admin token, realm `master` | `iss ${KEYCLOAK_PUBLIC_URL}/realms/master` | 401 `invalid_token` | `issuer` |

On the stack, Keycloak gives an ID token the client's audience. The audience check therefore refuses it before the `typ` check runs. The `typ` check alone is covered by `TestIDTokenRejected` and the table cases `typ ID (an ID token)`, `typ Refresh` and `no typ` in `internal/identity/keycloak/keycloak_test.go`, where a token with `aud nava-api` and `typ ID` is still refused.

## Isolation check

- **Bob (tenant B)** gets `200` with `tenant_id ...b1` and only his own group. His group id and Alice's are different, and neither response contains the other tenant's id.
- **Bob, forging tenant A** in a header, the query string and the body at once: still tenant B, with his own group.
- **The groups have the same name in both tenants** (`ou/eng/platform`). Each user sees only their own tenant's copy.
- The database side of the wall (no runtime role sees another tenant's principals or groups) was proven in S2 by `make isolation` and is not repeated here.

## Not included

- The full CI-5 adversarial identity suite (T19) moved to S5 (TICK-83), because its foreign-IdP case needs group sync (T25).
- Group paths in `GET /api/v1/me` (`group_paths`, ADR-002) are not in this story. `groups` holds group ids only.
- Signing in through a tenant's own IdP (the brokered `idp` claim) is covered by `TestBrokeredTokenCarriesOrganizationAndIdp` in the identity stack test, not by this walk-through. Every response above shows `"idp_alias":""`.
- No UI: sign-in through a browser arrives with the web app.

## Known limits

- **There is no seed command yet.** Bindings and principals are written as the bootstrap superuser by `accept.py`. In production, onboarding (T15) and group sync (T25) write them. Until group sync runs in compose, a Keycloak user who has not been synced gets 401 `tenant_unresolved` (`principal not synced`), even when their organization is bound. That fails closed. A seed target for the local stack would make "Try it" one command; if you want one, say so and the Technical Lead will file it.
- **The trusted issuer is pinned to `KEYCLOAK_PUBLIC_URL`** (TICK-160). Every token carries it whatever Host fetched it, so no `Host` header is needed. A token from another realm is still refused.
- The fresh stack took 220 s to become healthy on this host. That is above S1's 180 s CI gate, which this story does not change.
