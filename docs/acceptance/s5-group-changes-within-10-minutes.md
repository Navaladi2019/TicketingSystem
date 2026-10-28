# Acceptance kit: S5, group changes and deprovisioning take effect within 10 minutes

Story: TICK-47. Plan: PLAN-001 (`docs/delivery/plan-platform-foundation.md`), tasks T19, T25 and T26. Design: ADR-001 Annex A.1 and Annex B (CI-5), D-P9.
Plain-language explainer: [`s5-group-changes-within-10-minutes-explained.html`](s5-group-changes-within-10-minutes-explained.html) (open it in a browser).
Verified on 2026-10-27 by the Quality, Testing & CI Advisor, on a fresh compose stack built from `TICK-47/group-changes-within-10-minutes` at `0135a08`. That commit already contains `main` (`8cb0d54`). Tenants A and B were seeded, and the walk used the real Keycloak and a real `nava-groupsync`. Evidence: [`s5-evidence/`](s5-evidence/).

**Status: READY for the human.** All six criteria pass. Criterion 6's 403 half is required by T26-6 since S4 is on `main` (TODO-010 Fixed).

History: the first run on 2026-10-27 (`cd715c4` with `main` merged locally) found two problems. TICK-263: once `main` was in, a user linked to another tenant's login provider was accepted. TICK-264: story step 1 could not be done on the seed. Both are fixed (TICK-263, TICK-264, TICK-268), and this run re-checks them.

## What you can now do

