"""S6 acceptance driver (TICK-48): seed two tenants on a running stack, then
walk criteria 1, 2 and 4 of the story and print a transcript.

Run from the repository root, against a fresh `docker compose up --wait`:

    python -I docs/acceptance/s6-api-ground-rules/accept.py > transcript.txt

Environment: COMPOSE_PROJECT_NAME (as for docker compose), NAVA_CORE_PORT
(default 18080), KEYCLOAK_PORT (default 18081), KEYCLOAK_PUBLIC_URL (default
http://localhost:18081). Only the dev-only passwords from the realm file are
used. Tokens are never printed or put on a command line: the operator token
reaches the onboarding call through the environment of a throwaway curl
container on the stack's `internal` network (nava-onboard publishes no port).

Exit status 0 means every check passed.
"""

import concurrent.futures
import json
import os
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

CORE = "http://127.0.0.1:%s" % os.environ.get("NAVA_CORE_PORT", "18080")
KC = "http://127.0.0.1:%s" % os.environ.get("KEYCLOAK_PORT", "18081")
PUBLIC_URL = os.environ.get("KEYCLOAK_PUBLIC_URL", "http://localhost:18081").rstrip("/")
ISSUER = PUBLIC_URL + "/realms/nava"
PROJECT = os.environ.get("COMPOSE_PROJECT_NAME") or os.path.basename(os.getcwd()).lower()
CURL_IMAGE = "curlimages/curl:8.10.1"

TENANT_A = "00000000-0000-4000-8000-0000000000a1"
TENANT_B = "00000000-0000-4000-8000-0000000000b1"
USERS = {
    "alice@tenant-a.test": ("dev-only-alice", TENANT_A),
    "bob@tenant-b.test": ("dev-only-bob", TENANT_B),
}
OPERATOR = ("operator@nava.test", "dev-only-operator")
PROBLEM = "application/problem+json"

failures = []


def out(*a):
    print(*a, flush=True)


def check(name, ok, detail=""):
    out("  %s %s" % ("PASS" if ok else "FAIL", name))
    if not ok:
        failures.append(name)
    if detail:
        out("       " + detail.replace("\n", "\n       "))


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
            return r.status, {k.lower(): v for k, v in r.headers.items()}, r.read().decode()
    except urllib.error.HTTPError as e:
        return e.code, {k.lower(): v for k, v in e.headers.items()}, e.read().decode()


def token(user, password, client="nava-web"):
    st, _, b = http("POST", "%s/realms/nava/protocol/openid-connect/token" % KC,
                    form={"grant_type": "password", "client_id": client, "username": user,
                          "password": password, "scope": "openid"})
    if st != 200:
        raise SystemExit("token for %s: %d %s" % (user, st, b))
    return json.loads(b)["access_token"]


def compose(*args, stdin=None):
    r = subprocess.run(["docker", "compose", *args], input=stdin, capture_output=True, text=True)
    if r.returncode != 0:
        raise SystemExit("docker compose %s: %s" % (" ".join(args), r.stderr))
    return r.stdout.strip()


def psql(sql):
    return compose("exec", "-T", "postgres", "sh", "-c",
                   "PGPASSWORD=$(cat /run/pg_superuser_password) psql -v ON_ERROR_STOP=1 -qAt -U pgadmin -d nava",
                   stdin=sql)


def registry():
    """code -> status from api/errors.yaml (flat list, read without a YAML dependency)."""
    codes, code = {}, None
    with open("api/errors.yaml", encoding="utf-8") as f:
        for line in f:
            s = line.strip()
            if s.startswith("- code:"):
                code = s.split(":", 1)[1].strip()
            elif s.startswith("status:") and code:
                codes[code] = int(s.split(":", 1)[1])
                code = None
    return codes


REGISTRY = registry()


