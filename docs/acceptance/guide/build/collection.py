"""Builds the acceptance Postman collection and environment (TICK-287).

The collection is generated so every request carries the same problem-body and
JWT helpers without hand-copying them. Edit this file, then run
`python -I docs/acceptance/guide/build/build.py` (it calls build()).

Folders run in order and each is a scenario: get the token, call the endpoint,
then try the negative cases. Tokens are fetched by requests and kept in
collection variables; none is written in this file.
"""

import json
import uuid

A = "00000000-0000-4000-8000-0000000000a1"
B = "00000000-0000-4000-8000-0000000000b1"

# Shared helpers, prepended to every test script (Postman scripts share no scope).
PRELUDE = r"""
const N = {
  // RFC 9457 problem body as api/errors.yaml registers it.
  problem(code, status) {
    pm.test(`HTTP ${status}`, () => pm.response.to.have.status(status));
    pm.test(`problem+json, code ${code}`, () => {
      pm.expect(pm.response.headers.get('Content-Type') || '').to.include('application/problem+json');
      const b = pm.response.json();
      pm.expect(b).to.include.all.keys('type', 'title', 'status', 'code', 'detail', 'instance');
      pm.expect(b.code).to.eql(code);
      pm.expect(b.status).to.eql(status);
      pm.expect(b.type).to.eql('urn:nava:problem:' + code);
      pm.expect(N.REGISTERED).to.include(b.code);
    });
    pm.test('problem instance is this response\'s X-Request-Id', () => {
      const id = pm.response.headers.get('X-Request-Id');
      pm.expect(id, 'X-Request-Id header').to.be.a('string').and.not.empty;
      pm.expect(pm.response.json().instance).to.eql('urn:nava:request:' + id);
    });
  },
  REGISTERED: ['tenant_unresolved', 'invalid_token', 'idempotency_key_reuse', 'forbidden', 'invalid_idp_config',
    'domain_claimed', 'onboarding_conflict', 'idempotency_in_flight', 'invalid_cursor', 'rate_limited', 'not_found',
    'validation_failed', 'method_not_allowed', 'not_implemented', 'internal_error'],
  // Decodes a JWT payload (no verification: the server does that).
  jwt(t) {
    const p = String(t).split('.')[1].replace(/-/g, '+').replace(/_/g, '/');
    return JSON.parse(CryptoJS.enc.Base64.parse(p + '==='.slice((p.length + 3) % 4)).toString(CryptoJS.enc.Utf8));
  },
  // The response text never names the other tenant.
  noLeak(...needles) {
    pm.test('response names nothing of the other tenant', () => {
      const text = pm.response.text();
      needles.filter(Boolean).forEach(n => pm.expect(text).to.not.include(n));
    });
  },
  me(tenant) {
    pm.test('HTTP 200', () => pm.response.to.have.status(200));
    pm.test(`tenant_id is ${tenant}`, () => pm.expect(pm.response.json().tenant_id).to.eql(tenant));
    pm.test('Me has the contract fields', () => {
      const b = pm.response.json();
      pm.expect(b).to.include.all.keys('user_id', 'tenant_id', 'groups', 'idp_alias', 'group_paths');
      pm.expect(b.groups).to.be.an('array');
      pm.expect(b.group_paths).to.be.an('array');
      b.group_paths.forEach(g => {
        pm.expect(g.path, 'path relative to the tenant root').to.match(/^\//).and.not.include('/t/');
        pm.expect(b.groups).to.include(g.id);
      });
    });
  },
  paths() { return pm.response.json().group_paths.map(g => g.path); },
  v(name) { return pm.collectionVariables.get(name) || pm.environment.get(name); },
};
"""

# Fetches a Keycloak master-realm admin token into kc_admin_token before the
# request (its lifespan is 60 s, so every admin call gets a fresh one).
ADMIN_TOKEN_PRE = r"""
pm.sendRequest({
  url: pm.variables.replaceIn('{{keycloak_url}}/realms/master/protocol/openid-connect/token'),
  method: 'POST',
  header: { 'Content-Type': 'application/x-www-form-urlencoded' },
  body: { mode: 'urlencoded', urlencoded: [
    { key: 'grant_type', value: 'password' }, { key: 'client_id', value: 'admin-cli' },
    { key: 'username', value: pm.variables.replaceIn('{{kc_admin_user}}') },
    { key: 'password', value: pm.variables.replaceIn('{{kc_admin_password}}') }] }
}, (err, res) => {
  if (err || res.code !== 200) { console.error('Keycloak admin token failed', err || res.code); return; }
  pm.collectionVariables.set('kc_admin_token', res.json().access_token);
});
"""

# Pre-request wait for a polling request: none on the first attempt.
def poll_pre(ms_var="poll_interval_ms"):
    return (
        "const n = Number(pm.collectionVariables.get('poll_n') || 0);\n"
        f"if (n > 0) {{ setTimeout(() => {{}}, Number(pm.variables.get('{ms_var}') || 10000)); }}\n"
    )


def poll_test(done_expr, final_tests, since_var="change_at"):
    """Repeats the request (Collection Runner only) until done_expr holds or
    the poll limit passes, then runs final_tests once."""
    return (
        "const n = Number(pm.collectionVariables.get('poll_n') || 0);\n"
        f"const since = Number(pm.collectionVariables.get('{since_var}') || Date.now());\n"
        "const waited = (Date.now() - since) / 1000;\n"
        "const limit = Number(pm.variables.get('poll_limit_seconds') || 660);\n"
        f"const done = (() => {{ try {{ return {done_expr}; }} catch (e) {{ return false; }} }})();\n"
        "if (!done && waited < limit) {\n"
        "  pm.collectionVariables.set('poll_n', n + 1);\n"
        "  console.log(`${pm.info.requestName}: not yet after ${Math.round(waited)} s; waiting for the next group sync`);\n"
        "  pm.execution.setNextRequest(pm.info.requestName);\n"
        "} else {\n"
        "  pm.collectionVariables.set('poll_n', 0);\n"
        "  pm.test(`took ${Math.round(waited)} s, within the 10-minute bound`, () => pm.expect(waited).to.be.below(600));\n"
        + "  " + final_tests.replace("\n", "\n  ") + "\n"
        "}\n"
    )


def _url(raw):
    return raw


def req(name, method, url, *, desc="", headers=None, body=None, form=None, tests="", pre="", auth=None):
    """auth: None (no Authorization), or a variable name holding a bearer token."""
    h = []
    if auth:
        h.append({"key": "Authorization", "value": "Bearer {{%s}}" % auth})
    for k, v in (headers or {}).items():
        h.append({"key": k, "value": v})
    r = {"method": method, "header": h, "url": _url(url), "description": desc}
    if body is not None:
        h.append({"key": "Content-Type", "value": "application/json"})
        r["body"] = {"mode": "raw", "raw": body if isinstance(body, str) else json.dumps(body, indent=2),
                     "options": {"raw": {"language": "json"}}}
    if form is not None:
        r["body"] = {"mode": "urlencoded", "urlencoded": [{"key": k, "value": v} for k, v in form]}
    ev = []
    if pre:
        ev.append({"listen": "prerequest", "script": {"type": "text/javascript", "exec": pre.strip("\n").split("\n")}})
    if tests:
        ev.append({"listen": "test", "script": {"type": "text/javascript",
                                                 "exec": (PRELUDE + tests).strip("\n").split("\n")}})
    item = {"name": name, "request": r}
    if ev:
        item["event"] = ev
    return item


