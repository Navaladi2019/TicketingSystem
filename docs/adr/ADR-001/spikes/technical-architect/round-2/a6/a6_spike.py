"""Spike A6: Keycloak 26.8.1 brokered login into an organization.
Q1 is a brokered email outside the org's domains rejected?
Q2 does the identity_provider session note reach the token as `idp`?
Q3 does create-if-unique fail on an existing email (no link)?
Usage: python a6_spike.py http://localhost:18081   (throwaway dev-mode Keycloak)
"""
import sys, json, base64, re, html
import requests

BASE = sys.argv[1].rstrip("/")
INTERNAL = "http://localhost:8080"  # Keycloak calling itself (back channel) inside the container
ADMIN_PW = "<throwaway-dev-admin-password>"
PW = "<throwaway-user-password>"
R, IDPR, CLIENT, REDIRECT = "spike", "idpa", "nava-api", "http://localhost:9/cb"


def admin_headers():
    r = requests.post(f"{BASE}/realms/master/protocol/openid-connect/token", data={
        "grant_type": "password", "client_id": "admin-cli", "username": "admin", "password": ADMIN_PW})
    r.raise_for_status()
    return {"Authorization": "Bearer " + r.json()["access_token"]}


H = admin_headers()


def call(method, path, **kw):
    hdr = {**H, **kw.pop("headers", {})}
    return requests.request(method, f"{BASE}/admin/realms{path}", headers=hdr, **kw)


def jwt(t):
    p = t.split(".")[1]
    p += "=" * (-len(p) % 4)
    return json.loads(base64.urlsafe_b64decode(p))


def user_id(realm, username):
    u = call("GET", f"/{realm}/users", params={"username": username, "exact": "true"}).json()
    return u[0]["id"] if u else None


print("== version", requests.get(f"{BASE}/admin/serverinfo", headers=H).json()["systemInfo"]["version"])
auths = requests.get(f"{BASE}/admin/realms/master/authentication/authenticator-providers", headers=H).json()
print("== idp/org authenticators:", sorted(a["id"] for a in auths if a["id"].startswith("idp-") or "organization" in a["id"]))

for realm in (R, IDPR):
    call("DELETE", f"/{realm}")

# ---- tenant A's IdP (a second realm) ----
call("POST", "", json={"realm": IDPR, "enabled": True})
call("POST", f"/{IDPR}/clients", json={
    "clientId": "spike-broker", "publicClient": False, "secret": "<throwaway-client-secret>",
    "standardFlowEnabled": True, "redirectUris": [f"{BASE}/realms/{R}/broker/*"]})
for u, email in [("mallory", "bob@globex.example"), ("eve", "eve@acme.example"),
                 ("xavier", "xavier@other.example"), ("eve2", "eve2@acme.example"),
                 ("xavier2", "xavier2@other.example"), ("mallory2", "bob@globex.example")]:
    if u == "mallory2":
        continue
    call("POST", f"/{IDPR}/users", json={"username": u, "email": email, "emailVerified": True, "enabled": True,
                                        "firstName": u, "lastName": "x",
                                        "credentials": [{"type": "password", "value": PW, "temporary": False}]})

# ---- platform realm ----
call("POST", "", json={"realm": R, "enabled": True, "organizationsEnabled": True})
call("POST", f"/{R}/clients", json={"clientId": CLIENT, "publicClient": True, "standardFlowEnabled": True,
                                    "redirectUris": [REDIRECT]})
cid = call("GET", f"/{R}/clients", params={"clientId": CLIENT}).json()[0]["id"]
osc = [s for s in call("GET", f"/{R}/client-scopes").json() if s["name"] == "organization"][0]
call("PUT", f"/{R}/clients/{cid}/default-client-scopes/{osc['id']}")
for pm in osc["protocolMappers"]:
    if pm["protocolMapper"] == "oidc-organization-membership-mapper":
        pm["config"].update({"addOrganizationId": "true", "jsonType.label": "JSON"})
        call("PUT", f"/{R}/client-scopes/{osc['id']}/protocol-mappers/models/{pm['id']}", json=pm)
