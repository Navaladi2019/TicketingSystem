# S3 regression evidence (TICK-45 kit), main @ 4392c26, 2026-10-28, stack nava-regr

Command (kit step 2, verbatim): `NAVA_CORE_PORT=28080 KEYCLOAK_PORT=28081 python -I docs/acceptance/s3-tenant-signin/accept.py`
Transcripts (no tokens; `grep -c eyJ` = 0): `s3-accept-run1-verbatim.txt`, `s3-accept-run2-pinned-issuer-binding.txt`.

## Run 1, verbatim: result FAIL (10 of 16), cause = kit drift, not a product regression
`accept.py` hard-codes `ISSUER = "http://keycloak:8080/realms/nava"` (pre-TICK-160) for the bindings it seeds and relies on `Host: keycloak:8080` to get that issuer. Since TICK-160 Keycloak pins `iss` to `KC_HOSTNAME` (`KEYCLOAK_PUBLIC_URL`), so every token says `iss http://localhost:28081/realms/nava`; no binding matches:
```
alice -> 401 {"type":"urn:nava:problem:tenant_unresolved",...}   logged: tenant_unresolved: no tenant binding
```
Same failure on default ports (pinned issuer `http://localhost:18081/realms/nava` != `keycloak:8080`).

## Run 2: bindings re-seeded with the pinned issuer, one group sync, script re-run: 15 of 16 PASS
```
DELETE FROM tenant_identity_binding WHERE issuer='http://keycloak:8080/realms/nava';   -> DELETE 2
INSERT ... (A|B, 'http://localhost:28081/realms/nava', org A|B, '{idp-tenant-a|b}')    -> INSERT 0 2
docker compose run --rm --no-deps nava-groupsync --once  -> "nava-groupsync --once done","tenants":2,"failed":0
```

| # | Criterion | Result | Excerpt (run 2) |
|---|---|---|---|
| 1 | Alice /me -> tenant A + groups | PASS | `200 {"group_paths":[...,{"path":"/service-desk"}],"groups":[...],"tenant_id":"...a1"}`; admin-a `200 ... "groups":[]` tenant A |
| 2 | two orgs / no org -> 401 tenant_unresolved | PASS | 3/3 `401 WWW-Authenticate: Bearer {"code":"tenant_unresolved"}`; logged `organization count`, `no organization` x2 |
| 3 | ID token, untrusted issuer rejected | PASS (by hand for the issuer half) | ID token `401 invalid_token`, logged `audience`; master-realm token (`iss http://localhost:28081/realms/master`) `401 invalid_token`, logged `issuer`. Script case "same realm fetched without Host header" FAILs: it now returns **200**, because the issuer is pinned, so that token is a valid nava token. The case is obsolete, not a hole |
| step 3 | forged B in header/query/body/baggage -> A | PASS | 5/5 `200 ... tenant_id ...a1`; `POST /api/v1/records` `404 not_found` |
| iso | Bob -> B only | PASS | `200 tenant ...b1`, groups disjoint from Alice's; forging A -> still B |

Fix needed (kit/tooling, owner: whoever owns the S3 kit): update `accept.py` to bind `${KEYCLOAK_PUBLIC_URL}/realms/nava`, drop the `Host:` trick, and replace the "no Host header" untrusted-issuer case with a different-realm token. The kit's "Try it" step 3 (`Host: keycloak:8080`, port 18081) is stale for the same reason.
Side effect seen: accept.py seeds `/ou/eng/platform` as a row in `groups`; group sync models `/ou/...` as `org_units`, so the live worker deletes that row (`groups_deleted:1`) and the script re-creates it on the next run. Harmless, but the script's seed no longer matches the data model.