def token(name, user, pwd, store, *, client="nava-web", scope="openid", desc="", extra=""):
    """A Keycloak password-grant token request that stores the access token."""
    return req(
        name, "POST", "{{keycloak_url}}/realms/{{realm}}/protocol/openid-connect/token",
        desc=desc or f"Signs in as `{{{{{user}}}}}` through Keycloak (client `{client}`, password grant) and stores the access token in `{store}`.",
        form=[("grant_type", "password"), ("client_id", client), ("scope", scope),
              ("username", "{{%s}}" % user), ("password", "{{%s}}" % pwd)],
        tests=f"""
pm.test('HTTP 200 from Keycloak', () => pm.response.to.have.status(200));
const tok = pm.response.json();
pm.test('access token issued', () => pm.expect(tok.access_token).to.be.a('string'));
pm.collectionVariables.set('{store}', tok.access_token);
if (tok.id_token) pm.collectionVariables.set('{store}_id', tok.id_token);
const c = N.jwt(tok.access_token);
pm.collectionVariables.set('{store}_exp', String(c.exp));
pm.test('issuer is the pinned realm URL', () => pm.expect(c.iss).to.eql(pm.variables.replaceIn('{{{{keycloak_url}}}}/realms/{{{{realm}}}}')));
{extra}
""")


def problem_req(name, method, url, code, status, *, desc, auth=None, headers=None, body=None, extra="", pre=""):
    return req(name, method, url, desc=desc, auth=auth, headers=headers, body=body, pre=pre,
               tests=f"N.problem('{code}', {status});\n{extra}")


def kc_admin(name, method, path, *, desc, body=None, tests="", pre_extra=""):
    return req(name, method, "{{keycloak_url}}/admin/realms/{{realm}}" + path, desc=desc, auth="kc_admin_token",
               body=body, pre=ADMIN_TOKEN_PRE + pre_extra, tests=tests)


def onboard_body(domain_var, admin_local="admin", idp_host="93.184.216.34", extra=None):
    b = {
        "admin_email": admin_local + "@{{%s}}" % domain_var,
        "email_domains": ["{{%s}}" % domain_var],
        "identity_provider": {
            "provider_id": "oidc",
            "display_name": "Acceptance IdP",
            "config": {
                "clientId": "acceptance-client",
                "clientSecret": "dev-only-not-a-secret",
                "authorizationUrl": f"https://{idp_host}/auth",
                "tokenUrl": f"https://{idp_host}/token",
                "jwksUrl": f"https://{idp_host}/jwks",
                "useJwksUrl": "true",
                "defaultScope": "openid email profile",
            },
        },
    }
    if extra:
        b.update(extra)
    return b


# --------------------------------------------------------------------------- S1
def s1():
    return {
        "name": "S1 — Local stack starts healthy",
        "description": "TICK-43. Needs `tools/tunnels.sh up`: the admin listener (:9090) is never published, so the tunnel forwards it to `{{admin_url}}`.",
        "item": [
            req("S1.1 Liveness: GET /healthz lists every dependency", "GET", "{{admin_url}}/healthz",
                desc="nava-core's admin listener, through the tunnel. Every dependency is `ok`.",
                tests="""
const down = 'a dependency is down or still starting: start the stack (docker compose up --wait) and check docker compose ps';
pm.test('HTTP 200', () => pm.expect(pm.response.code, down).to.eql(200));
const b = pm.response.json();
pm.test('status ok', () => pm.expect(b.status, b.status === 'degraded' ? down : 'unexpected status').to.eql('ok'));
pm.test('checks keycloak, postgres and temporal, all ok', () => {
  pm.expect(b.checks, down).to.include({ keycloak: 'ok', postgres: 'ok', temporal: 'ok' });
});
"""),
            req("S1.2 Readiness: GET /readyz", "GET", "{{admin_url}}/readyz",
                desc="200 `ok` while PostgreSQL answers.",
                tests="""
pm.test('HTTP 200', () => pm.response.to.have.status(200));
pm.test('body ok', () => pm.expect(pm.response.text().trim()).to.eql('ok'));
"""),
            req("S1.3 Metrics: GET /metrics", "GET", "{{admin_url}}/metrics",
                desc="Prometheus text. `nava_dependency_degraded` is 0 for every dependency on a healthy stack.",
                tests="""
pm.test('HTTP 200', () => pm.response.to.have.status(200));
const t = pm.response.text();
pm.test('Prometheus exposition format', () => pm.expect(t).to.match(/^# (HELP|TYPE) /m));
pm.test('nava_dependency_degraded is 0 for keycloak, postgres and temporal', () => {
  ['keycloak', 'postgres', 'temporal'].forEach(d =>
    pm.expect(t).to.match(new RegExp('^nava_dependency_degraded\\\\{[^}]*dependency="' + d + '"[^}]*\\\\} 0$', 'm')));
});
"""),
            problem_req("S1.4 Negative: /healthz is not on the public port", "GET", "{{core_url}}/healthz",
                        "not_found", 404,
                        desc="The admin routes live only on the unpublished admin listener. The public port answers 404 problem."),
            problem_req("S1.5 Negative: /metrics is not on the public port", "GET", "{{core_url}}/metrics",
                        "not_found", 404, desc="Same as S1.4 for /metrics."),
            req("S1.6 Keycloak realm nava is up with the pinned issuer", "GET",
                "{{keycloak_url}}/realms/{{realm}}/.well-known/openid-configuration",
                desc="The issuer is pinned (TICK-160): the same value whatever Host the request used. nava-core trusts exactly this value.",
                tests="""
pm.test('HTTP 200', () => pm.response.to.have.status(200));
pm.test('issuer is {{keycloak_url}}/realms/{{realm}}', () =>
  pm.expect(pm.response.json().issuer).to.eql(pm.variables.replaceIn('{{keycloak_url}}/realms/{{realm}}')));
"""),
        ],
    }