def problem_ok(st, hd, b, want_code):
    """The response is problem+json, its code is registered with this status,
    and status, title, instance and code are in the body."""
    try:
        v = json.loads(b)
    except ValueError:
        return False, "body is not JSON: %r" % b[:200]
    ct = hd.get("content-type", "")
    code = v.get("code")
    ok = (ct.split(";")[0].strip() == PROBLEM and code == want_code and REGISTRY.get(code) == st
          and v.get("status") == st and v.get("title") and v.get("instance"))
    return ok, "-> %d Content-Type: %s\n%s" % (st, ct, json.dumps(v, sort_keys=True))


# ---------------------------------------------------------------- onboarding

def onboard(op_token, key, body, extra_headers=()):
    """POST /api/v1/operator/tenants on nava-onboard's own listener, from a
    throwaway curl container on the internal network. The token goes through
    the environment, never argv. Returns (status, headers, body)."""
    hdrs = " ".join("-H %s" % sh_quote(h) for h in extra_headers)
    script = ('curl -sS -i -m 120 -X POST -H "Authorization: Bearer $NAVA_OPERATOR_TOKEN" '
              '-H "Idempotency-Key: $1" -H "Content-Type: application/json" %s --data-binary "$2" '
              'http://nava-onboard:8081/api/v1/operator/tenants' % hdrs)
    env = dict(os.environ, NAVA_OPERATOR_TOKEN=op_token, MSYS_NO_PATHCONV="1")
    r = subprocess.run(["docker", "run", "--rm", "-e", "NAVA_OPERATOR_TOKEN", "--network", PROJECT + "_internal",
                        "--entrypoint", "sh", CURL_IMAGE, "-c", script, "sh", key, body],
                       capture_output=True, text=True, env=env)
    raw = r.stdout.replace("\r\n", "\n")
    head, _, resp = raw.partition("\n\n")
    lines = head.split("\n")
    if not lines or not lines[0].startswith("HTTP/"):
        raise SystemExit("no HTTP response from nava-onboard: %s %s" % (raw[:300], r.stderr[:300]))
    st = int(lines[0].split()[1])
    hd = {}
    for ln in lines[1:]:
        k, _, v = ln.partition(":")
        hd[k.strip().lower()] = v.strip()
    return st, hd, resp


def sh_quote(s):
    return "'" + s.replace("'", "'\\''") + "'"


def onboard_body(domain, display):
    # Public address literals, as the onboarding stack tests use: nava-onboard
    # refuses private hosts and resolves nothing on the internal network.
    return json.dumps({
        "admin_email": "admin@" + domain,
        "email_domains": [domain],
        "identity_provider": {
            "provider_id": "oidc", "display_name": display,
            "config": {
                "clientId": "nava-broker", "clientSecret": "dev-only-broker-secret",
                "clientAuthMethod": "client_secret_post",
                "authorizationUrl": "https://93.184.216.34/auth", "tokenUrl": "https://93.184.216.34/token",
                "jwksUrl": "https://93.184.216.34/jwks", "useJwksUrl": "true",
                "defaultScope": "openid email profile",
            },
        },
    }, sort_keys=True)


def counts(tenant):
    return psql("SELECT (SELECT count(*) FROM tenants) || ' tenants, ' || "
                "(SELECT count(*) FROM tenant_identity_binding) || ' bindings, ' || "
                "(SELECT count(*) FROM tenant_email_domain) || ' email domains, ' || "
                "(SELECT count(*) FROM pending_invite) || ' invites; this tenant: ' || "
                "coalesce((SELECT state FROM tenants WHERE id = '%s'), 'none');" % tenant)


# ---------------------------------------------------------------- seed

