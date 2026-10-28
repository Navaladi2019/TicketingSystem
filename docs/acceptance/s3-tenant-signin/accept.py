"""S3 acceptance driver (TICK-45): seed two tenants on a running stack, then
walk every acceptance criterion against GET /api/v1/me and print a transcript.

Run from the repository root, against a fresh `docker compose up --wait`:

    python -I docs/acceptance/s3-tenant-signin/accept.py > transcript.txt

Environment: COMPOSE_PROJECT_NAME (as for docker compose), NAVA_CORE_PORT
(default 18080), KEYCLOAK_PORT (default 18081), KEYCLOAK_ADMIN_PASSWORD
(default the dev-only value from compose.yaml). Only dev-only seeded
passwords from the realm file are used. Tokens are never printed: the
transcript shows decoded claims that matter (iss, typ, azp, aud,
organization) and the API's response.

Exit status 0 means every check passed.
"""

import base64
import json
import os
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request

CORE = "http://127.0.0.1:%s" % os.environ.get("NAVA_CORE_PORT", "18080")
KC = "http://127.0.0.1:%s" % os.environ.get("KEYCLOAK_PORT", "18081")
# nava-core trusts exactly http://keycloak:8080/realms/nava. Keycloak names
# the issuer after the request host (KC_HOSTNAME_STRICT=false), so a token
# fetched with this Host header is the one the platform accepts; the same
# request without it gets issuer http://127.0.0.1:<port>/realms/nava, which
# is not on the allow-list.
TRUSTED_HOST = "keycloak:8080"
ADMIN_PASSWORD = os.environ.get("KEYCLOAK_ADMIN_PASSWORD", "dev-only-admin")

TENANT_A = "00000000-0000-4000-8000-0000000000a1"
TENANT_B = "00000000-0000-4000-8000-0000000000b1"
ISSUER = "http://keycloak:8080/realms/nava"

USERS = {
    "alice@tenant-a.test": ("dev-only-alice", TENANT_A),
    "admin@tenant-a.test": ("dev-only-admin-a", TENANT_A),
    "bob@tenant-b.test": ("dev-only-bob", TENANT_B),
}
# Created by this script for criterion 2 (dev-only passwords).
TWO_ORGS = ("carol@two-orgs.test", "dev-only-carol")
NO_ORG = ("dave@no-org.test", "dev-only-dave")

failures = []


def out(*a):
    print(*a, flush=True)