# --------------------------------------------------------------------------- S2
def s2():
    forged = {"X-Tenant-Id": B, "X-Nava-Tenant": B, "X-Tenant": B, "baggage": f"nava.tenant_id={B},tenant_id={B}"}
    forged_a = {"X-Tenant-Id": A, "X-Nava-Tenant": A, "X-Tenant": A}
    return {
        "name": "S2 — A query without the right tenant returns nothing",
        "description": "TICK-44. The database wall is proven by `make isolation` (see the guide). Over HTTP the only tenant data route on `main` is `GET /api/v1/me`; these requests show that naming another tenant anywhere in the request changes nothing and never returns its data. Needs `tools/seed.sh`.",
        "item": [
            token("S2.1 Sign in as Alice (tenant A)", "alice_username", "alice_password", "alice_token"),
            token("S2.2 Sign in as Bob (tenant B)", "bob_username", "bob_password", "bob_token"),
            req("S2.3 Alice: GET /me is tenant A", "GET", "{{core_url}}/api/v1/me", auth="alice_token",
                desc="Baseline: Alice's own tenant and groups.",
                tests=f"N.me('{A}');\npm.collectionVariables.set('alice_groups', JSON.stringify(pm.response.json().groups));\nN.noLeak('{B}');"),
            req("S2.4 Bob: GET /me is tenant B, none of Alice's groups", "GET", "{{core_url}}/api/v1/me", auth="bob_token",
                desc="Bob's own tenant. No group id of Alice's appears, although both tenants have a group with the same name.",
                tests=f"""N.me('{B}');
N.noLeak('{A}');
pm.test('no group id of tenant A', () => {{
  const a = JSON.parse(pm.collectionVariables.get('alice_groups') || '[]');
  pm.response.json().groups.forEach(g => pm.expect(a).to.not.include(g));
}});
pm.test('both tenants have /service-desk, each its own id', () => pm.expect(N.paths()).to.include('/service-desk'));"""),
            req("S2.5 Negative: Alice names tenant B in headers, query and body", "GET",
                "{{core_url}}/api/v1/me?tenant_id=" + B + "&tenant=" + B, auth="alice_token", headers=forged,
                body={"tenant_id": B},
                desc="Tenant B's id in `X-Tenant-Id`, `X-Nava-Tenant`, `X-Tenant`, `baggage`, the query and a JSON body. The tenant comes only from the token: still tenant A.",
                tests=f"N.me('{A}');\nN.noLeak('{B}');"),
            req("S2.6 Negative: Bob names tenant A", "GET", "{{core_url}}/api/v1/me?tenant_id=" + A, auth="bob_token",
                headers=forged_a, desc="The mirror of S2.5: still tenant B.",
                tests=f"N.me('{B}');\nN.noLeak('{A}');"),
            problem_req("S2.7 Negative: Alice asks for a tenant B record by id", "GET",
                        "{{core_url}}/api/v1/records/" + B, "not_found", 404, auth="alice_token",
                        desc="There is no record route on `main` yet, so any by-id read is `404 not_found`, and the body says nothing about tenant B.",
                        extra=f"N.noLeak('{B}');"),
        ],
    }


# --------------------------------------------------------------------------- S3
def s3():
    return {
        "name": "S3 — Tenant user signs in, scoped to tenant",
        "description": "TICK-45. Sign in through Keycloak, call `GET /api/v1/me`, then the refused tokens: none, garbage, tampered, an ID token, a foreign issuer, an operator token, no organization and two organizations. The expired-token check is the last folder, because it waits for a 5-minute token to expire. Needs `tools/seed.sh`.",
        "item": [
            token("S3.1 Sign in as Alice", "alice_username", "alice_password", "alice_token",
                  extra=f"""pm.test('aud nava-api, typ Bearer, organization tenant-a ({A})', () => {{
  pm.expect([].concat(c.aud)).to.include('nava-api');
  pm.expect(c.typ).to.eql('Bearer');
  pm.expect(c.organization).to.have.property('tenant-a');
  pm.expect(c.organization['tenant-a'].id).to.eql('{A}');
}});
// Kept for the late expired-token check (last folder).
pm.collectionVariables.set('expiring_token', tok.access_token);
pm.collectionVariables.set('expiring_token_exp', String(c.exp));"""),
            req("S3.2 Alice: GET /me", "GET", "{{core_url}}/api/v1/me", auth="alice_token",
                desc="200: tenant A, her platform user id, her group ids and group paths.",
                tests=f"""N.me('{A}');
pm.test('Alice has groups, local login (idp_alias empty)', () => {{
  pm.expect(pm.response.json().groups).to.not.be.empty;
  pm.expect(pm.response.json().idp_alias).to.eql('');
}});
pm.test('user_id is the platform id, not the Keycloak id', () =>
  pm.expect(pm.response.json().user_id).to.not.eql(N.jwt(pm.collectionVariables.get('alice_token')).sub));"""),
            token("S3.3 Sign in as tenant A's admin", "admin_a_username", "admin_a_password", "admin_a_token"),
            req("S3.4 Tenant A admin: GET /me", "GET", "{{core_url}}/api/v1/me", auth="admin_a_token",
                desc="Also tenant A.", tests=f"N.me('{A}');\nN.noLeak('{B}');"),
            problem_req("S3.5 Negative: no token", "GET", "{{core_url}}/api/v1/me", "invalid_token", 401,
                        desc="No Authorization header: 401 with `WWW-Authenticate: Bearer`.",
                        extra="pm.test('WWW-Authenticate: Bearer', () => pm.expect(pm.response.headers.get('WWW-Authenticate')).to.match(/^Bearer/));"),
            problem_req("S3.6 Negative: garbage token", "GET", "{{core_url}}/api/v1/me", "invalid_token", 401,
                        desc="A bearer value that is not a JWT.", headers={"Authorization": "Bearer not-a-token"}),
            problem_req("S3.7 Negative: tampered token (payload changed)", "GET", "{{core_url}}/api/v1/me",
                        "invalid_token", 401, auth="tampered_token",
                        desc="Alice's token with the organization in its payload rewritten to tenant B. The signature no longer matches.",
                        pre=f"""const [h, p, s] = pm.collectionVariables.get('alice_token').split('.');
const claims = JSON.parse(CryptoJS.enc.Base64.parse(p.replace(/-/g, '+').replace(/_/g, '/') + '==='.slice((p.length + 3) % 4)).toString(CryptoJS.enc.Utf8));
claims.organization = {{ 'tenant-b': {{ id: '{B}' }} }};
const enc = CryptoJS.enc.Base64.stringify(CryptoJS.enc.Utf8.parse(JSON.stringify(claims))).replace(/=+$/, '').replace(/\\+/g, '-').replace(/\\//g, '_');
pm.collectionVariables.set('tampered_token', [h, enc, s].join('.'));""",
                        extra=f"N.noLeak('{B}');"),
            problem_req("S3.8 Negative: Alice's ID token used as an access token", "GET", "{{core_url}}/api/v1/me",
                        "invalid_token", 401, auth="alice_token_id",
                        desc="The ID token from S3.1 (`typ ID`, audience `nava-web`) is refused."),
            req("S3.9 Get a token from another issuer (Keycloak master realm)", "POST",
                "{{keycloak_url}}/realms/master/protocol/openid-connect/token",
                desc="A real, validly signed token, but from realm `master`, which is not on nava-core's issuer allow-list.",
                form=[("grant_type", "password"), ("client_id", "admin-cli"), ("username", "{{kc_admin_user}}"),
                      ("password", "{{kc_admin_password}}")],
                tests="""pm.test('HTTP 200', () => pm.response.to.have.status(200));
pm.collectionVariables.set('foreign_token', pm.response.json().access_token);
pm.test('issuer is realm master', () => pm.expect(N.jwt(pm.response.json().access_token).iss).to.match(/\\/realms\\/master$/));"""),
            problem_req("S3.10 Negative: wrong issuer", "GET", "{{core_url}}/api/v1/me", "invalid_token", 401,
                        auth="foreign_token", desc="The master-realm token from S3.9 is refused."),
            token("S3.11 Sign in as the operator", "operator_username", "operator_password", "operator_token",
                  client="nava-operator-cli",
                  desc="The platform operator signs in through `nava-operator-cli`. The token's audience is `nava-operator-api`, not `nava-api`.",
                  extra="pm.test('aud nava-operator-api', () => pm.expect([].concat(c.aud)).to.include('nava-operator-api'));"),
            problem_req("S3.12 Negative: operator token on a tenant endpoint", "GET", "{{core_url}}/api/v1/me",
                        "invalid_token", 401, auth="operator_token",
                        desc="An operator token is not a tenant token (S5 criterion 6, 401 half)."),
            kc_admin("S3.13 Setup: create Dave, a user in no organization", "POST", "/users",
                     desc="Keycloak admin API. 201 the first time, 409 when Dave already exists (a re-run).",
                     body={"username": "{{dave_username}}", "email": "{{dave_username}}", "enabled": True,
                           "emailVerified": True, "firstName": "Dave", "lastName": "NoOrg",
                           "credentials": [{"type": "password", "value": "{{dave_password}}", "temporary": False}]},
                     tests="pm.test('created (201) or already there (409)', () => pm.expect(pm.response.code).to.be.oneOf([201, 409]));"),
            token("S3.14 Sign in as Dave", "dave_username", "dave_password", "dave_token",
                  extra="pm.test('no organization claim', () => pm.expect(c.organization || {}).to.be.empty);"),
            problem_req("S3.15 Negative: a token with no organization", "GET", "{{core_url}}/api/v1/me",
                        "tenant_unresolved", 401, auth="dave_token",
                        desc="Signed in, but not tied to any tenant: 401 `tenant_unresolved`."),
            kc_admin("S3.16 Setup: create Carol", "POST", "/users",
                     desc="201, or 409 on a re-run.",
                     body={"username": "{{carol_username}}", "email": "{{carol_username}}", "enabled": True,
                           "emailVerified": True, "firstName": "Carol", "lastName": "TwoOrgs",
                           "credentials": [{"type": "password", "value": "{{carol_password}}", "temporary": False}]},
                     tests="pm.test('created (201) or already there (409)', () => pm.expect(pm.response.code).to.be.oneOf([201, 409]));"),
            kc_admin("S3.17 Setup: look up Carol's id", "GET", "/users?exact=true&username={{carol_username}}",
                     desc="Stores her Keycloak user id.",
                     tests="pm.test('HTTP 200, one user', () => { pm.response.to.have.status(200); pm.expect(pm.response.json()).to.have.length(1); });\npm.collectionVariables.set('carol_id', pm.response.json()[0].id);"),
            kc_admin("S3.18 Setup: add Carol to tenant A's organization", "POST", f"/organizations/{A}/members",
                     desc="201, or 409 when she is already a member.", body='"{{carol_id}}"',
                     tests="pm.test('member (201) or already (409)', () => pm.expect(pm.response.code).to.be.oneOf([201, 409]));"),
            kc_admin("S3.19 Setup: add Carol to tenant B's organization", "POST", f"/organizations/{B}/members",
                     desc="Carol is now in both organizations.", body='"{{carol_id}}"',
                     tests="pm.test('member (201) or already (409)', () => pm.expect(pm.response.code).to.be.oneOf([201, 409]));"),
            token("S3.20 Sign in as Carol asking for every organization", "carol_username", "carol_password",
                  "carol_token", scope="openid organization:*",
                  extra="pm.test('organization claim names two organizations', () => pm.expect(Object.keys(c.organization || {})).to.have.length(2));"),
            problem_req("S3.21 Negative: a token naming two organizations", "GET", "{{core_url}}/api/v1/me",
                        "tenant_unresolved", 401, auth="carol_token",
                        desc="Two organizations: the platform refuses to guess. 401 `tenant_unresolved`, and neither tenant is named.",
                        extra=f"N.noLeak('{A}', '{B}');"),
        ],
    }


