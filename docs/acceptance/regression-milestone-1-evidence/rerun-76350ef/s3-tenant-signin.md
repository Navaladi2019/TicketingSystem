# S3 regression re-run (TICK-45 kit), main @ 76350ef, 2026-10-28, stack nava-regr2

Proves fix #270 (TICK-318: `accept.py` derives the issuer from `KEYCLOAK_PUBLIC_URL`).

Command, kit step 2 verbatim, with the stack's ports in the environment as the script documents (`NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `KEYCLOAK_PUBLIC_URL`, `COMPOSE_PROJECT_NAME`):
```
python -I docs/acceptance/s3-tenant-signin/accept.py
```
No re-seeding, no binding edits: the script seeded the bindings itself with `http://localhost:28081/realms/nava`.

- Attempt 1 aborted before any check: `token for admin: 400 {"error":"unauthorized_client","error_description":"Unexpected error when authenticating client"}`. Cause: Keycloak's stale DB pool after the S1 Postgres drill (see `s1-platform-foundation.md`), not the kit and not #270. Cleared after 6 failed requests.
- Attempt 2, same command, nothing changed: **exit 0, `result: PASS`, 16 of 16 PASS.** Transcript: `s3-accept-verbatim.txt` (no tokens, `grep eyJ` = 0).

| # | Criterion | Result | Evidence (transcript) |
|---|---|---|---|
| 1 | Alice /me -> tenant A and her groups | PASS | `iss http://localhost:28081/realms/nava`, `200 ... "tenant_id":"...a1"`; admin-a `200 "groups":[]` |
| 2 | two orgs / none -> 401 tenant_unresolved | PASS | 3/3 `401 WWW-Authenticate: Bearer` `tenant_unresolved`; logged `organization count`, `no organization` x2 |
| 3 | ID token and untrusted issuer rejected | PASS | ID token `401 invalid_token` (logged `audience`); master realm `401 invalid_token` (logged `issuer`); other-Host token keeps pinned iss -> 200 tenant A (the rewritten case) |
| step 3 | tenant B in header/query/body/baggage -> A | PASS | 4/4 `200` tenant A; `POST /api/v1/records` `404 not_found` |
| iso | Bob -> B only | PASS | `200` tenant `...b1`, groups disjoint; forging A -> still B |

Fix #270 verified: run 1 of the previous regression (10/16, every token `tenant_unresolved: no tenant binding`) is gone.

Kit drift, cosmetic: the script labels say "her one group" but Alice now shows two (`/ou/eng/platform` seeded by the script plus `/service-desk` from the realm, synced by the live worker); the live group sync later deletes the script's `/ou/eng/platform` row (it is an org unit in the data model), as noted last run.
