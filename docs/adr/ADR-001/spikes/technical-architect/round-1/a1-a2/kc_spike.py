"""Spike A1/A2: Keycloak Organizations claim shape and organization scope selection.
Usage: python kc_spike.py http://localhost:18080
Throwaway spike code. Dev-mode Keycloak, throwaway admin password only.
"""
import sys, json, base64, re, html
import requests

BASE = sys.argv[1].rstrip("/")
ADMIN_PW = "<throwaway-dev-admin-password>"
R = "spike"
CLIENT = "nava-api"
REDIRECT = "http://localhost:9/cb"
USERPW = "<throwaway-user-password>"


def tok():
    r = requests.post(f"{BASE}/realms/master/protocol/openid-connect/token",
                      data={"grant_type": "password", "client_id": "admin-cli",
                            "username": "admin", "password": ADMIN_PW})
    r.raise_for_status()
    return {"Authorization": "Bearer " + r.json()["access_token"]}


H = tok()
A = f"{BASE}/admin/realms"


def call(method, path, **kw):
    hdr = {**H, **kw.pop("headers", {})}
    r = requests.request(method, A + path, headers=hdr, **kw)
    return r


def jwt_payload(t):
    p = t.split(".")[1]
    p += "=" * (-len(p) % 4)
    return json.loads(base64.urlsafe_b64decode(p))


print("== server version:", requests.get(f"{BASE}/admin/serverinfo", headers=H).json()["systemInfo"]["version"])

# mapper type properties (authoritative list of config knobs)
si = requests.get(f"{BASE}/admin/serverinfo", headers=H).json()
for m in si["protocolMapperTypes"]["openid-connect"]:
    if "organization" in m["id"]:
        print("== mapper type", m["id"], [(p["name"], p.get("defaultValue")) for p in m["properties"]])

call("DELETE", f"/{R}")
r = call("POST", "", json={"realm": R, "enabled": True, "organizationsEnabled": True})
print("create realm", r.status_code)

r = call("POST", f"/{R}/clients", json={
    "clientId": CLIENT, "publicClient": True, "directAccessGrantsEnabled": True,
    "standardFlowEnabled": True, "redirectUris": [REDIRECT]})
print("create client", r.status_code)
cid = call("GET", f"/{R}/clients", params={"clientId": CLIENT}).json()[0]["id"]

scopes = call("GET", f"/{R}/client-scopes").json()
org_scope = [s for s in scopes if s["name"] == "organization"]
print("organization client scope exists:", bool(org_scope))
osid = org_scope[0]["id"]
print("organization scope mappers:", json.dumps(org_scope[0].get("protocolMappers"), indent=1))
r = call("PUT", f"/{R}/clients/{cid}/optional-client-scopes/{osid}")
print("attach org scope as optional", r.status_code)

orgs = {}
for alias in ["acme", "globex", "initech"]:
    r = call("POST", f"/{R}/organizations", json={
        "name": alias.title() + " Inc", "alias": alias, "enabled": True,
        "domains": [{"name": f"{alias}.example", "verified": False}],
        "attributes": {"tenant_hint": ["x"]}})
    print("create org", alias, r.status_code, r.text[:200])
for o in call("GET", f"/{R}/organizations").json():
    orgs[o["alias"]] = o["id"]
print("orgs:", orgs)

users = {}
for u, member_of in [("alice", ["acme"]), ("bob", ["acme", "globex"]), ("carol", [])]:
    r = call("POST", f"/{R}/users", json={
        "username": u, "enabled": True, "email": f"{u}@mail.example", "emailVerified": True,
        "firstName": u, "lastName": "x",
        "credentials": [{"type": "password", "value": USERPW, "temporary": False}]})
    uid = call("GET", f"/{R}/users", params={"username": u, "exact": "true"}).json()[0]["id"]
    users[u] = uid
    for a in member_of:
        r = call("POST", f"/{R}/organizations/{orgs[a]}/members", data=json.dumps(uid),
                 headers={"Content-Type": "application/json"})
        print("add", u, "->", a, r.status_code, r.text[:120])


def direct(user, scope):
    r = requests.post(f"{BASE}/realms/{R}/protocol/openid-connect/token", data={
        "grant_type": "password", "client_id": CLIENT, "username": user,
        "password": USERPW, "scope": scope})
    if r.status_code != 200:
        return f"HTTP {r.status_code} {r.text[:200]}"
    p = jwt_payload(r.json()["access_token"])
    return {"scope": p.get("scope"), "organization": p.get("organization", "<absent>")}