def seed():
    """Tenants A and B active and bound to their realm organizations (what
    onboarding writes; the realm export fixes their ids, so the seed writes
    the rows as the bootstrap superuser), then one group sync cycle, which
    syncs Alice and Bob as principals with their groups."""
    rows = []
    for t in (TENANT_A, TENANT_B):
        rows.append("INSERT INTO tenants (id, state) VALUES ('%s', 'active') ON CONFLICT DO NOTHING;" % t)
        rows.append("INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ('%s', '%s', '%s') "
                    "ON CONFLICT DO NOTHING;" % (t, ISSUER, t))
    psql("\n".join(rows))
    compose("run", "--rm", "--no-deps", "nava-groupsync", "--once")
    out("seeded:")
    out(psql("SELECT 'tenant ' || id || ' ' || state FROM tenants ORDER BY id;"))
    out(psql("SELECT 'principal ' || tenant_id || ' ' || email FROM principals "
             "WHERE email IN ('alice@tenant-a.test', 'bob@tenant-b.test') ORDER BY tenant_id;"))


def me(tok, method="GET", path="/api/v1/me", headers=None):
    h = {"Authorization": "Bearer " + tok} if tok else {}
    h.update(headers or {})
    return http(method, CORE + path, headers=h)


# ---------------------------------------------------------------- criteria

def criterion1(alice, bob, op):
    out("\n## Criterion 1: every error is application/problem+json with a registered code")
    out("  registry api/errors.yaml: %d codes" % len(REGISTRY))

    st, hd, b = http("GET", CORE + "/api/v1/docs")
    check("GET /api/v1/docs serves the contract page", st == 200 and "text/html" in hd.get("content-type", "")
          and "/api/v1/openapi.yaml" in b, "-> %d Content-Type: %s, %d bytes" % (st, hd.get("content-type"), len(b)))
    st, hd, b = http("GET", CORE + "/api/v1/openapi.yaml")
    paths = [ln.strip().rstrip(":") for ln in b.splitlines() if ln.startswith("  /")]
    check("GET /api/v1/openapi.yaml is OpenAPI 3.1 and declares /me and onboarding",
          st == 200 and any(ln.startswith("openapi: 3.1") for ln in b.splitlines()) and "/api/v1/me" in paths
          and "/api/v1/operator/tenants" in paths,
          "-> %d Content-Type: %s, %s, paths %s" % (st, hd.get("content-type"), next((ln for ln in b.splitlines() if ln.startswith("openapi:")), "no openapi line"), paths))
    st, hd, b = http("GET", CORE + "/api/v1/openapi.yaml")
    with open("api/openapi.bundled.yaml", encoding="utf-8") as f:
        same = f.read().replace("\r\n", "\n") == b.replace("\r\n", "\n")
    check("the served contract is byte-identical to api/openapi.bundled.yaml on main", same)

    cases = [
        ("no token on GET /api/v1/me", lambda: me(None), "invalid_token"),
        ("garbage bearer token", lambda: me("not-a-jwt"), "invalid_token"),
        ("unknown route GET /api/v1/nope", lambda: me(alice, path="/api/v1/nope"), "not_found"),
        ("DELETE /api/v1/me", lambda: me(alice, method="DELETE"), "method_not_allowed"),
        ("tenant ingress POST /api/v1/operator/tenants (nava-core)",
         lambda: me(alice, method="POST", path="/api/v1/operator/tenants",
                    headers={"Idempotency-Key": "s6-x", "Content-Type": "application/json"}), "not_implemented"),
    ]
    for name, call, code in cases:
        ok, detail = problem_ok(*call(), code)
        check("%s -> %s" % (name, code), ok, detail)
    st, hd, b = me(None)
    check("401 carries WWW-Authenticate: Bearer", st == 401 and hd.get("www-authenticate") == "Bearer",
          "-> WWW-Authenticate: %s" % hd.get("www-authenticate"))

    domain = "s6-err-%s.test" % uuid.uuid4().hex[:8]
    ok, detail = problem_ok(*onboard(alice, "s6-tenant-token", onboard_body(domain, "S6")), "forbidden")
    check("nava-onboard, tenant token -> forbidden", ok, detail)
    ok, detail = problem_ok(*onboard(op, "s6-bad-body", json.dumps({"admin_email": 7})), "validation_failed")
    check("nava-onboard, body not matching the contract -> validation_failed", ok, detail)
    bad = json.loads(onboard_body(domain, "S6"))
    bad["identity_provider"]["config"]["tokenUrl"] = "https://127.0.0.1/token"
    ok, detail = problem_ok(*onboard(op, "s6-bad-idp", json.dumps(bad)), "invalid_idp_config")
    check("nava-onboard, IdP URL on loopback -> invalid_idp_config", ok, detail)
    # 409 idempotency_key_reuse and 429 rate_limited are checked the same way
    # under criteria 2 and 4.


