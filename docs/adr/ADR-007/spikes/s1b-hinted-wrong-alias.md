# ADR-007 spike S1b: hinted wrong alias

TICK-843 (RFC-018 rev 1 §7). Run on 2026-11-14 by the Technical Architect. This file is the body of the Technical Architect's TICK-843 comment `01a1d8b1-5c91-746d-bdd7-9e677522e01e`, committed as written by the Solution Architect. The raw test output is `s1b-raw.log` next to this file.

**Outcome as ruled by the CTO (TICK-843 comment `01a1d8b1-e376`):** claims 1 and 2 PASS 3/3; claim 3 (the Technical Architect's inference) disproved 3/3. None of the pre-stated fail conditions (form, error, `organization` claim) occurred, so the rev 1 §7 fail branch does not apply. ADR-007-A1 adopts the result as measured: D1-A1.4(b) verified, RR-A1-5 withdrawn.

---

**SPIKE: FAIL (S1b, RFC-018 rev 1 §7).** It failed on one of its three claims. Two claims pass 3/3: a hinted wrong alias issues a code, and the token has no `organization` claim, so `Resolve` returns 401. The third claim fails 3/3: a fresh user's first sign-in does **not** leave them with zero memberships. They join their own org, tenant-b. So RR-A1-5 and my round-1 F3 are wrong, and I'm retracting both. The fail branch as written in §7 doesn't fit what we saw (details below).

**Re-check of the blocking items.** The only blocking item in round 1 was Security F1, and the Security reviewer is re-checking it. All my round-1 items were non-blocking. Rev 1 adopted them as proposed, except for the corrections below. S1b also confirms RR-A1-6's premise at runtime: an alias that names no org gets `invalid_scope` (E14c).

**Evidence path.** I can't commit, so this comment is the evidence. Proposed path: `docs/adr/ADR-007/spikes/s1b-hinted-wrong-alias.md`. The Solution Architect can commit this comment's body there in the ADR-007-A1 docs PR. The raw log is attached as `s1b-raw.log`.

## Hypothesis and pass/fail (as stated in rev 1 §7, set before the run)
On Keycloak 26.8.1 with `realm-nava.json`, sending `kc_idp_hint=idp-tenant-b` together with `organization:tenant-a`:
1. for an existing tenant-B user, gives a 302 to `redirect_uri` that carries `code`;
2. the exchanged token has no `organization` claim;
3. for a fresh tenant-B user, the first sign-in leaves zero memberships.

**Pass:** all three hold, 3/3 runs. **Fail:** a form, an error, or an `organization` claim.

## Setup
- **Keycloak:** `quay.io/keycloak/keycloak:26.8.1` (`sha256:3f3c5f8e03c614774b4bd9173b88391fefa4153bfc010d8012d59ad825377332`). It ran with `start-dev` and H2 in a throwaway container on `127.0.0.1:18081`, with `KC_HOSTNAME=http://localhost:18081`. It imported the unchanged `deploy/compose/keycloak/import/realm-nava.json` from `main` `5e345c06`. No extra features were enabled: the startup log lists only the defaults, so `PARAMETERIZED_SCOPES` was off.
- **Upstream IdP realms:** `idp-tenant-a` and `idp-tenant-b`. The suite's `TestMain` imports them from `deploy/compose/keycloak/test-realms`.
- **Harness:** `internal/identitystack` (Go 1.26.11 toolchain, `-tags identitystack`), using its existing `newClient` cookie jar, `tokenForm`, `claims` and `stackAdapter` helpers. No browser.
- **Host client:** I registered `s1b-host` (public, with `organization` as an optional scope, not a default) as the embedding-client shape. I also repeated E14b against `nava-web`, which has bare `organization` as a default scope.
- **Users:** the existing user is `dave@tenant-b.test`, brokered in by the control run. Fresh users are `s1b-<random>@tenant-b.test`, created in realm `idp-tenant-b` through the admin API.
- **Host:** Windows 11, Docker 29.9.2. Date: 2026-11-14.

Differences from the compose stack: `start-dev` instead of `start --optimized`, and H2 instead of Postgres. Neither touches the authentication-flow or scope code.

## Raw results
| Case | Runs | Observed | Claim |
|---|---|---|---|
| Control: right alias `organization:tenant-b` plus hint | 1 | code issued; `organization={tenant-b:{id:…b1}}`; dave is a member of `[tenant-b]` | n/a |
| **E14b**, `s1b-host`: `organization:tenant-a` plus `kc_idp_hint=idp-tenant-b`, dave | 3/3 | **code issued**; `organization` claim absent; `scope="openid"`; `idp=idp-tenant-b`; `Resolve` returns `tenant_unresolved: no organization` | 1 **PASS**, 2 **PASS** |
| **E14b**, `nava-web`: same request | 3/3 | same result; `scope="openid profile email"`. The default bare `organization` is dropped too. | 1 **PASS**, 2 **PASS** |
| **E14d**: fresh user, first sign-in with `organization:tenant-a` plus the hint | 3/3 | code issued; claim absent; **memberships after the first sign-in: `[tenant-b]`** | 3 **FAIL** |
| E14d follow-up: the same user signs in again with `organization:tenant-b` | 3/3 | `organization={tenant-b:{id:…b1}}`. The user is not stranded. | n/a |
| E14d control: fresh user with the right alias | 1 | `[tenant-b]`, claim `tenant-b` | n/a |
| Side check, E14a: no hint, `organization:tenant-a` | 1 | 200 with `kc-form-login` (the realm form); no code. Without SSO only. | matches D1-A1.4(a) |
| Side check, E14c: `organization:no-such-org`, with and without the hint | 2 | 302 to `redirect_uri` with `error=invalid_scope`; no code | matches D1-A1.4(c) |

In every run, no token named an org the user isn't a member of, so the Security F4 invariant holds.

## Why claim 3 was wrong
I read `IdpAddOrganizationMemberAuthenticator.java:97-104` as resolving the requested alias (`tenant-a`) and skipping `tenant-b`. At runtime it doesn't. **Inferred, not traced:** during the first-broker-login request, `Organizations.resolveOrganization(session)` doesn't see the `organization:tenant-a` scope (for example, `OrganizationScope.getRequestedScopes` returns an empty scope when the context has no authentication session, `OrganizationScope.java:353-358`). In that case `organization == null` and the loop adds every org linked to the IdP. Whatever the exact cause, the runtime result is what counts. E14d pins it, so a Keycloak upgrade that changes it will show up.

## Implications for the design (text for rev 1's ADR, for the Solution Architect)
The fail branch in §7 ("form, error or an organization claim → revert D1-A1.4(b), CI-E14b and CI-E14d to the rev 0 'no token' text") would be wrong here. None of those fail conditions occurred, and E14b confirmed that a token is issued. I propose this narrower change instead:
- **D1-A1.4(b): keep it, and drop its last sentence.** New text: "Same, **with** `kc_idp_hint`: Keycloak brokers, issues a code and a token with **no `organization` claim**, and `Resolve` answers 401 `tenant_unresolved`. A first brokered sign-in still adds the user to the orgs linked to their IdP. *Verified by S1b, 3/3.*" Remove "pending S1b".
- **RR-A1-5: withdraw it.** Nothing is lost: the user's next sign-in with the right alias resolves.
- **CI-E14b: mark it verified by S1b,** and run it for both client shapes (default-optional and default-bare `organization`).
- **CI-E14d: restate it as** "Fresh user, first sign-in via `idp-<B>` with `organization:t-<A>`: the token has no `organization` claim, and afterwards the user is a member of **B only**. **Fails if** the user is a member of A (isolation break) or of no org (behaviour changed: re-open RR-A1-5)."
- **D1-A1.3 pairing sentence:** it no longer protects against F3. It's still good hygiene, since a mismatched pair costs the host a 401. Keeping it or dropping it is the Solution Architect's call.
- **CI-E14a with an SSO session:** S1b ran it without SSO only. Rev 0's S1 evidence covers the SSO case, and CI-E14a still asserts it.

With these edits, the S1 row in §7 is PASS on S1 + S1b evidence.

## How to rerun
1. Start the scratch Keycloak. Stop it afterwards with `docker rm -f nava-spike-s1b`.
   ```sh
   mkdir -p s1b/import && cp deploy/compose/keycloak/import/realm-nava.json s1b/import/
   docker run -d --name nava-spike-s1b -p 127.0.0.1:18081:8080 \
     -e KC_BOOTSTRAP_ADMIN_USERNAME=admin -e KC_BOOTSTRAP_ADMIN_PASSWORD=dev-only-admin \
     -e KC_HOSTNAME=http://localhost:18081 -e KC_HOSTNAME_BACKCHANNEL_DYNAMIC=true -e KC_HTTP_ENABLED=true \
     -v "$PWD/s1b/import:/opt/keycloak/data/import:ro" \
     quay.io/keycloak/keycloak:26.8.1 start-dev --import-realm
   ```
2. Save the file below as `internal/identitystack/s1b_spike_test.go`. Do not commit it; it is spike code, and the engineers write the real E14b and E14d.
3. Run `NAVA_TEST_KEYCLOAK_URL=http://localhost:18081 go test -tags identitystack -count=1 -v -run TestS1b ./internal/identitystack`.

The expected result is that `TestS1bE14dFreshUserWrongAlias` fails on "want none". That failure is the finding.

<details><summary>Spike code: <code>internal/identitystack/s1b_spike_test.go</code> (throwaway)</summary>

```go
//go:build identitystack

// SPIKE S1b (TICK-843, RFC-018 rev 1 §7). Throwaway: never committed.
// CI-E14b and CI-E14d written first, as the spike.
package identitystack

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

const s1bClient = "s1b-host"
const s1bRedirect = "http://localhost:9/cb"

// adminRealm calls the admin API of any realm with the master admin token.
func s1bAdmin(t *testing.T, tok, realm, method, path string, body any) (int, []byte) {
	t.Helper()
	var rdr *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = strings.NewReader(string(b))
	} else {
		rdr = strings.NewReader("")
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, baseURL()+"/admin/realms/"+realm+path, rdr)
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return do(t, newClient(t), req)
}

// ensureHostClient registers an embedding-style public client: no default
// organization scope, organization optional (the shape D1 prescribes).
func ensureHostClient(t *testing.T, adm string) {
	t.Helper()
	code, b := s1bAdmin(t, adm, "nava", http.MethodGet, "/clients?clientId="+s1bClient, nil)
	var cs []map[string]any
	_ = json.Unmarshal(b, &cs)
	if code == 200 && len(cs) == 1 {
		return
	}
	code, b = s1bAdmin(t, adm, "nava", http.MethodPost, "/clients", map[string]any{
		"clientId": s1bClient, "enabled": true, "publicClient": true, "standardFlowEnabled": true,
		"directAccessGrantsEnabled": false, "redirectUris": []string{s1bRedirect},
		"attributes":           map[string]string{"pkce.code.challenge.method": ""},
		"defaultClientScopes":  []string{"basic", "nava-idp", "nava-api-audience"},
		"optionalClientScopes": []string{"organization"},
	})
	if code != http.StatusCreated {
		t.Fatalf("create client: %d %s", code, b)
	}
}

type flowResult struct {
	codeParam, errParam string
	page                string // last page if the flow never reached the redirect uri
	status              int
}

// authFlow runs the authorization-code flow; when the IdP realm shows its
// login form, it signs in there. It never fills a form of realm nava.
func authFlow(t *testing.T, client, scope, idpHint, user, pass string) flowResult {
	t.Helper()
	c := newClient(t)
	var final *url.URL
	c.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		if strings.HasPrefix(req.URL.String(), s1bRedirect) {
			final = req.URL
			return http.ErrUseLastResponse
		}
		return nil
	}
	q := url.Values{"client_id": {client}, "response_type": {"code"}, "scope": {scope}, "redirect_uri": {s1bRedirect}}
	if idpHint != "" {
		q.Set("kc_idp_hint", idpHint)
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		baseURL()+"/realms/nava/protocol/openid-connect/auth?"+q.Encode(), nil)
	status, page := do(t, c, req)
	if final == nil {
		m := formAction.FindSubmatch(page)
		if m == nil {
			m = formActionAlt.FindSubmatch(page)
		}
		action := ""
		if m != nil {
			action = html.UnescapeString(string(m[1]))
		}
		if !strings.Contains(action, "/realms/idp-tenant-") {
			return flowResult{page: string(page), status: status}
		}
		post, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, action,
			strings.NewReader(url.Values{"username": {user}, "password": {pass}}.Encode()))
		post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		status, page = do(t, c, post)
	}
	if final == nil {
		return flowResult{page: string(page), status: status}
	}
	return flowResult{codeParam: final.Query().Get("code"), errParam: final.Query().Get("error"), status: 302}
}

func exchange(t *testing.T, client, code string) string {
	t.Helper()
	st, out := tokenForm(t, "nava", url.Values{
		"grant_type": {"authorization_code"}, "client_id": {client}, "code": {code}, "redirect_uri": {s1bRedirect},
	})
	if st != http.StatusOK {
		t.Fatalf("code exchange: %d %v", st, out)
	}
	return out["access_token"].(string)
}

func memberships(t *testing.T, adm, username string) []string {
	t.Helper()
	id := userIDRealm(t, adm, username)
	code, b := s1bAdmin(t, adm, "nava", http.MethodGet, "/organizations/members/"+id+"/organizations", nil)
	if code != 200 {
		t.Fatalf("memberships of %s: %d %s", username, code, b)
	}
	var os []struct{ Alias string }
	_ = json.Unmarshal(b, &os)
	var out []string
	for _, o := range os {
		out = append(out, o.Alias)
	}
	return out
}

func userIDRealm(t *testing.T, adm, username string) string {
	t.Helper()
	code, b := s1bAdmin(t, adm, "nava", http.MethodGet, "/users?exact=true&username="+url.QueryEscape(username), nil)
	var us []struct{ ID string }
	if code != 200 || json.Unmarshal(b, &us) != nil || len(us) != 1 {
		t.Fatalf("lookup %s in nava: %d %s", username, code, b)
	}
	return us[0].ID
}

func freshIdPUser(t *testing.T, adm string) (string, string) {
	t.Helper()
	r := make([]byte, 4)
	_, _ = rand.Read(r)
	u := "s1b-" + hex.EncodeToString(r) + "@tenant-b.test"
	p := "dev-only-s1b"
	code, b := s1bAdmin(t, adm, "idp-tenant-b", http.MethodPost, "/users", map[string]any{
		"username": u, "email": u, "emailVerified": true, "enabled": true, "firstName": "S", "lastName": "B",
		"credentials": []map[string]any{{"type": "password", "value": p, "temporary": false}},
	})
	if code != http.StatusCreated {
		t.Fatalf("create IdP user: %d %s", code, b)
	}
	return u, p
}

func orgKeys(c map[string]any) any {
	v, ok := c["organization"]
	if !ok {
		return "<absent>"
	}
	return v
}

// Control: the right alias with the hint resolves B (also creates dave in nava).
func TestS1bControlRightAlias(t *testing.T) {
	adm := masterToken(t)
	ensureHostClient(t, adm)
	r := authFlow(t, s1bClient, "openid organization:tenant-b", "idp-tenant-b", "dave@tenant-b.test", "dev-only-dave")
	if r.codeParam == "" {
		t.Fatalf("no code: %+v", r)
	}
	tok := exchange(t, s1bClient, r.codeParam)
	c := claims(t, tok)
	t.Logf("CONTROL organization=%v idp=%v scope=%v", orgKeys(c), c["idp"], c["scope"])
	orgClaim(t, c, "tenant-b")
	_, err := stackAdapter(t, tok).Resolve(t.Context(), tok)
	t.Logf("CONTROL Resolve err=%v", err)
	if err != nil && strings.Contains(err.Error(), "no organization") {
		t.Fatalf("control token rejected for missing organization: %v", err)
	}
	t.Logf("CONTROL dave memberships=%v", memberships(t, adm, "dave@tenant-b.test"))
}

// CI-E14b: existing B user, hint idp-tenant-b, alias tenant-a, 3 runs.
func TestS1bE14bHintedWrongAlias(t *testing.T) {
	adm := masterToken(t)
	ensureHostClient(t, adm)
	for _, client := range []string{s1bClient, "nava-web"} {
		for run := 1; run <= 3; run++ {
			r := authFlow(t, client, "openid organization:tenant-a", "idp-tenant-b", "dave@tenant-b.test", "dev-only-dave")
			if r.codeParam == "" {
				t.Errorf("E14b %s run %d: no code (error=%q status=%d page=%.200q)", client, run, r.errParam, r.status, r.page)
				continue
			}
			tok := exchange(t, client, r.codeParam)
			c := claims(t, tok)
			_, err := stackAdapter(t, tok).Resolve(t.Context(), tok)
			t.Logf("E14b %s run %d: code issued; organization=%v idp=%v scope=%q; Resolve=%v",
				client, run, orgKeys(c), c["idp"], c["scope"], err)
			if _, has := c["organization"]; has {
				t.Errorf("E14b %s run %d: organization claim present: %v", client, run, c["organization"])
			}
			if !errors.Is(err, identity.ErrTenantUnresolved) || !strings.Contains(err.Error(), "no organization") {
				t.Errorf("E14b %s run %d: Resolve = %v, want tenant_unresolved: no organization", client, run, err)
			}
		}
	}
}

// CI-E14d: fresh B user, first sign-in with hint idp-tenant-b and alias tenant-a, 3 runs;
// plus one fresh control user with the right alias.
func TestS1bE14dFreshUserWrongAlias(t *testing.T) {
	adm := masterToken(t)
	ensureHostClient(t, adm)
	for run := 1; run <= 3; run++ {
		u, p := freshIdPUser(t, adm)
		r := authFlow(t, s1bClient, "openid organization:tenant-a", "idp-tenant-b", u, p)
		if r.codeParam == "" {
			t.Errorf("E14d run %d: no code (error=%q status=%d page=%.200q)", run, r.errParam, r.status, r.page)
			continue
		}
		c := claims(t, exchange(t, s1bClient, r.codeParam))
		m := memberships(t, adm, u)
		t.Logf("E14d run %d: user %s; organization=%v; memberships after first sign-in=%v", run, u, orgKeys(c), m)
		if len(m) != 0 {
			t.Errorf("E14d run %d: memberships = %v, want none", run, m)
		}
		// Second sign-in with the RIGHT alias: does membership ever appear?
		r2 := authFlow(t, s1bClient, "openid organization:tenant-b", "idp-tenant-b", u, p)
		if r2.codeParam != "" {
			c2 := claims(t, exchange(t, s1bClient, r2.codeParam))
			t.Logf("E14d run %d: later sign-in with organization:tenant-b -> organization=%v memberships=%v",
				run, orgKeys(c2), memberships(t, adm, u))
		} else {
			t.Logf("E14d run %d: later sign-in with organization:tenant-b -> no code (error=%q status=%d page=%.200q)", run, r2.errParam, r2.status, r2.page)
		}
	}
	u, p := freshIdPUser(t, adm)
	r := authFlow(t, s1bClient, "openid organization:tenant-b", "idp-tenant-b", u, p)
	if r.codeParam == "" {
		t.Fatalf("E14d control: no code: %+v", r)
	}
	c := claims(t, exchange(t, s1bClient, r.codeParam))
	t.Logf("E14d CONTROL fresh %s right alias: organization=%v memberships=%v", u, orgKeys(c), memberships(t, adm, u))
}

// Side checks (not S1b criteria): E14a and E14c.
func TestS1bSideE14aE14c(t *testing.T) {
	adm := masterToken(t)
	ensureHostClient(t, adm)
	r := authFlow(t, s1bClient, "openid organization:tenant-a", "", "dave@tenant-b.test", "dev-only-dave")
	t.Logf("E14a (no hint): code=%q error=%q status=%d loginform=%v", r.codeParam, r.errParam, r.status,
		strings.Contains(r.page, "kc-form-login"))
	for _, hint := range []string{"", "idp-tenant-b"} {
		r := authFlow(t, s1bClient, "openid organization:no-such-org", hint, "dave@tenant-b.test", "dev-only-dave")
		t.Logf("E14c (hint=%q): code=%q error=%q status=%d", hint, r.codeParam, r.errParam, r.status)
	}
}
```

</details>

<details><summary>Raw test output</summary>

```text
    s1b_spike_test.go:185: CONTROL organization=map[tenant-b:map[id:00000000-0000-4000-8000-0000000000b1]] idp=idp-tenant-b scope=openid organization:tenant-b
    s1b_spike_test.go:188: CONTROL Resolve err=tenant_unresolved: no store
    s1b_spike_test.go:192: CONTROL dave memberships=[tenant-b]
--- PASS: TestS1bControlRightAlias (1.78s)
    s1b_spike_test.go:209: E14b s1b-host run 1: code issued; organization=<absent> idp=idp-tenant-b scope="openid"; Resolve=tenant_unresolved: no organization
    s1b_spike_test.go:209: E14b s1b-host run 2: code issued; organization=<absent> idp=idp-tenant-b scope="openid"; Resolve=tenant_unresolved: no organization
    s1b_spike_test.go:209: E14b s1b-host run 3: code issued; organization=<absent> idp=idp-tenant-b scope="openid"; Resolve=tenant_unresolved: no organization
    s1b_spike_test.go:209: E14b nava-web run 1: code issued; organization=<absent> idp=idp-tenant-b scope="openid profile email"; Resolve=tenant_unresolved: no organization
    s1b_spike_test.go:209: E14b nava-web run 2: code issued; organization=<absent> idp=idp-tenant-b scope="openid profile email"; Resolve=tenant_unresolved: no organization
    s1b_spike_test.go:209: E14b nava-web run 3: code issued; organization=<absent> idp=idp-tenant-b scope="openid profile email"; Resolve=tenant_unresolved: no organization
--- PASS: TestS1bE14bHintedWrongAlias (1.68s)
    s1b_spike_test.go:235: E14d run 1: user s1b-b2f7a62c@tenant-b.test; organization=<absent>; memberships after first sign-in=[tenant-b]
    s1b_spike_test.go:237: E14d run 1: memberships = [tenant-b], want none
    s1b_spike_test.go:243: E14d run 1: later sign-in with organization:tenant-b -> organization=map[tenant-b:map[id:00000000-0000-4000-8000-0000000000b1]] memberships=[tenant-b]
    s1b_spike_test.go:235: E14d run 2: user s1b-9297a4b9@tenant-b.test; organization=<absent>; memberships after first sign-in=[tenant-b]
    s1b_spike_test.go:237: E14d run 2: memberships = [tenant-b], want none
    s1b_spike_test.go:243: E14d run 2: later sign-in with organization:tenant-b -> organization=map[tenant-b:map[id:00000000-0000-4000-8000-0000000000b1]] memberships=[tenant-b]
    s1b_spike_test.go:235: E14d run 3: user s1b-0ae290d3@tenant-b.test; organization=<absent>; memberships after first sign-in=[tenant-b]
    s1b_spike_test.go:237: E14d run 3: memberships = [tenant-b], want none
    s1b_spike_test.go:243: E14d run 3: later sign-in with organization:tenant-b -> organization=map[tenant-b:map[id:00000000-0000-4000-8000-0000000000b1]] memberships=[tenant-b]
    s1b_spike_test.go:255: E14d CONTROL fresh s1b-5af3a7e9@tenant-b.test right alias: organization=map[tenant-b:map[id:00000000-0000-4000-8000-0000000000b1]] memberships=[tenant-b]
--- FAIL: TestS1bE14dFreshUserWrongAlias (2.41s)
    s1b_spike_test.go:263: E14a (no hint): code="" error="" status=200 loginform=true
    s1b_spike_test.go:267: E14c (hint=""): code="" error="invalid_scope" status=302
    s1b_spike_test.go:267: E14c (hint="idp-tenant-b"): code="" error="invalid_scope" status=302
--- PASS: TestS1bSideE14aE14c (0.14s)
FAIL
FAIL	github.com/Navaladi2019/TicketingSystem/internal/identitystack	10.292s
FAIL
```

</details>