r = call("POST", f"/{R}/clients/{cid}/protocol-mappers/models", json={
    "name": "idp", "protocol": "openid-connect", "protocolMapper": "oidc-usersessionmodel-note-mapper",
    "config": {"user.session.note": "identity_provider", "claim.name": "idp", "jsonType.label": "String",
               "access.token.claim": "true", "id.token.claim": "true"}})
print("idp session-note mapper:", r.status_code)

orgs = {}
for alias in ("acme", "globex"):
    call("POST", f"/{R}/organizations", json={"name": alias, "alias": alias, "enabled": True,
                                              "domains": [{"name": f"{alias}.example", "verified": True}]})
orgs = {o["alias"]: o["id"] for o in call("GET", f"/{R}/organizations").json()}
call("POST", f"/{R}/users", json={"username": "bob", "email": "bob@globex.example", "emailVerified": True,
                                  "enabled": True, "firstName": "bob", "lastName": "x",
                                  "credentials": [{"type": "password", "value": PW, "temporary": False}]})
bob = user_id(R, "bob")
call("POST", f"/{R}/organizations/{orgs['globex']}/members", data=json.dumps(bob),
     headers={"Content-Type": "application/json"})


def make_flow(alias, providers):
    call("POST", f"/{R}/authentication/flows", json={"alias": alias, "providerId": "basic-flow",
                                                     "topLevel": True, "builtIn": False})
    for p in providers:
        r = call("POST", f"/{R}/authentication/flows/{alias}/executions/execution", json={"provider": p})
        if r.status_code >= 300:
            print("  add execution", p, r.status_code, r.text[:150])
    for ex in call("GET", f"/{R}/authentication/flows/{alias}/executions").json():
        ex["requirement"] = "REQUIRED"
        call("PUT", f"/{R}/authentication/flows/{alias}/executions", json=ex)
    print("flow", alias, [(e["providerId"], e["requirement"]) for e in
                          call("GET", f"/{R}/authentication/flows/{alias}/executions").json()])


make_flow("create-if-unique-only", ["idp-create-user-if-unique"])
make_flow("create-if-unique-plus-org", ["idp-create-user-if-unique", "idp-add-organization-member"])

idp_cfg = {
    "alias": "idp-a", "providerId": "oidc", "enabled": True, "trustEmail": False,
    "firstBrokerLoginFlowAlias": "create-if-unique-only",
    "config": {
        "authorizationUrl": f"{BASE}/realms/{IDPR}/protocol/openid-connect/auth",
        "tokenUrl": f"{INTERNAL}/realms/{IDPR}/protocol/openid-connect/token",
        "disableUserInfo": "true",
        "clientId": "spike-broker", "clientSecret": "<throwaway-client-secret>", "clientAuthMethod": "client_secret_post",
        "defaultScope": "openid email profile", "syncMode": "IMPORT", "validateSignature": "false", "pkceEnabled": "false",
        "kc.org.domain": "acme.example", "kc.org.broker.redirect.mode.email-matches": "true",
    }}
r = call("POST", f"/{R}/identity-provider/instances", json=idp_cfg)
print("create idp-a:", r.status_code, r.text[:150])
r = call("POST", f"/{R}/organizations/{orgs['acme']}/identity-providers", data=json.dumps("idp-a"),
         headers={"Content-Type": "application/json"})
print("link idp-a -> acme:", r.status_code, r.text[:150])
print("idp-a config after link:", {k: v for k, v in call("GET", f"/{R}/identity-provider/instances/idp-a").json()["config"].items() if k.startswith("kc.org")})


def set_idp(**kw):
    cur = call("GET", f"/{R}/identity-provider/instances/idp-a").json()
    cfg = kw.pop("config", {})
    cur.update(kw)
    cur["config"].update(cfg)
    r = call("PUT", f"/{R}/identity-provider/instances/idp-a", json=cur)
    assert r.status_code < 300, r.text


def unsecure(s):
    for c in s.cookies:
        c.secure = False


def follow(s, r):
    while r.status_code in (301, 302, 303) and not r.headers["Location"].startswith(REDIRECT):
        unsecure(s)
        loc = r.headers["Location"]
        r = s.get(loc if loc.startswith("http") else BASE + loc, allow_redirects=False)
    unsecure(s)
    return r