def authcode(user, scope, pick=None):
    """Browser flow with requests; reports org-selection page and resulting claim."""
    s = requests.Session()
    r = s.get(f"{BASE}/realms/{R}/protocol/openid-connect/auth", params={
        "client_id": CLIENT, "redirect_uri": REDIRECT, "response_type": "code",
        "scope": "openid " + scope}, allow_redirects=True)
    notes = []
    for _ in range(6):
        m = re.search(r'<form[^>]*action="([^"]+)"', r.text)
        if not m:
            return {"error": "no form", "status": r.status_code, "snippet": r.text[:300]}
        action = html.unescape(m.group(1))
        if 'name="username"' in r.text and 'name="password"' in r.text:
            data = {"username": user, "password": USERPW}
        elif 'name="username"' in r.text:
            data = {"username": user}  # identity-first (organization login)
        elif 'name="password"' in r.text:
            data = {"password": USERPW}
        elif "kc.org" in r.text or "organization" in r.text.lower():
            choices = re.findall(r"\['kc\.org'\]\.value = '([^']+)'", r.text)
            notes.append(f"org selection page shown, choices={choices}")
            data = {"kc.org": pick or (choices[0] if choices else "")}
        else:
            return {"error": "unknown page", "snippet": re.sub(r"\s+", " ", r.text)[:400]}
        for c in s.cookies:
            c.secure = False  # dev-mode http: requests withholds Secure cookies otherwise
        r = s.post(action, data=data, allow_redirects=False)
        loc = r.headers.get("Location", "")
        if loc.startswith(REDIRECT):
            code = re.search(r"code=([^&]+)", loc).group(1)
            t = requests.post(f"{BASE}/realms/{R}/protocol/openid-connect/token", data={
                "grant_type": "authorization_code", "client_id": CLIENT, "code": code,
                "redirect_uri": REDIRECT}).json()
            if "access_token" not in t:
                return {"notes": notes, "token_error": t}
            p = jwt_payload(t["access_token"])
            return {"notes": notes, "scope": p.get("scope"), "organization": p.get("organization", "<absent>")}
        if loc:
            r = s.get(loc if loc.startswith("http") else BASE + loc)
        if r.status_code >= 400:
            msg = re.search(r'kc-feedback-text">([^<]+)|<p class="instruction">([^<]+)', r.text)
            return {"notes": notes, "status": r.status_code, "message": msg.group(0) if msg else re.sub(r"\s+", " ", r.text)[-300:]}
    return {"notes": notes, "error": "loop"}


def matrix(label):
    print(f"\n==== {label}")
    for user, scope in [("alice", "organization"), ("bob", "organization"),
                        ("bob", "organization:acme"), ("bob", "organization:globex"),
                        ("bob", "organization:*"), ("alice", "organization:globex"),
                        ("carol", "organization"), ("alice", "")]:
        print(f"direct  {user:5} scope={scope!r:24} ->", json.dumps(direct(user, scope)))
    for user, scope, pick in [("bob", "organization", None), ("bob", "organization", "globex"),
                              ("bob", "organization", "initech"), ("bob", "organization", "nonexistent"),
                              ("bob", "organization:globex", None), ("alice", "organization:globex", None),
                              ("alice", "organization", None)]:
        print(f"browser {user:5} scope={scope!r:24} pick={pick} ->", json.dumps(authcode(user, scope, pick)))


matrix("default organization mapper config")

# try enabling id / attributes on the mapper if the knobs exist
osc = call("GET", f"/{R}/client-scopes/{osid}").json()
for pm in osc.get("protocolMappers", []):
    if pm["protocolMapper"] == "oidc-organization-membership-mapper":
        cfg = dict(pm["config"])
        cfg.update({"addOrganizationId": "true", "addOrganizationAttributes": "true",
                    "multivalued": "true", "jsonType.label": "JSON"})
        pm2 = {**pm, "config": cfg}
        r = call("PUT", f"/{R}/client-scopes/{osid}/protocol-mappers/models/{pm['id']}", json=pm2)
        print("\nupdate mapper with addOrganizationId/addOrganizationAttributes:", r.status_code, r.text[:200])
        print("mapper config now:", call("GET", f"/{R}/client-scopes/{osid}/protocol-mappers/models/{pm['id']}").json()["config"])

matrix("mapper with addOrganizationId=true, addOrganizationAttributes=true, JSON")

# alias mutability (matters for the A1 fallback)
o = call("GET", f"/{R}/organizations/{orgs['acme']}").json()
o2 = {**o, "alias": "acme-renamed"}
r = call("PUT", f"/{R}/organizations/{orgs['acme']}", json=o2)
after = call("GET", f"/{R}/organizations/{orgs['acme']}").json()
print("\nalias change attempt:", r.status_code, r.text[:200], "-> alias now", after.get("alias"))
o3 = {**after, "name": "Acme Renamed Ltd"}
r = call("PUT", f"/{R}/organizations/{orgs['acme']}", json=o3)
print("name change attempt:", r.status_code, "-> name now", call("GET", f"/{R}/organizations/{orgs['acme']}").json().get("name"))

# membership ground truth after the run (tampering must not have changed it)
for a, oid in orgs.items():
    ms = [m["username"] for m in call("GET", f"/{R}/organizations/{oid}/members").json()]
    print(f"members of {a}: {ms}")
