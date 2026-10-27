# Acceptance kit: S5, group changes and deprovisioning take effect within 10 minutes

Story: TICK-47. Plan: PLAN-001 (`docs/delivery/plan-platform-foundation.md`), tasks T19, T25 and T26. Design: ADR-001 Annex A.1 and Annex B (CI-5), D-P9.
Plain-language explainer: [`s5-group-changes-within-10-minutes-explained.html`](s5-group-changes-within-10-minutes-explained.html) (open it in a browser).
Verified on 2026-10-27 by the Quality, Testing & CI Advisor, on a fresh compose stack built from the story branch `TICK-47/group-changes-within-10-minutes` (`cd715c4`) with `origin/main` (`a665f40`) merged in locally. The merge was not pushed. Its conflicts (`.gitignore`, `Makefile`, `docs/security/accepted-risks.md`) were resolved by keeping both sides. Evidence: [`s5-evidence/`](s5-evidence/).

**Status: NOT READY for the human.** Two fix tasks are open under the story:

| Task | Severity | What fails |
|---|---|---|
| TICK-263 | **Bug** (security, appears on merge) | With `main` merged in, a tenant A user linked to tenant B's IdP is **accepted** as tenant A. CI-5.10 fails. `main`'s TICK-223 (PR #147) drops unbound links before the resolver can refuse them. The story branch alone is correct. |
| TICK-264 | **Spec drift** | Story step 1 cannot be done on the seeded stack. Alice's seeded `eng/platform` is an org unit, not a group, so `GET /api/v1/me` shows `"groups":[]`. `/me` also lists group ids, not names. |

Every other criterion passes, below. After the two fixes merge, QA re-runs criteria 3 and 4 and the "Try it" walk, then hands the story over.

## What you can now do

- When an administrator removes someone from a team in Keycloak, the platform stops treating them as a member within 10 minutes, even on a sign-in they already hold.
- When an administrator disables someone, their existing sign-in stops working within 10 minutes, and they cannot sign in again.
- Each customer's copy of its users and teams holds only its own people.
- A set of automated attacks on sign-in (forged or swapped tokens, tampering with the login page, another customer's login system) runs on every PR and is refused.

## Try it (about 15 minutes, most of it waiting)

Needs Docker and Go 1.26, from a fresh clone of the story branch. The seeded users and their dev-only passwords are in `deploy/compose/keycloak/import/realm-nava.json`. The Keycloak admin is `admin`, with the dev-only password in `compose.yaml`.

1. Start the stack:
   ```sh
   make dev-ca
   docker compose build && docker compose up --wait
   ```
2. Seed tenants A and B in the platform database. The stack has no onboarding API yet. The two ids are the Keycloak organisations' fixed ids.
   ```sh
   docker compose exec -T postgres sh -c 'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava' <<'SQL'
   INSERT INTO tenants (id, state) VALUES ('00000000-0000-4000-8000-0000000000a1','active'), ('00000000-0000-4000-8000-0000000000b1','active');
   INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES
    ('00000000-0000-4000-8000-0000000000a1','http://keycloak:8080/realms/nava','00000000-0000-4000-8000-0000000000a1','{idp-tenant-a}'),
    ('00000000-0000-4000-8000-0000000000b1','http://keycloak:8080/realms/nava','00000000-0000-4000-8000-0000000000b1','{idp-tenant-b}');
   SQL
   ```
3. Run group sync. The stack has no `nava-groupsync` service yet, so run it once from the repository. It reaches PgBouncer and Keycloak on the stack's `internal` network:
   ```sh
   CGO_ENABLED=0 GOOS=linux go build -o bin/linux/nava-groupsync ./cmd/nava-groupsync
   docker run --rm --network nava_internal -v "$PWD/bin/linux:/b:ro" --entrypoint /b/nava-groupsync \
     -e NAVA_DATABASE_URL='postgres://nava_groupsync:dev-only-groupsync@pgbouncer:5432/nava?sslmode=disable' \
     -e NAVA_KEYCLOAK_URL=http://keycloak:8080 -e NAVA_GROUPSYNC_CLIENT_ID=nava-platform \
     -e NAVA_GROUPSYNC_CLIENT_SECRET=dev-only-platform-secret postgres:16.10 --once
   ```
   Drop `--once` and add `-d --name nava-groupsync` to run it as the worker, every 5 minutes.
4. Sign in as Alice and call the API. The token must come through the host name `keycloak:8080`, because that is the only issuer nava-core trusts:
   ```sh
   TOKEN=$(curl -s -H 'Host: keycloak:8080' http://127.0.0.1:18081/realms/nava/protocol/openid-connect/token \
     -d grant_type=password -d client_id=nava-web -d scope=openid \
     -d username=alice@tenant-a.test -d password=<Alice's dev-only password> | jq -r .access_token)
   curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:18080/api/v1/me
   ```
   You should see `"tenant_id":"00000000-0000-4000-8000-0000000000a1"`. `"groups"` is empty: see TICK-264. Until it is fixed, open the admin console at `http://127.0.0.1:18081/admin` (realm `nava`). Create a group `service-desk` under `t › 00000000-…-0000000000a1`, add Alice to it, run step 3 again, and call `/me` again. You see one group id.
5. In the admin console, remove Alice from `service-desk`. Call `/me` with the **same** `$TOKEN`. After the next sync (run step 3, or wait for the worker), `"groups":[]`.
6. Get a fresh `$TOKEN` (step 4), then disable Alice in the admin console. A new sign-in now fails with `Account disabled`. After the next sync, the same `$TOKEN` gets `401 tenant_unresolved`.
7. **Re-enable Alice**, then run the attack suite. Its tests sign in as Alice, so it fails at once while she is disabled.
   ```sh
   make identity-attacks
   ```
   Open `ci5-report.md`: 14 rows, all `PASS`. On the story branch with `main` merged in, CI-5.10 fails until TICK-263 is fixed.
8. Clean up with `docker compose down -v`.

If another stack already runs on the machine, set `COMPOSE_PROJECT_NAME`, `NAVA_CORE_PORT=0` and `KEYCLOAK_PORT=0`, and read the ports from `docker compose port`. The network is then `<project>_internal`, and `make identity-attacks` needs `KEYCLOAK_PORT=<port>`.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | Group removal is reflected within 10 minutes (test asserts the bound) | ✅ 180 s on a live worker | [04](s5-evidence/04-group-removal-timed.txt); T26 path `groupRemovalWithinBound` |
| 2 | A disabled user's already-issued token stops working within 10 minutes | ✅ 199 s on a live worker | [05](s5-evidence/05-disable-timed.txt); T26 path `disabledUserWithinBound` |
| 3 | Synced identity tables never show another tenant's users | ✅ | [06](s5-evidence/06-boundaries.txt), isolation section |
| 4 | `kc.org` tamper, ID token as access token, foreign-IdP user all rejected | ❌ **with `main` merged**: CI-5.10 fails (TICK-263). ✅ on the story branch alone, 14/14 | [ci5 merged](s5-evidence/ci5-report-merged-main.md), [ci5 story](s5-evidence/ci5-report-story-branch.md), [08](s5-evidence/08-foreign-link-merged.txt), [09](s5-evidence/09-foreign-link-story-branch.txt) |
| 5 | Tenant B's admin cannot obtain a token the API accepts for tenant A | ✅ | CI-5.13 PASS in both reports; Bob's `kc.org` tamper by hand in [06](s5-evidence/06-boundaries.txt) |
| 6 | Tenant admin gets 403 on `/api/v1/operator/*`; operator token gets 401 on a tenant endpoint | 401 half ✅. 403 half: **404 today**, as documented (S4 not on `main`, TODO-010) | [06](s5-evidence/06-boundaries.txt) |
| Try it | Story step 1, "`/me` shows group service-desk", on the seeded stack | ❌ (TICK-264) | [02](s5-evidence/02-first-sync.txt) |

### Criteria 1 and 2 evidence: the 10-minute bound on a live worker

`nava-groupsync` ran as a worker with its default interval (5 minutes, the maximum it accepts), against the real Keycloak. Each change was made through the Keycloak admin API on a freshly issued token (lifespan 300 s). That token then called `/me` every 15 s.

| Change | Made at (UTC) | Worker cycle | Same token's `/me` | Token still unexpired? |
|---|---|---|---|---|
| Remove Alice from `service-desk` | 07:44:51 | 07:47:44 | `200`, `"groups":[]` at +180 s | yes, it expired at 07:49:51 |
| Disable Alice | 07:49:34 | 07:52:44 | `401 tenant_unresolved` at +199 s | yes, it expired at 07:54:32 |

A new sign-in right after the disable failed with `invalid_grant: Account disabled`. Bob (tenant B) got `200` throughout. The worst case is one full interval (5 min), then one full token lifespan (5 min): 10 minutes, which is what `withinBound()` in the T26 test asserts against `config.MaxGroupsyncInterval` and the realm's `accessTokenLifespan`. The T26 tests replace Keycloak with a fixed snapshot. This walk is the end-to-end check against the real Keycloak.

### Criterion 3 evidence: isolation of the synced tables

The check ran as `nava_app` through PgBouncer, after a real sync of both tenants:

| `app.tenant_id` | Rows visible in `principals`, `groups`, `org_units` | Rows of the other tenant |
|---|---|---|
| none / `''` | 0 | 0 |
| tenant A | 2 principals (admin, alice), 1 group, 3 org units, all tenant A | 0 |
| tenant B | 1 principal (bob), 3 org units, all tenant B | 0 |

Sync logged `ignored_outside_subtree=0` for both tenants. The CI isolation suite (`internal/groupsync` isolation tests) covers a group outside `/t/<tenant>/`.

### Criterion 4 evidence and the merge bug (TICK-263)

| Run | Code | Result |
|---|---|---|
| `make identity-attacks` | story branch alone (`cd715c4`) | **14/14 PASS** |
| `make identity-attacks` | story branch + `main` | 13 PASS, **CI-5.10 FAIL**: `group sync did not record the link: "[]"` |
| By hand: link Alice to `idp-tenant-b`, sync, `/me` | story branch alone | `federated_links` holds the link; `/me` **401** |
| Same | story branch + `main` | `federated_links = []`; old and new token `/me` **200 as tenant A** |

By hand, an ID token on `/me` got `401 invalid_token`. Bob asking for `organization:tenant-a` got a token with no organisation and `401 tenant_unresolved`. `organization:*` still gave him only tenant B. The CI-5.06 negative fixture (26.1.2) was not re-run. It needs no stack, and the T19 review records it.

### Criterion 6 evidence

| Token | Request | Result |
|---|---|---|
| `admin@tenant-a.test` (`tenant-admin`) | `GET /api/v1/me` | 200, tenant A |
| `admin@tenant-a.test` | `GET /api/v1/operator/tenants` | **404** (route not on this branch, TODO-010) |
| `operator@nava.test` via `nava-operator-cli` | `GET /api/v1/me` | **401** `invalid_token` |
| `bob@tenant-b.test` via `nava-operator-cli` | sign-in | refused, "Access denied" |

## Isolation check

Covered by criteria 3, 4 and 5: tenant B's user and admin cannot get a tenant A answer from the API, and each tenant's identity rows are invisible to the other. The one cross-tenant failure is TICK-263, which appears only on the merge with `main`.

## Not included

- An onboarding API that creates tenants and bindings. Seeded by hand above (S3/T15 onboarding lives elsewhere).
- A `nava-groupsync` service in `compose.yaml`. Run by hand above.
- Operator routes `/api/v1/operator/*` (story S4).

## Known limits

- **The 403 on operator routes is unproven until S4 and S5 meet on `main`** (TODO-010). Today the route is 404, and the test is built to require 403 once it exists.
- **The story's "Try it" order breaks step 5.** Step 4 disables Alice, and `make identity-attacks` signs in as Alice, so it fails at once (`Account disabled`) unless she is re-enabled. The kit says so. A clearer test message would help (preference).
- **Leftover Docker networks.** On a shared dev host, leftover `nava-it-net-*` networks from killed integration runs can exhaust Docker's address pools. `up` then fails with `all predefined address pools have been fully subnetted`. Removing the networks with no containers fixed it.
- **Startup took 221 s** on a host running about 45 other test containers.