- When an administrator removes someone from a team in Keycloak, the platform stops treating them as a member within 10 minutes, even on a sign-in they already hold.
- When an administrator disables someone, they cannot sign in again, and the sign-in they already hold stops working within 10 minutes.
- Each customer's copy of its users and teams holds only its own people.
- A set of automated attacks on sign-in (forged or swapped tokens, tampering with the login page, another customer's login system) is refused, and the suite runs on every PR.

## Try it (about 15 minutes, most of it waiting)

Needs Docker, Go 1.26 and `jq`, from a fresh clone of the story branch. The seeded users and their dev-only passwords are in `deploy/compose/keycloak/import/realm-nava.json`. The Keycloak admin is `admin`, with the dev-only password in `compose.yaml`.

1. Start the stack:
   ```sh
   make dev-ca
   docker compose build && docker compose up --wait
   ```
2. Seed tenants A and B in the platform database. The stack has no onboarding API yet. The ids are the Keycloak organisations' fixed ids, and the issuer is the pinned `KEYCLOAK_PUBLIC_URL`.
   ```sh
   docker compose exec -T postgres sh -c 'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava' <<'SQL'
   INSERT INTO tenants (id, state) VALUES ('00000000-0000-4000-8000-0000000000a1','active'), ('00000000-0000-4000-8000-0000000000b1','active');
   INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES
    ('00000000-0000-4000-8000-0000000000a1','http://localhost:18081/realms/nava','00000000-0000-4000-8000-0000000000a1','{idp-tenant-a}'),
    ('00000000-0000-4000-8000-0000000000b1','http://localhost:18081/realms/nava','00000000-0000-4000-8000-0000000000b1','{idp-tenant-b}');
   SQL
   ```
3. Run group sync. The stack has no `nava-groupsync` service yet, so run it from the repository on the stack's `internal` network:
   ```sh
   CGO_ENABLED=0 GOOS=linux go build -o bin/linux/nava-groupsync ./cmd/nava-groupsync
   docker run --rm --network nava_internal -v "$PWD/bin/linux:/b:ro" --entrypoint /b/nava-groupsync \
     -e NAVA_DATABASE_URL='postgres://nava_groupsync:dev-only-groupsync@pgbouncer:5432/nava?sslmode=disable' \
     -e NAVA_KEYCLOAK_URL=http://keycloak:8080 -e NAVA_GROUPSYNC_CLIENT_ID=nava-platform \
     -e NAVA_GROUPSYNC_CLIENT_SECRET=dev-only-platform-secret postgres:16.10 --once
   ```
   To run it as the worker (every 5 minutes), replace `--rm` with `-d --name nava-groupsync` and drop `--once`.
4. **Story step 1.** Sign in as Alice and call the API:
   ```sh
   TOKEN=$(curl -s http://localhost:18081/realms/nava/protocol/openid-connect/token \
     -d grant_type=password -d client_id=nava-web -d scope=openid \
     -d username=alice@tenant-a.test -d password=<Alice's dev-only password> | jq -r .access_token)
   curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:18080/api/v1/me
   ```
   You should see tenant `…0000000000a1` and `"group_paths":[{"id":"…","path":"/service-desk"}]`.
5. **Story steps 2–3.** In the admin console (`http://localhost:18081/admin`, realm `nava`), go to Groups › `t` › `00000000-…-0000000000a1` › `service-desk` › Members and remove Alice. Call `/me` with the **same** `$TOKEN`. After the next sync (run step 3 again, or wait for the worker), `group_paths` and `groups` are empty. A token lasts 5 minutes, so get a new one (step 4) if it expired while you waited.
6. **Story step 4.** Get a fresh `$TOKEN`, then disable Alice (Users › alice › Enabled off). A new sign-in now fails with `Account disabled`. After the next sync, the same `$TOKEN` gets `401 tenant_unresolved`.
7. **Story step 5.** **Re-enable Alice first**: the attack suite signs in as her and fails at once while she is disabled. Then:
   ```sh
   make identity-attacks
   ```
   Open `ci5-report.md`: 14 rows, all `PASS`.
8. Clean up with `docker compose down -v`.

If another stack already uses ports 18080/18081, set `COMPOSE_PROJECT_NAME`, `NAVA_CORE_PORT`, `KEYCLOAK_PORT` and `KEYCLOAK_PUBLIC_URL=http://localhost:<KEYCLOAK_PORT>` together. Use that issuer in step 2, `<project>_internal` as the network in step 3, and `KEYCLOAK_PORT=<port>` for `make identity-attacks`.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | Group removal is reflected within 10 minutes (test asserts the bound) | ✅ 244 s on a live worker, same unexpired token | [04](s5-evidence/04-group-removal-timed.txt); T26 path `groupRemovalWithinBound` |
| 2 | A disabled user's already-issued token stops working within 10 minutes | ✅ 266 s on a live worker, same unexpired token | [05](s5-evidence/05-disable-timed.txt); T26 path `disabledUserWithinBound` |
| 3 | Synced identity tables never show another tenant's users | ✅ | [06](s5-evidence/06-boundaries.txt), isolation section |
| 4 | `kc.org` tamper, ID token as access token, foreign-IdP user all rejected | ✅ CI-5 14/14 locally, and CI-5.10 by hand | [ci5-report](s5-evidence/ci5-report.md), [08](s5-evidence/08-foreign-link-ci5-10.txt), [06](s5-evidence/06-boundaries.txt) |
| 5 | Tenant B's admin cannot obtain a token the API accepts for tenant A | ✅ | CI-5.13 PASS; Bob's `kc.org` tamper by hand in [06](s5-evidence/06-boundaries.txt) |
| 6 | Tenant admin gets 403 on `/api/v1/operator/*`; operator token gets 401 on a tenant endpoint | 401 half ✅. 403 half: required by T26-6 now S4 is on `main` (TODO-010 Fixed) | [06](s5-evidence/06-boundaries.txt) |
| Story steps 1–3 | `/me` shows `service-desk`; removing Alice in Keycloak removes it | ✅ | [01](s5-evidence/01-seed-and-step1.txt), [04](s5-evidence/04-group-removal-timed.txt) |

GitHub `ci` on the same tree: run [40553475037](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40553475037), all 8 jobs green, with the `ci5-report` artifact showing 14/14. It ran on the PR #211 head `b3e7518`, whose tree is identical to `0135a08`.

### Criteria 1 and 2 evidence: the 10-minute bound on a live worker

`nava-groupsync` ran as a worker with its default interval: 5 minutes, the most it accepts. Each change was made through the Keycloak admin API, after a token was freshly issued (lifespan 300 s). That token then called `/me` every 15 s.

| Change | Made at (UTC) | Worker cycle | Same token's `/me` | Token still unexpired? |
|---|---|---|---|---|
| Remove Alice from `service-desk` | 18:05:11 | 18:09:07 | `200`, `"group_paths":[]` at +244 s | yes, it expired at 18:10:10 |
| Disable Alice | 20:19:52 | 20:24:06 | `401 tenant_unresolved` at +266 s | yes, it expired at 20:24:49 |

A new sign-in right after the disable failed with `invalid_grant: Account disabled`. Bob (tenant B) kept `200` and his own `/service-desk` throughout. The worst case is one full interval (5 min) plus one token lifespan (5 min), which makes 10 minutes. `withinBound()` in the T26 test asserts that bound against `config.MaxGroupsyncInterval` and the realm's `accessTokenLifespan`. The T26 tests use a fixed directory snapshot, so this walk is the end-to-end check against the real Keycloak.

The worker logs `group sync applied` only for a tenant whose snapshot changed. Its `groupsync_lag_seconds` histogram counted 54 observations over 27 cycles in about 2 h 15 min, all under 5 s.

### Criterion 3 evidence: isolation of the synced tables

As `nava_app` through PgBouncer, after a real sync of both tenants:

| `app.tenant_id` | Rows visible in `principals`, `groups`, `org_units` | Rows of the other tenant |
|---|---|---|
| none / `''` | 0 | 0 |
| tenant A | 2 principals (admin, alice), `/service-desk`, 3 org units, all A | 0 |
| tenant B | 1 principal (bob), `/service-desk`, 3 org units, all B | 0 |

Both tenants have a group named `service-desk`. Each sees only its own, with a different id.

### Criterion 4 evidence: CI-5.10 on the merged tree (TICK-263)

| Step at `0135a08` | Result |
|---|---|
| Control: Alice, no links, `/me` | 200, tenant A |
| Link Alice to `idp-tenant-b` in Keycloak, then one sync | `federated_links = [{"identityProvider": "idp-tenant-b"}]`, an alias-only marker |
| Rows anywhere holding the foreign external id `ext-qa-s5` | 0 |
| Alice's token from before the link, `/me` | **401** `tenant_unresolved` |
| A new token, `/me` | **401** `tenant_unresolved` |
| Unlink, one sync, `/me` | 200, tenant A |

`make identity-attacks` at `0135a08` passed 14/14 in 135 s ([ci5-report](s5-evidence/ci5-report.md)). By hand: an ID token on `/me` got `401 invalid_token`, and Bob asking for `organization:tenant-a` got a token with no organisation, then `401`. Asking for `organization:*` still gave him only tenant B.

### Criterion 6 evidence

| Token | Request | Result |
|---|---|---|
| `admin@tenant-a.test` (`tenant-admin`) | `GET /api/v1/me` | 200, tenant A |
| `admin@tenant-a.test` | `GET /api/v1/operator/tenants` | **404**: the route is not on this branch (known limit) |
| `operator@nava.test` via `nava-operator-cli` | `GET /api/v1/me` | **401** `invalid_token` |
| `bob@tenant-b.test` via `nava-operator-cli` | sign-in | refused, "Access denied" |

## Isolation check

Tenant B's user and admin cannot get a tenant A answer from the API (criteria 4 and 5, CI-5.13). Each tenant's synced rows are invisible to the other (criterion 3). A tenant A user linked to tenant B's login provider is refused, and tenant B's external id never reaches tenant A's row (criterion 4).

## Not included

- An onboarding API that creates tenants and bindings: this kit seeds them by hand.
- A `nava-groupsync` service in `compose.yaml`: this kit runs it by hand.
- Operator routes `/api/v1/operator/*` (story S4).

## Known limits

- **The 403 on operator routes** is required by T26-6 now that S4 is on `main` (TODO-010 Fixed); the suite builds its handler with `server.WithOperator`.
- **The story's "Try it" order.** Step 4 disables Alice, and step 5's attack suite signs in as her. Re-enable her in between, as above.
- **A 5-minute token can expire while you wait for a sync** (step 5). Within the bound that is the same outcome, a refusal, but it hides which mechanism acted. Use `nava-groupsync --once` to see the sync effect itself.
- **Leftover Docker networks.** On a shared dev host, leftover `nava-it-net-*` networks from killed integration runs can exhaust Docker's address pools, and `up` fails with `all predefined address pools have been fully subnetted`. Removing the networks with no containers fixed it on 2026-10-27.