# --------------------------------------------------------------------------- S4
def s4():
    init = r"""
// One run id per pass, so the collection can run again on the same stack.
const run = Math.random().toString(16).slice(2, 10);
pm.collectionVariables.set('run_id', run);
pm.collectionVariables.set('onb_domain', `onb-${run}.test`);
pm.collectionVariables.set('onb_domain2', `onb2-${run}.test`);
pm.collectionVariables.set('onb_domain3', `onb3-${run}.test`);
pm.collectionVariables.set('onb_key', `tick287-${run}-1`);
"""
    reuse = onboard_body("onb_domain")
    reuse["identity_provider"]["display_name"] = "A different body"
    forged_tid = "11111111-1111-4111-8111-111111111111"
    return {
        "name": "S4 — Operator onboards a tenant",
        "description": "TICK-46. `POST /api/v1/operator/tenants` on nava-onboard (:8081), reached through `tools/tunnels.sh up` at `{{onboard_url}}`. The new admin's first sign-in goes through the tenant's own IdP in a browser, which Postman cannot drive: the guide runs `TestS4Acceptance` for that part.",
        "item": [
            token("S4.1 Sign in as the operator", "operator_username", "operator_password", "operator_token",
                  client="nava-operator-cli",
                  extra="pm.test('aud nava-operator-api, realm role nava-operator', () => {\n  pm.expect([].concat(c.aud)).to.include('nava-operator-api');\n  pm.expect((c.realm_access || {}).roles || []).to.include('nava-operator');\n});"),
            token("S4.2 Sign in as tenant A's admin", "admin_a_username", "admin_a_password", "admin_a_token"),
            problem_req("S4.3 Negative: onboarding with no token", "POST", "{{onboard_url}}/api/v1/operator/tenants",
                        "invalid_token", 401, desc="401 before anything else.", pre=init,
                        headers={"Idempotency-Key": "{{onb_key}}"}, body=onboard_body("onb_domain")),
            problem_req("S4.4 Negative: a tenant admin's token", "POST", "{{onboard_url}}/api/v1/operator/tenants",
                        "forbidden", 403, auth="admin_a_token",
                        desc="A valid tenant token is not an operator token: 403 `forbidden`.",
                        headers={"Idempotency-Key": "{{onb_key}}"}, body=onboard_body("onb_domain")),
            problem_req("S4.5 Negative: the operator route on the tenant ingress (nava-core)", "POST",
                        "{{core_url}}/api/v1/operator/tenants", "not_implemented", 501, auth="operator_token",
                        desc="nava-core never holds the onboarding credentials: it answers the operator route 501.",
                        headers={"Idempotency-Key": "{{onb_key}}"}, body=onboard_body("onb_domain")),
            problem_req("S4.6 Negative: no Idempotency-Key", "POST", "{{onboard_url}}/api/v1/operator/tenants",
                        "validation_failed", 400, auth="operator_token",
                        desc="The header is required. `errors` names the failing field and never echoes input.",
                        body=onboard_body("onb_domain"),
                        extra="pm.test('errors[] has a pointer', () => pm.expect(pm.response.json().errors[0]).to.have.property('pointer'));"),
            problem_req("S4.7 Negative: IdP URL on a private address", "POST", "{{onboard_url}}/api/v1/operator/tenants",
                        "invalid_idp_config", 422, auth="operator_token",
                        desc="IdP URLs must be https on public hosts. `10.0.0.1` is refused before anything is created.",
                        headers={"Idempotency-Key": "{{onb_key}}-private"},
                        body=onboard_body("onb_domain3", idp_host="10.0.0.1")),
            req("S4.8 Onboard a new tenant", "POST", "{{onboard_url}}/api/v1/operator/tenants", auth="operator_token",
                desc="201 with the server-derived tenant id and `state: active`. Creates the Keycloak organization and IdP, the binding, the domain claim and a pending invite for the admin.",
                headers={"Idempotency-Key": "{{onb_key}}"}, body=onboard_body("onb_domain"),
                tests="""pm.test('HTTP 201', () => pm.response.to.have.status(201));
const b = pm.response.json();
pm.test('tenant_id is a uuid, state active', () => {
  pm.expect(b.tenant_id).to.match(/^[0-9a-f-]{36}$/);
  pm.expect(b.state).to.eql('active');
});
pm.collectionVariables.set('onb_tenant_id', b.tenant_id);
pm.collectionVariables.set('onb_body', pm.response.text());"""),
            req("S4.9 Replay: same key, same body", "POST", "{{onboard_url}}/api/v1/operator/tenants", auth="operator_token",
                desc="200 with a byte-identical body. Nothing new is created.",
                headers={"Idempotency-Key": "{{onb_key}}"}, body=onboard_body("onb_domain"),
                tests="""pm.test('HTTP 200 (a replay)', () => pm.response.to.have.status(200));
pm.test('byte-identical body', () => pm.expect(pm.response.text()).to.eql(pm.collectionVariables.get('onb_body')));"""),
            problem_req("S4.10 Negative: same key, different body", "POST", "{{onboard_url}}/api/v1/operator/tenants",
                        "idempotency_key_reuse", 409, auth="operator_token",
                        desc="409 `idempotency_key_reuse`; nothing is created.",
                        headers={"Idempotency-Key": "{{onb_key}}"}, body=reuse),
            problem_req("S4.11 Negative: another tenant claims the same domain", "POST",
                        "{{onboard_url}}/api/v1/operator/tenants", "domain_claimed", 409, auth="operator_token",
                        desc="A new key with the S4.8 domain: 409 `domain_claimed`. The body names neither the owning tenant nor the domain.",
                        headers={"Idempotency-Key": "{{onb_key}}-dup"}, body=onboard_body("onb_domain"),
                        extra="N.noLeak(pm.collectionVariables.get('onb_tenant_id'), pm.collectionVariables.get('onb_domain'));"),
            problem_req("S4.12 Negative: claim seeded tenant A's domain", "POST",
                        "{{onboard_url}}/api/v1/operator/tenants", "domain_claimed", 409, auth="operator_token",
                        desc="`tenant-a.test` belongs to tenant A (seeded): 409 with no trace of tenant A.",
                        headers={"Idempotency-Key": "{{onb_key}}-tenant-a"},
                        body=onboard_body("seed_domain_a"),
                        extra=f"N.noLeak('{A}', 'tenant-a');"),
            req("S4.13 Negative: a tenant id in the body and a header is ignored", "POST",
                "{{onboard_url}}/api/v1/operator/tenants?tenant_id=" + forged_tid, auth="operator_token",
                desc="The tenant id is derived from the operator and the key. One sent in the body, query and `X-Tenant-Id` is ignored.",
                headers={"Idempotency-Key": "{{onb_key}}-forged", "X-Tenant-Id": forged_tid},
                body=onboard_body("onb_domain2", extra={"tenant_id": forged_tid}),
                tests=f"""pm.test('HTTP 201', () => pm.response.to.have.status(201));
pm.test('tenant_id is not the one sent', () => pm.expect(pm.response.json().tenant_id).to.not.eql('{forged_tid}'));
pm.test('a different key gives a different tenant', () => pm.expect(pm.response.json().tenant_id).to.not.eql(pm.collectionVariables.get('onb_tenant_id')));"""),
            problem_req("S4.14 Negative: the onboarding service serves operator routes only", "GET",
                        "{{onboard_url}}/api/v1/me", "not_found", 404, auth="alice_token",
                        desc="nava-onboard mounts nothing but `/api/v1/operator/*`: a valid tenant token on `/me` there is 404."),
        ],
    }