def http(method, url, headers=None, body=None, form=None):
    h = dict(headers or {})
    data = None
    if form is not None:
        data = urllib.parse.urlencode(form).encode()
        h["Content-Type"] = "application/x-www-form-urlencoded"
    elif body is not None:
        data = body if isinstance(body, bytes) else json.dumps(body).encode()
        h.setdefault("Content-Type", "application/json")
    req = urllib.request.Request(url, data=data, headers=h, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return r.status, dict(r.headers), r.read().decode()
    except urllib.error.HTTPError as e:
        return e.code, dict(e.headers), e.read().decode()


def claims(tok):
    p = tok.split(".")[1]
    return json.loads(base64.urlsafe_b64decode(p + "=" * (-len(p) % 4)))


def show_claims(tok):
    c = claims(tok)
    keep = {k: c.get(k) for k in ("iss", "typ", "azp", "aud", "organization", "idp") if k in c}
    return json.dumps(keep, sort_keys=True)


def token(user, password, host=TRUSTED_HOST, realm="nava", client="nava-web", scope="openid"):
    st, _, b = http("POST", "%s/realms/%s/protocol/openid-connect/token" % (KC, realm),
                    headers={"Host": host} if host else None,
                    form={"grant_type": "password", "client_id": client, "username": user,
                          "password": password, "scope": scope})
    if st != 200:
        raise SystemExit("token for %s: %d %s" % (user, st, b))
    return json.loads(b)


def admin():
    t = token("admin", ADMIN_PASSWORD, realm="master", client="admin-cli", host=None)["access_token"]
    return {"Authorization": "Bearer " + t}


def kc_admin(method, path, body=None):
    st, hd, b = http(method, KC + "/admin/realms/nava" + path, headers=admin(), body=body)
    if st >= 300 and st != 409:
        raise SystemExit("admin %s %s: %d %s" % (method, path, st, b))
    return st, hd, (json.loads(b) if b.strip().startswith(("[", "{")) else b)


def user_id(username):
    _, _, us = kc_admin("GET", "/users?exact=true&username=" + urllib.parse.quote(username))
    return us[0]["id"]


def ensure_user(username, password):
    kc_admin("POST", "/users", {"username": username, "email": username, "enabled": True,
                                "emailVerified": True, "firstName": "S3", "lastName": "Acceptance",
                                "credentials": [{"type": "password", "value": password, "temporary": False}]})
    return user_id(username)


def psql(sql):
    cmd = ["docker", "compose", "exec", "-T", "postgres", "sh", "-c",
           "PGPASSWORD=$(cat /run/pg_superuser_password) psql -v ON_ERROR_STOP=1 -qAt -U pgadmin -d nava"]
    r = subprocess.run(cmd, input=sql, capture_output=True, text=True)
    if r.returncode != 0:
        raise SystemExit("psql: " + r.stderr)
    return r.stdout.strip()


def seed():
    """Tenants A and B active, each bound to its organization, and the
    seeded users synced as principals with their Keycloak groups. This is
    what onboarding (T15) and group sync (T25) write; neither runs as a
    compose service yet, so the fixture writes it as the bootstrap superuser."""
    rows = []
    for t in (TENANT_A, TENANT_B):
        rows.append("INSERT INTO tenants (id, state) VALUES ('%s', 'active') ON CONFLICT DO NOTHING;" % t)
        rows.append("INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ('%s', '%s', '%s') ON CONFLICT DO NOTHING;"
                    % (t, ISSUER, t))
    for username, (_, tenant) in USERS.items():
        uid = user_id(username)
        rows.append("INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at) "
                    "VALUES ('%s', gen_random_uuid(), '%s', '%s', true, '[]', now()) ON CONFLICT DO NOTHING;"
                    % (tenant, uid, username))
        _, _, groups = kc_admin("GET", "/users/%s/groups?briefRepresentation=false" % uid)
        for g in groups:
            if not g["path"].startswith("/t/%s/" % tenant):
                continue
            rows.append("INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ('%s', gen_random_uuid(), '%s', '%s') ON CONFLICT DO NOTHING;"
                        % (tenant, g["id"], g["path"]))
            rows.append("INSERT INTO group_members (tenant_id, group_id, principal_id) "
                        "SELECT '%s', g.id, p.id FROM groups g, principals p WHERE g.tenant_id = '%s' AND p.tenant_id = '%s' "
                        "AND g.keycloak_group_id = '%s' AND p.keycloak_user_id = '%s' ON CONFLICT DO NOTHING;"
                        % (tenant, tenant, tenant, g["id"], uid))
    psql("\n".join(rows))
    out("seeded:")
    out(psql("SELECT 'tenant ' || id || ' ' || state FROM tenants ORDER BY id;"))
    out(psql("SELECT 'binding ' || tenant_id || ' ' || issuer || ' org=' || org_id FROM tenant_identity_binding ORDER BY tenant_id;"))
    out(psql("SELECT 'principal ' || tenant_id || ' ' || email || ' id=' || id FROM principals ORDER BY tenant_id, email;"))
    out(psql("SELECT 'group ' || g.tenant_id || ' ' || g.path || ' id=' || g.id || ' members=' || count(m.*) "
             "FROM groups g LEFT JOIN group_members m ON m.tenant_id = g.tenant_id AND m.group_id = g.id "
             "GROUP BY g.tenant_id, g.path, g.id ORDER BY g.tenant_id;"))


def me(tok, path="/api/v1/me", headers=None, body=None, method="GET"):
    h = {"Authorization": "Bearer " + tok}
    h.update(headers or {})
    return http(method, CORE + path, headers=h, body=body)


def check(name, ok, detail):
    out("  %s %s" % ("PASS" if ok else "FAIL", name))
    if not ok:
        failures.append(name)
    out("       " + detail)


def expect_me(name, tok, tenant, groups=None, **kw):
    st, _, b = me(tok, **kw)
    try:
        v = json.loads(b)
    except ValueError:
        v = {}
    ok = st == 200 and v.get("tenant_id") == tenant
    if groups is not None:
        ok = ok and sorted(v.get("groups", [])) == sorted(groups)
    other = TENANT_B if tenant == TENANT_A else TENANT_A
    ok = ok and other not in b
    check(name, ok, "-> %d %s" % (st, b.strip()))
    return v


def refusal_reason(request_urn):
    """The reason nava-core logged for one refused request, found by its
    request id. The log line carries only the resolver's fixed reason."""
    rid = request_urn.rsplit(":", 1)[-1]
    r = subprocess.run(["docker", "compose", "logs", "--no-color", "nava-core"], capture_output=True, text=True)
    for line in r.stdout.splitlines():
        if "authentication refused" in line and rid in line:
            return json.loads(line[line.index("{"):]).get("reason", "")
    return ""


def expect_401(name, tok, code, reason):
    st, hd, b = me(tok)
    try:
        v = json.loads(b)
    except ValueError:
        v = {}
    www = hd.get("WWW-Authenticate") or hd.get("Www-Authenticate")
    got = refusal_reason(v.get("instance", ""))
    ok = st == 401 and v.get("code") == code and www == "Bearer" and got.endswith(": " + reason)
    check(name, ok, "-> %d WWW-Authenticate: %s %s\n       logged reason: %s" % (st, www, b.strip(), got))


def groups_of(tenant, email):
    s = psql("SELECT string_agg(m.group_id::text, ',' ORDER BY m.group_id) FROM group_members m JOIN principals p "
             "ON p.tenant_id = m.tenant_id AND p.id = m.principal_id WHERE p.tenant_id = '%s' AND p.email = '%s';" % (tenant, email))
    return [g for g in s.split(",") if g]


def main():
    st, _, b = http("GET", CORE + "/api/v1/me")
    out("baseline: GET /api/v1/me with no token -> %d %s" % (st, b.strip()))
    seed()
    alice_groups = groups_of(TENANT_A, "alice@tenant-a.test")
    bob_groups = groups_of(TENANT_B, "bob@tenant-b.test")

    alice = token("alice@tenant-a.test", "dev-only-alice")
    bob = token("bob@tenant-b.test", "dev-only-bob")
    admin_a = token("admin@tenant-a.test", "dev-only-admin-a")

    out("\n## Criterion 1: GET /api/v1/me as Alice shows tenant A and her groups")
    out("  alice access token claims: " + show_claims(alice["access_token"]))
    expect_me("alice -> tenant A, her one group", alice["access_token"], TENANT_A, alice_groups)
    expect_me("admin@tenant-a -> tenant A, no groups", admin_a["access_token"], TENANT_A, [])

    out("\n## Story step 3: tenant B's id in a header, the query string or the body still returns tenant A")
    a = alice["access_token"]
    expect_me("header X-Tenant-Id / X-Nava-Tenant: B", a, TENANT_A, alice_groups,
              headers={"X-Tenant-Id": TENANT_B, "X-Nava-Tenant": TENANT_B, "X-Tenant": TENANT_B})
    expect_me("query ?tenant_id=B&tenant=B", a, TENANT_A, alice_groups,
              path="/api/v1/me?tenant_id=%s&tenant=%s" % (TENANT_B, TENANT_B))
    expect_me("JSON body {tenant_id: B} on GET", a, TENANT_A, alice_groups, body={"tenant_id": TENANT_B})
    expect_me("baggage nava.tenant_id=B", a, TENANT_A, alice_groups,
              headers={"baggage": "nava.tenant_id=%s,tenant_id=%s" % (TENANT_B, TENANT_B)})
    st, _, b = me(a, path="/api/v1/records", method="POST", body={"tenant_id": TENANT_B})
    check("POST /api/v1/records with body tenant_id B does not reach B", st != 401 and TENANT_B not in b,
          "-> %d %s" % (st, b.strip()))

    out("\n## Criterion 2: a token naming two organizations, or none, gets 401 tenant_unresolved")
    carol = ensure_user(*TWO_ORGS)
    for org in (TENANT_A, TENANT_B):
        kc_admin("POST", "/organizations/%s/members" % org, json.dumps(carol).encode())
    ensure_user(*NO_ORG)
    # A user in two organizations gets no organization claim with the default
    # scope; organization:* asks Keycloak to name every one of them.
    two = token(*TWO_ORGS, scope="openid organization:*")["access_token"]
    two_default = token(*TWO_ORGS)["access_token"]
    none = token(*NO_ORG)["access_token"]
    out("  two-org token claims: " + show_claims(two))
    out("  two-org token claims, default scope: " + show_claims(two_default))
    out("  no-org token claims:  " + show_claims(none))
    expect_401("two organizations -> 401 tenant_unresolved", two, "tenant_unresolved", "organization count")
    expect_401("two organizations, default scope (claim omitted) -> 401 tenant_unresolved", two_default,
               "tenant_unresolved", "no organization")
    expect_401("no organization -> 401 tenant_unresolved", none, "tenant_unresolved", "no organization")

    out("\n## Criterion 3: an ID token used as an access token, and a token from an issuer not on the allow-list, are rejected")
    out("  alice ID token claims: " + show_claims(alice["id_token"]))
    expect_401("ID token as bearer -> 401 invalid_token", alice["id_token"], "invalid_token", "audience")
    foreign = token("alice@tenant-a.test", "dev-only-alice", host=None)["access_token"]
    out("  untrusted-issuer token claims: " + show_claims(foreign))
    expect_401("same realm, issuer http://127.0.0.1:<port>/realms/nava (not allow-listed) -> 401 invalid_token",
               foreign, "invalid_token", "issuer")
    master = token("admin", ADMIN_PASSWORD, realm="master", client="admin-cli", host=TRUSTED_HOST)["access_token"]
    out("  master-realm token claims: " + show_claims(master))
    expect_401("realm master, issuer http://keycloak:8080/realms/master -> 401 invalid_token", master, "invalid_token", "issuer")

    out("\n## Cross-tenant: a tenant B user gets tenant B and never sees tenant A data")
    b_tok = bob["access_token"]
    v = expect_me("bob -> tenant B, his one group", b_tok, TENANT_B, bob_groups)
    check("bob's groups and alice's groups are disjoint", not set(v.get("groups", [])) & set(alice_groups),
          "bob=%s alice=%s" % (v.get("groups"), alice_groups))
    expect_me("bob with header/query/body naming A -> still tenant B", b_tok, TENANT_B, bob_groups,
              path="/api/v1/me?tenant_id=" + TENANT_A, headers={"X-Tenant-Id": TENANT_A}, body={"tenant_id": TENANT_A})

    out("\nresult: %s" % ("PASS" if not failures else "FAIL: " + "; ".join(failures)))
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