def brokered_login(user, extra=None):
    s = requests.Session()
    params = {"client_id": CLIENT, "redirect_uri": REDIRECT, "response_type": "code",
              "scope": "openid organization", "kc_idp_hint": "idp-a"}
    params.update(extra or {})
    r = follow(s, s.get(f"{BASE}/realms/{R}/protocol/openid-connect/auth", params=params, allow_redirects=False))
    pages = []
    for _ in range(8):
        unsecure(s)
        url = r.url
        if url.startswith(REDIRECT) or r.headers.get("Location", "").startswith(REDIRECT):
            break
        m = re.search(r'<form[^>]*action="([^"]+)"', r.text)
        title = re.search(r"<title>([^<]*)</title>", r.text)
        err = re.search(r'(?:kc-feedback-text|pf-m-danger|alert-error|instruction)[^>]*>\s*([^<]{3,200})', r.text)
        pages.append(f"{'idpa' if '/realms/idpa/' in url else 'spike'}:{title.group(1).strip() if title else '?'}"
                     + (f" [{err.group(1).strip()}]" if err else ""))
        if not m:
            return {"pages": pages, "result": "no form (terminal page)"}
        action = html.unescape(m.group(1))
        if 'name="password"' in r.text and "/realms/idpa/" in url:
            data = {"username": user, "password": PW}
        elif 'name="username"' in r.text and 'name="email"' in r.text:
            data = {"username": user, "email": re.search(r'name="email"[^>]*value="([^"]*)"', r.text).group(1),
                    "firstName": user, "lastName": "x"}  # review-profile page, submit unchanged
        else:
            return {"pages": pages, "result": "unexpected form", "fields": re.findall(r'name="([^"]+)"', r.text)[:10]}
        unsecure(s)
        r = follow(s, s.post(action, data=data, allow_redirects=False))
        if r.headers.get("Location", "").startswith(REDIRECT):
            break
    loc = r.headers.get("Location", r.url)
    code = re.search(r"[?&]code=([^&]+)", loc)
    if not code:
        e = re.search(r"[?&]error=([^&]+)", loc)
        return {"pages": pages, "result": f"no code ({e.group(1) if e else loc[:120]})"}
    t = requests.post(f"{BASE}/realms/{R}/protocol/openid-connect/token", data={
        "grant_type": "authorization_code", "client_id": CLIENT, "code": code.group(1), "redirect_uri": REDIRECT}).json()
    p = jwt(t["access_token"])
    return {"pages": pages, "result": "TOKEN", "sub": p.get("sub"), "email": p.get("email"),
            "organization": p.get("organization", "<absent>"), "idp": p.get("idp", "<absent>")}


def state(username):
    uid = user_id(R, username)
    if not uid:
        return f"{username}: no local user"
    fed = [f["identityProvider"] + ":" + f["userName"] for f in call("GET", f"/{R}/users/{uid}/federated-identity").json()]
    member = [a for a, oid in orgs.items() if any(m["id"] == uid for m in call("GET", f"/{R}/organizations/{oid}/members").json())]
    return f"{username}: links={fed} orgs={member}"


def case(label, user, local_name, **kw):
    print(f"\n-- {label}: idp-a user {user}")
    print("   ", json.dumps(brokered_login(user, kw.get("extra"))))
    for n in local_name:
        print("    state", state(n))


print("\n==== flow create-if-unique-only, trustEmail=OFF")
case("Q3 existing email (bob@globex.example)", "mallory", ["bob"])
case("Q2 in-domain eve@acme.example", "eve", ["eve"])
case("Q1 out-of-domain xavier@other.example", "xavier", ["xavier"])

print("\n==== flow create-if-unique + idp-add-organization-member, trustEmail=OFF")
set_idp(firstBrokerLoginFlowAlias="create-if-unique-plus-org")
case("in-domain eve2@acme.example", "eve2", ["eve2"])
case("Q1 out-of-domain xavier2@other.example", "xavier2", ["xavier2"])
case("Q1 out-of-domain, scope organization:acme", "xavier2", ["xavier2"], extra={"scope": "openid organization:acme"})

print("\n==== S3 premise: built-in 'first broker login' flow, trustEmail=ON")
set_idp(firstBrokerLoginFlowAlias="first broker login", trustEmail=True)
case("existing email (bob@globex.example)", "mallory", ["bob"])