# --------------------------------------------------------------------------- S5
def s5():
    alice_lookup = kc_admin("S5.1 Setup: look up Alice and tenant A's service-desk group", "GET",
                            "/users?exact=true&username={{alice_username}}",
                            desc="Stores Alice's Keycloak id. The group id is looked up in the test script.",
                            tests="""pm.test('HTTP 200, one user', () => { pm.response.to.have.status(200); pm.expect(pm.response.json()).to.have.length(1); });
pm.collectionVariables.set('alice_kc_id', pm.response.json()[0].id);
pm.sendRequest({ url: pm.variables.replaceIn('{{keycloak_url}}/admin/realms/{{realm}}/group-by-path/t/""" + A + """/service-desk'),
  header: { Authorization: 'Bearer ' + pm.collectionVariables.get('kc_admin_token') } }, (e, r) => {
  pm.test('service-desk group found', () => pm.expect(r.code).to.eql(200));
  pm.collectionVariables.set('alice_sd_group', r.json().id);
});""")
    return {
        "name": "S5 — Group changes and deprovisioning take effect within 10 minutes",
        "description": "TICK-47. An administrator changes Alice in Keycloak (admin API, as the console would); the same, still-valid token sees the change after the next `nava-groupsync` cycle. The polling requests repeat in the Collection Runner until the change shows (at most 11 min). Set `NAVA_GROUPSYNC_INTERVAL=15s` in `.env` to make this take seconds; with the default 5 min it takes up to 5 min per step. Every step restores Alice. Needs `tools/seed.sh`.",
        "item": [
            alice_lookup,
            token("S5.2 Sign in as Alice", "alice_username", "alice_password", "alice_token"),
            token("S5.3 Sign in as Bob (the control)", "bob_username", "bob_password", "bob_token"),
            req("S5.4 Alice: GET /me shows /service-desk", "GET", "{{core_url}}/api/v1/me", auth="alice_token",
                desc="Story step 1. `group_paths` is relative to the tenant root (ADR-002), sorted by id.",
                tests=f"""N.me('{A}');
pm.test('group_paths includes /service-desk', () => pm.expect(N.paths()).to.include('/service-desk'));"""),
            kc_admin("S5.5 Admin removes Alice from service-desk", "DELETE",
                     "/users/{{alice_kc_id}}/groups/{{alice_sd_group}}",
                     desc="Keycloak console: Groups › t › tenant A › service-desk › Members › Leave.",
                     tests="pm.test('HTTP 204', () => pm.response.to.have.status(204));\npm.collectionVariables.set('change_at', String(Date.now()));\npm.collectionVariables.set('poll_n', 0);"),
            req("S5.6 Poll: the same token's /me loses /service-desk", "GET", "{{core_url}}/api/v1/me",
                auth="alice_token", pre=poll_pre(),
                desc="Story step 3. Same token as S5.4. Repeats until the next sync removes the group. By hand: Send every 15 s.",
                tests=poll_test("!N.paths().includes('/service-desk')", f"""N.me('{A}');
pm.test('/service-desk is gone', () => pm.expect(N.paths()).to.not.include('/service-desk'));""")),
            req("S5.7 Control: Bob still has his /service-desk", "GET", "{{core_url}}/api/v1/me", auth="bob_token",
                desc="Tenant B is untouched.",
                tests=f"N.me('{B}');\npm.test('Bob keeps /service-desk', () => pm.expect(N.paths()).to.include('/service-desk'));"),
            kc_admin("S5.8 Restore: add Alice back to service-desk", "PUT",
                     "/users/{{alice_kc_id}}/groups/{{alice_sd_group}}", desc="Undo S5.5.",
                     tests="pm.test('HTTP 204', () => pm.response.to.have.status(204));\npm.collectionVariables.set('change_at', String(Date.now()));"),
            token("S5.9 Sign in as Alice again (fresh token)", "alice_username", "alice_password", "alice_token"),
            req("S5.10 Poll: /me shows /service-desk again", "GET", "{{core_url}}/api/v1/me", auth="alice_token",
                pre=poll_pre(), desc="Waits for the restore to sync, so the disable test starts clean.",
                tests=poll_test("N.paths().includes('/service-desk')",
                                f"N.me('{A}');\npm.test('/service-desk is back', () => pm.expect(N.paths()).to.include('/service-desk'));")),
            kc_admin("S5.11 Admin disables Alice", "PUT", "/users/{{alice_kc_id}}",
                     desc="Story step 4. Keycloak console: Users › alice › Enabled off. Alice's token from S5.9 is still valid.",
                     body={"enabled": False},
                     tests="pm.test('HTTP 204', () => pm.response.to.have.status(204));\npm.collectionVariables.set('change_at', String(Date.now()));\npm.collectionVariables.set('poll_n', 0);"),
            req("S5.12 Negative: Alice cannot sign in again", "POST",
                "{{keycloak_url}}/realms/{{realm}}/protocol/openid-connect/token",
                desc="Keycloak refuses at once: 400 `invalid_grant`, Account disabled.",
                form=[("grant_type", "password"), ("client_id", "nava-web"), ("scope", "openid"),
                      ("username", "{{alice_username}}"), ("password", "{{alice_password}}")],
                tests="""pm.test('HTTP 400 invalid_grant', () => { pm.response.to.have.status(400); pm.expect(pm.response.json().error).to.eql('invalid_grant'); });
pm.test('Account disabled', () => pm.expect(pm.response.json().error_description).to.match(/disabled/i));"""),
            req("S5.13 Poll: the same unexpired token gets 401 tenant_unresolved", "GET", "{{core_url}}/api/v1/me",
                auth="alice_token", pre=poll_pre(),
                desc="After the next sync the held token stops working, well before it expires.",
                tests=poll_test("pm.response.code === 401",
                                "N.problem('tenant_unresolved', 401);\npm.test('the token had not expired yet', () => pm.expect(Date.now() / 1000).to.be.below(Number(pm.collectionVariables.get('alice_token_exp'))));")),
            req("S5.14 Control: Bob is unaffected", "GET", "{{core_url}}/api/v1/me", auth="bob_token",
                desc="Bob's token from S5.3 may have expired by now with a 5-minute sync, so it is refreshed in the pre-request script.",
                pre="""pm.sendRequest({ url: pm.variables.replaceIn('{{keycloak_url}}/realms/{{realm}}/protocol/openid-connect/token'), method: 'POST',
  body: { mode: 'urlencoded', urlencoded: [{ key: 'grant_type', value: 'password' }, { key: 'client_id', value: 'nava-web' },
    { key: 'scope', value: 'openid' }, { key: 'username', value: pm.variables.replaceIn('{{bob_username}}') },
    { key: 'password', value: pm.variables.replaceIn('{{bob_password}}') }] } },
  (e, r) => pm.collectionVariables.set('bob_token', r.json().access_token));""",
                tests=f"N.me('{B}');"),
            kc_admin("S5.15 Restore: re-enable Alice", "PUT", "/users/{{alice_kc_id}}", body={"enabled": True},
                     desc="Undo S5.11.",
                     tests="pm.test('HTTP 204', () => pm.response.to.have.status(204));\npm.collectionVariables.set('change_at', String(Date.now()));\npm.collectionVariables.set('poll_n', 0);"),
            token("S5.16 Sign in as Alice (re-enabled)", "alice_username", "alice_password", "alice_token"),
            req("S5.17 Poll: Alice resolves again", "GET", "{{core_url}}/api/v1/me", auth="alice_token",
                pre=poll_pre(), desc="After the next sync Alice is back in tenant A with /service-desk.",
                tests=poll_test("pm.response.code === 200",
                                f"N.me('{A}');\npm.test('/service-desk', () => pm.expect(N.paths()).to.include('/service-desk'));")),
        ],
    }