def criterion2(op):
    out("\n## Criterion 2: same Idempotency-Key + same body returns the original response; "
        "same key + different body returns 409")
    domain = "s6-onb-%s.test" % uuid.uuid4().hex[:8]
    key = "s6-acceptance-%s" % uuid.uuid4().hex[:8]
    body = onboard_body(domain, "S6 tenant C")
    out("  Idempotency-Key: %s, domain %s" % (key, domain))
    out("  before: " + counts("00000000-0000-0000-0000-000000000000"))

    st1, hd1, b1 = onboard(op, key, body)
    v1 = json.loads(b1) if b1.strip().startswith("{") else {}
    tenant = v1.get("tenant_id", "")
    check("first call -> 201 with the tenant id, state active", st1 == 201 and v1.get("state") == "active" and tenant,
          "-> %d %s" % (st1, b1.strip()))
    after_first = counts(tenant)
    out("  after first call: " + after_first)

    st2, hd2, b2 = onboard(op, key, body)
    check("replay, same key and same body -> the original body, byte for byte",
          st2 in (200, 201) and b2 == b1, "-> %d %s" % (st2, b2.strip()))
    after_replay = counts(tenant)
    check("replay creates nothing", after_replay == after_first, "after replay: " + after_replay)

    other = onboard_body(domain, "S6 tenant C, renamed")
    st3, hd3, b3 = onboard(op, key, other)
    ok, detail = problem_ok(st3, hd3, b3, "idempotency_key_reuse")
    check("same key, different body -> 409 idempotency_key_reuse (problem+json)", ok, detail)
    after_reuse = counts(tenant)
    check("the refused call creates nothing", after_reuse == after_first, "after 409: " + after_reuse)

    st4, _, b4 = onboard(op, key, body)
    check("the original body still replays after the 409", st4 in (200, 201) and b4 == b1, "-> %d %s" % (st4, b4.strip()))
    return tenant


def criterion4(alice, bob):
    out("\n## Criterion 4: tenant A gets 429 while tenant B's calls succeed")
    out("  limit on this stack: NAVA_RATE_LIMIT_RPS=50, NAVA_RATE_LIMIT_BURST=100 (compose defaults)")
    stop = threading.Event()
    b_results = []

    throttled = threading.Event()  # set once tenant A has had its first 429

    def tenant_b():
        while not stop.is_set():
            st, _, b = me(bob)
            b_results.append((st, b, throttled.is_set()))
            time.sleep(0.05)

    def flood_a():
        # One worker keeps calling until the flood window closes, so tenant A
        # stays over its limit for the whole window whatever this host's speed.
        res = []
        while not stop.is_set():
            st, hd, b = me(alice)
            if st == 429:
                throttled.set()
            res.append((st, hd, b))
        return res

    FLOOD_SECONDS = 4
    tb = threading.Thread(target=tenant_b)
    tb.start()
    t0 = time.time()
    with concurrent.futures.ThreadPoolExecutor(max_workers=48) as pool:
        futures = [pool.submit(flood_a) for _ in range(48)]
        time.sleep(FLOOD_SECONDS)
        stop.set()
        a_results = [r for f in futures for r in f.result()]
    elapsed = time.time() - t0
    tb.join()

    first_429 = next((r for r in a_results if r[0] == 429), None)
    a_status = {}
    for st, _, _ in a_results:
        a_status[st] = a_status.get(st, 0) + 1
    out("  tenant A: %d calls in %.1f s (%.0f/s) from 48 workers: %s" % (
        len(a_results), elapsed, len(a_results) / elapsed, ", ".join("%d x %d" % (n, s) for s, n in sorted(a_status.items()))))
    check("tenant A is throttled", first_429 is not None and a_status.get(200, 0) > 0)
    if first_429:
        ok, detail = problem_ok(*first_429, "rate_limited")
        ra = first_429[1].get("retry-after", "")
        check("429 is problem+json rate_limited with Retry-After", ok and ra.isdigit() and int(ra) >= 1,
              detail + "\nRetry-After: %s" % ra)
        check("the 429 body names no tenant", TENANT_A not in first_429[2] and TENANT_B not in first_429[2])

    b_codes = {}
    for st, _, _ in b_results:
        b_codes[st] = b_codes.get(st, 0) + 1
    during = sum(1 for _, _, t in b_results if t)
    b_ok = all(st == 200 and json.loads(b).get("tenant_id") == TENANT_B for st, b, _ in b_results)
    check("tenant B's %d calls during the flood all succeed with tenant B (%d of them after tenant A's first 429)"
          % (len(b_results), during), during >= 5 and b_ok,
          "tenant B: " + ", ".join("%d x %d" % (n, s) for s, n in sorted(b_codes.items())))

    time.sleep(3)
    st, _, b = me(alice)
    check("tenant A recovers after Retry-After", st == 200 and json.loads(b).get("tenant_id") == TENANT_A,
          "-> %d %s" % (st, b.strip()))