# --------------------------------------------------------------------------- S6
def s6():
    return {
        "name": "S6 — API ground rules",
        "description": "TICK-48. Problem bodies, unknown routes, method rules, the published contract and request ids. Idempotent replay is in S4 (S4.9, S4.10). S6.11–S6.13: tenant A over its per-tenant rate limit gets 429 while tenant B's calls succeed. Tenant-bound cursors have no route on `main` yet (see the guide's coverage page).",
        "item": [
            req("S6.1 GET /api/v1/openapi.yaml (public)", "GET", "{{core_url}}/api/v1/openapi.yaml",
                desc="The contract, with no token.",
                tests="""pm.test('HTTP 200', () => pm.response.to.have.status(200));
pm.test('OpenAPI 3.1 with /api/v1/me', () => { pm.expect(pm.response.text()).to.include('openapi: 3.1.0'); pm.expect(pm.response.text()).to.include('/api/v1/me:'); });"""),
            req("S6.2 GET /api/v1/docs (public)", "GET", "{{core_url}}/api/v1/docs",
                desc="Human-readable docs generated from the contract.",
                tests="""pm.test('HTTP 200', () => pm.response.to.have.status(200));
pm.test('HTML naming the operations', () => { pm.expect(pm.response.headers.get('Content-Type')).to.include('text/html'); pm.expect(pm.response.text()).to.include('/api/v1/me'); });"""),
            token("S6.3 Sign in as Alice", "alice_username", "alice_password", "alice_token"),
            problem_req("S6.4 Negative: unknown /api/v1 route with no token", "GET", "{{core_url}}/api/v1/nope",
                        "invalid_token", 401,
                        desc="Fail closed: under /api/v1, authentication runs before routing, so an unknown path tells an anonymous caller nothing."),
            problem_req("S6.5 Negative: unknown /api/v1 route with a token", "GET", "{{core_url}}/api/v1/nope",
                        "not_found", 404, auth="alice_token", desc="404 problem behind a valid token."),
            problem_req("S6.6 Negative: unknown path outside /api/v1", "GET", "{{core_url}}/nope", "not_found", 404,
                        desc="404 problem, no token needed."),
            problem_req("S6.7 Negative: wrong method on /api/v1/me", "DELETE", "{{core_url}}/api/v1/me",
                        "method_not_allowed", 405, auth="alice_token",
                        desc="405 problem with an `Allow` header.",
                        extra="pm.test('Allow names GET', () => pm.expect(pm.response.headers.get('Allow') || '').to.include('GET'));"),
            problem_req("S6.8 Negative: wrong method with no token", "DELETE", "{{core_url}}/api/v1/me",
                        "invalid_token", 401, desc="Authentication first: no token is 401 even for a wrong method."),
            req("S6.9 Every response carries X-Request-Id; an inbound traceparent is accepted", "GET",
                "{{core_url}}/api/v1/me", auth="alice_token",
                headers={"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"},
                desc="The service starts its own trace and only links the caller's `traceparent` (it never continues an outside trace). Search the request id in the logs: `docker compose logs nava-core | grep <id>`.",
                tests=f"""N.me('{A}');
pm.test('X-Request-Id present', () => pm.expect(pm.response.headers.get('X-Request-Id')).to.be.a('string').and.not.empty);
pm.test('the caller\\'s trace id is not echoed back', () => pm.expect(pm.response.headers.get('traceparent') || '').to.not.include('4bf92f3577b34da6a3ce929d0e0e4736'));"""),
            problem_req("S6.10 Negative: request-specific input never appears in a problem", "GET",
                        "{{core_url}}/api/v1/nope-<script>alert(1)</script>", "not_found", 404, auth="alice_token",
                        desc="`detail` is fixed text per code; the path is not echoed.",
                        extra="N.noLeak('<script>', 'nope-');"),
            token("S6.11 Sign in as Bob (tenant B)", "bob_username", "bob_password", "bob_token"),
            req("S6.12 Tenant A over its limit gets 429 while tenant B succeeds", "GET", "{{core_url}}/api/v1/me",
                auth="bob_token",
                desc="Before this request, Alice (tenant A) fires `rate_burst_requests` (default 300) `GET /api/v1/me` calls at once, more than tenant A's bucket holds (`NAVA_RATE_LIMIT_BURST`, 100 by default, refilled at `NAVA_RATE_LIMIT_RPS`, 50/s), and Bob (tenant B) fires 20 in the same burst. Tenant A must see 429 `rate_limited` with `Retry-After`; every call of Bob's, and this request, must be 200 for tenant B. The limit is keyed only on the tenant in the token.",
                pre="""const n = Number(pm.variables.get('rate_burst_requests') || 300);
const me = pm.variables.replaceIn('{{core_url}}/api/v1/me');
const call = (tok, who) => pm.sendRequest({ url: me, method: 'GET', header: { Authorization: 'Bearer ' + tok } }, (err, res) => {
  const k = err ? 'error' : String(res.code);
  st[who][k] = (st[who][k] || 0) + 1;
  if (!err && res.code === 429 && !sample) {
    sample = { code: res.code, ct: res.headers.get('Content-Type') || '', retry: res.headers.get('Retry-After'),
      rid: res.headers.get('X-Request-Id'), body: res.text() };
  }
  if (!err && who === 'b' && res.code === 200) { try { st.bTenants.push(res.json().tenant_id); } catch (e) {} }
  if (--left === 0) {
    pm.collectionVariables.set('rl_stats', JSON.stringify(st));
    pm.collectionVariables.set('rl_sample', JSON.stringify(sample || {}));
    pm.collectionVariables.set('rl_burst_end', String(Date.now()));
  }
});
const st = { a: {}, b: {}, bTenants: [] };
let sample = null;
let left = n + 20;
const a = pm.collectionVariables.get('alice_token'), b = pm.collectionVariables.get('bob_token');
// Bob's 20 calls are spread through Alice's burst, not sent after it.
const step = Math.max(1, Math.floor(n / 20));
let sentB = 0;
for (let i = 0; i < n; i++) { call(a, 'a'); if (i % step === 0 && sentB < 20) { call(b, 'b'); sentB++; } }
while (sentB < 20) { call(b, 'b'); sentB++; }""",
                tests=f"""const st = JSON.parse(pm.collectionVariables.get('rl_stats') || '{{}}');
const s = JSON.parse(pm.collectionVariables.get('rl_sample') || '{{}}');
console.log('burst results (status: count): tenant A', JSON.stringify(st.a), 'tenant B', JSON.stringify(st.b));
pm.test('tenant A: some calls got 429', () => pm.expect(st.a && st.a['429'] || 0).to.be.above(0));
pm.test('tenant A: the calls within its burst got 200', () => pm.expect(st.a && st.a['200'] || 0).to.be.above(0));
pm.test('tenant A: nothing but 200 and 429', () => pm.expect(Object.keys(st.a || {{}}).filter(k => k !== '200' && k !== '429')).to.eql([]));
pm.test('the 429 is problem+json rate_limited', () => {{
  pm.expect(s.ct).to.include('application/problem+json');
  const b = JSON.parse(s.body);
  pm.expect(b.code).to.eql('rate_limited');
  pm.expect(b.status).to.eql(429);
  pm.expect(b.type).to.eql('urn:nava:problem:rate_limited');
  pm.expect(N.REGISTERED).to.include(b.code);
  pm.expect(b.instance).to.eql('urn:nava:request:' + s.rid);
}});
pm.test('the 429 carries Retry-After in whole seconds, at least 1', () => pm.expect(Number(s.retry)).to.be.at.least(1));
pm.test('the 429 names no tenant', () => pm.expect(s.body).to.not.include('{A}'));
pm.collectionVariables.set('rl_retry_after', String(s.retry || 2));
pm.test('tenant B: all 20 calls in the same burst got 200', () => pm.expect(st.b).to.eql({{ '200': 20 }}));
pm.test('tenant B: each answered for tenant B', () => (st.bTenants || []).forEach(t => pm.expect(t).to.eql('{B}')));
N.me('{B}');"""),
            req("S6.13 Tenant A is served again after Retry-After", "GET", "{{core_url}}/api/v1/me", auth="alice_token",
                desc="Waits the `Retry-After` seconds from S6.12, then calls as Alice: the bucket has refilled, so 200 for tenant A. A by-hand run: wait a couple of seconds after S6.12.",
                pre="""const end = Number(pm.collectionVariables.get('rl_burst_end') || 0);
const wait = Math.max(0, end + 1000 * Number(pm.collectionVariables.get('rl_retry_after') || 2) + 500 - Date.now());
if (wait > 0) setTimeout(() => {}, wait);""",
                tests=f"N.me('{A}');"),
        ],
    }


# --------------------------------------------------------------------------- S7
def s7():
    q = 'WorkflowType="nava.smoke.relay"'
    return {
        "name": "S7 — A committed outbox row runs exactly one workflow",
        "description": "TICK-49. There is no platform HTTP route for workflows. Run `make demo-outbox DEDUPE_KEY=accept` twice first (see the guide), and `tools/tunnels.sh up` (with `COMPOSE_PROFILES=operator`) for the Temporal UI at `{{temporal_ui_url}}`. These requests read the Temporal UI's own HTTP API, as an operator would.",
        "item": [
            req("S7.1 Temporal: the demo's workflow has exactly one execution", "GET",
                "{{temporal_ui_url}}/api/v1/namespaces/nava/workflows?query=" + q,
                desc="Selects the demo's own workflow: `demo_workflow_id` from the environment (the `workflow id` `make demo-outbox` printed), else the latest-started tenant A `nava.smoke.relay` workflow. That id has exactly one execution: running the demo twice with the same key did not start a second workflow. Other `nava.smoke.relay` workflows on the stack (another tenant's, or another demo key) do not affect the result; each must still carry a tenant-scoped id.",
                tests=f"""pm.test('HTTP 200', () => pm.response.to.have.status(200));
const ex = pm.response.json().executions || [];
const ids = ex.map(e => e.execution.workflowId);
pm.test('every nava.smoke.relay id is tenant-scoped: t:<tenant>:nava.smoke.relay:<start row>', () =>
  ids.forEach(i => pm.expect(i).to.match(/^t:[0-9a-f-]{{36}}:nava\\.smoke\\.relay:[0-9a-f-]{{36}}$/)));
// The demo writes for tenant A only (cmd/demo-outbox), so its workflow is one of tenant A's.
const mine = ex.filter(e => e.execution.workflowId.startsWith('t:{A}:nava.smoke.relay:'));
pm.test('at least one tenant A demo workflow (run make demo-outbox first)', () => pm.expect(mine.length).to.be.above(0));
const wanted = pm.variables.get('demo_workflow_id');
const id = wanted || (mine.slice().sort((x, y) => String(y.startTime).localeCompare(String(x.startTime)))[0] || {{ execution: {{}} }}).execution.workflowId;
console.log('S7.1 demo workflow', id, wanted ? '(demo_workflow_id)' : '(latest-started tenant A demo workflow)', '; nava.smoke.relay executions listed:', ex.length);
const runs = ex.filter(e => e.execution.workflowId === id);
pm.test('the demo workflow is listed and belongs to tenant A', () => {{
  pm.expect(runs.length, id).to.be.above(0);
  pm.expect(id).to.match(/^t:{A}:nava\\.smoke\\.relay:/);
}});
pm.test('the demo workflow has exactly one execution', () => pm.expect(runs).to.have.length(1));
if (runs.length) {{ pm.collectionVariables.set('wf_id', runs[0].execution.workflowId); pm.collectionVariables.set('wf_run', runs[0].execution.runId); }}"""),
            req("S7.2 Temporal: its history holds the start and two signals in commit order", "GET",
                "{{temporal_ui_url}}/api/v1/namespaces/nava/workflows/{{wf_id}}/history",
                desc="The two `demo.note` signals arrive once each, in the order their outbox rows were committed.",
                tests="""pm.test('HTTP 200', () => pm.response.to.have.status(200));
const ev = (pm.response.json().history || {}).events || [];
const types = ev.map(e => e.eventType);
pm.test('starts with WorkflowExecutionStarted', () => pm.expect(types[0]).to.match(/WORKFLOW_EXECUTION_STARTED|WorkflowExecutionStarted/));
const sig = ev.filter(e => /SIGNALED|Signaled/.test(e.eventType));
pm.test('exactly two demo.note signals', () => {
  pm.expect(sig).to.have.length(2);
  sig.forEach(s => pm.expect(s.workflowExecutionSignaledEventAttributes.signalName).to.eql('demo.note'));
});"""),
        ],
    }