def isolation(alice, bob, op, tenant_c):
    out("\n## Cross-tenant checks")
    st, _, b = me(bob, headers={"X-Tenant-Id": TENANT_A})
    check("bob naming tenant A in X-Tenant-Id still gets tenant B", st == 200 and json.loads(b).get("tenant_id") == TENANT_B
          and TENANT_A not in b, "-> %d %s" % (st, b.strip()))
    # Flood tenant B now, and tenant A must be unaffected: the limit is per
    # tenant in both directions, not a side effect of order.
    statuses = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=48) as pool:
        for st, _, _ in pool.map(lambda _: me(bob), range(400)):
            statuses[st] = statuses.get(st, 0) + 1
    st, _, b = me(alice)
    check("after flooding tenant B (%s), tenant A's call succeeds" %
          ", ".join("%d x %d" % (n, s) for s, n in sorted(statuses.items())),
          statuses.get(429, 0) > 0 and st == 200 and json.loads(b).get("tenant_id") == TENANT_A, "-> %d %s" % (st, b.strip()))
    check("onboarding's tenant C is not tenant A or B", tenant_c not in (TENANT_A, TENANT_B, ""), "tenant C = " + tenant_c)


def main():
    st, _, _ = http("GET", CORE + "/api/v1/me")
    out("baseline: GET /api/v1/me with no token -> %d" % st)
    out("stack: compose project %s, main at %s" % (PROJECT, subprocess.run(
        ["git", "rev-parse", "--short", "HEAD"], capture_output=True, text=True).stdout.strip()))
    seed()
    alice = token("alice@tenant-a.test", USERS["alice@tenant-a.test"][0])
    bob = token("bob@tenant-b.test", USERS["bob@tenant-b.test"][0])
    op = token(*OPERATOR, client="nava-operator-cli")
    for name, tok, tenant in (("alice", alice, TENANT_A), ("bob", bob, TENANT_B)):
        st, _, b = me(tok)
        check("%s -> GET /api/v1/me is tenant %s" % (name, tenant[-2:].upper()),
              st == 200 and json.loads(b).get("tenant_id") == tenant, "-> %d %s" % (st, b.strip()))

    criterion1(alice, bob, op)
    tenant_c = criterion2(op)
    time.sleep(3)  # let tenant A's bucket refill after criterion 1
    criterion4(alice, bob)
    time.sleep(3)
    isolation(alice, bob, op, tenant_c)

    out("\nresult: %s" % ("PASS" if not failures else "FAIL: " + "; ".join(failures)))
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