# --------------------------------------------------------------------------- late
def late():
    return {
        "name": "S3 (late) — An expired token is refused",
        "description": "TICK-45. Runs last: it waits until the token kept in S3.1 is past its 5-minute expiry plus nava-core's 30 s clock-skew allowance, then uses it.",
        "item": [
            problem_req("S3.22 Negative: expired token", "GET", "{{core_url}}/api/v1/me", "invalid_token", 401,
                        auth="expiring_token",
                        desc="Alice's token from S3.1, now expired. nava-core allows 30 s of clock skew, so by hand send it more than 5 min 30 s after S3.1.",
                        pre="""const exp = Number(pm.collectionVariables.get('expiring_token_exp'));
// nava-core allows 30 s of clock skew on exp (internal/identity/keycloak), so wait past that.
const wait = Math.max(0, exp * 1000 + 40000 - Date.now());
if (wait > 0) { console.log(`waiting ${Math.round(wait / 1000)} s for the S3.1 token to expire`); setTimeout(() => {}, wait); }""",
                        extra="pm.test('the token is past exp plus the 30 s skew', () => pm.expect(Date.now() / 1000).to.be.above(Number(pm.collectionVariables.get('expiring_token_exp')) + 30));"),
        ],
    }


def build():
    coll = {
        "info": {
            "_postman_id": str(uuid.uuid5(uuid.NAMESPACE_URL, "tick-287/acceptance-collection")),
            "name": "TicketingSystem — acceptance S1–S7",
            "description": "One folder per delivered story, run top to bottom. Before a Collection Runner pass: stack up, `tools/seed.sh`, `tools/tunnels.sh up`, and `make demo-outbox DEDUPE_KEY=accept` twice. Select the `TicketingSystem local` environment. Only the documented dev-only seed credentials appear in the environment; tokens are fetched at run time.",
            "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
        },
        "item": [s1(), s2(), s3(), s4(), s5(), s6(), s7(), late()],
        "variable": [
            {"key": "seed_domain_a", "value": "tenant-a.test"},
        ],
    }
    env = {
        "id": str(uuid.uuid5(uuid.NAMESPACE_URL, "tick-287/acceptance-env")),
        "name": "TicketingSystem local",
        "values": [
            {"key": k, "value": v, "type": t, "enabled": True} for k, v, t in [
                ("core_url", "http://127.0.0.1:18080", "default"),
                ("keycloak_url", "http://localhost:18081", "default"),
                ("onboard_url", "http://127.0.0.1:18082", "default"),
                ("admin_url", "http://127.0.0.1:19090", "default"),
                ("temporal_ui_url", "http://127.0.0.1:18233", "default"),
                ("realm", "nava", "default"),
                ("alice_username", "alice@tenant-a.test", "default"),
                ("alice_password", "dev-only-alice", "secret"),
                ("admin_a_username", "admin@tenant-a.test", "default"),
                ("admin_a_password", "dev-only-admin-a", "secret"),
                ("bob_username", "bob@tenant-b.test", "default"),
                ("bob_password", "dev-only-bob", "secret"),
                ("operator_username", "operator@nava.test", "default"),
                ("operator_password", "dev-only-operator", "secret"),
                ("dave_username", "dave@no-org.test", "default"),
                ("dave_password", "dev-only-dave", "secret"),
                ("carol_username", "carol@two-orgs.test", "default"),
                ("carol_password", "dev-only-carol", "secret"),
                ("kc_admin_user", "admin", "default"),
                ("kc_admin_password", "dev-only-admin", "secret"),
                ("poll_interval_ms", "10000", "default"),
                ("poll_limit_seconds", "660", "default"),
                ("demo_workflow_id", "", "default"),
                ("rate_burst_requests", "300", "default"),
            ]
        ],
        "_postman_variable_scope": "environment",
    }
    return coll, env


def requests(coll):
    """(folder, request name, method, url) for the coverage table."""
    for f in coll["item"]:
        for it in f["item"]:
            yield f["name"], it["name"], it["request"]["method"], it["request"]["url"]
